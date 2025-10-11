package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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
	ParseStatus                       func(map[string]interface{}) *types.AgenticSessionStatus
	GetGitHubToken                    func(context.Context, *kubernetes.Clientset, dynamic.Interface, string, string) (string, error)
	DeriveRepoFolderFromURL           func(string) string
)

// contentListItem represents a file/directory in the workspace
type contentListItem struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
}

// parseSpec parses AgenticSessionSpec with v1alpha1 fields
func parseSpec(spec map[string]interface{}) types.AgenticSessionSpec {
	result := types.AgenticSessionSpec{}

	if prompt, ok := spec["prompt"].(string); ok {
		result.Prompt = prompt
	}

	if interactive, ok := spec["interactive"].(bool); ok {
		result.Interactive = interactive
	}

	if autoPushOnComplete, ok := spec["autoPushOnComplete"].(bool); ok {
		result.AutoPushOnComplete = autoPushOnComplete
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

	if botAccount, ok := spec["botAccount"].(map[string]interface{}); ok {
		ba := &types.BotAccountRef{}
		if name, ok := botAccount["name"].(string); ok {
			ba.Name = name
		}
		result.BotAccount = ba
	}

	if resourceOverrides, ok := spec["resourceOverrides"].(map[string]interface{}); ok {
		ro := &types.ResourceOverrides{}
		if cpu, ok := resourceOverrides["cpu"].(string); ok {
			ro.CPU = cpu
		}
		if memory, ok := resourceOverrides["memory"].(string); ok {
			ro.Memory = memory
		}
		if storageClass, ok := resourceOverrides["storageClass"].(string); ok {
			ro.StorageClass = storageClass
		}
		if priorityClass, ok := resourceOverrides["priorityClass"].(string); ok {
			ro.PriorityClass = priorityClass
		}
		result.ResourceOverrides = ro
	}

	// Multi-repo parsing (unified repos)
	if arr, ok := spec["repos"].([]interface{}); ok {
		repos := make([]types.SessionRepoMapping, 0, len(arr))
		for _, it := range arr {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			r := types.SessionRepoMapping{}
			if in, ok := m["input"].(map[string]interface{}); ok {
				ng := types.NamedGitRepo{}
				if s, ok := in["url"].(string); ok {
					ng.URL = s
				}
				if s, ok := in["branch"].(string); ok && strings.TrimSpace(s) != "" {
					ng.Branch = StringPtr(s)
				}
				r.Input = ng
			}
			if out, ok := m["output"].(map[string]interface{}); ok {
				og := &types.OutputNamedGitRepo{}
				if s, ok := out["url"].(string); ok {
					og.URL = s
				}
				if s, ok := out["branch"].(string); ok && strings.TrimSpace(s) != "" {
					og.Branch = StringPtr(s)
				}
				r.Output = og
			}
			// Include per-repo status if present
			if st, ok := m["status"].(string); ok {
				r.Status = StringPtr(st)
			}
			if strings.TrimSpace(r.Input.URL) != "" {
				repos = append(repos, r)
			}
		}
		result.Repos = repos
	}
	if idx, ok := spec["mainRepoIndex"].(float64); ok {
		idxInt := int(idx)
		result.MainRepoIndex = &idxInt
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
			session.Status = ParseStatus(status)
		}

		sessions = append(sessions, session)
	}

	c.JSON(http.StatusOK, gin.H{"items": sessions})
}

func CreateSession(c *gin.Context) {
	project := c.GetString("project")
	// Use backend service account clients for CR writes
	if DynamicClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "backend not initialized"})
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

	session := map[string]interface{}{
		"apiVersion": "vteam.ambient-code/v1alpha1",
		"kind":       "AgenticSession",
		"metadata":   metadata,
		"spec": map[string]interface{}{
			"prompt":      req.Prompt,
			"displayName": req.DisplayName,
			"project":     project,
			"llmSettings": map[string]interface{}{
				"model":       llmSettings.Model,
				"temperature": llmSettings.Temperature,
				"maxTokens":   llmSettings.MaxTokens,
			},
			"timeout": timeout,
		},
		"status": map[string]interface{}{
			"phase": "Pending",
		},
	}

	// Optional environment variables passthrough (always, independent of git config presence)
	if len(req.EnvironmentVariables) > 0 {
		spec := session["spec"].(map[string]interface{})
		spec["environmentVariables"] = req.EnvironmentVariables
	}

	// Interactive flag
	if req.Interactive != nil {
		session["spec"].(map[string]interface{})["interactive"] = *req.Interactive
	}

	// AutoPushOnComplete flag
	if req.AutoPushOnComplete != nil {
		session["spec"].(map[string]interface{})["autoPushOnComplete"] = *req.AutoPushOnComplete
	}

	// Set multi-repo configuration on spec
	{
		spec := session["spec"].(map[string]interface{})
		// Multi-repo pass-through (unified repos)
		if len(req.Repos) > 0 {
			arr := make([]map[string]interface{}, 0, len(req.Repos))
			for _, r := range req.Repos {
				m := map[string]interface{}{}
				in := map[string]interface{}{"url": r.Input.URL}
				if r.Input.Branch != nil {
					in["branch"] = *r.Input.Branch
				}
				m["input"] = in
				if r.Output != nil {
					out := map[string]interface{}{"url": r.Output.URL}
					if r.Output.Branch != nil {
						out["branch"] = *r.Output.Branch
					}
					m["output"] = out
				}
				// Remove default repo status; status will be set explicitly when pushed/abandoned
				// m["status"] intentionally unset at creation time
				arr = append(arr, m)
			}
			spec["repos"] = arr
		}
		if req.MainRepoIndex != nil {
			spec["mainRepoIndex"] = *req.MainRepoIndex
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

	// Add botAccount if provided
	if req.BotAccount != nil {
		session["spec"].(map[string]interface{})["botAccount"] = map[string]interface{}{
			"name": req.BotAccount.Name,
		}
	}

	// Add resourceOverrides if provided
	if req.ResourceOverrides != nil {
		resourceOverrides := make(map[string]interface{})
		if req.ResourceOverrides.CPU != "" {
			resourceOverrides["cpu"] = req.ResourceOverrides.CPU
		}
		if req.ResourceOverrides.Memory != "" {
			resourceOverrides["memory"] = req.ResourceOverrides.Memory
		}
		if req.ResourceOverrides.StorageClass != "" {
			resourceOverrides["storageClass"] = req.ResourceOverrides.StorageClass
		}
		if req.ResourceOverrides.PriorityClass != "" {
			resourceOverrides["priorityClass"] = req.ResourceOverrides.PriorityClass
		}
		if len(resourceOverrides) > 0 {
			session["spec"].(map[string]interface{})["resourceOverrides"] = resourceOverrides
		}
	}

	gvr := GetAgenticSessionV1Alpha1Resource()
	obj := &unstructured.Unstructured{Object: session}

	created, err := DynamicClient.Resource(gvr).Namespace(project).Create(context.TODO(), obj, v1.CreateOptions{})
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

	// Preferred method: provision a per-session ServiceAccount token for the runner (backend SA)
	if err := provisionRunnerTokenForSession(c, K8sClient, DynamicClient, project, name); err != nil {
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
		Controller: BoolPtr(true),
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

	// Create Role with least-privilege for updating AgenticSession status
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
				Resources: []string{"agenticsessions/status"},
				Verbs:     []string{"get", "update", "patch"},
			},
			{
				APIGroups: []string{"vteam.ambient-code"},
				Resources: []string{"agenticsessions"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	if _, err := reqK8s.RbacV1().Roles(project).Create(c.Request.Context(), role, v1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
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

	// Store both tokens in a Secret
	secretName := fmt.Sprintf("ambient-runner-token-%s", sessionName)
	sec := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:            secretName,
			Namespace:       project,
			Labels:          map[string]string{"app": "ambient-runner-token"},
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: secretData,
	}
	if _, err := reqK8s.CoreV1().Secrets(project).Create(c.Request.Context(), sec, v1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
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
		session.Status = ParseStatus(status)
	}

	c.JSON(http.StatusOK, session)
}

// POST /api/projects/:projectName/agentic-sessions/:sessionName/github/token
// Auth: Authorization: Bearer <BOT_TOKEN> (K8s SA token with audience "ambient-backend")
// Validates the token via TokenReview, ensures SA matches CR annotation, and returns a short-lived GitHub token.
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
	userId := ""
	if spec != nil {
		if uc, ok := spec["userContext"].(map[string]interface{}); ok {
			if v, ok := uc["userId"].(string); ok {
				userId = strings.TrimSpace(v)
			}
		}
	}
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session missing user context"})
		return
	}

	// Get GitHub token (GitHub App or PAT fallback via project runner secret)
	tokenStr, err := GetGitHubToken(c.Request.Context(), K8sClient, DynamicClient, project, userId)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	// Note: PATs don't have expiration, so we omit expiresAt for simplicity
	// Runners should treat all tokens as short-lived and request new ones as needed
	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}

func UpdateSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	_ = reqK8s

	var req types.CreateAgenticSessionRequest
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

	// Update spec
	spec := item.Object["spec"].(map[string]interface{})
	spec["prompt"] = req.Prompt
	spec["displayName"] = req.DisplayName

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
		session.Status = ParseStatus(status)
	}

	c.JSON(http.StatusOK, session)
}

// PUT /api/projects/:projectName/agentic-sessions/:sessionName/displayname
// updateSessionDisplayName updates only the spec.displayName field on the AgenticSession
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
		session.Status = ParseStatus(st)
	}

	c.JSON(http.StatusOK, session)
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
		session.Status = ParseStatus(status)
	}

	c.JSON(http.StatusCreated, session)
}

func StartSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	_ = reqK8s
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

	// Update status to trigger start
	if item.Object["status"] == nil {
		item.Object["status"] = make(map[string]interface{})
	}

	status := item.Object["status"].(map[string]interface{})
	status["phase"] = "Creating"
	status["message"] = "Session start requested"
	status["startTime"] = time.Now().Format(time.RFC3339)

	// Update the resource
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to start agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start agentic session"})
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
		session.Status = ParseStatus(status)
	}

	c.JSON(http.StatusAccepted, session)
}

func StopSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
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

	// Check current status
	status, ok := item.Object["status"].(map[string]interface{})
	if !ok {
		status = make(map[string]interface{})
		item.Object["status"] = status
	}

	currentPhase, _ := status["phase"].(string)
	if currentPhase == "Completed" || currentPhase == "Failed" || currentPhase == "Stopped" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot stop session in %s state", currentPhase)})
		return
	}

	log.Printf("Attempting to stop agentic session %s in project %s (current phase: %s)", sessionName, project, currentPhase)

	// Get job name from status
	jobName, jobExists := status["jobName"].(string)
	if jobExists && jobName != "" {
		// Delete the job
		err := reqK8s.BatchV1().Jobs(project).Delete(context.TODO(), jobName, v1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Printf("Failed to delete job %s: %v", jobName, err)
			// Don't fail the request if job deletion fails - continue with status update
			log.Printf("Continuing with status update despite job deletion failure")
		} else {
			log.Printf("Deleted job %s for agentic session %s", jobName, sessionName)
		}
	} else {
		// Handle case where job was never created or jobName is missing
		log.Printf("No job found to delete for agentic session %s", sessionName)
	}

	// Update status to Stopped
	status["phase"] = "Stopped"
	status["message"] = "Session stopped by user"
	status["completionTime"] = time.Now().Format(time.RFC3339)

	// Update the resource
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Session was deleted while we were trying to update it
			log.Printf("Agentic session %s was deleted during stop operation", sessionName)
			c.JSON(http.StatusOK, gin.H{"message": "Session no longer exists (already deleted)"})
			return
		}
		log.Printf("Failed to update agentic session status %s: %v", sessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agentic session status"})
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
		session.Status = ParseStatus(status)
	}

	log.Printf("Successfully stopped agentic session %s", sessionName)
	c.JSON(http.StatusAccepted, session)
}

// PUT /api/projects/:projectName/agentic-sessions/:sessionName/status
// updateSessionStatus writes selected fields to PVC-backed files and updates CR status
func UpdateSessionStatus(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)

	var statusUpdate map[string]interface{}
	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
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
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Ensure status map
	if item.Object["status"] == nil {
		item.Object["status"] = make(map[string]interface{})
	}
	status := item.Object["status"].(map[string]interface{})

	// Accept standard fields and result summary fields from runner
	allowed := map[string]struct{}{
		"phase": {}, "completionTime": {}, "cost": {}, "message": {},
		"subtype": {}, "duration_ms": {}, "duration_api_ms": {}, "is_error": {},
		"num_turns": {}, "session_id": {}, "total_cost_usd": {}, "usage": {}, "result": {},
	}
	for k := range statusUpdate {
		if _, ok := allowed[k]; !ok {
			delete(statusUpdate, k)
		}
	}

	// Merge remaining fields into status
	for k, v := range statusUpdate {
		status[k] = v
	}

	// Update only the status subresource (requires agenticsessions/status perms)
	if _, err := reqDyn.Resource(gvr).Namespace(project).UpdateStatus(context.TODO(), item, v1.UpdateOptions{}); err != nil {
		log.Printf("Failed to update agentic session status %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agentic session status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "agentic session status updated"})
}

// setRepoStatus updates spec.repos[idx].status to a new value
func setRepoStatus(dyn dynamic.Interface, project, sessionName string, repoIndex int, newStatus string) error {
	gvr := GetAgenticSessionV1Alpha1Resource()
	item, err := dyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		return err
	}
	spec, _ := item.Object["spec"].(map[string]interface{})
	if spec == nil {
		spec = map[string]interface{}{}
	}
	repos, _ := spec["repos"].([]interface{})
	if repoIndex < 0 || repoIndex >= len(repos) {
		return fmt.Errorf("repo index out of range")
	}
	rm, _ := repos[repoIndex].(map[string]interface{})
	if rm == nil {
		rm = map[string]interface{}{}
	}
	rm["status"] = newStatus
	repos[repoIndex] = rm
	spec["repos"] = repos
	item.Object["spec"] = spec
	updated, err := dyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("setRepoStatus: update failed project=%s session=%s repoIndex=%d status=%s err=%v", project, sessionName, repoIndex, newStatus, err)
		return err
	}
	if updated != nil {
		log.Printf("setRepoStatus: update ok project=%s session=%s repoIndex=%d status=%s", project, sessionName, repoIndex, newStatus)
	}
	return nil
}

// listSessionWorkspace proxies to per-job content service for directory listing
func ListSessionWorkspace(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	rel := strings.TrimSpace(c.Query("path"))
	// Build absolute workspace path using plain session (no url.PathEscape to match FS paths)
	absPath := "/sessions/" + session + "/workspace"
	if rel != "" {
		absPath += "/" + rel
	}

	// Call per-job service directly to avoid any default base that targets per-namespace service
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}
	base := os.Getenv("SESSION_CONTENT_SERVICE_BASE")
	if base == "" {
		// Per-job Service name created by operator: ambient-content-<session> in project namespace
		base = "http://ambient-content-%s.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, session, project)
	u := fmt.Sprintf("%s/content/list?path=%s", endpoint, url.QueryEscape(absPath))
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Soften error to 200 with empty list so UI doesn't spam
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), b)
}

// getSessionWorkspaceFile reads a file via content service
func GetSessionWorkspaceFile(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	sub := strings.TrimPrefix(c.Param("path"), "/")
	absPath := "/sessions/" + session + "/workspace/" + sub
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}
	base := os.Getenv("SESSION_CONTENT_SERVICE_BASE")
	if base == "" {
		base = "http://ambient-content-%s.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, session, project)
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
	b, _ := ioutil.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), b)
}

// putSessionWorkspaceFile writes a file via content service
func PutSessionWorkspaceFile(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	sub := strings.TrimPrefix(c.Param("path"), "/")
	absPath := "/sessions/" + session + "/workspace/" + sub
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}
	base := os.Getenv("SESSION_CONTENT_SERVICE_BASE")
	if base == "" {
		base = "http://ambient-content-%s.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, session, project)
	payload, _ := ioutil.ReadAll(c.Request.Body)
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
	rb, _ := ioutil.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), rb)
}

// pushSessionRepo proxies a push request for a given session repo to the per-job content service.
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

	base := os.Getenv("SESSION_CONTENT_SERVICE_BASE")
	if base == "" {
		// Default: per-job service name ambient-content-<session> in the project namespace
		base = "http://ambient-content-%s.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, session, project)

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
			userId := ""
			if spec != nil {
				if uc, ok := spec["userContext"].(map[string]interface{}); ok {
					if v, ok := uc["userId"].(string); ok {
						userId = strings.TrimSpace(v)
					}
				}
			}
			if userId != "" {
				if tokenStr, err := GetGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userId); err == nil && strings.TrimSpace(tokenStr) != "" {
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
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
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
	if _, reqDyn := GetK8sClientsForRequest(c); reqDyn != nil {
		log.Printf("pushSessionRepo: setting repo status to 'pushed' for repoIndex=%d", body.RepoIndex)
		if err := setRepoStatus(reqDyn, project, session, body.RepoIndex, "pushed"); err != nil {
			log.Printf("pushSessionRepo: setRepoStatus failed project=%s session=%s repoIndex=%d err=%v", project, session, body.RepoIndex, err)
		}
	} else {
		log.Printf("pushSessionRepo: no dynamic client; cannot set repo status project=%s session=%s", project, session)
	}
	log.Printf("pushSessionRepo: content push succeeded status=%d body.len=%d", resp.StatusCode, len(bodyBytes))
	c.Data(http.StatusOK, "application/json", bodyBytes)
}

// abandonSessionRepo instructs sidecar to discard local changes for a repo
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
	base := os.Getenv("SESSION_CONTENT_SERVICE_BASE")
	if base == "" {
		base = "http://ambient-content-%s.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, session, project)
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
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("abandonSessionRepo: content returned status=%d body=%s", resp.StatusCode, string(bodyBytes))
		c.Data(resp.StatusCode, "application/json", bodyBytes)
		return
	}
	if _, reqDyn := GetK8sClientsForRequest(c); reqDyn != nil {
		if err := setRepoStatus(reqDyn, project, session, body.RepoIndex, "abandoned"); err != nil {
			log.Printf("abandonSessionRepo: setRepoStatus failed project=%s session=%s repoIndex=%d err=%v", project, session, body.RepoIndex, err)
		}
	} else {
		log.Printf("abandonSessionRepo: no dynamic client; cannot set repo status project=%s session=%s", project, session)
	}
	c.Data(http.StatusOK, "application/json", bodyBytes)
}

// diffSessionRepo proxies diff counts for a given session repo to the content sidecar
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
	base := os.Getenv("SESSION_CONTENT_SERVICE_BASE")
	if base == "" {
		base = "http://ambient-content-%s.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, session, project)
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
		c.JSON(http.StatusOK, gin.H{"added": 0, "modified": 0, "deleted": 0, "renamed": 0, "untracked": 0})
		return
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", b)
}
