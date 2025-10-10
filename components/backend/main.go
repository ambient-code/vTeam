package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"ambient-code-backend/git"
	"ambient-code-backend/handlers"
	"ambient-code-backend/jira"
	"ambient-code-backend/server"
	"ambient-code-backend/types"
	"ambient-code-backend/websocket"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	initializeGitHubTokenManager()

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
	git.GetProjectSettingsResource = getProjectSettingsResource
	git.GetGitHubInstallation = func(ctx context.Context, userID string) (interface{}, error) {
		return getGitHubInstallation(ctx, userID)
	}
	git.GitHubTokenManager = githubTokenManager

	// Initialize content handlers
	handlers.StateBaseDir = stateBaseDir
	handlers.GitPushRepo = git.PushRepo
	handlers.GitAbandonRepo = git.AbandonRepo
	handlers.GitDiffRepo = git.DiffRepo

	// Initialize GitHub auth handlers
	handlers.K8sClient = k8sClient
	handlers.Namespace = namespace
	handlers.GithubTokenManager = githubTokenManager

	// Initialize project handlers
	handlers.GetOpenShiftProjectResource = getOpenShiftProjectResource

	// Initialize session handlers
	handlers.GetAgenticSessionV1Alpha1Resource = getAgenticSessionV1Alpha1Resource
	handlers.DynamicClient = dynamicClient
	handlers.ParseStatus = parseStatus
	handlers.GetGitHubToken = git.GetGitHubToken
	handlers.DeriveRepoFolderFromURL = git.DeriveRepoFolderFromURL

	// Initialize RFE workflow handlers
	handlers.GetRFEWorkflowResource = getRFEWorkflowResource
	handlers.UpsertProjectRFEWorkflowCR = upsertProjectRFEWorkflowCR
	handlers.PerformRepoSeeding = performRepoSeeding
	handlers.CheckRepoSeeding = checkRepoSeeding
	handlers.RfeFromUnstructured = jira.RFEFromUnstructured

	// Initialize Jira handler
	jiraHandler := &jira.Handler{
		GetK8sClientsForRequest:    getK8sClientsForRequest,
		GetProjectSettingsResource: getProjectSettingsResource,
		GetRFEWorkflowResource:     getRFEWorkflowResource,
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

// getAgenticSessionV1Alpha1Resource returns the GroupVersionResource for AgenticSession v1alpha1
func getAgenticSessionV1Alpha1Resource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "agenticsessions",
	}
}

// getProjectSettingsResource returns the GroupVersionResource for ProjectSettings
func getProjectSettingsResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "projectsettings",
	}
}

// getRFEWorkflowResource returns the GroupVersionResource for RFEWorkflow CRD
func getRFEWorkflowResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "rfeworkflows",
	}
}

// ===== CRD helpers for project-scoped RFE workflows =====

func rfeWorkflowToCRObject(workflow *RFEWorkflow) map[string]interface{} {
	// Build spec
	spec := map[string]interface{}{
		"title":         workflow.Title,
		"description":   workflow.Description,
		"workspacePath": workflow.WorkspacePath,
	}
	if len(workflow.JiraLinks) > 0 {
		links := make([]map[string]interface{}, 0, len(workflow.JiraLinks))
		for _, l := range workflow.JiraLinks {
			links = append(links, map[string]interface{}{"path": l.Path, "jiraKey": l.JiraKey})
		}
		spec["jiraLinks"] = links
	}
	if workflow.ParentOutcome != nil && *workflow.ParentOutcome != "" {
		spec["parentOutcome"] = *workflow.ParentOutcome
	}

	// Prefer umbrellaRepo/supportingRepos; fallback to legacy repositories array
	if workflow.UmbrellaRepo != nil {
		u := map[string]interface{}{"url": workflow.UmbrellaRepo.URL}
		if workflow.UmbrellaRepo.Branch != nil {
			u["branch"] = *workflow.UmbrellaRepo.Branch
		}
		spec["umbrellaRepo"] = u
	}
	if len(workflow.SupportingRepos) > 0 {
		items := make([]map[string]interface{}, 0, len(workflow.SupportingRepos))
		for _, r := range workflow.SupportingRepos {
			rm := map[string]interface{}{"url": r.URL}
			if r.Branch != nil {
				rm["branch"] = *r.Branch
			}
			items = append(items, rm)
		}
		spec["supportingRepos"] = items
	}

	labels := map[string]string{
		"project":      workflow.Project,
		"rfe-workflow": workflow.ID,
	}

	return map[string]interface{}{
		"apiVersion": "vteam.ambient-code/v1alpha1",
		"kind":       "RFEWorkflow",
		"metadata": map[string]interface{}{
			"name":      workflow.ID,
			"namespace": workflow.Project,
			"labels":    labels,
		},
		"spec": spec,
	}
}

func upsertProjectRFEWorkflowCR(dyn dynamic.Interface, workflow *RFEWorkflow) error {
	if workflow.Project == "" {
		// Only manage CRD for project-scoped workflows
		return nil
	}
	if dyn == nil {
		return fmt.Errorf("no dynamic client provided")
	}
	gvr := getRFEWorkflowResource()
	obj := &unstructured.Unstructured{Object: rfeWorkflowToCRObject(workflow)}
	// Try create, if exists then update
	_, err := dyn.Resource(gvr).Namespace(workflow.Project).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			_, uerr := dyn.Resource(gvr).Namespace(workflow.Project).Update(context.TODO(), obj, v1.UpdateOptions{})
			if uerr != nil {
				return fmt.Errorf("failed to update RFEWorkflow CR: %v", uerr)
			}
			return nil
		}
		return fmt.Errorf("failed to create RFEWorkflow CR: %v", err)
	}
	return nil
}

// getOpenShiftProjectResource returns the GroupVersionResource for OpenShift Project
func getOpenShiftProjectResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "project.openshift.io",
		Version:  "v1",
		Resource: "projects",
	}
}

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

// Adapter types for git package interfaces
type repoAdapter struct {
	wf *RFEWorkflow
}

type gitRepoAdapter struct {
	repo *types.GitRepository
}

type gitRepo interface {
	GetURL() string
	GetBranch() *string
}

// Wrapper for git.PerformRepoSeeding that accepts *RFEWorkflow
func performRepoSeeding(ctx context.Context, wf *RFEWorkflow, githubToken, agentURL, agentBranch, agentPath, specKitVersion, specKitTemplate string) error {
	adapter := &repoAdapter{wf: wf}
	return git.PerformRepoSeeding(ctx, adapter, githubToken, agentURL, agentBranch, agentPath, specKitVersion, specKitTemplate)
}

// GetUmbrellaRepo implements the workflow interface for git.PerformRepoSeeding
func (r *repoAdapter) GetUmbrellaRepo() gitRepo {
	if r.wf.UmbrellaRepo == nil {
		return nil
	}
	return &gitRepoAdapter{repo: r.wf.UmbrellaRepo}
}

func (g *gitRepoAdapter) GetURL() string {
	if g.repo == nil {
		return ""
	}
	return g.repo.URL
}

func (g *gitRepoAdapter) GetBranch() *string {
	if g.repo == nil {
		return nil
	}
	return g.repo.Branch
}

// Wrapper for git.CheckRepoSeeding
func checkRepoSeeding(ctx context.Context, repoURL string, branch *string, githubToken string) (bool, map[string]interface{}, error) {
	return git.CheckRepoSeeding(ctx, repoURL, branch, githubToken)
}
