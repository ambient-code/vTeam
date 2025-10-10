package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ambient-code-backend/git"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// POST /api/projects/:projectName/rfe-workflows/:id/jira { path }
// Creates or updates a Jira issue from a GitHub file and updates the RFEWorkflow CR with the linkage
func publishWorkflowFileToJira(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	// Load runner secrets for Jira config
	_, reqDyn := getK8sClientsForRequest(c)
	reqK8s, _ := getK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}

	// Determine configured secret name
	secretName := ""
	if reqDyn != nil {
		gvr := getProjectSettingsResource()
		if obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), "projectsettings", v1.GetOptions{}); err == nil {
			if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
				if v, ok := spec["runnerSecretsName"].(string); ok {
					secretName = strings.TrimSpace(v)
				}
			}
		}
	}
	if secretName == "" {
		secretName = "ambient-runner-secrets"
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
	jiraProject := strings.TrimSpace(get("JIRA_PROJECT"))
	jiraToken := strings.TrimSpace(get("JIRA_API_TOKEN"))
	if jiraURL == "" || jiraProject == "" || jiraToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing Jira configuration in runner secret (JIRA_URL, JIRA_PROJECT, JIRA_API_TOKEN required)"})
		return
	}

	// Load workflow
	gvrWf := getRFEWorkflowResource()
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}
	item, err := reqDyn.Resource(gvrWf).Namespace(project).Get(c.Request.Context(), id, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	wf := rfeFromUnstructured(item)
	if wf == nil || wf.UmbrellaRepo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workflow has no umbrella repo configured"})
		return
	}

	// Get user ID and GitHub token
	userID, _ := c.Get("userID")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User identity required"})
		return
	}
	githubToken, err := git.GetGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get GitHub token", "details": err.Error()})
		return
	}

	// Read file from GitHub
	owner, repo, err := git.ParseGitHubURL(wf.UmbrellaRepo.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid umbrella repo URL", "details": err.Error()})
		return
	}
	branch := "main"
	if wf.UmbrellaRepo.Branch != nil && strings.TrimSpace(*wf.UmbrellaRepo.Branch) != "" {
		branch = strings.TrimSpace(*wf.UmbrellaRepo.Branch)
	}
	content, err := git.ReadGitHubFile(c.Request.Context(), owner, repo, branch, req.Path, githubToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read file from GitHub", "details": err.Error()})
		return
	}

	// Extract title from markdown content
	title := extractTitleFromContent(string(content))
	if title == "" {
		title = wf.Title // Fallback to workflow title
	}

	// Check if Jira link already exists for this path
	existingKey := ""
	for _, jl := range wf.JiraLinks {
		if strings.TrimSpace(jl.Path) == strings.TrimSpace(req.Path) {
			existingKey = jl.JiraKey
			break
		}
	}

	// Determine auth header (Cloud vs Server)
	authHeader := ""
	if strings.Contains(jiraURL, "atlassian.net") {
		// Jira Cloud - assume token is email:api_token format
		encoded := base64.StdEncoding.EncodeToString([]byte(jiraToken))
		authHeader = "Basic " + encoded
	} else {
		// Jira Server/Data Center
		authHeader = "Bearer " + jiraToken
	}

	// Create or update Jira issue
	jiraBase := strings.TrimRight(jiraURL, "/")
	var httpReq *http.Request

	if existingKey == "" {
		// CREATE new issue
		jiraEndpoint := fmt.Sprintf("%s/rest/api/2/issue", jiraBase)

		fields := map[string]interface{}{
			"project":     map[string]string{"key": jiraProject},
			"summary":     title,
			"description": string(content),
			"issuetype":   map[string]string{"name": "Feature"},
		}

		// Add parent Outcome if specified
		if wf.ParentOutcome != nil && *wf.ParentOutcome != "" {
			fields["parent"] = map[string]string{"key": *wf.ParentOutcome}
		}

		reqBody := map[string]interface{}{"fields": fields}
		payload, _ := json.Marshal(reqBody)
		httpReq, _ = http.NewRequestWithContext(c.Request.Context(), "POST", jiraEndpoint, bytes.NewReader(payload))
	} else {
		// UPDATE existing issue
		jiraEndpoint := fmt.Sprintf("%s/rest/api/2/issue/%s", jiraBase, url.PathEscape(existingKey))
		reqBody := map[string]interface{}{
			"fields": map[string]interface{}{
				"summary":     title,
				"description": string(content),
			},
		}
		payload, _ := json.Marshal(reqBody)
		httpReq, _ = http.NewRequestWithContext(c.Request.Context(), "PUT", jiraEndpoint, bytes.NewReader(payload))
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", authHeader)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	httpResp, httpErr := httpClient.Do(httpReq)
	if httpErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Jira request failed", "details": httpErr.Error()})
		return
	}
	defer httpResp.Body.Close()

	respBody, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		c.Data(httpResp.StatusCode, "application/json", respBody)
		return
	}

	// Extract Jira key from response
	var outKey string
	if existingKey == "" {
		var created struct {
			Key string `json:"key"`
		}
		_ = json.Unmarshal(respBody, &created)
		if strings.TrimSpace(created.Key) == "" {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Jira creation returned no key"})
			return
		}
		outKey = created.Key
	} else {
		outKey = existingKey
	}

	// Update RFEWorkflow CR with Jira link
	obj := item.DeepCopy()
	spec, _ := obj.Object["spec"].(map[string]interface{})
	if spec == nil {
		spec = map[string]interface{}{}
		obj.Object["spec"] = spec
	}

	var links []interface{}
	if existing, ok := spec["jiraLinks"].([]interface{}); ok {
		links = existing
	}

	// Add or update link
	found := false
	for _, li := range links {
		if m, ok := li.(map[string]interface{}); ok {
			if fmt.Sprintf("%v", m["path"]) == req.Path {
				m["jiraKey"] = outKey
				found = true
				break
			}
		}
	}
	if !found {
		links = append(links, map[string]interface{}{"path": req.Path, "jiraKey": outKey})
	}
	spec["jiraLinks"] = links

	if _, err := reqDyn.Resource(gvrWf).Namespace(project).Update(c.Request.Context(), obj, v1.UpdateOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update workflow with Jira link", "details": err.Error()})
		return
	}

	// Return success
	c.JSON(http.StatusOK, gin.H{
		"key": outKey,
		"url": fmt.Sprintf("%s/browse/%s", jiraBase, outKey),
	})
}
