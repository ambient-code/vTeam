package main

import (
	"context"
	"fmt"

	"ambient-code-backend/handlers"
)

var githubTokenManager *GitHubTokenManager

// initializeGitHubTokenManager initializes the GitHub token manager after envs are loaded
func initializeGitHubTokenManager() {
	var err error
	githubTokenManager, err = NewGitHubTokenManager()
	if err != nil {
		// Log error but don't fail - GitHub App might not be configured
		fmt.Printf("Warning: GitHub App not configured: %v\n", err)
	}
}

// Removed legacy project-scoped link handler and GitHubAppInstallation type (moved to handlers/github_auth.go)

// getGitHubInstallation retrieves GitHub App installation for a user (wrapper to handlers package)
func getGitHubInstallation(ctx context.Context, userID string) (*handlers.GitHubAppInstallation, error) {
	return handlers.GetGitHubInstallation(ctx, userID)
}

// Removed helper functions (moved to handlers/repo.go):
// - githubAPIBaseURL
// - parseOwnerRepo
// - mintInstallationToken
// - doGitHubRequest

// Removed repo browsing handlers (moved to handlers/repo.go):
// - listUserForks
// - createUserFork
// - getRepoTree
// - getRepoBlob
