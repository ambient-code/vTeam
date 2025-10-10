package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ambient-code-backend/handlers"

	"github.com/gin-gonic/gin"
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

// helper: resolve GitHub API base URL from host
func githubAPIBaseURL(host string) string {
	if host == "" || host == "github.com" {
		return "https://api.github.com"
	}
	// GitHub Enterprise default
	return fmt.Sprintf("https://%s/api/v3", host)
}

// helper: parse repo into owner/repo from either owner/repo or full URL/SSH
func parseOwnerRepo(full string) (string, string, error) {
	s := strings.TrimSpace(full)
	s = strings.TrimSuffix(s, ".git")
	// Handle URLs or SSH forms like git@github.com:owner/repo.git
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "ssh://") || strings.Contains(s, "@") {
		// Normalize SSH to https-like then split
		s = strings.NewReplacer(":", "/", "git@", "https://").Replace(s)
		parts := strings.Split(s, "/")
		if len(parts) >= 2 {
			owner := parts[len(parts)-2]
			repo := parts[len(parts)-1]
			if owner != "" && repo != "" {
				return owner, repo, nil
			}
		}
		return "", "", fmt.Errorf("invalid repo format, expected owner/repo")
	}
	// owner/repo
	parts := strings.Split(s, "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("invalid repo format, expected owner/repo")
}

// helper: mint installation token or error
func mintInstallationToken(ctx context.Context, installationID int64, host string) (string, error) {
	if githubTokenManager == nil {
		return "", fmt.Errorf("GitHub App not configured")
	}
	token, _, err := githubTokenManager.MintInstallationTokenForHost(ctx, installationID, host)
	if err != nil {
		return "", err
	}
	return token, nil
}

// doGitHubRequest executes an HTTP request to the GitHub API
func doGitHubRequest(ctx context.Context, method string, url string, authHeader string, accept string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if accept == "" {
		accept = "application/vnd.github+json"
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "vTeam-Backend")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	// Optional If-None-Match can be set by callers via context
	if v := ctx.Value("ifNoneMatch"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			req.Header.Set("If-None-Match", s)
		}
	}
	client := &http.Client{Timeout: 15 * time.Second}
	return client.Do(req)
}

// Removed legacy project-scoped link handler and GitHubAppInstallation type (moved to handlers/github_auth.go)

// getGitHubInstallation retrieves GitHub App installation for a user (wrapper to handlers package)
func getGitHubInstallation(ctx context.Context, userID string) (*handlers.GitHubAppInstallation, error) {
	return handlers.GetGitHubInstallation(ctx, userID)
}

// listUserForks handles GET /projects/:projectName/users/forks
// List user forks for an upstream repo (RBAC-scoped)
func listUserForks(c *gin.Context) {
	project := c.Param("projectName")
	upstreamRepo := c.Query("upstreamRepo")

	if upstreamRepo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "upstreamRepo query parameter required"})
		return
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := getK8sClientsForRequest(c)

	// Try to get GitHub token (GitHub App or PAT from runner secret)
	token, err := getGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	owner, repoName, err := parseOwnerRepo(upstreamRepo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	api := githubAPIBaseURL("github.com")
	// Fetch all pages of forks (public + any accessible private). Cap pages for safety.
	allForksResp := make([]map[string]interface{}, 0, 100)
	const perPage = 100
	for page := 1; page <= 10; page++ { // safety cap: up to 1000 forks
		url := fmt.Sprintf("%s/repos/%s/%s/forks?per_page=%d&page=%d", api, owner, repoName, perPage, page)
		resp, err := doGitHubRequest(c.Request.Context(), http.MethodGet, url, "Bearer "+token, "", nil)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("GitHub request failed: %v", err)})
			return
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			c.JSON(resp.StatusCode, gin.H{"error": string(b)})
			return
		}
		var pageForks []map[string]interface{}
		decErr := json.NewDecoder(resp.Body).Decode(&pageForks)
		_ = resp.Body.Close()
		if decErr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to parse GitHub response: %v", decErr)})
			return
		}
		if len(pageForks) == 0 {
			break
		}
		allForksResp = append(allForksResp, pageForks...)
		if len(pageForks) < perPage {
			break
		}
	}
	// Map all forks
	all := make([]map[string]interface{}, 0, len(allForksResp))
	for _, f := range allForksResp {
		name, _ := f["name"].(string)
		full, _ := f["full_name"].(string)
		html, _ := f["html_url"].(string)
		all = append(all, map[string]interface{}{
			"name":     name,
			"fullName": full,
			"url":      html,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"forks": all,
	})
}

// createUserFork handles POST /projects/:projectName/users/forks
// Create a fork of the upstream umbrella repo for the user
func createUserFork(c *gin.Context) {
	project := c.Param("projectName")

	var req struct {
		UpstreamRepo string `json:"upstreamRepo" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := getK8sClientsForRequest(c)

	// Try to get GitHub token (GitHub App or PAT from runner secret)
	token, err := getGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	owner, repoName, err := parseOwnerRepo(req.UpstreamRepo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	api := githubAPIBaseURL("github.com")
	url := fmt.Sprintf("%s/repos/%s/%s/forks", api, owner, repoName)
	resp, err := doGitHubRequest(c.Request.Context(), http.MethodPost, url, "Bearer "+token, "", nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("GitHub request failed: %v", err)})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		b, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{"error": string(b)})
		return
	}
	// Respond that fork creation is in progress or created
	c.JSON(http.StatusAccepted, gin.H{"message": "Fork creation requested", "upstreamRepo": req.UpstreamRepo})
}

// getRepoTree handles GET /projects/:projectName/repo/tree
// Fetch repo tree entries via backend proxy
func getRepoTree(c *gin.Context) {
	project := c.Param("projectName")
	repo := c.Query("repo")
	ref := c.Query("ref")
	path := c.Query("path")

	if repo == "" || ref == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo and ref query parameters required"})
		return
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := getK8sClientsForRequest(c)

	// Try to get GitHub token (GitHub App or PAT from runner secret)
	token, err := getGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	owner, repoName, err := parseOwnerRepo(repo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	api := githubAPIBaseURL("github.com")
	p := path
	if p == "" || p == "/" {
		p = ""
	}
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", api, owner, repoName, strings.TrimPrefix(p, "/"), ref)
	resp, err := doGitHubRequest(c.Request.Context(), http.MethodGet, url, "Bearer "+token, "", nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("GitHub request failed: %v", err)})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{"error": string(b)})
		return
	}
	var decoded interface{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to parse GitHub response: %v", err)})
		return
	}
	entries := []map[string]interface{}{}
	if arr, ok := decoded.([]interface{}); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				name, _ := m["name"].(string)
				typ, _ := m["type"].(string)
				size, _ := m["size"].(float64)
				mapped := "blob"
				switch strings.ToLower(typ) {
				case "dir":
					mapped = "tree"
				case "file", "symlink", "submodule":
					mapped = "blob"
				default:
					if strings.TrimSpace(typ) == "" {
						mapped = "blob"
					}
				}
				entries = append(entries, map[string]interface{}{"name": name, "type": mapped, "size": int(size)})
			}
		}
	} else if m, ok := decoded.(map[string]interface{}); ok {
		// single file; present as one entry
		name, _ := m["name"].(string)
		typ, _ := m["type"].(string)
		size, _ := m["size"].(float64)
		mapped := "blob"
		if strings.ToLower(typ) == "dir" {
			mapped = "tree"
		}
		entries = append(entries, map[string]interface{}{"name": name, "type": mapped, "size": int(size)})
	}
	c.JSON(http.StatusOK, map[string]interface{}{"path": path, "entries": entries})
}

// getRepoBlob handles GET /projects/:projectName/repo/blob
// Fetch blob (text) via backend proxy
func getRepoBlob(c *gin.Context) {
	project := c.Param("projectName")
	repo := c.Query("repo")
	ref := c.Query("ref")
	path := c.Query("path")

	if repo == "" || ref == "" || path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo, ref, and path query parameters required"})
		return
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := getK8sClientsForRequest(c)

	// Try to get GitHub token (GitHub App or PAT from runner secret)
	token, err := getGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	owner, repoName, err := parseOwnerRepo(repo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	api := githubAPIBaseURL("github.com")
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", api, owner, repoName, strings.TrimPrefix(path, "/"), ref)
	resp, err := doGitHubRequest(c.Request.Context(), http.MethodGet, url, "Bearer "+token, "", nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("GitHub request failed: %v", err)})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{"error": string(b)})
		return
	}
	// Decode generically first because GitHub returns an array for directories
	var decoded interface{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to parse GitHub response: %v", err)})
		return
	}
	// If the response is an array, the path is a directory. Return entries for convenience.
	if arr, ok := decoded.([]interface{}); ok {
		entries := []map[string]interface{}{}
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				name, _ := m["name"].(string)
				typ, _ := m["type"].(string)
				size, _ := m["size"].(float64)
				mapped := "blob"
				switch strings.ToLower(typ) {
				case "dir":
					mapped = "tree"
				case "file", "symlink", "submodule":
					mapped = "blob"
				default:
					if strings.TrimSpace(typ) == "" {
						mapped = "blob"
					}
				}
				entries = append(entries, map[string]interface{}{"name": name, "type": mapped, "size": int(size)})
			}
		}
		c.JSON(http.StatusOK, gin.H{"isDir": true, "path": path, "entries": entries})
		return
	}
	// Otherwise, treat as a file object
	if m, ok := decoded.(map[string]interface{}); ok {
		content, _ := m["content"].(string)
		encoding, _ := m["encoding"].(string)
		if strings.ToLower(encoding) == "base64" {
			raw := strings.ReplaceAll(content, "\n", "")
			if data, err := base64.StdEncoding.DecodeString(raw); err == nil {
				c.JSON(http.StatusOK, gin.H{"content": string(data), "encoding": "utf-8"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"content": content, "encoding": encoding})
		return
	}
	// Fallback unexpected structure
	c.JSON(http.StatusBadGateway, gin.H{"error": "unexpected GitHub response structure"})
}

// Removed hasProjectAccess in favor of relying on Kubernetes RBAC via downstream API calls
