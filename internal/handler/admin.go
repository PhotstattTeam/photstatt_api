package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/middleware"
	"consiliumous-backend/internal/service"
)

// requireAdmin 校验当前用户是否为系统管理员，不是则返回 403。
func (h *Handler) requireAdmin(c *gin.Context) bool {
	ok, err := service.IsSystemAdmin(h.Auth.DB, middleware.CurrentUserID(c))
	if err != nil || !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要系统管理员权限"})
		return false
	}
	return true
}

// AdminListUserPlans 系统管理员查看某用户的个人计划（PRD 权限矩阵表 D）。
func (h *Handler) AdminListUserPlans(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	ownerID, ok := parsePathID(c, "userId")
	if !ok {
		return
	}
	var f service.PersonalPlanFilter
	if err := c.ShouldBindQuery(&f); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID := middleware.CurrentUserID(c)
	plans, err := h.Personal.List(actorID, ownerID, f)
	if err != nil {
		respondError(c, err)
		return
	}
	if actorID != ownerID {
		_ = h.Personal.LogAdminView(actorID, ownerID) // 系统管理员查看他人个人计划须留痕（PRD 7）
	}
	c.JSON(http.StatusOK, plans)
}

// AdminUpdateUserPlan 系统管理员修改某用户的个人计划。
func (h *Handler) AdminUpdateUserPlan(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	ownerID, ok1 := parsePathID(c, "userId")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	var in service.UpdatePersonalPlanInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.Personal.Update(middleware.CurrentUserID(c), ownerID, planID, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// AdminDeleteUserPlan 系统管理员删除某用户的个人计划。
func (h *Handler) AdminDeleteUserPlan(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	ownerID, ok1 := parsePathID(c, "userId")
	planID, ok2 := parsePathID(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	if err := h.Personal.Delete(middleware.CurrentUserID(c), ownerID, planID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
