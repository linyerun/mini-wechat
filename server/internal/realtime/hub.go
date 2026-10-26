// Package realtime 实现基于 WebSocket 的实时消息推送 Hub。
package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"mini-wechat/server/internal/observability"
)

// Event 是下发给客户端的消息协议，Type 为 ready / message / typing。
type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// client 表示一条用户连接。同一用户可能有多个连接（多端登录）。
type client struct {
	userID int64
	conn   *websocket.Conn
	send   chan []byte
	ctx    context.Context
	span   trace.Span
}

// Hub 维护 userID -> 连接集合 的注册表，负责事件的并发分发。
type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[*client]struct{}
	// Members 由外部注入，用于解析会话成员以转发 typing 事件。
	Members func(ctx context.Context, conversationID int64) ([]int64, error)
	Metrics *observability.Metrics
}

// NewHub 创建一个空的 Hub。
func NewHub() *Hub {
	return &Hub{clients: map[int64]map[*client]struct{}{}}
}

// add 把连接登记到注册表。
func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = map[*client]struct{}{}
	}
	h.clients[c.userID][c] = struct{}{}
}

// remove 注销连接；只有真正移除时才递减在线计数，避免重复调用导致负值。
func (h *Hub) remove(c *client) {
	h.mu.Lock()
	removed := false
	if set, ok := h.clients[c.userID]; ok {
		if _, ok := set[c]; ok {
			delete(set, c)
			removed = true
		}
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
	h.mu.Unlock()
	if removed {
		h.Metrics.WSDisconnected()
	}
}

// Handle 接管连接并立即返回，读写由两个 goroutine 负责。
// 连接级 span 会挂到握手请求的 trace 上（以 link 形式），
// 因为握手 span 在升级完成后就结束了，而连接会存活很久。
func (h *Hub) Handle(ctx context.Context, userID int64, conn *websocket.Conn) {
	c := &client{userID: userID, conn: conn, send: make(chan []byte, 64)}

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attribute.Int64("enduser.id", userID)),
	}
	if parent := trace.SpanContextFromContext(ctx); parent.IsValid() {
		opts = append(opts, trace.WithLinks(trace.Link{SpanContext: parent}))
	}
	c.ctx, c.span = otel.Tracer("mini-wechat/realtime").Start(context.Background(), "ws.connect", opts...)

	h.add(c)
	h.Metrics.WSConnected()
	c.enqueue(Event{Type: "ready"})
	go c.writePump()
	go c.readPump(h)
}

// SendToUsers 向指定用户的所有连接广播事件，exclude 中的用户会被跳过。
func (h *Hub) SendToUsers(userIDs []int64, ev Event, exclude ...int64) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	skip := make(map[int64]bool, len(exclude))
	for _, id := range exclude {
		skip[id] = true
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, id := range userIDs {
		if skip[id] {
			continue
		}
		for c := range h.clients[id] {
			// 非阻塞投递：慢客户端只会丢事件，不能拖垮整个广播。
			select {
			case c.send <- data:
				h.Metrics.WSEventSent(ev.Type)
			default:
				h.Metrics.WSEventDropped(ev.Type)
			}
		}
	}
}

// inbound 是客户端上行消息，目前只处理 typing。
type inbound struct {
	Type           string `json:"type"`
	ConversationID int64  `json:"conversationId"`
}

// enqueue 把事件放入该连接的发送队列，队列满时直接丢弃。
func (c *client) enqueue(ev Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}

// 读写超时参数：pingPeriod 必须小于 pongWait，否则空闲连接会被判定失活。
const (
	writeWait  = 10 * time.Second
	pongWait   = 70 * time.Second
	pingPeriod = 30 * time.Second
)

// writePump 串行写出队列中的消息，并周期性发送 ping 保活。
func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case data, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump 读取上行消息；读取出错（含超时）即视为连接结束并负责清理。
func (c *client) readPump(h *Hub) {
	defer func() {
		h.remove(c)
		c.span.End()
		c.conn.Close()
	}()
	c.conn.SetReadLimit(16 << 10)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg inbound
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		if msg.Type != "typing" || msg.ConversationID <= 0 || h.Members == nil {
			continue
		}
		c.handleTyping(h, msg)
	}
}

// handleTyping 校验发送方确实是会话成员后，把输入中提示转发给其他成员。
func (c *client) handleTyping(h *Hub, msg inbound) {
	ctx, span := otel.Tracer("mini-wechat/realtime").Start(c.ctx, "ws.typing",
		trace.WithAttributes(attribute.Int64("conversation.id", msg.ConversationID)),
	)
	defer span.End()

	memberIDs, err := h.Members(ctx, msg.ConversationID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "resolve conversation members")
		return
	}
	isMember := false
	for _, id := range memberIDs {
		if id == c.userID {
			isMember = true
			break
		}
	}
	if !isMember {
		return
	}
	span.SetAttributes(attribute.Int("recipient.count", len(memberIDs)))
	h.SendToUsers(memberIDs, Event{
		Type: "typing",
		Payload: map[string]any{
			"conversationId": msg.ConversationID,
			"userId":         c.userID,
		},
	}, c.userID)
}
