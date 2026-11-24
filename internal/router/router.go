package router

import (
	"github.com/gin-gonic/gin"

	"consiliumous-backend/internal/config"
	"consiliumous-backend/internal/handler"
	"consiliumous-backend/internal/middleware"
)

// New 组装路由。认证中间件只挂在需要登录的路由组上。
func New(h *handler.Handler, cfg *config.Config) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(cfg.CorsAllowedOrigins))

	// 无需登录
	r.GET("/health", h.Health)
	r.GET("/version", h.Version)

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
		}

		// 需要登录
		authed := api.Group("", middleware.Auth(cfg.JWTSecret))
		{
			authed.GET("/me", h.Me)
			authed.GET("/users/search", h.SearchUsers)

			projects := authed.Group("/projects")
			{
				projects.POST("", h.CreateProject)
				projects.GET("", h.ListProjects)
				projects.GET("/:id", h.GetProject)
				projects.POST("/:id/archive", h.ArchiveProject)
				projects.POST("/:id/unarchive", h.UnarchiveProject)
				projects.DELETE("/:id", h.DeleteProject)
				projects.GET("/:id/members", h.ListMembers)
				projects.POST("/:id/members", h.AddMember)
				projects.DELETE("/:id/members/:userId", h.RemoveMember)
				projects.POST("/:id/admins/:userId/revoke", h.RevokeAdmin)

				// 子项目（两级）
				projects.GET("/:id/subprojects", h.ListSubProjects)
				projects.POST("/:id/subprojects", h.CreateSubProject)
				projects.PUT("/:id/subprojects/:spId", h.UpdateSubProject)
				projects.DELETE("/:id/subprojects/:spId", h.DeleteSubProject)

				// Issue 池
				projects.GET("/:id/issues", h.ListIssues)
				projects.POST("/:id/issues", h.CreateIssue)
				projects.GET("/:id/issues/:issueId", h.GetIssue)
				projects.PUT("/:id/issues/:issueId", h.UpdateIssue)
				projects.DELETE("/:id/issues/:issueId", h.DeleteIssue)
				projects.POST("/:id/issues/:issueId/status", h.ChangeIssueStatus)
				projects.POST("/:id/issues/:issueId/comments", h.AddIssueComment)
				projects.GET("/:id/issues/:issueId/comments", h.ListIssueComments)

				// 排期与计划
				projects.POST("/:id/schedule", h.Schedule)
				projects.GET("/:id/plans", h.ListPlans)
				projects.POST("/:id/plans", h.CreatePlan)
				projects.GET("/:id/plans/:planId", h.GetPlan)
				projects.PUT("/:id/plans/:planId", h.UpdatePlan)
				projects.PUT("/:id/plans/:planId/owner", h.AssignPlanOwner)
				projects.PUT("/:id/plans/:planId/issue", h.LinkPlanIssue)
				projects.DELETE("/:id/plans/:planId/issue", h.UnlinkPlanIssue)
				projects.PATCH("/:id/plans/:planId/state", h.UpdatePlanState)
				projects.POST("/:id/plans/:planId/cancel", h.CancelPlan)
				projects.DELETE("/:id/plans/:planId", h.DeletePlan)
				projects.POST("/:id/plans/:planId/progress/restore-auto", h.RestorePlanProgress)

				// 计划任务
				projects.GET("/:id/plans/:planId/tasks", h.ListTasks)
				projects.POST("/:id/plans/:planId/tasks", h.CreateTask)
				projects.PUT("/:id/plans/:planId/tasks/:taskId", h.UpdateTask)
				projects.DELETE("/:id/plans/:planId/tasks/:taskId", h.DeleteTask)
				projects.PATCH("/:id/plans/:planId/tasks/:taskId/status", h.UpdateTaskStatus)

				// 视图
				projects.GET("/:id/mindmap", h.ProjectMindmap)
			}

			// 个人空间
			authed.GET("/me/plans", h.ListMyPlans)
			authed.POST("/me/plans", h.CreateMyPlan)
			authed.GET("/me/plans/:planId", h.GetMyPlan)
			authed.PUT("/me/plans/:planId", h.UpdateMyPlan)
			authed.DELETE("/me/plans/:planId", h.DeleteMyPlan)
			authed.PATCH("/me/plans/:planId/state", h.UpdateMyPlanState)
			authed.GET("/me/assigned-plans", h.ListMyAssignedPlans)
			authed.GET("/me/plan-groups", h.ListMyPlanGroups)
			authed.POST("/me/plan-groups", h.CreateMyPlanGroup)
			authed.PUT("/me/plan-groups/:groupId", h.UpdateMyPlanGroup)
			authed.DELETE("/me/plan-groups/:groupId", h.DeleteMyPlanGroup)
			authed.GET("/me/preferences", h.GetPreferences)
			authed.PUT("/me/preferences/:key", h.SetPreference)

			// 移动端同步
			authed.POST("/sync/plans", h.SyncUpload)
			authed.POST("/sync/bind", h.SyncBind)

			// 系统管理员
			admin := authed.Group("/admin")
			{
				admin.GET("/users/:userId/plans", h.AdminListUserPlans)
				admin.PUT("/users/:userId/plans/:planId", h.AdminUpdateUserPlan)
				admin.DELETE("/users/:userId/plans/:planId", h.AdminDeleteUserPlan)
			}
		}
	}

	return r
}
