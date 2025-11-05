package k8s

import (
	"context"
	"fmt"

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
func StoreGitLabToken(ctx context.Context, clientset *kubernetes.Clientset, namespace, userID, token string) error {
	secretsClient := clientset.CoreV1().Secrets(namespace)

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
		if err != nil {
			return fmt.Errorf("failed to create GitLab tokens secret: %w", err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get GitLab tokens secret: %w", err)
	}

	// Update existing secret
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	if secret.StringData == nil {
		secret.StringData = make(map[string]string)
	}

	secret.StringData[userID] = token

	_, err = secretsClient.Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update GitLab tokens secret: %w", err)
	}

	return nil
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
func DeleteGitLabToken(ctx context.Context, clientset *kubernetes.Clientset, namespace, userID string) error {
	secretsClient := clientset.CoreV1().Secrets(namespace)

	secret, err := secretsClient.Get(ctx, GitLabTokensSecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // Already doesn't exist
		}
		return fmt.Errorf("failed to get GitLab tokens secret: %w", err)
	}

	if secret.Data == nil {
		return nil // No data to delete
	}

	delete(secret.Data, userID)

	_, err = secretsClient.Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update GitLab tokens secret: %w", err)
	}

	return nil
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
