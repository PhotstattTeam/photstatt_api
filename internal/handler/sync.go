package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/middleware"
	"consiliumous-backend/internal/service"
)

// SyncUpload 移动端一次性批量上传个人计划（幂等 + 冲突处理）。
func (h *Handler) SyncUpload(c *gin.Context) {
	var in service.SyncUploadInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	results, err := h.Sync.Upload(middleware.CurrentUserID(c), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

// SyncBind 记录移动端绑定后端域名事件（审计）。
func (h *Handler) SyncBind(c *gin.Context) {
	var in struct {
		Domain string `json:"domain"`
	}
	_ = c.ShouldBindJSON(&in)
	if err := h.Sync.RecordBind(middleware.CurrentUserID(c), in.Domain); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
