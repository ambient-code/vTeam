package types

// ProjectSettings represents the ProjectSettings CRD spec
type ProjectSettings struct {
	GroupAccess       []GroupAccess `json:"groupAccess,omitempty"`
	RunnerSecretsName string        `json:"runnerSecretsName,omitempty"`
	Repos             []ProjectRepo `json:"repos,omitempty"`
}

// GroupAccess represents RBAC group configuration
type GroupAccess struct {
	GroupName string `json:"groupName" binding:"required"`
	Role      string `json:"role" binding:"required"` // admin, edit, or view
}

// ProjectRepo represents a repository available to this project
type ProjectRepo struct {
	Name          string `json:"name" binding:"required"`
	URL           string `json:"url" binding:"required"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
}
