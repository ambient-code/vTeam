package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// GitHubAppInstallation represents a GitHub App installation for a user
type GitHubAppInstallation struct {
	UserID         string    `json:"userId"`
	GitHubUserID   string    `json:"githubUserId"`
	InstallationID int64     `json:"installationId"`
	Host           string    `json:"host"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ===== OAuth during installation (user verification) =====

// signState signs a payload with HMAC SHA-256
func signState(secret string, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func handleGitHubUserOAuthCallback(c *gin.Context) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	stateSecret := os.Getenv("GITHUB_STATE_SECRET")
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" || strings.TrimSpace(stateSecret) == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth not configured"})
		return
	}
	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}
	// Defaults when no state provided
	var retB64 string
	var instID int64
	// Validate state if present
	if state != "" {
		raw, err := base64.RawURLEncoding.DecodeString(state)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}
		parts := strings.SplitN(string(raw), ".", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}
		payload, sig := parts[0], parts[1]
		if signState(stateSecret, payload) != sig {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad state signature"})
			return
		}
		fields := strings.Split(payload, ":")
		if len(fields) != 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad state payload"})
			return
		}
		userInState := fields[0]
		ts := fields[1]
		retB64 = fields[3]
		instB64 := fields[4]
		if sec, err := strconv.ParseInt(ts, 10, 64); err == nil {
			if time.Since(time.Unix(sec, 0)) > 10*time.Minute {
				c.JSON(http.StatusBadRequest, gin.H{"error": "state expired"})
				return
			}
		}
		// Confirm current session user matches state user
		userID, _ := c.Get("userID")
		if userID == nil || userInState != userID.(string) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user mismatch"})
			return
		}
		// Decode installation id from state
		instBytes, _ := base64.RawURLEncoding.DecodeString(instB64)
		instStr := string(instBytes)
		instID, _ = strconv.ParseInt(instStr, 10, 64)
	} else {
		// No state (install started outside our UI). Require user session and read installation_id from query.
		userID, _ := c.Get("userID")
		if userID == nil || strings.TrimSpace(userID.(string)) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
			return
		}
		instStr := c.Query("installation_id")
		var err error
		instID, err = strconv.ParseInt(instStr, 10, 64)
		if err != nil || instID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid installation id"})
			return
		}
	}
	// Exchange code → user token
	token, err := exchangeOAuthCodeForUserToken(clientID, clientSecret, code)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "oauth exchange failed"})
		return
	}
	// Verify ownership: GET /user/installations includes the installation
	owns, login, err := userOwnsInstallation(token, instID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "verification failed"})
		return
	}
	if !owns {
		c.JSON(http.StatusForbidden, gin.H{"error": "installation not owned by user"})
		return
	}
	// Store mapping
	installation := GitHubAppInstallation{
		UserID:         c.GetString("userID"),
		GitHubUserID:   login,
		InstallationID: instID,
		Host:           "github.com",
		UpdatedAt:      time.Now(),
	}
	if err := storeGitHubInstallation(c.Request.Context(), "", &installation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store installation"})
		return
	}
	// Redirect back to return_to if present
	retURL := "/integrations"
	if retB64 != "" {
		if b, err := base64.RawURLEncoding.DecodeString(retB64); err == nil {
			retURL = string(b)
		}
	}
	if retURL == "" {
		retURL = "/integrations"
	}
	c.Redirect(http.StatusFound, retURL)
}

func exchangeOAuthCodeForUserToken(clientID, clientSecret, code string) (string, error) {
	reqBody := strings.NewReader(fmt.Sprintf("client_id=%s&client_secret=%s&code=%s", clientID, clientSecret, code))
	req, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", reqBody)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("empty token")
	}
	return parsed.AccessToken, nil
}

func userOwnsInstallation(userToken string, installationID int64) (bool, string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user/installations", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "token "+userToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	var data struct {
		Installations []struct {
			Id      int64 `json:"id"`
			Account struct {
				Login string `json:"login"`
			} `json:"account"`
		} `json:"installations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, "", err
	}
	for _, inst := range data.Installations {
		if inst.Id == installationID {
			return true, inst.Account.Login, nil
		}
	}
	return false, "", nil
}

// Removed legacy project-scoped link handler

// storeGitHubInstallation persists the GitHub App installation mapping
func storeGitHubInstallation(ctx context.Context, projectName string, installation *GitHubAppInstallation) error {
	if installation == nil || installation.UserID == "" {
		return fmt.Errorf("invalid installation payload")
	}
	// Cluster-scoped by server namespace; ignore projectName for storage
	const cmName = "github-app-installations"
	for i := 0; i < 3; i++ { // retry on conflict
		cm, err := k8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, cmName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// create
				cm = &corev1.ConfigMap{ObjectMeta: v1.ObjectMeta{Name: cmName, Namespace: namespace}, Data: map[string]string{}}
				if _, cerr := k8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, cm, v1.CreateOptions{}); cerr != nil && !errors.IsAlreadyExists(cerr) {
					return fmt.Errorf("failed to create ConfigMap: %w", cerr)
				}
				// fetch again to get resourceVersion
				cm, err = k8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, cmName, v1.GetOptions{})
				if err != nil {
					return fmt.Errorf("failed to fetch ConfigMap after create: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get ConfigMap: %w", err)
			}
		}
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		b, err := json.Marshal(installation)
		if err != nil {
			return fmt.Errorf("failed to marshal installation: %w", err)
		}
		cm.Data[installation.UserID] = string(b)
		if _, uerr := k8sClient.CoreV1().ConfigMaps(namespace).Update(ctx, cm, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue // retry
			}
			return fmt.Errorf("failed to update ConfigMap: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update ConfigMap after retries")
}

// getGitHubInstallation retrieves GitHub App installation for a user
func getGitHubInstallation(ctx context.Context, userID string) (*GitHubAppInstallation, error) {
	const cmName = "github-app-installations"
	cm, err := k8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, cmName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("installation not found")
		}
		return nil, fmt.Errorf("failed to read ConfigMap: %w", err)
	}
	if cm.Data == nil {
		return nil, fmt.Errorf("installation not found")
	}
	raw, ok := cm.Data[userID]
	if !ok || raw == "" {
		return nil, fmt.Errorf("installation not found")
	}
	var inst GitHubAppInstallation
	if err := json.Unmarshal([]byte(raw), &inst); err != nil {
		return nil, fmt.Errorf("failed to decode installation: %w", err)
	}
	return &inst, nil
}

// listUserForks handles GET /projects/:projectName/users/forks
// List user forks for an upstream repo (RBAC-scoped)
func listUserForks(c *gin.Context) {
	upstreamRepo := c.Query("upstreamRepo")

	if upstreamRepo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "upstreamRepo query parameter required"})
		return
	}

	// Project access is enforced by Kubernetes RBAC on downstream operations

	userID, _ := c.Get("userID")

	// Get user's GitHub installation
	installation, err := getGitHubInstallation(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "GitHub App not installed for user"})
		return
	}

	owner, repoName, err := parseOwnerRepo(upstreamRepo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := mintInstallationToken(c.Request.Context(), installation.InstallationID, installation.Host)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to mint token: %v", err)})
		return
	}
	api := githubAPIBaseURL(installation.Host)
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
	fmt.Printf("forksResp: %+v\n", allForksResp)
	fmt.Printf("installation: %+v\n", installation)
	fmt.Printf("all: %+v\n", all)
	// Return all forks without filtering
	c.JSON(http.StatusOK, gin.H{
		"forks": all,
	})
}

// createUserFork handles POST /projects/:projectName/users/forks
// Create a fork of the upstream umbrella repo for the user
func createUserFork(c *gin.Context) {

	var req struct {
		UpstreamRepo string `json:"upstreamRepo" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Project access is enforced by Kubernetes RBAC on downstream operations

	userID, _ := c.Get("userID")

	// Get user's GitHub installation
	installation, err := getGitHubInstallation(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "GitHub App not installed for user"})
		return
	}
	owner, repoName, err := parseOwnerRepo(req.UpstreamRepo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := mintInstallationToken(c.Request.Context(), installation.InstallationID, installation.Host)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to mint token: %v", err)})
		return
	}
	api := githubAPIBaseURL(installation.Host)
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
	repo := c.Query("repo")
	ref := c.Query("ref")
	path := c.Query("path")

	if repo == "" || ref == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo and ref query parameters required"})
		return
	}

	// Project access is enforced by Kubernetes RBAC on downstream operations

	userID, _ := c.Get("userID")
	installation, err := getGitHubInstallation(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "GitHub App not installed for user"})
		return
	}
	owner, repoName, err := parseOwnerRepo(repo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := mintInstallationToken(c.Request.Context(), installation.InstallationID, installation.Host)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to mint token: %v", err)})
		return
	}
	api := githubAPIBaseURL(installation.Host)
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
	repo := c.Query("repo")
	ref := c.Query("ref")
	path := c.Query("path")

	if repo == "" || ref == "" || path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo, ref, and path query parameters required"})
		return
	}

	// Project access is enforced by Kubernetes RBAC on downstream operations

	userID, _ := c.Get("userID")
	installation, err := getGitHubInstallation(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "GitHub App not installed for user"})
		return
	}
	owner, repoName, err := parseOwnerRepo(repo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := mintInstallationToken(c.Request.Context(), installation.InstallationID, installation.Host)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to mint token: %v", err)})
		return
	}
	api := githubAPIBaseURL(installation.Host)
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

// ===== Global, non-project-scoped endpoints =====

// linkGitHubInstallationGlobal handles POST /auth/github/install
// Links the current SSO user to a GitHub App installation ID.
func linkGitHubInstallationGlobal(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil || strings.TrimSpace(userID.(string)) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
		return
	}
	var req struct {
		InstallationID int64 `json:"installationId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	installation := GitHubAppInstallation{
		UserID:         userID.(string),
		InstallationID: req.InstallationID,
		Host:           "github.com",
		UpdatedAt:      time.Now(),
	}
	// Best-effort: enrich with GitHub account login for the installation
	if githubTokenManager != nil {
		if jwt, err := githubTokenManager.GenerateJWT(); err == nil {
			api := githubAPIBaseURL(installation.Host)
			url := fmt.Sprintf("%s/app/installations/%d", api, req.InstallationID)
			resp, err := doGitHubRequest(c.Request.Context(), http.MethodGet, url, "Bearer "+jwt, "", nil)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					var instObj map[string]interface{}
					if err := json.NewDecoder(resp.Body).Decode(&instObj); err == nil {
						if acct, ok := instObj["account"].(map[string]interface{}); ok {
							if login, ok := acct["login"].(string); ok {
								installation.GitHubUserID = login
							}
						}
					}
				}
			}
		}
	}
	if err := storeGitHubInstallation(c.Request.Context(), "", &installation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store installation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "GitHub App installation linked successfully", "installationId": req.InstallationID})
}

// getGitHubStatusGlobal handles GET /auth/github/status
func getGitHubStatusGlobal(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil || strings.TrimSpace(userID.(string)) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
		return
	}
	inst, err := getGitHubInstallation(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"installed": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"installed":      true,
		"installationId": inst.InstallationID,
		"host":           inst.Host,
		"githubUserId":   inst.GitHubUserID,
		"userId":         inst.UserID,
		"updatedAt":      inst.UpdatedAt.Format(time.RFC3339),
	})
}

// deleteGitHubInstallation removes the user mapping from ConfigMap
func deleteGitHubInstallation(ctx context.Context, userID string) error {
	const cmName = "github-app-installations"
	cm, err := k8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, cmName, v1.GetOptions{})
	if err != nil {
		return err
	}
	if cm.Data == nil {
		return nil
	}
	delete(cm.Data, userID)
	_, uerr := k8sClient.CoreV1().ConfigMaps(namespace).Update(ctx, cm, v1.UpdateOptions{})
	return uerr
}

// disconnectGitHubGlobal handles POST /auth/github/disconnect
func disconnectGitHubGlobal(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil || strings.TrimSpace(userID.(string)) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
		return
	}
	if err := deleteGitHubInstallation(c.Request.Context(), userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlink installation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "GitHub account disconnected"})
}
