// Sessions API Component
// C4 Architecture: Agentic session lifecycle management
// Manages AgenticSession Custom Resources

package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// SessionsAPI handles agentic session lifecycle management
type SessionsAPI struct {
	dynamicClient dynamic.Interface
	gvr          schema.GroupVersionResource
}

// NewSessionsAPI creates a new Sessions API handler
func NewSessionsAPI(client dynamic.Interface) *SessionsAPI {
	return &SessionsAPI{
		dynamicClient: client,
		gvr: schema.GroupVersionResource{
			Group:    "vteam.io",
			Version:  "v1alpha1",
			Resource: "agenticsessions",
		},
	}
}

// AgenticSession represents an AI-powered automation session
type AgenticSession struct {
	Name        string                 `json:"name"`
	Project     string                 `json:"project"`
	Description string                 `json:"description"`
	Task        string                 `json:"task"`
	Agent       string                 `json:"agent"`
	Parameters  map[string]interface{} `json:"parameters"`
	Secrets     []string               `json:"secrets"`
	Priority    string                 `json:"priority"`
	Timeout     int                    `json:"timeout"`
}

// SessionStatus represents the current state of a session
type SessionStatus struct {
	Phase      string    `json:"phase"`
	Message    string    `json:"message"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime,omitempty"`
	Output     string    `json:"output,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// ListSessions returns all sessions for a project
func (s *SessionsAPI) ListSessions(c *gin.Context) {
	projectName := c.Param("project")
	namespace := fmt.Sprintf("vteam-%s", projectName)

	// List sessions in project namespace
	sessions, err := s.dynamicClient.Resource(s.gvr).Namespace(namespace).List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("project=%s", projectName),
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to list sessions: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, sessions.Items)
}

// CreateSession creates a new agentic session
func (s *SessionsAPI) CreateSession(c *gin.Context) {
	projectName := c.Param("project")
	namespace := fmt.Sprintf("vteam-%s", projectName)

	var session AgenticSession
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid session data: %v", err),
		})
		return
	}

	// Set defaults
	if session.Priority == "" {
		session.Priority = "normal"
	}
	if session.Timeout == 0 {
		session.Timeout = 3600 // 1 hour default
	}
	if session.Agent == "" {
		session.Agent = "general-purpose"
	}

	// Create AgenticSession Custom Resource
	sessionCR := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.io/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"generateName": fmt.Sprintf("%s-", session.Name),
				"namespace":    namespace,
				"labels": map[string]interface{}{
					"project":  projectName,
					"app":      "vteam",
					"agent":    session.Agent,
					"priority": session.Priority,
					"user":     c.GetString("user"),
				},
			},
			"spec": map[string]interface{}{
				"description": session.Description,
				"task":        session.Task,
				"agent":       session.Agent,
				"parameters":  session.Parameters,
				"secrets":     session.Secrets,
				"timeout":     session.Timeout,
				"runner": map[string]interface{}{
					"image": "quay.io/ambient_code/claude-runner:latest",
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"memory": "2Gi",
							"cpu":    "1",
						},
						"limits": map[string]interface{}{
							"memory": "4Gi",
							"cpu":    "2",
						},
					},
				},
			},
			"status": map[string]interface{}{
				"phase":     "Pending",
				"message":   "Session created, waiting for operator to schedule",
				"startTime": time.Now().Format(time.RFC3339),
			},
		},
	}

	// Create the AgenticSession CR
	created, err := s.dynamicClient.Resource(s.gvr).Namespace(namespace).Create(
		context.TODO(),
		sessionCR,
		metav1.CreateOptions{},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create session: %v", err),
		})
		return
	}

	c.JSON(http.StatusCreated, created.Object)
}

// GetSession retrieves a specific session
func (s *SessionsAPI) GetSession(c *gin.Context) {
	projectName := c.Param("project")
	sessionName := c.Param("name")
	namespace := fmt.Sprintf("vteam-%s", projectName)

	session, err := s.dynamicClient.Resource(s.gvr).Namespace(namespace).Get(
		context.TODO(),
		sessionName,
		metav1.GetOptions{},
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Session not found: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, session.Object)
}

// UpdateSession updates session parameters (limited to pending sessions)
func (s *SessionsAPI) UpdateSession(c *gin.Context) {
	projectName := c.Param("project")
	sessionName := c.Param("name")
	namespace := fmt.Sprintf("vteam-%s", projectName)

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid update data: %v", err),
		})
		return
	}

	// Get existing session
	existing, err := s.dynamicClient.Resource(s.gvr).Namespace(namespace).Get(
		context.TODO(),
		sessionName,
		metav1.GetOptions{},
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Session not found",
		})
		return
	}

	// Check if session is still pending
	phase, _, _ := unstructured.NestedString(existing.Object, "status", "phase")
	if phase != "Pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Can only update sessions in Pending state",
		})
		return
	}

	// Update spec fields
	if spec, ok := updates["spec"].(map[string]interface{}); ok {
		for key, value := range spec {
			unstructured.SetNestedField(existing.Object, value, "spec", key)
		}
	}

	// Update the AgenticSession CR
	updated, err := s.dynamicClient.Resource(s.gvr).Namespace(namespace).Update(
		context.TODO(),
		existing,
		metav1.UpdateOptions{},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update session: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, updated.Object)
}

// DeleteSession cancels and deletes a session
func (s *SessionsAPI) DeleteSession(c *gin.Context) {
	projectName := c.Param("project")
	sessionName := c.Param("name")
	namespace := fmt.Sprintf("vteam-%s", projectName)

	// Get session to check ownership
	session, err := s.dynamicClient.Resource(s.gvr).Namespace(namespace).Get(
		context.TODO(),
		sessionName,
		metav1.GetOptions{},
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Session not found",
		})
		return
	}

	// Check ownership
	labels := session.GetLabels()
	if labels["user"] != c.GetString("user") && !c.GetBool("isAdmin") {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only session owner can delete the session",
		})
		return
	}

	// Delete the AgenticSession CR
	err = s.dynamicClient.Resource(s.gvr).Namespace(namespace).Delete(
		context.TODO(),
		sessionName,
		metav1.DeleteOptions{},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to delete session: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Session %s deleted successfully", sessionName),
	})
}

// GetSessionStatus retrieves real-time session status
func (s *SessionsAPI) GetSessionStatus(c *gin.Context) {
	projectName := c.Param("project")
	sessionName := c.Param("name")
	namespace := fmt.Sprintf("vteam-%s", projectName)

	session, err := s.dynamicClient.Resource(s.gvr).Namespace(namespace).Get(
		context.TODO(),
		sessionName,
		metav1.GetOptions{},
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Session not found: %v", err),
		})
		return
	}

	// Extract status fields
	status, found, err := unstructured.NestedMap(session.Object, "status")
	if !found || err != nil {
		c.JSON(http.StatusOK, gin.H{
			"phase":   "Unknown",
			"message": "Status not available",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}