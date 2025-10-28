package main

import (
	"ambient-code-backend/handlers"
	"ambient-code-backend/jira"
	"ambient-code-backend/websocket"

	"github.com/gin-gonic/gin"
)

func registerContentRoutes(r *gin.Engine) {
	r.POST("/content/write", handlers.ContentWrite)
	r.GET("/content/file", handlers.ContentRead)
	r.GET("/content/list", handlers.ContentList)
	r.POST("/content/github/push", handlers.ContentGitPush)
	r.POST("/content/github/abandon", handlers.ContentGitAbandon)
	r.GET("/content/github/diff", handlers.ContentGitDiff)
}

func registerRoutes(r *gin.Engine, jiraHandler *jira.Handler) {
	// API routes
	api := r.Group("/api")
	{
		api.POST("/projects/:projectName/agentic-sessions/:sessionName/github/token", handlers.MintSessionGitHubToken)

		projectGroup := api.Group("/projects/:projectName", handlers.ValidateProjectContext())
		{
			projectGroup.GET("/access", handlers.AccessCheck)
			projectGroup.GET("/users/forks", handlers.ListUserForks)
			projectGroup.POST("/users/forks", handlers.CreateUserFork)

			projectGroup.GET("/repo/tree", handlers.GetRepoTree)
			projectGroup.GET("/repo/blob", handlers.GetRepoBlob)

			projectGroup.GET("/agentic-sessions", handlers.ListSessions)
			projectGroup.POST("/agentic-sessions", handlers.CreateSession)
			projectGroup.GET("/agentic-sessions/:sessionName", handlers.GetSession)
			projectGroup.PUT("/agentic-sessions/:sessionName", handlers.UpdateSession)
			projectGroup.DELETE("/agentic-sessions/:sessionName", handlers.DeleteSession)
			projectGroup.POST("/agentic-sessions/:sessionName/clone", handlers.CloneSession)
			projectGroup.POST("/agentic-sessions/:sessionName/start", handlers.StartSession)
			projectGroup.POST("/agentic-sessions/:sessionName/stop", handlers.StopSession)
			projectGroup.PUT("/agentic-sessions/:sessionName/status", handlers.UpdateSessionStatus)
			projectGroup.GET("/agentic-sessions/:sessionName/workspace", handlers.ListSessionWorkspace)
			projectGroup.GET("/agentic-sessions/:sessionName/workspace/*path", handlers.GetSessionWorkspaceFile)
			projectGroup.PUT("/agentic-sessions/:sessionName/workspace/*path", handlers.PutSessionWorkspaceFile)
			projectGroup.POST("/agentic-sessions/:sessionName/github/push", handlers.PushSessionRepo)
			projectGroup.POST("/agentic-sessions/:sessionName/github/abandon", handlers.AbandonSessionRepo)
			projectGroup.GET("/agentic-sessions/:sessionName/github/diff", handlers.DiffSessionRepo)

			projectGroup.GET("/rfe-workflows", handlers.ListProjectRFEWorkflows)
			projectGroup.POST("/rfe-workflows", handlers.CreateProjectRFEWorkflow)
			projectGroup.GET("/rfe-workflows/:id", handlers.GetProjectRFEWorkflow)
			projectGroup.GET("/rfe-workflows/:id/summary", handlers.GetProjectRFEWorkflowSummary)
			projectGroup.DELETE("/rfe-workflows/:id", handlers.DeleteProjectRFEWorkflow)
			projectGroup.POST("/rfe-workflows/:id/seed", handlers.SeedProjectRFEWorkflow)
			projectGroup.GET("/rfe-workflows/:id/check-seeding", handlers.CheckProjectRFEWorkflowSeeding)
			projectGroup.GET("/rfe-workflows/:id/agents", handlers.GetProjectRFEWorkflowAgents)

			projectGroup.GET("/sessions/:sessionId/ws", websocket.HandleSessionWebSocket)
			projectGroup.GET("/sessions/:sessionId/messages", websocket.GetSessionMessagesWS)
			projectGroup.POST("/sessions/:sessionId/messages", websocket.PostSessionMessageWS)
			projectGroup.POST("/rfe-workflows/:id/jira", jiraHandler.PublishWorkflowFileToJira)
			projectGroup.GET("/rfe-workflows/:id/jira", handlers.GetWorkflowJira)
			projectGroup.GET("/rfe-workflows/:id/sessions", handlers.ListProjectRFEWorkflowSessions)
			projectGroup.POST("/rfe-workflows/:id/sessions/link", handlers.AddProjectRFEWorkflowSession)
			projectGroup.DELETE("/rfe-workflows/:id/sessions/:sessionName", handlers.RemoveProjectRFEWorkflowSession)

			projectGroup.GET("/permissions", handlers.ListProjectPermissions)
			projectGroup.POST("/permissions", handlers.AddProjectPermission)
			projectGroup.DELETE("/permissions/:subjectType/:subjectName", handlers.RemoveProjectPermission)

			projectGroup.GET("/keys", handlers.ListProjectKeys)
			projectGroup.POST("/keys", handlers.CreateProjectKey)
			projectGroup.DELETE("/keys/:keyId", handlers.DeleteProjectKey)

			projectGroup.GET("/secrets", handlers.ListNamespaceSecrets)
			projectGroup.GET("/runner-secrets/config", handlers.GetRunnerSecretsConfig)
			projectGroup.PUT("/runner-secrets/config", handlers.UpdateRunnerSecretsConfig)
			projectGroup.GET("/runner-secrets", handlers.ListRunnerSecrets)
			projectGroup.PUT("/runner-secrets", handlers.UpdateRunnerSecrets)
		}

		api.POST("/auth/github/install", handlers.LinkGitHubInstallationGlobal)
		api.GET("/auth/github/status", handlers.GetGitHubStatusGlobal)
		api.POST("/auth/github/disconnect", handlers.DisconnectGitHubGlobal)
		api.GET("/auth/github/user/callback", handlers.HandleGitHubUserOAuthCallback)

		// Cluster info endpoint (public, no auth required)
		api.GET("/cluster-info", handlers.GetClusterInfo)

		api.GET("/projects", handlers.ListProjects)
		api.POST("/projects", handlers.CreateProject)
		api.GET("/projects/:projectName", handlers.GetProject)
		api.PUT("/projects/:projectName", handlers.UpdateProject)
		api.DELETE("/projects/:projectName", handlers.DeleteProject)
	}

	// Health check endpoint
	r.GET("/health", handlers.Health)
}
