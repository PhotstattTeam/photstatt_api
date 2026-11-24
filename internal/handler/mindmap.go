package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/middleware"
)

// ProjectMindmap 项目思维导图数据（项目 → 子项目 → 计划 → 任务）。
func (h *Handler) ProjectMindmap(c *gin.Context) {
	pid, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	from := c.Query("from")
	to := c.Query("to")
	result, err := h.Plan.Mindmap(pid, middleware.CurrentUserID(c), from, to)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
