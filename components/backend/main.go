package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"ambient-code-backend/handlers"
	"ambient-code-backend/server"
	"ambient-code-backend/types"

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

	// Initialize content handlers
	handlers.StateBaseDir = stateBaseDir
	handlers.GitPushRepo = gitPushRepo
	handlers.GitAbandonRepo = gitAbandonRepo
	handlers.GitDiffRepo = gitDiffRepo

	// Content service mode
	if os.Getenv("CONTENT_SERVICE_MODE") == "true" {
		if err := server.RunContentService(registerContentRoutes); err != nil {
			log.Fatalf("Content service error: %v", err)
		}
		return
	}

	// Normal server mode
	if err := server.Run(registerRoutes); err != nil {
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

func registerRoutes(r *gin.Engine) {
	// API routes
	api := r.Group("/api")
	{
		api.POST("/projects/:projectName/agentic-sessions/:sessionName/github/token", mintSessionGitHubToken)

		projectGroup := api.Group("/projects/:projectName", validateProjectContext())
		{
			projectGroup.GET("/access", accessCheck)
			projectGroup.GET("/users/forks", listUserForks)
			projectGroup.POST("/users/forks", createUserFork)

			projectGroup.GET("/repo/tree", getRepoTree)
			projectGroup.GET("/repo/blob", getRepoBlob)

			projectGroup.GET("/agentic-sessions", listSessions)
			projectGroup.POST("/agentic-sessions", createSession)
			projectGroup.GET("/agentic-sessions/:sessionName", getSession)
			projectGroup.PUT("/agentic-sessions/:sessionName", updateSession)
			projectGroup.DELETE("/agentic-sessions/:sessionName", deleteSession)
			projectGroup.POST("/agentic-sessions/:sessionName/clone", cloneSession)
			projectGroup.POST("/agentic-sessions/:sessionName/start", startSession)
			projectGroup.POST("/agentic-sessions/:sessionName/stop", stopSession)
			projectGroup.PUT("/agentic-sessions/:sessionName/status", updateSessionStatus)
			projectGroup.GET("/agentic-sessions/:sessionName/workspace", listSessionWorkspace)
			projectGroup.GET("/agentic-sessions/:sessionName/workspace/*path", getSessionWorkspaceFile)
			projectGroup.PUT("/agentic-sessions/:sessionName/workspace/*path", putSessionWorkspaceFile)
			projectGroup.POST("/agentic-sessions/:sessionName/github/push", pushSessionRepo)
			projectGroup.POST("/agentic-sessions/:sessionName/github/abandon", abandonSessionRepo)
			projectGroup.GET("/agentic-sessions/:sessionName/github/diff", diffSessionRepo)

			projectGroup.GET("/rfe-workflows", listProjectRFEWorkflows)
			projectGroup.POST("/rfe-workflows", createProjectRFEWorkflow)
			projectGroup.GET("/rfe-workflows/:id", getProjectRFEWorkflow)
			projectGroup.GET("/rfe-workflows/:id/summary", getProjectRFEWorkflowSummary)
			projectGroup.DELETE("/rfe-workflows/:id", deleteProjectRFEWorkflow)
			projectGroup.POST("/rfe-workflows/:id/seed", seedProjectRFEWorkflow)
			projectGroup.GET("/rfe-workflows/:id/check-seeding", checkProjectRFEWorkflowSeeding)

			projectGroup.GET("/sessions/:sessionId/ws", handleSessionWebSocket)
			projectGroup.GET("/sessions/:sessionId/messages", getSessionMessagesWS)
			projectGroup.POST("/sessions/:sessionId/messages", postSessionMessageWS)
			projectGroup.POST("/rfe-workflows/:id/jira", publishWorkflowFileToJira)
			projectGroup.GET("/rfe-workflows/:id/jira", getWorkflowJira)
			projectGroup.GET("/rfe-workflows/:id/sessions", listProjectRFEWorkflowSessions)
			projectGroup.POST("/rfe-workflows/:id/sessions/link", addProjectRFEWorkflowSession)
			projectGroup.DELETE("/rfe-workflows/:id/sessions/:sessionName", removeProjectRFEWorkflowSession)

			projectGroup.GET("/permissions", listProjectPermissions)
			projectGroup.POST("/permissions", addProjectPermission)
			projectGroup.DELETE("/permissions/:subjectType/:subjectName", removeProjectPermission)

			projectGroup.GET("/keys", listProjectKeys)
			projectGroup.POST("/keys", createProjectKey)
			projectGroup.DELETE("/keys/:keyId", deleteProjectKey)

			projectGroup.GET("/secrets", listNamespaceSecrets)
			projectGroup.GET("/runner-secrets/config", getRunnerSecretsConfig)
			projectGroup.PUT("/runner-secrets/config", updateRunnerSecretsConfig)
			projectGroup.GET("/runner-secrets", listRunnerSecrets)
			projectGroup.PUT("/runner-secrets", updateRunnerSecrets)
		}

		api.POST("/auth/github/install", linkGitHubInstallationGlobal)
		api.GET("/auth/github/status", getGitHubStatusGlobal)
		api.POST("/auth/github/disconnect", disconnectGitHubGlobal)
		api.GET("/auth/github/user/callback", handleGitHubUserOAuthCallback)

		api.GET("/projects", listProjects)
		api.POST("/projects", createProject)
		api.GET("/projects/:projectName", getProject)
		api.PUT("/projects/:projectName", updateProject)
		api.DELETE("/projects/:projectName", deleteProject)
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
