package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/service"
)

// Handler 聚合各业务服务，供路由使用。
type Handler struct {
	Auth       *service.AuthService
	Project    *service.ProjectService
	SubProject *service.SubProjectService
	Issue      *service.IssueService
	Plan       *service.PlanService
	Task       *service.TaskService
	Personal   *service.PersonalPlanService
	Sync       *service.SyncService
	Prefs      *service.PreferenceService
}

func NewHandler(auth *service.AuthService, project *service.ProjectService, subProject *service.SubProjectService, issue *service.IssueService, plan *service.PlanService, task *service.TaskService, personal *service.PersonalPlanService, sync *service.SyncService, prefs *service.PreferenceService) *Handler {
	return &Handler{Auth: auth, Project: project, SubProject: subProject, Issue: issue, Plan: plan, Task: task, Personal: personal, Sync: sync, Prefs: prefs}
}

// respondError 把 service 层错误统一映射为 HTTP 状态码。
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrLastAdmin):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}
