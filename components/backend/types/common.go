// Package types defines common type definitions for AgenticSession, ProjectSettings, and RFE workflows.
package types

// Common types used across the application

type GitRepository struct {
	URL      string       `json:"url"`
	Branch   *string      `json:"branch,omitempty"`
	Provider ProviderType `json:"provider,omitempty"` // Optional: auto-detected if not specified
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

// Common repository browsing types (used by both GitHub and GitLab)

// Branch represents a Git branch (common format for UI)
type Branch struct {
	Name      string     `json:"name"`
	Protected bool       `json:"protected"`
	Default   bool       `json:"default,omitempty"`
	Commit    CommitInfo `json:"commit,omitempty"`
}

// CommitInfo represents basic commit information
type CommitInfo struct {
	SHA       string `json:"sha"`
	Message   string `json:"message,omitempty"`
	Author    string `json:"author,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// TreeEntry represents a file or directory in a repository
type TreeEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "blob" (file) or "tree" (directory)
	Mode string `json:"mode,omitempty"`
	SHA  string `json:"sha,omitempty"`
	Size int    `json:"size,omitempty"`
}

// FileContent represents file contents from a repository
type FileContent struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"` // "base64" or "utf-8"
	Size     int    `json:"size"`
	SHA      string `json:"sha,omitempty"`
}

// Helper functions for pointer types
func BoolPtr(b bool) *bool {
	return &b
}

func StringPtr(s string) *string {
	return &s
}

func IntPtr(i int) *int {
	return &i
}
