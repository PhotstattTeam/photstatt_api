package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/middleware"
	"consiliumous-backend/internal/service"
)

// CreatePlan 直接新建项目计划（不经过 Issue）。
func (h *Handler) CreatePlan(c *gin.Context) {
	pid, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var in service.CreatePlanInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.Plan.CreateDirect(middleware.CurrentUserID(c), pid, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

// Schedule 勾选 Issue 批量排期。
func (h *Handler) Schedule(c *gin.Context) {
	pid, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var in service.ScheduleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	plans, err := h.Plan.ScheduleFromIssues(middleware.CurrentUserID(c), pid, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, plans)
}

// ListPlans 列出项目计划。
func (h *Handler) ListPlans(c *gin.Context) {
	pid, ok := parsePathID(c, "id")
	if !ok {
		return
	}
	var f service.PlanFilter
	if err := c.ShouldBindQuery(&f); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	plans, err := h.Plan.List(pid, middleware.CurrentUserID(c), f)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, plans)
}

// GetPlan 查看单个计划。
func (h *Handler) GetPlan(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	p, err := h.Plan.Get(pid, planID, middleware.CurrentUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// UpdatePlan 编辑计划。
func (h *Handler) UpdatePlan(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	var in service.UpdatePlanInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.Plan.Update(middleware.CurrentUserID(c), pid, planID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// AssignPlanOwner 分配/变更负责人。
func (h *Handler) AssignPlanOwner(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	var in service.AssignOwnerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.Plan.AssignOwner(middleware.CurrentUserID(c), pid, planID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// LinkPlanIssue 把计划改挂到另一个 Issue。
func (h *Handler) LinkPlanIssue(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	var in service.LinkIssueInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.Plan.LinkIssue(middleware.CurrentUserID(c), pid, planID, in.IssueID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// UnlinkPlanIssue 解除计划与 Issue 的关联。
func (h *Handler) UnlinkPlanIssue(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	p, err := h.Plan.UnlinkIssue(middleware.CurrentUserID(c), pid, planID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// UpdatePlanState 负责人更新计划状态与进度。
func (h *Handler) UpdatePlanState(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	var in service.PlanStateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.Plan.UpdateState(middleware.CurrentUserID(c), pid, planID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// CancelPlan 取消计划。
func (h *Handler) CancelPlan(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	p, err := h.Plan.Cancel(middleware.CurrentUserID(c), pid, planID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// DeletePlan 删除计划。
func (h *Handler) DeletePlan(c *gin.Context) {
	pid, ok1 := parsePathID(c, "id")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	if err := h.Plan.Delete(middleware.CurrentUserID(c), pid, planID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
