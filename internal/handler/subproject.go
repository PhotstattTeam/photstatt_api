package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/middleware"
	"consiliumous-backend/internal/service"
)

// ListSubProjects 列出项目下的子项目。
func (h *Handler) ListSubProjects(c *gin.Context) {
	pid, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	sps, err := h.SubProject.List(pid, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, sps)
}

// CreateSubProject 创建子项目。
func (h *Handler) CreateSubProject(c *gin.Context) {
	pid, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var in service.SubProjectInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sp, err := h.SubProject.Create(middleware.CurrentUserID(c), pid, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, sp)
}

// UpdateSubProject 更新子项目。
func (h *Handler) UpdateSubProject(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	spID, ok2 := parsePathID(c, "spId")
	if !ok1 || !ok2 {
		return
	}
	var in service.SubProjectInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sp, err := h.SubProject.Update(middleware.CurrentUserID(c), pid, spID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, sp)
}

// DeleteSubProject 删除子项目。
func (h *Handler) DeleteSubProject(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	spID, ok2 := parsePathID(c, "spId")
	if !ok1 || !ok2 {
		return
	}
	if err := h.SubProject.Delete(middleware.CurrentUserID(c), pid, spID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
