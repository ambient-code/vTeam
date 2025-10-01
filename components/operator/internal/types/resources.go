package types

import "k8s.io/apimachinery/pkg/runtime/schema"

// GetAgenticSessionResource returns the GroupVersionResource for AgenticSession v1alpha1
func GetAgenticSessionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "agenticsessions",
	}
}

// GetProjectSettingsResource returns the GroupVersionResource for ProjectSettings v1alpha1
func GetProjectSettingsResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "projectsettings",
	}
}