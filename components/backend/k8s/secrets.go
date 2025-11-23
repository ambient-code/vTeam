package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// GitLabTokensSecretName is the name of the secret storing GitLab PATs
	GitLabTokensSecretName = "gitlab-user-tokens"
)

// StoreGitLabToken stores a GitLab Personal Access Token in Kubernetes Secrets
// Uses optimistic concurrency control with retry to handle concurrent updates
func StoreGitLabToken(ctx context.Context, clientset *kubernetes.Clientset, namespace, userID, token string) error {
	secretsClient := clientset.CoreV1().Secrets(namespace)

	// Retry up to 3 times with exponential backoff
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Get existing secret or create new one
		secret, err := secretsClient.Get(ctx, GitLabTokensSecretName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			// Create new secret
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GitLabTokensSecretName,
					Namespace: namespace,
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					userID: token,
				},
			}

			_, err = secretsClient.Create(ctx, secret, metav1.CreateOptions{})
			if err != nil && !errors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create GitLab tokens secret: %w", err)
			}
			if err == nil {
				return nil
			}
			// If AlreadyExists, retry the Get-Update loop
			lastErr = err
			time.Sleep(time.Millisecond * 100 * time.Duration(attempt+1))
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get GitLab tokens secret: %w", err)
		}

		// Update existing secret
		// Make a deep copy to avoid modifying the original
		secretCopy := secret.DeepCopy()

		// Update the data in the copy
		if secretCopy.Data == nil {
			secretCopy.Data = make(map[string][]byte)
		}
		secretCopy.Data[userID] = []byte(token)

		// Attempt update with current ResourceVersion (optimistic concurrency)
		_, err = secretsClient.Update(ctx, secretCopy, metav1.UpdateOptions{})
		if err == nil {
			return nil
		}

		// If conflict, retry
		if errors.IsConflict(err) {
			lastErr = err
			// Exponential backoff: 100ms, 200ms, 400ms
			time.Sleep(time.Millisecond * 100 * time.Duration(attempt+1))
			continue
		}

		// Other errors are not retryable
		return fmt.Errorf("failed to update GitLab tokens secret: %w", err)
	}

	return fmt.Errorf("failed to update GitLab tokens secret after %d retries: %w", maxRetries, lastErr)
}

// GetGitLabToken retrieves a GitLab Personal Access Token from Kubernetes Secrets
func GetGitLabToken(ctx context.Context, clientset *kubernetes.Clientset, namespace, userID string) (string, error) {
	secretsClient := clientset.CoreV1().Secrets(namespace)

	secret, err := secretsClient.Get(ctx, GitLabTokensSecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("GitLab tokens secret not found")
		}
		return "", fmt.Errorf("failed to get GitLab tokens secret: %w", err)
	}

	tokenBytes, exists := secret.Data[userID]
	if !exists {
		return "", fmt.Errorf("no GitLab token found for user %s", userID)
	}

	return string(tokenBytes), nil
}

// DeleteGitLabToken removes a GitLab Personal Access Token from Kubernetes Secrets
// Uses optimistic concurrency control with retry to handle concurrent updates
func DeleteGitLabToken(ctx context.Context, clientset *kubernetes.Clientset, namespace, userID string) error {
	secretsClient := clientset.CoreV1().Secrets(namespace)

	// Retry up to 3 times with exponential backoff
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		secret, err := secretsClient.Get(ctx, GitLabTokensSecretName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil // Already doesn't exist
			}
			return fmt.Errorf("failed to get GitLab tokens secret: %w", err)
		}

		if secret.Data == nil || secret.Data[userID] == nil {
			return nil // No data to delete
		}

		// Make a deep copy to avoid modifying the original
		secretCopy := secret.DeepCopy()
		delete(secretCopy.Data, userID)

		// Attempt update with current ResourceVersion (optimistic concurrency)
		_, err = secretsClient.Update(ctx, secretCopy, metav1.UpdateOptions{})
		if err == nil {
			return nil
		}

		// If conflict, retry
		if errors.IsConflict(err) {
			lastErr = err
			// Exponential backoff: 100ms, 200ms, 400ms
			time.Sleep(time.Millisecond * 100 * time.Duration(attempt+1))
			continue
		}

		// Other errors are not retryable
		return fmt.Errorf("failed to update GitLab tokens secret: %w", err)
	}

	return fmt.Errorf("failed to delete GitLab token after %d retries: %w", maxRetries, lastErr)
}

// HasGitLabToken checks if a user has a GitLab token stored
func HasGitLabToken(ctx context.Context, clientset *kubernetes.Clientset, namespace, userID string) (bool, error) {
	secretsClient := clientset.CoreV1().Secrets(namespace)

	secret, err := secretsClient.Get(ctx, GitLabTokensSecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get GitLab tokens secret: %w", err)
	}

	_, exists := secret.Data[userID]
	return exists, nil
}
