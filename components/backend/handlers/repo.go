package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Dependencies injected from main package
var (
	GetK8sClientsForRequestRepo func(*gin.Context) (*kubernetes.Clientset, dynamic.Interface)
	GetGitHubTokenRepo          func(context.Context, *kubernetes.Clientset, dynamic.Interface, string, string) (string, error)
)

// ===== Helper Functions =====

// parseOwnerRepo parses repo into owner/repo from either owner/repo or full URL/SSH
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

// Note: githubAPIBaseURL and doGitHubRequest are defined in github_auth.go

// ===== Handler Functions =====

// AccessCheck verifies if the caller has write access to ProjectSettings in the project namespace
// It performs a Kubernetes SelfSubjectAccessReview using the caller token (user or API key).
func AccessCheck(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequestRepo(c)

	// Build the SSAR spec for RoleBinding management in the project namespace
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "rbac.authorization.k8s.io",
				Resource:  "rolebindings",
				Verb:      "create",
				Namespace: projectName,
			},
		},
	}

	// Perform the review
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(c.Request.Context(), ssar, v1.CreateOptions{})
	if err != nil {
		log.Printf("SSAR failed for project %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to perform access review"})
		return
	}

	role := "view"
	if res.Status.Allowed {
		// If update on ProjectSettings is allowed, treat as admin for this page
		role = "admin"
	} else {
		// Optional: try a lesser check for create sessions to infer "edit"
		editSSAR := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authv1.ResourceAttributes{
					Group:     "vteam.ambient-code",
					Resource:  "agenticsessions",
					Verb:      "create",
					Namespace: projectName,
				},
			},
		}
		res2, err2 := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(c.Request.Context(), editSSAR, v1.CreateOptions{})
		if err2 == nil && res2.Status.Allowed {
			role = "edit"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"project":  projectName,
		"allowed":  res.Status.Allowed,
		"reason":   res.Status.Reason,
		"userRole": role,
	})
}

// ListUserForks handles GET /projects/:projectName/users/forks
// List user forks for an upstream repo (RBAC-scoped)
func ListUserForks(c *gin.Context) {
	project := c.Param("projectName")
	upstreamRepo := c.Query("upstreamRepo")

	if upstreamRepo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "upstreamRepo query parameter required"})
		return
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := GetK8sClientsForRequestRepo(c)

	// Try to get GitHub token (GitHub App or PAT from runner secret)
	token, err := GetGitHubTokenRepo(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
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

// CreateUserFork handles POST /projects/:projectName/users/forks
// Create a fork of the upstream umbrella repo for the user
func CreateUserFork(c *gin.Context) {
	project := c.Param("projectName")

	var req struct {
		UpstreamRepo string `json:"upstreamRepo" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := GetK8sClientsForRequestRepo(c)

	// Try to get GitHub token (GitHub App or PAT from runner secret)
	token, err := GetGitHubTokenRepo(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
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

// GetRepoTree handles GET /projects/:projectName/repo/tree
// Fetch repo tree entries via backend proxy
func GetRepoTree(c *gin.Context) {
	project := c.Param("projectName")
	repo := c.Query("repo")
	ref := c.Query("ref")
	path := c.Query("path")

	if repo == "" || ref == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo and ref query parameters required"})
		return
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := GetK8sClientsForRequestRepo(c)

	// Try to get GitHub token (GitHub App or PAT from runner secret)
	token, err := GetGitHubTokenRepo(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
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

// ListRepoBranches handles GET /projects/:projectName/repo/branches
// List all branches in a repository
func ListRepoBranches(c *gin.Context) {
	project := c.Param("projectName")
	repo := c.Query("repo")

	if repo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo query parameter required"})
		return
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := GetK8sClientsForRequestRepo(c)

	// Try to get GitHub token (GitHub App or PAT from runner secret)
	token, err := GetGitHubTokenRepo(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
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
	url := fmt.Sprintf("%s/repos/%s/%s/branches", api, owner, repoName)
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

	var branchesResp []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&branchesResp); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to parse GitHub response: %v", err)})
		return
	}

	// Map branches to a simpler format
	branches := make([]map[string]interface{}, 0, len(branchesResp))
	for _, b := range branchesResp {
		name, _ := b["name"].(string)
		if name != "" {
			branches = append(branches, map[string]interface{}{
				"name": name,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"branches": branches,
	})
}

// GetRepoBlob handles GET /projects/:projectName/repo/blob
// Fetch blob (text) via backend proxy
func GetRepoBlob(c *gin.Context) {
	project := c.Param("projectName")
	repo := c.Query("repo")
	ref := c.Query("ref")
	path := c.Query("path")

	if repo == "" || ref == "" || path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo, ref, and path query parameters required"})
		return
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := GetK8sClientsForRequestRepo(c)

	// Try to get GitHub token (GitHub App or PAT from runner secret)
	token, err := GetGitHubTokenRepo(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
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
