package api

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"mini-wechat/server/internal/store"
)

// listMoments 返回朋友圈时间线，点赞状态按当前查看者计算。
func (s *Server) listMoments(c *gin.Context) {
	moments, err := s.store.ListMoments(userID(c), 50)
	if err != nil {
		fail(c, http.StatusInternalServerError, "加载朋友圈失败")
		return
	}
	c.JSON(http.StatusOK, moments)
}

// createMoment 发布一条动态，文字与图片至少有一项。
func (s *Server) createMoment(c *gin.Context) {
	var req struct {
		Content string   `json:"content"`
		Images  []string `json:"images"`
	}
	if !bindJSON(c, &req) {
		return
	}
	content := strings.TrimSpace(req.Content)
	if utf8.RuneCountInString(content) > 2000 {
		fail(c, http.StatusBadRequest, "内容最多 2000 个字符")
		return
	}
	if len(req.Images) > 9 {
		fail(c, http.StatusBadRequest, "最多上传 9 张图片")
		return
	}
	// 图片同样只允许本站上传目录的相对路径。
	for _, img := range req.Images {
		if !strings.HasPrefix(img, "/uploads/") || len(img) > 500 {
			fail(c, http.StatusBadRequest, "图片地址不合法")
			return
		}
	}
	if content == "" && len(req.Images) == 0 {
		fail(c, http.StatusBadRequest, "说点什么或选一张图片吧")
		return
	}

	me := userID(c)
	id, err := s.store.CreateMoment(me, content, req.Images)
	if err != nil {
		fail(c, http.StatusInternalServerError, "发布失败，请稍后再试")
		return
	}
	// 重新查询一次，返回带点赞/评论结构的完整对象。
	moment, err := s.store.MomentByID(me, id)
	if err != nil {
		fail(c, http.StatusInternalServerError, "发布失败，请稍后再试")
		return
	}
	c.JSON(http.StatusOK, moment)
}

// toggleLike 点赞或取消点赞，返回最新状态与完整点赞列表。
func (s *Server) toggleLike(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	me := userID(c)
	if _, err := s.store.MomentByID(me, id); errors.Is(err, store.ErrNotFound) {
		fail(c, http.StatusNotFound, "动态不存在")
		return
	} else if err != nil {
		fail(c, http.StatusInternalServerError, "操作失败")
		return
	}
	liked, err := s.store.ToggleLike(id, me)
	if err != nil {
		fail(c, http.StatusInternalServerError, "操作失败")
		return
	}
	likes, err := s.store.MomentLikes(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, "操作失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"liked": liked, "likes": likes})
}

// commentMoment 为动态添加评论。
func (s *Server) commentMoment(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if !bindJSON(c, &req) {
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		fail(c, http.StatusBadRequest, "评论不能为空")
		return
	}
	if utf8.RuneCountInString(content) > 500 {
		fail(c, http.StatusBadRequest, "评论最多 500 个字符")
		return
	}
	me := userID(c)
	if _, err := s.store.MomentByID(me, id); errors.Is(err, store.ErrNotFound) {
		fail(c, http.StatusNotFound, "动态不存在")
		return
	} else if err != nil {
		fail(c, http.StatusInternalServerError, "评论失败")
		return
	}
	comment, err := s.store.AddComment(id, me, content)
	if err != nil {
		fail(c, http.StatusInternalServerError, "评论失败，请稍后再试")
		return
	}
	c.JSON(http.StatusOK, comment)
}
