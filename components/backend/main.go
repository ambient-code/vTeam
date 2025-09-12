package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	dynamicClient  dynamic.Interface
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

	// Load existing RFE workflows from persistent storage
	if err := loadAllRFEWorkflows(); err != nil {
		log.Printf("⚠️ Failed to load RFE workflows: %v", err)
	}

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
	}

	// API routes (all consolidated under /api) remain available
	api := r.Group("/api")
	{
		// Legacy agentic sessions (keep for backwards compatibility)
		api.GET("/agentic-sessions", listAgenticSessions)
		api.GET("/agentic-sessions/:name", getAgenticSession)
		api.POST("/agentic-sessions", createAgenticSession)
		api.DELETE("/agentic-sessions/:name", deleteAgenticSession)
		api.PUT("/agentic-sessions/:name/status", updateAgenticSessionStatus)
		api.PUT("/agentic-sessions/:name/displayname", updateAgenticSessionDisplayName)
		api.POST("/agentic-sessions/:name/stop", stopAgenticSession)

		// RFE workflow endpoints
		api.GET("/rfe-workflows", listRFEWorkflows)
		api.POST("/rfe-workflows", createRFEWorkflow)
		api.GET("/rfe-workflows/:id", getRFEWorkflow)
		api.DELETE("/rfe-workflows/:id", deleteRFEWorkflow)
		api.POST("/rfe-workflows/:id/pause", pauseRFEWorkflow)
		api.POST("/rfe-workflows/:id/resume", resumeRFEWorkflow)
		api.POST("/rfe-workflows/:id/advance-phase", advanceRFEWorkflowPhase)
		api.POST("/rfe-workflows/:id/push-to-git", pushRFEWorkflowToGit)
		api.POST("/rfe-workflows/:id/scan-artifacts", scanRFEWorkflowArtifacts)
		api.GET("/rfe-workflows/:id/artifacts/*path", getRFEWorkflowArtifact)
		api.PUT("/rfe-workflows/:id/artifacts/*path", updateRFEWorkflowArtifact)
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
	WebsiteURL        string             `json:"websiteURL" binding:"required,url"`
	DisplayName       string             `json:"displayName"`
	LLMSettings       LLMSettings        `json:"llmSettings"`
	Timeout           int                `json:"timeout"`
	UserContext       *UserContext       `json:"userContext,omitempty"`
	BotAccount        *BotAccountRef     `json:"botAccount,omitempty"`
	ResourceOverrides *ResourceOverrides `json:"resourceOverrides,omitempty"`
	Project           string             `json:"project,omitempty"`
	GitConfig         *GitConfig         `json:"gitConfig,omitempty"`
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

type MessageObject struct {
	Content        string `json:"content,omitempty"`
	ToolUseID      string `json:"tool_use_id,omitempty"`
	ToolUseName    string `json:"tool_use_name,omitempty"`
	ToolUseInput   string `json:"tool_use_input,omitempty"`
	ToolUseIsError *bool  `json:"tool_use_is_error,omitempty"`
}

type AgenticSessionStatus struct {
	Phase          string          `json:"phase,omitempty"`
	Message        string          `json:"message,omitempty"`
	StartTime      *string         `json:"startTime,omitempty"`
	CompletionTime *string         `json:"completionTime,omitempty"`
	JobName        string          `json:"jobName,omitempty"`
	FinalOutput    string          `json:"finalOutput,omitempty"`
	Cost           *float64        `json:"cost,omitempty"`
	Messages       []MessageObject `json:"messages,omitempty"`
	StateDir       string          `json:"stateDir,omitempty"`
	ArtifactsCount int             `json:"artifactsCount,omitempty"`
	MessagesCount  int             `json:"messagesCount,omitempty"`
}

type CreateAgenticSessionRequest struct {
	Prompt            string             `json:"prompt" binding:"required"`
	WebsiteURL        string             `json:"websiteURL" binding:"required,url"`
	DisplayName       string             `json:"displayName,omitempty"`
	LLMSettings       *LLMSettings       `json:"llmSettings,omitempty"`
	Timeout           *int               `json:"timeout,omitempty"`
	GitConfig         *GitConfig         `json:"gitConfig,omitempty"`
	UserContext       *UserContext       `json:"userContext,omitempty"`
	BotAccount        *BotAccountRef     `json:"botAccount,omitempty"`
	ResourceOverrides *ResourceOverrides `json:"resourceOverrides,omitempty"`
}

// RFE Workflow Data Structures
type RFEWorkflow struct {
	ID               string                 `json:"id"`
	Title            string                 `json:"title"`
	Description      string                 `json:"description"`
	Status           string                 `json:"status"`       // "draft", "in_progress", "completed", "failed"
	CurrentPhase     string                 `json:"currentPhase"` // "specify", "plan", "tasks", "completed"
	SelectedAgents   []string               `json:"selectedAgents"`
	TargetRepoUrl    string                 `json:"targetRepoUrl"`
	TargetRepoBranch string                 `json:"targetRepoBranch"`
	GitUserName      *string                `json:"gitUserName,omitempty"`
	GitUserEmail     *string                `json:"gitUserEmail,omitempty"`
	CreatedAt        string                 `json:"createdAt"`
	UpdatedAt        string                 `json:"updatedAt"`
	AgentSessions    []RFEAgentSession      `json:"agentSessions"`
	Artifacts        []RFEArtifact          `json:"artifacts"`
	PhaseResults     map[string]PhaseResult `json:"phaseResults"` // "specify" -> result, "plan" -> result, etc.
}

type RFEAgentSession struct {
	ID           string   `json:"id"`
	AgentPersona string   `json:"agentPersona"` // e.g., "ENGINEERING_MANAGER"
	Phase        string   `json:"phase"`        // "specify", "plan", "tasks"
	Status       string   `json:"status"`       // "pending", "running", "completed", "failed"
	StartedAt    *string  `json:"startedAt,omitempty"`
	CompletedAt  *string  `json:"completedAt,omitempty"`
	Result       *string  `json:"result,omitempty"`
	Cost         *float64 `json:"cost,omitempty"`
}

type RFEArtifact struct {
	Path       string `json:"path"`
	Name       string `json:"name"`      // filename for display
	Type       string `json:"type"`      // "specification", "plan", "tasks", "code", "docs"
	Phase      string `json:"phase"`     // which phase created this artifact
	CreatedBy  string `json:"createdBy"` // which agent created this
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
}

type PhaseResult struct {
	Phase       string                 `json:"phase"`
	Status      string                 `json:"status"`    // "completed", "in_progress", "failed"
	Agents      []string               `json:"agents"`    // agents that worked on this phase
	Artifacts   []string               `json:"artifacts"` // artifact paths created in this phase
	Summary     string                 `json:"summary"`
	StartedAt   string                 `json:"startedAt"`
	CompletedAt *string                `json:"completedAt,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type CreateRFEWorkflowRequest struct {
	Title            string   `json:"title" binding:"required"`
	Description      string   `json:"description" binding:"required"`
	TargetRepoUrl    string   `json:"targetRepoUrl" binding:"required,url"`
	TargetRepoBranch string   `json:"targetRepoBranch" binding:"required"`
	SelectedAgents   []string `json:"selectedAgents" binding:"required,min=1"`
	GitUserName      *string  `json:"gitUserName,omitempty"`
	GitUserEmail     *string  `json:"gitUserEmail,omitempty"`
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
	// ProjectType removed
	// ResourceQuota removed
}

// ResourceQuota types removed

// ProjectSettings types
type ProjectSettings struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       ProjectSettingsSpec    `json:"spec"`
	Status     *ProjectSettingsStatus `json:"status,omitempty"`
}

type ProjectSettingsSpec struct {
	Project       string        `json:"project" binding:"required"`
	Bots          []BotConfig   `json:"bots,omitempty"`
	GroupAccess   []GroupAccess `json:"groupAccess,omitempty"`
	ResourceAvail ResourceAvail `json:"resourceAvailability"`
	Constraints   Constraints   `json:"constraints"`
	Integrations  Integrations  `json:"integrations"`
}

type BotConfig struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Token       string `json:"token,omitempty"`
}

type GroupAccess struct {
	GroupName   string   `json:"groupName" binding:"required"`
	Role        string   `json:"role" binding:"required"`
	Permissions []string `json:"permissions,omitempty"`
}

type ResourceAvail struct {
	Models          []string          `json:"models"`
	Features        []string          `json:"features"`
	ResourceLimits  map[string]string `json:"resourceLimits"`
	PriorityClasses []string          `json:"priorityClasses"`
}

type Constraints struct {
	MaxSessionsPerUser   int     `json:"maxSessionsPerUser"`
	MaxCostPerSession    float64 `json:"maxCostPerSession"`
	MaxCostPerUserPerDay float64 `json:"maxCostPerUserPerDay"`
	AllowSessionCloning  bool    `json:"allowSessionCloning"`
	AllowBotAccounts     bool    `json:"allowBotAccounts"`
}

type Integrations struct {
	Jira JiraIntegration `json:"jira"`
}

type JiraIntegration struct {
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhookUrl,omitempty"`
	Secret     string `json:"secret,omitempty"`
}

type ProjectSettingsStatus struct {
	Phase                string            `json:"phase,omitempty"`
	BotsCreated          int               `json:"botsCreated,omitempty"`
	GroupBindingsCreated int               `json:"groupBindingsCreated,omitempty"`
	LastReconciled       *string           `json:"lastReconciled,omitempty"`
	CurrentUsage         *ProjectUsage     `json:"currentUsage,omitempty"`
	Conditions           []StatusCondition `json:"conditions,omitempty"`
}

type ProjectUsage struct {
	ActiveSessions int     `json:"activeSessions"`
	TotalCostToday float64 `json:"totalCostToday"`
}

type StatusCondition struct {
	Type    string `json:"type" binding:"required"`
	Status  string `json:"status" binding:"required"`
	Reason  string `json:"reason" binding:"required"`
	Message string `json:"message" binding:"required"`
}

// Request types
type CloneSessionRequest struct {
	TargetProject  string `json:"targetProject" binding:"required"`
	NewSessionName string `json:"newSessionName" binding:"required"`
}

type JiraWebhookPayload struct {
	WebhookEvent string                 `json:"webhookEvent"`
	Issue        map[string]interface{} `json:"issue,omitempty"`
	User         map[string]interface{} `json:"user,omitempty"`
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

// getOpenShiftProjectResource returns the GroupVersionResource for OpenShift Project
func getOpenShiftProjectResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "project.openshift.io",
		Version:  "v1",
		Resource: "projects",
	}
}

// Removed legacy v1 handlers

// Helper functions for parsing moved to handlers.go to avoid duplication

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

	if finalOutput, ok := status["finalOutput"].(string); ok {
		result.FinalOutput = finalOutput
	}

	if cost, ok := status["cost"].(float64); ok {
		result.Cost = &cost
	}

	if messages, ok := status["messages"].([]interface{}); ok {
		result.Messages = make([]MessageObject, len(messages))
		for i, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				messageObj := MessageObject{}

				if content, ok := msgMap["content"].(string); ok {
					messageObj.Content = content
				}

				if toolUseID, ok := msgMap["tool_use_id"].(string); ok {
					messageObj.ToolUseID = toolUseID
				}

				if toolUseName, ok := msgMap["tool_use_name"].(string); ok {
					messageObj.ToolUseName = toolUseName
				}

				if toolUseInput, ok := msgMap["tool_use_input"].(string); ok {
					messageObj.ToolUseInput = toolUseInput
				}

				if toolUseIsError, ok := msgMap["tool_use_is_error"].(bool); ok {
					messageObj.ToolUseIsError = &toolUseIsError
				}

				result.Messages[i] = messageObj
			}
		}
	}

	if stateDir, ok := status["stateDir"].(string); ok {
		result.StateDir = stateDir
	}
	if ac, ok := status["artifactsCount"].(float64); ok {
		result.ArtifactsCount = int(ac)
	}
	if mc, ok := status["messagesCount"].(float64); ok {
		result.MessagesCount = int(mc)
	}

	return result
}

// RFE Workflow API Handlers

// In-memory storage for RFE workflows (with file-based persistence)
var rfeWorkflows = make(map[string]*RFEWorkflow)

// File paths for persistent storage
func getRFEWorkflowFilePath(id string) string {
	return filepath.Join(stateBaseDir, "rfe-workflows", id+".json")
}

func getRFEWorkflowsDir() string {
	return filepath.Join(stateBaseDir, "rfe-workflows")
}

// Save workflow to persistent storage
func saveRFEWorkflow(workflow *RFEWorkflow) error {
	workflowsDir := getRFEWorkflowsDir()
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflows directory: %v", err)
	}

	filePath := getRFEWorkflowFilePath(workflow.ID)
	data, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %v", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %v", err)
	}

	log.Printf("💾 Saved RFE workflow %s to disk", workflow.ID)
	return nil
}

// Load workflow from persistent storage
func loadRFEWorkflow(id string) (*RFEWorkflow, error) {
	filePath := getRFEWorkflowFilePath(id)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %v", err)
	}

	var workflow RFEWorkflow
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow: %v", err)
	}

	return &workflow, nil
}

// Load all workflows from persistent storage
func loadAllRFEWorkflows() error {
	workflowsDir := getRFEWorkflowsDir()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflows directory: %v", err)
	}

	files, err := ioutil.ReadDir(workflowsDir)
	if err != nil {
		return fmt.Errorf("failed to read workflows directory: %v", err)
	}

	loadedCount := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			id := strings.TrimSuffix(file.Name(), ".json")
			workflow, err := loadRFEWorkflow(id)
			if err != nil {
				log.Printf("⚠️ Failed to load workflow %s: %v", id, err)
				continue
			}
			rfeWorkflows[id] = workflow
			loadedCount++
		}
	}

	log.Printf("📂 Loaded %d RFE workflows from persistent storage", loadedCount)
	return nil
}

// Delete workflow from persistent storage
func deleteRFEWorkflowFile(id string) error {
	filePath := getRFEWorkflowFilePath(id)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete workflow file: %v", err)
	}
	log.Printf("🗑️ Deleted RFE workflow %s from disk", id)
	return nil
}

// Sync agent session statuses from Kubernetes AgenticSession resources
func syncAgentSessionStatuses(workflow *RFEWorkflow) error {
	if workflow == nil || len(workflow.AgentSessions) == 0 {
		return nil
	}

	// Define resource for AgenticSession
	gvr := getAgenticSessionResource()

	for i := range workflow.AgentSessions {
		session := &workflow.AgentSessions[i]
		sessionName := session.ID

		// Get the AgenticSession resource from Kubernetes
		item, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), sessionName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("AgenticSession %s not found in Kubernetes, keeping status as %s", sessionName, session.Status)
				continue
			}
			log.Printf("Failed to get AgenticSession %s: %v", sessionName, err)
			continue
		}

		// Parse the status from the AgenticSession resource
		if status, ok := item.Object["status"].(map[string]interface{}); ok {
			// Update phase status
			if phase, ok := status["phase"].(string); ok {
				// Map Kubernetes phase to our session status
				switch phase {
				case "Pending", "Creating":
					session.Status = "pending"
				case "Running":
					session.Status = "running"
				case "Completed":
					session.Status = "completed"
					// Set completion time if available
					if completionTime, ok := status["completionTime"].(string); ok {
						session.CompletedAt = &completionTime
					}
				case "Failed", "Error":
					session.Status = "failed"
					if completionTime, ok := status["completionTime"].(string); ok {
						session.CompletedAt = &completionTime
					}
				case "Stopped":
					session.Status = "failed" // Treat stopped as failed for UI purposes
					if completionTime, ok := status["completionTime"].(string); ok {
						session.CompletedAt = &completionTime
					}
				}
			}

			// Set start time if available and not already set
			if session.StartedAt == nil {
				if startTime, ok := status["startTime"].(string); ok {
					session.StartedAt = &startTime
				}
			}

			// Set cost if available
			if cost, ok := status["cost"].(float64); ok {
				session.Cost = &cost
			}
		}
	}

	// Save the updated workflow back to disk so the status persists
	err := saveRFEWorkflow(workflow)
	if err != nil {
		log.Printf("Failed to save workflow %s after syncing agent session statuses: %v", workflow.ID, err)
	}

	return nil
}

// RFE Workspace utility functions
func getRFEWorkspaceDir(workflowID string) string {
	return filepath.Join(pvcBaseDir, workflowID)
}

func getRFEGitRepoDir(workflowID string) string {
	return filepath.Join(getRFEWorkspaceDir(workflowID), "git-repo")
}

func getRFEAgentsDir(workflowID string) string {
	return filepath.Join(getRFEWorkspaceDir(workflowID), "agents")
}

func getRFEUIEditsDir(workflowID string) string {
	return filepath.Join(getRFEWorkspaceDir(workflowID), "ui-edits")
}

// Create workspace directory structure for RFE
func createRFEWorkspace(workflowID string) error {
	workspaceDir := getRFEWorkspaceDir(workflowID)

	// Create all required directories
	dirs := []string{
		workspaceDir,
		getRFEGitRepoDir(workflowID),
		getRFEAgentsDir(workflowID),
		filepath.Join(getRFEAgentsDir(workflowID), "specify"),
		filepath.Join(getRFEAgentsDir(workflowID), "plan"),
		filepath.Join(getRFEAgentsDir(workflowID), "tasks"),
		getRFEUIEditsDir(workflowID),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	log.Printf("📁 Created RFE workspace structure at %s", workspaceDir)
	return nil
}

// Get the full path to an artifact file in the workspace
func getRFEArtifactPath(workflowID, artifactPath string) string {
	// Check if it's in git-repo or agents directory
	if strings.HasPrefix(artifactPath, "git-repo/") {
		return filepath.Join(getRFEWorkspaceDir(workflowID), artifactPath)
	} else if strings.HasPrefix(artifactPath, "agents/") {
		return filepath.Join(getRFEWorkspaceDir(workflowID), artifactPath)
	} else {
		// Default to git-repo for backward compatibility
		return filepath.Join(getRFEGitRepoDir(workflowID), artifactPath)
	}
}

// Push workflow artifacts to Git repository
func pushWorkflowToGitRepo(workflow *RFEWorkflow) error {
	workspaceDir := getRFEWorkspaceDir(workflow.ID)
	gitRepoDir := getRFEGitRepoDir(workflow.ID)
	agentsDir := getRFEAgentsDir(workflow.ID)

	// Check if git repo directory exists, if not clone it
	if _, err := os.Stat(gitRepoDir); os.IsNotExist(err) {
		log.Printf("📥 Cloning repository %s to %s", workflow.TargetRepoUrl, gitRepoDir)

		// Clone the repository
		cloneCmd := exec.Command("git", "clone", "-b", workflow.TargetRepoBranch, workflow.TargetRepoUrl, gitRepoDir)
		cloneCmd.Dir = workspaceDir
		if output, err := cloneCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to clone repository: %v, output: %s", err, string(output))
		}
	}

	// Configure git user if provided
	if workflow.GitUserName != nil && *workflow.GitUserName != "" {
		configCmd := exec.Command("git", "config", "user.name", *workflow.GitUserName)
		configCmd.Dir = gitRepoDir
		if err := configCmd.Run(); err != nil {
			log.Printf("⚠️ Failed to set git user.name: %v", err)
		}
	}

	if workflow.GitUserEmail != nil && *workflow.GitUserEmail != "" {
		configCmd := exec.Command("git", "config", "user.email", *workflow.GitUserEmail)
		configCmd.Dir = gitRepoDir
		if err := configCmd.Run(); err != nil {
			log.Printf("⚠️ Failed to set git user.email: %v", err)
		}
	}

	// Pull latest changes
	pullCmd := exec.Command("git", "pull", "origin", workflow.TargetRepoBranch)
	pullCmd.Dir = gitRepoDir
	if output, err := pullCmd.CombinedOutput(); err != nil {
		log.Printf("⚠️ Failed to pull latest changes: %v, output: %s", err, string(output))
		// Continue anyway - might be the first push
	}

	// Create spec-kit compatible structure in git repo
	specsDir := filepath.Join(gitRepoDir, "specs", workflow.ID)
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		return fmt.Errorf("failed to create specs directory: %v", err)
	}

	// Convert agent outputs to spec-kit format
	if err := convertAgentOutputsToSpecKit(agentsDir, specsDir, workflow); err != nil {
		log.Printf("⚠️ Failed to convert agent outputs to spec-kit format: %v", err)
		// Continue anyway - copy raw outputs as fallback
		if _, err := os.Stat(agentsDir); err == nil {
			copyCmd := exec.Command("cp", "-r", agentsDir, specsDir)
			if output, err := copyCmd.CombinedOutput(); err != nil {
				log.Printf("⚠️ Failed to copy agents directory: %v, output: %s", err, string(output))
			}
		}
	}

	// Add all changes
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = gitRepoDir
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add changes: %v, output: %s", err, string(output))
	}

	// Check if there are changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = gitRepoDir
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %v", err)
	}

	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		log.Printf("ℹ️ No changes to commit for workflow %s", workflow.ID)
		return nil
	}

	// Commit changes with spec-kit compatible message
	commitMessage := fmt.Sprintf("Add %s phase for RFE %s: %s\n\nGenerated spec-kit compatible artifacts:\n- spec.md (feature specification)\n- plan.md (implementation plan)\n- tasks.md (task breakdown)\n\nPhase: %s\nAgents: %d sessions completed\n\n🤖 Generated with vTeam RFE System",
		workflow.CurrentPhase, workflow.ID, workflow.Title, workflow.CurrentPhase, len(workflow.AgentSessions))

	commitCmd := exec.Command("git", "commit", "-m", commitMessage)
	commitCmd.Dir = gitRepoDir
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit changes: %v, output: %s", err, string(output))
	}

	// Push changes
	pushCmd := exec.Command("git", "push", "origin", workflow.TargetRepoBranch)
	pushCmd.Dir = gitRepoDir
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push changes: %v, output: %s", err, string(output))
	}

	log.Printf("🚀 Successfully pushed RFE %s artifacts to %s", workflow.ID, workflow.TargetRepoUrl)
	return nil
}

// Convert agent outputs to spec-kit compatible format
func convertAgentOutputsToSpecKit(agentsDir, specsDir string, workflow *RFEWorkflow) error {
	// Read agent outputs by phase
	phases := []string{"specify", "plan", "tasks"}

	for _, phase := range phases {
		phaseDir := filepath.Join(agentsDir, phase)
		if _, err := os.Stat(phaseDir); os.IsNotExist(err) {
			continue // Skip phases that don't exist
		}

		// Consolidate all agent outputs for this phase
		var consolidatedContent strings.Builder

		// Add phase header
		phaseTitle := strings.Title(phase)
		consolidatedContent.WriteString(fmt.Sprintf("# %s Phase: %s\n\n", phaseTitle, workflow.Title))
		consolidatedContent.WriteString(fmt.Sprintf("**RFE ID**: %s  \n", workflow.ID))
		consolidatedContent.WriteString(fmt.Sprintf("**Repository**: %s  \n", workflow.TargetRepoUrl))
		consolidatedContent.WriteString(fmt.Sprintf("**Branch**: %s  \n\n", workflow.TargetRepoBranch))

		if phase == "specify" {
			consolidatedContent.WriteString("## Feature Specification\n\n")
			consolidatedContent.WriteString(fmt.Sprintf("**Description**: %s\n\n", workflow.Description))
		}

		// Read and combine all agent files in this phase
		files, err := ioutil.ReadDir(phaseDir)
		if err != nil {
			return fmt.Errorf("failed to read phase directory %s: %v", phaseDir, err)
		}

		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
				agentName := strings.TrimSuffix(file.Name(), ".md")
				agentTitle := strings.Title(strings.ReplaceAll(agentName, "-", " "))

				filePath := filepath.Join(phaseDir, file.Name())
				content, err := ioutil.ReadFile(filePath)
				if err != nil {
					log.Printf("⚠️ Failed to read agent file %s: %v", filePath, err)
					continue
				}

				// Add agent section
				consolidatedContent.WriteString(fmt.Sprintf("## %s Agent Output\n\n", agentTitle))
				consolidatedContent.WriteString(string(content))
				consolidatedContent.WriteString("\n\n---\n\n")
			}
		}

		// Determine output filename based on spec-kit conventions
		var outputFile string
		switch phase {
		case "specify":
			outputFile = "spec.md"
		case "plan":
			outputFile = "plan.md"
		case "tasks":
			outputFile = "tasks.md"
		default:
			outputFile = fmt.Sprintf("%s.md", phase)
		}

		// Write consolidated output
		outputPath := filepath.Join(specsDir, outputFile)
		if err := ioutil.WriteFile(outputPath, []byte(consolidatedContent.String()), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %v", outputPath, err)
		}

		log.Printf("📝 Created spec-kit file: %s", outputFile)
	}

	// Create a summary README for the spec
	readmePath := filepath.Join(specsDir, "README.md")
	readmeContent := fmt.Sprintf("# RFE %s: %s\n\n**Status**: %s\n**Phase**: %s\n**Repository**: %s\n**Branch**: %s\n\n## Description\n%s\n\n## Generated Files\n- `spec.md` - Feature specification from /specify phase\n- `plan.md` - Implementation plan from /plan phase\n- `tasks.md` - Task breakdown from /tasks phase\n\n## Agent Sessions\n%d agent sessions completed across all phases.\n\n---\n*Generated by vTeam RFE System*\n",
		workflow.ID, workflow.Title, workflow.Status, workflow.CurrentPhase,
		workflow.TargetRepoUrl, workflow.TargetRepoBranch, workflow.Description,
		len(workflow.AgentSessions))

	if err := ioutil.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %v", err)
	}

	log.Printf("📝 Created spec-kit README.md")
	return nil
}

// Create AgenticSessions for all selected agents in a specific phase
func createAgentSessionsForPhase(workflow *RFEWorkflow, phase string) error {
	if len(workflow.SelectedAgents) == 0 {
		return fmt.Errorf("no agents selected for workflow %s", workflow.ID)
	}

	// Convert agent personas to session names
	var createdSessions []RFEAgentSession

	for _, agentPersona := range workflow.SelectedAgents {
		sessionName := fmt.Sprintf("%s-%s-%s", workflow.ID, phase, strings.ToLower(strings.ReplaceAll(agentPersona, "_", "-")))

		// Create the AgenticSession resource for this agent
		// Note: For RFE workflows, we do NOT set websiteURL - this allows the claude-code-runner to
		// detect it as an agent-specific session and run _handle_agent_rfe_session() instead of browser automation
		sessionSpec := map[string]interface{}{
			"prompt":      fmt.Sprintf("/%s %s", phase, workflow.Description),
			"displayName": fmt.Sprintf("%s - %s (%s)", workflow.Title, agentPersona, phase),
			"llmSettings": map[string]interface{}{
				"model":       "claude-3-5-sonnet-20241022",
				"temperature": 0.7,
				"maxTokens":   8192,
			},
			"timeout": 3600, // 1 hour timeout
			"gitConfig": map[string]interface{}{
				"repositories": []map[string]interface{}{
					{
						"url":       workflow.TargetRepoUrl,
						"branch":    workflow.TargetRepoBranch,
						"clonePath": "target-repo",
					},
				},
			},
			// Add RFE-specific environment variables - these trigger _handle_agent_rfe_session() in claude-code-runner
			"environmentVariables": map[string]interface{}{
				"AGENT_PERSONA":    agentPersona,
				"WORKFLOW_PHASE":   phase,
				"PARENT_RFE":       workflow.ID,
				"SHARED_WORKSPACE": fmt.Sprintf("/workspace/%s", workflow.ID),
			},
		}

		// Add Git user configuration if provided
		if workflow.GitUserName != nil && *workflow.GitUserName != "" {
			gitConfig := sessionSpec["gitConfig"].(map[string]interface{})
			gitConfig["user"] = map[string]interface{}{
				"name":  *workflow.GitUserName,
				"email": workflow.GitUserEmail, // Can be nil, will be omitted in JSON
			}
		}

		session := map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      sessionName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"rfe-workflow":  workflow.ID,
					"rfe-phase":     phase,
					"agent-persona": agentPersona,
				},
			},
			"spec": sessionSpec,
			"status": map[string]interface{}{
				"phase": "Pending",
			},
		}

		gvr := getAgenticSessionResource()
		obj := &unstructured.Unstructured{Object: session}

		created, err := dynamicClient.Resource(gvr).Namespace(namespace).Create(context.TODO(), obj, v1.CreateOptions{})
		if err != nil {
			log.Printf("❌ Failed to create AgenticSession %s: %v", sessionName, err)
			return fmt.Errorf("failed to create agent session %s: %v", sessionName, err)
		}

		// Add to workflow's agent sessions
		agentSession := RFEAgentSession{
			ID:           sessionName,
			AgentPersona: agentPersona,
			Phase:        phase,
			Status:       "pending",
			StartedAt:    nil, // Will be set when session actually starts
			CompletedAt:  nil,
			Result:       nil,
			Cost:         nil,
		}
		createdSessions = append(createdSessions, agentSession)

		log.Printf("🤖 Created AgenticSession %s for agent %s in phase %s", sessionName, agentPersona, phase)
		_ = created // Suppress unused variable warning
	}

	// Update workflow with created sessions
	workflow.AgentSessions = append(workflow.AgentSessions, createdSessions...)
	workflow.Status = "in_progress"
	workflow.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Save updated workflow
	if err := saveRFEWorkflow(workflow); err != nil {
		log.Printf("⚠️ Failed to save workflow after creating agent sessions: %v", err)
	}

	log.Printf("✅ Created %d AgenticSessions for workflow %s phase %s", len(createdSessions), workflow.ID, phase)
	return nil
}

// Scan workspace and update workflow artifacts list
func scanAndUpdateWorkflowArtifacts(workflow *RFEWorkflow) error {
	workspaceDir := getRFEWorkspaceDir(workflow.ID)

	var artifacts []RFEArtifact

	// Scan agents directory for generated files
	agentsDir := getRFEAgentsDir(workflow.ID)
	if _, err := os.Stat(agentsDir); err == nil {
		err := filepath.Walk(agentsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
				// Get relative path from workspace
				relPath, err := filepath.Rel(workspaceDir, path)
				if err != nil {
					return err
				}

				// Determine which agent and phase this belongs to
				pathParts := strings.Split(relPath, string(filepath.Separator))
				var agent, phase string
				if len(pathParts) >= 3 && pathParts[0] == "agents" {
					phase = pathParts[1]      // e.g., "specify"
					agentFile := pathParts[2] // e.g., "engineering-manager.md"
					agent = strings.TrimSuffix(agentFile, ".md")
				}

				artifact := RFEArtifact{
					Path:       relPath,
					Name:       info.Name(),
					Type:       "specification", // Default type for agent-generated files
					Phase:      phase,
					CreatedBy:  agent,
					Size:       info.Size(),
					ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
				}

				artifacts = append(artifacts, artifact)
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to scan agents directory: %v", err)
		}
	}

	// Update workflow artifacts
	workflow.Artifacts = artifacts
	workflow.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Save updated workflow
	if err := saveRFEWorkflow(workflow); err != nil {
		return fmt.Errorf("failed to save workflow after artifact scan: %v", err)
	}

	log.Printf("📊 Scanned and found %d artifacts for workflow %s", len(artifacts), workflow.ID)
	return nil
}

func listRFEWorkflows(c *gin.Context) {
	var workflows []RFEWorkflow
	for _, workflow := range rfeWorkflows {
		// Sync agent session statuses before returning
		err := syncAgentSessionStatuses(workflow)
		if err != nil {
			log.Printf("⚠️ Failed to sync agent session statuses for workflow %s: %v", workflow.ID, err)
			// Don't fail the request, just log the warning
		}
		workflows = append(workflows, *workflow)
	}

	if workflows == nil {
		workflows = []RFEWorkflow{}
	}

	c.JSON(http.StatusOK, gin.H{"workflows": workflows})
}

func createRFEWorkflow(c *gin.Context) {
	var req CreateRFEWorkflowRequest

	// Log the raw request body for debugging
	bodyBytes, _ := c.GetRawData()
	log.Printf("📥 Raw request body: %s", string(bodyBytes))

	// Reset the body for binding
	c.Request.Body = ioutil.NopCloser(strings.NewReader(string(bodyBytes)))

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Validation error creating RFE workflow: %v", err)
		log.Printf("📝 Request payload validation failed for: %+v", req)
		log.Printf("🔍 SelectedAgents received: %+v (length: %d)", req.SelectedAgents, len(req.SelectedAgents))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed: " + err.Error(),
			"details": "Check required fields: title, description, targetRepoUrl, targetRepoBranch, selectedAgents",
		})
		return
	}

	log.Printf("✅ Successfully parsed RFE workflow request")
	log.Printf("📝 SelectedAgents: %+v", req.SelectedAgents)

	// Generate unique ID for the workflow
	workflowID := fmt.Sprintf("rfe-%d", time.Now().Unix())

	now := time.Now().UTC().Format(time.RFC3339)

	workflow := &RFEWorkflow{
		ID:               workflowID,
		Title:            req.Title,
		Description:      req.Description,
		Status:           "draft",
		CurrentPhase:     "specify",
		SelectedAgents:   req.SelectedAgents,
		TargetRepoUrl:    req.TargetRepoUrl,
		TargetRepoBranch: req.TargetRepoBranch,
		GitUserName:      req.GitUserName,
		GitUserEmail:     req.GitUserEmail,
		CreatedAt:        now,
		UpdatedAt:        now,
		AgentSessions:    []RFEAgentSession{},
		Artifacts:        []RFEArtifact{},
		PhaseResults:     make(map[string]PhaseResult),
	}

	// Store workflow in memory
	rfeWorkflows[workflowID] = workflow

	// Create workspace directory structure
	if err := createRFEWorkspace(workflowID); err != nil {
		log.Printf("⚠️ Failed to create workspace: %v", err)
		// Continue anyway - workspace will be created when needed
	}

	// Save workflow to persistent storage
	if err := saveRFEWorkflow(workflow); err != nil {
		log.Printf("⚠️ Failed to save workflow to disk: %v", err)
		// Continue anyway - the workflow is still in memory
	}

	// Create initial AgenticSessions for the specify phase
	if err := createAgentSessionsForPhase(workflow, "specify"); err != nil {
		log.Printf("⚠️ Failed to create agent sessions for specify phase: %v", err)
		// Continue anyway - sessions can be created manually later
	}

	log.Printf("✅ Created RFE workflow %s with agents: %v", workflowID, req.SelectedAgents)
	log.Printf("📊 Workflow details: Title='%s', Repo='%s', Branch='%s'", req.Title, req.TargetRepoUrl, req.TargetRepoBranch)

	c.JSON(http.StatusCreated, workflow)
}

func getRFEWorkflow(c *gin.Context) {
	id := c.Param("id")

	// First try to get from memory
	workflow, exists := rfeWorkflows[id]
	if !exists {
		// If not in memory, try to load from disk
		loadedWorkflow, err := loadRFEWorkflow(id)
		if err != nil {
			log.Printf("❌ Failed to load workflow %s from disk: %v", id, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
			return
		}

		// Add back to memory cache for future requests
		rfeWorkflows[id] = loadedWorkflow
		workflow = loadedWorkflow
		log.Printf("✅ Loaded workflow %s from disk and cached in memory", id)
	}

	// Sync agent session statuses from Kubernetes AgenticSession resources
	err := syncAgentSessionStatuses(workflow)
	if err != nil {
		log.Printf("⚠️ Failed to sync agent session statuses for workflow %s: %v", id, err)
		// Don't fail the request, just log the warning
	}

	// Scan for new artifacts before returning (if workspace is accessible)
	if workspaceDir := getRFEWorkspaceDir(workflow.ID); workspaceDir != "" {
		if _, err := os.Stat(filepath.Dir(workspaceDir)); err == nil {
			err := scanAndUpdateWorkflowArtifacts(workflow)
			if err != nil {
				log.Printf("⚠️ Failed to scan artifacts for workflow %s: %v", id, err)
				// Don't fail the request, just log the warning
			}
		} else {
			log.Printf("ℹ️ Workspace directory not accessible for workflow %s: %v", id, err)
		}
	}

	// Try to marshal to JSON first to catch any serialization issues
	_, jsonErr := json.Marshal(workflow)
	if jsonErr != nil {
		log.Printf("❌ Failed to marshal workflow %s to JSON: %v", id, jsonErr)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error - failed to serialize workflow data",
		})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

func deleteRFEWorkflow(c *gin.Context) {
	id := c.Param("id")

	_, exists := rfeWorkflows[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	// TODO: Clean up associated AgenticSessions and PVC data
	delete(rfeWorkflows, id)

	// Delete from persistent storage
	if err := deleteRFEWorkflowFile(id); err != nil {
		log.Printf("⚠️ Failed to delete workflow file: %v", err)
		// Continue anyway - the workflow is deleted from memory
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workflow deleted successfully"})
}

func pauseRFEWorkflow(c *gin.Context) {
	id := c.Param("id")

	workflow, exists := rfeWorkflows[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	// TODO: Pause running AgenticSessions
	workflow.Status = "paused"
	workflow.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Save updated workflow to persistent storage
	if err := saveRFEWorkflow(workflow); err != nil {
		log.Printf("⚠️ Failed to save workflow after pause: %v", err)
	}

	c.JSON(http.StatusOK, workflow)
}

func resumeRFEWorkflow(c *gin.Context) {
	id := c.Param("id")

	workflow, exists := rfeWorkflows[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	// TODO: Resume paused AgenticSessions
	workflow.Status = "in_progress"
	workflow.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Save updated workflow to persistent storage
	if err := saveRFEWorkflow(workflow); err != nil {
		log.Printf("⚠️ Failed to save workflow after resume: %v", err)
	}

	c.JSON(http.StatusOK, workflow)
}

func advanceRFEWorkflowPhase(c *gin.Context) {
	id := c.Param("id")

	var req AdvancePhaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflow, exists := rfeWorkflows[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	// Determine next phase
	var nextPhase string
	switch workflow.CurrentPhase {
	case "specify":
		nextPhase = "plan"
	case "plan":
		nextPhase = "tasks"
	case "tasks":
		nextPhase = "completed"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot advance from current phase"})
		return
	}

	// TODO: Validate current phase is complete (unless force=true)
	// Create AgenticSessions for next phase
	if nextPhase != "completed" {
		if err := createAgentSessionsForPhase(workflow, nextPhase); err != nil {
			log.Printf("⚠️ Failed to create agent sessions for phase %s: %v", nextPhase, err)
			// Continue anyway - phase was advanced, sessions can be created manually
		}
	}

	workflow.CurrentPhase = nextPhase
	workflow.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if nextPhase == "completed" {
		workflow.Status = "completed"
	}

	// Save updated workflow to persistent storage
	if err := saveRFEWorkflow(workflow); err != nil {
		log.Printf("⚠️ Failed to save workflow after phase advance: %v", err)
	}

	log.Printf("Advanced workflow %s to phase: %s", id, nextPhase)

	c.JSON(http.StatusOK, workflow)
}

func pushRFEWorkflowToGit(c *gin.Context) {
	id := c.Param("id")

	workflow, exists := rfeWorkflows[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	// Implement Git push functionality
	err := pushWorkflowToGitRepo(workflow)
	if err != nil {
		log.Printf("❌ Failed to push workflow %s to Git: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to push to Git repository",
			"details": err.Error(),
		})
		return
	}

	// Update workflow status
	workflow.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := saveRFEWorkflow(workflow); err != nil {
		log.Printf("⚠️ Failed to save workflow after Git push: %v", err)
	}

	log.Printf("✅ Successfully pushed workflow %s artifacts to Git: %s", id, workflow.TargetRepoUrl)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Successfully pushed to Git repository",
		"repository": workflow.TargetRepoUrl,
		"branch":     workflow.TargetRepoBranch,
	})
}

func scanRFEWorkflowArtifacts(c *gin.Context) {
	id := c.Param("id")

	workflow, exists := rfeWorkflows[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	err := scanAndUpdateWorkflowArtifacts(workflow)
	if err != nil {
		log.Printf("❌ Failed to scan artifacts for workflow %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to scan workspace artifacts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Artifacts scanned successfully",
		"artifactCount": len(workflow.Artifacts),
	})
}

func getRFEWorkflowArtifact(c *gin.Context) {
	id := c.Param("id")
	artifactPath := c.Param("path")

	_, exists := rfeWorkflows[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	// Read artifact content from workspace
	fullPath := getRFEArtifactPath(id, artifactPath)

	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, return empty content for new files
			c.Header("Content-Type", "text/plain")
			c.String(http.StatusOK, "")
			return
		}
		log.Printf("❌ Failed to read artifact %s: %v", fullPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read artifact content"})
		return
	}

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, string(content))
}

func updateRFEWorkflowArtifact(c *gin.Context) {
	id := c.Param("id")
	artifactPath := c.Param("path")

	workflow, exists := rfeWorkflows[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	// Read the content from request body
	content, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read content"})
		return
	}

	// Write artifact content to workspace
	fullPath := getRFEArtifactPath(id, artifactPath)

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		log.Printf("❌ Failed to create directory for artifact %s: %v", fullPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	// Write the content to file
	if err := ioutil.WriteFile(fullPath, content, 0644); err != nil {
		log.Printf("❌ Failed to write artifact %s: %v", fullPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write artifact content"})
		return
	}

	log.Printf("💾 Updated artifact %s for workflow %s (%d bytes)", artifactPath, id, len(content))

	// Update artifact metadata if it exists
	now := time.Now().UTC().Format(time.RFC3339)
	for i, artifact := range workflow.Artifacts {
		if artifact.Path == artifactPath {
			workflow.Artifacts[i].Size = int64(len(content))
			workflow.Artifacts[i].ModifiedAt = now
			break
		}
	}

	workflow.UpdatedAt = now

	// Save updated workflow to persistent storage
	if err := saveRFEWorkflow(workflow); err != nil {
		log.Printf("⚠️ Failed to save workflow after artifact update: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Artifact updated successfully",
		"path":    artifactPath,
		"size":    len(content),
	})
}
