package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"ambient-code-backend/internal/middleware"
	"ambient-code-backend/internal/services"
	"ambient-code-backend/internal/types"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// resolveWorkspaceAbsPath normalizes a workspace-relative or absolute path to the
// absolute workspace path for a given session.
func resolveWorkspaceAbsPath(sessionName string, relOrAbs string) string {
	base := fmt.Sprintf("/sessions/%s/workspace", sessionName)
	trimmed := strings.TrimSpace(relOrAbs)
	if trimmed == "" || trimmed == "/" {
		return base
	}
	cleaned := "/" + strings.TrimLeft(trimmed, "/")
	if cleaned == base || strings.HasPrefix(cleaned, base+"/") {
		return cleaned
	}
	// Join under base for any other relative path
	return filepath.Join(base, strings.TrimPrefix(cleaned, "/"))
}

// GetSessionWorkspace lists the workspace contents for an agentic session
// Lists the contents of a session's workspace by delegating to the per-project content service
func GetSessionWorkspace(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")

	// Optional subpath within the workspace to list
	rel := strings.TrimSpace(c.Query("path"))
	absPath := resolveWorkspaceAbsPath(sessionName, rel)

	items, err := services.ListProjectContent(c, project, absPath)
	if err == nil {
		// If content/list returns exactly this file (non-dir), serve file bytes
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			b, ferr := services.ReadProjectContentFile(c, project, absPath)
			if ferr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read workspace file"})
				return
			}
			c.Data(http.StatusOK, "application/octet-stream", b)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}
	// Fallback: try file read directly
	b, ferr := services.ReadProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// GetSessionWorkspaceFile reads a specific file from the session workspace
// Reads a file from a session's workspace by delegating to the per-project content service
func GetSessionWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	pathParam := c.Param("path")

	absPath := resolveWorkspaceAbsPath(sessionName, pathParam)

	// Try directory listing first to determine type
	items, err := services.ListProjectContent(c, project, absPath)
	if err == nil {
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			// It's a file
			b, ferr := services.ReadProjectContentFile(c, project, absPath)
			if ferr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read workspace file"})
				return
			}
			c.Data(http.StatusOK, "application/octet-stream", b)
			return
		}
		// It's a directory
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}
	// Fallback to file read
	b, ferr := services.ReadProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// PutSessionWorkspaceFile writes a file to the session workspace
// Writes a file into a session's workspace via the per-project content service
func PutSessionWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	pathParam := c.Param("path")

	absPath := resolveWorkspaceAbsPath(sessionName, pathParam)

	// Read raw request body and forward as-is (treat as text/binary pass-through)
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if err := services.WriteProjectContentFile(c, project, absPath, data); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to write workspace file"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// resolveWorkflowWorkspaceAbsPath normalizes a workspace-relative or absolute path to the
// absolute workspace path for a given RFE workflow.
func resolveWorkflowWorkspaceAbsPath(workflowID string, relOrAbs string) string {
	base := fmt.Sprintf("/rfe-workflows/%s/workspace", workflowID)
	trimmed := strings.TrimSpace(relOrAbs)
	if trimmed == "" || trimmed == "/" {
		return base
	}
	cleaned := "/" + strings.TrimLeft(trimmed, "/")
	if cleaned == base || strings.HasPrefix(cleaned, base+"/") {
		return cleaned
	}
	// Join under base for any other relative path
	return filepath.Join(base, strings.TrimPrefix(cleaned, "/"))
}

// GetRFEWorkflowWorkspace lists the workspace contents for an RFE workflow
// Lists the contents of a workflow's workspace by delegating to the per-project content service
func GetRFEWorkflowWorkspace(c *gin.Context) {
	project := c.GetString("project")
	workflowID := c.Param("id")

	// Optional subpath within the workspace to list
	rel := strings.TrimSpace(c.Query("path"))
	absPath := resolveWorkflowWorkspaceAbsPath(workflowID, rel)

	items, err := services.ListProjectContent(c, project, absPath)
	if err == nil {
		// If content/list returns exactly this file (non-dir), serve file bytes
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			b, ferr := services.ReadProjectContentFile(c, project, absPath)
			if ferr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read workspace file"})
				return
			}
			c.Data(http.StatusOK, "application/octet-stream", b)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}
	// Fallback: try file read directly
	b, ferr := services.ReadProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// GetRFEWorkflowWorkspaceFile reads a specific file from the RFE workflow workspace
// Reads a file from a workflow's workspace by delegating to the per-project content service
func GetRFEWorkflowWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	workflowID := c.Param("id")
	pathParam := c.Param("path")

	absPath := resolveWorkflowWorkspaceAbsPath(workflowID, pathParam)

	// Try directory listing first to determine type
	items, err := services.ListProjectContent(c, project, absPath)
	if err == nil {
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			// It's a file
			b, ferr := services.ReadProjectContentFile(c, project, absPath)
			if ferr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read workspace file"})
				return
			}
			c.Data(http.StatusOK, "application/octet-stream", b)
			return
		}
		// It's a directory
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}
	// Fallback to file read
	b, ferr := services.ReadProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// PutRFEWorkflowWorkspaceFile writes a file to the RFE workflow workspace
// Writes a file into a workflow's workspace via the per-project content service
func PutRFEWorkflowWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	workflowID := c.Param("id")
	pathParam := c.Param("path")

	absPath := resolveWorkflowWorkspaceAbsPath(workflowID, pathParam)

	// Read raw request body and forward as-is (treat as text/binary pass-through)
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if err := services.WriteProjectContentFile(c, project, absPath, data); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to write workspace file"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// extractTitleFromContent attempts to extract a title from markdown content
// by looking for the first # heading
func extractTitleFromContent(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

// PublishWorkflowFileToJira publishes a workflow file to Jira and records linkage
func PublishWorkflowFileToJira(c *gin.Context) {
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
	// Reuse listRunnerSecrets helpers indirectly by reading the Secret directly
	_, reqDyn := middleware.GetK8sClientsForRequest(c)
	reqK8s, _ := middleware.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}

	// Determine configured secret name
	secretName := ""
	if reqDyn != nil {
		gvr := types.GetProjectSettingsResource()
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

	// Load workflow for title
	gvrWf := types.GetRFEWorkflowResource()
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}
	item, err := reqDyn.Resource(gvrWf).Namespace(project).Get(c.Request.Context(), id, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	wf := services.RFEFromUnstructured(item)

	// Read file content
	absPath := resolveWorkflowWorkspaceAbsPath(id, req.Path)
	b, ferr := services.ReadProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read workspace file"})
		return
	}
	content := string(b)

	// Extract title from spec content or fallback to workflow title
	title := extractTitleFromContent(content)
	if title == "" {
		title = wf.Title
	}

	// Create or update Jira issue (v2 API)
	jiraBase := strings.TrimRight(jiraURL, "/")
	// Check existing link for this path
	existingKey := ""
	for _, jl := range wf.JiraLinks {
		if strings.TrimSpace(jl.Path) == strings.TrimSpace(req.Path) {
			existingKey = jl.JiraKey
			break
		}
	}
	var httpReq *http.Request
	if existingKey == "" {
		// Create
		jiraEndpoint := fmt.Sprintf("%s/rest/api/2/issue", jiraBase)
		// Determine issue type based on file type
		issueType := "Feature"
		if strings.Contains(req.Path, "plan.md") {
			issueType = "Feature" // plan.md creates Features for now (was Epic)
		}

		reqBody := map[string]interface{}{
			"fields": map[string]interface{}{
				"project":     map[string]string{"key": jiraProject},
				"summary":     title,
				"description": content,
				"issuetype":   map[string]string{"name": issueType},
			},
		}
		payload, _ := json.Marshal(reqBody)
		httpReq, _ = http.NewRequest("POST", jiraEndpoint, bytes.NewReader(payload))
	} else {
		// Update existing
		jiraEndpoint := fmt.Sprintf("%s/rest/api/2/issue/%s", jiraBase, url.PathEscape(existingKey))
		reqBody := map[string]interface{}{
			"fields": map[string]interface{}{
				"summary":     title,
				"description": content,
			},
		}
		payload, _ := json.Marshal(reqBody)
		httpReq, _ = http.NewRequest("PUT", jiraEndpoint, bytes.NewReader(payload))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+jiraToken)
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

	// Update CR: append jiraLinks entry
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
	// Add only if new; if exists, update key
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

	c.JSON(http.StatusOK, gin.H{"key": outKey, "url": fmt.Sprintf("%s/browse/%s", jiraBase, outKey)})
}

// GetWorkflowJira gets Jira linkage information for a workflow
func GetWorkflowJira(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")
	reqPath := strings.TrimSpace(c.Query("path"))
	if reqPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}
	_, reqDyn := middleware.GetK8sClientsForRequest(c)
	reqK8s, _ := middleware.GetK8sClientsForRequest(c)
	if reqDyn == nil || reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}
	// Load workflow to find key
	gvrWf := types.GetRFEWorkflowResource()
	item, err := reqDyn.Resource(gvrWf).Namespace(project).Get(c.Request.Context(), id, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	wf := services.RFEFromUnstructured(item)
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
	if obj, err := reqDyn.Resource(types.GetProjectSettingsResource()).Namespace(project).Get(c.Request.Context(), "projectsettings", v1.GetOptions{}); err == nil {
		if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
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
	jiraBase := strings.TrimRight(jiraURL, "/")
	endpoint := fmt.Sprintf("%s/rest/api/2/issue/%s", jiraBase, url.PathEscape(key))
	httpReq, _ := http.NewRequest("GET", endpoint, nil)
	httpReq.Header.Set("Authorization", "Bearer "+jiraToken)
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
