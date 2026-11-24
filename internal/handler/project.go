package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/middleware"
	"consiliumous-backend/internal/service"
)

// CreateProject 创建项目。
func (h *Handler) CreateProject(c *gin.Context) {
	var in service.CreateProjectInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.Project.Create(middleware.CurrentUserID(c), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

// ListProjects 我参与的项目。
func (h *Handler) ListProjects(c *gin.Context) {
	ps, err := h.Project.ListForUser(middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ps)
}

// GetProject 项目详情。
func (h *Handler) GetProject(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	p, err := h.Project.GetForUser(id, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// ListMembers 项目成员列表。
func (h *Handler) ListMembers(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	rows, err := h.Project.ListMembers(id, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

// AddMember 添加成员。
func (h *Handler) AddMember(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var in service.AddMemberInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Project.AddMember(middleware.CurrentUserID(c), id, in); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

// RemoveMember 移除成员。
func (h *Handler) RemoveMember(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	uid, ok := parsePathID(c, "userId")
	if !ok {
		return
	}
	if err := h.Project.RemoveMember(middleware.CurrentUserID(c), id, uid); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// RevokeAdmin 撤销管理员。接替者通过查询参数 successor_id 传入。
func (h *Handler) RevokeAdmin(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	uid, ok := parsePathID(c, "userId")
	if !ok {
		return
	}
	var in service.RevokeAdminInput
	if s := c.Query("successor_id"); s != "" {
		v, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "非法的 successor_id"})
			return
		}
		sid := uint(v)
		in.SuccessorID = &sid
	}
	if err := h.Project.RevokeAdmin(middleware.CurrentUserID(c), id, uid, in); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ArchiveProject 归档项目。
func (h *Handler) ArchiveProject(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	p, err := h.Project.Archive(middleware.CurrentUserID(c), id, true)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// UnarchiveProject 取消归档项目。
func (h *Handler) UnarchiveProject(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	p, err := h.Project.Archive(middleware.CurrentUserID(c), id, false)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// DeleteProject 永久删除项目（仅系统管理员）。
func (h *Handler) DeleteProject(c *gin.Context) {
	id, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.Project.Delete(middleware.CurrentUserID(c), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// parsePathID 解析路径参数为 uint，失败时写出 400 并返回 false。
func parsePathID(c *gin.Context, key string) (uint, bool) {
	v, err := strconv.ParseUint(c.Param(key), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非法的 " + key})
		return 0, false
	}
	return uint(v), true
}
