package routes

import (
	"ambient-code-backend/internal/config"
	"ambient-code-backend/internal/handlers"
	"ambient-code-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all routes for the application
func SetupRoutes(appConfig *config.AppConfig) *gin.Engine {
	r := gin.Default()

	// Setup middleware
	r.Use(middleware.ForwardedIdentityMiddleware())
	middleware.SetupCORS(r)

	// Content service mode: expose minimal file APIs for per-namespace writer service
	if config.IsContentServiceMode() {
		r.POST("/content/write", handlers.ContentWrite)
		r.GET("/content/file", handlers.ContentRead)
		r.GET("/content/list", handlers.ContentList)
	}

	// API routes (all consolidated under /api) remain available
	api := r.Group("/api")
	{
		// Project-scoped routes for multi-tenant session management
		projectGroup := api.Group("/projects/:projectName", middleware.ValidateProjectContext())
		{
			// Access check (SSAR based)
			projectGroup.GET("/access", handlers.AccessCheck)
			// Agentic sessions under a project
			projectGroup.GET("/agentic-sessions", handlers.ListSessions)
			projectGroup.POST("/agentic-sessions", handlers.CreateSession)
			projectGroup.GET("/agentic-sessions/:sessionName", handlers.GetSession)
			projectGroup.PUT("/agentic-sessions/:sessionName", handlers.UpdateSession)
			projectGroup.DELETE("/agentic-sessions/:sessionName", handlers.DeleteSession)
			projectGroup.POST("/agentic-sessions/:sessionName/clone", handlers.CloneSession)
			projectGroup.POST("/agentic-sessions/:sessionName/start", handlers.StartSession)
			projectGroup.POST("/agentic-sessions/:sessionName/stop", handlers.StopSession)
			projectGroup.PUT("/agentic-sessions/:sessionName/status", handlers.UpdateSessionStatus)
			projectGroup.PUT("/agentic-sessions/:sessionName/displayname", handlers.UpdateSessionDisplayName)
			projectGroup.GET("/agentic-sessions/:sessionName/messages", handlers.GetSessionMessages)
			projectGroup.POST("/agentic-sessions/:sessionName/messages", handlers.PostSessionMessage)
			// Session workspace APIs
			projectGroup.GET("/agentic-sessions/:sessionName/workspace", handlers.GetSessionWorkspace)
			projectGroup.GET("/agentic-sessions/:sessionName/workspace/*path", handlers.GetSessionWorkspaceFile)
			projectGroup.PUT("/agentic-sessions/:sessionName/workspace/*path", handlers.PutSessionWorkspaceFile)

			// RFE workflow endpoints (project-scoped)
			projectGroup.GET("/rfe-workflows", handlers.ListProjectRFEWorkflows)
			projectGroup.POST("/rfe-workflows", handlers.CreateProjectRFEWorkflow)
			projectGroup.GET("/rfe-workflows/:id", handlers.GetProjectRFEWorkflow)
			projectGroup.GET("/rfe-workflows/:id/summary", handlers.GetProjectRFEWorkflowSummary)
			projectGroup.DELETE("/rfe-workflows/:id", handlers.DeleteProjectRFEWorkflow)
			// Workflow workspace APIs
			projectGroup.GET("/rfe-workflows/:id/workspace", handlers.GetRFEWorkflowWorkspace)
			projectGroup.GET("/rfe-workflows/:id/workspace/*path", handlers.GetRFEWorkflowWorkspaceFile)
			projectGroup.PUT("/rfe-workflows/:id/workspace/*path", handlers.PutRFEWorkflowWorkspaceFile)
			// Publish a workspace file to Jira and record linkage on the CR
			projectGroup.POST("/rfe-workflows/:id/jira", handlers.PublishWorkflowFileToJira)
			projectGroup.GET("/rfe-workflows/:id/jira", handlers.GetWorkflowJira)
			// Sessions linkage within an RFE
			projectGroup.GET("/rfe-workflows/:id/sessions", handlers.ListProjectRFEWorkflowSessions)
			projectGroup.POST("/rfe-workflows/:id/sessions", handlers.AddProjectRFEWorkflowSession)
			projectGroup.DELETE("/rfe-workflows/:id/sessions/:sessionName", handlers.RemoveProjectRFEWorkflowSession)

			// Agents
			projectGroup.GET("/agents", handlers.ListAgents)
			projectGroup.GET("/agents/:persona/markdown", handlers.GetAgentMarkdown)

			// Permissions (users & groups)
			projectGroup.GET("/permissions", handlers.ListProjectPermissions)
			projectGroup.POST("/permissions", handlers.AddProjectPermission)
			projectGroup.DELETE("/permissions/:subjectType/:subjectName", handlers.RemoveProjectPermission)

			// Project access keys
			projectGroup.GET("/keys", handlers.ListProjectKeys)
			projectGroup.POST("/keys", handlers.CreateProjectKey)
			projectGroup.DELETE("/keys/:keyId", handlers.DeleteProjectKey)

			// Runner secrets configuration and CRUD
			projectGroup.GET("/secrets", handlers.ListNamespaceSecrets)
			projectGroup.GET("/runner-secrets/config", handlers.GetRunnerSecretsConfig)
			projectGroup.PUT("/runner-secrets/config", handlers.UpdateRunnerSecretsConfig)
			projectGroup.GET("/runner-secrets", handlers.ListRunnerSecrets)
			projectGroup.PUT("/runner-secrets", handlers.UpdateRunnerSecrets)
		}

		// Project management (cluster-wide)
		api.GET("/projects", handlers.ListProjects)
		api.POST("/projects", handlers.CreateProject)
		api.GET("/projects/:projectName", handlers.GetProject)
		api.PUT("/projects/:projectName", handlers.UpdateProject)
		api.DELETE("/projects/:projectName", handlers.DeleteProject)
	}

	// Metrics endpoint
	r.GET("/metrics", handlers.GetMetrics)

	// Health check endpoint
	r.GET("/health", handlers.HealthCheck)

	return r
}