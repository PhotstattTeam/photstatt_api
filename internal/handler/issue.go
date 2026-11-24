package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/middleware"
	"consiliumous-backend/internal/service"
)

// CreateIssue 新建 Issue。
func (h *Handler) CreateIssue(c *gin.Context) {
	pid, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var in service.CreateIssueInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	is, err := h.Issue.Create(middleware.CurrentUserID(c), pid, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, is)
}

// ListIssues 列出 Issue 池（支持筛选）。
func (h *Handler) ListIssues(c *gin.Context) {
	pid, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var f service.IssueFilter
	if err := c.ShouldBindQuery(&f); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	issues, err := h.Issue.List(pid, middleware.CurrentUserID(c), f)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, issues)
}

// GetIssue 查看单个 Issue。
func (h *Handler) GetIssue(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	issueID, ok2 := parsePathID(c, "issueId")
	if !ok1 || !ok2 {
		return
	}
	is, err := h.Issue.Get(pid, issueID, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, is)
}

// UpdateIssue 编辑 Issue。
func (h *Handler) UpdateIssue(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	issueID, ok2 := parsePathID(c, "issueId")
	if !ok1 || !ok2 {
		return
	}
	var in service.UpdateIssueInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	is, err := h.Issue.Update(middleware.CurrentUserID(c), pid, issueID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, is)
}

// DeleteIssue 删除 Issue。
func (h *Handler) DeleteIssue(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	issueID, ok2 := parsePathID(c, "issueId")
	if !ok1 || !ok2 {
		return
	}
	if err := h.Issue.Delete(middleware.CurrentUserID(c), pid, issueID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ChangeIssueStatus 关闭 / 拒绝 Issue。
func (h *Handler) ChangeIssueStatus(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	issueID, ok2 := parsePathID(c, "issueId")
	if !ok1 || !ok2 {
		return
	}
	var in service.IssueStatusInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	is, err := h.Issue.ChangeStatus(middleware.CurrentUserID(c), pid, issueID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, is)
}

// AddIssueComment 评论 Issue。
func (h *Handler) AddIssueComment(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	issueID, ok2 := parsePathID(c, "issueId")
	if !ok1 || !ok2 {
		return
	}
	var in struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cm, err := h.Issue.AddComment(middleware.CurrentUserID(c), pid, issueID, in.Content)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, cm)
}

// ListIssueComments 列出评论。
func (h *Handler) ListIssueComments(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	issueID, ok2 := parsePathID(c, "issueId")
	if !ok1 || !ok2 {
		return
	}
	cms, err := h.Issue.ListComments(pid, issueID, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, cms)
}
