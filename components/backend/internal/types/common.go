package types

// Git related types
type GitUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GitAuthentication struct {
	SSHKeySecret *string `json:"sshKeySecret,omitempty"`
	TokenSecret  *string `json:"tokenSecret,omitempty"`
}

type GitRepository struct {
	URL       string  `json:"url"`
	Branch    *string `json:"branch,omitempty"`
	ClonePath *string `json:"clonePath,omitempty"`
}

type GitConfig struct {
	User           *GitUser           `json:"user,omitempty"`
	Authentication *GitAuthentication `json:"authentication,omitempty"`
	Repositories   []GitRepository    `json:"repositories,omitempty"`
}

// Multi-tenant support types
type UserContext struct {
	UserID      string   `json:"userId" binding:"required"`
	DisplayName string   `json:"displayName" binding:"required"`
	Groups      []string `json:"groups" binding:"required"`
}

type BotAccountRef struct {
	Name string `json:"name" binding:"required"`
}

type ResourceOverrides struct {
	CPU           string `json:"cpu,omitempty"`
	Memory        string `json:"memory,omitempty"`
	StorageClass  string `json:"storageClass,omitempty"`
	PriorityClass string `json:"priorityClass,omitempty"`
}

// Utility functions
func BoolPtr(b bool) *bool {
	return &b
}

func StringPtr(s string) *string {
	return &s
}