package types

// RFE Workflow Data Structures
type RFEWorkflow struct {
	ID              string             `json:"id"`
	Title           string             `json:"title"`
	Description     string             `json:"description"`
	UmbrellaRepo    *GitRepository     `json:"umbrellaRepo,omitempty"`
	SupportingRepos []GitRepository    `json:"supportingRepos,omitempty"`
	Project         string             `json:"project,omitempty"`
	WorkspacePath   string             `json:"workspacePath"`
	CreatedAt       string             `json:"createdAt"`
	UpdatedAt       string             `json:"updatedAt"`
	JiraLinks       []WorkflowJiraLink `json:"jiraLinks,omitempty"`
	ParentOutcome   *string            `json:"parentOutcome,omitempty"`
}

type WorkflowJiraLink struct {
	Path    string `json:"path"`
	JiraKey string `json:"jiraKey"`
}

type CreateRFEWorkflowRequest struct {
	Title           string          `json:"title" binding:"required"`
	Description     string          `json:"description" binding:"required"`
	UmbrellaRepo    GitRepository   `json:"umbrellaRepo"`
	SupportingRepos []GitRepository `json:"supportingRepos,omitempty"`
	WorkspacePath   string          `json:"workspacePath,omitempty"`
	ParentOutcome   *string         `json:"parentOutcome,omitempty"`
}

type AdvancePhaseRequest struct {
	Force bool `json:"force,omitempty"` // Force advance even if current phase isn't complete
}
