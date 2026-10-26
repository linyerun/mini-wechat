package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// allowedImageExt 按扩展名白名单放行，避免上传可被服务端解析的文件类型。
var allowedImageExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

// upload 保存图片并返回可直接访问的相对地址。
func (s *Server) upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "请选择要上传的图片")
		return
	}
	if file.Size <= 0 || file.Size > 8<<20 {
		fail(c, http.StatusBadRequest, "图片大小需在 8MB 以内")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedImageExt[ext] {
		fail(c, http.StatusBadRequest, "仅支持 jpg / png / gif / webp 图片")
		return
	}
	if err := os.MkdirAll(s.uploadsDir, 0o755); err != nil {
		fail(c, http.StatusInternalServerError, "上传失败")
		return
	}
	// 用时间戳 + 随机串重命名，既避免覆盖，也不信任用户提供的文件名。
	name := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), randomHex(4), ext)
	if err := c.SaveUploadedFile(file, filepath.Join(s.uploadsDir, name)); err != nil {
		fail(c, http.StatusInternalServerError, "上传失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": "/uploads/" + name})
}

// randomHex 生成 n 字节随机十六进制串；随机源异常时退化为时间戳。
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
