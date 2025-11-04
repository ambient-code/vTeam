// Package handlers provides HTTP handlers for the backend API.
// This file contains CRUD operations for agentic sessions.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ListSessions lists all agentic sessions in the project namespace.
// V2 API Handler for multi-tenant session management.
func ListSessions(c *gin.Context) {
	project := c.GetString("project")
	_, reqDyn := GetK8sClientsForRequest(c)
	gvr := GetAgenticSessionV1Alpha1Resource()

	list, err := reqDyn.Resource(gvr).Namespace(project).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list agentic sessions in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list agentic sessions"})
		return
	}

	var sessions []types.AgenticSession
	for _, item := range list.Items {
		metadata, ok := GetMetadataMap(&item)
		if !ok {
			log.Printf("Warning: session missing metadata, skipping")
			continue
		}

		session := types.AgenticSession{
			APIVersion: item.GetAPIVersion(),
			Kind:       item.GetKind(),
			Metadata:   metadata,
		}

		if spec, ok := GetSpecMap(&item); ok {
			session.Spec = parseSpec(spec)
		}

		if status, ok := GetStatusMap(&item); ok {
			session.Status = parseStatus(status)
		}

		sessions = append(sessions, session)
	}

	c.JSON(http.StatusOK, gin.H{"items": sessions})
}

// CreateSession creates a new agentic session with support for multi-repo configuration and RFE workflows.
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
		log.Printf("Creating continuation session from parent %s", req.ParentSessionID)

		// Clean up temp-content pod from parent session to free the PVC
		// This prevents Multi-Attach errors when the new session tries to mount the same workspace
		reqK8s, _ := GetK8sClientsForRequest(c)
		if reqK8s == nil {
			log.Printf("CreateSession: Cannot cleanup temp pod, no K8s client available (non-fatal)")
		} else {
			tempPodName := fmt.Sprintf("temp-content-%s", req.ParentSessionID)
			if err := reqK8s.CoreV1().Pods(project).Delete(c.Request.Context(), tempPodName, v1.DeleteOptions{}); err != nil {
				if !errors.IsNotFound(err) {
					log.Printf("CreateSession: failed to delete temp-content pod %s (non-fatal): %v", tempPodName, err)
				}
			} else {
				log.Printf("CreateSession: deleted temp-content pod %s to free PVC for continuation", tempPodName)
			}
		}
	}

	// Get spec for modifications (we know it exists since we just created the session object)
	sessionSpec, ok := session["spec"].(map[string]interface{})
	if !ok {
		log.Printf("Warning: session spec has unexpected type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error creating session"})
		return
	}

	if len(envVars) > 0 {
		sessionSpec["environmentVariables"] = envVars
	}

	// Interactive flag
	if req.Interactive != nil {
		sessionSpec["interactive"] = *req.Interactive
	}

	// AutoPushOnComplete flag
	if req.AutoPushOnComplete != nil {
		sessionSpec["autoPushOnComplete"] = *req.AutoPushOnComplete
	}

	// Set multi-repo configuration on spec
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
		sessionSpec["repos"] = arr
	}
	if req.MainRepoIndex != nil {
		sessionSpec["mainRepoIndex"] = *req.MainRepoIndex
	}

	// Handle RFE workflow branch management
	{
		rfeWorkflowID := ""
		// Check if RFE workflow ID is in labels
		if len(req.Labels) > 0 {
			if id, ok := req.Labels["rfe-workflow"]; ok {
				rfeWorkflowID = id
			}
		}

		// If linked to an RFE workflow, fetch it and set the branch
		if rfeWorkflowID != "" {
			// Get request-scoped dynamic client for fetching RFE workflow
			_, reqDyn := GetK8sClientsForRequest(c)
			if reqDyn != nil {
				rfeGvr := GetRFEWorkflowResource()
				if rfeGvr != (schema.GroupVersionResource{}) {
					rfeObj, err := reqDyn.Resource(rfeGvr).Namespace(project).Get(c.Request.Context(), rfeWorkflowID, v1.GetOptions{})
					if err == nil {
						rfeWf := RfeFromUnstructured(rfeObj)
						if rfeWf != nil && rfeWf.BranchName != "" {
							// Override branch for all repos to use feature branch
							if repos, ok := sessionSpec["repos"].([]map[string]interface{}); ok {
								for i := range repos {
									// Always override input branch with feature branch
									if input, ok := repos[i]["input"].(map[string]interface{}); ok {
										input["branch"] = rfeWf.BranchName
									}
									// Always override output branch with feature branch
									if output, ok := repos[i]["output"].(map[string]interface{}); ok {
										output["branch"] = rfeWf.BranchName
									}
								}
							}

							log.Printf("Set RFE branch %s for session %s", rfeWf.BranchName, name)
						}
					} else {
						log.Printf("Warning: Failed to fetch RFE workflow %s: %v", rfeWorkflowID, err)
					}
				}
			}
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
			sessionSpec["userContext"] = map[string]interface{}{
				"userId":      uid,
				"displayName": displayName,
				"groups":      groups,
			}
		}
	}

	// Add botAccount if provided
	if req.BotAccount != nil {
		sessionSpec["botAccount"] = map[string]interface{}{
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
			sessionSpec["resourceOverrides"] = resourceOverrides
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
				Resources: []string{"agenticsessions/status"},
				Verbs:     []string{"get", "update", "patch"},
			},
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

	// Create Secret (with OwnerReference to be cleaned up when Session is deleted)
	secretName := fmt.Sprintf("ambient-session-%s-token", sessionName)
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:            secretName,
			Namespace:       project,
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: secretData,
	}
	if _, err := reqK8s.CoreV1().Secrets(project).Create(c.Request.Context(), secret, v1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing secret with new token
			if _, err := reqK8s.CoreV1().Secrets(project).Update(c.Request.Context(), secret, v1.UpdateOptions{}); err != nil {
				return fmt.Errorf("update Secret: %w", err)
			}
		} else {
			return fmt.Errorf("create Secret: %w", err)
		}
	}

	// Annotate the session with the secret name (runner will look for it)
	patch := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/metadata/annotations/vteam.ambient-code~1runner-token-secret",
			"value": secretName,
		},
	}
	patchData, _ := json.Marshal(patch)
	if _, err := reqDyn.Resource(gvr).Namespace(project).Patch(c.Request.Context(), sessionName, k8stypes.JSONPatchType, patchData, v1.PatchOptions{}); err != nil {
		// Fall back to regular merge patch if JSON patch fails
		mergePatch := map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{
					"vteam.ambient-code/runner-token-secret": secretName,
				},
			},
		}
		patchData, _ := json.Marshal(mergePatch)
		if _, err := reqDyn.Resource(gvr).Namespace(project).Patch(c.Request.Context(), sessionName, k8stypes.MergePatchType, patchData, v1.PatchOptions{}); err != nil {
			return fmt.Errorf("annotate AgenticSession: %w", err)
		}
	}

	log.Printf("Provisioned runner token for session %s/%s", project, sessionName)
	return nil
}

// GetSession retrieves a single agentic session by name.
func GetSession(c *gin.Context) {
	project := c.GetString("project")
	name := c.Param("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session name is required"})
		return
	}

	_, reqDyn := GetK8sClientsForRequest(c)
	gvr := GetAgenticSessionV1Alpha1Resource()

	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		} else {
			log.Printf("Failed to get agentic session %s in project %s: %v", name, project, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		}
		return
	}

	metadata, ok := GetMetadataMap(obj)
	if !ok {
		log.Printf("Session %s missing metadata", name)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session data"})
		return
	}

	session := types.AgenticSession{
		APIVersion: obj.GetAPIVersion(),
		Kind:       obj.GetKind(),
		Metadata:   metadata,
	}

	if spec, ok := GetSpecMap(obj); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := GetStatusMap(obj); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusOK, session)
}

// PatchSession patches an agentic session's annotations.
// Only annotations are supported for patching currently.
func PatchSession(c *gin.Context) {
	project := c.GetString("project")
	name := c.Param("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session name is required"})
		return
	}

	var patchReq map[string]interface{}
	if err := c.ShouldBindJSON(&patchReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Only support patching annotations
	if metadata, ok := patchReq["metadata"].(map[string]interface{}); ok {
		if _, hasAnnotations := metadata["annotations"]; !hasAnnotations {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only metadata.annotations can be patched"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only metadata.annotations can be patched"})
		return
	}

	// Use backend service account for writes
	if DynamicClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "backend not initialized"})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Create merge patch
	patchData, err := json.Marshal(patchReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal patch data"})
		return
	}

	// Apply patch
	patched, err := DynamicClient.Resource(gvr).Namespace(project).Patch(context.TODO(), name, k8stypes.MergePatchType, patchData, v1.PatchOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		} else {
			log.Printf("Failed to patch agentic session %s in project %s: %v", name, project, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to patch agentic session"})
		}
		return
	}

	metadata, ok := GetMetadataMap(patched)
	if !ok {
		log.Printf("Patched session %s missing metadata", name)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session data"})
		return
	}

	session := types.AgenticSession{
		APIVersion: patched.GetAPIVersion(),
		Kind:       patched.GetKind(),
		Metadata:   metadata,
	}

	if spec, ok := GetSpecMap(patched); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := GetStatusMap(patched); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusOK, session)
}

// UpdateSession updates an agentic session's prompt, displayName, LLMSettings, and timeout.
func UpdateSession(c *gin.Context) {
	project := c.GetString("project")
	name := c.Param("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session name is required"})
		return
	}

	var updateReq types.UpdateAgenticSessionRequest
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use backend service account for writes
	if DynamicClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "backend not initialized"})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get the existing session
	existing, err := DynamicClient.Resource(gvr).Namespace(project).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		} else {
			log.Printf("Failed to get agentic session %s in project %s: %v", name, project, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		}
		return
	}

	// Update the spec fields
	spec, ok := GetSpecMap(existing)
	if !ok {
		log.Printf("Session %s missing spec", name)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session data"})
		return
	}

	if updateReq.Prompt != nil {
		spec["prompt"] = *updateReq.Prompt
	}
	if updateReq.DisplayName != nil {
		spec["displayName"] = *updateReq.DisplayName
	}
	if updateReq.Timeout != nil {
		spec["timeout"] = *updateReq.Timeout
	}

	// Update LLM settings if provided
	if updateReq.LLMSettings != nil {
		llmSettings, ok := spec["llmSettings"].(map[string]interface{})
		if !ok {
			llmSettings = make(map[string]interface{})
		}
		if updateReq.LLMSettings.Model != "" {
			llmSettings["model"] = updateReq.LLMSettings.Model
		}
		if updateReq.LLMSettings.Temperature != 0 {
			llmSettings["temperature"] = updateReq.LLMSettings.Temperature
		}
		if updateReq.LLMSettings.MaxTokens != 0 {
			llmSettings["maxTokens"] = updateReq.LLMSettings.MaxTokens
		}
		spec["llmSettings"] = llmSettings
	}

	// Update the resource
	updated, err := DynamicClient.Resource(gvr).Namespace(project).Update(context.TODO(), existing, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update agentic session %s in project %s: %v", name, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agentic session"})
		return
	}

	metadata, ok := GetMetadataMap(updated)
	if !ok {
		log.Printf("Updated session %s missing metadata", name)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session data"})
		return
	}

	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   metadata,
	}

	if spec, ok := GetSpecMap(updated); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := GetStatusMap(updated); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusOK, session)
}

// UpdateSessionDisplayName updates only the displayName field of an agentic session.
// This is a convenience endpoint for updating just the display name.
func UpdateSessionDisplayName(c *gin.Context) {
	project := c.GetString("project")
	name := c.Param("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session name is required"})
		return
	}

	var updateReq struct {
		DisplayName string `json:"displayName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use backend service account for writes
	if DynamicClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "backend not initialized"})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Create a merge patch to update just the displayName
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"displayName": updateReq.DisplayName,
		},
	}

	patchData, err := json.Marshal(patch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal patch data"})
		return
	}

	// Apply the patch
	updated, err := DynamicClient.Resource(gvr).Namespace(project).Patch(context.TODO(), name, k8stypes.MergePatchType, patchData, v1.PatchOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		} else {
			log.Printf("Failed to update display name for agentic session %s in project %s: %v", name, project, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update display name"})
		}
		return
	}

	metadata, ok := GetMetadataMap(updated)
	if !ok {
		log.Printf("Updated session %s missing metadata", name)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session data"})
		return
	}

	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   metadata,
	}

	if spec, ok := GetSpecMap(updated); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := GetStatusMap(updated); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusOK, session)
}

// DeleteSession deletes an agentic session.
func DeleteSession(c *gin.Context) {
	project := c.GetString("project")
	name := c.Param("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session name is required"})
		return
	}

	// Use backend service account for writes
	if DynamicClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "backend not initialized"})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	if err := DynamicClient.Resource(gvr).Namespace(project).Delete(context.TODO(), name, v1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		} else {
			log.Printf("Failed to delete agentic session %s in project %s: %v", name, project, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agentic session"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session deleted successfully"})
}

// CloneSession clones an existing agentic session to another project.
// It supports cross-project cloning for OpenShift environments.
func CloneSession(c *gin.Context) {
	sourceProject := c.GetString("project")
	sourceName := c.Param("name")

	if sourceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session name is required"})
		return
	}

	var cloneReq types.CloneAgenticSessionRequest
	if err := c.ShouldBindJSON(&cloneReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use backend service account for writes
	if DynamicClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "backend not initialized"})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get the source session using request-scoped clients
	_, reqDyn := GetK8sClientsForRequest(c)
	sourceObj, err := reqDyn.Resource(gvr).Namespace(sourceProject).Get(context.TODO(), sourceName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Source session not found"})
		} else {
			log.Printf("Failed to get source session %s in project %s: %v", sourceName, sourceProject, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get source session"})
		}
		return
	}

	// Determine target project
	targetProject := sourceProject
	if cloneReq.TargetProject != "" {
		targetProject = cloneReq.TargetProject

		// For cross-project cloning, verify user has access to target project
		if targetProject != sourceProject {
			// Check if OpenShift project resource exists
			if GetOpenShiftProjectResource != nil && GetOpenShiftProjectResource() != (schema.GroupVersionResource{}) {
				projGvr := GetOpenShiftProjectResource()
				if _, err := reqDyn.Resource(projGvr).Get(context.TODO(), targetProject, v1.GetOptions{}); err != nil {
					if errors.IsNotFound(err) {
						c.JSON(http.StatusNotFound, gin.H{"error": "Target project not found"})
					} else {
						log.Printf("Failed to verify access to target project %s: %v", targetProject, err)
						c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to target project"})
					}
					return
				}
			}
		}
	}

	// Check for naming conflicts if a specific target name is requested
	targetName := cloneReq.TargetSessionName
	if targetName == "" {
		// Generate unique name if not provided
		timestamp := time.Now().Unix()
		targetName = fmt.Sprintf("agentic-session-%d", timestamp)
	} else {
		// Check if name already exists in target project
		if _, err := DynamicClient.Resource(gvr).Namespace(targetProject).Get(context.TODO(), targetName, v1.GetOptions{}); err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Session with target name already exists in target project"})
			return
		} else if !errors.IsNotFound(err) {
			log.Printf("Failed to check target session existence: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify target name availability"})
			return
		}
	}

	// Create a deep copy of the source session
	sourceSpec, ok := GetSpecMap(sourceObj)
	if !ok {
		log.Printf("Source session %s missing spec", sourceName)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid source session data"})
		return
	}

	clonedMetadata := map[string]interface{}{
		"name":      targetName,
		"namespace": targetProject,
	}

	// Copy labels if present
	if labels := sourceObj.GetLabels(); len(labels) > 0 {
		newLabels := make(map[string]interface{})
		for k, v := range labels {
			newLabels[k] = v
		}
		clonedMetadata["labels"] = newLabels
	}

	// Copy annotations, excluding system annotations
	if annotations := sourceObj.GetAnnotations(); len(annotations) > 0 {
		newAnnotations := make(map[string]interface{})
		for k, v := range annotations {
			// Skip system annotations like runner token secrets
			if !strings.HasPrefix(k, "vteam.ambient-code/runner-token") &&
				!strings.HasPrefix(k, "kubectl.kubernetes.io/") {
				newAnnotations[k] = v
			}
		}
		// Add clone metadata
		newAnnotations["vteam.ambient-code/cloned-from"] = fmt.Sprintf("%s/%s", sourceProject, sourceName)
		newAnnotations["vteam.ambient-code/cloned-at"] = time.Now().UTC().Format(time.RFC3339)
		clonedMetadata["annotations"] = newAnnotations
	}

	clonedSession := map[string]interface{}{
		"apiVersion": sourceObj.GetAPIVersion(),
		"kind":       sourceObj.GetKind(),
		"metadata":   clonedMetadata,
		"spec":       sourceSpec,
	}

	// Update spec with clone request overrides
	spec := sourceSpec
	if cloneReq.DisplayName != "" {
		spec["displayName"] = cloneReq.DisplayName
	}
	if cloneReq.Prompt != "" {
		spec["prompt"] = cloneReq.Prompt
	}

	// Update project field in spec to target project
	spec["project"] = targetProject

	// Reset status for new session
	clonedSession["status"] = map[string]interface{}{
		"phase": "Pending",
	}

	// Create the cloned session
	obj := &unstructured.Unstructured{Object: clonedSession}
	created, err := DynamicClient.Resource(gvr).Namespace(targetProject).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create cloned session in project %s: %v", targetProject, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cloned session"})
		return
	}

	// Provision runner token for cloned session
	if err := provisionRunnerTokenForSession(c, K8sClient, DynamicClient, targetProject, targetName); err != nil {
		log.Printf("Warning: failed to provision runner token for cloned session %s/%s: %v", targetProject, targetName, err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Session cloned successfully",
		"name":          targetName,
		"targetProject": targetProject,
		"uid":           created.GetUID(),
	})
}
