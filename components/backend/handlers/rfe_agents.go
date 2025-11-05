// Package handlers provides HTTP handlers for the backend API.
// This file contains agent operations for RFE workflows.
package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Agent represents an agent definition from .claude/agents directory
type Agent struct {
	Persona     string `json:"persona"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Description string `json:"description"`
}

// GetProjectRFEWorkflowAgents fetches agent definitions from the workflow's umbrella repository
// GET /api/projects/:projectName/rfe-workflows/:id/agents
func GetProjectRFEWorkflowAgents(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	// Get the workflow
	gvr := GetRFEWorkflowResource()
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil || reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	item, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), id, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		} else if errors.IsForbidden(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this workflow"})
		} else {
			log.Printf("Failed to get workflow %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve workflow"})
		}
		return
	}
	wf := RfeFromUnstructured(item)
	if wf == nil || wf.UmbrellaRepo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No spec repo configured"})
		return
	}

	// Get user ID from forwarded identity middleware
	userID, _ := c.Get("userID")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User identity required"})
		return
	}

	githubToken, err := GetGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse repo owner/name from umbrella repo URL
	repoURL := wf.UmbrellaRepo.URL
	owner, repoName, err := parseOwnerRepoFromURL(repoURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid repository URL: %v", err)})
		return
	}

	// Get ref (branch) - use the generated feature branch, not the base branch
	ref := "main"
	if wf.BranchName != "" {
		ref = wf.BranchName
	} else if wf.UmbrellaRepo.Branch != nil {
		ref = *wf.UmbrellaRepo.Branch
	}

	// Fetch agents from .claude/agents directory
	agents, err := fetchAgentsFromRepo(c.Request.Context(), owner, repoName, ref, githubToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

// parseOwnerRepoFromURL extracts owner and repo name from a GitHub URL
func parseOwnerRepoFromURL(repoURL string) (string, string, error) {
	// Remove .git suffix
	repoURL = strings.TrimSuffix(repoURL, ".git")

	// Handle https://github.com/owner/repo
	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") {
		parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(repoURL, "https://"), "http://"), "/")
		if len(parts) >= 3 {
			return parts[1], parts[2], nil
		}
	}

	// Handle git@github.com:owner/repo
	if strings.Contains(repoURL, "@") {
		parts := strings.Split(repoURL, ":")
		if len(parts) == 2 {
			repoParts := strings.Split(parts[1], "/")
			if len(repoParts) == 2 {
				return repoParts[0], repoParts[1], nil
			}
		}
	}

	// Handle owner/repo format
	parts := strings.Split(repoURL, "/")
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("unable to parse repository URL")
}

// fetchAgentsFromRepo fetches and parses agent definitions from .claude/agents directory
func fetchAgentsFromRepo(ctx context.Context, owner, repo, ref, token string) ([]Agent, error) {
	api := "https://api.github.com"
	agentsPath := ".claude/agents"

	// Fetch directory listing
	treeURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", api, owner, repo, agentsPath, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, treeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No .claude/agents directory - return empty array
		return []Agent{}, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var treeEntries []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&treeEntries); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	// Filter for .md files
	var agentFiles []string
	for _, entry := range treeEntries {
		name, _ := entry["name"].(string)
		typ, _ := entry["type"].(string)
		if typ == "file" && strings.HasSuffix(name, ".md") {
			agentFiles = append(agentFiles, name)
		}
	}

	// Fetch and parse each agent file
	agents := make([]Agent, 0, len(agentFiles))
	for _, filename := range agentFiles {
		agent, err := fetchAndParseAgentFile(ctx, api, owner, repo, ref, filename, token)
		if err != nil {
			log.Printf("Warning: failed to parse agent file %s: %v", filename, err)
			continue
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// fetchAndParseAgentFile fetches a single agent file and parses its metadata
func fetchAndParseAgentFile(ctx context.Context, api, owner, repo, ref, filename, token string) (Agent, error) {
	agentPath := fmt.Sprintf(".claude/agents/%s", filename)
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", api, owner, repo, agentPath, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Agent{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Agent{}, fmt.Errorf("GitHub request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Agent{}, fmt.Errorf("GitHub returned status %d", resp.StatusCode)
	}

	var fileData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&fileData); err != nil {
		return Agent{}, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	// Decode base64 content
	content, _ := fileData["content"].(string)
	encoding, _ := fileData["encoding"].(string)

	var decodedContent string
	if strings.ToLower(encoding) == "base64" {
		raw := strings.ReplaceAll(content, "\n", "")
		data, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return Agent{}, fmt.Errorf("failed to decode base64 content: %w", err)
		}
		decodedContent = string(data)
	} else {
		decodedContent = content
	}

	// Parse persona from filename
	persona := strings.TrimSuffix(filename, ".md")

	// Generate default name from filename
	nameParts := strings.FieldsFunc(persona, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, part := range nameParts {
		if len(part) > 0 {
			nameParts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	name := strings.Join(nameParts, " ")

	role := ""
	description := ""

	// Try to extract metadata from YAML frontmatter
	// Simple regex-based parsing (consider using a YAML library for production)
	lines := strings.Split(decodedContent, "\n")
	inFrontmatter := false
	for i, line := range lines {
		if i == 0 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.TrimSpace(line) == "---" {
			break
		}
		if inFrontmatter {
			if strings.HasPrefix(line, "name:") {
				name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			} else if strings.HasPrefix(line, "role:") {
				role = strings.TrimSpace(strings.TrimPrefix(line, "role:"))
			} else if strings.HasPrefix(line, "description:") {
				description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			}
		}
	}

	// If no description found, use first non-empty line after frontmatter
	if description == "" {
		afterFrontmatter := false
		for _, line := range lines {
			if afterFrontmatter {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
					description = trimmed
					if len(description) > 150 {
						description = description[:150]
					}
					break
				}
			}
			if strings.TrimSpace(line) == "---" {
				if afterFrontmatter {
					break
				}
				afterFrontmatter = true
			}
		}
	}

	if description == "" {
		description = "No description available"
	}

	return Agent{
		Persona:     persona,
		Name:        name,
		Role:        role,
		Description: description,
	}, nil
}

// GetWorkflowJira proxies Jira issue fetch for a linked path
// GET /api/projects/:projectName/rfe-workflows/:id/jira?path=...
func GetWorkflowJira(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")
	reqPath := strings.TrimSpace(c.Query("path"))
	if reqPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}
	_, reqDyn := GetK8sClientsForRequest(c)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqDyn == nil || reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}
	// Load workflow to find key
	gvrWf := GetRFEWorkflowResource()
	item, err := reqDyn.Resource(gvrWf).Namespace(project).Get(c.Request.Context(), id, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	wf := RfeFromUnstructured(item)
	var key string
	for _, jl := range wf.JiraLinks {
		if strings.TrimSpace(jl.Path) == reqPath {
			key = jl.JiraKey
			break
		}
	}
	if key == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No Jira linked for path"})
		return
	}
	// Load Jira creds
	// Determine secret name
	secretName := "ambient-runner-secrets"
	if obj, err := reqDyn.Resource(GetProjectSettingsResource()).Namespace(project).Get(c.Request.Context(), "projectsettings", v1.GetOptions{}); err == nil {
		if spec, ok := GetSpecMap(obj); ok {
			if v, ok := spec["runnerSecretsName"].(string); ok && strings.TrimSpace(v) != "" {
				secretName = strings.TrimSpace(v)
			}
		}
	}
	sec, err := reqK8s.CoreV1().Secrets(project).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read runner secret", "details": err.Error()})
		return
	}
	get := func(k string) string {
		if b, ok := sec.Data[k]; ok {
			return string(b)
		}
		return ""
	}
	jiraURL := strings.TrimSpace(get("JIRA_URL"))
	jiraToken := strings.TrimSpace(get("JIRA_API_TOKEN"))
	if jiraURL == "" || jiraToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing Jira configuration in runner secret (JIRA_URL, JIRA_API_TOKEN required)"})
		return
	}
	// Determine auth header (Cloud vs Server/Data Center)
	authHeader := ""
	if strings.Contains(jiraURL, "atlassian.net") {
		// Jira Cloud - assume token is email:api_token format
		encoded := base64.StdEncoding.EncodeToString([]byte(jiraToken))
		authHeader = "Basic " + encoded
	} else {
		// Jira Server/Data Center
		authHeader = "Bearer " + jiraToken
	}

	jiraBase := strings.TrimRight(jiraURL, "/")
	endpoint := fmt.Sprintf("%s/rest/api/2/issue/%s", jiraBase, url.PathEscape(key))
	httpReq, _ := http.NewRequest("GET", endpoint, nil)
	httpReq.Header.Set("Authorization", authHeader)
	httpClient := &http.Client{Timeout: 30 * time.Second}
	httpResp, httpErr := httpClient.Do(httpReq)
	if httpErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Jira request failed", "details": httpErr.Error()})
		return
	}
	defer httpResp.Body.Close()
	respBody, _ := io.ReadAll(httpResp.Body)
	c.Data(httpResp.StatusCode, "application/json", respBody)
}
