package api

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"mini-wechat/server/internal/auth"
	"mini-wechat/server/internal/store"
)

// usernamePattern 限定用户名只包含字母、数字和下划线，便于直接用于登录查询。
var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)

// register 创建账号并直接返回令牌，前端无需再登录一次。
func (s *Server) register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if !bindJSON(c, &req) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Nickname = strings.TrimSpace(req.Nickname)
	if !usernamePattern.MatchString(req.Username) {
		fail(c, http.StatusBadRequest, "用户名需为 3-20 位字母、数字或下划线")
		return
	}
	if len(req.Password) < 6 || len(req.Password) > 32 {
		fail(c, http.StatusBadRequest, "密码长度需为 6-32 位")
		return
	}
	if req.Nickname == "" {
		req.Nickname = req.Username
	}
	// 昵称按字符数而非字节数校验，中文昵称才不会被误判超长。
	if utf8.RuneCountInString(req.Nickname) > 20 {
		fail(c, http.StatusBadRequest, "昵称最多 20 个字符")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		fail(c, http.StatusInternalServerError, "注册失败，请稍后再试")
		return
	}
	user, err := s.store.CreateUser(req.Username, hash, req.Nickname)
	if errors.Is(err, store.ErrConflict) {
		fail(c, http.StatusConflict, "用户名已被占用")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "注册失败，请稍后再试")
		return
	}
	token, err := s.tokenFor(user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "登录失败，请稍后再试")
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// login 校验密码并签发 JWT。
func (s *Server) login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !bindJSON(c, &req) {
		return
	}
	user, hash, err := s.store.UserByUsername(strings.TrimSpace(req.Username))
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		fail(c, http.StatusInternalServerError, "登录失败，请稍后再试")
		return
	}
	// 用户不存在与密码错误返回同一提示，避免暴露账号是否存在。
	if errors.Is(err, store.ErrNotFound) || !auth.CheckPassword(hash, req.Password) {
		fail(c, http.StatusUnauthorized, "用户名或密码不正确")
		return
	}
	token, err := s.tokenFor(user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "登录失败，请稍后再试")
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// me 返回当前登录用户资料，前端启动时用它校验令牌是否仍然有效。
func (s *Server) me(c *gin.Context) {
	user, err := s.store.UserByID(userID(c))
	if err != nil {
		fail(c, http.StatusUnauthorized, "用户不存在")
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// updateMe 局部更新资料：字段为 nil 表示不修改，因此指针区分“未传”和“传空值”。
func (s *Server) updateMe(c *gin.Context) {
	var req struct {
		Nickname  *string `json:"nickname"`
		Signature *string `json:"signature"`
		Avatar    *string `json:"avatar"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if req.Nickname != nil {
		v := strings.TrimSpace(*req.Nickname)
		if v == "" || utf8.RuneCountInString(v) > 20 {
			fail(c, http.StatusBadRequest, "昵称需为 1-20 个字符")
			return
		}
		req.Nickname = &v
	}
	if req.Signature != nil {
		v := strings.TrimSpace(*req.Signature)
		if utf8.RuneCountInString(v) > 60 {
			fail(c, http.StatusBadRequest, "个性签名最多 60 个字符")
			return
		}
		req.Signature = &v
	}
	if req.Avatar != nil {
		// 只接受本站上传目录的相对路径，防止把外部 URL 或脚本写进头像字段。
		v := strings.TrimSpace(*req.Avatar)
		if v != "" && !strings.HasPrefix(v, "/uploads/") {
			fail(c, http.StatusBadRequest, "头像地址不合法")
			return
		}
		if len(v) > 500 {
			fail(c, http.StatusBadRequest, "头像地址过长")
			return
		}
		req.Avatar = &v
	}
	user, err := s.store.UpdateUser(userID(c), req.Nickname, req.Signature, req.Avatar)
	if err != nil {
		fail(c, http.StatusInternalServerError, "保存失败，请稍后再试")
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// listUsers 返回除自己以外的全部用户，用作通讯录。
func (s *Server) listUsers(c *gin.Context) {
	users, err := s.store.ListUsers(userID(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, "加载联系人失败")
		return
	}
	c.JSON(http.StatusOK, users)
}
