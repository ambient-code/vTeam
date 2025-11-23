package main

import (
	"ambient-code-backend/handlers"
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
	r.GET("/content/git-status", handlers.ContentGitStatus)
	r.POST("/content/git-configure-remote", handlers.ContentGitConfigureRemote)
	r.POST("/content/git-sync", handlers.ContentGitSync)
	r.GET("/content/workflow-metadata", handlers.ContentWorkflowMetadata)
	r.GET("/content/git-merge-status", handlers.ContentGitMergeStatus)
	r.POST("/content/git-pull", handlers.ContentGitPull)
	r.POST("/content/git-push", handlers.ContentGitPushToBranch)
	r.POST("/content/git-create-branch", handlers.ContentGitCreateBranch)
	r.GET("/content/git-list-branches", handlers.ContentGitListBranches)
}

func registerRoutes(r *gin.Engine) {
	// API routes
	api := r.Group("/api")
	{
		// Public endpoints (no auth required)
		api.GET("/workflows/ootb", handlers.ListOOTBWorkflows)

		api.POST("/projects/:projectName/agentic-sessions/:sessionName/github/token", handlers.MintSessionGitHubToken)

		projectGroup := api.Group("/projects/:projectName", handlers.ValidateProjectContext())
		{
			projectGroup.GET("/access", handlers.AccessCheck)
			projectGroup.GET("/users/forks", handlers.ListUserForks)
			projectGroup.POST("/users/forks", handlers.CreateUserFork)

			projectGroup.GET("/repo/tree", handlers.GetRepoTree)
			projectGroup.GET("/repo/blob", handlers.GetRepoBlob)
			projectGroup.GET("/repo/branches", handlers.ListRepoBranches)
			projectGroup.GET("/repo/seed-status", handlers.GetRepoSeedStatus)
			projectGroup.POST("/repo/seed", handlers.SeedRepositoryEndpoint)

			projectGroup.GET("/agentic-sessions", handlers.ListSessions)
			projectGroup.POST("/agentic-sessions", handlers.CreateSession)
			projectGroup.GET("/agentic-sessions/:sessionName", handlers.GetSession)
			projectGroup.PUT("/agentic-sessions/:sessionName", handlers.UpdateSession)
			projectGroup.PATCH("/agentic-sessions/:sessionName", handlers.PatchSession)
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
			projectGroup.GET("/agentic-sessions/:sessionName/git/status", handlers.GetGitStatus)
			projectGroup.POST("/agentic-sessions/:sessionName/git/configure-remote", handlers.ConfigureGitRemote)
			projectGroup.POST("/agentic-sessions/:sessionName/git/synchronize", handlers.SynchronizeGit)
			projectGroup.GET("/agentic-sessions/:sessionName/git/merge-status", handlers.GetGitMergeStatus)
			projectGroup.POST("/agentic-sessions/:sessionName/git/pull", handlers.GitPullSession)
			projectGroup.POST("/agentic-sessions/:sessionName/git/push", handlers.GitPushSession)
			projectGroup.POST("/agentic-sessions/:sessionName/git/create-branch", handlers.GitCreateBranchSession)
			projectGroup.GET("/agentic-sessions/:sessionName/git/list-branches", handlers.GitListBranchesSession)
			projectGroup.GET("/agentic-sessions/:sessionName/k8s-resources", handlers.GetSessionK8sResources)
			projectGroup.POST("/agentic-sessions/:sessionName/spawn-content-pod", handlers.SpawnContentPod)
			projectGroup.GET("/agentic-sessions/:sessionName/content-pod-status", handlers.GetContentPodStatus)
			projectGroup.DELETE("/agentic-sessions/:sessionName/content-pod", handlers.DeleteContentPod)
			projectGroup.POST("/agentic-sessions/:sessionName/workflow", handlers.SelectWorkflow)
			projectGroup.GET("/agentic-sessions/:sessionName/workflow/metadata", handlers.GetWorkflowMetadata)
			projectGroup.POST("/agentic-sessions/:sessionName/repos", handlers.AddRepo)
			projectGroup.DELETE("/agentic-sessions/:sessionName/repos/:repoName", handlers.RemoveRepo)

			projectGroup.GET("/sessions/:sessionId/ws", websocket.HandleSessionWebSocket)
			projectGroup.GET("/sessions/:sessionId/messages", websocket.GetSessionMessagesWS)
			// Removed: /messages/claude-format - Using SDK's built-in resume with persisted ~/.claude state
			projectGroup.POST("/sessions/:sessionId/messages", websocket.PostSessionMessageWS)

			projectGroup.GET("/permissions", handlers.ListProjectPermissions)
			projectGroup.POST("/permissions", handlers.AddProjectPermission)
			projectGroup.DELETE("/permissions/:subjectType/:subjectName", handlers.RemoveProjectPermission)

			projectGroup.GET("/keys", handlers.ListProjectKeys)
			projectGroup.POST("/keys", handlers.CreateProjectKey)
			projectGroup.DELETE("/keys/:keyId", handlers.DeleteProjectKey)

			projectGroup.GET("/secrets", handlers.ListNamespaceSecrets)
			projectGroup.GET("/runner-secrets", handlers.ListRunnerSecrets)
			projectGroup.PUT("/runner-secrets", handlers.UpdateRunnerSecrets)
			projectGroup.GET("/integration-secrets", handlers.ListIntegrationSecrets)
			projectGroup.PUT("/integration-secrets", handlers.UpdateIntegrationSecrets)

			// GitLab authentication endpoints (project-scoped)
			projectGroup.POST("/auth/gitlab/connect", handlers.ConnectGitLabGlobal)
			projectGroup.GET("/auth/gitlab/status", handlers.GetGitLabStatusGlobal)
			projectGroup.POST("/auth/gitlab/disconnect", handlers.DisconnectGitLabGlobal)
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
