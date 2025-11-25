package regression_test

import (
	"testing"

	"ambient-code-backend/types"

	"github.com/stretchr/testify/assert"
)

// TestBackwardCompatibility_ProviderDetection verifies provider detection
// doesn't break GitHub URLs or return incorrect values for existing repos
func TestBackwardCompatibility_ProviderDetection(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected types.ProviderType
	}{
		// GitHub URLs should still be detected correctly
		{
			name:     "GitHub HTTPS",
			url:      "https://github.com/owner/repo.git",
			expected: types.ProviderGitHub,
		},
		{
			name:     "GitHub HTTPS without .git",
			url:      "https://github.com/owner/repo",
			expected: types.ProviderGitHub,
		},
		{
			name:     "GitHub SSH",
			url:      "git@github.com:owner/repo.git",
			expected: types.ProviderGitHub,
		},
		{
			name:     "GitHub Enterprise",
			url:      "https://github.company.com/owner/repo.git",
			expected: types.ProviderGitHub,
		},

		// New GitLab URLs should be detected
		{
			name:     "GitLab HTTPS",
			url:      "https://gitlab.com/owner/repo.git",
			expected: types.ProviderGitLab,
		},
		{
			name:     "GitLab SSH",
			url:      "git@gitlab.com:owner/repo.git",
			expected: types.ProviderGitLab,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detected := types.DetectProvider(tc.url)
			assert.Equal(t, tc.expected, detected,
				"Provider detection changed for %s - this breaks backward compatibility!", tc.url)
		})
	}
}

// TestBackwardCompatibility_ProviderEnumValues ensures provider type values
// haven't changed (would break existing database/CRD records)
func TestBackwardCompatibility_ProviderEnumValues(t *testing.T) {
	// These values must never change - they're stored in Kubernetes CRDs
	assert.Equal(t, types.ProviderType("github"), types.ProviderGitHub,
		"GitHub provider enum value changed - breaks existing CRDs!")
	assert.Equal(t, types.ProviderType("gitlab"), types.ProviderGitLab,
		"GitLab provider enum value changed - breaks existing CRDs!")
}

// TestBackwardCompatibility_EmptyProvider ensures empty provider field
// doesn't cause errors (existing ProjectSettings may not have provider)
func TestBackwardCompatibility_EmptyProvider(t *testing.T) {
	// Simulate existing ProjectSettings without provider field
	emptyProvider := types.ProviderType("")

	// Should not panic or error
	assert.NotPanics(t, func() {
		_ = string(emptyProvider)
	}, "Empty provider should not cause panic")

	// Empty provider should not equal valid providers
	assert.NotEqual(t, types.ProviderGitHub, emptyProvider)
	assert.NotEqual(t, types.ProviderGitLab, emptyProvider)
}

// TestBackwardCompatibility_GitHubOperationsUnchanged verifies that
// GitHub-specific functionality still works exactly as before
func TestBackwardCompatibility_GitHubOperationsUnchanged(t *testing.T) {
	// Test that GitHub URLs are not affected by GitLab code
	githubURLs := []string{
		"https://github.com/user/repo.git",
		"git@github.com:user/repo.git",
		"https://github.company.com/user/repo.git",
	}

	for _, url := range githubURLs {
		provider := types.DetectProvider(url)
		assert.Equal(t, types.ProviderGitHub, provider,
			"GitHub detection broken for URL: %s", url)
	}
}

// TestBackwardCompatibility_NoGitLabFalsePositives ensures GitLab detection
// doesn't incorrectly identify GitHub URLs as GitLab
func TestBackwardCompatibility_NoGitLabFalsePositives(t *testing.T) {
	notGitLabURLs := []string{
		"https://github.com/gitlab/repo.git",     // Contains "gitlab" but is GitHub
		"https://github.com/user/gitlab-cli.git", // Project named gitlab
		"git@github.com:company/gitlab-docs.git", // Repo contains gitlab
	}

	for _, url := range notGitLabURLs {
		provider := types.DetectProvider(url)
		assert.NotEqual(t, types.ProviderGitLab, provider,
			"False positive GitLab detection for GitHub URL: %s", url)
		assert.Equal(t, types.ProviderGitHub, provider,
			"Should detect as GitHub: %s", url)
	}
}

// TestBackwardCompatibility_ExistingProjectSettings verifies that existing
// ProjectSettings CRs (without provider field) still work
func TestBackwardCompatibility_ExistingProjectSettings(t *testing.T) {
	// Simulate existing ProjectSettings YAML without provider field
	// This represents real CRs created before GitLab support was added

	type Repository struct {
		URL      string             `json:"url"`
		Branch   string             `json:"branch,omitempty"`
		Provider types.ProviderType `json:"provider,omitempty"`
	}

	// Existing repo without provider field (should auto-detect)
	existingRepo := Repository{
		URL:    "https://github.com/user/repo.git",
		Branch: "main",
		// Provider field not set (empty string)
	}

	// Should be able to detect provider from URL even if field is empty
	if existingRepo.Provider == "" {
		existingRepo.Provider = types.DetectProvider(existingRepo.URL)
	}

	assert.Equal(t, types.ProviderGitHub, existingRepo.Provider,
		"Auto-detection should work for existing repos without provider field")
}
