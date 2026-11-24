package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/middleware"
	"consiliumous-backend/internal/service"
)

// ---- 个人计划 ----

func (h *Handler) ListMyPlans(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	var f service.PersonalPlanFilter
	if err := c.ShouldBindQuery(&f); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	plans, err := h.Personal.List(me, me, f)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, plans)
}

func (h *Handler) CreateMyPlan(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	var in service.CreatePersonalPlanInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.Personal.Create(me, me, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) GetMyPlan(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	planID, ok := parsePathID(c, "planId")
	if !ok {
		return
	}
	p, err := h.Personal.Get(me, me, planID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpdateMyPlan(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	planID, ok := parsePathID(c, "planId")
	if !ok {
		return
	}
	var in service.UpdatePersonalPlanInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.Personal.Update(me, me, planID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) DeleteMyPlan(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	planID, ok := parsePathID(c, "planId")
	if !ok {
		return
	}
	if err := h.Personal.Delete(me, me, planID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) UpdateMyPlanState(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	planID, ok := parsePathID(c, "planId")
	if !ok {
		return
	}
	var in service.PersonalPlanStateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.Personal.UpdateState(me, me, planID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// ListMyAssignedPlans 虚拟"项目计划组"：分配给我的项目计划。
func (h *Handler) ListMyAssignedPlans(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	rows, err := h.Personal.ListAssigned(me)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---- 个人计划组 ----

func (h *Handler) ListMyPlanGroups(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	groups, err := h.Personal.ListGroups(me, me)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (h *Handler) CreateMyPlanGroup(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	var in service.PlanGroupInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g, err := h.Personal.CreateGroup(me, me, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, g)
}

func (h *Handler) UpdateMyPlanGroup(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	groupID, ok := parsePathID(c, "groupId")
	if !ok {
		return
	}
	var in service.PlanGroupInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g, err := h.Personal.UpdateGroup(me, me, groupID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, g)
}

func (h *Handler) DeleteMyPlanGroup(c *gin.Context) {
	me := middleware.CurrentUserID(c)
	groupID, ok := parsePathID(c, "groupId")
	if !ok {
		return
	}
	if err := h.Personal.DeleteGroup(me, me, groupID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
