package types

import "strings"

// ProviderType distinguishes between Git hosting providers
type ProviderType string

const (
	// ProviderGitHub represents GitHub repositories
	ProviderGitHub ProviderType = "github"
	// ProviderGitLab represents GitLab repositories
	ProviderGitLab ProviderType = "gitlab"
)

// DetectProvider determines the Git provider from a repository URL
func DetectProvider(repoURL string) ProviderType {
	lowerURL := strings.ToLower(repoURL)

	if strings.Contains(lowerURL, "github.com") || strings.Contains(lowerURL, "github.") {
		return ProviderGitHub
	}
	if strings.Contains(lowerURL, "gitlab.com") || strings.Contains(lowerURL, "gitlab.") {
		return ProviderGitLab
	}

	// Default to empty string for unknown providers
	return ""
}

// String returns the string representation of the provider type
func (p ProviderType) String() string {
	return string(p)
}

// IsValid checks if the provider type is valid
func (p ProviderType) IsValid() bool {
	return p == ProviderGitHub || p == ProviderGitLab
}
