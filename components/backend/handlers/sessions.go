// Package handlers implements HTTP request handlers for the vTeam backend API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"ambient-code-backend/git"
	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	authnv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Package-level variables for session handlers (set from main package)
var (
	GetAgenticSessionV1Alpha1Resource func() schema.GroupVersionResource
	DynamicClient                     dynamic.Interface
	GetGitHubToken                    func(context.Context, *kubernetes.Clientset, dynamic.Interface, string, string) (string, error)
	DeriveRepoFolderFromURL           func(string) string
	SendMessageToSession              func(string, string, map[string]interface{})
)

const runnerTokenRefreshedAtAnnotation = "ambient-code.io/token-refreshed-at"

// parseSpec parses AgenticSessionSpec with v1alpha1 fields
func parseSpec(spec map[string]interface{}) types.AgenticSessionSpec {
	result := types.AgenticSessionSpec{}

	if prompt, ok := spec["initialPrompt"].(string); ok {
		result.InitialPrompt = prompt
	}

	if interactive, ok := spec["interactive"].(bool); ok {
		result.Interactive = interactive
	}

	if displayName, ok := spec["displayName"].(string); ok {
		result.DisplayName = displayName
	}

	if project, ok := spec["project"].(string); ok {
		result.Project = project
	}

	if timeout, ok := spec["timeout"].(float64); ok {
		result.Timeout = int(timeout)
	}

	if llmSettings, ok := spec["llmSettings"].(map[string]interface{}); ok {
		if model, ok := llmSettings["model"].(string); ok {
			result.LLMSettings.Model = model
		}
		if temperature, ok := llmSettings["temperature"].(float64); ok {
			result.LLMSettings.Temperature = temperature
		}
		if maxTokens, ok := llmSettings["maxTokens"].(float64); ok {
			result.LLMSettings.MaxTokens = int(maxTokens)
		}
	}

	// environmentVariables passthrough
	if env, ok := spec["environmentVariables"].(map[string]interface{}); ok {
		resultEnv := make(map[string]string, len(env))
		for k, v := range env {
			if s, ok := v.(string); ok {
				resultEnv[k] = s
			}
		}
		if len(resultEnv) > 0 {
			result.EnvironmentVariables = resultEnv
		}
	}

	if userContext, ok := spec["userContext"].(map[string]interface{}); ok {
		uc := &types.UserContext{}
		if userID, ok := userContext["userId"].(string); ok {
			uc.UserID = userID
		}
		if displayName, ok := userContext["displayName"].(string); ok {
			uc.DisplayName = displayName
		}
		if groups, ok := userContext["groups"].([]interface{}); ok {
			for _, group := range groups {
				if groupStr, ok := group.(string); ok {
					uc.Groups = append(uc.Groups, groupStr)
				}
			}
		}
		result.UserContext = uc
	}

	// Multi-repo parsing (simplified format)
	if arr, ok := spec["repos"].([]interface{}); ok {
		repos := make([]types.SimpleRepo, 0, len(arr))
		for _, it := range arr {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			r := types.SimpleRepo{}
			if url, ok := m["url"].(string); ok {
				r.URL = url
			}
			if branch, ok := m["branch"].(string); ok && strings.TrimSpace(branch) != "" {
				r.Branch = types.StringPtr(branch)
			}
			if strings.TrimSpace(r.URL) != "" {
				repos = append(repos, r)
			}
		}
		result.Repos = repos
	}

	// Parse activeWorkflow
	if workflow, ok := spec["activeWorkflow"].(map[string]interface{}); ok {
		ws := &types.WorkflowSelection{}
		if gitURL, ok := workflow["gitUrl"].(string); ok {
			ws.GitURL = gitURL
		}
		if branch, ok := workflow["branch"].(string); ok {
			ws.Branch = branch
		}
		if path, ok := workflow["path"].(string); ok {
			ws.Path = path
		}
		result.ActiveWorkflow = ws
	}

	return result
}

// parseStatus parses AgenticSessionStatus with detailed reconciliation fields
func parseStatus(status map[string]interface{}) *types.AgenticSessionStatus {
	if status == nil {
		return nil
	}

	result := &types.AgenticSessionStatus{}

	if og, ok := status["observedGeneration"]; ok {
		switch v := og.(type) {
		case int64:
			result.ObservedGeneration = v
		case int32:
			result.ObservedGeneration = int64(v)
		case float64:
			result.ObservedGeneration = int64(v)
		case json.Number:
			if parsed, err := v.Int64(); err == nil {
				result.ObservedGeneration = parsed
			}
		}
	}

	if phase, ok := status["phase"].(string); ok {
		result.Phase = phase
	}

	if startTime, ok := status["startTime"].(string); ok && strings.TrimSpace(startTime) != "" {
		result.StartTime = types.StringPtr(startTime)
	}

	if completionTime, ok := status["completionTime"].(string); ok && strings.TrimSpace(completionTime) != "" {
		result.CompletionTime = types.StringPtr(completionTime)
	}

	// jobName and runnerPodName removed - they go stale on restarts
	// Use GET /k8s-resources endpoint for live job/pod information

	if sdkSessionID, ok := status["sdkSessionId"].(string); ok {
		result.SDKSessionID = sdkSessionID
	}

	if restarts, ok := status["sdkRestartCount"]; ok {
		switch v := restarts.(type) {
		case int64:
			result.SDKRestartCount = int(v)
		case int32:
			result.SDKRestartCount = int(v)
		case float64:
			result.SDKRestartCount = int(v)
		case json.Number:
			if parsed, err := v.Int64(); err == nil {
				result.SDKRestartCount = int(parsed)
			}
		}
	}

	if repos, ok := status["reconciledRepos"].([]interface{}); ok && len(repos) > 0 {
		result.ReconciledRepos = make([]types.ReconciledRepo, 0, len(repos))
		for _, entry := range repos {
			m, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			repo := types.ReconciledRepo{}
			if url, ok := m["url"].(string); ok {
				repo.URL = url
			}
			if branch, ok := m["branch"].(string); ok {
				repo.Branch = branch
			}
			if name, ok := m["name"].(string); ok {
				repo.Name = name
			}
			if statusVal, ok := m["status"].(string); ok {
				repo.Status = statusVal
			}
			if clonedAt, ok := m["clonedAt"].(string); ok && strings.TrimSpace(clonedAt) != "" {
				repo.ClonedAt = types.StringPtr(clonedAt)
			}
			result.ReconciledRepos = append(result.ReconciledRepos, repo)
		}
	}

	if wf, ok := status["reconciledWorkflow"].(map[string]interface{}); ok && len(wf) > 0 {
		reconciled := &types.ReconciledWorkflow{}
		if gitURL, ok := wf["gitUrl"].(string); ok {
			reconciled.GitURL = gitURL
		}
		if branch, ok := wf["branch"].(string); ok {
			reconciled.Branch = branch
		}
		if state, ok := wf["status"].(string); ok {
			reconciled.Status = state
		}
		if appliedAt, ok := wf["appliedAt"].(string); ok && strings.TrimSpace(appliedAt) != "" {
			reconciled.AppliedAt = types.StringPtr(appliedAt)
		}
		result.ReconciledWorkflow = reconciled
	}

	if conds, ok := status["conditions"].([]interface{}); ok && len(conds) > 0 {
		result.Conditions = make([]types.Condition, 0, len(conds))
		for _, entry := range conds {
			m, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			cond := types.Condition{}
			if t, ok := m["type"].(string); ok {
				cond.Type = t
			}
			if s, ok := m["status"].(string); ok {
				cond.Status = s
			}
			if reason, ok := m["reason"].(string); ok {
				cond.Reason = reason
			}
			if message, ok := m["message"].(string); ok {
				cond.Message = message
			}
			if ts, ok := m["lastTransitionTime"].(string); ok {
				cond.LastTransitionTime = ts
			}
			if og, ok := m["observedGeneration"]; ok {
				switch v := og.(type) {
				case int64:
					cond.ObservedGeneration = v
				case int32:
					cond.ObservedGeneration = int64(v)
				case float64:
					cond.ObservedGeneration = int64(v)
				case json.Number:
					if parsed, err := v.Int64(); err == nil {
						cond.ObservedGeneration = parsed
					}
				}
			}
			result.Conditions = append(result.Conditions, cond)
		}
	}

	return result
}

// V2 API Handlers - Multi-tenant session management

func ListSessions(c *gin.Context) {
	project := c.GetString("project")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	_ = reqK8s
	gvr := GetAgenticSessionV1Alpha1Resource()

	list, err := reqDyn.Resource(gvr).Namespace(project).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list agentic sessions in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list agentic sessions"})
		return
	}

	var sessions []types.AgenticSession
	for _, item := range list.Items {
		session := types.AgenticSession{
			APIVersion: item.GetAPIVersion(),
			Kind:       item.GetKind(),
			Metadata:   item.Object["metadata"].(map[string]interface{}),
		}

		if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
			session.Spec = parseSpec(spec)
		}

		if status, ok := item.Object["status"].(map[string]interface{}); ok {
			session.Status = parseStatus(status)
		}

		sessions = append(sessions, session)
	}

	c.JSON(http.StatusOK, gin.H{"items": sessions})
}

func CreateSession(c *gin.Context) {
	project := c.GetString("project")
	// Get user-scoped clients for creating the AgenticSession (enforces user RBAC)
	_, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		return
	}
	var req types.CreateAgenticSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validation for multi-repo can be added here if needed

	// Set defaults for LLM settings if not provided
	llmSettings := types.LLMSettings{
		Model:       "sonnet",
		Temperature: 0.7,
		MaxTokens:   4000,
	}
	if req.LLMSettings != nil {
		if req.LLMSettings.Model != "" {
			llmSettings.Model = req.LLMSettings.Model
		}
		if req.LLMSettings.Temperature != 0 {
			llmSettings.Temperature = req.LLMSettings.Temperature
		}
		if req.LLMSettings.MaxTokens != 0 {
			llmSettings.MaxTokens = req.LLMSettings.MaxTokens
		}
	}

	timeout := 300
	if req.Timeout != nil {
		timeout = *req.Timeout
	}

	// Generate unique name
	timestamp := time.Now().Unix()
	name := fmt.Sprintf("agentic-session-%d", timestamp)

	// Create the custom resource
	// Metadata
	metadata := map[string]interface{}{
		"name":      name,
		"namespace": project,
	}
	if len(req.Labels) > 0 {
		labels := map[string]interface{}{}
		for k, v := range req.Labels {
			labels[k] = v
		}
		metadata["labels"] = labels
	}
	if len(req.Annotations) > 0 {
		annotations := map[string]interface{}{}
		for k, v := range req.Annotations {
			annotations[k] = v
		}
		metadata["annotations"] = annotations
	}

	spec := map[string]interface{}{
		"displayName": req.DisplayName,
		"project":     project,
		"llmSettings": map[string]interface{}{
			"model":       llmSettings.Model,
			"temperature": llmSettings.Temperature,
			"maxTokens":   llmSettings.MaxTokens,
		},
		"timeout": timeout,
	}
	if strings.TrimSpace(req.InitialPrompt) != "" {
		spec["initialPrompt"] = req.InitialPrompt
	}

	session := map[string]interface{}{
		"apiVersion": "vteam.ambient-code/v1alpha1",
		"kind":       "AgenticSession",
		"metadata":   metadata,
		"spec":       spec,
		"status": map[string]interface{}{
			"phase": "Pending",
		},
	}

	// Optional environment variables passthrough (always, independent of git config presence)
	envVars := make(map[string]string)
	for k, v := range req.EnvironmentVariables {
		envVars[k] = v
	}

	// Handle session continuation
	if req.ParentSessionID != "" {
		envVars["PARENT_SESSION_ID"] = req.ParentSessionID
		// Add annotation to track continuation lineage
		if metadata["annotations"] == nil {
			metadata["annotations"] = make(map[string]interface{})
		}
		annotations := metadata["annotations"].(map[string]interface{})
		annotations["vteam.ambient-code/parent-session-id"] = req.ParentSessionID
		log.Printf("Creating continuation session from parent %s (operator will handle temp pod cleanup)", req.ParentSessionID)
		// Note: Operator will delete temp pod when session starts (desired-phase=Running)
	}

	if len(envVars) > 0 {
		spec := session["spec"].(map[string]interface{})
		spec["environmentVariables"] = envVars
	}

	// Interactive flag
	if req.Interactive != nil {
		session["spec"].(map[string]interface{})["interactive"] = *req.Interactive
	}

	// AutoPushOnComplete flag
	if req.AutoPushOnComplete != nil {
		session["spec"].(map[string]interface{})["autoPushOnComplete"] = *req.AutoPushOnComplete
	}

	// Set multi-repo configuration on spec (simplified format)
	{
		spec := session["spec"].(map[string]interface{})
		if len(req.Repos) > 0 {
			arr := make([]map[string]interface{}, 0, len(req.Repos))
			for _, r := range req.Repos {
				m := map[string]interface{}{"url": r.URL}
				if r.Branch != nil {
					m["branch"] = *r.Branch
				}
				arr = append(arr, m)
			}
			spec["repos"] = arr
		}
	}

	// Add userContext derived from authenticated caller; ignore client-supplied userId
	{
		uidVal, _ := c.Get("userID")
		uid, _ := uidVal.(string)
		uid = strings.TrimSpace(uid)
		if uid != "" {
			displayName := ""
			if v, ok := c.Get("userName"); ok {
				if s, ok2 := v.(string); ok2 {
					displayName = s
				}
			}
			groups := []string{}
			if v, ok := c.Get("userGroups"); ok {
				if gg, ok2 := v.([]string); ok2 {
					groups = gg
				}
			}
			// Fallbacks for non-identity fields only
			if displayName == "" && req.UserContext != nil {
				displayName = req.UserContext.DisplayName
			}
			if len(groups) == 0 && req.UserContext != nil {
				groups = req.UserContext.Groups
			}
			session["spec"].(map[string]interface{})["userContext"] = map[string]interface{}{
				"userId":      uid,
				"displayName": displayName,
				"groups":      groups,
			}
		}
	}

	gvr := GetAgenticSessionV1Alpha1Resource()
	obj := &unstructured.Unstructured{Object: session}

	// Create AgenticSession using user token (enforces user RBAC permissions)
	created, err := reqDyn.Resource(gvr).Namespace(project).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create agentic session in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agentic session"})
		return
	}

	// Best-effort prefill of agent markdown into PVC workspace for immediate UI availability
	// Uses AGENT_PERSONAS or AGENT_PERSONA if provided in request environment variables
	func() {
		defer func() { _ = recover() }()
		personasCsv := ""
		if v, ok := req.EnvironmentVariables["AGENT_PERSONAS"]; ok && strings.TrimSpace(v) != "" {
			personasCsv = v
		} else if v, ok := req.EnvironmentVariables["AGENT_PERSONA"]; ok && strings.TrimSpace(v) != "" {
			personasCsv = v
		}
		if strings.TrimSpace(personasCsv) == "" {
			return
		}
		// content service removed; skip workspace path handling
		// Write each agent markdown
		for _, p := range strings.Split(personasCsv, ",") {
			persona := strings.TrimSpace(p)
			if persona == "" {
				continue
			}
			// ambient-content removed: skip agent prefill writes
		}
	}()

	// Provision runner token using backend SA (requires elevated permissions for SA/Role/Secret creation)
	if DynamicClient == nil || K8sClient == nil {
		log.Printf("Warning: backend SA clients not available, skipping runner token provisioning for session %s/%s", project, name)
	} else if err := provisionRunnerTokenForSession(c, K8sClient, DynamicClient, project, name); err != nil {
		// Non-fatal: log and continue. Operator may retry later if implemented.
		log.Printf("Warning: failed to provision runner token for session %s/%s: %v", project, name, err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Agentic session created successfully",
		"name":    name,
		"uid":     created.GetUID(),
	})
}

// provisionRunnerTokenForSession creates a per-session ServiceAccount, grants minimal RBAC,
// mints a short-lived token, stores it in a Secret, and annotates the AgenticSession with the Secret name.
func provisionRunnerTokenForSession(c *gin.Context, reqK8s *kubernetes.Clientset, reqDyn dynamic.Interface, project string, sessionName string) error {
	// Load owning AgenticSession to parent all resources
	gvr := GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), sessionName, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get AgenticSession: %w", err)
	}
	ownerRef := v1.OwnerReference{
		APIVersion: obj.GetAPIVersion(),
		Kind:       obj.GetKind(),
		Name:       obj.GetName(),
		UID:        obj.GetUID(),
		Controller: types.BoolPtr(true),
	}

	// Create ServiceAccount
	saName := fmt.Sprintf("ambient-session-%s", sessionName)
	sa := &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:            saName,
			Namespace:       project,
			Labels:          map[string]string{"app": "ambient-runner"},
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
	}
	if _, err := reqK8s.CoreV1().ServiceAccounts(project).Create(c.Request.Context(), sa, v1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("create SA: %w", err)
		}
	}

	// Create Role with least-privilege for updating AgenticSession status and annotations
	roleName := fmt.Sprintf("ambient-session-%s-role", sessionName)
	role := &rbacv1.Role{
		ObjectMeta: v1.ObjectMeta{
			Name:            roleName,
			Namespace:       project,
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"vteam.ambient-code"},
				Resources: []string{"agenticsessions"},
				Verbs:     []string{"get", "list", "watch", "update", "patch"}, // Added update, patch for annotations
			},
			{
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"selfsubjectaccessreviews"},
				Verbs:     []string{"create"},
			},
		},
	}
	// Try to create or update the Role to ensure it has latest permissions
	if _, err := reqK8s.RbacV1().Roles(project).Create(c.Request.Context(), role, v1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			// Role exists - update it to ensure it has the latest permissions (including update/patch)
			log.Printf("Role %s already exists, updating with latest permissions", roleName)
			if _, err := reqK8s.RbacV1().Roles(project).Update(c.Request.Context(), role, v1.UpdateOptions{}); err != nil {
				return fmt.Errorf("update Role: %w", err)
			}
			log.Printf("Successfully updated Role %s with annotation update permissions", roleName)
		} else {
			return fmt.Errorf("create Role: %w", err)
		}
	}

	// Bind Role to the ServiceAccount
	rbName := fmt.Sprintf("ambient-session-%s-rb", sessionName)
	rb := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:            rbName,
			Namespace:       project,
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
		RoleRef:  rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: roleName},
		Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: saName, Namespace: project}},
	}
	if _, err := reqK8s.RbacV1().RoleBindings(project).Create(context.TODO(), rb, v1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("create RoleBinding: %w", err)
		}
	}

	// Mint short-lived K8s ServiceAccount token for CR status updates
	tr := &authnv1.TokenRequest{Spec: authnv1.TokenRequestSpec{}}
	tok, err := reqK8s.CoreV1().ServiceAccounts(project).CreateToken(c.Request.Context(), saName, tr, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("mint token: %w", err)
	}
	k8sToken := tok.Status.Token
	if strings.TrimSpace(k8sToken) == "" {
		return fmt.Errorf("received empty token for SA %s", saName)
	}

	// Only store the K8s token; GitHub tokens are minted on-demand by the runner
	secretData := map[string]string{
		"k8s-token": k8sToken,
	}

	// Store token in a Secret (update if exists to refresh token)
	secretName := fmt.Sprintf("ambient-runner-token-%s", sessionName)
	refreshedAt := time.Now().UTC().Format(time.RFC3339)
	sec := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:            secretName,
			Namespace:       project,
			Labels:          map[string]string{"app": "ambient-runner-token"},
			OwnerReferences: []v1.OwnerReference{ownerRef},
			Annotations: map[string]string{
				runnerTokenRefreshedAtAnnotation: refreshedAt,
			},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: secretData,
	}

	// Try to create the secret
	if _, err := reqK8s.CoreV1().Secrets(project).Create(c.Request.Context(), sec, v1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			// Secret exists - update it with fresh token
			log.Printf("Updating existing secret %s with fresh token", secretName)
			existing, getErr := reqK8s.CoreV1().Secrets(project).Get(c.Request.Context(), secretName, v1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("get Secret for update: %w", getErr)
			}
			secretCopy := existing.DeepCopy()
			if secretCopy.Data == nil {
				secretCopy.Data = map[string][]byte{}
			}
			secretCopy.Data["k8s-token"] = []byte(k8sToken)
			if secretCopy.Annotations == nil {
				secretCopy.Annotations = map[string]string{}
			}
			secretCopy.Annotations[runnerTokenRefreshedAtAnnotation] = refreshedAt
			if _, err := reqK8s.CoreV1().Secrets(project).Update(c.Request.Context(), secretCopy, v1.UpdateOptions{}); err != nil {
				return fmt.Errorf("update Secret: %w", err)
			}
			log.Printf("Successfully updated secret %s with fresh token", secretName)
		} else {
			return fmt.Errorf("create Secret: %w", err)
		}
	}

	// Annotate the AgenticSession with the Secret and SA names (conflict-safe patch)
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				"ambient-code.io/runner-token-secret": secretName,
				"ambient-code.io/runner-sa":           saName,
			},
		},
	}
	b, _ := json.Marshal(patch)
	if _, err := reqDyn.Resource(gvr).Namespace(project).Patch(c.Request.Context(), obj.GetName(), ktypes.MergePatchType, b, v1.PatchOptions{}); err != nil {
		return fmt.Errorf("annotate AgenticSession: %w", err)
	}

	return nil
}

func GetSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	_ = reqK8s
	gvr := GetAgenticSessionV1Alpha1Resource()

	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	session := types.AgenticSession{
		APIVersion: item.GetAPIVersion(),
		Kind:       item.GetKind(),
		Metadata:   item.Object["metadata"].(map[string]interface{}),
	}

	if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := item.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusOK, session)
}

// MintSessionGitHubToken validates the token via TokenReview, ensures SA matches CR annotation, and returns a short-lived GitHub token.
// POST /api/projects/:projectName/agentic-sessions/:sessionName/github/token
// Auth: Authorization: Bearer <BOT_TOKEN> (K8s SA token with audience "ambient-backend")
func MintSessionGitHubToken(c *gin.Context) {
	project := c.Param("projectName")
	sessionName := c.Param("sessionName")

	rawAuth := strings.TrimSpace(c.GetHeader("Authorization"))
	if rawAuth == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
		return
	}
	parts := strings.SplitN(rawAuth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
		return
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "empty token"})
		return
	}

	// TokenReview using default audience (works with standard SA tokens)
	tr := &authnv1.TokenReview{Spec: authnv1.TokenReviewSpec{Token: token}}
	rv, err := K8sClient.AuthenticationV1().TokenReviews().Create(c.Request.Context(), tr, v1.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token review failed"})
		return
	}
	if rv.Status.Error != "" || !rv.Status.Authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	subj := strings.TrimSpace(rv.Status.User.Username)
	const pfx = "system:serviceaccount:"
	if !strings.HasPrefix(subj, pfx) {
		c.JSON(http.StatusForbidden, gin.H{"error": "subject is not a service account"})
		return
	}
	rest := strings.TrimPrefix(subj, pfx)
	segs := strings.SplitN(rest, ":", 2)
	if len(segs) != 2 {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid service account subject"})
		return
	}
	nsFromToken, saFromToken := segs[0], segs[1]
	if nsFromToken != project {
		c.JSON(http.StatusForbidden, gin.H{"error": "namespace mismatch"})
		return
	}

	// Load session and verify SA matches annotation
	gvr := GetAgenticSessionV1Alpha1Resource()
	obj, err := DynamicClient.Resource(gvr).Namespace(project).Get(c.Request.Context(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read session"})
		return
	}
	meta, _ := obj.Object["metadata"].(map[string]interface{})
	anns, _ := meta["annotations"].(map[string]interface{})
	expectedSA := ""
	if anns != nil {
		if v, ok := anns["ambient-code.io/runner-sa"].(string); ok {
			expectedSA = strings.TrimSpace(v)
		}
	}
	if expectedSA == "" || expectedSA != saFromToken {
		c.JSON(http.StatusForbidden, gin.H{"error": "service account not authorized for session"})
		return
	}

	// Read authoritative userId from spec.userContext.userId
	spec, _ := obj.Object["spec"].(map[string]interface{})
	userID := ""
	if spec != nil {
		if uc, ok := spec["userContext"].(map[string]interface{}); ok {
			if v, ok := uc["userId"].(string); ok {
				userID = strings.TrimSpace(v)
			}
		}
	}
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session missing user context"})
		return
	}

	// Get GitHub token (GitHub App or PAT fallback via project runner secret)
	tokenStr, err := GetGitHubToken(c.Request.Context(), K8sClient, DynamicClient, project, userID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	// Note: PATs don't have expiration, so we omit expiresAt for simplicity
	// Runners should treat all tokens as short-lived and request new ones as needed
	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}

func PatchSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)

	var patch map[string]interface{}
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get current resource
	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	// Apply patch to metadata annotations
	if metaPatch, ok := patch["metadata"].(map[string]interface{}); ok {
		if annsPatch, ok := metaPatch["annotations"].(map[string]interface{}); ok {
			metadata := item.Object["metadata"].(map[string]interface{})
			if metadata["annotations"] == nil {
				metadata["annotations"] = make(map[string]interface{})
			}
			anns := metadata["annotations"].(map[string]interface{})
			for k, v := range annsPatch {
				anns[k] = v
			}
		}
	}

	// Update the resource
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to patch agentic session %s: %v", sessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to patch session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session patched successfully", "annotations": updated.GetAnnotations()})
}

func UpdateSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	_ = reqK8s

	var req types.UpdateAgenticSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get current resource with brief retry to avoid race on creation
	var item *unstructured.Unstructured
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		item, err = reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
		if err == nil {
			break
		}
		if errors.IsNotFound(err) {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Prevent spec changes while session is running or being created
	if status, ok := item.Object["status"].(map[string]interface{}); ok {
		if phase, ok := status["phase"].(string); ok {
			if strings.EqualFold(phase, "Running") || strings.EqualFold(phase, "Creating") {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Cannot modify session specification while the session is running",
					"phase": phase,
				})
				return
			}
		}
	}

	// Update spec
	spec := item.Object["spec"].(map[string]interface{})
	if req.InitialPrompt != nil {
		spec["initialPrompt"] = *req.InitialPrompt
	}
	if req.DisplayName != nil {
		spec["displayName"] = *req.DisplayName
	}

	if req.LLMSettings != nil {
		llmSettings := make(map[string]interface{})
		if req.LLMSettings.Model != "" {
			llmSettings["model"] = req.LLMSettings.Model
		}
		if req.LLMSettings.Temperature != 0 {
			llmSettings["temperature"] = req.LLMSettings.Temperature
		}
		if req.LLMSettings.MaxTokens != 0 {
			llmSettings["maxTokens"] = req.LLMSettings.MaxTokens
		}
		spec["llmSettings"] = llmSettings
	}

	if req.Timeout != nil {
		spec["timeout"] = *req.Timeout
	}

	// Update the resource
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agentic session"})
		return
	}

	// Parse and return updated session
	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}

	if spec, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusOK, session)
}

// UpdateSessionDisplayName updates only the spec.displayName field on the AgenticSession.
// PUT /api/projects/:projectName/agentic-sessions/:sessionName/displayname
func UpdateSessionDisplayName(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)

	var req struct {
		DisplayName string `json:"displayName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Retrieve current resource
	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Update only displayName in spec
	spec, ok := item.Object["spec"].(map[string]interface{})
	if !ok {
		spec = make(map[string]interface{})
		item.Object["spec"] = spec
	}
	spec["displayName"] = req.DisplayName

	// Persist the change
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update display name for agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update display name"})
		return
	}

	// Respond with updated session summary
	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}
	if s, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(s)
	}
	if st, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(st)
	}

	c.JSON(http.StatusOK, session)
}

// SelectWorkflow sets the active workflow for a session
// POST /api/projects/:projectName/agentic-sessions/:sessionName/workflow
func SelectWorkflow(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)

	var req types.WorkflowSelection
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Retrieve current resource
	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	if err := ensureRuntimeMutationAllowed(item); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Update activeWorkflow in spec
	spec, ok := item.Object["spec"].(map[string]interface{})
	if !ok {
		spec = make(map[string]interface{})
		item.Object["spec"] = spec
	}

	// Set activeWorkflow
	workflowMap := map[string]interface{}{
		"gitUrl": req.GitURL,
	}
	if req.Branch != "" {
		workflowMap["branch"] = req.Branch
	} else {
		workflowMap["branch"] = "main"
	}
	if req.Path != "" {
		workflowMap["path"] = req.Path
	}
	spec["activeWorkflow"] = workflowMap

	// Persist the change
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update workflow for agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update workflow"})
		return
	}

	log.Printf("Workflow updated for session %s: %s@%s", sessionName, req.GitURL, workflowMap["branch"])

	// Respond with updated session summary
	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}
	if s, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(s)
	}
	if st, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(st)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Workflow updated successfully",
		"session": session,
	})
}

// AddRepo adds a new repository to a running session
// POST /api/projects/:projectName/agentic-sessions/:sessionName/repos
func AddRepo(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)

	var req struct {
		URL    string `json:"url" binding:"required"`
		Branch string `json:"branch"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Branch == "" {
		req.Branch = "main"
	}

	gvr := GetAgenticSessionV1Alpha1Resource()
	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	if err := ensureRuntimeMutationAllowed(item); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Update spec.repos
	spec, ok := item.Object["spec"].(map[string]interface{})
	if !ok {
		spec = make(map[string]interface{})
		item.Object["spec"] = spec
	}
	repos, _ := spec["repos"].([]interface{})
	if repos == nil {
		repos = []interface{}{}
	}

	newRepo := map[string]interface{}{
		"url":    req.URL,
		"branch": req.Branch,
	}
	repos = append(repos, newRepo)
	spec["repos"] = repos

	// Persist change
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}
	if specMap, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(specMap)
	}
	if statusMap, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(statusMap)
	}

	log.Printf("Added repository %s to session %s in project %s", req.URL, sessionName, project)
	c.JSON(http.StatusOK, gin.H{"message": "Repository added", "session": session})
}

// RemoveRepo removes a repository from a running session
// DELETE /api/projects/:projectName/agentic-sessions/:sessionName/repos/:repoName
func RemoveRepo(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	repoName := c.Param("repoName")
	_, reqDyn := GetK8sClientsForRequest(c)

	gvr := GetAgenticSessionV1Alpha1Resource()
	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	if err := ensureRuntimeMutationAllowed(item); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Update spec.repos
	spec, ok := item.Object["spec"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session has no spec"})
		return
	}
	repos, _ := spec["repos"].([]interface{})

	filteredRepos := []interface{}{}
	found := false
	for _, r := range repos {
		rm, _ := r.(map[string]interface{})
		url, _ := rm["url"].(string)
		if DeriveRepoFolderFromURL(url) != repoName {
			filteredRepos = append(filteredRepos, r)
		} else {
			found = true
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found in session"})
		return
	}

	spec["repos"] = filteredRepos

	// Persist change
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}
	if specMap, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(specMap)
	}
	if statusMap, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(statusMap)
	}

	log.Printf("Removed repository %s from session %s in project %s", repoName, sessionName, project)
	c.JSON(http.StatusOK, gin.H{"message": "Repository removed", "session": session})
}

// GetWorkflowMetadata retrieves commands and agents metadata from the active workflow
// GET /api/projects/:projectName/agentic-sessions/:sessionName/workflow/metadata
func GetWorkflowMetadata(c *gin.Context) {
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	sessionName := c.Param("sessionName")

	if project == "" {
		log.Printf("GetWorkflowMetadata: project is empty, session=%s", sessionName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	// Get authorization token
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// Try temp service first (for completed sessions), then regular service
	serviceName := fmt.Sprintf("temp-content-%s", sessionName)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			// Temp service doesn't exist, use regular service
			serviceName = fmt.Sprintf("ambient-content-%s", sessionName)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", sessionName)
	}

	// Build URL to content service
	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	u := fmt.Sprintf("%s/content/workflow-metadata?session=%s", endpoint, sessionName)

	log.Printf("GetWorkflowMetadata: project=%s session=%s endpoint=%s", project, sessionName, endpoint)

	// Create and send request to content pod
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GetWorkflowMetadata: content service request failed: %v", err)
		// Return empty metadata on error
		c.JSON(http.StatusOK, gin.H{"commands": []interface{}{}, "agents": []interface{}{}})
		return
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", b)
}

// fetchGitHubFileContent fetches a file from GitHub via API
// token is optional - works for public repos without authentication (but has rate limits)
func fetchGitHubFileContent(ctx context.Context, owner, repo, ref, path, token string) ([]byte, error) {
	api := "https://api.github.com"
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", api, owner, repo, path, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Only set Authorization header if token is provided
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github.raw")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("file not found")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// fetchGitHubDirectoryListing lists files/folders in a GitHub directory
// token is optional - works for public repos without authentication (but has rate limits)
func fetchGitHubDirectoryListing(ctx context.Context, owner, repo, ref, path, token string) ([]map[string]interface{}, error) {
	api := "https://api.github.com"
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", api, owner, repo, path, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Only set Authorization header if token is provided
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var entries []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}

	return entries, nil
}

// OOTBWorkflow represents an out-of-the-box workflow
type OOTBWorkflow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	GitURL      string `json:"gitUrl"`
	Branch      string `json:"branch"`
	Path        string `json:"path,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// ListOOTBWorkflows returns the list of out-of-the-box workflows dynamically discovered from GitHub
// Attempts to use user's GitHub token for better rate limits, falls back to unauthenticated for public repos
// GET /api/workflows/ootb?project=<projectName>
func ListOOTBWorkflows(c *gin.Context) {
	// Try to get user's GitHub token (best effort - not required)
	// This gives better rate limits (5000/hr vs 60/hr) and supports private repos
	// Project is optional - if provided, we'll try to get the user's token
	token := ""
	project := c.Query("project") // Optional query parameter
	if project != "" {
		userID, _ := c.Get("userID")
		if reqK8s, reqDyn := GetK8sClientsForRequest(c); reqK8s != nil {
			if userIDStr, ok := userID.(string); ok && userIDStr != "" {
				if githubToken, err := GetGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userIDStr); err == nil {
					token = githubToken
					log.Printf("ListOOTBWorkflows: using user's GitHub token for project %s (better rate limits)", project)
				} else {
					log.Printf("ListOOTBWorkflows: failed to get GitHub token for project %s: %v", project, err)
				}
			}
		}
	}
	if token == "" {
		log.Printf("ListOOTBWorkflows: proceeding without GitHub token (public repo, lower rate limits)")
	}

	// Read OOTB repo configuration from environment
	ootbRepo := strings.TrimSpace(os.Getenv("OOTB_WORKFLOWS_REPO"))
	if ootbRepo == "" {
		ootbRepo = "https://github.com/ambient-code/ootb-ambient-workflows.git"
	}

	ootbBranch := strings.TrimSpace(os.Getenv("OOTB_WORKFLOWS_BRANCH"))
	if ootbBranch == "" {
		ootbBranch = "main"
	}

	ootbWorkflowsPath := strings.TrimSpace(os.Getenv("OOTB_WORKFLOWS_PATH"))
	if ootbWorkflowsPath == "" {
		ootbWorkflowsPath = "workflows"
	}

	// Parse GitHub URL
	owner, repoName, err := git.ParseGitHubURL(ootbRepo)
	if err != nil {
		log.Printf("ListOOTBWorkflows: invalid repo URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid OOTB repo URL"})
		return
	}

	// List workflow directories
	entries, err := fetchGitHubDirectoryListing(c.Request.Context(), owner, repoName, ootbBranch, ootbWorkflowsPath, token)
	if err != nil {
		log.Printf("ListOOTBWorkflows: failed to list workflows directory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to discover OOTB workflows"})
		return
	}

	// Scan each subdirectory for ambient.json
	workflows := []OOTBWorkflow{}
	for _, entry := range entries {
		entryType, _ := entry["type"].(string)
		entryName, _ := entry["name"].(string)

		if entryType != "dir" {
			continue
		}

		// Try to fetch ambient.json from this workflow directory
		ambientPath := fmt.Sprintf("%s/%s/.ambient/ambient.json", ootbWorkflowsPath, entryName)
		ambientData, err := fetchGitHubFileContent(c.Request.Context(), owner, repoName, ootbBranch, ambientPath, token)

		var ambientConfig struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err == nil {
			// Parse ambient.json if found
			if parseErr := json.Unmarshal(ambientData, &ambientConfig); parseErr != nil {
				log.Printf("ListOOTBWorkflows: failed to parse ambient.json for %s: %v", entryName, parseErr)
			}
		}

		// Use ambient.json values or fallback to directory name
		workflowName := ambientConfig.Name
		if workflowName == "" {
			workflowName = strings.ReplaceAll(entryName, "-", " ")
			workflowName = strings.Title(workflowName)
		}

		workflows = append(workflows, OOTBWorkflow{
			ID:          entryName,
			Name:        workflowName,
			Description: ambientConfig.Description,
			GitURL:      ootbRepo,
			Branch:      ootbBranch,
			Path:        fmt.Sprintf("%s/%s", ootbWorkflowsPath, entryName),
			Enabled:     true,
		})
	}

	log.Printf("ListOOTBWorkflows: discovered %d workflows from %s", len(workflows), ootbRepo)
	c.JSON(http.StatusOK, gin.H{"workflows": workflows})
}

func DeleteSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	_ = reqK8s
	gvr := GetAgenticSessionV1Alpha1Resource()

	err := reqDyn.Resource(gvr).Namespace(project).Delete(context.TODO(), sessionName, v1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to delete agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agentic session"})
		return
	}

	c.Status(http.StatusNoContent)
}

func CloneSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)

	var req types.CloneSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get source session
	sourceItem, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Source session not found"})
			return
		}
		log.Printf("Failed to get source agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get source agentic session"})
		return
	}

	// Validate target project exists and is managed by Ambient via OpenShift Project
	projGvr := GetOpenShiftProjectResource()
	projObj, err := reqDyn.Resource(projGvr).Get(context.TODO(), req.TargetProject, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Target project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate target project"})
		return
	}

	isAmbient := false
	if meta, ok := projObj.Object["metadata"].(map[string]interface{}); ok {
		if raw, ok := meta["labels"].(map[string]interface{}); ok {
			if v, ok := raw["ambient-code.io/managed"].(string); ok && v == "true" {
				isAmbient = true
			}
		}
	}
	if !isAmbient {
		c.JSON(http.StatusForbidden, gin.H{"error": "Target project is not managed by Ambient"})
		return
	}

	// Ensure unique target session name in target namespace; if exists, append "-duplicate" (and numeric suffix)
	newName := strings.TrimSpace(req.NewSessionName)
	if newName == "" {
		newName = sessionName
	}
	finalName := newName
	conflicted := false
	for i := 0; i < 50; i++ {
		_, getErr := reqDyn.Resource(gvr).Namespace(req.TargetProject).Get(context.TODO(), finalName, v1.GetOptions{})
		if errors.IsNotFound(getErr) {
			break
		}
		if getErr != nil && !errors.IsNotFound(getErr) {
			// On unexpected error, still attempt to proceed with a duplicate suffix to reduce collision chance
			log.Printf("cloneSession: name check encountered error for %s/%s: %v", req.TargetProject, finalName, getErr)
		}
		conflicted = true
		if i == 0 {
			finalName = fmt.Sprintf("%s-duplicate", newName)
		} else {
			finalName = fmt.Sprintf("%s-duplicate-%d", newName, i+1)
		}
	}

	// Create cloned session
	clonedSession := map[string]interface{}{
		"apiVersion": "vteam.ambient-code/v1alpha1",
		"kind":       "AgenticSession",
		"metadata": map[string]interface{}{
			"name":      finalName,
			"namespace": req.TargetProject,
		},
		"spec": sourceItem.Object["spec"],
		"status": map[string]interface{}{
			"phase": "Pending",
		},
	}

	// Update project in spec
	clonedSpec := clonedSession["spec"].(map[string]interface{})
	clonedSpec["project"] = req.TargetProject
	if conflicted {
		if dn, ok := clonedSpec["displayName"].(string); ok && strings.TrimSpace(dn) != "" {
			clonedSpec["displayName"] = fmt.Sprintf("%s (Duplicate)", dn)
		} else {
			clonedSpec["displayName"] = fmt.Sprintf("%s (Duplicate)", finalName)
		}
	}

	obj := &unstructured.Unstructured{Object: clonedSession}

	created, err := reqDyn.Resource(gvr).Namespace(req.TargetProject).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create cloned agentic session in project %s: %v", req.TargetProject, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cloned agentic session"})
		return
	}

	// Parse and return created session
	session := types.AgenticSession{
		APIVersion: created.GetAPIVersion(),
		Kind:       created.GetKind(),
		Metadata:   created.Object["metadata"].(map[string]interface{}),
	}

	if spec, ok := created.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := created.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusCreated, session)
}

// ensureRunnerRolePermissions updates the runner role to ensure it has all required permissions
// This is useful for existing sessions that were created before we added new permissions
func ensureRunnerRolePermissions(c *gin.Context, reqK8s *kubernetes.Clientset, project string, sessionName string) error {
	roleName := fmt.Sprintf("ambient-session-%s-role", sessionName)

	// Get existing role
	existingRole, err := reqK8s.RbacV1().Roles(project).Get(c.Request.Context(), roleName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Role %s not found for session %s - will be created by operator", roleName, sessionName)
			return nil
		}
		return fmt.Errorf("get role: %w", err)
	}

	// Check if role has selfsubjectaccessreviews permission
	hasSelfSubjectAccessReview := false
	for _, rule := range existingRole.Rules {
		for _, apiGroup := range rule.APIGroups {
			if apiGroup == "authorization.k8s.io" {
				for _, resource := range rule.Resources {
					if resource == "selfsubjectaccessreviews" {
						hasSelfSubjectAccessReview = true
						break
					}
				}
			}
		}
	}

	if hasSelfSubjectAccessReview {
		log.Printf("Role %s already has selfsubjectaccessreviews permission", roleName)
		return nil
	}

	// Add missing permission
	log.Printf("Updating role %s to add selfsubjectaccessreviews permission", roleName)
	existingRole.Rules = append(existingRole.Rules, rbacv1.PolicyRule{
		APIGroups: []string{"authorization.k8s.io"},
		Resources: []string{"selfsubjectaccessreviews"},
		Verbs:     []string{"create"},
	})

	_, err = reqK8s.RbacV1().Roles(project).Update(c.Request.Context(), existingRole, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}

	log.Printf("Successfully updated role %s with selfsubjectaccessreviews permission", roleName)
	return nil
}

func StartSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)
	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get current resource
	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Check if this is a continuation (session is in a terminal phase)
	isActualContinuation := false
	if currentStatus, ok := item.Object["status"].(map[string]interface{}); ok {
		if phase, ok := currentStatus["phase"].(string); ok {
			terminalPhases := []string{"Completed", "Failed", "Stopped", "Error"}
			for _, terminalPhase := range terminalPhases {
				if phase == terminalPhase {
					isActualContinuation = true
					log.Printf("StartSession: Detected continuation - session is in terminal phase: %s", phase)
					break
				}
			}
		}
	}

	// Set annotations to signal desired state to operator
	annotations := item.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Signal start/restart request to operator
	annotations["ambient-code.io/desired-phase"] = "Running"
	annotations["ambient-code.io/start-requested-at"] = time.Now().Format(time.RFC3339)

	// For continuations, set parent-session-id so operator reuses PVC
	if isActualContinuation {
		annotations["vteam.ambient-code/parent-session-id"] = sessionName
		log.Printf("StartSession: Continuation detected - set parent-session-id=%s for PVC reuse", sessionName)
	}

	item.SetAnnotations(annotations)

	// For headless sessions being continued, force interactive mode
	if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
		if interactive, ok := spec["interactive"].(bool); !ok || !interactive {
			spec["interactive"] = true
			log.Printf("StartSession: Converting headless session to interactive for continuation")
		}
	}

	// Update spec and annotations (operator will observe and handle job lifecycle)
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	log.Printf("StartSession: Set desired-phase=Running annotation (operator will reconcile)")

	// Parse and return updated session
	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}

	if spec, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusAccepted, session)
}

func ensureRuntimeMutationAllowed(item *unstructured.Unstructured) error {
	if item == nil {
		return fmt.Errorf("session not loaded")
	}

	spec, _ := item.Object["spec"].(map[string]interface{})
	interactive := false
	if spec != nil {
		if v, ok := spec["interactive"].(bool); ok {
			interactive = v
		}
	}
	if !interactive {
		return fmt.Errorf("session is not interactive")
	}

	status, _ := item.Object["status"].(map[string]interface{})
	phase := ""
	if status != nil {
		if p, ok := status["phase"].(string); ok {
			phase = strings.TrimSpace(strings.ToLower(p))
		}
	}

	if phase != "running" {
		displayPhase := "unknown"
		if status != nil {
			if original, ok := status["phase"].(string); ok && strings.TrimSpace(original) != "" {
				displayPhase = original
			}
		}
		return fmt.Errorf("session must be Running to mutate spec (current phase: %s)", displayPhase)
	}

	return nil
}

func StopSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)
	gvr := GetAgenticSessionV1Alpha1Resource()

	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Set annotations to signal desired state to operator
	annotations := item.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Signal stop request to operator
	annotations["ambient-code.io/desired-phase"] = "Stopped"
	annotations["ambient-code.io/stop-requested-at"] = time.Now().Format(time.RFC3339)
	item.SetAnnotations(annotations)

	// Force interactive mode so session can be restarted later
	if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
		if interactive, ok := spec["interactive"].(bool); !ok || !interactive {
			spec["interactive"] = true
			log.Printf("StopSession: Converting headless session to interactive for future restart capability")
		}
	}

	// Update spec and annotations (operator will observe and handle job cleanup)
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"message": "Session no longer exists (already deleted)"})
			return
		}
		log.Printf("Failed to update agentic session %s: %v", sessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	log.Printf("StopSession: Set desired-phase=Stopped annotation (operator will reconcile)")

	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}
	if specMap, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(specMap)
	}
	if statusMap, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(statusMap)
	}

	c.JSON(http.StatusAccepted, session)
}

// EnableWorkspaceAccess requests a temporary content pod for workspace access on stopped sessions
// POST /api/projects/:projectName/agentic-sessions/:sessionName/workspace/enable
func EnableWorkspaceAccess(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)
	gvr := GetAgenticSessionV1Alpha1Resource()

	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	// Only allow for stopped/completed/failed sessions
	status, _ := item.Object["status"].(map[string]interface{})
	phase, _ := status["phase"].(string)
	if phase != "Stopped" && phase != "Completed" && phase != "Failed" {
		c.JSON(http.StatusConflict, gin.H{"error": "Workspace access only available for stopped sessions"})
		return
	}

	// Set annotation to request temp pod
	annotations := item.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	annotations["ambient-code.io/temp-content-requested"] = "true"
	annotations["ambient-code.io/temp-content-last-accessed"] = now
	item.SetAnnotations(annotations)

	// Update CR
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable workspace access"})
		return
	}

	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}
	if spec, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(spec)
	}
	if status, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(status)
	}

	log.Printf("EnableWorkspaceAccess: Set temp-content-requested annotation for %s", sessionName)
	c.JSON(http.StatusAccepted, session)
}

// TouchWorkspaceAccess updates the last-accessed timestamp to keep temp pod alive
// POST /api/projects/:projectName/agentic-sessions/:sessionName/workspace/touch
func TouchWorkspaceAccess(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)
	gvr := GetAgenticSessionV1Alpha1Resource()

	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	annotations := item.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["ambient-code.io/temp-content-last-accessed"] = time.Now().UTC().Format(time.RFC3339)
	item.SetAnnotations(annotations)

	if _, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update timestamp"})
		return
	}

	log.Printf("TouchWorkspaceAccess: Updated last-accessed timestamp for %s", sessionName)
	c.JSON(http.StatusOK, gin.H{"message": "Workspace access timestamp updated"})
}

// GetSessionK8sResources returns job, pod, and PVC information for a session
// GET /api/projects/:projectName/agentic-sessions/:sessionName/k8s-resources
func GetSessionK8sResources(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	sessionName := c.Param("sessionName")

	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get session to find job name
	gvr := GetAgenticSessionV1Alpha1Resource()
	session, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), sessionName, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	status, _ := session.Object["status"].(map[string]interface{})
	jobName, _ := status["jobName"].(string)
	if jobName == "" {
		jobName = fmt.Sprintf("%s-job", sessionName)
	}

	result := map[string]interface{}{}

	// Get Job status
	job, err := reqK8s.BatchV1().Jobs(project).Get(c.Request.Context(), jobName, v1.GetOptions{})
	jobExists := err == nil

	if jobExists {
		result["jobName"] = jobName
		jobStatus := "Unknown"
		if job.Status.Active > 0 {
			jobStatus = "Active"
		} else if job.Status.Succeeded > 0 {
			jobStatus = "Succeeded"
		} else if job.Status.Failed > 0 {
			jobStatus = "Failed"
		}
		result["jobStatus"] = jobStatus
		result["jobConditions"] = job.Status.Conditions
	} else if errors.IsNotFound(err) {
		// Job not found - don't return job info at all
		log.Printf("GetSessionK8sResources: Job %s not found, omitting from response", jobName)
		// Don't include jobName or jobStatus in result
	} else {
		// Other error - still show job name but with error status
		result["jobName"] = jobName
		result["jobStatus"] = "Error"
		log.Printf("GetSessionK8sResources: Error getting job %s: %v", jobName, err)
	}

	// Get Pods for this job (only if job exists)
	podInfos := []map[string]interface{}{}
	if jobExists {
		pods, err := reqK8s.CoreV1().Pods(project).List(c.Request.Context(), v1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", jobName),
		})
		if err == nil {
			for _, pod := range pods.Items {
				// Check if pod is terminating (has DeletionTimestamp)
				podPhase := string(pod.Status.Phase)
				if pod.DeletionTimestamp != nil {
					podPhase = "Terminating"
				}

				containerInfos := []map[string]interface{}{}
				for _, cs := range pod.Status.ContainerStatuses {
					state := "Unknown"
					var exitCode *int32
					var reason string
					if cs.State.Running != nil {
						state = "Running"
						// If pod is terminating but container still shows running, mark it as terminating
						if pod.DeletionTimestamp != nil {
							state = "Terminating"
						}
					} else if cs.State.Terminated != nil {
						state = "Terminated"
						exitCode = &cs.State.Terminated.ExitCode
						reason = cs.State.Terminated.Reason
					} else if cs.State.Waiting != nil {
						state = "Waiting"
						reason = cs.State.Waiting.Reason
					}
					containerInfos = append(containerInfos, map[string]interface{}{
						"name":     cs.Name,
						"state":    state,
						"exitCode": exitCode,
						"reason":   reason,
					})
				}
				podInfos = append(podInfos, map[string]interface{}{
					"name":       pod.Name,
					"phase":      podPhase,
					"containers": containerInfos,
				})
			}
		}
	}

	// Check for temp-content pod
	tempPodName := fmt.Sprintf("temp-content-%s", sessionName)
	tempPod, err := reqK8s.CoreV1().Pods(project).Get(c.Request.Context(), tempPodName, v1.GetOptions{})
	if err == nil {
		tempPodPhase := string(tempPod.Status.Phase)
		if tempPod.DeletionTimestamp != nil {
			tempPodPhase = "Terminating"
		}

		containerInfos := []map[string]interface{}{}
		for _, cs := range tempPod.Status.ContainerStatuses {
			state := "Unknown"
			var exitCode *int32
			var reason string
			if cs.State.Running != nil {
				state = "Running"
				// If pod is terminating but container still shows running, mark as terminating
				if tempPod.DeletionTimestamp != nil {
					state = "Terminating"
				}
			} else if cs.State.Terminated != nil {
				state = "Terminated"
				exitCode = &cs.State.Terminated.ExitCode
				reason = cs.State.Terminated.Reason
			} else if cs.State.Waiting != nil {
				state = "Waiting"
				reason = cs.State.Waiting.Reason
			}
			containerInfos = append(containerInfos, map[string]interface{}{
				"name":     cs.Name,
				"state":    state,
				"exitCode": exitCode,
				"reason":   reason,
			})
		}
		podInfos = append(podInfos, map[string]interface{}{
			"name":       tempPod.Name,
			"phase":      tempPodPhase,
			"containers": containerInfos,
			"isTempPod":  true,
		})
	}

	result["pods"] = podInfos

	// Get PVC info - always use session's own PVC name
	// Note: If session was created with parent_session_id (via API), the operator handles PVC reuse
	pvcName := fmt.Sprintf("ambient-workspace-%s", sessionName)
	pvc, err := reqK8s.CoreV1().PersistentVolumeClaims(project).Get(c.Request.Context(), pvcName, v1.GetOptions{})
	result["pvcName"] = pvcName
	if err == nil {
		result["pvcExists"] = true
		if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
			result["pvcSize"] = storage.String()
		}
	} else {
		result["pvcExists"] = false
	}

	c.JSON(http.StatusOK, result)
}

// setRepoStatus removed - status.repos no longer in CRD (status simplified to phase, message, is_error)

// ListSessionWorkspace proxies to per-job content service for directory listing.
func ListSessionWorkspace(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		log.Printf("ListSessionWorkspace: project is empty, session=%s", session)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	rel := strings.TrimSpace(c.Query("path"))
	// Build absolute workspace path using plain session (no url.PathEscape to match FS paths)
	absPath := "/sessions/" + session + "/workspace"
	if rel != "" {
		absPath += "/" + rel
	}

	// Call per-job service or temp service for completed sessions
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// Try temp service first (for completed sessions), then regular service
	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			// Temp service doesn't exist, use regular service
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	u := fmt.Sprintf("%s/content/list?path=%s", endpoint, url.QueryEscape(absPath))
	log.Printf("ListSessionWorkspace: project=%s session=%s endpoint=%s", project, session, endpoint)
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ListSessionWorkspace: content service request failed: %v", err)
		// Soften error to 200 with empty list so UI doesn't spam
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	// If content service returns 404, check if it's because workspace doesn't exist yet
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("ListSessionWorkspace: workspace not found (may not be created yet by runner)")
		// Return empty list instead of error for better UX during session startup
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), b)
}

// GetSessionWorkspaceFile reads a file via content service.
func GetSessionWorkspaceFile(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		log.Printf("GetSessionWorkspaceFile: project is empty, session=%s", session)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	sub := strings.TrimPrefix(c.Param("path"), "/")
	absPath := "/sessions/" + session + "/workspace/" + sub
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// Try temp service first (for completed sessions), then regular service
	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	u := fmt.Sprintf("%s/content/file?path=%s", endpoint, url.QueryEscape(absPath))
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), b)
}

// PutSessionWorkspaceFile writes a file via content service.
func PutSessionWorkspaceFile(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		log.Printf("PutSessionWorkspaceFile: project is empty, session=%s", session)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}
	sub := strings.TrimPrefix(c.Param("path"), "/")
	absPath := "/sessions/" + session + "/workspace/" + sub
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// Try temp service first (for completed sessions), then regular service
	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			// Temp service doesn't exist, use regular service
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	log.Printf("PutSessionWorkspaceFile: using service %s for session %s", serviceName, session)
	payload, _ := io.ReadAll(c.Request.Body)
	wreq := struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}{Path: absPath, Content: string(payload), Encoding: "utf8"}
	b, _ := json.Marshal(wreq)
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint+"/content/write", strings.NewReader(string(b)))
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), rb)
}

// PushSessionRepo proxies a push request for a given session repo to the per-job content service.
// POST /api/projects/:projectName/agentic-sessions/:sessionName/github/push
// Body: { repoIndex: number, commitMessage?: string, branch?: string }
func PushSessionRepo(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	var body struct {
		RepoIndex     int    `json:"repoIndex"`
		CommitMessage string `json:"commitMessage"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	log.Printf("pushSessionRepo: request project=%s session=%s repoIndex=%d commitLen=%d", project, session, body.RepoIndex, len(strings.TrimSpace(body.CommitMessage)))

	// Try temp service first (for completed sessions), then regular service
	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}
	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	log.Printf("pushSessionRepo: using service %s", serviceName)

	// Simplified: 1) get session; 2) compute repoPath from INPUT repo folder; 3) get output url/branch; 4) proxy
	resolvedRepoPath := ""
	// default branch when not defined on output
	resolvedBranch := fmt.Sprintf("sessions/%s", session)
	resolvedOutputURL := ""
	if _, reqDyn := GetK8sClientsForRequest(c); reqDyn != nil {
		gvr := GetAgenticSessionV1Alpha1Resource()
		obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read session"})
			return
		}
		spec, _ := obj.Object["spec"].(map[string]interface{})
		repos, _ := spec["repos"].([]interface{})
		if body.RepoIndex < 0 || body.RepoIndex >= len(repos) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repo index"})
			return
		}
		rm, _ := repos[body.RepoIndex].(map[string]interface{})
		// Derive repoPath from input URL folder name
		if in, ok := rm["input"].(map[string]interface{}); ok {
			if urlv, ok2 := in["url"].(string); ok2 && strings.TrimSpace(urlv) != "" {
				folder := DeriveRepoFolderFromURL(strings.TrimSpace(urlv))
				if folder != "" {
					resolvedRepoPath = fmt.Sprintf("/sessions/%s/workspace/%s", session, folder)
				}
			}
		}
		if out, ok := rm["output"].(map[string]interface{}); ok {
			if urlv, ok2 := out["url"].(string); ok2 && strings.TrimSpace(urlv) != "" {
				resolvedOutputURL = strings.TrimSpace(urlv)
			}
			if bs, ok2 := out["branch"].(string); ok2 && strings.TrimSpace(bs) != "" {
				resolvedBranch = strings.TrimSpace(bs)
			} else if bv, ok2 := out["branch"].(*string); ok2 && bv != nil && strings.TrimSpace(*bv) != "" {
				resolvedBranch = strings.TrimSpace(*bv)
			}
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no dynamic client"})
		return
	}
	// If input URL missing or unparsable, fall back to numeric index path (last resort)
	if strings.TrimSpace(resolvedRepoPath) == "" {
		if body.RepoIndex >= 0 {
			resolvedRepoPath = fmt.Sprintf("/sessions/%s/workspace/%d", session, body.RepoIndex)
		} else {
			resolvedRepoPath = fmt.Sprintf("/sessions/%s/workspace", session)
		}
	}
	if strings.TrimSpace(resolvedOutputURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing output repo url"})
		return
	}
	log.Printf("pushSessionRepo: resolved repoPath=%q outputUrl=%q branch=%q", resolvedRepoPath, resolvedOutputURL, resolvedBranch)

	payload := map[string]interface{}{
		"repoPath":      resolvedRepoPath,
		"commitMessage": body.CommitMessage,
		"branch":        resolvedBranch,
		"outputRepoUrl": resolvedOutputURL,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint+"/content/github/push", strings.NewReader(string(b)))
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}
	if v := c.GetHeader("X-Forwarded-Access-Token"); v != "" {
		req.Header.Set("X-Forwarded-Access-Token", v)
	}
	req.Header.Set("Content-Type", "application/json")

	// Attach short-lived GitHub token for one-shot authenticated push
	if reqK8s, reqDyn := GetK8sClientsForRequest(c); reqK8s != nil {
		// Load session to get authoritative userId
		gvr := GetAgenticSessionV1Alpha1Resource()
		obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
		if err == nil {
			spec, _ := obj.Object["spec"].(map[string]interface{})
			userID := ""
			if spec != nil {
				if uc, ok := spec["userContext"].(map[string]interface{}); ok {
					if v, ok := uc["userId"].(string); ok {
						userID = strings.TrimSpace(v)
					}
				}
			}
			if userID != "" {
				if tokenStr, err := GetGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userID); err == nil && strings.TrimSpace(tokenStr) != "" {
					req.Header.Set("X-GitHub-Token", tokenStr)
					log.Printf("pushSessionRepo: attached short-lived GitHub token for project=%s session=%s", project, session)
				} else if err != nil {
					log.Printf("pushSessionRepo: failed to resolve GitHub token: %v", err)
				}
			} else {
				log.Printf("pushSessionRepo: session %s/%s missing userContext.userId; proceeding without token", project, session)
			}
		} else {
			log.Printf("pushSessionRepo: failed to read session for token attach: %v", err)
		}
	}

	log.Printf("pushSessionRepo: proxy push project=%s session=%s repoIndex=%d repoPath=%s endpoint=%s", project, session, body.RepoIndex, resolvedRepoPath, endpoint+"/content/github/push")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("pushSessionRepo: content returned status=%d body.snip=%q", resp.StatusCode, func() string {
			s := string(bodyBytes)
			if len(s) > 1500 {
				return s[:1500] + "..."
			}
			return s
		}())
		c.Data(resp.StatusCode, "application/json", bodyBytes)
		return
	}
	// Note: status.repos removed from CRD - no longer tracking per-repo status
	log.Printf("pushSessionRepo: content push succeeded status=%d body.len=%d", resp.StatusCode, len(bodyBytes))
	c.Data(http.StatusOK, "application/json", bodyBytes)
}

// AbandonSessionRepo instructs sidecar to discard local changes for a repo.
func AbandonSessionRepo(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	var body struct {
		RepoIndex int    `json:"repoIndex"`
		RepoPath  string `json:"repoPath"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	// Try temp service first (for completed sessions), then regular service
	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}
	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	log.Printf("AbandonSessionRepo: using service %s", serviceName)
	repoPath := strings.TrimSpace(body.RepoPath)
	if repoPath == "" {
		if body.RepoIndex >= 0 {
			repoPath = fmt.Sprintf("/sessions/%s/workspace/%d", session, body.RepoIndex)
		} else {
			repoPath = fmt.Sprintf("/sessions/%s/workspace", session)
		}
	}
	payload := map[string]interface{}{
		"repoPath": repoPath,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint+"/content/github/abandon", strings.NewReader(string(b)))
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}
	if v := c.GetHeader("X-Forwarded-Access-Token"); v != "" {
		req.Header.Set("X-Forwarded-Access-Token", v)
	}
	req.Header.Set("Content-Type", "application/json")
	log.Printf("abandonSessionRepo: proxy abandon project=%s session=%s repoIndex=%d repoPath=%s", project, session, body.RepoIndex, repoPath)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("abandonSessionRepo: content returned status=%d body=%s", resp.StatusCode, string(bodyBytes))
		c.Data(resp.StatusCode, "application/json", bodyBytes)
		return
	}
	// Note: status.repos removed from CRD - no longer tracking per-repo status
	c.Data(http.StatusOK, "application/json", bodyBytes)
}

// DiffSessionRepo proxies diff counts for a given session repo to the content sidecar.
// GET /api/projects/:projectName/agentic-sessions/:sessionName/github/diff?repoIndex=0&repoPath=...
func DiffSessionRepo(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	repoIndexStr := strings.TrimSpace(c.Query("repoIndex"))
	repoPath := strings.TrimSpace(c.Query("repoPath"))
	if repoPath == "" && repoIndexStr != "" {
		repoPath = fmt.Sprintf("/sessions/%s/workspace/%s", session, repoIndexStr)
	}
	if repoPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing repoPath/repoIndex"})
		return
	}

	// Try temp service first (for completed sessions), then regular service
	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}
	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	log.Printf("DiffSessionRepo: using service %s", serviceName)
	url := fmt.Sprintf("%s/content/github/diff?repoPath=%s", endpoint, url.QueryEscape(repoPath))
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}
	if v := c.GetHeader("X-Forwarded-Access-Token"); v != "" {
		req.Header.Set("X-Forwarded-Access-Token", v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"files": gin.H{
				"added":   0,
				"removed": 0,
			},
			"total_added":   0,
			"total_removed": 0,
		})
		return
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GetGitStatus returns git status for a directory in the workspace
// GET /api/projects/:projectName/agentic-sessions/:sessionName/git/status?path=artifacts
func GetGitStatus(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	relativePath := strings.TrimSpace(c.Query("path"))

	if relativePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter required"})
		return
	}

	// Build absolute path
	absPath := fmt.Sprintf("/sessions/%s/workspace/%s", session, relativePath)

	// Get content service endpoint
	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-status?path=%s", serviceName, project, url.QueryEscape(absPath))

	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, endpoint, nil)
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// ConfigureGitRemote initializes git and configures remote for a workspace directory
// Body: { path: string, remoteURL: string, branch: string }
// POST /api/projects/:projectName/agentic-sessions/:sessionName/git/configure-remote
func ConfigureGitRemote(c *gin.Context) {
	project := c.Param("projectName")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)

	var body struct {
		Path      string `json:"path" binding:"required"`
		RemoteURL string `json:"remoteUrl" binding:"required"`
		Branch    string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if body.Branch == "" {
		body.Branch = "main"
	}

	// Build absolute path
	absPath := fmt.Sprintf("/sessions/%s/workspace/%s", sessionName, body.Path)

	// Get content service endpoint
	serviceName := fmt.Sprintf("temp-content-%s", sessionName)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", sessionName)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", sessionName)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-configure-remote", serviceName, project)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"path":      absPath,
		"remoteUrl": body.RemoteURL,
		"branch":    body.Branch,
	})

	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, strings.NewReader(string(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	// Get and forward GitHub token for authenticated remote URL
	if reqK8s != nil && reqDyn != nil && GetGitHubToken != nil {
		if token, err := GetGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, ""); err == nil && token != "" {
			req.Header.Set("X-GitHub-Token", token)
			log.Printf("Forwarding GitHub token for remote configuration")
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	// If successful, persist remote config to session annotations for persistence
	if resp.StatusCode == http.StatusOK {
		// Persist remote config in annotations (supports multiple directories)
		gvr := GetAgenticSessionV1Alpha1Resource()
		item, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), sessionName, v1.GetOptions{})
		if err == nil {
			metadata := item.Object["metadata"].(map[string]interface{})
			if metadata["annotations"] == nil {
				metadata["annotations"] = make(map[string]interface{})
			}
			anns := metadata["annotations"].(map[string]interface{})

			// Derive safe annotation key from path (use :: as separator to avoid conflicts with hyphens in path)
			annotationKey := strings.ReplaceAll(body.Path, "/", "::")
			anns[fmt.Sprintf("ambient-code.io/remote-%s-url", annotationKey)] = body.RemoteURL
			anns[fmt.Sprintf("ambient-code.io/remote-%s-branch", annotationKey)] = body.Branch

			_, err = reqDyn.Resource(gvr).Namespace(project).Update(c.Request.Context(), item, v1.UpdateOptions{})
			if err != nil {
				log.Printf("Warning: Failed to persist remote config to annotations: %v", err)
			} else {
				log.Printf("Persisted remote config for %s to session annotations: %s@%s", body.Path, body.RemoteURL, body.Branch)
			}
		}
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// SynchronizeGit commits, pulls, and pushes changes for a workspace directory
// Body: { path: string, message?: string, branch?: string }
// POST /api/projects/:projectName/agentic-sessions/:sessionName/git/synchronize
func SynchronizeGit(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	var body struct {
		Path    string `json:"path" binding:"required"`
		Message string `json:"message"`
		Branch  string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Auto-generate commit message if not provided
	if body.Message == "" {
		body.Message = fmt.Sprintf("Session %s - %s", session, time.Now().Format(time.RFC3339))
	}

	// Build absolute path
	absPath := fmt.Sprintf("/sessions/%s/workspace/%s", session, body.Path)

	// Get content service endpoint
	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-sync", serviceName, project)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"path":    absPath,
		"message": body.Message,
		"branch":  body.Branch,
	})

	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, strings.NewReader(string(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GetGitMergeStatus checks if local and remote can merge cleanly
// GET /api/projects/:projectName/agentic-sessions/:sessionName/git/merge-status?path=&branch=
func GetGitMergeStatus(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	relativePath := strings.TrimSpace(c.Query("path"))
	branch := strings.TrimSpace(c.Query("branch"))

	if relativePath == "" {
		relativePath = "artifacts"
	}
	if branch == "" {
		branch = "main"
	}

	absPath := fmt.Sprintf("/sessions/%s/workspace/%s", session, relativePath)

	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-merge-status?path=%s&branch=%s",
		serviceName, project, url.QueryEscape(absPath), url.QueryEscape(branch))

	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, endpoint, nil)
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GitPullSession pulls changes from remote
// POST /api/projects/:projectName/agentic-sessions/:sessionName/git/pull
func GitPullSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	var body struct {
		Path   string `json:"path"`
		Branch string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if body.Path == "" {
		body.Path = "artifacts"
	}
	if body.Branch == "" {
		body.Branch = "main"
	}

	absPath := fmt.Sprintf("/sessions/%s/workspace/%s", session, body.Path)

	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-pull", serviceName, project)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"path":   absPath,
		"branch": body.Branch,
	})

	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, strings.NewReader(string(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GitPushSession pushes changes to remote branch
// POST /api/projects/:projectName/agentic-sessions/:sessionName/git/push
func GitPushSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	var body struct {
		Path    string `json:"path"`
		Branch  string `json:"branch"`
		Message string `json:"message"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if body.Path == "" {
		body.Path = "artifacts"
	}
	if body.Branch == "" {
		body.Branch = "main"
	}
	if body.Message == "" {
		body.Message = fmt.Sprintf("Session %s artifacts", session)
	}

	absPath := fmt.Sprintf("/sessions/%s/workspace/%s", session, body.Path)

	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-push", serviceName, project)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"path":    absPath,
		"branch":  body.Branch,
		"message": body.Message,
	})

	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, strings.NewReader(string(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GitCreateBranchSession creates a new git branch
// POST /api/projects/:projectName/agentic-sessions/:sessionName/git/create-branch
func GitCreateBranchSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	var body struct {
		Path       string `json:"path"`
		BranchName string `json:"branchName" binding:"required"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if body.Path == "" {
		body.Path = "artifacts"
	}

	absPath := fmt.Sprintf("/sessions/%s/workspace/%s", session, body.Path)

	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-create-branch", serviceName, project)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"path":       absPath,
		"branchName": body.BranchName,
	})

	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, strings.NewReader(string(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GitListBranchesSession lists all remote branches
// GET /api/projects/:projectName/agentic-sessions/:sessionName/git/list-branches?path=
func GitListBranchesSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	relativePath := strings.TrimSpace(c.Query("path"))

	if relativePath == "" {
		relativePath = "artifacts"
	}

	absPath := fmt.Sprintf("/sessions/%s/workspace/%s", session, relativePath)

	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-list-branches?path=%s",
		serviceName, project, url.QueryEscape(absPath))

	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, endpoint, nil)
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}
