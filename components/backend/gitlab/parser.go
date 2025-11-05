package gitlab

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"ambient-code-backend/types"
)

// ParseGitLabURL parses a GitLab repository URL and returns structured information
func ParseGitLabURL(repoURL string) (*types.ParsedGitLabRepo, error) {
	if repoURL == "" {
		return nil, fmt.Errorf("repository URL cannot be empty")
	}

	// Normalize the URL first
	normalized, err := NormalizeGitLabURL(repoURL)
	if err != nil {
		return nil, err
	}

	// Parse the normalized URL
	parsed, err := url.Parse(normalized)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %w", err)
	}

	// Extract host
	host := parsed.Host
	if host == "" {
		return nil, fmt.Errorf("unable to extract host from URL: %s", repoURL)
	}

	// Extract owner and repo from path
	// Path format: /owner/repo or /owner/repo.git
	path := strings.TrimPrefix(parsed.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitLab URL format, expected /owner/repo: %s", repoURL)
	}

	owner := parts[0]
	repo := parts[1]

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repository name are required")
	}

	// Detect if self-hosted or GitLab.com
	apiURL := ConstructAPIURL(host)

	// Create project ID (URL-encoded path for GitLab API)
	projectID := url.PathEscape(fmt.Sprintf("%s/%s", owner, repo))

	return &types.ParsedGitLabRepo{
		Host:      host,
		Owner:     owner,
		Repo:      repo,
		APIURL:    apiURL,
		ProjectID: projectID,
	}, nil
}

// NormalizeGitLabURL converts various GitLab URL formats to a canonical HTTPS format
func NormalizeGitLabURL(repoURL string) (string, error) {
	// Trim whitespace
	repoURL = strings.TrimSpace(repoURL)

	// Handle SSH format: git@gitlab.com:owner/repo.git
	sshPattern := regexp.MustCompile(`^git@([^:]+):(.+)$`)
	if matches := sshPattern.FindStringSubmatch(repoURL); matches != nil {
		host := matches[1]
		path := matches[2]
		path = strings.TrimSuffix(path, ".git")
		return fmt.Sprintf("https://%s/%s", host, path), nil
	}

	// Handle HTTPS URLs
	if strings.HasPrefix(repoURL, "https://") || strings.HasPrefix(repoURL, "http://") {
		// Upgrade HTTP to HTTPS for security
		if strings.HasPrefix(repoURL, "http://") {
			repoURL = strings.Replace(repoURL, "http://", "https://", 1)
		}

		// Remove .git suffix if present
		repoURL = strings.TrimSuffix(repoURL, ".git")

		return repoURL, nil
	}

	// If no protocol, assume https://
	if !strings.Contains(repoURL, "://") {
		return fmt.Sprintf("https://%s", repoURL), nil
	}

	return "", fmt.Errorf("unsupported URL format: %s", repoURL)
}

// IsGitLabSelfHosted determines if a host is a self-hosted GitLab instance
func IsGitLabSelfHosted(host string) bool {
	// GitLab.com is not self-hosted
	if host == "gitlab.com" || strings.HasSuffix(host, ".gitlab.com") {
		return false
	}

	// Everything else containing "gitlab" is assumed to be self-hosted
	// This includes domains like gitlab.company.com, gitlab.internal.example.com, etc.
	return strings.Contains(strings.ToLower(host), "gitlab")
}

// ConstructAPIURL builds the GitLab API base URL from a host
func ConstructAPIURL(host string) string {
	// For all GitLab instances (both .com and self-hosted), API is at /api/v4
	// Handle ports if present
	return fmt.Sprintf("https://%s/api/v4", host)
}

// ValidateGitLabURL checks if a URL is a valid GitLab repository URL
func ValidateGitLabURL(repoURL string) error {
	parsed, err := ParseGitLabURL(repoURL)
	if err != nil {
		return err
	}

	// Basic validation
	if parsed.Owner == "" || parsed.Repo == "" {
		return fmt.Errorf("invalid repository URL: missing owner or repository name")
	}

	// Ensure the host contains "gitlab"
	if !strings.Contains(strings.ToLower(parsed.Host), "gitlab") {
		return fmt.Errorf("URL does not appear to be a GitLab repository: %s", repoURL)
	}

	return nil
}
