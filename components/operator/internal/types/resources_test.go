package types

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetAgenticSessionResource(t *testing.T) {
	gvr := GetAgenticSessionResource()

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
			name:     "Resource should be agenticsessions",
			expected: "agenticsessions",
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

func TestGVRStrings(t *testing.T) {
	tests := []struct {
		name string
		gvr  schema.GroupVersionResource
	}{
		{
			name: "AgenticSession GVR String",
			gvr:  GetAgenticSessionResource(),
		},
		{
			name: "ProjectSettings GVR String",
			gvr:  GetProjectSettingsResource(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gvrString := tt.gvr.String()
			// Verify string contains the expected components
			if gvrString == "" {
				t.Error("GVR string should not be empty")
			}
			// The GVR String() format varies, just ensure it's not empty
			t.Logf("GVR string format: %s", gvrString)
		})
	}
}

func TestGVRNotEmpty(t *testing.T) {
	tests := []struct {
		name string
		gvr  schema.GroupVersionResource
	}{
		{
			name: "AgenticSession GVR not empty",
			gvr:  GetAgenticSessionResource(),
		},
		{
			name: "ProjectSettings GVR not empty",
			gvr:  GetProjectSettingsResource(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.gvr.Empty() {
				t.Error("GVR should not be empty")
			}
		})
	}
}

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{
			name:     "AmbientVertexSecretName",
			actual:   AmbientVertexSecretName,
			expected: "ambient-vertex",
		},
		{
			name:     "CopiedFromAnnotation",
			actual:   CopiedFromAnnotation,
			expected: "vteam.ambient-code/copied-from",
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

// TestGVRConsistency verifies that both GVRs use the same API group and version
func TestGVRConsistency(t *testing.T) {
	sessionGVR := GetAgenticSessionResource()
	settingsGVR := GetProjectSettingsResource()

	if sessionGVR.Group != settingsGVR.Group {
		t.Errorf("GVRs should use the same group: session=%s, settings=%s", sessionGVR.Group, settingsGVR.Group)
	}

	if sessionGVR.Version != settingsGVR.Version {
		t.Errorf("GVRs should use the same version: session=%s, settings=%s", sessionGVR.Version, settingsGVR.Version)
	}

	// Resources should be different
	if sessionGVR.Resource == settingsGVR.Resource {
		t.Error("GVRs should have different resource names")
	}
}

// BenchmarkGetAgenticSessionResource measures performance of GVR creation
func BenchmarkGetAgenticSessionResource(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetAgenticSessionResource()
	}
}

// BenchmarkGetProjectSettingsResource measures performance of GVR creation
func BenchmarkGetProjectSettingsResource(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetProjectSettingsResource()
	}
}
