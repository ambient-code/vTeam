package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
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
		api.GET("/research-sessions", listResearchSessions)
		api.GET("/research-sessions/:name", getResearchSession)
		api.POST("/research-sessions", createResearchSession)
		api.DELETE("/research-sessions/:name", deleteResearchSession)
		api.PUT("/research-sessions/:name/status", updateResearchSessionStatus)
		api.PUT("/research-sessions/:name/displayname", updateResearchSessionDisplayName)
		api.POST("/research-sessions/:name/stop", stopResearchSession)
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

// ResearchSession represents the structure of our custom resource
type ResearchSession struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       ResearchSessionSpec    `json:"spec"`
	Status     *ResearchSessionStatus `json:"status,omitempty"`
}

type ResearchSessionSpec struct {
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

type MessageObject struct {
	Content        string `json:"content,omitempty"`
	ToolUseID      string `json:"tool_use_id,omitempty"`
	ToolUseName    string `json:"tool_use_name,omitempty"`
	ToolUseInput   string `json:"tool_use_input,omitempty"`
	ToolUseIsError *bool  `json:"tool_use_is_error,omitempty"`
}

type ResearchSessionStatus struct {
	Phase          string          `json:"phase,omitempty"`
	Message        string          `json:"message,omitempty"`
	StartTime      *string         `json:"startTime,omitempty"`
	CompletionTime *string         `json:"completionTime,omitempty"`
	JobName        string          `json:"jobName,omitempty"`
	FinalOutput    string          `json:"finalOutput,omitempty"`
	Cost           *float64        `json:"cost,omitempty"`
	Messages       []MessageObject `json:"messages,omitempty"`
}

type CreateResearchSessionRequest struct {
	Prompt      string       `json:"prompt" binding:"required"`
	WebsiteURL  string       `json:"websiteURL" binding:"required,url"`
	DisplayName string       `json:"displayName,omitempty"`
	LLMSettings *LLMSettings `json:"llmSettings,omitempty"`
	Timeout     *int         `json:"timeout,omitempty"`
}

// getResearchSessionResource returns the GroupVersionResource for ResearchSession
func getResearchSessionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "research.example.com",
		Version:  "v1",
		Resource: "researchsessions",
	}
}

func listResearchSessions(c *gin.Context) {
	gvr := getResearchSessionResource()

	list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list research sessions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list research sessions"})
		return
	}

	var sessions []ResearchSession
	for _, item := range list.Items {
		session := ResearchSession{
			APIVersion: item.GetAPIVersion(),
			Kind:       item.GetKind(),
			Metadata:   item.Object["metadata"].(map[string]interface{}),
		}

		if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
			session.Spec = parseSpec(spec)
		}

		if status, ok := item.Object["status"].(map[string]interface{}); ok {
			session.Status = parseStatus(status)
		}

		sessions = append(sessions, session)
	}

	c.JSON(http.StatusOK, sessions)
}

func getResearchSession(c *gin.Context) {
	name := c.Param("name")
	gvr := getResearchSessionResource()

	item, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Research session not found"})
			return
		}
		log.Printf("Failed to get research session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get research session"})
		return
	}

	session := ResearchSession{
		APIVersion: item.GetAPIVersion(),
		Kind:       item.GetKind(),
		Metadata:   item.Object["metadata"].(map[string]interface{}),
	}

	if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := item.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusOK, session)
}

func createResearchSession(c *gin.Context) {
	var req CreateResearchSessionRequest
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
	name := fmt.Sprintf("research-session-%d", timestamp)

	// Create the custom resource
	session := map[string]interface{}{
		"apiVersion": "research.example.com/v1",
		"kind":       "ResearchSession",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"prompt":      req.Prompt,
			"websiteURL":  req.WebsiteURL,
			"displayName": req.DisplayName,
			"llmSettings": map[string]interface{}{
				"model":       llmSettings.Model,
				"temperature": llmSettings.Temperature,
				"maxTokens":   llmSettings.MaxTokens,
			},
			"timeout": timeout,
		},
		"status": map[string]interface{}{
			"phase": "Pending",
		},
	}

	gvr := getResearchSessionResource()
	obj := &unstructured.Unstructured{Object: session}

	created, err := dynamicClient.Resource(gvr).Namespace(namespace).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create research session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create research session"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Research session created successfully",
		"name":    name,
		"uid":     created.GetUID(),
	})
}

func deleteResearchSession(c *gin.Context) {
	name := c.Param("name")
	gvr := getResearchSessionResource()

	err := dynamicClient.Resource(gvr).Namespace(namespace).Delete(context.TODO(), name, v1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Research session not found"})
			return
		}
		log.Printf("Failed to delete research session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete research session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Research session deleted successfully"})
}

func updateResearchSessionStatus(c *gin.Context) {
	name := c.Param("name")

	var statusUpdate map[string]interface{}
	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := getResearchSessionResource()

	// Get current resource
	item, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Research session not found"})
			return
		}
		log.Printf("Failed to get research session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get research session"})
		return
	}

	// Update status
	if item.Object["status"] == nil {
		item.Object["status"] = make(map[string]interface{})
	}

	status := item.Object["status"].(map[string]interface{})
	for key, value := range statusUpdate {
		status[key] = value
	}

	// Update the resource
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update research session status %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update research session status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Research session status updated successfully"})
}

func updateResearchSessionDisplayName(c *gin.Context) {
	name := c.Param("name")

	var displayNameUpdate struct {
		DisplayName string `json:"displayName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&displayNameUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := getResearchSessionResource()

	// Get current resource
	item, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Research session not found"})
			return
		}
		log.Printf("Failed to get research session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get research session"})
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
		log.Printf("Failed to update research session displayName %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update research session displayName"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Research session displayName updated successfully"})
}

func stopResearchSession(c *gin.Context) {
	name := c.Param("name")
	gvr := getResearchSessionResource()

	// Get current resource
	item, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Research session not found"})
			return
		}
		log.Printf("Failed to get research session %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get research session"})
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

	log.Printf("Attempting to stop research session %s (current phase: %s)", name, currentPhase)

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
			log.Printf("Deleted job %s for research session %s", jobName, name)
		}
	} else {
		// Handle case where job was never created or jobName is missing
		log.Printf("No job found to delete for research session %s", name)
	}

	// Update status to Stopped
	status["phase"] = "Stopped"
	status["message"] = "Research session stopped by user"
	status["completionTime"] = time.Now().Format(time.RFC3339)

	// Update the resource
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Session was deleted while we were trying to update it
			log.Printf("Research session %s was deleted during stop operation", name)
			c.JSON(http.StatusOK, gin.H{"message": "Research session no longer exists (already deleted)"})
			return
		}
		log.Printf("Failed to update research session status %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update research session status"})
		return
	}

	log.Printf("Successfully stopped research session %s", name)
	c.JSON(http.StatusOK, gin.H{"message": "Research session stopped successfully"})
}

// Helper functions for parsing
func parseSpec(spec map[string]interface{}) ResearchSessionSpec {
	result := ResearchSessionSpec{}

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

func parseStatus(status map[string]interface{}) *ResearchSessionStatus {
	result := &ResearchSessionStatus{}

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
