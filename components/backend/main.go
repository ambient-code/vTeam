package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	k8sClient      *kubernetes.Clientset
	namespace      string
	stateBaseDir   string
	pvcBaseDir     string
	baseKubeConfig *rest.Config
)

func main() {
	// Initialize Kubernetes clients
	if err := initK8sClients(); err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}

	// Get namespace from environment or use default
	namespace = os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// Get state storage base directory
	stateBaseDir = os.Getenv("STATE_BASE_DIR")
	if stateBaseDir == "" {
		stateBaseDir = "/data/state"
	}

	// Get PVC base directory for RFE workspaces
	pvcBaseDir = os.Getenv("PVC_BASE_DIR")
	if pvcBaseDir == "" {
		pvcBaseDir = "/workspace"
	}

	// Project-scoped storage; no global preload required

	// Setup Gin router
	r := gin.Default()

	// Middleware to populate user context from forwarded headers
	r.Use(forwardedIdentityMiddleware())

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Content service mode: expose minimal file APIs for per-namespace writer service
	if os.Getenv("CONTENT_SERVICE_MODE") == "true" {
		r.POST("/content/write", contentWrite)
		r.GET("/content/file", contentRead)
		r.GET("/content/list", contentList)
	}

	// API routes (all consolidated under /api) remain available
	api := r.Group("/api")
	{
		// Legacy non-project agentic session routes removed

		// RFE workflows are project-scoped only (legacy non-project routes removed)
		// Project-scoped routes for multi-tenant session management
		projectGroup := api.Group("/projects/:projectName", validateProjectContext())
		{
			// Access check (SSAR based)
			projectGroup.GET("/access", accessCheck)
			// Agentic sessions under a project
			projectGroup.GET("/agentic-sessions", listSessions)
			projectGroup.POST("/agentic-sessions", createSession)
			projectGroup.GET("/agentic-sessions/:sessionName", getSession)
			projectGroup.PUT("/agentic-sessions/:sessionName", updateSession)
			projectGroup.DELETE("/agentic-sessions/:sessionName", deleteSession)
			projectGroup.POST("/agentic-sessions/:sessionName/clone", cloneSession)
			projectGroup.POST("/agentic-sessions/:sessionName/start", startSession)
			projectGroup.POST("/agentic-sessions/:sessionName/stop", stopSession)
			projectGroup.PUT("/agentic-sessions/:sessionName/status", updateSessionStatus)
			projectGroup.PUT("/agentic-sessions/:sessionName/displayname", updateSessionDisplayName)
			projectGroup.GET("/agentic-sessions/:sessionName/messages", getSessionMessages)
			projectGroup.POST("/agentic-sessions/:sessionName/messages", postSessionMessage)
			// Session workspace APIs
			projectGroup.GET("/agentic-sessions/:sessionName/workspace", getSessionWorkspace)
			projectGroup.GET("/agentic-sessions/:sessionName/workspace/*path", getSessionWorkspaceFile)
			projectGroup.PUT("/agentic-sessions/:sessionName/workspace/*path", putSessionWorkspaceFile)

			// RFE workflow endpoints (project-scoped)
			projectGroup.GET("/rfe-workflows", listProjectRFEWorkflows)
			projectGroup.POST("/rfe-workflows", createProjectRFEWorkflow)
			projectGroup.GET("/rfe-workflows/:id", getProjectRFEWorkflow)
			projectGroup.GET("/rfe-workflows/:id/summary", getProjectRFEWorkflowSummary)
			projectGroup.DELETE("/rfe-workflows/:id", deleteProjectRFEWorkflow)
			// Workflow workspace APIs
			projectGroup.GET("/rfe-workflows/:id/workspace", getRFEWorkflowWorkspace)
			projectGroup.GET("/rfe-workflows/:id/workspace/*path", getRFEWorkflowWorkspaceFile)
			projectGroup.PUT("/rfe-workflows/:id/workspace/*path", putRFEWorkflowWorkspaceFile)
			// Publish a workspace file to Jira and record linkage on the CR
			projectGroup.POST("/rfe-workflows/:id/jira", publishWorkflowFileToJira)
			projectGroup.GET("/rfe-workflows/:id/jira", getWorkflowJira)
			// Sessions linkage within an RFE
			projectGroup.GET("/rfe-workflows/:id/sessions", listProjectRFEWorkflowSessions)
			projectGroup.POST("/rfe-workflows/:id/sessions", addProjectRFEWorkflowSession)
			projectGroup.DELETE("/rfe-workflows/:id/sessions/:sessionName", removeProjectRFEWorkflowSession)

			// Agents
			projectGroup.GET("/agents", listAgents)
			projectGroup.GET("/agents/:persona/markdown", getAgentMarkdown)

			// Permissions (users & groups)
			projectGroup.GET("/permissions", listProjectPermissions)
			projectGroup.POST("/permissions", addProjectPermission)
			projectGroup.DELETE("/permissions/:subjectType/:subjectName", removeProjectPermission)

			// Project access keys
			projectGroup.GET("/keys", listProjectKeys)
			projectGroup.POST("/keys", createProjectKey)
			projectGroup.DELETE("/keys/:keyId", deleteProjectKey)

			// Runner secrets configuration and CRUD
			projectGroup.GET("/secrets", listNamespaceSecrets)
			projectGroup.GET("/runner-secrets/config", getRunnerSecretsConfig)
			projectGroup.PUT("/runner-secrets/config", updateRunnerSecretsConfig)
			projectGroup.GET("/runner-secrets", listRunnerSecrets)
			projectGroup.PUT("/runner-secrets", updateRunnerSecrets)
		}

		// Project management (cluster-wide)
		api.GET("/projects", listProjects)
		api.POST("/projects", createProject)
		api.GET("/projects/:projectName", getProject)
		api.PUT("/projects/:projectName", updateProject)
		api.DELETE("/projects/:projectName", deleteProject)
	}

	// Metrics endpoint
	r.GET("/metrics", getMetrics)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Using namespace: %s", namespace)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initK8sClients() error {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	if config, err = rest.InClusterConfig(); err != nil {
		// If in-cluster config fails, try kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
		}

		if config, err = clientcmd.BuildConfigFromFlags("", kubeconfig); err != nil {
			return fmt.Errorf("failed to create Kubernetes config: %v", err)
		}
	}

	// Create standard Kubernetes client
	k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Save base config for per-request impersonation/user-token clients
	baseKubeConfig = config

	return nil
}

// forwardedIdentityMiddleware populates Gin context from common OAuth proxy headers
func forwardedIdentityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if v := c.GetHeader("X-Forwarded-User"); v != "" {
			c.Set("userID", v)
		}
		// Prefer preferred username; fallback to user id
		name := c.GetHeader("X-Forwarded-Preferred-Username")
		if name == "" {
			name = c.GetHeader("X-Forwarded-User")
		}
		if name != "" {
			c.Set("userName", name)
		}
		if v := c.GetHeader("X-Forwarded-Email"); v != "" {
			c.Set("userEmail", v)
		}
		if v := c.GetHeader("X-Forwarded-Groups"); v != "" {
			c.Set("userGroups", strings.Split(v, ","))
		}
		// Also expose access token if present
		auth := c.GetHeader("Authorization")
		if auth != "" {
			c.Set("authorizationHeader", auth)
		}
		if v := c.GetHeader("X-Forwarded-Access-Token"); v != "" {
			c.Set("forwardedAccessToken", v)
		}
		c.Next()
	}
}

// AgenticSession represents the structure of our custom resource
type AgenticSession struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       AgenticSessionSpec     `json:"spec"`
	Status     *AgenticSessionStatus  `json:"status,omitempty"`
}

type AgenticSessionSpec struct {
	Prompt            string             `json:"prompt" binding:"required"`
	Interactive       bool               `json:"interactive,omitempty"`
	DisplayName       string             `json:"displayName"`
	LLMSettings       LLMSettings        `json:"llmSettings"`
	Timeout           int                `json:"timeout"`
	UserContext       *UserContext       `json:"userContext,omitempty"`
	BotAccount        *BotAccountRef     `json:"botAccount,omitempty"`
	ResourceOverrides *ResourceOverrides `json:"resourceOverrides,omitempty"`
	Project           string             `json:"project,omitempty"`
	GitConfig         *GitConfig         `json:"gitConfig,omitempty"`
	Paths             *Paths             `json:"paths,omitempty"`
}

type LLMSettings struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

type GitUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GitAuthentication struct {
	SSHKeySecret *string `json:"sshKeySecret,omitempty"`
	TokenSecret  *string `json:"tokenSecret,omitempty"`
}

type GitRepository struct {
	URL       string  `json:"url"`
	Branch    *string `json:"branch,omitempty"`
	ClonePath *string `json:"clonePath,omitempty"`
}

type GitConfig struct {
	User           *GitUser           `json:"user,omitempty"`
	Authentication *GitAuthentication `json:"authentication,omitempty"`
	Repositories   []GitRepository    `json:"repositories,omitempty"`
}

type Paths struct {
	Workspace string `json:"workspace,omitempty"`
	Messages  string `json:"messages,omitempty"`
	Inbox     string `json:"inbox,omitempty"`
}

type AgenticSessionStatus struct {
	Phase          string  `json:"phase,omitempty"`
	Message        string  `json:"message,omitempty"`
	StartTime      *string `json:"startTime,omitempty"`
	CompletionTime *string `json:"completionTime,omitempty"`
	JobName        string  `json:"jobName,omitempty"`
	StateDir       string  `json:"stateDir,omitempty"`
	// Result summary fields from runner
	Subtype      string                 `json:"subtype,omitempty"`
	IsError      bool                   `json:"is_error,omitempty"`
	NumTurns     int                    `json:"num_turns,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	TotalCostUSD *float64               `json:"total_cost_usd,omitempty"`
	Usage        map[string]interface{} `json:"usage,omitempty"`
	Result       *string                `json:"result,omitempty"`
}

type CreateAgenticSessionRequest struct {
	Prompt               string             `json:"prompt" binding:"required"`
	DisplayName          string             `json:"displayName,omitempty"`
	LLMSettings          *LLMSettings       `json:"llmSettings,omitempty"`
	Timeout              *int               `json:"timeout,omitempty"`
	Interactive          *bool              `json:"interactive,omitempty"`
	WorkspacePath        string             `json:"workspacePath,omitempty"`
	GitConfig            *GitConfig         `json:"gitConfig,omitempty"`
	UserContext          *UserContext       `json:"userContext,omitempty"`
	BotAccount           *BotAccountRef     `json:"botAccount,omitempty"`
	ResourceOverrides    *ResourceOverrides `json:"resourceOverrides,omitempty"`
	EnvironmentVariables map[string]string  `json:"environmentVariables,omitempty"`
	Labels               map[string]string  `json:"labels,omitempty"`
	Annotations          map[string]string  `json:"annotations,omitempty"`
}

type CloneSessionRequest struct {
	TargetProject  string `json:"targetProject" binding:"required"`
	NewSessionName string `json:"newSessionName" binding:"required"`
}

// RFE Workflow Data Structures
type RFEWorkflow struct {
	ID            string             `json:"id"`
	Title         string             `json:"title"`
	Description   string             `json:"description"`
	Repositories  []GitRepository    `json:"repositories,omitempty"`
	Project       string             `json:"project,omitempty"`
	WorkspacePath string             `json:"workspacePath"`
	CreatedAt     string             `json:"createdAt"`
	UpdatedAt     string             `json:"updatedAt"`
	JiraLinks     []WorkflowJiraLink `json:"jiraLinks,omitempty"`
}

type WorkflowJiraLink struct {
	Path    string `json:"path"`
	JiraKey string `json:"jiraKey"`
}

type CreateRFEWorkflowRequest struct {
	Title         string          `json:"title" binding:"required"`
	Description   string          `json:"description" binding:"required"`
	Repositories  []GitRepository `json:"repositories,omitempty"`
	WorkspacePath string          `json:"workspacePath,omitempty"`
}

type AdvancePhaseRequest struct {
	Force bool `json:"force,omitempty"` // Force advance even if current phase isn't complete
}

// New types for multi-tenant support
type UserContext struct {
	UserID      string   `json:"userId" binding:"required"`
	DisplayName string   `json:"displayName" binding:"required"`
	Groups      []string `json:"groups" binding:"required"`
}

type BotAccountRef struct {
	Name string `json:"name" binding:"required"`
}

type ResourceOverrides struct {
	CPU           string `json:"cpu,omitempty"`
	Memory        string `json:"memory,omitempty"`
	StorageClass  string `json:"storageClass,omitempty"`
	PriorityClass string `json:"priorityClass,omitempty"`
}

// Project management types
type AmbientProject struct {
	Name              string            `json:"name"`
	DisplayName       string            `json:"displayName"`
	Description       string            `json:"description,omitempty"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	CreationTimestamp string            `json:"creationTimestamp"`
	Status            string            `json:"status"`
}

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
	Description string `json:"description,omitempty"`
}

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

	if len(workflow.Repositories) > 0 {
		repos := make([]map[string]interface{}, 0, len(workflow.Repositories))
		for _, r := range workflow.Repositories {
			rm := map[string]interface{}{
				"url": r.URL,
			}
			if r.Branch != nil {
				rm["branch"] = *r.Branch
			}
			if r.ClonePath != nil {
				rm["clonePath"] = *r.ClonePath
			}
			repos = append(repos, rm)
		}
		spec["repositories"] = repos
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
