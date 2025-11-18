// Package github implements GitHub App authentication and API integration.
package github

import (
	"context"
	"fmt"
	"time"

	"ambient-code-backend/handlers"
)

// Package-level variable for token manager
var (
	Manager *TokenManager
)

// InitializeTokenManager initializes the GitHub token manager after envs are loaded
func InitializeTokenManager() {
	var err error
	Manager, err = NewTokenManager()
	if err != nil {
		// Log error but don't fail - GitHub App might not be configured
		fmt.Printf("Warning: GitHub App not configured: %v\n", err)
	}
}

// GetInstallation retrieves GitHub App installation for a user (wrapper to handlers package)
func GetInstallation(ctx context.Context, userID string) (*handlers.GitHubAppInstallation, error) {
	return handlers.GetGitHubInstallation(ctx, userID)
}

// MintSessionToken creates a GitHub access token for a session
// Returns the token and expiry time to be injected as a Kubernetes Secret
func MintSessionToken(ctx context.Context, userID string) (string, time.Time, error) {
	if Manager == nil {
		return "", time.Time{}, fmt.Errorf("GitHub App not configured")
	}

	// Get user's GitHub installation
	installation, err := GetInstallation(ctx, userID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get GitHub installation: %w", err)
	}

	// Mint short-lived token for the installation's host
	token, expiresAt, err := Manager.MintInstallationTokenForHost(ctx, installation.InstallationID, installation.Host)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to mint installation token: %w", err)
	}

	return token, expiresAt, nil
}
