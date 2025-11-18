// Package k8s provides Kubernetes client creation and configuration utilities.
package k8s

import "k8s.io/apimachinery/pkg/runtime/schema"

// GetAgenticSessionV1Alpha1Resource returns the GroupVersionResource for AgenticSession v1alpha1
func GetAgenticSessionV1Alpha1Resource() schema.GroupVersionResource {
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

// GetOpenShiftProjectResource returns the GroupVersionResource for OpenShift Project
func GetOpenShiftProjectResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "project.openshift.io",
		Version:  "v1",
		Resource: "projects",
	}
}

// GetOpenShiftProjectRequestResource returns the GroupVersionResource for OpenShift ProjectRequest
func GetOpenShiftProjectRequestResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "project.openshift.io",
		Version:  "v1",
		Resource: "projectrequests",
	}
}
