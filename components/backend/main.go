package main

import (
	"context"
	"log"
	"os"

	"ambient-code-backend/crd"
	"ambient-code-backend/git"
	"ambient-code-backend/github"
	"ambient-code-backend/handlers"
	"ambient-code-backend/jira"
	"ambient-code-backend/k8s"
	"ambient-code-backend/server"
	"ambient-code-backend/types"
	"ambient-code-backend/websocket"

	"github.com/gin-gonic/gin"
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

	// Initialize CRD package
	crd.GetRFEWorkflowResource = k8s.GetRFEWorkflowResource

	// Initialize content handlers
	handlers.StateBaseDir = server.StateBaseDir
	handlers.GitPushRepo = git.PushRepo
	handlers.GitAbandonRepo = git.AbandonRepo
	handlers.GitDiffRepo = git.DiffRepo

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

	// Initialize RFE workflow handlers
	handlers.GetRFEWorkflowResource = k8s.GetRFEWorkflowResource
	handlers.UpsertProjectRFEWorkflowCR = crd.UpsertProjectRFEWorkflowCR
	handlers.PerformRepoSeeding = performRepoSeeding
	handlers.CheckRepoSeeding = checkRepoSeeding
	handlers.CheckBranchExists = checkBranchExists
	handlers.RfeFromUnstructured = jira.RFEFromUnstructured

	// Initialize Jira handler
	jiraHandler := &jira.Handler{
		GetK8sClientsForRequest:    handlers.GetK8sClientsForRequest,
		GetProjectSettingsResource: k8s.GetProjectSettingsResource,
		GetRFEWorkflowResource:     k8s.GetRFEWorkflowResource,
	}

	// Initialize repo handlers
	handlers.GetK8sClientsForRequestRepo = handlers.GetK8sClientsForRequest
	handlers.GetGitHubTokenRepo = git.GetGitHubToken

	// Initialize middleware
	handlers.BaseKubeConfig = server.BaseKubeConfig
	handlers.K8sClientMw = server.K8sClient

	// Initialize websocket package
	websocket.StateBaseDir = server.StateBaseDir

	// Normal server mode - create closure to capture jiraHandler
	registerRoutesWithJira := func(r *gin.Engine) {
		registerRoutes(r, jiraHandler)
	}
	if err := server.Run(registerRoutesWithJira); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// Adapter types to implement git package interfaces for RFEWorkflow
type rfeWorkflowAdapter struct {
	wf *types.RFEWorkflow
}

type gitRepositoryAdapter struct {
	repo *types.GitRepository
}

// Wrapper for git.PerformRepoSeeding that adapts *types.RFEWorkflow to git.Workflow interface
func performRepoSeeding(ctx context.Context, wf *types.RFEWorkflow, branchName, githubToken, agentURL, agentBranch, agentPath, specKitRepo, specKitVersion, specKitTemplate string) (bool, error) {
	return git.PerformRepoSeeding(ctx, &rfeWorkflowAdapter{wf: wf}, branchName, githubToken, agentURL, agentBranch, agentPath, specKitRepo, specKitVersion, specKitTemplate)
}

// GetUmbrellaRepo implements git.Workflow interface
func (r *rfeWorkflowAdapter) GetUmbrellaRepo() git.GitRepo {
	if r.wf.UmbrellaRepo == nil {
		return nil
	}
	return &gitRepositoryAdapter{repo: r.wf.UmbrellaRepo}
}

// GetSupportingRepos implements git.Workflow interface
func (r *rfeWorkflowAdapter) GetSupportingRepos() []git.GitRepo {
	if len(r.wf.SupportingRepos) == 0 {
		return nil
	}
	repos := make([]git.GitRepo, 0, len(r.wf.SupportingRepos))
	for i := range r.wf.SupportingRepos {
		repos = append(repos, &gitRepositoryAdapter{repo: &r.wf.SupportingRepos[i]})
	}
	return repos
}

// GetURL implements git.GitRepo interface
func (g *gitRepositoryAdapter) GetURL() string {
	if g.repo == nil {
		return ""
	}
	return g.repo.URL
}

// GetBranch implements git.GitRepo interface
func (g *gitRepositoryAdapter) GetBranch() *string {
	if g.repo == nil {
		return nil
	}
	return g.repo.Branch
}

// Wrapper for git.CheckRepoSeeding
func checkRepoSeeding(ctx context.Context, repoURL string, branch *string, githubToken string) (bool, map[string]interface{}, error) {
	return git.CheckRepoSeeding(ctx, repoURL, branch, githubToken)
}

// Wrapper for git.CheckBranchExists
func checkBranchExists(ctx context.Context, repoURL, branchName, githubToken string) (bool, error) {
	return git.CheckBranchExists(ctx, repoURL, branchName, githubToken)
}
