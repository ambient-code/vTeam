package types

import (
	"net/url"
	"strings"
)

// ProviderType distinguishes between Git hosting providers
type ProviderType string

const (
	// ProviderGitHub represents GitHub repositories
	ProviderGitHub ProviderType = "github"
	// ProviderGitLab represents GitLab repositories
	ProviderGitLab ProviderType = "gitlab"
)

// DetectProvider determines the Git provider from a repository URL
// Uses precise hostname matching to prevent false positives
func DetectProvider(repoURL string) ProviderType {
	if repoURL == "" {
		return ""
	}

	// Normalize SSH URLs (git@host:path) to https://host/path for parsing
	normalizedURL := repoURL
	if strings.HasPrefix(repoURL, "git@") {
		// Convert git@github.com:owner/repo.git to https://github.com/owner/repo.git
		normalizedURL = strings.Replace(repoURL, ":", "/", 1)
		normalizedURL = strings.Replace(normalizedURL, "git@", "https://", 1)
	}

	// Parse the URL to extract hostname
	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		// Fallback to basic string matching if URL parsing fails
		lowerURL := strings.ToLower(repoURL)
		if strings.Contains(lowerURL, "github.com") {
			return ProviderGitHub
		}
		if strings.Contains(lowerURL, "gitlab.com") {
			return ProviderGitLab
		}
		return ""
	}

	hostname := strings.ToLower(parsedURL.Hostname())
	if hostname == "" {
		return ""
	}

	// Check for GitHub (github.com or *.github.com for enterprise)
	if hostname == "github.com" || strings.HasSuffix(hostname, ".github.com") {
		return ProviderGitHub
	}

	// Check for GitLab (gitlab.com or any hostname containing "gitlab" for self-hosted)
	// GitLab self-hosted instances typically use gitlab.company.com or gitlab-ce.company.com
	if hostname == "gitlab.com" || strings.Contains(hostname, "gitlab") {
		return ProviderGitLab
	}

	// Unknown provider
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
