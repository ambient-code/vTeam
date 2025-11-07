// Package types defines GVR (GroupVersionResource) definitions and resource helpers for custom resources.
package types

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	// AmbientVertexSecretName is the name of the secret containing Vertex AI credentials
	AmbientVertexSecretName = "ambient-vertex"

	// CopiedFromAnnotation is the annotation key used to track secrets copied by the operator
	CopiedFromAnnotation = "vteam.ambient-code/copied-from"
)

// GetAgenticSessionResource returns the GroupVersionResource for AgenticSession
func GetAgenticSessionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "agenticsessions",
	}
}

// GetProjectSettingsResource returns the GroupVersionResource for ProjectSettings
func GetProjectSettingsResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "projectsettings",
	}
}
