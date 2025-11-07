// Package jira provides JIRA integration for publishing RFE workflows.
package jira

import (
	"bytes"
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

	"ambient-code-backend/git"
	"ambient-code-backend/handlers"
	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Handler dependencies
type Handler struct {
	GetK8sClientsForRequest    func(*gin.Context) (*kubernetes.Clientset, dynamic.Interface)
	GetProjectSettingsResource func() schema.GroupVersionResource
	GetRFEWorkflowResource     func() schema.GroupVersionResource
}

// RFEFromUnstructured converts an unstructured RFEWorkflow CR into our RFEWorkflow struct
func RFEFromUnstructured(item *unstructured.Unstructured) *types.RFEWorkflow {
	if item == nil {
		return nil
	}
	obj := item.Object
	spec, _ := obj["spec"].(map[string]interface{})

	created := ""
	if item.GetCreationTimestamp().Time != (time.Time{}) {
		created = item.GetCreationTimestamp().Time.UTC().Format(time.RFC3339)
	}

	// Extract branchName safely - avoid converting nil to "<nil>" string
	branchName := ""
	if bn, ok := spec["branchName"].(string); ok && strings.TrimSpace(bn) != "" {
		branchName = strings.TrimSpace(bn)
	}

	wf := &types.RFEWorkflow{
		ID:            item.GetName(),
		Title:         fmt.Sprintf("%v", spec["title"]),
		Description:   fmt.Sprintf("%v", spec["description"]),
		BranchName:    branchName,
		Project:       item.GetNamespace(),
		WorkspacePath: fmt.Sprintf("%v", spec["workspacePath"]),
		CreatedAt:     created,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	// Parse umbrellaRepo/supportingRepos when present; fallback to repositories
	if um, ok := spec["umbrellaRepo"].(map[string]interface{}); ok {
		repo := types.GitRepository{}
		if u, ok := um["url"].(string); ok {
			repo.URL = u
		}
		if b, ok := um["branch"].(string); ok && strings.TrimSpace(b) != "" {
			repo.Branch = handlers.StringPtr(b)
		}
		wf.UmbrellaRepo = &repo
	}
	if srs, ok := spec["supportingRepos"].([]interface{}); ok {
		wf.SupportingRepos = make([]types.GitRepository, 0, len(srs))
		for _, r := range srs {
			if rm, ok := r.(map[string]interface{}); ok {
				repo := types.GitRepository{}
				if u, ok := rm["url"].(string); ok {
					repo.URL = u
				}
				if b, ok := rm["branch"].(string); ok && strings.TrimSpace(b) != "" {
					repo.Branch = handlers.StringPtr(b)
				}
				wf.SupportingRepos = append(wf.SupportingRepos, repo)
			}
		}
	} else if repos, ok := spec["repositories"].([]interface{}); ok {
		// Backward compatibility: map legacy repositories -> umbrellaRepo (first) + supportingRepos (rest)
		for i, r := range repos {
			if rm, ok := r.(map[string]interface{}); ok {
				repo := types.GitRepository{}
				if u, ok := rm["url"].(string); ok {
					repo.URL = u
				}
				if b, ok := rm["branch"].(string); ok && strings.TrimSpace(b) != "" {
					repo.Branch = handlers.StringPtr(b)
				}
				if i == 0 {
					rcopy := repo
					wf.UmbrellaRepo = &rcopy
				} else {
					wf.SupportingRepos = append(wf.SupportingRepos, repo)
				}
			}
		}
	}

	// Parse jiraLinks
	if links, ok := spec["jiraLinks"].([]interface{}); ok {
		for _, it := range links {
			if m, ok := it.(map[string]interface{}); ok {
				path := fmt.Sprintf("%v", m["path"])
				jiraKey := fmt.Sprintf("%v", m["jiraKey"])
				if strings.TrimSpace(path) != "" && strings.TrimSpace(jiraKey) != "" {
					wf.JiraLinks = append(wf.JiraLinks, types.WorkflowJiraLink{Path: path, JiraKey: jiraKey})
				}
			}
		}
	}

	// Parse parentOutcome
	if po, ok := spec["parentOutcome"].(string); ok && strings.TrimSpace(po) != "" {
		wf.ParentOutcome = handlers.StringPtr(strings.TrimSpace(po))
	}

	return wf
}

// ExtractTitleFromContent attempts to extract a title from markdown content
// by looking for the first # heading
func ExtractTitleFromContent(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

// StripExecutionFlow removes the "Execution Flow" section from markdown content
// This section is typically found in spec.md and plan.md artifacts
func StripExecutionFlow(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inExecutionFlow := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is the start of the Execution Flow section
		if strings.HasPrefix(trimmed, "##") && strings.Contains(strings.ToLower(trimmed), "execution flow") {
			inExecutionFlow = true
			continue
		}

		// Check if we've hit the next section (another ## heading)
		if inExecutionFlow && strings.HasPrefix(trimmed, "##") {
			inExecutionFlow = false
		}

		// Add line if not in Execution Flow section
		if !inExecutionFlow {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// AttachFileToJiraIssue attaches a file to a Jira issue
func AttachFileToJiraIssue(ctx context.Context, jiraBase, issueKey, authHeader string, filename string, content []byte) error {
	endpoint := fmt.Sprintf("%s/rest/api/2/issue/%s/attachments", jiraBase, url.PathEscape(issueKey))

	// Create multipart form body
	body := &bytes.Buffer{}
	writer := NewMultipartWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(content); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", authHeader)
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("X-Atlassian-Token", "no-check")

	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jira API error: %s (body: %s)", resp.Status, string(respBody))
	}

	return nil
}

// MultipartWriter creates a new multipart writer with boundary.
type MultipartWriter struct {
	w        io.Writer
	boundary string
	closed   bool
}

func NewMultipartWriter(w io.Writer) *MultipartWriter {
	boundary := fmt.Sprintf("----WebKitFormBoundary%d", time.Now().UnixNano())
	return &MultipartWriter{w: w, boundary: boundary}
}

func (mw *MultipartWriter) FormDataContentType() string {
	return fmt.Sprintf("multipart/form-data; boundary=%s", mw.boundary)
}

func (mw *MultipartWriter) CreateFormFile(fieldname, filename string) (io.Writer, error) {
	h := fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\nContent-Type: application/octet-stream\r\n\r\n",
		mw.boundary, fieldname, filename)
	_, err := mw.w.Write([]byte(h))
	return mw.w, err
}

func (mw *MultipartWriter) Close() error {
	if mw.closed {
		return nil
	}
	mw.closed = true
	_, err := fmt.Fprintf(mw.w, "\r\n--%s--\r\n", mw.boundary)
	return err
}

// PublishWorkflowFileToJira creates or updates a Jira issue from a GitHub file and updates the RFEWorkflow CR with the linkage.
// POST /api/projects/:projectName/rfe-workflows/:id/jira { path, phase }
// Supports phase-specific logic: specify (Feature + rfe.md attachment), plan (Epic with artifact links), tasks (attach tasks.md)
func (h *Handler) PublishWorkflowFileToJira(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	var req struct {
		Path  string `json:"path" binding:"required"`
		Phase string `json:"phase"` // Optional: specify, plan, tasks
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	log.Printf("DEBUG JIRA: Received phase='%s' for path='%s'", req.Phase, req.Path)

	// Load runner secrets for Jira config
	_, reqDyn := h.GetK8sClientsForRequest(c)
	reqK8s, _ := h.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}

	// Determine configured secret name
	secretName := ""
	if reqDyn != nil {
		gvr := h.GetProjectSettingsResource()
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
	gvrWf := h.GetRFEWorkflowResource()
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}
	item, err := reqDyn.Resource(gvrWf).Namespace(project).Get(c.Request.Context(), id, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	wf := RFEFromUnstructured(item)
	if wf == nil || wf.UmbrellaRepo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workflow has no spec repo configured"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid spec repo URL", "details": err.Error()})
		return
	}
	// Use the generated feature branch - specs only exist on feature branch
	if wf.BranchName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "RFE workflow has no feature branch. Please seed the repository first."})
		return
	}

	branch := wf.BranchName

	content, err := git.ReadGitHubFile(c.Request.Context(), owner, repo, branch, req.Path, githubToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read file from GitHub", "details": err.Error()})
		return
	}

	// Extract title from markdown content
	title := ExtractTitleFromContent(string(content))
	if title == "" {
		title = wf.Title // Fallback to workflow title
	}

	// Build GitHub URL for the file
	githubURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, branch, req.Path)

	// Strip Execution Flow section from spec.md and plan.md
	var processedContent []byte
	if req.Phase == "specify" || req.Phase == "plan" || req.Phase == "tasks" {
		stripped := StripExecutionFlow(string(content))

		// For tasks phase (Epic), add reference to parent Feature if it exists
		featureReference := ""
		if req.Phase == "tasks" {
			for _, jl := range wf.JiraLinks {
				if strings.Contains(jl.Path, "plan.md") {
					featureReference = fmt.Sprintf("\n\n**Implements Feature:** %s\n\n---", jl.JiraKey)
					break
				}
			}
		}

		// Prepend GitHub URL reference at the top
		processedContent = []byte(fmt.Sprintf("**Source:** %s%s\n\n---\n\n%s", githubURL, featureReference, stripped))
	} else {
		// For other files, just prepend the GitHub URL
		processedContent = []byte(fmt.Sprintf("**Source:** %s\n\n---\n\n%s", githubURL, string(content)))
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

	// Determine issue type based on phase
	// specify -> Feature Request, plan -> Feature, tasks -> Epic
	issueType := "Feature" // default
	switch req.Phase {
	case "specify":
		issueType = "Feature Request"
	case "tasks":
		issueType = "Epic"
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
			"description": string(processedContent),
			"issuetype":   map[string]string{"name": issueType},
		}

		// TODO: decide correct hierarchy for parent/children jira objects
		// For Epic (tasks phase), the Feature reference is added to the description instead
		parentKey := ""
		if req.Phase != "tasks" {
			// For non-Epic phases: use parent Outcome if specified
			if wf.ParentOutcome != nil && *wf.ParentOutcome != "" {
				parentKey = *wf.ParentOutcome
			}
		}

		if parentKey != "" {
			fields["parent"] = map[string]string{"key": parentKey}
		}

		reqBody := map[string]interface{}{"fields": fields}
		payload, _ := json.Marshal(reqBody)
		log.Printf("DEBUG JIRA: Creating issue with type '%s', payload: %s", issueType, string(payload))
		httpReq, _ = http.NewRequestWithContext(c.Request.Context(), "POST", jiraEndpoint, bytes.NewReader(payload))
	} else {
		// UPDATE existing issue
		jiraEndpoint := fmt.Sprintf("%s/rest/api/2/issue/%s", jiraBase, url.PathEscape(existingKey))
		reqBody := map[string]interface{}{
			"fields": map[string]interface{}{
				"summary":     title,
				"description": string(processedContent),
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

	// Phase-specific attachment handling
	switch req.Phase {
	case "specify":
		// For specify phase (Feature Request): attach rfe.md if it exists
		rfeContent, err := git.ReadGitHubFile(c.Request.Context(), owner, repo, branch, "rfe.md", githubToken)
		if err == nil && len(rfeContent) > 0 {
			if attachErr := AttachFileToJiraIssue(c.Request.Context(), jiraBase, outKey, authHeader, "rfe.md", rfeContent); attachErr != nil {
				log.Printf("Warning: failed to attach rfe.md to %s: %v", outKey, attachErr)
			}
		}

	case "plan":
		// For plan phase (Feature): attach supporting documents if they exist
		// Extract directory path from plan.md path (e.g., "specs/001-feature/plan.md" -> "specs/001-feature")
		pathParts := strings.Split(req.Path, "/")
		var dirPath string
		if len(pathParts) > 1 {
			dirPath = strings.Join(pathParts[:len(pathParts)-1], "/")
		}

		// List of supporting documents to attach (if they exist)
		supportingDocs := []string{"data-model.md", "quickstart.md", "research.md"}

		for _, docName := range supportingDocs {
			docPath := docName
			if dirPath != "" {
				docPath = dirPath + "/" + docName
			}

			// Try to read the file - skip silently if it doesn't exist
			docContent, err := git.ReadGitHubFile(c.Request.Context(), owner, repo, branch, docPath, githubToken)
			if err == nil && len(docContent) > 0 {
				if attachErr := AttachFileToJiraIssue(c.Request.Context(), jiraBase, outKey, authHeader, docName, docContent); attachErr != nil {
					log.Printf("Warning: failed to attach %s to %s: %v", docName, outKey, attachErr)
				} else {
					log.Printf("Successfully attached %s to %s", docName, outKey)
				}
			}
		}

	case "tasks":
		// For tasks phase: Epic is created and linked to the Feature (plan.md)
		// No additional attachment handling needed - Epic is standalone with parent link
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
