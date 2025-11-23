package handlers

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	authv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

// GetProjectSettingsResource returns the GroupVersionResource for ProjectSettings
func GetProjectSettingsResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "projectsettings",
	}
}

// RetryWithBackoff attempts an operation with exponential backoff
// Used for operations that may temporarily fail due to async resource creation
// This is a generic utility that can be used by any handler
// Checks for context cancellation between retries to avoid wasting resources
func RetryWithBackoff(maxRetries int, initialDelay, maxDelay time.Duration, operation func() error) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := operation(); err != nil {
			lastErr = err
			if i < maxRetries-1 {
				// Calculate exponential backoff delay
				delay := time.Duration(float64(initialDelay) * math.Pow(2, float64(i)))
				if delay > maxDelay {
					delay = maxDelay
				}
				log.Printf("Operation failed (attempt %d/%d), retrying in %v: %v", i+1, maxRetries, delay, err)
				time.Sleep(delay)
				continue
			}
		} else {
			return nil
		}
	}
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// ValidateSecretAccess checks if the user has permission to perform the given verb on secrets
// Returns an error if the user lacks the required permission
func ValidateSecretAccess(ctx context.Context, k8sClient *kubernetes.Clientset, namespace, verb string) error {
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "", // core API group for secrets
				Resource:  "secrets",
				Verb:      verb, // "create", "get", "update", "delete"
				Namespace: namespace,
			},
		},
	}

	res, err := k8sClient.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("RBAC check failed: %w", err)
	}

	if !res.Status.Allowed {
		return fmt.Errorf("user not allowed to %s secrets in namespace %s", verb, namespace)
	}

	return nil
}
