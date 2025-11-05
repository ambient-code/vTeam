package main

import (
	"context"
	"log"
	"os"

	"ambient-code-backend/git"
	"ambient-code-backend/github"
	"ambient-code-backend/handlers"
	"ambient-code-backend/k8s"
	"ambient-code-backend/server"
	"ambient-code-backend/websocket"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment from .env in development if present
	_ = godotenv.Overload(".env.local")
	_ = godotenv.Overload(".env")

	// Content service mode - minimal initialization, no K8s access needed
	if os.Getenv("CONTENT_SERVICE_MODE") == "true" {
		log.Println("Starting in CONTENT_SERVICE_MODE (no K8s client initialization)")

		// Initialize config to set StateBaseDir from environment
		server.InitConfig()

		// Only initialize what content service needs
		handlers.StateBaseDir = server.StateBaseDir
		handlers.GitPushRepo = git.PushRepo
		handlers.GitAbandonRepo = git.AbandonRepo
		handlers.GitDiffRepo = git.DiffRepo
		handlers.GitCheckMergeStatus = git.CheckMergeStatus
		handlers.GitPullRepo = git.PullRepo
		handlers.GitPushToRepo = git.PushToRepo
		handlers.GitCreateBranch = git.CreateBranch
		handlers.GitListRemoteBranches = git.ListRemoteBranches

		log.Printf("Content service using StateBaseDir: %s", server.StateBaseDir)

		if err := server.RunContentService(registerContentRoutes); err != nil {
			log.Fatalf("Content service error: %v", err)
		}
		return
	}

	// Normal server mode - full initialization
	log.Println("Starting in normal server mode with K8s client initialization")

	// Initialize components
	github.InitializeTokenManager()

	if err := server.InitK8sClients(); err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}

	server.InitConfig()

	// Initialize git package
	git.GetProjectSettingsResource = k8s.GetProjectSettingsResource
	git.GetGitHubInstallation = func(ctx context.Context, userID string) (interface{}, error) {
		return github.GetInstallation(ctx, userID)
	}
	git.GitHubTokenManager = github.Manager
	git.GetBackendNamespace = func() string {
		return server.Namespace
	}

	// Initialize content handlers
	handlers.StateBaseDir = server.StateBaseDir
	handlers.GitPushRepo = git.PushRepo
	handlers.GitAbandonRepo = git.AbandonRepo
	handlers.GitDiffRepo = git.DiffRepo
	handlers.GitCheckMergeStatus = git.CheckMergeStatus
	handlers.GitPullRepo = git.PullRepo
	handlers.GitPushToRepo = git.PushToRepo
	handlers.GitCreateBranch = git.CreateBranch
	handlers.GitListRemoteBranches = git.ListRemoteBranches

	// Initialize GitHub auth handlers
	handlers.K8sClient = server.K8sClient
	handlers.Namespace = server.Namespace
	handlers.GithubTokenManager = github.Manager

	// Initialize project handlers
	handlers.GetOpenShiftProjectResource = k8s.GetOpenShiftProjectResource
	handlers.K8sClientProjects = server.K8sClient         // Backend SA client for namespace operations
	handlers.DynamicClientProjects = server.DynamicClient // Backend SA dynamic client for Project operations

	// Initialize session handlers
	handlers.GetAgenticSessionV1Alpha1Resource = k8s.GetAgenticSessionV1Alpha1Resource
	handlers.DynamicClient = server.DynamicClient
	handlers.GetGitHubToken = git.GetGitHubToken
	handlers.DeriveRepoFolderFromURL = git.DeriveRepoFolderFromURL
	handlers.SendMessageToSession = websocket.SendMessageToSession

	// Initialize repo handlers
	handlers.GetK8sClientsForRequestRepo = handlers.GetK8sClientsForRequest
	handlers.GetGitHubTokenRepo = git.GetGitHubToken

	// Initialize middleware
	handlers.BaseKubeConfig = server.BaseKubeConfig
	handlers.K8sClientMw = server.K8sClient

	// Initialize websocket package
	websocket.StateBaseDir = server.StateBaseDir

	// Normal server mode
	if err := server.Run(registerRoutes); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
