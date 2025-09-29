package types

import "k8s.io/apimachinery/pkg/runtime/schema"

// RFE Workflow Data Structures
type RFEWorkflow struct {
	ID            string             `json:"id"`
	Title         string             `json:"title"`
	Description   string             `json:"description"`
	Repositories  []GitRepository    `json:"repositories,omitempty"`
	Project       string             `json:"project,omitempty"`
	WorkspacePath string             `json:"workspacePath"`
	CreatedAt     string             `json:"createdAt"`
	UpdatedAt     string             `json:"updatedAt"`
	JiraLinks     []WorkflowJiraLink `json:"jiraLinks,omitempty"`
}

type WorkflowJiraLink struct {
	Path    string `json:"path"`
	JiraKey string `json:"jiraKey"`
}

type CreateRFEWorkflowRequest struct {
	Title         string          `json:"title" binding:"required"`
	Description   string          `json:"description" binding:"required"`
	Repositories  []GitRepository `json:"repositories,omitempty"`
	WorkspacePath string          `json:"workspacePath,omitempty"`
}

type AdvancePhaseRequest struct {
	Force bool `json:"force,omitempty"` // Force advance even if current phase isn't complete
}

// getRFEWorkflowResource returns the GroupVersionResource for RFEWorkflow CRD
func GetRFEWorkflowResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "rfeworkflows",
	}
}