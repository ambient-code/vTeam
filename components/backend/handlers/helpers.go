package handlers

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetProjectSettingsResource returns the GroupVersionResource for ProjectSettings CRD.
// Returns a GVR for group "vteam.ambient-code", version "v1alpha1", resource "projectsettings".
func GetProjectSettingsResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "projectsettings",
	}
}

// RetryWithBackoff attempts an operation with exponential backoff.
// Used for operations that may temporarily fail due to async resource creation.
//
// Parameters:
//   - maxRetries: Maximum number of retry attempts
//   - initialDelay: Initial delay duration before first retry
//   - maxDelay: Maximum delay duration cap for exponential backoff
//   - operation: Function to execute that returns an error if it fails
//
// Returns an error if all retries are exhausted, nil if operation succeeds.
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

// GetMetadataMap safely extracts the metadata field from an unstructured Kubernetes object.
// This function provides type-safe access with nil checking to prevent runtime panics.
//
// Parameters:
//   - obj: Pointer to an unstructured.Unstructured object (Kubernetes Custom Resource)
//
// Returns:
//   - map[string]interface{}: The metadata map if extraction succeeds, nil otherwise
//   - bool: true if metadata was successfully extracted and has the expected type, false otherwise
//
// Use this instead of unsafe type assertions like: obj.Object["metadata"].(map[string]interface{})
func GetMetadataMap(obj *unstructured.Unstructured) (map[string]interface{}, bool) {
	if obj == nil || obj.Object == nil {
		return nil, false
	}
	metadata, ok := obj.Object["metadata"].(map[string]interface{})
	return metadata, ok
}

// GetSpecMap safely extracts the spec field from an unstructured Kubernetes object.
// This function provides type-safe access with nil checking to prevent runtime panics.
//
// Parameters:
//   - obj: Pointer to an unstructured.Unstructured object (Kubernetes Custom Resource)
//
// Returns:
//   - map[string]interface{}: The spec map if extraction succeeds, nil otherwise
//   - bool: true if spec was successfully extracted and has the expected type, false otherwise
//
// Use this instead of unsafe type assertions like: obj.Object["spec"].(map[string]interface{})
func GetSpecMap(obj *unstructured.Unstructured) (map[string]interface{}, bool) {
	if obj == nil || obj.Object == nil {
		return nil, false
	}
	spec, ok := obj.Object["spec"].(map[string]interface{})
	return spec, ok
}

// GetStatusMap safely extracts the status field from an unstructured Kubernetes object.
// This function provides type-safe access with nil checking to prevent runtime panics.
//
// Parameters:
//   - obj: Pointer to an unstructured.Unstructured object (Kubernetes Custom Resource)
//
// Returns:
//   - map[string]interface{}: The status map if extraction succeeds, nil otherwise
//   - bool: true if status was successfully extracted and has the expected type, false otherwise
//
// Use this instead of unsafe type assertions like: obj.Object["status"].(map[string]interface{})
func GetStatusMap(obj *unstructured.Unstructured) (map[string]interface{}, bool) {
	if obj == nil || obj.Object == nil {
		return nil, false
	}
	status, ok := obj.Object["status"].(map[string]interface{})
	return status, ok
}

// ResolveContentServiceName determines the correct content service name for a session.
// This function handles the logic of choosing between temp-content and ambient-content services.
//
// For completed/stopped sessions, a temporary service (temp-content-{session}) is created.
// For running sessions, the regular service (ambient-content-{session}) is used.
// This function tries the temp service first, and falls back to the regular service if not found.
//
// Parameters:
//   - c: Gin context for the current HTTP request
//   - project: The Kubernetes namespace/project name
//   - session: The agentic session name
//
// Returns:
//   - string: The resolved service name (either temp-content-{session} or ambient-content-{session})
func ResolveContentServiceName(c *gin.Context, project, session string) string {
	// Try temp service first (for completed sessions)
	tempServiceName := fmt.Sprintf("temp-content-%s", session)

	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		// Check if temp service exists
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), tempServiceName, v1.GetOptions{}); err == nil {
			// Temp service exists, use it
			return tempServiceName
		}
	}

	// Fall back to regular service (for running sessions or if temp service doesn't exist)
	return fmt.Sprintf("ambient-content-%s", session)
}
