package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Package-level variables for dependency injection (RFE-specific)
var (
	GetRFEWorkflowResource     func() schema.GroupVersionResource
	UpsertProjectRFEWorkflowCR func(dynamic.Interface, *types.RFEWorkflow) error
	PerformRepoSeeding         func(context.Context, *types.RFEWorkflow, string, string, string, string, string, string) error
	CheckRepoSeeding           func(context.Context, string, *string, string) (bool, map[string]interface{}, error)
	RfeFromUnstructured        func(*unstructured.Unstructured) *types.RFEWorkflow
)

// Type aliases for RFE workflow types
type RFEWorkflow = types.RFEWorkflow
type CreateRFEWorkflowRequest = types.CreateRFEWorkflowRequest
type GitRepository = types.GitRepository
type WorkflowJiraLink = types.WorkflowJiraLink

// rfeLinkSessionRequest holds the request body for linking a session to an RFE workflow
type rfeLinkSessionRequest struct {
	ExistingName string `json:"existingName"`
	Phase        string `json:"phase"`
}

// ListProjectRFEWorkflows lists all RFE workflows for a project
func ListProjectRFEWorkflows(c *gin.Context) {
	project := c.Param("projectName")
	var workflows []RFEWorkflow
	// Prefer CRD list with request-scoped client; fallback to file scan if unavailable or fails
	gvr := GetRFEWorkflowResource()
	_, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn != nil {
		if list, err := reqDyn.Resource(gvr).Namespace(project).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("project=%s", project)}); err == nil {
			for _, item := range list.Items {
				wf := RfeFromUnstructured(&item)
				if wf == nil {
					continue
				}
				workflows = append(workflows, *wf)
			}
		}
	}
	if workflows == nil {
		workflows = []RFEWorkflow{}
	}
	// Return slim summaries: omit artifacts/agentSessions/phaseResults/status/currentPhase
	summaries := make([]map[string]interface{}, 0, len(workflows))
	for _, w := range workflows {
		item := map[string]interface{}{
			"id":            w.ID,
			"title":         w.Title,
			"description":   w.Description,
			"project":       w.Project,
			"workspacePath": w.WorkspacePath,
			"createdAt":     w.CreatedAt,
			"updatedAt":     w.UpdatedAt,
		}
		if w.UmbrellaRepo != nil {
			u := map[string]interface{}{"url": w.UmbrellaRepo.URL}
			if w.UmbrellaRepo.Branch != nil {
				u["branch"] = *w.UmbrellaRepo.Branch
			}
			item["umbrellaRepo"] = u
		}
		if len(w.SupportingRepos) > 0 {
			repos := make([]map[string]interface{}, 0, len(w.SupportingRepos))
			for _, r := range w.SupportingRepos {
				rm := map[string]interface{}{"url": r.URL}
				if r.Branch != nil {
					rm["branch"] = *r.Branch
				}
				repos = append(repos, rm)
			}
			item["supportingRepos"] = repos
		}
		summaries = append(summaries, item)
	}
	c.JSON(http.StatusOK, gin.H{"workflows": summaries})
}

// CreateProjectRFEWorkflow creates a new RFE workflow for a project
func CreateProjectRFEWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	var req CreateRFEWorkflowRequest
	bodyBytes, _ := c.GetRawData()
	c.Request.Body = ioutil.NopCloser(strings.NewReader(string(bodyBytes)))
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed: " + err.Error()})
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	workflowID := fmt.Sprintf("rfe-%d", time.Now().Unix())
	workflow := &RFEWorkflow{
		ID:              workflowID,
		Title:           req.Title,
		Description:     req.Description,
		UmbrellaRepo:    &req.UmbrellaRepo,
		SupportingRepos: req.SupportingRepos,
		WorkspacePath:   req.WorkspacePath,
		Project:         project,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_, reqDyn := GetK8sClientsForRequest(c)
	if err := UpsertProjectRFEWorkflowCR(reqDyn, workflow); err != nil {
		log.Printf("⚠️ Failed to upsert RFEWorkflow CR: %v", err)
	}

	// Seeding (spec-kit + agents) is now handled by POST /seed endpoint after creation

	c.JSON(http.StatusCreated, workflow)
}

// SeedProjectRFEWorkflow seeds the umbrella repo with spec-kit and agents via direct git operations
func SeedProjectRFEWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	// Get the workflow
	gvr := GetRFEWorkflowResource()
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil || reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), id, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	wf := RfeFromUnstructured(item)
	if wf == nil || wf.UmbrellaRepo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No umbrella repo configured"})
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

	// Read request body for optional agent source
	type SeedRequest struct {
		AgentSourceURL    string `json:"agentSourceUrl,omitempty"`
		AgentSourceBranch string `json:"agentSourceBranch,omitempty"`
		AgentSourcePath   string `json:"agentSourcePath,omitempty"`
		SpecKitVersion    string `json:"specKitVersion,omitempty"`
		SpecKitTemplate   string `json:"specKitTemplate,omitempty"`
	}
	var req SeedRequest
	_ = c.ShouldBindJSON(&req)

	// Defaults
	agentURL := req.AgentSourceURL
	if agentURL == "" {
		agentURL = "https://github.com/ambient-code/vTeam.git"
	}
	agentBranch := req.AgentSourceBranch
	if agentBranch == "" {
		agentBranch = "main"
	}
	agentPath := req.AgentSourcePath
	if agentPath == "" {
		agentPath = "agents"
	}
	specKitVersion := req.SpecKitVersion
	if specKitVersion == "" {
		specKitVersion = "v0.0.55"
	}
	specKitTemplate := req.SpecKitTemplate
	if specKitTemplate == "" {
		specKitTemplate = "spec-kit-template-claude-sh"
	}

	// Perform seeding operations
	seedErr := PerformRepoSeeding(c.Request.Context(), wf, githubToken, agentURL, agentBranch, agentPath, specKitVersion, specKitTemplate)

	if seedErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": seedErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "completed",
		"message": "Repository seeded successfully",
	})
}

// CheckProjectRFEWorkflowSeeding checks if the umbrella repo is seeded by querying GitHub API
func CheckProjectRFEWorkflowSeeding(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	// Get the workflow
	gvr := GetRFEWorkflowResource()
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil || reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), id, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	wf := RfeFromUnstructured(item)
	if wf == nil || wf.UmbrellaRepo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No umbrella repo configured"})
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

	// Check if repo is seeded
	isSeeded, details, err := CheckRepoSeeding(c.Request.Context(), wf.UmbrellaRepo.URL, wf.UmbrellaRepo.Branch, githubToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"isSeeded": isSeeded,
		"details":  details,
	})
}

// GetProjectRFEWorkflow retrieves a specific RFE workflow by ID
func GetProjectRFEWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")
	// Try CRD with request-scoped client first
	gvr := GetRFEWorkflowResource()
	_, reqDyn := GetK8sClientsForRequest(c)
	var wf *RFEWorkflow
	var err error
	if reqDyn != nil {
		if item, gerr := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), id, v1.GetOptions{}); gerr == nil {
			wf = RfeFromUnstructured(item)
			err = nil
		} else {
			err = gerr
		}
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	// Return slim object without artifacts/agentSessions/phaseResults/status/currentPhase
	resp := map[string]interface{}{
		"id":            wf.ID,
		"title":         wf.Title,
		"description":   wf.Description,
		"project":       wf.Project,
		"workspacePath": wf.WorkspacePath,
		"createdAt":     wf.CreatedAt,
		"updatedAt":     wf.UpdatedAt,
	}
	if wf.ParentOutcome != nil {
		resp["parentOutcome"] = *wf.ParentOutcome
	}
	if len(wf.JiraLinks) > 0 {
		links := make([]map[string]interface{}, 0, len(wf.JiraLinks))
		for _, l := range wf.JiraLinks {
			links = append(links, map[string]interface{}{"path": l.Path, "jiraKey": l.JiraKey})
		}
		resp["jiraLinks"] = links
	}
	if wf.UmbrellaRepo != nil {
		u := map[string]interface{}{"url": wf.UmbrellaRepo.URL}
		if wf.UmbrellaRepo.Branch != nil {
			u["branch"] = *wf.UmbrellaRepo.Branch
		}
		resp["umbrellaRepo"] = u
	}
	if len(wf.SupportingRepos) > 0 {
		repos := make([]map[string]interface{}, 0, len(wf.SupportingRepos))
		for _, r := range wf.SupportingRepos {
			rm := map[string]interface{}{"url": r.URL}
			if r.Branch != nil {
				rm["branch"] = *r.Branch
			}
			repos = append(repos, rm)
		}
		resp["supportingRepos"] = repos
	}
	c.JSON(http.StatusOK, resp)
}

// GetProjectRFEWorkflowSummary computes derived phase/status and progress based on workspace files and linked sessions
// GET /api/projects/:projectName/rfe-workflows/:id/summary
func GetProjectRFEWorkflowSummary(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	// Determine workspace and expected files
	// workspace content removed
	specsItems := []contentListItem{}

	hasSpec := false
	hasPlan := false
	hasTasks := false

	// helper to scan a list for target filenames
	scanFor := func(items []contentListItem) (bool, bool, bool) {
		s, p, t := false, false, false
		for _, it := range items {
			if it.IsDir {
				continue
			}
			switch strings.ToLower(it.Name) {
			case "spec.md":
				s = true
			case "plan.md":
				p = true
			case "tasks.md":
				t = true
			}
		}
		return s, p, t
	}

	// First check directly under specs/
	if len(specsItems) > 0 {
		s, p, t := scanFor(specsItems)
		hasSpec, hasPlan, hasTasks = s, p, t
		// If not found, check first subfolder under specs/
		if !(hasSpec || hasPlan || hasTasks) {
			for _, it := range specsItems {
				if it.IsDir {
					subItems := []contentListItem{}
					s2, p2, t2 := scanFor(subItems)
					hasSpec, hasPlan, hasTasks = s2, p2, t2
					break
				}
			}
		}
	}

	// Sessions: find linked sessions and compute running/failed flags
	gvr := GetAgenticSessionV1Alpha1Resource()
	_, reqDyn := GetK8sClientsForRequest(c)
	anyRunning := false
	anyFailed := false
	if reqDyn != nil {
		selector := fmt.Sprintf("rfe-workflow=%s,project=%s", id, project)
		if list, err := reqDyn.Resource(gvr).Namespace(project).List(context.TODO(), v1.ListOptions{LabelSelector: selector}); err == nil {
			for _, item := range list.Items {
				status, _ := item.Object["status"].(map[string]interface{})
				phaseStr := strings.ToLower(fmt.Sprintf("%v", status["phase"]))
				if phaseStr == "running" || phaseStr == "creating" || phaseStr == "pending" {
					anyRunning = true
				}
				if phaseStr == "failed" || phaseStr == "error" {
					anyFailed = true
				}
			}
		}
	}

	// Derive phase and status
	var phase string
	switch {
	case !hasSpec && !hasPlan && !hasTasks:
		phase = "pre"
	case !hasSpec:
		phase = "specify"
	case !hasPlan:
		phase = "plan"
	case !hasTasks:
		phase = "tasks"
	default:
		phase = "completed"
	}

	status := "not started"
	if anyRunning {
		status = "running"
	} else if hasSpec || hasPlan || hasTasks {
		status = "in progress"
	}
	if hasSpec && hasPlan && hasTasks && !anyRunning {
		status = "completed"
	}
	if anyFailed && status != "running" {
		status = "attention"
	}

	progress := float64(0)
	done := 0
	if hasSpec {
		done++
	}
	if hasPlan {
		done++
	}
	if hasTasks {
		done++
	}
	progress = float64(done) / 3.0 * 100.0

	c.JSON(http.StatusOK, gin.H{
		"phase":    phase,
		"status":   status,
		"progress": progress,
		"files": gin.H{
			"spec":  hasSpec,
			"plan":  hasPlan,
			"tasks": hasTasks,
		},
	})
}

// DeleteProjectRFEWorkflow deletes an RFE workflow
func DeleteProjectRFEWorkflow(c *gin.Context) {
	id := c.Param("id")
	// Delete CR
	gvr := GetRFEWorkflowResource()
	_, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn != nil {
		_ = reqDyn.Resource(gvr).Namespace(c.Param("projectName")).Delete(context.TODO(), id, v1.DeleteOptions{})
	}
	c.JSON(http.StatusOK, gin.H{"message": "Workflow deleted successfully"})
}

// ListProjectRFEWorkflowSessions lists sessions linked to a project-scoped RFE workflow by label selector
func ListProjectRFEWorkflowSessions(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")
	gvr := GetAgenticSessionV1Alpha1Resource()
	selector := fmt.Sprintf("rfe-workflow=%s,project=%s", id, project)
	_, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}
	list, err := reqDyn.Resource(gvr).Namespace(project).List(context.TODO(), v1.ListOptions{LabelSelector: selector})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions", "details": err.Error()})
		return
	}

	// Return full session objects for UI
	sessions := make([]map[string]interface{}, 0, len(list.Items))
	for _, item := range list.Items {
		sessions = append(sessions, item.Object)
	}
	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// AddProjectRFEWorkflowSession adds/links an existing session to an RFE by applying labels
func AddProjectRFEWorkflowSession(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")
	var req rfeLinkSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	if req.ExistingName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "existingName is required for linking in this version"})
		return
	}
	gvr := GetAgenticSessionV1Alpha1Resource()
	_, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), req.ExistingName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch session", "details": err.Error()})
		return
	}
	meta, _ := obj.Object["metadata"].(map[string]interface{})
	labels, _ := meta["labels"].(map[string]interface{})
	if labels == nil {
		labels = map[string]interface{}{}
		meta["labels"] = labels
	}
	labels["project"] = project
	labels["rfe-workflow"] = id
	if req.Phase != "" {
		labels["rfe-phase"] = req.Phase
	}
	// Update the resource
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session labels", "details": err.Error()})
		return
	}
	_ = updated
	c.JSON(http.StatusOK, gin.H{"message": "Session linked to RFE", "session": req.ExistingName})
}

// RemoveProjectRFEWorkflowSession removes/unlinks a session from an RFE by clearing linkage labels (non-destructive)
func RemoveProjectRFEWorkflowSession(c *gin.Context) {
	project := c.Param("projectName")
	_ = project // currently unused but kept for parity/logging if needed
	id := c.Param("id")
	sessionName := c.Param("sessionName")
	gvr := GetAgenticSessionV1Alpha1Resource()
	_, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch session", "details": err.Error()})
		return
	}
	meta, _ := obj.Object["metadata"].(map[string]interface{})
	labels, _ := meta["labels"].(map[string]interface{})
	if labels != nil {
		delete(labels, "rfe-workflow")
		delete(labels, "rfe-phase")
	}
	if _, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), obj, v1.UpdateOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session labels", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Session unlinked from RFE", "session": sessionName, "rfe": id})
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
