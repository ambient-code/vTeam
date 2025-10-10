package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// feature flags and small helpers
var (
	boolPtr   = func(b bool) *bool { return &b }
	stringPtr = func(s string) *string { return &s }
)

type contentListItem struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
}

// getK8sClientsForRequest returns K8s typed and dynamic clients using the caller's token when provided.
// It supports both Authorization: Bearer and X-Forwarded-Access-Token and NEVER falls back to the backend service account.
// Returns nil, nil if no valid user token is provided - all API operations require user authentication.
func getK8sClientsForRequest(c *gin.Context) (*kubernetes.Clientset, dynamic.Interface) {
	// Prefer Authorization header (Bearer <token>)
	rawAuth := c.GetHeader("Authorization")
	rawFwd := c.GetHeader("X-Forwarded-Access-Token")
	tokenSource := "none"
	token := rawAuth

	if token != "" {
		tokenSource = "authorization"
		parts := strings.SplitN(token, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			token = strings.TrimSpace(parts[1])
		} else {
			token = strings.TrimSpace(token)
		}
	}
	// Fallback to X-Forwarded-Access-Token
	if token == "" {
		if rawFwd != "" {
			tokenSource = "x-forwarded-access-token"
		}
		token = rawFwd
	}

	// Debug: basic auth header state (do not log token)
	hasAuthHeader := strings.TrimSpace(rawAuth) != ""
	hasFwdToken := strings.TrimSpace(rawFwd) != ""

	if token != "" && baseKubeConfig != nil {
		cfg := *baseKubeConfig
		cfg.BearerToken = token
		// Ensure we do NOT fall back to the in-cluster SA token or other auth providers
		cfg.BearerTokenFile = ""
		cfg.AuthProvider = nil
		cfg.ExecProvider = nil
		cfg.Username = ""
		cfg.Password = ""

		kc, err1 := kubernetes.NewForConfig(&cfg)
		dc, err2 := dynamic.NewForConfig(&cfg)

		if err1 == nil && err2 == nil {

			// Best-effort update last-used for service account tokens
			updateAccessKeyLastUsedAnnotation(c)
			return kc, dc
		}
		// Token provided but client build failed – treat as invalid token
		log.Printf("Failed to build user-scoped k8s clients (source=%s tokenLen=%d) typedErr=%v dynamicErr=%v for %s", tokenSource, len(token), err1, err2, c.FullPath())
		return nil, nil
	} else {
		// No token provided
		log.Printf("No user token found for %s (hasAuthHeader=%t hasFwdToken=%t)", c.FullPath(), hasAuthHeader, hasFwdToken)
		return nil, nil
	}
}

// updateAccessKeyLastUsedAnnotation attempts to update the ServiceAccount's last-used annotation
// when the incoming token is a ServiceAccount JWT. Uses the backend service account client strictly
// for this telemetry update and only for SAs labeled app=ambient-access-key. Best-effort; errors ignored.
func updateAccessKeyLastUsedAnnotation(c *gin.Context) {
	// Parse Authorization header
	rawAuth := c.GetHeader("Authorization")
	parts := strings.SplitN(rawAuth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return
	}

	// Decode JWT payload (second segment)
	segs := strings.Split(token, ".")
	if len(segs) < 2 {
		return
	}
	payloadB64 := segs[1]
	// JWT uses base64url without padding; add padding if necessary
	if m := len(payloadB64) % 4; m != 0 {
		payloadB64 += strings.Repeat("=", 4-m)
	}
	data, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}
	// Expect sub like: system:serviceaccount:<namespace>:<sa-name>
	sub, _ := payload["sub"].(string)
	const prefix = "system:serviceaccount:"
	if !strings.HasPrefix(sub, prefix) {
		return
	}
	rest := strings.TrimPrefix(sub, prefix)
	parts2 := strings.SplitN(rest, ":", 2)
	if len(parts2) != 2 {
		return
	}
	ns := parts2[0]
	saName := parts2[1]

	// Backend client must exist
	if k8sClient == nil {
		return
	}

	// Ensure the SA is an Ambient access key (label check) before writing
	saObj, err := k8sClient.CoreV1().ServiceAccounts(ns).Get(c.Request.Context(), saName, v1.GetOptions{})
	if err != nil {
		return
	}
	if saObj.Labels == nil || saObj.Labels["app"] != "ambient-access-key" {
		return
	}

	// Patch the annotation
	now := time.Now().Format(time.RFC3339)
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				"ambient-code.io/last-used-at": now,
			},
		},
	}
	b, err := json.Marshal(patch)
	if err != nil {
		return
	}
	_, err = k8sClient.CoreV1().ServiceAccounts(ns).Patch(c.Request.Context(), saName, types.MergePatchType, b, v1.PatchOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to update last-used annotation for SA %s/%s: %v", ns, saName, err)
	}
}

// extractServiceAccountFromAuth extracts namespace and ServiceAccount name from the Authorization Bearer JWT 'sub' claim
// Returns (namespace, saName, true) when a SA subject is present, otherwise ("","",false)
func extractServiceAccountFromAuth(c *gin.Context) (string, string, bool) {
	rawAuth := c.GetHeader("Authorization")
	parts := strings.SplitN(rawAuth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", "", false
	}
	segs := strings.Split(token, ".")
	if len(segs) < 2 {
		return "", "", false
	}
	payloadB64 := segs[1]
	if m := len(payloadB64) % 4; m != 0 {
		payloadB64 += strings.Repeat("=", 4-m)
	}
	data, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", "", false
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", "", false
	}
	sub, _ := payload["sub"].(string)
	const prefix = "system:serviceaccount:"
	if !strings.HasPrefix(sub, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(sub, prefix)
	parts2 := strings.SplitN(rest, ":", 2)
	if len(parts2) != 2 {
		return "", "", false
	}
	return parts2[0], parts2[1], true
}

// Middleware for project context validation
func validateProjectContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow token via query parameter for websocket/agent callers
		if c.GetHeader("Authorization") == "" && c.GetHeader("X-Forwarded-Access-Token") == "" {
			if qp := strings.TrimSpace(c.Query("token")); qp != "" {
				c.Request.Header.Set("Authorization", "Bearer "+qp)
			}
		}
		// Require user/API key token; do not fall back to service account
		if c.GetHeader("Authorization") == "" && c.GetHeader("X-Forwarded-Access-Token") == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
			c.Abort()
			return
		}
		reqK8s, _ := getK8sClientsForRequest(c)
		if reqK8s == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
			c.Abort()
			return
		}
		// Prefer project from route param; fallback to header for backward compatibility
		projectHeader := c.Param("projectName")
		if projectHeader == "" {
			projectHeader = c.GetHeader("X-OpenShift-Project")
		}
		if projectHeader == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Project is required in path /api/projects/:projectName or X-OpenShift-Project header"})
			c.Abort()
			return
		}

		// Ensure the caller has at least list permission on agenticsessions in the namespace
		ssar := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authv1.ResourceAttributes{
					Group:     "vteam.ambient-code",
					Resource:  "agenticsessions",
					Verb:      "list",
					Namespace: projectHeader,
				},
			},
		}
		res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(c.Request.Context(), ssar, v1.CreateOptions{})
		if err != nil {
			log.Printf("validateProjectContext: SSAR failed for %s: %v", projectHeader, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to perform access review"})
			c.Abort()
			return
		}
		if !res.Status.Allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to access project"})
			c.Abort()
			return
		}

		// Store project in context for handlers
		c.Set("project", projectHeader)
		c.Next()
	}
}

// accessCheck verifies if the caller has write access to ProjectSettings in the project namespace
// It performs a Kubernetes SelfSubjectAccessReview using the caller token (user or API key).
func accessCheck(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := getK8sClientsForRequest(c)

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


// handleJiraWebhook removed; use standard session creation endpoint instead

// (removed) Metrics handler placeholder; real implementation lives in observability.go

// ========================= Project-scoped RFE Handlers =========================

// rfeFromUnstructured converts an unstructured RFEWorkflow CR into our RFEWorkflow struct
func rfeFromUnstructured(item *unstructured.Unstructured) *RFEWorkflow {
	if item == nil {
		return nil
	}
	obj := item.Object
	spec, _ := obj["spec"].(map[string]interface{})

	created := ""
	if item.GetCreationTimestamp().Time != (time.Time{}) {
		created = item.GetCreationTimestamp().Time.UTC().Format(time.RFC3339)
	}
	wf := &RFEWorkflow{
		ID:            item.GetName(),
		Title:         fmt.Sprintf("%v", spec["title"]),
		Description:   fmt.Sprintf("%v", spec["description"]),
		Project:       item.GetNamespace(),
		WorkspacePath: fmt.Sprintf("%v", spec["workspacePath"]),
		CreatedAt:     created,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	// Parse umbrellaRepo/supportingRepos when present; fallback to repositories
	if um, ok := spec["umbrellaRepo"].(map[string]interface{}); ok {
		repo := GitRepository{}
		if u, ok := um["url"].(string); ok {
			repo.URL = u
		}
		if b, ok := um["branch"].(string); ok && strings.TrimSpace(b) != "" {
			repo.Branch = stringPtr(b)
		}
		wf.UmbrellaRepo = &repo
	}
	if srs, ok := spec["supportingRepos"].([]interface{}); ok {
		wf.SupportingRepos = make([]GitRepository, 0, len(srs))
		for _, r := range srs {
			if rm, ok := r.(map[string]interface{}); ok {
				repo := GitRepository{}
				if u, ok := rm["url"].(string); ok {
					repo.URL = u
				}
				if b, ok := rm["branch"].(string); ok && strings.TrimSpace(b) != "" {
					repo.Branch = stringPtr(b)
				}
				wf.SupportingRepos = append(wf.SupportingRepos, repo)
			}
		}
	} else if repos, ok := spec["repositories"].([]interface{}); ok {
		// Backward compatibility: map legacy repositories -> umbrellaRepo (first) + supportingRepos (rest)
		for i, r := range repos {
			if rm, ok := r.(map[string]interface{}); ok {
				repo := GitRepository{}
				if u, ok := rm["url"].(string); ok {
					repo.URL = u
				}
				if b, ok := rm["branch"].(string); ok && strings.TrimSpace(b) != "" {
					repo.Branch = stringPtr(b)
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
					wf.JiraLinks = append(wf.JiraLinks, WorkflowJiraLink{Path: path, JiraKey: jiraKey})
				}
			}
		}
	}

	// Parse parentOutcome
	if po, ok := spec["parentOutcome"].(string); ok && strings.TrimSpace(po) != "" {
		wf.ParentOutcome = stringPtr(strings.TrimSpace(po))
	}

	return wf
}

func listProjectRFEWorkflows(c *gin.Context) {
	project := c.Param("projectName")
	var workflows []RFEWorkflow
	// Prefer CRD list with request-scoped client; fallback to file scan if unavailable or fails
	gvr := getRFEWorkflowResource()
	_, reqDyn := getK8sClientsForRequest(c)
	if reqDyn != nil {
		if list, err := reqDyn.Resource(gvr).Namespace(project).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("project=%s", project)}); err == nil {
			for _, item := range list.Items {
				wf := rfeFromUnstructured(&item)
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

func createProjectRFEWorkflow(c *gin.Context) {
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
	_, reqDyn := getK8sClientsForRequest(c)
	if err := upsertProjectRFEWorkflowCR(reqDyn, workflow); err != nil {
		log.Printf("⚠️ Failed to upsert RFEWorkflow CR: %v", err)
	}

	// Seeding (spec-kit + agents) is now handled by POST /seed endpoint after creation

	c.JSON(http.StatusCreated, workflow)
}

// seedProjectRFEWorkflow seeds the umbrella repo with spec-kit and agents via direct git operations
func seedProjectRFEWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	// Get the workflow
	gvr := getRFEWorkflowResource()
	reqK8s, reqDyn := getK8sClientsForRequest(c)
	if reqDyn == nil || reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), id, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	wf := rfeFromUnstructured(item)
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

	githubToken, err := getGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userIDStr)
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
		specKitTemplate = "spec-kit-template-cursor-sh"
	}

	// Perform seeding operations
	seedErr := performRepoSeeding(c.Request.Context(), wf, githubToken, agentURL, agentBranch, agentPath, specKitVersion, specKitTemplate)

	if seedErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": seedErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "completed",
		"message": "Repository seeded successfully",
	})
}

// checkProjectRFEWorkflowSeeding checks if the umbrella repo is seeded by querying GitHub API
func checkProjectRFEWorkflowSeeding(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	// Get the workflow
	gvr := getRFEWorkflowResource()
	reqK8s, reqDyn := getK8sClientsForRequest(c)
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), id, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	wf := rfeFromUnstructured(item)
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

	githubToken, err := getGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if repo is seeded
	isSeeded, details, err := checkRepoSeeding(c.Request.Context(), wf.UmbrellaRepo.URL, wf.UmbrellaRepo.Branch, githubToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"isSeeded": isSeeded,
		"details":  details,
	})
}

func getProjectRFEWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")
	// Try CRD with request-scoped client first
	gvr := getRFEWorkflowResource()
	_, reqDyn := getK8sClientsForRequest(c)
	var wf *RFEWorkflow
	var err error
	if reqDyn != nil {
		if item, gerr := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), id, v1.GetOptions{}); gerr == nil {
			wf = rfeFromUnstructured(item)
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

// GET /api/projects/:projectName/rfe-workflows/:id/summary
// Computes derived phase/status and progress based on workspace files and linked sessions
func getProjectRFEWorkflowSummary(c *gin.Context) {
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
	gvr := getAgenticSessionV1Alpha1Resource()
	_, reqDyn := getK8sClientsForRequest(c)
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

func deleteProjectRFEWorkflow(c *gin.Context) {
	id := c.Param("id")
	// Delete CR
	gvr := getRFEWorkflowResource()
	_, reqDyn := getK8sClientsForRequest(c)
	if reqDyn != nil {
		_ = reqDyn.Resource(gvr).Namespace(c.Param("projectName")).Delete(context.TODO(), id, v1.DeleteOptions{})
	}
	c.JSON(http.StatusOK, gin.H{"message": "Workflow deleted successfully"})
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

// publishWorkflowFileToJira is now in jira.go

// List sessions linked to a project-scoped RFE workflow by label selector
func listProjectRFEWorkflowSessions(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")
	gvr := getAgenticSessionV1Alpha1Resource()
	selector := fmt.Sprintf("rfe-workflow=%s,project=%s", id, project)
	_, reqDyn := getK8sClientsForRequest(c)
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

type rfeLinkSessionRequest struct {
	ExistingName string `json:"existingName"`
	Phase        string `json:"phase"`
}

// Add/link an existing session to an RFE by applying labels
func addProjectRFEWorkflowSession(c *gin.Context) {
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
	gvr := getAgenticSessionV1Alpha1Resource()
	_, reqDyn := getK8sClientsForRequest(c)
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

// Remove/unlink a session from an RFE by clearing linkage labels (non-destructive)
func removeProjectRFEWorkflowSession(c *gin.Context) {
	project := c.Param("projectName")
	_ = project // currently unused but kept for parity/logging if needed
	id := c.Param("id")
	sessionName := c.Param("sessionName")
	gvr := getAgenticSessionV1Alpha1Resource()
	_, reqDyn := getK8sClientsForRequest(c)
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

// GET /api/projects/:projectName/rfe-workflows/:id/jira?path=...
// Proxies Jira issue fetch for a linked path
func getWorkflowJira(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")
	reqPath := strings.TrimSpace(c.Query("path"))
	if reqPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}
	_, reqDyn := getK8sClientsForRequest(c)
	reqK8s, _ := getK8sClientsForRequest(c)
	if reqDyn == nil || reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}
	// Load workflow to find key
	gvrWf := getRFEWorkflowResource()
	item, err := reqDyn.Resource(gvrWf).Namespace(project).Get(c.Request.Context(), id, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	wf := rfeFromUnstructured(item)
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
	if obj, err := reqDyn.Resource(getProjectSettingsResource()).Namespace(project).Get(c.Request.Context(), "projectsettings", v1.GetOptions{}); err == nil {
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

