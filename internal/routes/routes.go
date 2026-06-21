package routes

import (
	"bugyou-backend/internal/constants"
	"bugyou-backend/internal/controllers"
	"bugyou-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine) {
	api := router.Group("/api")

	// ── Public auth routes ────────────────────────────────────────────────────
	auth := api.Group("/auth")
	auth.POST("/register", controllers.Register)
	auth.POST("/login", controllers.Login)
	auth.GET("/me", middleware.AuthRequired(), controllers.Me)

	// ── Issue routes (all authenticated users) ────────────────────────────────
	issues := api.Group("/issues")
	issues.POST("", middleware.AuthRequired(), controllers.CreateIssue)
	issues.GET("/my", middleware.AuthRequired(), controllers.MyIssues)
	issues.GET("/track/:ticketId", middleware.AuthRequired(), controllers.TrackIssue)

	// ── Developer + Admin shared routes ───────────────────────────────────────
	devAdmin := api.Group("/developer")
	devAdmin.Use(middleware.AuthRequired(), middleware.RequireRole(constants.RoleDeveloper, constants.RoleAdmin))
	devAdmin.GET("/dashboard", controllers.Dashboard)
	devAdmin.GET("/issues", controllers.AllIssues)
	devAdmin.PATCH("/issues/:id/status", controllers.UpdateStatus)
	devAdmin.GET("/reminders", controllers.Reminders)
	devAdmin.GET("/analytics/issue-trend", controllers.IssueTrend)

	// Developer-only routes (self-assign, my-tasks)
	devOnly := api.Group("/developer")
	devOnly.Use(middleware.AuthRequired(), middleware.RequireRole(constants.RoleDeveloper, constants.RoleAdmin))
	devOnly.GET("/issues/my-tasks", controllers.MyTasks)
	devOnly.PATCH("/issues/:id/assign-self", controllers.SelfAssign)

	// ── Admin-only routes ─────────────────────────────────────────────────────
	admin := api.Group("/admin")
	// Seed endpoint is public (protected only by ADMIN_SECRET env var)
	admin.POST("/seed", controllers.SeedAdmin)

	// All other admin routes require auth + admin role
	adminProtected := api.Group("/admin")
	adminProtected.Use(middleware.AuthRequired(), middleware.RequireRole(constants.RoleAdmin))
	adminProtected.POST("/create-developer", controllers.CreateDeveloper)
	adminProtected.GET("/developers", controllers.ListDevelopers)
	adminProtected.PATCH("/issues/:id/assign", controllers.AssignDeveloper)
	adminProtected.DELETE("/issues/:id", controllers.DeleteIssue)
}
