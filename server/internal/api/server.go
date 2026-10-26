// Package api 提供 HTTP 路由装配、鉴权中间件以及各业务接口处理器。
package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"mini-wechat/server/internal/auth"
	"mini-wechat/server/internal/observability"
	"mini-wechat/server/internal/realtime"
	"mini-wechat/server/internal/store"
)

// Server 聚合各处理函数依赖的共享组件。
type Server struct {
	store      *store.Store
	hub        *realtime.Hub
	metrics    *observability.Metrics
	secret     string
	uploadsDir string
}

// New 构造 API 服务。metrics 与 hub 由 main 统一创建，便于共享同一份状态。
func New(st *store.Store, hub *realtime.Hub, metrics *observability.Metrics, secret, uploadsDir string) *Server {
	return &Server{store: st, hub: hub, metrics: metrics, secret: secret, uploadsDir: uploadsDir}
}

// upgrader 放开跨域校验，方便本地前后端分离调试；生产部署建议按可信域名收敛。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// ctxUserID 是登录用户 ID 在 Gin 上下文中的键。
const ctxUserID = "userID"

// Router 装配中间件与全部路由。
// 中间件顺序：链路追踪 -> 指标 -> 访问日志 -> panic 恢复 -> CORS，
// 保证即使处理器 panic，指标与 span 仍能记录到 5xx 状态。
func (s *Server) Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(
		observability.Tracing(),
		s.metrics.Middleware(),
		accessLogger(),
		gin.Recovery(),
		corsMiddleware(),
	)

	r.Static("/uploads", s.uploadsDir)
	r.GET("/metrics", gin.WrapH(s.metrics.Handler()))
	r.GET("/ws", s.handleWS)

	api := r.Group("/api")
	api.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	api.POST("/auth/register", s.register)
	api.POST("/auth/login", s.login)

	// authed 分组下的所有接口都要求携带 Bearer JWT。
	authed := api.Group("", s.requireAuth())
	authed.GET("/me", s.me)
	authed.PATCH("/me", s.updateMe)
	authed.GET("/users", s.listUsers)
	authed.POST("/upload", s.upload)

	authed.GET("/conversations", s.listConversations)
	authed.POST("/conversations/direct", s.createDirect)
	authed.POST("/conversations/group", s.createGroup)
	authed.GET("/conversations/:id", s.getConversation)
	authed.GET("/conversations/:id/messages", s.listMessages)
	authed.POST("/conversations/:id/messages", s.sendMessage)
	authed.POST("/conversations/:id/read", s.markRead)

	authed.GET("/moments", s.listMoments)
	authed.POST("/moments", s.createMoment)
	authed.POST("/moments/:id/like", s.toggleLike)
	authed.POST("/moments/:id/comments", s.commentMoment)

	return r
}

// requireAuth 校验 Authorization 头中的 JWT，并把用户 ID 写入请求上下文。
func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		uid, err := auth.ParseToken(s.secret, strings.TrimSpace(strings.TrimPrefix(header, prefix)))
		if err != nil {
			fail(c, http.StatusUnauthorized, "登录已过期，请重新登录")
			c.Abort()
			return
		}
		c.Set(ctxUserID, uid)
		observability.SetUser(c.Request, uid)
		c.Next()
	}
}

// tokenFor 为指定用户签发访问令牌，有效期 7 天。
func (s *Server) tokenFor(id int64) (string, error) {
	return auth.NewToken(s.secret, id, 7*24*time.Hour)
}

// userID 取出当前登录用户 ID；未登录时返回 0。
func userID(c *gin.Context) int64 {
	v, _ := c.Get(ctxUserID)
	id, _ := v.(int64)
	return id
}

// fail 统一错误响应格式：{"error": "提示文案"}。
func fail(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// bindJSON 解析请求体，失败时直接返回 400。
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式不正确")
		return false
	}
	return true
}

// pathID 解析路径中的数字 ID 参数，非法时返回 400。
func pathID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		fail(c, http.StatusBadRequest, "参数不正确")
		return 0, false
	}
	return id, true
}

// handleWS 完成 WebSocket 握手。浏览器原生 WebSocket 无法自定义请求头，
// 因此令牌通过查询参数 token 传递。
func (s *Server) handleWS(c *gin.Context) {
	uid, err := auth.ParseToken(s.secret, c.Query("token"))
	if err != nil {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}
	observability.SetUser(c.Request, uid)
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	s.hub.Handle(c.Request.Context(), uid, conn)
}

// accessLogger 沿用 Gin 默认日志格式，但去掉查询串，
// 否则 WebSocket 握手携带的 JWT 会被永久记录到容器日志中。
func accessLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		path, _, _ := strings.Cut(p.Path, "?")
		return fmt.Sprintf("[GIN] %v |%3d| %13v | %15s |%-7s %q\n",
			p.TimeStamp.Format("2006/01/02 - 15:04:05"),
			p.StatusCode,
			p.Latency,
			p.ClientIP,
			p.Method,
			path,
		)
	})
}

// corsMiddleware 回显请求 Origin 以支持本地跨域调试，并直接短路响应预检请求。
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
