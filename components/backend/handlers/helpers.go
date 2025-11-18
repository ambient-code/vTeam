package handlers

import (
	"fmt"
	"log"
	"math"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
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
