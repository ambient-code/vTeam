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

			// RFE workflow endpoints (project-scoped)
			projectGroup.GET("/rfe-workflows", listProjectRFEWorkflows)
			projectGroup.POST("/rfe-workflows", createProjectRFEWorkflow)
			projectGroup.GET("/rfe-workflows/:id", getProjectRFEWorkflow)
			projectGroup.DELETE("/rfe-workflows/:id", deleteProjectRFEWorkflow)
			projectGroup.POST("/rfe-workflows/:id/pause", pauseProjectRFEWorkflow)
			projectGroup.POST("/rfe-workflows/:id/resume", resumeProjectRFEWorkflow)
			projectGroup.POST("/rfe-workflows/:id/advance-phase", advanceProjectRFEWorkflowPhase)
			projectGroup.POST("/rfe-workflows/:id/push-to-git", pushProjectRFEWorkflowToGit)
			projectGroup.POST("/rfe-workflows/:id/export-spec-kit", exportProjectRFEWorkflowSpecKit)
			projectGroup.POST("/rfe-workflows/:id/scan-artifacts", scanProjectRFEWorkflowArtifacts)
			projectGroup.GET("/rfe-workflows/:id/artifacts/*path", getProjectRFEWorkflowArtifact)
			projectGroup.PUT("/rfe-workflows/:id/artifacts/*path", updateProjectRFEWorkflowArtifact)
			// Sessions linkage within an RFE
			projectGroup.GET("/rfe-workflows/:id/sessions", listProjectRFEWorkflowSessions)
			projectGroup.POST("/rfe-workflows/:id/sessions", addProjectRFEWorkflowSession)
			projectGroup.DELETE("/rfe-workflows/:id/sessions/:sessionName", removeProjectRFEWorkflowSession)

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
	Project          string                 `json:"project,omitempty"`
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
		"project":          workflow.Project,
		"title":            workflow.Title,
		"description":      workflow.Description,
		"selectedAgents":   workflow.SelectedAgents,
		"targetRepoUrl":    workflow.TargetRepoUrl,
		"targetRepoBranch": workflow.TargetRepoBranch,
	}

	// Build status
	sessions := make([]map[string]interface{}, 0, len(workflow.AgentSessions))
	for _, s := range workflow.AgentSessions {
		sessions = append(sessions, map[string]interface{}{
			"id":           s.ID,
			"agentPersona": s.AgentPersona,
			"phase":        s.Phase,
			"state":        s.Status,
			"startedAt":    s.StartedAt,
			"completedAt":  s.CompletedAt,
		})
	}
	artifacts := make([]map[string]interface{}, 0, len(workflow.Artifacts))
	for _, a := range workflow.Artifacts {
		artifacts = append(artifacts, map[string]interface{}{
			"path":       a.Path,
			"name":       a.Name,
			"type":       a.Type,
			"phase":      a.Phase,
			"createdBy":  a.CreatedBy,
			"size":       a.Size,
			"modifiedAt": a.ModifiedAt,
		})
	}
	status := map[string]interface{}{
		"status":        workflow.Status,
		"currentPhase":  workflow.CurrentPhase,
		"agentSessions": sessions,
		"artifacts":     artifacts,
		"phaseResults":  workflow.PhaseResults,
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
		"spec":   spec,
		"status": status,
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

func loadProjectRFEWorkflowFromCRWithClient(dyn dynamic.Interface, project, id string) (*RFEWorkflow, error) {
	gvr := getRFEWorkflowResource()
	if dyn == nil {
		return nil, fmt.Errorf("no dynamic client provided")
	}
	item, err := dyn.Resource(gvr).Namespace(project).Get(context.TODO(), id, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	obj := item.Object
	spec, _ := obj["spec"].(map[string]interface{})
	status, _ := obj["status"].(map[string]interface{})

	wf := &RFEWorkflow{
		ID:               id,
		Title:            fmt.Sprintf("%v", spec["title"]),
		Description:      fmt.Sprintf("%v", spec["description"]),
		Status:           fmt.Sprintf("%v", status["status"]),
		CurrentPhase:     fmt.Sprintf("%v", status["currentPhase"]),
		SelectedAgents:   []string{},
		TargetRepoUrl:    fmt.Sprintf("%v", spec["targetRepoUrl"]),
		TargetRepoBranch: fmt.Sprintf("%v", spec["targetRepoBranch"]),
		Project:          fmt.Sprintf("%v", spec["project"]),
		CreatedAt:        "",
		UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
		AgentSessions:    []RFEAgentSession{},
		Artifacts:        []RFEArtifact{},
		PhaseResults:     map[string]PhaseResult{},
	}
	if sa, ok := spec["selectedAgents"].([]interface{}); ok {
		for _, v := range sa {
			wf.SelectedAgents = append(wf.SelectedAgents, fmt.Sprintf("%v", v))
		}
	}
	if sess, ok := status["agentSessions"].([]interface{}); ok {
		for _, v := range sess {
			if m, ok := v.(map[string]interface{}); ok {
				s := RFEAgentSession{
					ID:           fmt.Sprintf("%v", m["id"]),
					AgentPersona: fmt.Sprintf("%v", m["agentPersona"]),
					Phase:        fmt.Sprintf("%v", m["phase"]),
					Status:       fmt.Sprintf("%v", m["state"]),
				}
				if started, ok := m["startedAt"].(string); ok {
					s.StartedAt = &started
				}
				if completed, ok := m["completedAt"].(string); ok {
					s.CompletedAt = &completed
				}
				wf.AgentSessions = append(wf.AgentSessions, s)
			}
		}
	}
	if arts, ok := status["artifacts"].([]interface{}); ok {
		for _, v := range arts {
			if m, ok := v.(map[string]interface{}); ok {
				a := RFEArtifact{
					Path:       fmt.Sprintf("%v", m["path"]),
					Name:       fmt.Sprintf("%v", m["name"]),
					Type:       fmt.Sprintf("%v", m["type"]),
					Phase:      fmt.Sprintf("%v", m["phase"]),
					CreatedBy:  fmt.Sprintf("%v", m["createdBy"]),
					ModifiedAt: fmt.Sprintf("%v", m["modifiedAt"]),
				}
				if size, ok := m["size"].(float64); ok {
					a.Size = int64(size)
				}
				wf.Artifacts = append(wf.Artifacts, a)
			}
		}
	}
	return wf, nil
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

// Legacy in-memory workflow storage removed (project-scoped only)

// File paths for persistent storage
// Legacy non-project file paths removed

// Project-scoped file paths for persistent storage
func getProjectRFEWorkflowsDir(project string) string {
	return filepath.Join(stateBaseDir, "projects", project, "rfe-workflows")
}

func getProjectRFEWorkflowFilePath(project, id string) string {
	return filepath.Join(getProjectRFEWorkflowsDir(project), id+".json")
}

// Save workflow to persistent storage
// Legacy non-project save removed

// Save project-scoped workflow to persistent storage
func saveProjectRFEWorkflow(workflow *RFEWorkflow) error {
	if workflow.Project == "" {
		return fmt.Errorf("project is required for project-scoped workflow save")
	}
	// Persist via per-project content service under /rfe-workflows/<id>.json (handled at call sites)
	data, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %v", err)
	}
	// No Gin context in this helper; use direct file write as a fallback is NOT allowed.
	// Instead, try best-effort: write to local state dir to avoid data loss when no content service token is present.
	// The authoritative store is the CRD; file is a convenience mirror.
	// Keep local mirror for compatibility with existing read paths.
	localDir := getProjectRFEWorkflowsDir(workflow.Project)
	_ = os.MkdirAll(localDir, 0755)
	_ = ioutil.WriteFile(getProjectRFEWorkflowFilePath(workflow.Project, workflow.ID), data, 0644)
	log.Printf("💾 Saved project RFE workflow %s (project=%s) [content service expected via handlers; local mirror written]", workflow.ID, workflow.Project)
	return nil
}

// Load workflow from persistent storage
// Legacy non-project load removed

// Load project-scoped workflow from persistent storage
func loadProjectRFEWorkflow(project, id string) (*RFEWorkflow, error) {
	// Primary source of truth is CRD; this function remains as fallback for legacy paths.
	filePath := getProjectRFEWorkflowFilePath(project, id)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read project workflow file: %v", err)
	}
	var workflow RFEWorkflow
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project workflow: %v", err)
	}
	return &workflow, nil
}

// Load all workflows from persistent storage
// Legacy non-project preload removed

// Delete workflow from persistent storage
// Legacy non-project delete removed

// Delete project-scoped workflow from persistent storage
func deleteProjectRFEWorkflowFile(project, id string) error {
	filePath := getProjectRFEWorkflowFilePath(project, id)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete project workflow file: %v", err)
	}
	log.Printf("🗑️ Deleted RFE workflow %s (project=%s) from disk", id, project)
	return nil
}

// Sync agent session statuses from Kubernetes AgenticSession resources
func syncAgentSessionStatuses(dyn dynamic.Interface, workflow *RFEWorkflow) error {
	if workflow == nil || len(workflow.AgentSessions) == 0 {
		return nil
	}

	// Define resource for AgenticSession
	gvr := getAgenticSessionV1Alpha1Resource()

	for i := range workflow.AgentSessions {
		session := &workflow.AgentSessions[i]
		sessionName := session.ID

		// Get the AgenticSession resource from Kubernetes
		item, err := dyn.Resource(gvr).Namespace(workflow.Project).Get(context.TODO(), sessionName, v1.GetOptions{})
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

	return nil
}

// RFE Workspace utility functions
// Legacy non-project workspace dir removed

// Project-scoped workspace helpers
func getProjectRFEWorkspaceDir(project, workflowID string) string {
	return filepath.Join(pvcBaseDir, project, workflowID)
}

// Legacy non-project git repo dir removed

func getProjectRFEGitRepoDir(project, workflowID string) string {
	return filepath.Join(getProjectRFEWorkspaceDir(project, workflowID), "git-repo")
}

// Legacy non-project agents dir removed

func getProjectRFEAgentsDir(project, workflowID string) string {
	return filepath.Join(getProjectRFEWorkspaceDir(project, workflowID), "agents")
}

// Legacy non-project UI edits dir removed

// Deprecated: UI edits dir no longer used

// Create workspace directory structure for RFE
// Legacy non-project workspace creation removed

// Create workspace directory structure for project-scoped RFE
func createProjectRFEWorkspace(project, workflowID string) error {
	workspaceDir := getProjectRFEWorkspaceDir(project, workflowID)
	dirs := []string{
		workspaceDir,
		getProjectRFEGitRepoDir(project, workflowID),
		getProjectRFEAgentsDir(project, workflowID),
		filepath.Join(getProjectRFEAgentsDir(project, workflowID), "specify"),
		filepath.Join(getProjectRFEAgentsDir(project, workflowID), "plan"),
		filepath.Join(getProjectRFEAgentsDir(project, workflowID), "tasks"),
		filepath.Join(workspaceDir, "sessions"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	log.Printf("📁 Created Project RFE workspace at %s (project=%s)", workspaceDir, project)
	return nil
}

// Get the full path to an artifact file in the workspace
// Legacy non-project artifact path removed

// Get the full path to an artifact file in a project-scoped workspace
func getProjectRFEArtifactPath(project, workflowID, artifactPath string) string {
	// Check if it's in git-repo or agents directory
	if strings.HasPrefix(artifactPath, "git-repo/") {
		return filepath.Join(getProjectRFEWorkspaceDir(project, workflowID), artifactPath)
	} else if strings.HasPrefix(artifactPath, "agents/") {
		return filepath.Join(getProjectRFEWorkspaceDir(project, workflowID), artifactPath)
	} else if strings.HasPrefix(artifactPath, "sessions/") {
		return filepath.Join(getProjectRFEWorkspaceDir(project, workflowID), artifactPath)
	} else {
		// Default to git-repo for backward compatibility
		return filepath.Join(getProjectRFEGitRepoDir(project, workflowID), artifactPath)
	}
}

// Push workflow artifacts to Git repository
// Legacy non-project git push removed

// Project-scoped Git push
func pushProjectWorkflowToGitRepo(workflow *RFEWorkflow) error {
	gitRepoDir := getProjectRFEGitRepoDir(workflow.Project, workflow.ID)
	workspaceDir := getProjectRFEWorkspaceDir(workflow.Project, workflow.ID)
	agentsDir := getProjectRFEAgentsDir(workflow.Project, workflow.ID)

	if _, err := os.Stat(gitRepoDir); os.IsNotExist(err) {
		log.Printf("📥 Cloning repository %s to %s", workflow.TargetRepoUrl, gitRepoDir)
		cloneCmd := exec.Command("git", "clone", "-b", workflow.TargetRepoBranch, workflow.TargetRepoUrl, gitRepoDir)
		cloneCmd.Dir = workspaceDir
		if output, err := cloneCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to clone repository: %v, output: %s", err, string(output))
		}
	}
	if workflow.GitUserName != nil && *workflow.GitUserName != "" {
		configCmd := exec.Command("git", "config", "user.name", *workflow.GitUserName)
		configCmd.Dir = gitRepoDir
		_ = configCmd.Run()
	}
	if workflow.GitUserEmail != nil && *workflow.GitUserEmail != "" {
		configCmd := exec.Command("git", "config", "user.email", *workflow.GitUserEmail)
		configCmd.Dir = gitRepoDir
		_ = configCmd.Run()
	}
	pullCmd := exec.Command("git", "pull", "origin", workflow.TargetRepoBranch)
	pullCmd.Dir = gitRepoDir
	_, _ = pullCmd.CombinedOutput()

	specsDir := filepath.Join(gitRepoDir, "specs", workflow.ID)
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		return fmt.Errorf("failed to create specs directory: %v", err)
	}
	if err := convertAgentOutputsToSpecKit(agentsDir, specsDir, workflow); err != nil {
		log.Printf("⚠️ Failed to convert agent outputs to spec-kit format: %v", err)
		if _, err := os.Stat(agentsDir); err == nil {
			copyCmd := exec.Command("cp", "-r", agentsDir, specsDir)
			_, _ = copyCmd.CombinedOutput()
		}
	}
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = gitRepoDir
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add changes: %v, output: %s", err, string(output))
	}
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = gitRepoDir
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %v", err)
	}
	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		log.Printf("ℹ️ No changes to commit for workflow %s (project=%s)", workflow.ID, workflow.Project)
		return nil
	}
	commitMessage := fmt.Sprintf("Add %s phase for RFE %s: %s\n\nGenerated spec-kit compatible artifacts:\n- spec.md\n- plan.md\n- tasks.md\n\nPhase: %s\nProject: %s\nAgents: %d sessions completed\n\n🤖 Generated with vTeam RFE System",
		workflow.CurrentPhase, workflow.ID, workflow.Title, workflow.CurrentPhase, workflow.Project, len(workflow.AgentSessions))
	commitCmd := exec.Command("git", "commit", "-m", commitMessage)
	commitCmd.Dir = gitRepoDir
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit changes: %v, output: %s", err, string(output))
	}
	pushCmd := exec.Command("git", "push", "origin", workflow.TargetRepoBranch)
	pushCmd.Dir = gitRepoDir
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push changes: %v, output: %s", err, string(output))
	}
	log.Printf("🚀 Successfully pushed project RFE %s artifacts to %s (project=%s)", workflow.ID, workflow.TargetRepoUrl, workflow.Project)
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

// Helper: build AgenticSession spec and labels for an RFE session (shared builder)
func buildRFESessionSpecAndLabels(workflow *RFEWorkflow, phase, agentPersona string, projectOpt *string) (map[string]interface{}, map[string]interface{}) {
	sharedWorkspace := fmt.Sprintf("/workspace/%s", workflow.ID)
	if projectOpt != nil && *projectOpt != "" {
		sharedWorkspace = fmt.Sprintf("/workspace/%s/%s", *projectOpt, workflow.ID)
	}

	sessionSpec := map[string]interface{}{
		"prompt":      fmt.Sprintf("/%s %s", phase, workflow.Description),
		"displayName": fmt.Sprintf("%s - %s (%s)", workflow.Title, agentPersona, phase),
		"llmSettings": map[string]interface{}{
			"model":       "claude-3-5-sonnet-20241022",
			"temperature": 0.7,
			"maxTokens":   8192,
		},
		"timeout": 3600,
		"gitConfig": map[string]interface{}{
			"repositories": []map[string]interface{}{
				{
					"url":       workflow.TargetRepoUrl,
					"branch":    workflow.TargetRepoBranch,
					"clonePath": "target-repo",
				},
			},
		},
		"environmentVariables": map[string]interface{}{
			"AGENT_PERSONA":    agentPersona,
			"WORKFLOW_PHASE":   phase,
			"PARENT_RFE":       workflow.ID,
			"SHARED_WORKSPACE": sharedWorkspace,
		},
	}
	if projectOpt != nil && *projectOpt != "" {
		sessionSpec["project"] = *projectOpt
	}
	if workflow.GitUserName != nil && *workflow.GitUserName != "" {
		gitConfig := sessionSpec["gitConfig"].(map[string]interface{})
		gitConfig["user"] = map[string]interface{}{
			"name":  *workflow.GitUserName,
			"email": workflow.GitUserEmail,
		}
	}

	labels := map[string]interface{}{
		"rfe-workflow":  workflow.ID,
		"rfe-phase":     phase,
		"agent-persona": agentPersona,
	}
	if projectOpt != nil && *projectOpt != "" {
		labels["project"] = *projectOpt
	}
	return sessionSpec, labels
}

// Create AgenticSessions for all selected agents in a specific phase
// Legacy non-project session creation removed

// Project-scoped: Create AgenticSessions for all selected agents
func createAgentSessionsForPhaseProject(dyn dynamic.Interface, workflow *RFEWorkflow, phase string) error {
	if len(workflow.SelectedAgents) == 0 {
		return fmt.Errorf("no agents selected for workflow %s", workflow.ID)
	}
	if dyn == nil {
		return fmt.Errorf("no dynamic client provided")
	}
	var createdSessions []RFEAgentSession
	for _, agentPersona := range workflow.SelectedAgents {
		sessionName := fmt.Sprintf("%s-%s-%s", workflow.ID, phase, strings.ToLower(strings.ReplaceAll(agentPersona, "_", "-")))
		sessionSpec, labels := buildRFESessionSpecAndLabels(workflow, phase, agentPersona, &workflow.Project)
		session := map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      sessionName,
				"namespace": workflow.Project,
				"labels":    labels,
			},
			"spec":   sessionSpec,
			"status": map[string]interface{}{"phase": "Pending"},
		}
		gvr := getAgenticSessionV1Alpha1Resource()
		obj := &unstructured.Unstructured{Object: session}
		if _, err := dyn.Resource(gvr).Namespace(workflow.Project).Create(context.TODO(), obj, v1.CreateOptions{}); err != nil {
			log.Printf("❌ Failed to create AgenticSession %s: %v", sessionName, err)
			return fmt.Errorf("failed to create agent session %s: %v", sessionName, err)
		}
		agentSession := RFEAgentSession{ID: sessionName, AgentPersona: agentPersona, Phase: phase, Status: "pending"}
		createdSessions = append(createdSessions, agentSession)
		log.Printf("🤖 Created AgenticSession %s for agent %s in phase %s (project=%s)", sessionName, agentPersona, phase, workflow.Project)
	}
	workflow.AgentSessions = append(workflow.AgentSessions, createdSessions...)
	workflow.Status = "in_progress"
	workflow.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := saveProjectRFEWorkflow(workflow); err != nil {
		log.Printf("⚠️ Failed to save workflow after creating agent sessions: %v", err)
	}
	log.Printf("✅ Created %d AgenticSessions for workflow %s phase %s (project=%s)", len(createdSessions), workflow.ID, phase, workflow.Project)
	return nil
}

// Scan workspace and update workflow artifacts list
// Legacy non-project artifact scan removed

// Project-scoped artifact scan
func scanAndUpdateWorkflowArtifactsProject(workflow *RFEWorkflow) error {
	workspaceDir := getProjectRFEWorkspaceDir(workflow.Project, workflow.ID)
	var artifacts []RFEArtifact
	// Scan agents directory
	agentsDir := getProjectRFEAgentsDir(workflow.Project, workflow.ID)
	if _, err := os.Stat(agentsDir); err == nil {
		err := filepath.Walk(agentsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Scan per-session artifacts directory
			sessionsDir := filepath.Join(workspaceDir, "sessions")
			if _, err := os.Stat(sessionsDir); err == nil {
				err := filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if info.IsDir() {
						return nil
					}
					relPath, err := filepath.Rel(workspaceDir, path)
					if err != nil {
						return err
					}
					parts := strings.Split(relPath, string(filepath.Separator))
					if len(parts) >= 3 && parts[0] == "sessions" && parts[2] == "artifacts" {
						sessionName := parts[1]
						artifact := RFEArtifact{
							Path:       relPath,
							Name:       info.Name(),
							Type:       "artifact",
							Phase:      "",
							CreatedBy:  sessionName,
							Size:       info.Size(),
							ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
						}
						artifacts = append(artifacts, artifact)
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("failed to scan sessions artifacts: %v", err)
				}
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
				relPath, err := filepath.Rel(workspaceDir, path)
				if err != nil {
					return err
				}
				pathParts := strings.Split(relPath, string(filepath.Separator))
				var agent, phase string
				if len(pathParts) >= 3 && pathParts[0] == "agents" {
					phase = pathParts[1]
					agentFile := pathParts[2]
					agent = strings.TrimSuffix(agentFile, ".md")
				}
				artifact := RFEArtifact{
					Path:       relPath,
					Name:       info.Name(),
					Type:       "specification",
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
	// TODO: scan sessions/*/artifacts as we standardize
	workflow.Artifacts = artifacts
	workflow.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := saveProjectRFEWorkflow(workflow); err != nil {
		return fmt.Errorf("failed to save workflow after artifact scan: %v", err)
	}
	log.Printf("📊 Scanned and found %d artifacts for workflow %s (project=%s)", len(artifacts), workflow.ID, workflow.Project)
	return nil
}
