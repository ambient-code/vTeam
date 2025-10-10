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
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Expose server globals for backward compatibility
var (
	k8sClient      *kubernetes.Clientset
	dynamicClient  dynamic.Interface
	namespace      string
	stateBaseDir   string
	pvcBaseDir     string
	baseKubeConfig *rest.Config
)

// getK8sClientsForRequest is a wrapper for the handlers package version
func getK8sClientsForRequest(c *gin.Context) (*kubernetes.Clientset, dynamic.Interface) {
	return handlers.GetK8sClientsForRequest(c)
}

func main() {
	// Load environment from .env in development if present
	_ = godotenv.Overload(".env.local")
	_ = godotenv.Overload(".env")

	// Initialize components
	github.InitializeTokenManager()

	if err := server.InitK8sClients(); err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}

	server.InitConfig()

	// Sync server globals to main package globals for backward compatibility
	k8sClient = server.K8sClient
	dynamicClient = server.DynamicClient
	namespace = server.Namespace
	stateBaseDir = server.StateBaseDir
	pvcBaseDir = server.PvcBaseDir
	baseKubeConfig = server.BaseKubeConfig

	// Initialize git package
	git.GetProjectSettingsResource = k8s.GetProjectSettingsResource
	git.GetGitHubInstallation = func(ctx context.Context, userID string) (interface{}, error) {
		return github.GetInstallation(ctx, userID)
	}
	git.GitHubTokenManager = github.Manager

	// Initialize CRD package
	crd.GetRFEWorkflowResource = k8s.GetRFEWorkflowResource

	// Initialize content handlers
	handlers.StateBaseDir = stateBaseDir
	handlers.GitPushRepo = git.PushRepo
	handlers.GitAbandonRepo = git.AbandonRepo
	handlers.GitDiffRepo = git.DiffRepo

	// Initialize GitHub auth handlers
	handlers.K8sClient = k8sClient
	handlers.Namespace = namespace
	handlers.GithubTokenManager = github.Manager

	// Initialize project handlers
	handlers.GetOpenShiftProjectResource = k8s.GetOpenShiftProjectResource

	// Initialize session handlers
	handlers.GetAgenticSessionV1Alpha1Resource = k8s.GetAgenticSessionV1Alpha1Resource
	handlers.DynamicClient = dynamicClient
	handlers.ParseStatus = parseStatus
	handlers.GetGitHubToken = git.GetGitHubToken
	handlers.DeriveRepoFolderFromURL = git.DeriveRepoFolderFromURL

	// Initialize RFE workflow handlers
	handlers.GetRFEWorkflowResource = k8s.GetRFEWorkflowResource
	handlers.UpsertProjectRFEWorkflowCR = crd.UpsertProjectRFEWorkflowCR
	handlers.PerformRepoSeeding = performRepoSeeding
	handlers.CheckRepoSeeding = checkRepoSeeding
	handlers.RfeFromUnstructured = jira.RFEFromUnstructured

	// Initialize Jira handler
	jiraHandler := &jira.Handler{
		GetK8sClientsForRequest:    getK8sClientsForRequest,
		GetProjectSettingsResource: k8s.GetProjectSettingsResource,
		GetRFEWorkflowResource:     k8s.GetRFEWorkflowResource,
	}

	// Initialize repo handlers
	handlers.GetK8sClientsForRequestRepo = getK8sClientsForRequest
	handlers.GetGitHubTokenRepo = git.GetGitHubToken

	// Initialize middleware
	handlers.BaseKubeConfig = baseKubeConfig
	handlers.K8sClientMw = k8sClient

	// Initialize websocket package
	websocket.StateBaseDir = stateBaseDir

	// Content service mode
	if os.Getenv("CONTENT_SERVICE_MODE") == "true" {
		if err := server.RunContentService(registerContentRoutes); err != nil {
			log.Fatalf("Content service error: %v", err)
		}
		return
	}

	// Normal server mode - create closure to capture jiraHandler
	registerRoutesWithJira := func(r *gin.Engine) {
		registerRoutes(r, jiraHandler)
	}
	if err := server.Run(registerRoutesWithJira); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

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

		api.GET("/projects", handlers.ListProjects)
		api.POST("/projects", handlers.CreateProject)
		api.GET("/projects/:projectName", handlers.GetProject)
		api.PUT("/projects/:projectName", handlers.UpdateProject)
		api.DELETE("/projects/:projectName", handlers.DeleteProject)
	}

	// Health check endpoint
	r.GET("/health", handlers.Health)
}

// Type aliases to types package - preserves backward compatibility
type AgenticSession = types.AgenticSession
type AgenticSessionSpec = types.AgenticSessionSpec
type AgenticSessionStatus = types.AgenticSessionStatus
type CreateAgenticSessionRequest = types.CreateAgenticSessionRequest
type CloneSessionRequest = types.CloneSessionRequest
type NamedGitRepo = types.NamedGitRepo
type OutputNamedGitRepo = types.OutputNamedGitRepo
type SessionRepoMapping = types.SessionRepoMapping
type LLMSettings = types.LLMSettings
type GitRepository = types.GitRepository
type GitConfig = types.GitConfig
type Paths = types.Paths
type RFEWorkflow = types.RFEWorkflow
type WorkflowJiraLink = types.WorkflowJiraLink
type CreateRFEWorkflowRequest = types.CreateRFEWorkflowRequest
type AdvancePhaseRequest = types.AdvancePhaseRequest
type UserContext = types.UserContext
type BotAccountRef = types.BotAccountRef
type ResourceOverrides = types.ResourceOverrides
type AmbientProject = types.AmbientProject
type CreateProjectRequest = types.CreateProjectRequest

func parseStatus(status map[string]interface{}) *AgenticSessionStatus {
	result := &AgenticSessionStatus{}

	if phase, ok := status["phase"].(string); ok {
		result.Phase = phase
	}

	if message, ok := status["message"].(string); ok {
		result.Message = message
	}

	if startTime, ok := status["startTime"].(string); ok {
		result.StartTime = &startTime
	}

	if completionTime, ok := status["completionTime"].(string); ok {
		result.CompletionTime = &completionTime
	}

	if jobName, ok := status["jobName"].(string); ok {
		result.JobName = jobName
	}

	// New: result summary fields (top-level in status)
	if st, ok := status["subtype"].(string); ok {
		result.Subtype = st
	}

	if ie, ok := status["is_error"].(bool); ok {
		result.IsError = ie
	}
	if nt, ok := status["num_turns"].(float64); ok {
		result.NumTurns = int(nt)
	}
	if sid, ok := status["session_id"].(string); ok {
		result.SessionID = sid
	}
	if tcu, ok := status["total_cost_usd"].(float64); ok {
		result.TotalCostUSD = &tcu
	}
	if usage, ok := status["usage"].(map[string]interface{}); ok {
		result.Usage = usage
	}
	if res, ok := status["result"].(string); ok {
		result.Result = &res
	}

	if stateDir, ok := status["stateDir"].(string); ok {
		result.StateDir = stateDir
	}

	return result
}

// Adapter types to implement git package interfaces for RFEWorkflow
type rfeWorkflowAdapter struct {
	wf *RFEWorkflow
}

type gitRepositoryAdapter struct {
	repo *types.GitRepository
}

// Wrapper for git.PerformRepoSeeding that adapts *RFEWorkflow to git.Workflow interface
func performRepoSeeding(ctx context.Context, wf *RFEWorkflow, githubToken, agentURL, agentBranch, agentPath, specKitVersion, specKitTemplate string) error {
	return git.PerformRepoSeeding(ctx, &rfeWorkflowAdapter{wf: wf}, githubToken, agentURL, agentBranch, agentPath, specKitVersion, specKitTemplate)
}

// GetUmbrellaRepo implements git.Workflow interface
func (r *rfeWorkflowAdapter) GetUmbrellaRepo() git.GitRepo {
	if r.wf.UmbrellaRepo == nil {
		return nil
	}
	return &gitRepositoryAdapter{repo: r.wf.UmbrellaRepo}
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
