package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health 健康检查，供部署与移动端绑定域名时探测可达性（PRD 2.4）。
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Version 返回服务版本，供移动端校验版本兼容性（PRD 2.4 / US-OFFLINE-02）。
// min_api_version 是移动端可连接的最低 API 版本，低于此值应提示"版本过低"。
func (h *Handler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": "0.1.0", "api_version": "v1", "min_api_version": "v1"})
}
