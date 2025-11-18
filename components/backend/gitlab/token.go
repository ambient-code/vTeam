package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"

	"ambient-code-backend/types"
)

// GitLabUser represents a GitLab user from the /user API
type GitLabUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

// TokenValidationResult contains the result of token validation
type TokenValidationResult struct {
	Valid        bool
	User         *GitLabUser
	InstanceURL  string
	ErrorMessage string
	ErrorCode    int
}

// ValidateGitLabToken validates a GitLab Personal Access Token
func ValidateGitLabToken(ctx context.Context, token, instanceURL string) (*TokenValidationResult, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	if instanceURL == "" {
		instanceURL = "https://gitlab.com"
	}

	// Construct API URL
	apiURL := ConstructAPIURL(ExtractHost(instanceURL))
	client := NewClient(apiURL, token)

	// Call /user API to validate token
	user, err := GetCurrentUser(ctx, client)
	if err != nil {
		// Check if it's a GitLabAPIError
		if gitlabErr, ok := err.(*types.GitLabAPIError); ok {
			return &TokenValidationResult{
				Valid:        false,
				ErrorMessage: gitlabErr.Message,
				ErrorCode:    gitlabErr.StatusCode,
			}, nil
		}

		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	return &TokenValidationResult{
		Valid:       true,
		User:        user,
		InstanceURL: instanceURL,
	}, nil
}

// GetCurrentUser retrieves the current authenticated user from GitLab API
func GetCurrentUser(ctx context.Context, client *Client) (*GitLabUser, error) {
	resp, err := client.doRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var user GitLabUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w", err)
	}

	return &user, nil
}

// ValidateRepositoryAccess checks if the token has access to a specific repository
func ValidateRepositoryAccess(ctx context.Context, client *Client, owner, repo string) error {
	// Construct project path
	projectPath := fmt.Sprintf("%s/%s", owner, repo)
	projectID := EncodeProjectPath(projectPath)

	// Try to get project information
	resp, err := client.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s", projectID), nil)
	if err != nil {
		return fmt.Errorf("failed to access repository: %w", err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		// Customize error message for repository access
		if gitlabErr, ok := err.(*types.GitLabAPIError); ok {
			if gitlabErr.StatusCode == 404 {
				gitlabErr.Message = fmt.Sprintf("Repository '%s/%s' not found or you don't have access", owner, repo)
				gitlabErr.Remediation = "Verify the repository URL and ensure your token has access to this repository"
			}
		}
		return err
	}

	return nil
}

// ValidateTokenAndRepository performs comprehensive validation of token and repository access
func ValidateTokenAndRepository(ctx context.Context, token, repoURL string) (*TokenValidationResult, error) {
	// Parse repository URL
	parsed, err := ParseGitLabURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	// Construct instance URL from host
	instanceURL := fmt.Sprintf("https://%s", parsed.Host)

	// Validate token
	result, err := ValidateGitLabToken(ctx, token, instanceURL)
	if err != nil {
		return nil, err
	}

	if !result.Valid {
		return result, nil
	}

	// Create client for repository access check
	client := NewClient(parsed.APIURL, token)

	// Validate repository access
	if err := ValidateRepositoryAccess(ctx, client, parsed.Owner, parsed.Repo); err != nil {
		if gitlabErr, ok := err.(*types.GitLabAPIError); ok {
			return &TokenValidationResult{
				Valid:        false,
				ErrorMessage: gitlabErr.Message,
				ErrorCode:    gitlabErr.StatusCode,
			}, nil
		}
		return nil, err
	}

	return result, nil
}

// ExtractHost extracts the host from a full URL
func ExtractHost(urlStr string) string {
	// Remove protocol
	host := urlStr
	if len(host) > 8 && host[:8] == "https://" {
		host = host[8:]
	} else if len(host) > 7 && host[:7] == "http://" {
		host = host[7:]
	}

	// Remove path
	if idx := len(host); idx > 0 {
		for i, ch := range host {
			if ch == '/' {
				idx = i
				break
			}
		}
		host = host[:idx]
	}

	return host
}

// EncodeProjectPath URL-encodes a GitLab project path for API calls
func EncodeProjectPath(projectPath string) string {
	// GitLab API accepts URL-encoded project paths
	// e.g., "namespace/project" becomes "namespace%2Fproject"
	// Use url.PathEscape for safe, standards-compliant encoding
	return url.PathEscape(projectPath)
}

// TokenInfo contains metadata about a GitLab token
type TokenInfo struct {
	UserID      int
	Username    string
	InstanceURL string
	ValidatedAt time.Time
}

// GetTokenInfo retrieves information about a validated token
func GetTokenInfo(ctx context.Context, token, instanceURL string) (*TokenInfo, error) {
	result, err := ValidateGitLabToken(ctx, token, instanceURL)
	if err != nil {
		return nil, err
	}

	if !result.Valid {
		return nil, fmt.Errorf("token is invalid: %s", result.ErrorMessage)
	}

	return &TokenInfo{
		UserID:      result.User.ID,
		Username:    result.User.Username,
		InstanceURL: instanceURL,
		ValidatedAt: time.Now(),
	}, nil
}
