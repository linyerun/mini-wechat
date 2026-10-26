package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"mini-wechat/server/internal/realtime"
	"mini-wechat/server/internal/store"
)

// listConversations 返回当前用户的会话列表，含最后一条消息与未读数。
func (s *Server) listConversations(c *gin.Context) {
	conversations, err := s.store.ListConversations(userID(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, "加载会话失败")
		return
	}
	c.JSON(http.StatusOK, conversations)
}

// createDirect 打开单聊：同一对用户复用已有会话，避免重复创建。
func (s *Server) createDirect(c *gin.Context) {
	var req struct {
		UserID int64 `json:"userId"`
	}
	if !bindJSON(c, &req) {
		return
	}
	me := userID(c)
	if req.UserID <= 0 || req.UserID == me {
		fail(c, http.StatusBadRequest, "请选择一位联系人")
		return
	}
	if _, err := s.store.UserByID(req.UserID); err != nil {
		fail(c, http.StatusNotFound, "联系人不存在")
		return
	}

	convID, err := s.store.FindDirect(me, req.UserID)
	if errors.Is(err, store.ErrNotFound) {
		convID, err = s.store.CreateConversation("direct", "", 0, []int64{me, req.UserID})
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "发起聊天失败")
		return
	}
	conv, err := s.store.ConversationForUser(convID, me)
	if err != nil {
		fail(c, http.StatusInternalServerError, "发起聊天失败")
		return
	}
	c.JSON(http.StatusOK, conv)
}

// createGroup 创建群聊，成员会去重并过滤非法 ID。
func (s *Server) createGroup(c *gin.Context) {
	var req struct {
		Name      string  `json:"name"`
		MemberIDs []int64 `json:"memberIds"`
	}
	if !bindJSON(c, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 30 {
		fail(c, http.StatusBadRequest, "群名称需为 1-30 个字符")
		return
	}
	me := userID(c)
	seen := map[int64]bool{me: true}
	members := []int64{me}
	for _, id := range req.MemberIDs {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		members = append(members, id)
	}
	if len(members) < 2 {
		fail(c, http.StatusBadRequest, "请至少选择一位联系人")
		return
	}
	if len(members) > 50 {
		fail(c, http.StatusBadRequest, "群成员最多 50 人")
		return
	}
	// 先整体校验成员是否存在，避免写入半截的群成员关系。
	ok, err := s.store.UsersExist(members)
	if err != nil {
		fail(c, http.StatusInternalServerError, "创建群聊失败")
		return
	}
	if !ok {
		fail(c, http.StatusBadRequest, "部分成员不存在")
		return
	}

	convID, err := s.store.CreateConversation("group", name, me, members)
	if err != nil {
		fail(c, http.StatusInternalServerError, "创建群聊失败")
		return
	}
	conv, err := s.store.ConversationForUser(convID, me)
	if err != nil {
		fail(c, http.StatusInternalServerError, "创建群聊失败")
		return
	}
	c.JSON(http.StatusOK, conv)
}

// getConversation 返回单个会话详情。
func (s *Server) getConversation(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	conv, err := s.store.ConversationForUser(id, userID(c))
	if errors.Is(err, store.ErrNotFound) {
		fail(c, http.StatusNotFound, "会话不存在")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "加载会话失败")
		return
	}
	c.JSON(http.StatusOK, conv)
}

// listMessages 倒序分页拉取历史消息，before 传上一页最早一条的消息 ID。
func (s *Server) listMessages(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	me := userID(c)
	member, err := s.store.IsMember(id, me)
	if err != nil {
		fail(c, http.StatusInternalServerError, "加载消息失败")
		return
	}
	if !member {
		fail(c, http.StatusForbidden, "无权查看该会话")
		return
	}
	before, _ := strconv.ParseInt(c.Query("before"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))

	messages, err := s.store.ListMessages(id, before, limit)
	if err != nil {
		fail(c, http.StatusInternalServerError, "加载消息失败")
		return
	}
	c.JSON(http.StatusOK, messages)
}

// sendMessage 保存消息并通过 WebSocket 广播给会话成员。
func (s *Server) sendMessage(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	me := userID(c)
	member, err := s.store.IsMember(id, me)
	if err != nil {
		fail(c, http.StatusInternalServerError, "发送失败")
		return
	}
	if !member {
		fail(c, http.StatusForbidden, "无权在该会话中发言")
		return
	}

	var req struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if req.Type == "" {
		req.Type = "text"
	}
	if req.Type != "text" && req.Type != "image" {
		fail(c, http.StatusBadRequest, "不支持的消息类型")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		fail(c, http.StatusBadRequest, "消息不能为空")
		return
	}
	if utf8.RuneCountInString(content) > 4000 {
		fail(c, http.StatusBadRequest, "消息太长了")
		return
	}
	// 图片消息只保存地址，且限制为本站上传目录，避免外链注入。
	if req.Type == "image" && !strings.HasPrefix(content, "/uploads/") {
		fail(c, http.StatusBadRequest, "图片地址不合法")
		return
	}

	sender, err := s.store.UserByID(me)
	if err != nil {
		fail(c, http.StatusUnauthorized, "用户不存在")
		return
	}
	message, err := s.store.CreateMessage(id, me, req.Type, content, sender)
	if err != nil {
		fail(c, http.StatusInternalServerError, "发送失败，请稍后再试")
		return
	}
	s.metrics.MessageCreated(req.Type)
	// 广播失败不影响发送结果：消息已落库，客户端下次刷新仍能看到。
	if memberIDs, err := s.store.MemberIDs(c.Request.Context(), id); err == nil {
		s.hub.SendToUsers(memberIDs, realtime.Event{Type: "message", Payload: message})
	}
	c.JSON(http.StatusOK, message)
}

// markRead 把会话的已读位点推进到指定消息，用于清除未读角标。
func (s *Server) markRead(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	me := userID(c)
	member, err := s.store.IsMember(id, me)
	if err != nil {
		fail(c, http.StatusInternalServerError, "操作失败")
		return
	}
	if !member {
		fail(c, http.StatusForbidden, "无权操作该会话")
		return
	}
	var req struct {
		MessageID int64 `json:"messageId"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if err := s.store.MarkRead(id, me, req.MessageID); err != nil {
		fail(c, http.StatusInternalServerError, "操作失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
