// Package types defines common type definitions for AgenticSession, ProjectSettings, and RFE workflows.
package types

// Common types used across the application

type GitRepository struct {
	URL    string  `json:"url"`
	Branch *string `json:"branch,omitempty"`
}

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

type LLMSettings struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

type GitConfig struct {
	Repositories []GitRepository `json:"repositories,omitempty"`
}

type Paths struct {
	Workspace string `json:"workspace,omitempty"`
	Messages  string `json:"messages,omitempty"`
	Inbox     string `json:"inbox,omitempty"`
}

// BoolPtr returns a pointer to the given bool value.
func BoolPtr(b bool) *bool {
	return &b
}

func StringPtr(s string) *string {
	return &s
}

func IntPtr(i int) *int {
	return &i
}
