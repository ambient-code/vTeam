package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ambient-code-backend/types"
	"github.com/google/uuid"
)

// Client represents a GitLab API client
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient creates a new GitLab API client with 15-second timeout
func NewClient(baseURL, token string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL: baseURL,
		token:   token,
	}
}

// doRequest performs an HTTP request with GitLab authentication
// Includes standardized logging and request ID tracking for debugging
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	// Generate unique request ID for tracking
	requestID := uuid.New().String()

	// Log request start (with redacted URL)
	startTime := time.Now()
	LogInfo("[ReqID: %s] GitLab API request: %s %s", requestID, method, RedactURL(url))

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		LogError("[ReqID: %s] Failed to create request: %v", requestID, err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitLab authentication header
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", requestID) // Include request ID in headers for GitLab correlation

	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		LogError("[ReqID: %s] GitLab API request failed after %v: %v", requestID, duration, err)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Log response with status and timing
	LogInfo("[ReqID: %s] GitLab API response: %d %s (took %v)",
		requestID, resp.StatusCode, http.StatusText(resp.StatusCode), duration)

	// Log warning for non-2xx responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		LogWarning("[ReqID: %s] GitLab API returned non-success status: %d", requestID, resp.StatusCode)
	}

	return resp, nil
}

// ParseErrorResponse parses a GitLab API error response and returns a structured error
func ParseErrorResponse(resp *http.Response) *types.GitLabAPIError {
	defer resp.Body.Close()

	// Extract request ID from response headers if present
	requestID := resp.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = resp.Request.Header.Get("X-Request-ID") // Fallback to request header
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		LogError("[ReqID: %s] Failed to read GitLab error response: %v", requestID, err)
		return &types.GitLabAPIError{
			StatusCode:  resp.StatusCode,
			Message:     "Failed to read error response from GitLab API",
			Remediation: "Please try again or contact support if the issue persists",
			RawError:    err.Error(),
			RequestID:   requestID,
		}
	}

	// Try to parse GitLab error format
	var gitlabError struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(body, &gitlabError); err == nil {
		apiErr := MapGitLabAPIError(resp.StatusCode, gitlabError.Message, gitlabError.Error, string(body))
		apiErr.RequestID = requestID
		LogError("[ReqID: %s] GitLab API error: %s (status: %d)", requestID, apiErr.Message, resp.StatusCode)
		return apiErr
	}

	// Fallback to generic error with raw body
	apiErr := MapGitLabAPIError(resp.StatusCode, "", "", string(body))
	apiErr.RequestID = requestID
	LogError("[ReqID: %s] GitLab API error (status: %d): %s", requestID, resp.StatusCode, string(body))
	return apiErr
}

// MapGitLabAPIError maps HTTP status codes to user-friendly error messages
func MapGitLabAPIError(statusCode int, message, errorType, rawBody string) *types.GitLabAPIError {
	apiError := &types.GitLabAPIError{
		StatusCode: statusCode,
		RawError:   rawBody,
	}

	switch statusCode {
	case 401:
		apiError.Message = "GitLab token is invalid or expired"
		apiError.Remediation = "Please reconnect your GitLab account with a valid Personal Access Token"

	case 403:
		apiError.Message = "GitLab token lacks required permissions"
		if message != "" {
			apiError.Message = fmt.Sprintf("GitLab error: %s", message)
		}
		apiError.Remediation = "Ensure your token has 'api', 'read_repository', and 'write_repository' scopes and try again"

	case 404:
		apiError.Message = "GitLab repository not found"
		apiError.Remediation = "Verify the repository URL and your access permissions"

	case 429:
		apiError.Message = "GitLab API rate limit exceeded"
		apiError.Remediation = "Please wait a few minutes before retrying. GitLab.com allows 300 requests per minute"

	case 500, 502, 503, 504:
		apiError.Message = "GitLab API is experiencing issues"
		apiError.Remediation = "Please try again in a few minutes or contact support if the issue persists"

	default:
		if message != "" {
			apiError.Message = fmt.Sprintf("GitLab API error: %s", message)
		} else {
			apiError.Message = fmt.Sprintf("GitLab API returned status code %d", statusCode)
		}
		apiError.Remediation = "Please check your request and try again"
	}

	return apiError
}

// CheckResponse checks an HTTP response for errors and returns a GitLabAPIError if found
func CheckResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return ParseErrorResponse(resp)
}
