package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"ambient-code-backend/internal/middleware"
	"ambient-code-backend/internal/services"
	"ambient-code-backend/internal/types"

	"github.com/gin-gonic/gin"
	authnv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ListSessions lists all agentic sessions in a project
func ListSessions(c *gin.Context) {
	project := c.GetString("project")
	reqK8s, reqDyn := middleware.GetK8sClientsForRequest(c)
	_ = reqK8s
	gvr := types.GetAgenticSessionV1Alpha1Resource()

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
			session.Spec = services.ParseSpec(spec)
		}

		if status, ok := item.Object["status"].(map[string]interface{}); ok {
			session.Status = services.ParseStatus(status)
		}

		sessions = append(sessions, session)
	}

	c.JSON(http.StatusOK, gin.H{"items": sessions})
}

// CreateSession creates a new agentic session
func CreateSession(c *gin.Context) {
	project := c.GetString("project")
	reqK8s, reqDyn := middleware.GetK8sClientsForRequest(c)
	_ = reqK8s
	var req types.CreateAgenticSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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
	if defaultGitConfig, err := services.LoadGitConfigFromConfigMapForProject(c, reqK8s, project); err != nil {
		log.Printf("Warning: failed to load Git config from ConfigMap in %s: %v", project, err)
	} else {
		mergedGitConfig := services.MergeGitConfigs(req.GitConfig, defaultGitConfig)
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

	gvr := types.GetAgenticSessionV1Alpha1Resource()
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
			// TODO: Import renderAgentMarkdownContent from agents package
			md := fmt.Sprintf("# Agent: %s\n\nAgent content for %s", persona, persona)
			path := fmt.Sprintf("%s/.claude/agents/%s.md", workspaceBase, persona)
			if err := services.WriteProjectContentFile(c, project, path, []byte(md)); err != nil {
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
	gvr := types.GetAgenticSessionV1Alpha1Resource()
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
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:            secretName,
			Namespace:       project,
			Labels:          map[string]string{"app": "ambient-runner-token"},
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"token": token},
	}
	if _, err := reqK8s.CoreV1().Secrets(project).Create(c.Request.Context(), secret, v1.CreateOptions{}); err != nil {
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

// GetSession gets a specific agentic session
func GetSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := middleware.GetK8sClientsForRequest(c)
	_ = reqK8s

	gvr := types.GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	session := types.AgenticSession{
		APIVersion: obj.GetAPIVersion(),
		Kind:       obj.GetKind(),
		Metadata:   obj.Object["metadata"].(map[string]interface{}),
	}

	if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
		session.Spec = services.ParseSpec(spec)
	}

	if status, ok := obj.Object["status"].(map[string]interface{}); ok {
		session.Status = services.ParseStatus(status)
	}

	c.JSON(http.StatusOK, session)
}

// UpdateSession updates a specific agentic session
func UpdateSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := middleware.GetK8sClientsForRequest(c)
	_ = reqK8s

	var req types.UpdateAgenticSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := types.GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Update fields if provided
	spec := obj.Object["spec"].(map[string]interface{})
	if req.DisplayName != nil {
		spec["displayName"] = *req.DisplayName
	}
	if req.LLMSettings != nil {
		llmSettings := spec["llmSettings"].(map[string]interface{})
		if req.LLMSettings.Model != "" {
			llmSettings["model"] = req.LLMSettings.Model
		}
		if req.LLMSettings.Temperature != 0 {
			llmSettings["temperature"] = req.LLMSettings.Temperature
		}
		if req.LLMSettings.MaxTokens != 0 {
			llmSettings["maxTokens"] = req.LLMSettings.MaxTokens
		}
	}
	if req.Timeout != nil {
		spec["timeout"] = *req.Timeout
	}

	_, err = reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agentic session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agentic session updated successfully"})
}

// DeleteSession deletes a specific agentic session
func DeleteSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := middleware.GetK8sClientsForRequest(c)
	_ = reqK8s

	gvr := types.GetAgenticSessionV1Alpha1Resource()
	err := reqDyn.Resource(gvr).Namespace(project).Delete(context.TODO(), sessionName, v1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to delete agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agentic session"})
		return
	}

	c.Status(http.StatusNoContent)
}

// CloneSession creates a clone of an existing agentic session
func CloneSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := middleware.GetK8sClientsForRequest(c)
	_ = reqK8s

	var req types.CloneAgenticSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := types.GetAgenticSessionV1Alpha1Resource()
	original, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Original agentic session not found"})
			return
		}
		log.Printf("Failed to get original agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get original agentic session"})
		return
	}

	// Create new session based on original
	timestamp := time.Now().Unix()
	newName := fmt.Sprintf("agentic-session-%d", timestamp)
	if req.Name != "" {
		newName = req.Name
	}

	clone := map[string]interface{}{
		"apiVersion": original.Object["apiVersion"],
		"kind":       original.Object["kind"],
		"metadata": map[string]interface{}{
			"name":      newName,
			"namespace": project,
		},
		"spec":   original.Object["spec"],
		"status": map[string]interface{}{"phase": "Pending"},
	}

	// Override display name if provided
	if req.DisplayName != "" {
		spec := clone["spec"].(map[string]interface{})
		spec["displayName"] = req.DisplayName
	}

	obj := &unstructured.Unstructured{Object: clone}
	created, err := reqDyn.Resource(gvr).Namespace(project).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create cloned agentic session in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cloned agentic session"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Agentic session cloned successfully",
		"name":    newName,
		"uid":     created.GetUID(),
	})
}

// StartSession starts an agentic session
func StartSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := middleware.GetK8sClientsForRequest(c)
	_ = reqK8s

	gvr := types.GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Update status to Running
	status := obj.Object["status"].(map[string]interface{})
	status["phase"] = "Running"
	status["startedAt"] = time.Now().UTC().Format(time.RFC3339)

	_, err = reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to start agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start agentic session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agentic session started successfully"})
}

// StopSession stops an agentic session
func StopSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := middleware.GetK8sClientsForRequest(c)
	_ = reqK8s

	gvr := types.GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Update status to Stopped
	status := obj.Object["status"].(map[string]interface{})
	status["phase"] = "Stopped"
	status["stoppedAt"] = time.Now().UTC().Format(time.RFC3339)

	_, err = reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to stop agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop agentic session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agentic session stopped successfully"})
}

// UpdateSessionStatus updates the status of an agentic session
func UpdateSessionStatus(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := middleware.GetK8sClientsForRequest(c)
	_ = reqK8s

	var statusUpdate map[string]interface{}
	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := types.GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Ensure status map exists
	if obj.Object["status"] == nil {
		obj.Object["status"] = make(map[string]interface{})
	}
	status := obj.Object["status"].(map[string]interface{})

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

	_, err = reqDyn.Resource(gvr).Namespace(project).UpdateStatus(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update status of agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session status updated successfully"})
}

// UpdateSessionDisplayName updates the display name of an agentic session
func UpdateSessionDisplayName(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := middleware.GetK8sClientsForRequest(c)
	_ = reqK8s

	var req struct {
		DisplayName string `json:"displayName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := types.GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agentic session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Update display name
	spec := obj.Object["spec"].(map[string]interface{})
	spec["displayName"] = req.DisplayName

	_, err = reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update display name of agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session display name"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session display name updated successfully"})
}

// GetSessionMessages gets messages for an agentic session
// Returns the messages.json content for a session by fetching from the per-project content service
func GetSessionMessages(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")

	// First try via per-namespace content service using caller's token
	data, err := services.ReadProjectContentFile(c, project, fmt.Sprintf("/sessions/%s/messages.json", sessionName))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch messages"})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// PostSessionMessage posts a message to an agentic session
// Appends a user message to the session inbox (JSONL) using the per-project content service
func PostSessionMessage(c *gin.Context) {
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
	cur, _ := services.ReadProjectContentFile(c, project, inboxPath)
	curStr := string(cur)
	if curStr != "" && !strings.HasSuffix(curStr, "\n") {
		curStr += "\n"
	}
	b, _ := json.Marshal(entry)
	newContent := curStr + string(b) + "\n"

	if err := services.WriteProjectContentFile(c, project, inboxPath, []byte(newContent)); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to write inbox"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
