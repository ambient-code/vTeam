package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	k8sClient     *kubernetes.Clientset
	dynamicClient dynamic.Interface
	namespace     string
	stateBaseDir  string
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

	// Setup Gin router
	r := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// API routes
	api := r.Group("/api")
	{
		api.GET("/agentic-sessions", listAgenticSessions)
		api.GET("/agentic-sessions/:name", getAgenticSession)
		api.POST("/agentic-sessions", createAgenticSession)
		api.DELETE("/agentic-sessions/:name", deleteAgenticSession)
		api.PUT("/agentic-sessions/:name/status", updateAgenticSessionStatus)
		api.PUT("/agentic-sessions/:name/displayname", updateAgenticSessionDisplayName)
		api.POST("/agentic-sessions/:name/stop", stopAgenticSession)
	}

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

	// Create dynamic client for custom resources
	dynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %v", err)
	}

	return nil
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
	Prompt      string      `json:"prompt" binding:"required"`
	WebsiteURL  string      `json:"websiteURL" binding:"required,url"`
	DisplayName string      `json:"displayName"`
	LLMSettings LLMSettings `json:"llmSettings"`
	Timeout     int         `json:"timeout"`
}

type LLMSettings struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

type GitUser struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type GitAuthentication struct {
	SSHKeySecret      string `json:"sshKeySecret,omitempty"`
	TokenSecret       string `json:"tokenSecret,omitempty"`
	KnownHostsSecret  string `json:"knownHostsSecret,omitempty"`
}

type GitRepository struct {
	URL       string `json:"url"`
	Branch    string `json:"branch,omitempty"`
	ClonePath string `json:"clonePath,omitempty"`
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
}

type CreateAgenticSessionRequest struct {
	Prompt      string       `json:"prompt" binding:"required"`
	WebsiteURL  string       `json:"websiteURL,omitempty"`
	DisplayName string       `json:"displayName,omitempty"`
	LLMSettings *LLMSettings `json:"llmSettings,omitempty"`
	Timeout     *int         `json:"timeout,omitempty"`
	GitConfig   *GitConfig   `json:"gitConfig,omitempty"`
}

// getAgenticSessionResource returns the GroupVersionResource for AgenticSession
func getAgenticSessionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1",
		Resource: "agenticsessions",
	}
}

func listAgenticSessions(c *gin.Context) {
	gvr := getAgenticSessionResource()

	list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list agentic sessions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list agentic sessions"})
		return
	}

	var sessions []AgenticSession
	for _, item := range list.Items {
		session := AgenticSession{
			APIVersion: item.GetAPIVersion(),
			Kind:       item.GetKind(),
			Metadata:   item.Object["metadata"].(map[string]interface{}),
		}

		if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
			session.Spec = parseSpec(spec)
		}

		if status, ok := item.Object["status"].(map[string]interface{}); ok {
			session.Status = parseStatus(status)
			// Read additional data from files
			if session.Status != nil {
				sessionName := item.GetName()
				readDataFromFiles(sessionName, session.Status)
			}
		}

		sessions = append(sessions, session)
	}

	c.JSON(http.StatusOK, sessions)
}

func getAgenticSession(c *gin.Context) {
	name := c.Param("name")
	gvr := getAgenticSessionResource()

	item, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	session := AgenticSession{
		APIVersion: item.GetAPIVersion(),
		Kind:       item.GetKind(),
		Metadata:   item.Object["metadata"].(map[string]interface{}),
	}

	if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := item.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(status)
		// Read additional data from files
		if session.Status != nil {
			readDataFromFiles(name, session.Status)
		}
	}

	c.JSON(http.StatusOK, session)
}

func createAgenticSession(c *gin.Context) {
	var req CreateAgenticSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults for LLM settings if not provided
	llmSettings := LLMSettings{
		Model:       "claude-3-5-sonnet-20241022",
		Temperature: 0.7,
		MaxTokens:   4000,
	}
	if req.LLMSettings != nil {
		if req.LLMSettings.Model != "" {
			llmSettings.Model = req.LLMSettings.Model
		}
		if req.LLMSettings.Temperature != 0 {
			llmSettings.Temperature = req.LLMSettings.Temperature
		}
		if req.LLMSettings.MaxTokens != 0 {
			llmSettings.MaxTokens = req.LLMSettings.MaxTokens
		}
	}

	timeout := 300
	if req.Timeout != nil {
		timeout = *req.Timeout
	}

	// Generate unique name
	timestamp := time.Now().Unix()
	name := fmt.Sprintf("agentic-session-%d", timestamp)

	// Create the custom resource
	session := map[string]interface{}{
		"apiVersion": "vteam.ambient-code/v1",
		"kind":       "AgenticSession",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"spec": buildSessionSpec(req, llmSettings, timeout),
		"status": map[string]interface{}{
			"phase": "Pending",
		},
	}

	gvr := getAgenticSessionResource()
	obj := &unstructured.Unstructured{Object: session}

	created, err := dynamicClient.Resource(gvr).Namespace(namespace).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create agentic session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agentic session"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Agentic session created successfully",
		"name":    name,
		"uid":     created.GetUID(),
	})
}

// buildSessionSpec builds the spec section of an AgenticSession
func buildSessionSpec(req CreateAgenticSessionRequest, llmSettings LLMSettings, timeout int) map[string]interface{} {
	// Provide default website URL if not provided
	websiteURL := req.WebsiteURL
	if websiteURL == "" {
		websiteURL = "https://example.com"
	}

	spec := map[string]interface{}{
		"prompt":      req.Prompt,
		"websiteURL":  websiteURL,
		"displayName": req.DisplayName,
		"llmSettings": map[string]interface{}{
			"model":       llmSettings.Model,
			"temperature": llmSettings.Temperature,
			"maxTokens":   llmSettings.MaxTokens,
		},
		"timeout": timeout,
	}

	// Add Git configuration if provided
	if req.GitConfig != nil {
		gitConfig := make(map[string]interface{})

		// Add user configuration
		if req.GitConfig.User != nil {
			user := make(map[string]interface{})
			if req.GitConfig.User.Name != "" {
				user["name"] = req.GitConfig.User.Name
			}
			if req.GitConfig.User.Email != "" {
				user["email"] = req.GitConfig.User.Email
			}
			if len(user) > 0 {
				gitConfig["user"] = user
			}
		}

		// Add authentication configuration
		if req.GitConfig.Authentication != nil {
			auth := make(map[string]interface{})
			if req.GitConfig.Authentication.SSHKeySecret != "" {
				auth["sshKeySecret"] = req.GitConfig.Authentication.SSHKeySecret
			}
			if req.GitConfig.Authentication.TokenSecret != "" {
				auth["tokenSecret"] = req.GitConfig.Authentication.TokenSecret
			}
			if req.GitConfig.Authentication.KnownHostsSecret != "" {
				auth["knownHostsSecret"] = req.GitConfig.Authentication.KnownHostsSecret
			}
			if len(auth) > 0 {
				gitConfig["authentication"] = auth
			}
		}

		// Add repositories
		if len(req.GitConfig.Repositories) > 0 {
			repositories := make([]map[string]interface{}, 0, len(req.GitConfig.Repositories))
			for _, repo := range req.GitConfig.Repositories {
				repository := map[string]interface{}{
					"url": repo.URL,
				}
				if repo.Branch != "" {
					repository["branch"] = repo.Branch
				}
				if repo.ClonePath != "" {
					repository["clonePath"] = repo.ClonePath
				}
				repositories = append(repositories, repository)
			}
			gitConfig["repositories"] = repositories
		}

		if len(gitConfig) > 0 {
			spec["gitConfig"] = gitConfig
		}
	}

	return spec
}

func deleteAgenticSession(c *gin.Context) {
	name := c.Param("name")
	gvr := getAgenticSessionResource()

	err := dynamicClient.Resource(gvr).Namespace(namespace).Delete(context.TODO(), name, v1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to delete agentic session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agentic session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agentic session deleted successfully"})
}

func updateAgenticSessionStatus(c *gin.Context) {
	name := c.Param("name")

	var statusUpdate map[string]interface{}
	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := getAgenticSessionResource()

	// Get current resource
	item, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Update status
	if item.Object["status"] == nil {
		item.Object["status"] = make(map[string]interface{})
	}

	status := item.Object["status"].(map[string]interface{})

	// Write data to files before updating CR
	writeDataToFiles(name, statusUpdate)

	for key, value := range statusUpdate {
		status[key] = value
	}

	// Update the resource
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update agentic session status %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agentic session status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agentic session status updated successfully"})
}

func updateAgenticSessionDisplayName(c *gin.Context) {
	name := c.Param("name")

	var displayNameUpdate struct {
		DisplayName string `json:"displayName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&displayNameUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := getAgenticSessionResource()

	// Get current resource
	item, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Update displayName in spec
	if item.Object["spec"] == nil {
		item.Object["spec"] = make(map[string]interface{})
	}

	spec := item.Object["spec"].(map[string]interface{})
	spec["displayName"] = displayNameUpdate.DisplayName

	// Update the resource
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update agentic session displayName %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agentic session displayName"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agentic session displayName updated successfully"})
}

func stopAgenticSession(c *gin.Context) {
	name := c.Param("name")
	gvr := getAgenticSessionResource()

	// Get current resource
	item, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Check current status
	status, ok := item.Object["status"].(map[string]interface{})
	if !ok {
		status = make(map[string]interface{})
		item.Object["status"] = status
	}

	currentPhase, _ := status["phase"].(string)
	if currentPhase == "Completed" || currentPhase == "Failed" || currentPhase == "Stopped" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot stop session in %s state", currentPhase)})
		return
	}

	log.Printf("Attempting to stop agentic session %s (current phase: %s)", name, currentPhase)

	// Get job name from status
	jobName, jobExists := status["jobName"].(string)
	if jobExists && jobName != "" {
		// Delete the job
		err := k8sClient.BatchV1().Jobs(namespace).Delete(context.TODO(), jobName, v1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Printf("Failed to delete job %s: %v", jobName, err)
			// Don't fail the request if job deletion fails - continue with status update
			log.Printf("Continuing with status update despite job deletion failure")
		} else {
			log.Printf("Deleted job %s for agentic session %s", jobName, name)
		}
	} else {
		// Handle case where job was never created or jobName is missing
		log.Printf("No job found to delete for agentic session %s", name)
	}

	// Update status to Stopped
	status["phase"] = "Stopped"
	status["message"] = "Agentic session stopped by user"
	status["completionTime"] = time.Now().Format(time.RFC3339)

	// Update the resource
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Session was deleted while we were trying to update it
			log.Printf("Agentic session %s was deleted during stop operation", name)
			c.JSON(http.StatusOK, gin.H{"message": "Agentic session no longer exists (already deleted)"})
			return
		}
		log.Printf("Failed to update agentic session status %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agentic session status"})
		return
	}

	log.Printf("Successfully stopped agentic session %s", name)
	c.JSON(http.StatusOK, gin.H{"message": "Agentic session stopped successfully"})
}

// Helper functions for parsing
func parseSpec(spec map[string]interface{}) AgenticSessionSpec {
	result := AgenticSessionSpec{}

	if prompt, ok := spec["prompt"].(string); ok {
		result.Prompt = prompt
	}

	if websiteURL, ok := spec["websiteURL"].(string); ok {
		result.WebsiteURL = websiteURL
	}

	if displayName, ok := spec["displayName"].(string); ok {
		result.DisplayName = displayName
	}

	if timeout, ok := spec["timeout"].(float64); ok {
		result.Timeout = int(timeout)
	}

	if llmSettings, ok := spec["llmSettings"].(map[string]interface{}); ok {
		if model, ok := llmSettings["model"].(string); ok {
			result.LLMSettings.Model = model
		}
		if temperature, ok := llmSettings["temperature"].(float64); ok {
			result.LLMSettings.Temperature = temperature
		}
		if maxTokens, ok := llmSettings["maxTokens"].(float64); ok {
			result.LLMSettings.MaxTokens = int(maxTokens)
		}
	}

	return result
}

// Write session data to persistent files
func writeDataToFiles(sessionName string, statusUpdate map[string]interface{}) {
	// Create session directory
	sessionDir := filepath.Join(stateBaseDir, sessionName)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		log.Printf("Warning: failed to create session directory %s: %v", sessionDir, err)
		return
	}

	// Write final output to file if present
	if finalOutput, ok := statusUpdate["finalOutput"].(string); ok && finalOutput != "" {
		finalOutputFile := filepath.Join(sessionDir, "final-output.txt")
		if err := ioutil.WriteFile(finalOutputFile, []byte(finalOutput), 0644); err != nil {
			log.Printf("Warning: failed to write final output for %s: %v", sessionName, err)
		} else {
			log.Printf("Wrote final output to file for session %s (%d chars)", sessionName, len(finalOutput))
			// Remove from status update to avoid storing in CR
			delete(statusUpdate, "finalOutput")
		}
	}

	// Write messages to file if present
	if messages, ok := statusUpdate["messages"].([]interface{}); ok && len(messages) > 0 {
		messagesFile := filepath.Join(sessionDir, "messages.json")
		if messagesBytes, err := json.MarshalIndent(messages, "", "  "); err == nil {
			if err := ioutil.WriteFile(messagesFile, messagesBytes, 0644); err != nil {
				log.Printf("Warning: failed to write messages for %s: %v", sessionName, err)
			} else {
				log.Printf("Wrote %d messages to file for session %s", len(messages), sessionName)
				// Remove from status update to avoid storing in CR
				delete(statusUpdate, "messages")
			}
		}
	}
}

// Read session data from persistent files and populate status
func readDataFromFiles(sessionName string, status *AgenticSessionStatus) {
	sessionDir := filepath.Join(stateBaseDir, sessionName)

	// Read final output from file if it exists
	finalOutputFile := filepath.Join(sessionDir, "final-output.txt")
	if finalOutputBytes, err := ioutil.ReadFile(finalOutputFile); err == nil {
		status.FinalOutput = string(finalOutputBytes)
	}

	// Read messages from file if it exists
	messagesFile := filepath.Join(sessionDir, "messages.json")
	if messagesBytes, err := ioutil.ReadFile(messagesFile); err == nil {
		var messages []MessageObject
		if err := json.Unmarshal(messagesBytes, &messages); err == nil {
			status.Messages = messages
		} else {
			log.Printf("Warning: failed to unmarshal messages for %s: %v", sessionName, err)
		}
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

	return result
}

