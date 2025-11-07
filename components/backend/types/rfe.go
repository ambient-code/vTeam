package types

// RFEWorkflow represents RFE (Request For Enhancement) workflow data structures.
type RFEWorkflow struct {
	ID              string             `json:"id"`
	Title           string             `json:"title"`
	Description     string             `json:"description"`
	BranchName      string             `json:"branchName"`
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
	BranchName      string          `json:"branchName" binding:"required"`
	UmbrellaRepo    GitRepository   `json:"umbrellaRepo"`
	SupportingRepos []GitRepository `json:"supportingRepos,omitempty"`
	WorkspacePath   string          `json:"workspacePath,omitempty"`
	ParentOutcome   *string         `json:"parentOutcome,omitempty"`
}

type UpdateRFEWorkflowRequest struct {
	Title           *string         `json:"title,omitempty"`
	Description     *string         `json:"description,omitempty"`
	UmbrellaRepo    *GitRepository  `json:"umbrellaRepo,omitempty"`
	SupportingRepos []GitRepository `json:"supportingRepos,omitempty"`
	ParentOutcome   *string         `json:"parentOutcome,omitempty"`
}

type AdvancePhaseRequest struct {
	Force bool `json:"force,omitempty"` // Force advance even if current phase isn't complete
}
