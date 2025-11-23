package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"ambient-code-backend/types"
	"github.com/google/uuid"
)

const (
	// DefaultMaxPaginationPages is the default limit for pagination loops
	DefaultMaxPaginationPages = 100
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

// PaginationInfo contains pagination metadata from GitLab API responses
type PaginationInfo struct {
	TotalPages  int
	NextPage    int
	PrevPage    int
	PerPage     int
	Total       int
	CurrentPage int
}

// extractPaginationInfo extracts pagination info from response headers
func extractPaginationInfo(resp *http.Response) *PaginationInfo {
	info := &PaginationInfo{}

	// GitLab uses X-Total-Pages, X-Next-Page, X-Per-Page headers
	if totalPages := resp.Header.Get("X-Total-Pages"); totalPages != "" {
		fmt.Sscanf(totalPages, "%d", &info.TotalPages)
	}
	if nextPage := resp.Header.Get("X-Next-Page"); nextPage != "" {
		fmt.Sscanf(nextPage, "%d", &info.NextPage)
	}
	if prevPage := resp.Header.Get("X-Prev-Page"); prevPage != "" {
		fmt.Sscanf(prevPage, "%d", &info.PrevPage)
	}
	if perPage := resp.Header.Get("X-Per-Page"); perPage != "" {
		fmt.Sscanf(perPage, "%d", &info.PerPage)
	}
	if total := resp.Header.Get("X-Total"); total != "" {
		fmt.Sscanf(total, "%d", &info.Total)
	}
	if page := resp.Header.Get("X-Page"); page != "" {
		fmt.Sscanf(page, "%d", &info.CurrentPage)
	}

	return info
}

// GetBranches retrieves all branches for a GitLab repository with pagination support
func (c *Client) GetBranches(ctx context.Context, projectID string, page, perPage int) ([]types.GitLabBranch, *PaginationInfo, error) {
	if perPage == 0 {
		perPage = 100 // Max page size for GitLab API
	}

	path := fmt.Sprintf("/projects/%s/repository/branches?page=%d&per_page=%d", projectID, page, perPage)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read branches response: %w", err)
	}

	var branches []types.GitLabBranch
	if err := json.Unmarshal(body, &branches); err != nil {
		return nil, nil, fmt.Errorf("failed to parse branches response: %w", err)
	}

	pagination := extractPaginationInfo(resp)

	return branches, pagination, nil
}

// getMaxPaginationPages returns the configured maximum pagination pages
// Can be overridden via GITLAB_MAX_PAGINATION_PAGES environment variable
func getMaxPaginationPages() int {
	if envVal := os.Getenv("GITLAB_MAX_PAGINATION_PAGES"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			return val
		}
		log.Printf("Warning: Invalid GITLAB_MAX_PAGINATION_PAGES value '%s', using default %d", envVal, DefaultMaxPaginationPages)
	}
	return DefaultMaxPaginationPages
}

// GetAllBranches retrieves all branches across all pages
// Pagination limit can be configured via GITLAB_MAX_PAGINATION_PAGES environment variable
func (c *Client) GetAllBranches(ctx context.Context, projectID string) ([]types.GitLabBranch, error) {
	var allBranches []types.GitLabBranch
	page := 1
	perPage := 100
	maxPages := getMaxPaginationPages()

	for {
		branches, pagination, err := c.GetBranches(ctx, projectID, page, perPage)
		if err != nil {
			return nil, err
		}

		allBranches = append(allBranches, branches...)

		// Check if there are more pages
		if pagination.NextPage == 0 || len(branches) == 0 {
			break
		}

		page = pagination.NextPage

		// Safety limit to prevent infinite loops (configurable)
		if page > maxPages {
			log.Printf("Warning: Repository %s has more than %d pages of branches, truncating results", projectID, maxPages)
			return allBranches, fmt.Errorf("exceeded pagination limit (%d pages). Increase GITLAB_MAX_PAGINATION_PAGES if needed", maxPages)
		}

		// Log warning when approaching limit
		if page > maxPages-10 {
			log.Printf("Warning: Pagination for repository %s is approaching limit (page %d/%d)", projectID, page, maxPages)
		}
	}

	return allBranches, nil
}

// GetTree retrieves the directory tree for a GitLab repository
func (c *Client) GetTree(ctx context.Context, projectID, ref, path string, page, perPage int) ([]types.GitLabTreeEntry, *PaginationInfo, error) {
	if perPage == 0 {
		perPage = 100
	}

	// Build the API path
	apiPath := fmt.Sprintf("/projects/%s/repository/tree?ref=%s&page=%d&per_page=%d",
		projectID, ref, page, perPage)

	if path != "" && path != "/" {
		apiPath += "&path=" + path
	}

	resp, err := c.doRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read tree response: %w", err)
	}

	var entries []types.GitLabTreeEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, nil, fmt.Errorf("failed to parse tree response: %w", err)
	}

	pagination := extractPaginationInfo(resp)

	return entries, pagination, nil
}

// GetAllTreeEntries retrieves all tree entries across all pages
func (c *Client) GetAllTreeEntries(ctx context.Context, projectID, ref, path string) ([]types.GitLabTreeEntry, error) {
	var allEntries []types.GitLabTreeEntry
	page := 1
	perPage := 100

	for {
		entries, pagination, err := c.GetTree(ctx, projectID, ref, path, page, perPage)
		if err != nil {
			return nil, err
		}

		allEntries = append(allEntries, entries...)

		if pagination.NextPage == 0 || len(entries) == 0 {
			break
		}

		page = pagination.NextPage

		// Safety limit
		if page > 100 {
			return nil, fmt.Errorf("exceeded pagination limit (100 pages)")
		}
	}

	return allEntries, nil
}

// GitLabFileContent represents the response from GitLab file content API
type GitLabFileContent struct {
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	Size         int    `json:"size"`
	Encoding     string `json:"encoding"`
	Content      string `json:"content"`
	ContentSHA   string `json:"content_sha256"`
	Ref          string `json:"ref"`
	BlobID       string `json:"blob_id"`
	CommitID     string `json:"commit_id"`
	LastCommitID string `json:"last_commit_id"`
}

// GetFileContents retrieves the contents of a file from a GitLab repository
func (c *Client) GetFileContents(ctx context.Context, projectID, filePath, ref string) (*GitLabFileContent, error) {
	// URL encode the file path using url.PathEscape for safe encoding
	encodedPath := url.PathEscape(filePath)

	path := fmt.Sprintf("/projects/%s/repository/files/%s?ref=%s", projectID, encodedPath, ref)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content response: %w", err)
	}

	var fileContent GitLabFileContent
	if err := json.Unmarshal(body, &fileContent); err != nil {
		return nil, fmt.Errorf("failed to parse file content response: %w", err)
	}

	return &fileContent, nil
}

// GetRawFileContents retrieves the raw contents of a file (without base64 encoding)
func (c *Client) GetRawFileContents(ctx context.Context, projectID, filePath, ref string) ([]byte, error) {
	// URL encode the file path using url.PathEscape for safe encoding
	encodedPath := url.PathEscape(filePath)

	path := fmt.Sprintf("/projects/%s/repository/files/%s/raw?ref=%s", projectID, encodedPath, ref)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read raw file content: %w", err)
	}

	return body, nil
}
