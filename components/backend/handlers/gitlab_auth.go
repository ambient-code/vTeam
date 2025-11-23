package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"

	"ambient-code-backend/gitlab"
)

// GitLabAuthHandler handles GitLab authentication endpoints
type GitLabAuthHandler struct {
	connectionManager *gitlab.ConnectionManager
}

// NewGitLabAuthHandler creates a new GitLab authentication handler
func NewGitLabAuthHandler(clientset *kubernetes.Clientset, namespace string) *GitLabAuthHandler {
	return &GitLabAuthHandler{
		connectionManager: gitlab.NewConnectionManager(clientset, namespace),
	}
}

// ConnectGitLabRequest represents a request to connect a GitLab account
type ConnectGitLabRequest struct {
	PersonalAccessToken string `json:"personalAccessToken" binding:"required"`
	InstanceURL         string `json:"instanceUrl"`
}

// ConnectGitLabResponse represents the response from connecting a GitLab account
type ConnectGitLabResponse struct {
	UserID       string `json:"userId"`
	GitLabUserID string `json:"gitlabUserId"`
	Username     string `json:"username"`
	InstanceURL  string `json:"instanceUrl"`
	Connected    bool   `json:"connected"`
	Message      string `json:"message"`
}

// GitLabStatusResponse represents the GitLab connection status
type GitLabStatusResponse struct {
	Connected    bool   `json:"connected"`
	Username     string `json:"username,omitempty"`
	InstanceURL  string `json:"instanceUrl,omitempty"`
	GitLabUserID string `json:"gitlabUserId,omitempty"`
}

// validateGitLabInput validates GitLab connection request input
func validateGitLabInput(instanceURL, token string) error {
	// Validate instance URL
	if instanceURL != "" {
		parsedURL, err := url.Parse(instanceURL)
		if err != nil {
			return fmt.Errorf("invalid instance URL format")
		}

		// Require HTTPS for security
		if parsedURL.Scheme != "https" {
			return fmt.Errorf("instance URL must use HTTPS")
		}

		// Validate hostname is not empty
		if parsedURL.Host == "" {
			return fmt.Errorf("instance URL must have a valid hostname")
		}

		// Prevent common injection attempts
		if strings.Contains(parsedURL.Host, "@") {
			return fmt.Errorf("instance URL hostname cannot contain '@'")
		}
	}

	// Validate token length (GitLab PATs are 20 chars, but allow for future changes)
	// Min: 20 chars, Max: 255 chars (reasonable upper bound)
	if len(token) < 20 {
		return fmt.Errorf("token must be at least 20 characters")
	}
	if len(token) > 255 {
		return fmt.Errorf("token must not exceed 255 characters")
	}

	// Validate token contains only valid characters (alphanumeric and some special chars)
	// GitLab tokens use: a-z, A-Z, 0-9, -, _
	for _, char := range token {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return fmt.Errorf("token contains invalid characters")
		}
	}

	return nil
}

// ConnectGitLab handles POST /projects/:projectName/auth/gitlab/connect
func (h *GitLabAuthHandler) ConnectGitLab(c *gin.Context) {
	// Get project from URL parameter
	project := c.Param("projectName")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Project name is required",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	var req ConnectGitLabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid request body",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	// Default to GitLab.com if no instance URL provided
	if req.InstanceURL == "" {
		req.InstanceURL = "https://gitlab.com"
	}

	// Validate input
	if err := validateGitLabInput(req.InstanceURL, req.PersonalAccessToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      fmt.Sprintf("Invalid input: %v", err),
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	// Get user ID from context (set by authentication middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "User not authenticated",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Invalid user ID format",
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	// RBAC: Verify user can create/update secrets in this project
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "Invalid or missing token",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	ctx := c.Request.Context()
	if err := ValidateSecretAccess(ctx, reqK8s, project, "create"); err != nil {
		gitlab.LogError("RBAC check failed for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusForbidden, gin.H{
			"error":      "Insufficient permissions to manage GitLab credentials",
			"statusCode": http.StatusForbidden,
		})
		return
	}

	// Store GitLab connection (now project-scoped)
	connection, err := h.connectionManager.StoreGitLabConnection(ctx, userIDStr, req.PersonalAccessToken, req.InstanceURL)
	if err != nil {
		gitlab.LogError("Failed to store GitLab connection for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      err.Error(),
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, ConnectGitLabResponse{
		UserID:       connection.UserID,
		GitLabUserID: connection.GitLabUserID,
		Username:     connection.Username,
		InstanceURL:  connection.InstanceURL,
		Connected:    true,
		Message:      "GitLab account connected successfully to project " + project,
	})
}

// GetGitLabStatus handles GET /projects/:projectName/auth/gitlab/status
func (h *GitLabAuthHandler) GetGitLabStatus(c *gin.Context) {
	// Get project from URL parameter
	project := c.Param("projectName")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Project name is required",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "User not authenticated",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Invalid user ID format",
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	// RBAC: Verify user can read secrets in this project
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "Invalid or missing token",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	ctx := c.Request.Context()
	if err := ValidateSecretAccess(ctx, reqK8s, project, "get"); err != nil {
		gitlab.LogError("RBAC check failed for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusForbidden, gin.H{
			"error":      "Insufficient permissions to read GitLab credentials",
			"statusCode": http.StatusForbidden,
		})
		return
	}

	// Get connection status (project-scoped)
	status, err := h.connectionManager.GetConnectionStatus(ctx, userIDStr)
	if err != nil {
		gitlab.LogError("Failed to get GitLab status for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to retrieve GitLab connection status",
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	if !status.Connected {
		c.JSON(http.StatusOK, GitLabStatusResponse{
			Connected: false,
		})
		return
	}

	c.JSON(http.StatusOK, GitLabStatusResponse{
		Connected:    true,
		Username:     status.Username,
		InstanceURL:  status.InstanceURL,
		GitLabUserID: status.GitLabUserID,
	})
}

// DisconnectGitLab handles POST /projects/:projectName/auth/gitlab/disconnect
func (h *GitLabAuthHandler) DisconnectGitLab(c *gin.Context) {
	// Get project from URL parameter
	project := c.Param("projectName")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Project name is required",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "User not authenticated",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Invalid user ID format",
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	// RBAC: Verify user can update secrets in this project
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "Invalid or missing token",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	ctx := c.Request.Context()
	if err := ValidateSecretAccess(ctx, reqK8s, project, "update"); err != nil {
		gitlab.LogError("RBAC check failed for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusForbidden, gin.H{
			"error":      "Insufficient permissions to manage GitLab credentials",
			"statusCode": http.StatusForbidden,
		})
		return
	}

	// Delete GitLab connection (project-scoped)
	if err := h.connectionManager.DeleteGitLabConnection(ctx, userIDStr); err != nil {
		gitlab.LogError("Failed to disconnect GitLab for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to disconnect GitLab account",
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "GitLab account disconnected successfully from project " + project,
		"connected": false,
	})
}

// Global wrapper functions for routes (now project-scoped)

// ConnectGitLabGlobal is the global handler for POST /projects/:projectName/auth/gitlab/connect
func ConnectGitLabGlobal(c *gin.Context) {
	// Get project from URL parameter - this is the namespace where tokens will be stored
	project := c.Param("projectName")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project name is required"})
		return
	}

	// Get user-scoped K8s client (RBAC enforcement)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Create handler with user-scoped client (multi-tenant isolation)
	handler := NewGitLabAuthHandler(reqK8s, project)
	handler.ConnectGitLab(c)
}

// GetGitLabStatusGlobal is the global handler for GET /projects/:projectName/auth/gitlab/status
func GetGitLabStatusGlobal(c *gin.Context) {
	// Get project from URL parameter
	project := c.Param("projectName")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project name is required"})
		return
	}

	// Get user-scoped K8s client (RBAC enforcement)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Create handler with user-scoped client
	handler := NewGitLabAuthHandler(reqK8s, project)
	handler.GetGitLabStatus(c)
}

// DisconnectGitLabGlobal is the global handler for POST /projects/:projectName/auth/gitlab/disconnect
func DisconnectGitLabGlobal(c *gin.Context) {
	// Get project from URL parameter
	project := c.Param("projectName")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project name is required"})
		return
	}

	// Get user-scoped K8s client (RBAC enforcement)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Create handler with user-scoped client
	handler := NewGitLabAuthHandler(reqK8s, project)
	handler.DisconnectGitLab(c)
}
