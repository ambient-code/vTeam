package k8s

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"ambient-code-backend/types"
)

const (
	// GitLabConnectionsConfigMapName is the name of the ConfigMap storing GitLab connection metadata
	GitLabConnectionsConfigMapName = "gitlab-connections"
)

// StoreGitLabConnection stores GitLab connection metadata in a ConfigMap
func StoreGitLabConnection(ctx context.Context, clientset *kubernetes.Clientset, namespace string, connection *types.GitLabConnection) error {
	configMapsClient := clientset.CoreV1().ConfigMaps(namespace)

	// Serialize connection to JSON
	connectionJSON, err := json.Marshal(connection)
	if err != nil {
		return fmt.Errorf("failed to serialize connection: %w", err)
	}

	// Get existing ConfigMap or create new one
	configMap, err := configMapsClient.Get(ctx, GitLabConnectionsConfigMapName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// Create new ConfigMap
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GitLabConnectionsConfigMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				connection.UserID: string(connectionJSON),
			},
		}

		_, err = configMapsClient.Create(ctx, configMap, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create GitLab connections ConfigMap: %w", err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get GitLab connections ConfigMap: %w", err)
	}

	// Update existing ConfigMap
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	configMap.Data[connection.UserID] = string(connectionJSON)

	_, err = configMapsClient.Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update GitLab connections ConfigMap: %w", err)
	}

	return nil
}

// GetGitLabConnection retrieves GitLab connection metadata from a ConfigMap
func GetGitLabConnection(ctx context.Context, clientset *kubernetes.Clientset, namespace, userID string) (*types.GitLabConnection, error) {
	configMapsClient := clientset.CoreV1().ConfigMaps(namespace)

	configMap, err := configMapsClient.Get(ctx, GitLabConnectionsConfigMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("GitLab connections ConfigMap not found")
		}
		return nil, fmt.Errorf("failed to get GitLab connections ConfigMap: %w", err)
	}

	connectionJSON, exists := configMap.Data[userID]
	if !exists {
		return nil, fmt.Errorf("no GitLab connection found for user %s", userID)
	}

	var connection types.GitLabConnection
	if err := json.Unmarshal([]byte(connectionJSON), &connection); err != nil {
		return nil, fmt.Errorf("failed to parse connection data: %w", err)
	}

	return &connection, nil
}

// DeleteGitLabConnection removes GitLab connection metadata from a ConfigMap
func DeleteGitLabConnection(ctx context.Context, clientset *kubernetes.Clientset, namespace, userID string) error {
	configMapsClient := clientset.CoreV1().ConfigMaps(namespace)

	configMap, err := configMapsClient.Get(ctx, GitLabConnectionsConfigMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // Already doesn't exist
		}
		return fmt.Errorf("failed to get GitLab connections ConfigMap: %w", err)
	}

	if configMap.Data == nil {
		return nil // No data to delete
	}

	delete(configMap.Data, userID)

	_, err = configMapsClient.Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update GitLab connections ConfigMap: %w", err)
	}

	return nil
}

// HasGitLabConnection checks if a user has a GitLab connection stored
func HasGitLabConnection(ctx context.Context, clientset *kubernetes.Clientset, namespace, userID string) (bool, error) {
	configMapsClient := clientset.CoreV1().ConfigMaps(namespace)

	configMap, err := configMapsClient.Get(ctx, GitLabConnectionsConfigMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get GitLab connections ConfigMap: %w", err)
	}

	_, exists := configMap.Data[userID]
	return exists, nil
}

// ListGitLabConnections retrieves all GitLab connections from a ConfigMap
func ListGitLabConnections(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]*types.GitLabConnection, error) {
	configMapsClient := clientset.CoreV1().ConfigMaps(namespace)

	configMap, err := configMapsClient.Get(ctx, GitLabConnectionsConfigMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return []*types.GitLabConnection{}, nil
		}
		return nil, fmt.Errorf("failed to get GitLab connections ConfigMap: %w", err)
	}

	connections := make([]*types.GitLabConnection, 0, len(configMap.Data))

	for _, connectionJSON := range configMap.Data {
		var connection types.GitLabConnection
		if err := json.Unmarshal([]byte(connectionJSON), &connection); err != nil {
			// Skip invalid entries
			continue
		}
		connections = append(connections, &connection)
	}

	return connections, nil
}
