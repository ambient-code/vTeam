package resources

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecretsReconciler handles reconciliation of secrets across namespaces
//
// The main secret (DefaultRunnerSecretsName) should contain the following keys:
//   - ANTHROPIC_API_KEY: API key for Claude/Anthropic services (required)
//   - GITHUB_TOKEN: GitHub personal access token for repository operations (optional)
//   - GIT_TOKEN: Alternative git token for non-GitHub git providers (optional)
//   - GIT_SSH_KEY: SSH private key for git operations (optional)
//
// These keys are imported into runner containers as environment variables via EnvFrom.
type SecretsReconciler struct {
	client           *kubernetes.Clientset
	operatorNS       string
	sourceNS         string
	secretsToCopy    []string
}

// NewSecretsReconciler creates a new secrets reconciler
func NewSecretsReconciler(client *kubernetes.Clientset, operatorNS string) *SecretsReconciler {
	// Get source namespace from environment or use operator namespace
	sourceNamespace := os.Getenv("SECRETS_SOURCE_NAMESPACE")
	if sourceNamespace == "" {
		sourceNamespace = operatorNS
	}

	// Define secrets to copy (configurable via environment)
	secretsToCopy := []string{DefaultRunnerSecretsName}
	if envSecrets := os.Getenv("SECRETS_TO_COPY"); envSecrets != "" {
		secretsToCopy = strings.Split(envSecrets, ",")
	}

	return &SecretsReconciler{
		client:        client,
		operatorNS:    operatorNS,
		sourceNS:      sourceNamespace,
		secretsToCopy: secretsToCopy,
	}
}

// ReconcileSecretsForNamespace copies essential secrets from the source namespace to the target namespace
func (r *SecretsReconciler) ReconcileSecretsForNamespace(targetNamespace string) error {
	// Skip copying to the operator's own namespace
	if targetNamespace == r.operatorNS {
		return nil
	}

	log.Printf("Copying secrets from namespace %s to %s: %v", r.sourceNS, targetNamespace, r.secretsToCopy)

	for _, secretName := range r.secretsToCopy {
		secretName = strings.TrimSpace(secretName)
		if secretName == "" {
			continue
		}

		if err := r.copySecret(secretName, targetNamespace); err != nil {
			return fmt.Errorf("failed to copy secret %s to namespace %s: %v", secretName, targetNamespace, err)
		}

		// Validate the copied secret if it's the main runner secret
		if secretName == DefaultRunnerSecretsName {
			if missingKeys, err := r.ValidateRunnerSecret(targetNamespace, secretName); err != nil {
				log.Printf("Warning: Could not validate secret %s in namespace %s: %v", secretName, targetNamespace, err)
			} else if len(missingKeys) > 0 {
				log.Printf("Warning: Secret %s in namespace %s is missing required keys: %v", secretName, targetNamespace, missingKeys)
			} else {
				log.Printf("Secret %s in namespace %s validated successfully", secretName, targetNamespace)
			}
		}
	}

	return nil
}

// ReconcileAnthropicAPIKey ensures the Anthropic API key secret exists in the target namespace
func (r *SecretsReconciler) ReconcileAnthropicAPIKey(targetNamespace string) error {
	// This could be part of the general secrets copy, but having it as a separate method
	// allows for more specific handling of the Anthropic API key if needed
	return r.copySecretIfNotExists(DefaultRunnerSecretsName, targetNamespace)
}

// copySecret copies a single secret from source to target namespace
func (r *SecretsReconciler) copySecret(secretName, targetNamespace string) error {
	// Get source secret
	sourceSecret, err := r.client.CoreV1().Secrets(r.sourceNS).Get(context.TODO(), secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Source secret %s not found in namespace %s, skipping", secretName, r.sourceNS)
			return nil // Not an error - secret might be optional
		}
		return fmt.Errorf("failed to get source secret %s from namespace %s: %v", secretName, r.sourceNS, err)
	}

	// Check if target secret already exists
	_, err = r.client.CoreV1().Secrets(targetNamespace).Get(context.TODO(), secretName, v1.GetOptions{})
	if err == nil {
		log.Printf("Secret %s already exists in namespace %s, skipping", secretName, targetNamespace)
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("error checking if secret %s exists in namespace %s: %v", secretName, targetNamespace, err)
	}

	// Create target secret (copy data but reset metadata)
	targetSecret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      secretName,
			Namespace: targetNamespace,
			Labels: map[string]string{
				ManagedLabelKey:      ManagedLabelValue,
				CopiedFromLabelKey:   r.sourceNS,
				SecretTypeLabelKey:   RunnerSecretsLabelValue,
			},
			Annotations: map[string]string{
				CopiedAtAnnotationKey: time.Now().Format(time.RFC3339),
			},
		},
		Type: sourceSecret.Type,
		Data: sourceSecret.Data,
	}

	// Create the secret in target namespace
	_, err = r.client.CoreV1().Secrets(targetNamespace).Create(context.TODO(), targetSecret, v1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Printf("Secret %s already exists in namespace %s", secretName, targetNamespace)
			return nil
		}
		return fmt.Errorf("failed to create secret %s in namespace %s: %v", secretName, targetNamespace, err)
	}

	log.Printf("Successfully copied secret %s from %s to %s", secretName, r.sourceNS, targetNamespace)
	return nil
}

// copySecretIfNotExists copies a secret only if it doesn't exist in the target namespace
func (r *SecretsReconciler) copySecretIfNotExists(secretName, targetNamespace string) error {
	// Check if target secret already exists
	_, err := r.client.CoreV1().Secrets(targetNamespace).Get(context.TODO(), secretName, v1.GetOptions{})
	if err == nil {
		// Secret already exists, nothing to do
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("error checking if secret %s exists in namespace %s: %v", secretName, targetNamespace, err)
	}

	// Secret doesn't exist, copy it
	return r.copySecret(secretName, targetNamespace)
}

// UpdateSecretsConfiguration updates the secrets configuration (e.g., when environment variables change)
func (r *SecretsReconciler) UpdateSecretsConfiguration() {
	// Get source namespace from environment or use operator namespace
	sourceNamespace := os.Getenv("SECRETS_SOURCE_NAMESPACE")
	if sourceNamespace == "" {
		sourceNamespace = r.operatorNS
	}
	r.sourceNS = sourceNamespace

	// Define secrets to copy (configurable via environment)
	secretsToCopy := []string{DefaultRunnerSecretsName}
	if envSecrets := os.Getenv("SECRETS_TO_COPY"); envSecrets != "" {
		secretsToCopy = strings.Split(envSecrets, ",")
	}
	r.secretsToCopy = secretsToCopy

	log.Printf("Updated secrets configuration: source=%s, secrets=%v", r.sourceNS, r.secretsToCopy)
}

// ListManagedSecrets returns a list of secrets managed by this reconciler in the given namespace
func (r *SecretsReconciler) ListManagedSecrets(namespace string) ([]corev1.Secret, error) {
	secrets, err := r.client.CoreV1().Secrets(namespace).List(context.TODO(), v1.ListOptions{
		LabelSelector: ManagedLabelKey + "=" + ManagedLabelValue + "," + SecretTypeLabelKey + "=" + RunnerSecretsLabelValue,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list managed secrets in namespace %s: %v", namespace, err)
	}

	return secrets.Items, nil
}

// DeleteManagedSecrets removes all managed secrets from the given namespace
func (r *SecretsReconciler) DeleteManagedSecrets(namespace string) error {
	secrets, err := r.ListManagedSecrets(namespace)
	if err != nil {
		return fmt.Errorf("failed to list managed secrets for deletion: %v", err)
	}

	for _, secret := range secrets {
		err := r.client.CoreV1().Secrets(namespace).Delete(context.TODO(), secret.Name, v1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Printf("Failed to delete managed secret %s in namespace %s: %v", secret.Name, namespace, err)
		} else {
			log.Printf("Deleted managed secret %s from namespace %s", secret.Name, namespace)
		}
	}

	return nil
}

// ValidateRunnerSecret checks if the runner secret contains required keys
func (r *SecretsReconciler) ValidateRunnerSecret(namespace, secretName string) ([]string, error) {
	secret, err := r.client.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s in namespace %s: %v", secretName, namespace, err)
	}

	var missingKeys []string
	requiredKeys := []string{AnthropicAPIKeySecretKey}
	optionalKeys := []string{GitHubTokenSecretKey, GitTokenSecretKey, GitSSHKeySecretKey}

	// Check required keys
	for _, key := range requiredKeys {
		if _, exists := secret.Data[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}

	// Git authentication is optional - log info about what's available
	gitAuthMethods := []string{}
	for _, key := range optionalKeys {
		if _, exists := secret.Data[key]; exists {
			gitAuthMethods = append(gitAuthMethods, key)
		}
	}

	if len(gitAuthMethods) == 0 {
		log.Printf("Info: No git authentication found in secret %s/%s - git operations may not work", namespace, secretName)
	} else {
		log.Printf("Info: Git authentication available in secret %s/%s: %v", namespace, secretName, gitAuthMethods)
	}

	return missingKeys, nil
}

// GetExpectedSecretKeys returns the list of expected secret keys with descriptions
func (r *SecretsReconciler) GetExpectedSecretKeys() map[string]string {
	return map[string]string{
		AnthropicAPIKeySecretKey: "API key for Claude/Anthropic services (required)",
		GitHubTokenSecretKey:     "GitHub personal access token for repository operations (optional)",
		GitTokenSecretKey:        "Git token for non-GitHub providers (optional)",
		GitSSHKeySecretKey:       "SSH private key for git operations (optional)",
	}
}

// GetRequiredSecretKeys returns only the required secret keys
func (r *SecretsReconciler) GetRequiredSecretKeys() []string {
	return []string{AnthropicAPIKeySecretKey}
}

// GetOptionalSecretKeys returns only the optional secret keys
func (r *SecretsReconciler) GetOptionalSecretKeys() []string {
	return []string{GitHubTokenSecretKey, GitTokenSecretKey, GitSSHKeySecretKey}
}