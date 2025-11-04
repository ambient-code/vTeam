package gitlab

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"k8s.io/client-go/kubernetes"

	"ambient-code-backend/k8s"
	"ambient-code-backend/types"
)

// ConnectionManager handles GitLab connection operations
type ConnectionManager struct {
	clientset *kubernetes.Clientset
	namespace string
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(clientset *kubernetes.Clientset, namespace string) *ConnectionManager {
	return &ConnectionManager{
		clientset: clientset,
		namespace: namespace,
	}
}

// StoreGitLabConnection stores a GitLab connection (metadata in ConfigMap, token in Secret)
func (cm *ConnectionManager) StoreGitLabConnection(ctx context.Context, userID, token, instanceURL string) (*types.GitLabConnection, error) {
	// Validate token and get user information
	result, err := ValidateGitLabToken(ctx, token, instanceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	if !result.Valid {
		return nil, fmt.Errorf("invalid token: %s", result.ErrorMessage)
	}

	// Create connection metadata
	connection := &types.GitLabConnection{
		UserID:       userID,
		GitLabUserID: strconv.Itoa(result.User.ID),
		InstanceURL:  instanceURL,
		Username:     result.User.Username,
		UpdatedAt:    time.Now(),
	}

	// Store token in Kubernetes Secret
	if err := k8s.StoreGitLabToken(ctx, cm.clientset, cm.namespace, userID, token); err != nil {
		return nil, fmt.Errorf("failed to store token: %w", err)
	}

	// Store connection metadata in ConfigMap
	if err := k8s.StoreGitLabConnection(ctx, cm.clientset, cm.namespace, connection); err != nil {
		// Cleanup: try to delete the token we just stored
		_ = k8s.DeleteGitLabToken(ctx, cm.clientset, cm.namespace, userID)
		return nil, fmt.Errorf("failed to store connection: %w", err)
	}

	LogInfo("GitLab connection stored for user %s (GitLab user: %s)", userID, result.User.Username)

	return connection, nil
}

// GetGitLabConnection retrieves a GitLab connection for a user
func (cm *ConnectionManager) GetGitLabConnection(ctx context.Context, userID string) (*types.GitLabConnection, error) {
	connection, err := k8s.GetGitLabConnection(ctx, cm.clientset, cm.namespace, userID)
	if err != nil {
		return nil, err
	}

	return connection, nil
}

// GetGitLabConnectionWithToken retrieves both connection metadata and token
func (cm *ConnectionManager) GetGitLabConnectionWithToken(ctx context.Context, userID string) (*types.GitLabConnection, string, error) {
	// Get connection metadata
	connection, err := k8s.GetGitLabConnection(ctx, cm.clientset, cm.namespace, userID)
	if err != nil {
		return nil, "", err
	}

	// Get token
	token, err := k8s.GetGitLabToken(ctx, cm.clientset, cm.namespace, userID)
	if err != nil {
		return nil, "", fmt.Errorf("connection exists but token not found: %w", err)
	}

	return connection, token, nil
}

// UpdateGitLabConnection updates an existing GitLab connection
func (cm *ConnectionManager) UpdateGitLabConnection(ctx context.Context, userID, token, instanceURL string) (*types.GitLabConnection, error) {
	// This is essentially the same as storing a new connection
	// It will overwrite the existing one
	return cm.StoreGitLabConnection(ctx, userID, token, instanceURL)
}

// DeleteGitLabConnection removes a GitLab connection (both metadata and token)
func (cm *ConnectionManager) DeleteGitLabConnection(ctx context.Context, userID string) error {
	// Delete token from Secret
	if err := k8s.DeleteGitLabToken(ctx, cm.clientset, cm.namespace, userID); err != nil {
		LogWarning("Failed to delete token for user %s: %v", userID, err)
		// Continue with ConfigMap deletion even if Secret deletion fails
	}

	// Delete connection metadata from ConfigMap
	if err := k8s.DeleteGitLabConnection(ctx, cm.clientset, cm.namespace, userID); err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	LogInfo("GitLab connection deleted for user %s", userID)

	return nil
}

// HasGitLabConnection checks if a user has a GitLab connection
func (cm *ConnectionManager) HasGitLabConnection(ctx context.Context, userID string) (bool, error) {
	return k8s.HasGitLabConnection(ctx, cm.clientset, cm.namespace, userID)
}

// GetConnectionStatus retrieves the connection status for a user
func (cm *ConnectionManager) GetConnectionStatus(ctx context.Context, userID string) (*ConnectionStatus, error) {
	// Check if connection exists
	hasConnection, err := cm.HasGitLabConnection(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !hasConnection {
		return &ConnectionStatus{
			Connected: false,
		}, nil
	}

	// Get connection details
	connection, err := cm.GetGitLabConnection(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if token exists
	hasToken, err := k8s.HasGitLabToken(ctx, cm.clientset, cm.namespace, userID)
	if err != nil {
		return nil, err
	}

	return &ConnectionStatus{
		Connected:    true,
		Username:     connection.Username,
		InstanceURL:  connection.InstanceURL,
		GitLabUserID: connection.GitLabUserID,
		UpdatedAt:    connection.UpdatedAt,
		HasToken:     hasToken,
	}, nil
}

// ConnectionStatus represents the status of a GitLab connection
type ConnectionStatus struct {
	Connected    bool      `json:"connected"`
	Username     string    `json:"username,omitempty"`
	InstanceURL  string    `json:"instanceUrl,omitempty"`
	GitLabUserID string    `json:"gitlabUserId,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
	HasToken     bool      `json:"hasToken"`
}

// ValidateExistingConnection validates that an existing connection still works
func (cm *ConnectionManager) ValidateExistingConnection(ctx context.Context, userID string) (bool, error) {
	// Get connection and token
	connection, token, err := cm.GetGitLabConnectionWithToken(ctx, userID)
	if err != nil {
		return false, err
	}

	// Validate the token is still valid
	result, err := ValidateGitLabToken(ctx, token, connection.InstanceURL)
	if err != nil {
		return false, err
	}

	return result.Valid, nil
}

// ListConnections returns all GitLab connections in the namespace
func (cm *ConnectionManager) ListConnections(ctx context.Context) ([]*types.GitLabConnection, error) {
	return k8s.ListGitLabConnections(ctx, cm.clientset, cm.namespace)
}
