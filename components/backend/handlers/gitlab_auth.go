package handlers

import (
	"net/http"

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

// ConnectGitLab handles POST /auth/gitlab/connect
func (h *GitLabAuthHandler) ConnectGitLab(c *gin.Context) {
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

	// Store GitLab connection
	ctx := c.Request.Context()
	connection, err := h.connectionManager.StoreGitLabConnection(ctx, userIDStr, req.PersonalAccessToken, req.InstanceURL)
	if err != nil {
		gitlab.LogError("Failed to store GitLab connection for user %s: %v", userIDStr, err)
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
		Message:      "GitLab account connected successfully",
	})
}

// GetGitLabStatus handles GET /auth/gitlab/status
func (h *GitLabAuthHandler) GetGitLabStatus(c *gin.Context) {
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

	// Get connection status
	ctx := c.Request.Context()
	status, err := h.connectionManager.GetConnectionStatus(ctx, userIDStr)
	if err != nil {
		gitlab.LogError("Failed to get GitLab status for user %s: %v", userIDStr, err)
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

// DisconnectGitLab handles POST /auth/gitlab/disconnect
func (h *GitLabAuthHandler) DisconnectGitLab(c *gin.Context) {
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

	// Delete GitLab connection
	ctx := c.Request.Context()
	if err := h.connectionManager.DeleteGitLabConnection(ctx, userIDStr); err != nil {
		gitlab.LogError("Failed to disconnect GitLab for user %s: %v", userIDStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to disconnect GitLab account",
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "GitLab account disconnected successfully",
		"connected": false,
	})
}

// Global wrapper functions for routes

// ConnectGitLabGlobal is the global handler for POST /auth/gitlab/connect
func ConnectGitLabGlobal(c *gin.Context) {
	handler := NewGitLabAuthHandler(K8sClient, Namespace)
	handler.ConnectGitLab(c)
}

// GetGitLabStatusGlobal is the global handler for GET /auth/gitlab/status
func GetGitLabStatusGlobal(c *gin.Context) {
	handler := NewGitLabAuthHandler(K8sClient, Namespace)
	handler.GetGitLabStatus(c)
}

// DisconnectGitLabGlobal is the global handler for POST /auth/gitlab/disconnect
func DisconnectGitLabGlobal(c *gin.Context) {
	handler := NewGitLabAuthHandler(K8sClient, Namespace)
	handler.DisconnectGitLab(c)
}
