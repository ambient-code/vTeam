package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	authnv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// feature flags and small helpers
var (
	boolPtr = func(b bool) *bool { return &b }
)

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

// Middleware for project context validation
func validateProjectContext() gin.HandlerFunc {
	return func(c *gin.Context) {
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

// parseSpec parses AgenticSessionSpec with v1alpha1 fields
func parseSpec(spec map[string]interface{}) AgenticSessionSpec {
	result := AgenticSessionSpec{}

	if prompt, ok := spec["prompt"].(string); ok {
		result.Prompt = prompt
	}

	if interactive, ok := spec["interactive"].(bool); ok {
		result.Interactive = interactive
	}

	if paths, ok := spec["paths"].(map[string]interface{}); ok {
		p := &Paths{}
		if ws, ok := paths["workspace"].(string); ok {
			p.Workspace = ws
		}
		if ms, ok := paths["messages"].(string); ok {
			p.Messages = ms
		}
		if ib, ok := paths["inbox"].(string); ok {
			p.Inbox = ib
		}
		result.Paths = p
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

	if userContext, ok := spec["userContext"].(map[string]interface{}); ok {
		uc := &UserContext{}
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
		ba := &BotAccountRef{}
		if name, ok := botAccount["name"].(string); ok {
			ba.Name = name
		}
		result.BotAccount = ba
	}

	if resourceOverrides, ok := spec["resourceOverrides"].(map[string]interface{}); ok {
		ro := &ResourceOverrides{}
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

	// Parse Git configuration
	if gitConfig, ok := spec["gitConfig"].(map[string]interface{}); ok {
		result.GitConfig = &GitConfig{}

		// Parse user
		if user, ok := gitConfig["user"].(map[string]interface{}); ok {
			result.GitConfig.User = &GitUser{}
			if name, ok := user["name"].(string); ok {
				result.GitConfig.User.Name = name
			}
			if email, ok := user["email"].(string); ok {
				result.GitConfig.User.Email = email
			}
		}

		// Parse authentication
		if auth, ok := gitConfig["authentication"].(map[string]interface{}); ok {
			result.GitConfig.Authentication = &GitAuthentication{}
			if sshKeySecret, ok := auth["sshKeySecret"].(string); ok {
				result.GitConfig.Authentication.SSHKeySecret = &sshKeySecret
			}
			if tokenSecret, ok := auth["tokenSecret"].(string); ok {
				result.GitConfig.Authentication.TokenSecret = &tokenSecret
			}
		}

		// Parse repositories
		if repos, ok := gitConfig["repositories"].([]interface{}); ok {
			result.GitConfig.Repositories = make([]GitRepository, len(repos))
			for i, repo := range repos {
				if repoMap, ok := repo.(map[string]interface{}); ok {
					gitRepo := GitRepository{}
					if url, ok := repoMap["url"].(string); ok {
						gitRepo.URL = url
					}
					if branch, ok := repoMap["branch"].(string); ok {
						gitRepo.Branch = &branch
					}
					if clonePath, ok := repoMap["clonePath"].(string); ok {
						gitRepo.ClonePath = &clonePath
					}
					result.GitConfig.Repositories[i] = gitRepo
				}
			}
		}
	}

	return result
}

// V2 API Handlers - Multi-tenant session management

func listSessions(c *gin.Context) {
	project := c.GetString("project")
	reqK8s, reqDyn := getK8sClientsForRequest(c)
	_ = reqK8s
	gvr := getAgenticSessionV1Alpha1Resource()

	list, err := reqDyn.Resource(gvr).Namespace(project).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list agentic sessions in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list agentic sessions"})
		return
	}

	var sessions []AgenticSession
	for _, item := range list.Items {
		session := AgenticSession{
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

func createSession(c *gin.Context) {
	project := c.GetString("project")
	reqK8s, reqDyn := getK8sClientsForRequest(c)
	_ = reqK8s
	var req CreateAgenticSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults for LLM settings if not provided
	llmSettings := LLMSettings{
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

	// Only include paths if a workspacePath was provided
	if strings.TrimSpace(req.WorkspacePath) != "" {
		spec := session["spec"].(map[string]interface{})
		spec["paths"] = map[string]interface{}{
			"workspace": req.WorkspacePath,
		}
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

	// Load Git configuration from ConfigMap and merge with user-provided config
	if defaultGitConfig, err := loadGitConfigFromConfigMapForProject(c, reqK8s, project); err != nil {
		log.Printf("Warning: failed to load Git config from ConfigMap in %s: %v", project, err)
	} else {
		mergedGitConfig := mergeGitConfigs(req.GitConfig, defaultGitConfig)
		if mergedGitConfig != nil {
			gitConfig := map[string]interface{}{}
			if mergedGitConfig.User != nil {
				gitConfig["user"] = map[string]interface{}{
					"name":  mergedGitConfig.User.Name,
					"email": mergedGitConfig.User.Email,
				}
			}

			if mergedGitConfig.Authentication != nil {
				auth := map[string]interface{}{}
				if mergedGitConfig.Authentication.SSHKeySecret != nil {
					auth["sshKeySecret"] = *mergedGitConfig.Authentication.SSHKeySecret
				}
				if mergedGitConfig.Authentication.TokenSecret != nil {
					auth["tokenSecret"] = *mergedGitConfig.Authentication.TokenSecret
				}
				if len(auth) > 0 {
					gitConfig["authentication"] = auth
				}
			}
			if len(mergedGitConfig.Repositories) > 0 {
				repos := make([]map[string]interface{}, len(mergedGitConfig.Repositories))
				for i, repo := range mergedGitConfig.Repositories {
					repoMap := map[string]interface{}{
						"url": repo.URL,
					}
					if repo.Branch != nil {
						repoMap["branch"] = *repo.Branch
					}
					if repo.ClonePath != nil {
						repoMap["clonePath"] = *repo.ClonePath
					}
					repos[i] = repoMap
				}
				gitConfig["repositories"] = repos
			}
			if len(gitConfig) > 0 {
				session["spec"].(map[string]interface{})["gitConfig"] = gitConfig
			}
		}
	}

	// Add userContext if provided
	if req.UserContext != nil {
		session["spec"].(map[string]interface{})["userContext"] = map[string]interface{}{
			"userId":      req.UserContext.UserID,
			"displayName": req.UserContext.DisplayName,
			"groups":      req.UserContext.Groups,
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

	gvr := getAgenticSessionV1Alpha1Resource()
	obj := &unstructured.Unstructured{Object: session}

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
		// Determine workspace base path in PVC
		workspaceBase := req.WorkspacePath
		if strings.TrimSpace(workspaceBase) == "" {
			workspaceBase = fmt.Sprintf("/sessions/%s/workspace", name)
		}
		// Write each agent markdown
		for _, p := range strings.Split(personasCsv, ",") {
			persona := strings.TrimSpace(p)
			if persona == "" {
				continue
			}
			md, err := renderAgentMarkdownContent(persona)
			if err != nil {
				log.Printf("agent prefill: failed to render persona %s: %v", persona, err)
				continue
			}
			path := fmt.Sprintf("%s/.claude/agents/%s.md", workspaceBase, persona)
			if err := writeProjectContentFile(c, project, path, []byte(md)); err != nil {
				log.Printf("agent prefill: write failed for %s: %v", path, err)
			}
		}
	}()

	// Preferred method: provision a per-session ServiceAccount token for the runner
	if err := provisionRunnerTokenForSession(c, reqK8s, reqDyn, project, name); err != nil {
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
	gvr := getAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), sessionName, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get AgenticSession: %w", err)
	}
	ownerRef := v1.OwnerReference{
		APIVersion: obj.GetAPIVersion(),
		Kind:       obj.GetKind(),
		Name:       obj.GetName(),
		UID:        obj.GetUID(),
		Controller: boolPtr(true),
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
				Resources: []string{"agenticsessions"},
				Verbs:     []string{"get", "list", "watch", "update", "patch"},
			},
			{
				APIGroups: []string{"vteam.ambient-code"},
				Resources: []string{"agenticsessions/status"},
				Verbs:     []string{"update", "patch", "get"},
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
	if _, err := reqK8s.RbacV1().RoleBindings(project).Create(c.Request.Context(), rb, v1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("create RoleBinding: %w", err)
		}
	}

	// Mint short-lived token for the ServiceAccount
	tr := &authnv1.TokenRequest{Spec: authnv1.TokenRequestSpec{}}
	tok, err := reqK8s.CoreV1().ServiceAccounts(project).CreateToken(c.Request.Context(), saName, tr, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("mint token: %w", err)
	}
	token := tok.Status.Token
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("received empty token for SA %s", saName)
	}

	// Store token in a Secret
	secretName := fmt.Sprintf("ambient-runner-token-%s", sessionName)
	sec := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:            secretName,
			Namespace:       project,
			Labels:          map[string]string{"app": "ambient-runner-token"},
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"token": token},
	}
	if _, err := reqK8s.CoreV1().Secrets(project).Create(c.Request.Context(), sec, v1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("create Secret: %w", err)
		}
	}

	// Annotate the AgenticSession with the Secret and SA names
	meta, _ := obj.Object["metadata"].(map[string]interface{})
	if meta == nil {
		meta = map[string]interface{}{}
		obj.Object["metadata"] = meta
	}
	anns, _ := meta["annotations"].(map[string]interface{})
	if anns == nil {
		anns = map[string]interface{}{}
		meta["annotations"] = anns
	}
	anns["ambient-code.io/runner-token-secret"] = secretName
	anns["ambient-code.io/runner-sa"] = saName
	if _, err := reqDyn.Resource(gvr).Namespace(project).Update(c.Request.Context(), obj, v1.UpdateOptions{}); err != nil {
		return fmt.Errorf("annotate AgenticSession: %w", err)
	}

	return nil
}

func getSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := getK8sClientsForRequest(c)
	_ = reqK8s
	gvr := getAgenticSessionV1Alpha1Resource()

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

	session := AgenticSession{
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

// GET /api/projects/:projectName/agentic-sessions/:sessionName/messages
// Returns the messages.json content for a session by fetching from the per-project content service
// and falling back to local state directory if the content service is unavailable.
func getSessionMessages(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")

	// First try via per-namespace content service using caller's token
	data, err := readProjectContentFile(c, project, fmt.Sprintf("/sessions/%s/messages.json", sessionName))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch messages"})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// POST /api/projects/:projectName/agentic-sessions/:sessionName/messages
// Appends a user message to the session inbox (JSONL) using the per-project content service
func postSessionMessage(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")

	var body struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	entry := map[string]interface{}{
		"type":      "user_message",
		"content":   body.Content,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	inboxPath := fmt.Sprintf("/sessions/%s/inbox.jsonl", sessionName)

	// Read current inbox (best effort)
	cur, _ := readProjectContentFile(c, project, inboxPath)
	curStr := string(cur)
	if curStr != "" && !strings.HasSuffix(curStr, "\n") {
		curStr += "\n"
	}
	b, _ := json.Marshal(entry)
	newContent := curStr + string(b) + "\n"

	if err := writeProjectContentFile(c, project, inboxPath, []byte(newContent)); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to write inbox"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

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

// GET /api/projects/:projectName/agentic-sessions/:sessionName/workspace
// Lists the contents of a session's workspace by delegating to the per-project content service
func getSessionWorkspace(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")

	// Optional subpath within the workspace to list
	rel := strings.TrimSpace(c.Query("path"))
	absPath := resolveWorkspaceAbsPath(sessionName, rel)

	items, err := listProjectContent(c, project, absPath)
	if err == nil {
		// If content/list returns exactly this file (non-dir), serve file bytes
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			b, ferr := readProjectContentFile(c, project, absPath)
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
	b, ferr := readProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// GET /api/projects/:projectName/agentic-sessions/:sessionName/workspace/*path
// Reads a file from a session's workspace by delegating to the per-project content service
func getSessionWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	pathParam := c.Param("path")

	absPath := resolveWorkspaceAbsPath(sessionName, pathParam)

	// Try directory listing first to determine type
	items, err := listProjectContent(c, project, absPath)
	if err == nil {
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			// It's a file
			b, ferr := readProjectContentFile(c, project, absPath)
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
	b, ferr := readProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// PUT /api/projects/:projectName/agentic-sessions/:sessionName/workspace/*path
// Writes a file into a session's workspace via the per-project content service
func putSessionWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	pathParam := c.Param("path")

	absPath := resolveWorkspaceAbsPath(sessionName, pathParam)

	// Read raw request body and forward as-is (treat as text/binary pass-through)
	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if err := writeProjectContentFile(c, project, absPath, data); err != nil {
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

// GET /api/projects/:projectName/rfe-workflows/:id/workspace
// Lists the contents of a workflow's workspace by delegating to the per-project content service
func getRFEWorkflowWorkspace(c *gin.Context) {
	project := c.GetString("project")
	workflowID := c.Param("id")

	// Optional subpath within the workspace to list
	rel := strings.TrimSpace(c.Query("path"))
	absPath := resolveWorkflowWorkspaceAbsPath(workflowID, rel)

	items, err := listProjectContent(c, project, absPath)
	if err == nil {
		// If content/list returns exactly this file (non-dir), serve file bytes
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			b, ferr := readProjectContentFile(c, project, absPath)
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
	b, ferr := readProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// GET /api/projects/:projectName/rfe-workflows/:id/workspace/*path
// Reads a file from a workflow's workspace by delegating to the per-project content service
func getRFEWorkflowWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	workflowID := c.Param("id")
	pathParam := c.Param("path")

	absPath := resolveWorkflowWorkspaceAbsPath(workflowID, pathParam)

	// Try directory listing first to determine type
	items, err := listProjectContent(c, project, absPath)
	if err == nil {
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			// It's a file
			b, ferr := readProjectContentFile(c, project, absPath)
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
	b, ferr := readProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// PUT /api/projects/:projectName/rfe-workflows/:id/workspace/*path
// Writes a file into a workflow's workspace via the per-project content service
func putRFEWorkflowWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	workflowID := c.Param("id")
	pathParam := c.Param("path")

	absPath := resolveWorkflowWorkspaceAbsPath(workflowID, pathParam)

	// Read raw request body and forward as-is (treat as text/binary pass-through)
	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if err := writeProjectContentFile(c, project, absPath, data); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to write workspace file"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// --- Git helpers (project-scoped) ---

func stringPtr(s string) *string { return &s }

// loadGitConfigFromConfigMapForProject reads Git defaults from ConfigMap "git-config" in the project namespace
func loadGitConfigFromConfigMapForProject(c *gin.Context, reqK8s *kubernetes.Clientset, project string) (*GitConfig, error) {
	configMap, err := reqK8s.CoreV1().ConfigMaps(project).Get(c.Request.Context(), "git-config", v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get git-config ConfigMap: %v", err)
	}

	gitConfig := &GitConfig{}

	if name := configMap.Data["git-user-name"]; name != "" {
		if gitConfig.User == nil {
			gitConfig.User = &GitUser{}
		}
		gitConfig.User.Name = name
	}
	if email := configMap.Data["git-user-email"]; email != "" {
		if gitConfig.User == nil {
			gitConfig.User = &GitUser{}
		}
		gitConfig.User.Email = email
	}

	if sshKeySecret := configMap.Data["git-ssh-key-secret"]; sshKeySecret != "" {
		if gitConfig.Authentication == nil {
			gitConfig.Authentication = &GitAuthentication{}
		}
		gitConfig.Authentication.SSHKeySecret = &sshKeySecret
	}
	if tokenSecret := configMap.Data["git-token-secret"]; tokenSecret != "" {
		if gitConfig.Authentication == nil {
			gitConfig.Authentication = &GitAuthentication{}
		}
		gitConfig.Authentication.TokenSecret = &tokenSecret
	}

	if reposList := configMap.Data["git-repositories"]; reposList != "" {
		lines := strings.Split(strings.TrimSpace(reposList), "\n")
		var repos []GitRepository
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				repos = append(repos, GitRepository{URL: line, Branch: stringPtr("main")})
			}
		}
		if len(repos) > 0 {
			gitConfig.Repositories = repos
		}
	}

	return gitConfig, nil
}

// mergeGitConfigs merges user-provided GitConfig with ConfigMap defaults
func mergeGitConfigs(userConfig, defaultConfig *GitConfig) *GitConfig {
	if userConfig == nil && defaultConfig == nil {
		return nil
	}
	if userConfig == nil {
		return defaultConfig
	}
	if defaultConfig == nil {
		return userConfig
	}

	merged := &GitConfig{}
	if userConfig.User != nil {
		merged.User = userConfig.User
	} else if defaultConfig.User != nil {
		merged.User = defaultConfig.User
	}
	if userConfig.Authentication != nil {
		merged.Authentication = userConfig.Authentication
	} else if defaultConfig.Authentication != nil {
		merged.Authentication = defaultConfig.Authentication
	}

	if len(userConfig.Repositories) > 0 || len(defaultConfig.Repositories) > 0 {
		merged.Repositories = make([]GitRepository, 0, len(userConfig.Repositories)+len(defaultConfig.Repositories))
		merged.Repositories = append(merged.Repositories, userConfig.Repositories...)
		for _, def := range defaultConfig.Repositories {
			conflict := false
			for _, usr := range userConfig.Repositories {
				if usr.URL == def.URL {
					conflict = true
					break
				}
			}
			if !conflict {
				merged.Repositories = append(merged.Repositories, def)
			}
		}
	}
	return merged
}

// countArtifacts recursively counts files under the provided directory (returns 0 if missing)
func countArtifacts(artifactsDir string) int {
	info, err := os.Stat(artifactsDir)
	if err != nil || !info.IsDir() {
		return 0
	}
	var count int
	var walk func(string)
	walk = func(dir string) {
		entries, err := ioutil.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if e.IsDir() {
				walk(p)
			} else {
				count++
			}
		}
	}
	walk(artifactsDir)
	return count
}

func updateSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := getK8sClientsForRequest(c)
	_ = reqK8s

	var req CreateAgenticSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := getAgenticSessionV1Alpha1Resource()

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
	session := AgenticSession{
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

// PUT /api/projects/:projectName/agentic-sessions/:sessionName/displayname
// updateSessionDisplayName updates only the spec.displayName field on the AgenticSession
func updateSessionDisplayName(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := getK8sClientsForRequest(c)

	var req struct {
		DisplayName string `json:"displayName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := getAgenticSessionV1Alpha1Resource()

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
	session := AgenticSession{
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

func deleteSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := getK8sClientsForRequest(c)
	_ = reqK8s
	gvr := getAgenticSessionV1Alpha1Resource()

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

func cloneSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := getK8sClientsForRequest(c)

	var req CloneSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := getAgenticSessionV1Alpha1Resource()

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
	projGvr := getOpenShiftProjectResource()
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
	session := AgenticSession{
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

func startSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := getK8sClientsForRequest(c)
	_ = reqK8s
	gvr := getAgenticSessionV1Alpha1Resource()

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
	session := AgenticSession{
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

func stopSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := getK8sClientsForRequest(c)
	gvr := getAgenticSessionV1Alpha1Resource()

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
	session := AgenticSession{
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

	log.Printf("Successfully stopped agentic session %s", sessionName)
	c.JSON(http.StatusAccepted, session)
}

// PUT /api/projects/:projectName/agentic-sessions/:sessionName/status
// updateSessionStatus writes selected fields to PVC-backed files and updates CR status
func updateSessionStatus(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := getK8sClientsForRequest(c)

	var statusUpdate map[string]interface{}
	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := getAgenticSessionV1Alpha1Resource()

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

// proxyContentWrites forwards write operations to the per-namespace content service using the caller token
func proxyContentWrites(c *gin.Context, project, sessionName string, statusUpdate map[string]interface{}) error {
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		log.Printf("content proxy: skip write (no Authorization token) project=%s session=%s", project, sessionName)
		return nil
	}
	base := os.Getenv("CONTENT_SERVICE_BASE")
	if base == "" {
		base = "http://ambient-content.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, project)
	log.Printf("content proxy: preparing writes project=%s session=%s endpoint=%s tokenLen=%d", project, sessionName, endpoint, len(token))

	type writeReq struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	// Serialize writes we care about
	writes := []writeReq{}

	if msgs, ok := statusUpdate["messages"].([]interface{}); ok && len(msgs) > 0 {
		if b, err := json.Marshal(msgs); err == nil {
			writes = append(writes, writeReq{Path: fmt.Sprintf("/sessions/%s/messages.json", sessionName), Content: string(b), Encoding: "utf8"})
			delete(statusUpdate, "messages")
		}
	}
	// Always write condensed status.json
	if b, err := json.Marshal(statusUpdate); err == nil {
		writes = append(writes, writeReq{Path: fmt.Sprintf("/sessions/%s/status.json", sessionName), Content: string(b), Encoding: "utf8"})
	}

	log.Printf("content proxy: total writes=%d project=%s session=%s", len(writes), project, sessionName)

	client := &http.Client{Timeout: 10 * time.Second}
	for _, w := range writes {
		b, _ := json.Marshal(w)
		log.Printf("content proxy: POST /content/write path=%s encoding=%s contentLen=%d", w.Path, w.Encoding, len(w.Content))
		req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint+"/content/write", strings.NewReader(string(b)))
		req.Header.Set("Authorization", token)
		req.Header.Set("Content-Type", "application/json")
		if resp, err := client.Do(req); err != nil {
			log.Printf("content proxy: write failed path=%s err=%v", w.Path, err)
			continue
		} else {
			code := resp.StatusCode
			_ = resp.Body.Close()
			if code >= 200 && code < 300 {
				log.Printf("content proxy: write ok path=%s status=%d", w.Path, code)
			} else {
				log.Printf("content proxy: write non-2xx path=%s status=%d", w.Path, code)
			}
		}
	}
	return nil
}

// writeProjectContentFile writes arbitrary file content to the per-namespace content service
// using the caller's Authorization token. The path must be absolute (starts with "/").
func writeProjectContentFile(c *gin.Context, project string, absPath string, data []byte) error {
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		// Fallback to X-Forwarded-Access-Token if present
		token = c.GetHeader("X-Forwarded-Access-Token")
	}
	if !strings.HasPrefix(absPath, "/") {
		absPath = "/" + absPath
	}
	base := os.Getenv("CONTENT_SERVICE_BASE")
	if base == "" {
		base = "http://ambient-content.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, project)
	type writeReq struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	reqBody := writeReq{Path: absPath, Content: string(data), Encoding: "utf8"}
	b, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint+"/content/write", strings.NewReader(string(b)))
	if strings.TrimSpace(token) != "" {
		httpReq.Header.Set("Authorization", token)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("content write failed: status %d", resp.StatusCode)
	}
	return nil
}

// readProjectContentFile reads file content from the per-namespace content service
// using the caller's Authorization token. The path must be absolute (starts with "/").
func readProjectContentFile(c *gin.Context, project string, absPath string) ([]byte, error) {
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		// Fallback to X-Forwarded-Access-Token if present
		token = c.GetHeader("X-Forwarded-Access-Token")
	}
	if !strings.HasPrefix(absPath, "/") {
		absPath = "/" + absPath
	}
	base := os.Getenv("CONTENT_SERVICE_BASE")
	if base == "" {
		base = "http://ambient-content.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, project)
	// Normalize any accidental double slashes in path parameter
	cleanedPath := "/" + strings.TrimLeft(absPath, "/")
	u := fmt.Sprintf("%s/content/file?path=%s", endpoint, url.QueryEscape(cleanedPath))
	httpReq, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if strings.TrimSpace(token) != "" {
		httpReq.Header.Set("Authorization", token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("content read failed: status %d", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

type contentListItem struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
}

// listProjectContent lists directory entries from the per-namespace content service
func listProjectContent(c *gin.Context, project string, absPath string) ([]contentListItem, error) {
	token := c.GetHeader("Authorization")
	if !strings.HasPrefix(absPath, "/") {
		absPath = "/" + absPath
	}
	base := os.Getenv("CONTENT_SERVICE_BASE")
	if base == "" {
		base = "http://ambient-content.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, project)
	u := fmt.Sprintf("%s/content/list?path=%s", endpoint, url.QueryEscape(absPath))
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list failed: status %d", resp.StatusCode)
	}
	var out struct {
		Items []contentListItem `json:"items"`
	}
	b, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// contentWrite handles POST /content/write when running in CONTENT_SERVICE_MODE
// Body: { path: "/sessions/<name>/status.json", content: "...", encoding: "utf8"|"base64" }
func contentWrite(c *gin.Context) {
	var req struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	path := filepath.Clean("/" + strings.TrimSpace(req.Path))
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	abs := filepath.Join(stateBaseDir, path)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create directory"})
		return
	}
	var data []byte
	if strings.EqualFold(req.Encoding, "base64") {
		b, err := base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 content"})
			return
		}
		data = b
	} else {
		data = []byte(req.Content)
	}
	if err := ioutil.WriteFile(abs, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// contentRead handles GET /content/file?path=
func contentRead(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	abs := filepath.Join(stateBaseDir, path)
	b, err := ioutil.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "read failed"})
		}
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// contentList handles GET /content/list?path=
// Returns directory entries (non-recursive) under the provided path rooted at stateBaseDir
func contentList(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	abs := filepath.Join(stateBaseDir, path)
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "stat failed"})
		}
		return
	}
	if !info.IsDir() {
		// If it's a file, return single entry metadata
		c.JSON(http.StatusOK, gin.H{"items": []gin.H{{
			"name":       filepath.Base(abs),
			"path":       path,
			"isDir":      false,
			"size":       info.Size(),
			"modifiedAt": info.ModTime().UTC().Format(time.RFC3339),
		}}})
		return
	}
	entries, err := ioutil.ReadDir(abs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "readdir failed"})
		return
	}
	items := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		items = append(items, gin.H{
			"name":       e.Name(),
			"path":       filepath.Join(path, e.Name()),
			"isDir":      e.IsDir(),
			"size":       e.Size(),
			"modifiedAt": e.ModTime().UTC().Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Project management handlers
func listProjects(c *gin.Context) {
	_, reqDyn := getK8sClientsForRequest(c)

	// List OpenShift Projects the user can see; filter to Ambient-managed
	projGvr := getOpenShiftProjectResource()
	list, err := reqDyn.Resource(projGvr).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list OpenShift Projects: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
		return
	}

	toStringMap := func(in map[string]interface{}) map[string]string {
		if in == nil {
			return map[string]string{}
		}
		out := make(map[string]string, len(in))
		for k, v := range in {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
		return out
	}

	var projects []AmbientProject
	for _, item := range list.Items {
		meta, _ := item.Object["metadata"].(map[string]interface{})
		name := item.GetName()
		if name == "" && meta != nil {
			if n, ok := meta["name"].(string); ok {
				name = n
			}
		}
		labels := map[string]string{}
		annotations := map[string]string{}
		if meta != nil {
			if raw, ok := meta["labels"].(map[string]interface{}); ok {
				labels = toStringMap(raw)
			}
			if raw, ok := meta["annotations"].(map[string]interface{}); ok {
				annotations = toStringMap(raw)
			}
		}

		// Filter to Ambient-managed projects when label is present
		if v, ok := labels["ambient-code.io/managed"]; !ok || v != "true" {
			continue
		}

		displayName := annotations["openshift.io/display-name"]
		description := annotations["openshift.io/description"]
		created := item.GetCreationTimestamp().Time

		status := ""
		if st, ok := item.Object["status"].(map[string]interface{}); ok {
			if phase, ok := st["phase"].(string); ok {
				status = phase
			}
		}

		project := AmbientProject{
			Name:              name,
			DisplayName:       displayName,
			Description:       description,
			Labels:            labels,
			Annotations:       annotations,
			CreationTimestamp: created.Format(time.RFC3339),
			Status:            status,
		}
		projects = append(projects, project)
	}

	c.JSON(http.StatusOK, gin.H{"items": projects})
}

func createProject(c *gin.Context) {
	reqK8s, _ := getK8sClientsForRequest(c)
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract user info from context
	userID, hasUser := c.Get("userID")
	userName, hasName := c.Get("userName")

	// Create namespace with Ambient labels (T049: Project labeling logic)
	ns := &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: req.Name,
			Labels: map[string]string{
				"ambient-code.io/managed": "true", // Critical label for Ambient project identification
			},
			Annotations: map[string]string{
				"openshift.io/display-name": req.DisplayName,
			},
		},
	}

	// Add optional annotations
	if req.Description != "" {
		ns.Annotations["openshift.io/description"] = req.Description
	}
	// Prefer requester as user name; fallback to user ID when available
	if hasName && userName != nil {
		ns.Annotations["openshift.io/requester"] = fmt.Sprintf("%v", userName)
	} else if hasUser && userID != nil {
		ns.Annotations["openshift.io/requester"] = fmt.Sprintf("%v", userID)
	}

	created, err := reqK8s.CoreV1().Namespaces().Create(context.TODO(), ns, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create project %s: %v", req.Name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	// Do not create ProjectSettings here. The operator will reconcile when it
	// sees the managed label and create the ProjectSettings in the project namespace.

	project := AmbientProject{
		Name:              created.Name,
		DisplayName:       created.Annotations["openshift.io/display-name"],
		Description:       created.Annotations["openshift.io/description"],
		Labels:            created.Labels,
		Annotations:       created.Annotations,
		CreationTimestamp: created.CreationTimestamp.Format(time.RFC3339),
		Status:            string(created.Status.Phase),
	}

	c.JSON(http.StatusCreated, project)
}

func getProject(c *gin.Context) {
	projectName := c.Param("projectName")
	_, reqDyn := getK8sClientsForRequest(c)

	// Read OpenShift Project (user context) and validate Ambient label
	projGvr := getOpenShiftProjectResource()
	projObj, err := reqDyn.Resource(projGvr).Get(context.TODO(), projectName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		if errors.IsUnauthorized(err) || errors.IsForbidden(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to access project"})
			return
		}
		log.Printf("Failed to get OpenShift Project %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
		return
	}

	// Extract labels/annotations and validate Ambient label
	labels := map[string]string{}
	annotations := map[string]string{}
	if meta, ok := projObj.Object["metadata"].(map[string]interface{}); ok {
		if raw, ok := meta["labels"].(map[string]interface{}); ok {
			for k, v := range raw {
				if s, ok := v.(string); ok {
					labels[k] = s
				}
			}
		}
		if raw, ok := meta["annotations"].(map[string]interface{}); ok {
			for k, v := range raw {
				if s, ok := v.(string); ok {
					annotations[k] = s
				}
			}
		}
	}
	if labels["ambient-code.io/managed"] != "true" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not an Ambient project"})
		return
	}

	displayName := annotations["openshift.io/display-name"]
	description := annotations["openshift.io/description"]
	created := projObj.GetCreationTimestamp().Time
	status := ""
	if st, ok := projObj.Object["status"].(map[string]interface{}); ok {
		if phase, ok := st["phase"].(string); ok {
			status = phase
		}
	}

	project := AmbientProject{
		Name:              projectName,
		DisplayName:       displayName,
		Description:       description,
		Labels:            labels,
		Annotations:       annotations,
		CreationTimestamp: created.Format(time.RFC3339),
		Status:            status,
	}

	c.JSON(http.StatusOK, project)
}

func deleteProject(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := getK8sClientsForRequest(c)
	err := reqK8s.CoreV1().Namespaces().Delete(context.TODO(), projectName, v1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		log.Printf("Failed to delete project %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Update basic project metadata (annotations)
func updateProject(c *gin.Context) {
	projectName := c.Param("projectName")
	_, reqDyn := getK8sClientsForRequest(c)

	var req struct {
		Name        string            `json:"name"`
		DisplayName string            `json:"displayName"`
		Description string            `json:"description"`
		Annotations map[string]string `json:"annotations"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" && req.Name != projectName {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project name in URL does not match request body"})
		return
	}

	// Validate project exists and is Ambient via OpenShift Project
	projGvr := getOpenShiftProjectResource()
	projObj, err := reqDyn.Resource(projGvr).Get(context.TODO(), projectName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		log.Printf("Failed to get OpenShift Project %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get OpenShift Project"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not an Ambient project"})
		return
	}

	// Update OpenShift Project annotations for display name and description

	// Ensure metadata.annotations exists
	meta, _ := projObj.Object["metadata"].(map[string]interface{})
	if meta == nil {
		meta = map[string]interface{}{}
		projObj.Object["metadata"] = meta
	}
	anns, _ := meta["annotations"].(map[string]interface{})
	if anns == nil {
		anns = map[string]interface{}{}
		meta["annotations"] = anns
	}

	if req.DisplayName != "" {
		anns["openshift.io/display-name"] = req.DisplayName
	}
	if req.Description != "" {
		anns["openshift.io/description"] = req.Description
	}

	// Persist Project changes
	_, updateErr := reqDyn.Resource(projGvr).Update(context.TODO(), projObj, v1.UpdateOptions{})
	if updateErr != nil {
		log.Printf("Failed to update OpenShift Project %s: %v", projectName, updateErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	// Read back display/description from Project after update
	projObj, _ = reqDyn.Resource(projGvr).Get(context.TODO(), projectName, v1.GetOptions{})
	displayName := ""
	description := ""
	if projObj != nil {
		if meta, ok := projObj.Object["metadata"].(map[string]interface{}); ok {
			if anns, ok := meta["annotations"].(map[string]interface{}); ok {
				if v, ok := anns["openshift.io/display-name"].(string); ok {
					displayName = v
				}
				if v, ok := anns["openshift.io/description"].(string); ok {
					description = v
				}
			}
		}
	}

	// Extract labels/annotations and status from Project for response
	labels := map[string]string{}
	annotations := map[string]string{}
	if projObj != nil {
		if meta, ok := projObj.Object["metadata"].(map[string]interface{}); ok {
			if raw, ok := meta["labels"].(map[string]interface{}); ok {
				for k, v := range raw {
					if s, ok := v.(string); ok {
						labels[k] = s
					}
				}
			}
			if raw, ok := meta["annotations"].(map[string]interface{}); ok {
				for k, v := range raw {
					if s, ok := v.(string); ok {
						annotations[k] = s
					}
				}
			}
		}
	}
	created := projObj.GetCreationTimestamp().Time
	status := ""
	if st, ok := projObj.Object["status"].(map[string]interface{}); ok {
		if phase, ok := st["phase"].(string); ok {
			status = phase
		}
	}

	project := AmbientProject{
		Name:              projectName,
		DisplayName:       displayName,
		Description:       description,
		Labels:            labels,
		Annotations:       annotations,
		CreationTimestamp: created.Format(time.RFC3339),
		Status:            status,
	}

	c.JSON(http.StatusOK, project)
}

// Project settings endpoints removed in favor of native RBAC RoleBindings approach

// Group management via RoleBindings
const (
	ambientRoleAdmin = "ambient-project-admin"
	ambientRoleEdit  = "ambient-project-edit"
	ambientRoleView  = "ambient-project-view"
)

func sanitizeName(input string) string {
	s := strings.ToLower(input)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else {
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
		if b.Len() >= 63 {
			break
		}
	}
	out := b.String()
	out = strings.Trim(out, "-")
	if out == "" {
		out = "group"
	}
	return out
}

// Unified permissions (users and groups)
type PermissionAssignment struct {
	SubjectType string `json:"subjectType"`
	SubjectName string `json:"subjectName"`
	Role        string `json:"role"`
}

// GET /api/projects/:projectName/permissions
func listProjectPermissions(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := getK8sClientsForRequest(c)

	// Prefer new label, but also include legacy group-access for backward-compat listing
	rbsAll, err := reqK8s.RbacV1().RoleBindings(projectName).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list RoleBindings in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list permissions"})
		return
	}

	validRoles := map[string]string{
		ambientRoleAdmin: "admin",
		ambientRoleEdit:  "edit",
		ambientRoleView:  "view",
	}

	type key struct{ kind, name, role string }
	seen := map[key]struct{}{}
	assignments := []PermissionAssignment{}

	for _, rb := range rbsAll.Items {
		// Filter to Ambient-managed permission rolebindings
		if rb.Labels["app"] != "ambient-permission" && rb.Labels["app"] != "ambient-group-access" {
			continue
		}

		// Determine role from RoleRef or annotation
		role := ""
		if r, ok := validRoles[rb.RoleRef.Name]; ok && rb.RoleRef.Kind == "ClusterRole" {
			role = r
		}
		if annRole := rb.Annotations["ambient-code.io/role"]; annRole != "" {
			role = strings.ToLower(annRole)
		}
		if role == "" {
			continue
		}

		for _, sub := range rb.Subjects {
			if !strings.EqualFold(sub.Kind, "Group") && !strings.EqualFold(sub.Kind, "User") {
				continue
			}
			subjectType := "group"
			if strings.EqualFold(sub.Kind, "User") {
				subjectType = "user"
			}
			subjectName := sub.Name
			if v := rb.Annotations["ambient-code.io/subject-name"]; v != "" {
				subjectName = v
			}
			if v := rb.Annotations["ambient-code.io/groupName"]; v != "" && subjectType == "group" {
				subjectName = v
			}

			k := key{kind: subjectType, name: subjectName, role: role}
			if _, exists := seen[k]; exists {
				continue
			}
			seen[k] = struct{}{}
			assignments = append(assignments, PermissionAssignment{SubjectType: subjectType, SubjectName: subjectName, Role: role})
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": assignments})
}

// POST /api/projects/:projectName/permissions
func addProjectPermission(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := getK8sClientsForRequest(c)

	var req struct {
		SubjectType string `json:"subjectType" binding:"required"`
		SubjectName string `json:"subjectName" binding:"required"`
		Role        string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	st := strings.ToLower(strings.TrimSpace(req.SubjectType))
	if st != "group" && st != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subjectType must be one of: group, user"})
		return
	}
	subjectKind := "Group"
	if st == "user" {
		subjectKind = "User"
	}

	roleRefName := ""
	switch strings.ToLower(req.Role) {
	case "admin":
		roleRefName = ambientRoleAdmin
	case "edit":
		roleRefName = ambientRoleEdit
	case "view":
		roleRefName = ambientRoleView
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be one of: admin, edit, view"})
		return
	}

	rbName := "ambient-permission-" + strings.ToLower(req.Role) + "-" + sanitizeName(req.SubjectName) + "-" + st
	rb := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      rbName,
			Namespace: projectName,
			Labels: map[string]string{
				"app": "ambient-permission",
			},
			Annotations: map[string]string{
				"ambient-code.io/subject-kind": subjectKind,
				"ambient-code.io/subject-name": req.SubjectName,
				"ambient-code.io/role":         strings.ToLower(req.Role),
			},
		},
		RoleRef:  rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: roleRefName},
		Subjects: []rbacv1.Subject{{Kind: subjectKind, APIGroup: "rbac.authorization.k8s.io", Name: req.SubjectName}},
	}

	if _, err := reqK8s.RbacV1().RoleBindings(projectName).Create(context.TODO(), rb, v1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "permission already exists for this subject and role"})
			return
		}
		log.Printf("Failed to create RoleBinding in %s for %s %s: %v", projectName, st, req.SubjectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant permission"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "permission added"})
}

// DELETE /api/projects/:projectName/permissions/:subjectType/:subjectName
func removeProjectPermission(c *gin.Context) {
	projectName := c.Param("projectName")
	subjectType := strings.ToLower(c.Param("subjectType"))
	subjectName := c.Param("subjectName")
	reqK8s, _ := getK8sClientsForRequest(c)

	if subjectType != "group" && subjectType != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subjectType must be one of: group, user"})
		return
	}
	if strings.TrimSpace(subjectName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subjectName is required"})
		return
	}

	rbs, err := reqK8s.RbacV1().RoleBindings(projectName).List(context.TODO(), v1.ListOptions{LabelSelector: "app=ambient-permission"})
	if err != nil {
		log.Printf("Failed to list RoleBindings in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove permission"})
		return
	}

	for _, rb := range rbs.Items {
		for _, sub := range rb.Subjects {
			if strings.EqualFold(sub.Kind, "Group") && subjectType == "group" && sub.Name == subjectName {
				_ = reqK8s.RbacV1().RoleBindings(projectName).Delete(context.TODO(), rb.Name, v1.DeleteOptions{})
				break
			}
			if strings.EqualFold(sub.Kind, "User") && subjectType == "user" && sub.Name == subjectName {
				_ = reqK8s.RbacV1().RoleBindings(projectName).Delete(context.TODO(), rb.Name, v1.DeleteOptions{})
				break
			}
		}
	}

	c.Status(http.StatusNoContent)
}

// Webhook handlers - placeholder implementations
// Access key management: list/create/delete keys stored as Secrets with hashed value
func listProjectKeys(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := getK8sClientsForRequest(c)

	// List ServiceAccounts with label app=ambient-access-key
	sas, err := reqK8s.CoreV1().ServiceAccounts(projectName).List(context.TODO(), v1.ListOptions{LabelSelector: "app=ambient-access-key"})
	if err != nil {
		log.Printf("Failed to list access keys in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list access keys"})
		return
	}

	// Map ServiceAccount -> role by scanning RoleBindings with the same label
	roleBySA := map[string]string{}
	if rbs, err := reqK8s.RbacV1().RoleBindings(projectName).List(context.TODO(), v1.ListOptions{LabelSelector: "app=ambient-access-key"}); err == nil {
		for _, rb := range rbs.Items {
			role := strings.ToLower(rb.Annotations["ambient-code.io/role"])
			if role == "" {
				switch rb.RoleRef.Name {
				case ambientRoleAdmin:
					role = "admin"
				case ambientRoleEdit:
					role = "edit"
				case ambientRoleView:
					role = "view"
				}
			}
			for _, sub := range rb.Subjects {
				if strings.EqualFold(sub.Kind, "ServiceAccount") {
					roleBySA[sub.Name] = role
				}
			}
		}
	}

	type KeyInfo struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		CreatedAt   string `json:"createdAt"`
		LastUsedAt  string `json:"lastUsedAt"`
		Description string `json:"description,omitempty"`
		Role        string `json:"role,omitempty"`
	}

	items := []KeyInfo{}
	for _, sa := range sas.Items {
		ki := KeyInfo{ID: sa.Name, Name: sa.Annotations["ambient-code.io/key-name"], Description: sa.Annotations["ambient-code.io/description"], Role: roleBySA[sa.Name]}
		if t := sa.CreationTimestamp; !t.IsZero() {
			ki.CreatedAt = t.Time.Format(time.RFC3339)
		}
		if lu := sa.Annotations["ambient-code.io/last-used-at"]; lu != "" {
			ki.LastUsedAt = lu
		}
		items = append(items, ki)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func createProjectKey(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := getK8sClientsForRequest(c)

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Role        string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine role to bind; default edit
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role == "" {
		role = "edit"
	}
	var roleRefName string
	switch role {
	case "admin":
		roleRefName = ambientRoleAdmin
	case "edit":
		roleRefName = ambientRoleEdit
	case "view":
		roleRefName = ambientRoleView
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be one of: admin, edit, view"})
		return
	}

	// Create a dedicated ServiceAccount per key
	ts := time.Now().Unix()
	saName := fmt.Sprintf("ambient-key-%s-%d", sanitizeName(req.Name), ts)
	sa := &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      saName,
			Namespace: projectName,
			Labels:    map[string]string{"app": "ambient-access-key"},
			Annotations: map[string]string{
				"ambient-code.io/key-name":    req.Name,
				"ambient-code.io/description": req.Description,
				"ambient-code.io/created-at":  time.Now().Format(time.RFC3339),
				"ambient-code.io/role":        role,
			},
		},
	}
	if _, err := reqK8s.CoreV1().ServiceAccounts(projectName).Create(context.TODO(), sa, v1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		log.Printf("Failed to create ServiceAccount %s in %s: %v", saName, projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service account"})
		return
	}

	// Bind the SA to the selected role via RoleBinding
	rbName := fmt.Sprintf("ambient-key-%s-%s-%d", role, sanitizeName(req.Name), ts)
	rb := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      rbName,
			Namespace: projectName,
			Labels:    map[string]string{"app": "ambient-access-key"},
			Annotations: map[string]string{
				"ambient-code.io/key-name": req.Name,
				"ambient-code.io/sa-name":  saName,
				"ambient-code.io/role":     role,
			},
		},
		RoleRef:  rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: roleRefName},
		Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: saName, Namespace: projectName}},
	}
	if _, err := reqK8s.RbacV1().RoleBindings(projectName).Create(context.TODO(), rb, v1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		log.Printf("Failed to create RoleBinding %s in %s: %v", rbName, projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bind service account"})
		return
	}

	// Issue a one-time JWT token for this ServiceAccount
	tr := &authnv1.TokenRequest{Spec: authnv1.TokenRequestSpec{}}
	tok, err := reqK8s.CoreV1().ServiceAccounts(projectName).CreateToken(context.TODO(), saName, tr, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create token for SA %s/%s: %v", projectName, saName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          saName,
		"name":        req.Name,
		"key":         tok.Status.Token,
		"description": req.Description,
		"role":        role,
		"lastUsedAt":  "",
	})
}

func deleteProjectKey(c *gin.Context) {
	projectName := c.Param("projectName")
	keyID := c.Param("keyId")
	reqK8s, _ := getK8sClientsForRequest(c)

	// Delete associated RoleBindings
	rbs, _ := reqK8s.RbacV1().RoleBindings(projectName).List(context.TODO(), v1.ListOptions{LabelSelector: "app=ambient-access-key"})
	for _, rb := range rbs.Items {
		if rb.Annotations["ambient-code.io/sa-name"] == keyID {
			_ = reqK8s.RbacV1().RoleBindings(projectName).Delete(context.TODO(), rb.Name, v1.DeleteOptions{})
		}
	}

	// Delete the ServiceAccount itself
	if err := reqK8s.CoreV1().ServiceAccounts(projectName).Delete(context.TODO(), keyID, v1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			log.Printf("Failed to delete service account %s in %s: %v", keyID, projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete access key"})
			return
		}
	}

	c.Status(http.StatusNoContent)
}

// handleJiraWebhook removed; use standard session creation endpoint instead

// Metrics handler - placeholder implementation
func getMetrics(c *gin.Context) {
	// TODO: Implement Prometheus metrics
	metrics := `# HELP agenticsession_total Total number of agentic sessions
# TYPE agenticsession_total counter
agenticsession_total 0
`
	c.String(http.StatusOK, metrics)
}

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

	// Parse repositories array if present
	if repos, ok := spec["repositories"].([]interface{}); ok {
		wf.Repositories = make([]GitRepository, 0, len(repos))
		for _, r := range repos {
			if rm, ok := r.(map[string]interface{}); ok {
				repo := GitRepository{}
				if u, ok := rm["url"].(string); ok {
					repo.URL = u
				}
				if b, ok := rm["branch"].(string); ok && strings.TrimSpace(b) != "" {
					repo.Branch = stringPtr(b)
				}
				if cp, ok := rm["clonePath"].(string); ok && strings.TrimSpace(cp) != "" {
					repo.ClonePath = stringPtr(cp)
				}
				wf.Repositories = append(wf.Repositories, repo)
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
		if len(w.Repositories) > 0 {
			repos := make([]map[string]interface{}, 0, len(w.Repositories))
			for _, r := range w.Repositories {
				rm := map[string]interface{}{"url": r.URL}
				if r.Branch != nil {
					rm["branch"] = *r.Branch
				}
				if r.ClonePath != nil {
					rm["clonePath"] = *r.ClonePath
				}
				repos = append(repos, rm)
			}
			item["repositories"] = repos
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
		ID:            workflowID,
		Title:         req.Title,
		Description:   req.Description,
		Repositories:  req.Repositories,
		WorkspacePath: req.WorkspacePath,
		Project:       project,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_, reqDyn := getK8sClientsForRequest(c)
	if err := upsertProjectRFEWorkflowCR(reqDyn, workflow); err != nil {
		log.Printf("⚠️ Failed to upsert RFEWorkflow CR: %v", err)
	}

	// Initialize workspace structure and optionally seed repositories
	workspaceRoot := resolveWorkflowWorkspaceAbsPath(workflowID, "")

	// Initialize Spec Kit template into workspace (version via SPEC_KIT_VERSION)
	if err := initSpecKitInWorkspace(c, project, workspaceRoot); err != nil {
		log.Printf("spec-kit init failed for %s/%s: %v", project, workflowID, err)
	}

	// Clone repositories into workspace (full repo contents); preserve dot-prefixed paths
	for _, r := range workflow.Repositories {
		targetDir := ""
		if r.ClonePath != nil && strings.TrimSpace(*r.ClonePath) != "" {
			targetDir = *r.ClonePath
		} else {
			name := filepath.Base(strings.TrimSuffix(strings.TrimSuffix(r.URL, ".git"), "/"))
			targetDir = filepath.Join("repos", name)
		}
		absTarget := filepath.Join(workspaceRoot, targetDir)

		// Ensure target directory exists in content service
		_ = writeProjectContentFile(c, project, filepath.Join(absTarget, ".keep"), []byte(""))

		// Perform shallow clone to a temp dir on backend container filesystem
		tmpDir, terr := os.MkdirTemp("", "clone-*")
		if terr != nil {
			log.Printf("repo clone: temp dir failed for %s: %v", r.URL, terr)
			continue
		}
		defer os.RemoveAll(tmpDir)

		// Use git CLI for shallow clone
		args := []string{"clone", "--depth", "1"}
		if r.Branch != nil && strings.TrimSpace(*r.Branch) != "" {
			args = append(args, "--branch", strings.TrimSpace(*r.Branch))
		}
		args = append(args, r.URL, tmpDir)
		cmd := exec.Command("git", args...)
		cmd.Env = os.Environ()
		if out, cerr := cmd.CombinedOutput(); cerr != nil {
			log.Printf("repo clone failed: %s: %v output=%s", r.URL, cerr, string(out))
			continue
		}

		// Walk cloned files and write each to content service (skip .git directory)
		_ = filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, rerr := filepath.Rel(tmpDir, path)
			if rerr != nil {
				return nil
			}
			unixRel := strings.ReplaceAll(rel, "\\", "/")
			// skip git metadata and root
			if unixRel == "." || strings.HasPrefix(unixRel, ".git/") || unixRel == ".git" {
				return nil
			}
			if d.IsDir() {
				// ensure directory exists by placing a marker (harmless if overwritten later)
				_ = writeProjectContentFile(c, project, filepath.Join(absTarget, unixRel, ".keep"), []byte(""))
				return nil
			}
			// file: read and write
			b, rerr2 := os.ReadFile(path)
			if rerr2 != nil {
				return nil
			}
			if werr := writeProjectContentFile(c, project, filepath.Join(absTarget, unixRel), b); werr != nil {
				log.Printf("repo write failed: %s -> %s: %v", path, filepath.Join(absTarget, unixRel), werr)
			}
			return nil
		})
	}

	// Best-effort prefill of all agent markdown into workflow workspace for immediate UI availability
	func() {
		defer func() { _ = recover() }()
		agentsBase := filepath.Join(workspaceRoot, ".claude", "agents")
		_ = writeProjectContentFile(c, project, filepath.Join(agentsBase, ".keep"), []byte(""))
		dir := resolveAgentsDir()
		agents, err := readAllAgentYAMLs(dir)
		if err != nil {
			log.Printf("agent prefill: failed to read agents: %v", err)
			return
		}
		for _, a := range agents {
			persona := extractPersonaFromName(a.Name)
			if persona == "" {
				continue
			}
			md, err := renderAgentMarkdownContent(persona)
			if err != nil {
				log.Printf("agent prefill: failed to render persona %s: %v", persona, err)
				continue
			}
			path := fmt.Sprintf("%s/%s.md", agentsBase, persona)
			if err := writeProjectContentFile(c, project, path, []byte(md)); err != nil {
				log.Printf("agent prefill: write failed for %s: %v", path, err)
			}
		}
	}()

	c.JSON(http.StatusCreated, workflow)
}

// initSpecKitInWorkspace downloads a Spec Kit template zip and writes its contents into the workflow workspace
// SPEC_KIT_VERSION env var controls version tag (e.g., v0.0.50). Template assumed: spec-kit-template-claude-sh-<ver>.zip
func initSpecKitInWorkspace(c *gin.Context, project, workspaceRoot string) error {
	version := strings.TrimSpace(os.Getenv("SPEC_KIT_VERSION"))
	if version == "" {
		version = "v0.0.50"
	}
	tmplName := strings.TrimSpace(os.Getenv("SPEC_KIT_TEMPLATE_NAME"))
	if tmplName == "" {
		tmplName = "spec-kit-template-claude-sh"
	}
	url := fmt.Sprintf("https://github.com/github/spec-kit/releases/download/%s/%s-%s.zip", version, tmplName, version)

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download spec-kit template failed: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	// Extract files
	total := len(zr.File)
	var filesWritten, skippedDirs, openErrors, readErrors, writeErrors int
	log.Printf("initSpecKitInWorkspace: extracting spec-kit template: %d entries", total)
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			skippedDirs++
			log.Printf("spec-kit: skipping directory: %s", f.Name)
			continue
		}
		rc, err := f.Open()
		if err != nil {
			openErrors++
			log.Printf("spec-kit: open failed: %s: %v", f.Name, err)
			continue
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			readErrors++
			log.Printf("spec-kit: read failed: %s: %v", f.Name, err)
			continue
		}
		// Normalize path: keep leading dots intact; only trim explicit "./" prefix
		rel := f.Name
		origRel := rel
		rel = strings.TrimPrefix(rel, "./")
		// Ensure we do not write outside workspace
		rel = strings.ReplaceAll(rel, "\\", "/")
		for strings.Contains(rel, "../") {
			rel = strings.ReplaceAll(rel, "../", "")
		}
		if rel != origRel {
			log.Printf("spec-kit: normalized path %q -> %q", origRel, rel)
		}
		target := filepath.Join(workspaceRoot, rel)
		if err := writeProjectContentFile(c, project, target, b); err != nil {
			writeErrors++
			log.Printf("write spec-kit file failed: %s: %v", target, err)
		} else {
			filesWritten++
			log.Printf("spec-kit: wrote %s (%d bytes)", target, len(b))
		}
	}
	log.Printf("initSpecKitInWorkspace: extraction summary: written=%d, skipped_dirs=%d, open_errors=%d, read_errors=%d, write_errors=%d", filesWritten, skippedDirs, openErrors, readErrors, writeErrors)
	return nil
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
	if len(wf.JiraLinks) > 0 {
		links := make([]map[string]interface{}, 0, len(wf.JiraLinks))
		for _, l := range wf.JiraLinks {
			links = append(links, map[string]interface{}{"path": l.Path, "jiraKey": l.JiraKey})
		}
		resp["jiraLinks"] = links
	}
	if len(wf.Repositories) > 0 {
		repos := make([]map[string]interface{}, 0, len(wf.Repositories))
		for _, r := range wf.Repositories {
			rm := map[string]interface{}{"url": r.URL}
			if r.Branch != nil {
				rm["branch"] = *r.Branch
			}
			if r.ClonePath != nil {
				rm["clonePath"] = *r.ClonePath
			}
			repos = append(repos, rm)
		}
		resp["repositories"] = repos
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/projects/:projectName/rfe-workflows/:id/summary
// Computes derived phase/status and progress based on workspace files and linked sessions
func getProjectRFEWorkflowSummary(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	// Determine workspace and expected files
	workspaceRoot := resolveWorkflowWorkspaceAbsPath(id, "")
	specsPath := filepath.Join(workspaceRoot, "specs")
	specsItems, _ := listProjectContent(c, project, specsPath)

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
					subItems, _ := listProjectContent(c, project, filepath.Join(specsPath, it.Name))
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

// POST /api/projects/:projectName/rfe-workflows/:id/jira { path }
// Creates a Jira issue from a workspace file and updates the RFEWorkflow CR with the linkage
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
	// Reuse listRunnerSecrets helpers indirectly by reading the Secret directly
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

	// Load workflow for title
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

	// Read file content
	absPath := resolveWorkflowWorkspaceAbsPath(id, req.Path)
	b, ferr := readProjectContentFile(c, project, absPath)
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
			issueType = "Feature"  // plan.md creates Features for now (was Epic)
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

// Runner secrets management
// Config is stored in ProjectSettings.spec.runnerSecretsName
// The Secret lives in the project namespace and stores key/value pairs for runners

// GET /api/projects/:projectName/secrets -> { items: [{name, createdAt}] }
func listNamespaceSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := getK8sClientsForRequest(c)

	list, err := reqK8s.CoreV1().Secrets(projectName).List(c.Request.Context(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list secrets in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list secrets"})
		return
	}

	type Item struct {
		Name      string `json:"name"`
		CreatedAt string `json:"createdAt,omitempty"`
		Type      string `json:"type"`
	}
	items := []Item{}
	for _, s := range list.Items {
		// Only include runner/session secrets: Opaque + annotated
		if s.Type != corev1.SecretTypeOpaque {
			continue
		}
		if s.Annotations == nil || s.Annotations["ambient-code.io/runner-secret"] != "true" {
			continue
		}
		it := Item{Name: s.Name, Type: string(s.Type)}
		if !s.CreationTimestamp.IsZero() {
			it.CreatedAt = s.CreationTimestamp.Time.Format(time.RFC3339)
		}
		items = append(items, it)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GET /api/projects/:projectName/runner-secrets/config
func getRunnerSecretsConfig(c *gin.Context) {
	projectName := c.Param("projectName")
	_, reqDyn := getK8sClientsForRequest(c)

	gvr := getProjectSettingsResource()
	// ProjectSettings is a singleton per namespace named 'projectsettings'
	obj, err := reqDyn.Resource(gvr).Namespace(projectName).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to read ProjectSettings for %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets config"})
		return
	}

	secretName := ""
	if obj != nil {
		if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
			if v, ok := spec["runnerSecretsName"].(string); ok {
				secretName = v
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"secretName": secretName})
}

// PUT /api/projects/:projectName/runner-secrets/config { secretName }
func updateRunnerSecretsConfig(c *gin.Context) {
	projectName := c.Param("projectName")
	_, reqDyn := getK8sClientsForRequest(c)

	var req struct {
		SecretName string `json:"secretName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.SecretName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "secretName is required"})
		return
	}

	// Operator owns ProjectSettings. If it exists, update; otherwise, return not found.
	gvr := getProjectSettingsResource()
	obj, err := reqDyn.Resource(gvr).Namespace(projectName).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if errors.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "ProjectSettings not found. Ensure the namespace is labeled ambient-code.io/managed=true and wait for operator."})
		return
	}
	if err != nil {
		log.Printf("Failed to read ProjectSettings for %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets config"})
		return
	}

	// Update spec.runnerSecretsName
	spec, _ := obj.Object["spec"].(map[string]interface{})
	if spec == nil {
		spec = map[string]interface{}{}
		obj.Object["spec"] = spec
	}
	spec["runnerSecretsName"] = req.SecretName

	if _, err := reqDyn.Resource(gvr).Namespace(projectName).Update(c.Request.Context(), obj, v1.UpdateOptions{}); err != nil {
		log.Printf("Failed to update ProjectSettings for %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update runner secrets config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"secretName": req.SecretName})
}

// GET /api/projects/:projectName/runner-secrets -> { data: { key: value } }
func listRunnerSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, reqDyn := getK8sClientsForRequest(c)

	// Read config
	gvr := getProjectSettingsResource()
	obj, err := reqDyn.Resource(gvr).Namespace(projectName).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to read ProjectSettings for %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets config"})
		return
	}
	secretName := ""
	if obj != nil {
		if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
			if v, ok := spec["runnerSecretsName"].(string); ok {
				secretName = v
			}
		}
	}
	if secretName == "" {
		c.JSON(http.StatusOK, gin.H{"data": map[string]string{}})
		return
	}

	sec, err := reqK8s.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"data": map[string]string{}})
			return
		}
		log.Printf("Failed to get Secret %s/%s: %v", projectName, secretName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets"})
		return
	}

	out := map[string]string{}
	for k, v := range sec.Data {
		out[k] = string(v)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// PUT /api/projects/:projectName/runner-secrets { data: { key: value } }
func updateRunnerSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, reqDyn := getK8sClientsForRequest(c)

	var req struct {
		Data map[string]string `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Read config for secret name
	gvr := getProjectSettingsResource()
	obj, err := reqDyn.Resource(gvr).Namespace(projectName).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to read ProjectSettings for %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets config"})
		return
	}
	secretName := ""
	if obj != nil {
		if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
			if v, ok := spec["runnerSecretsName"].(string); ok {
				secretName = strings.TrimSpace(v)
			}
		}
	}
	if secretName == "" {
		secretName = "ambient-runner-secrets"
	}

	// Do not create/update ProjectSettings here. The operator owns it.

	// Try to get existing Secret
	sec, err := reqK8s.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if errors.IsNotFound(err) {
		// Create new Secret
		newSec := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      secretName,
				Namespace: projectName,
				Labels:    map[string]string{"app": "ambient-runner-secrets"},
				Annotations: map[string]string{
					"ambient-code.io/runner-secret": "true",
				},
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: req.Data,
		}
		if _, err := reqK8s.CoreV1().Secrets(projectName).Create(c.Request.Context(), newSec, v1.CreateOptions{}); err != nil {
			log.Printf("Failed to create Secret %s/%s: %v", projectName, secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create runner secrets"})
			return
		}
	} else if err != nil {
		log.Printf("Failed to get Secret %s/%s: %v", projectName, secretName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets"})
		return
	} else {
		// Update existing - replace Data
		sec.Type = corev1.SecretTypeOpaque
		sec.Data = map[string][]byte{}
		for k, v := range req.Data {
			sec.Data[k] = []byte(v)
		}
		if _, err := reqK8s.CoreV1().Secrets(projectName).Update(c.Request.Context(), sec, v1.UpdateOptions{}); err != nil {
			log.Printf("Failed to update Secret %s/%s: %v", projectName, secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update runner secrets"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "runner secrets updated"})
}
