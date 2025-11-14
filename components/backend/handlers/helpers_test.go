package handlers

import (
	"errors"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetProjectSettingsResource(t *testing.T) {
	gvr := GetProjectSettingsResource()

	tests := []struct {
		name     string
		expected string
		actual   string
	}{
		{
			name:     "Group should be vteam.ambient-code",
			expected: "vteam.ambient-code",
			actual:   gvr.Group,
		},
		{
			name:     "Version should be v1alpha1",
			expected: "v1alpha1",
			actual:   gvr.Version,
		},
		{
			name:     "Resource should be projectsettings",
			expected: "projectsettings",
			actual:   gvr.Resource,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.actual)
			}
		})
	}
}

func TestRetryWithBackoff(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return nil
		}

		err := RetryWithBackoff(3, 10*time.Millisecond, 100*time.Millisecond, operation)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("success after retries", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary failure")
			}
			return nil
		}

		err := RetryWithBackoff(5, 10*time.Millisecond, 100*time.Millisecond, operation)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("failure after max retries", func(t *testing.T) {
		attempts := 0
		expectedError := errors.New("persistent failure")
		operation := func() error {
			attempts++
			return expectedError
		}

		err := RetryWithBackoff(3, 10*time.Millisecond, 100*time.Millisecond, operation)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("respects max delay", func(t *testing.T) {
		startTime := time.Now()
		attempts := 0
		operation := func() error {
			attempts++
			return errors.New("failure")
		}

		maxDelay := 50 * time.Millisecond
		RetryWithBackoff(3, 10*time.Millisecond, maxDelay, operation)
		duration := time.Since(startTime)

		// With 3 retries and max delay of 50ms, total time should be less than 150ms
		// (allowing some buffer for execution time)
		if duration > 200*time.Millisecond {
			t.Errorf("expected duration less than 200ms, got %v", duration)
		}
	})
}

func TestRetryWithBackoffZeroRetries(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		return errors.New("failure")
	}

	err := RetryWithBackoff(0, 10*time.Millisecond, 100*time.Millisecond, operation)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if attempts != 0 {
		t.Errorf("expected 0 attempts, got %d", attempts)
	}
}

func BenchmarkRetryWithBackoffSuccess(b *testing.B) {
	operation := func() error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RetryWithBackoff(3, 1*time.Millisecond, 10*time.Millisecond, operation)
	}
}

// TestGroupVersionResource verifies the GVR format is correct
func TestGroupVersionResource(t *testing.T) {
	gvr := GetProjectSettingsResource()

	// Verify it's a valid GVR that can be used with dynamic client
	if gvr.Empty() {
		t.Error("GVR should not be empty")
	}

	// Verify string representation contains expected parts
	gvrString := gvr.String()
	if !strings.Contains(gvrString, "vteam.ambient-code") {
		t.Errorf("GVR string should contain group: %s", gvrString)
	}
	if !strings.Contains(gvrString, "v1alpha1") {
		t.Errorf("GVR string should contain version: %s", gvrString)
	}
	if !strings.Contains(gvrString, "projectsettings") {
		t.Errorf("GVR string should contain resource: %s", gvrString)
	}
}

// Mock test for schema validation
func TestSchemaGroupVersionResource(t *testing.T) {
	gvr := GetProjectSettingsResource()

	// Verify the type
	var _ schema.GroupVersionResource = gvr

	// Verify the individual components instead of string format
	if gvr.Group != "vteam.ambient-code" {
		t.Errorf("Expected group 'vteam.ambient-code', got '%s'", gvr.Group)
	}
	if gvr.Version != "v1alpha1" {
		t.Errorf("Expected version 'v1alpha1', got '%s'", gvr.Version)
	}
	if gvr.Resource != "projectsettings" {
		t.Errorf("Expected resource 'projectsettings', got '%s'", gvr.Resource)
	}
}
