package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
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

// Package-level variables for project handlers (set from main package)
var (
	// GetOpenShiftProjectResource returns the GVR for OpenShift Project resources
	GetOpenShiftProjectResource func() schema.GroupVersionResource
	// K8sClientProjects is the backend service account client used for namespace operations
	// that require elevated permissions (e.g., creating namespaces, assigning roles)
	K8sClientProjects *kubernetes.Clientset
	// DynamicClientProjects is the backend SA dynamic client for OpenShift Project operations
	DynamicClientProjects dynamic.Interface
)

var (
	isOpenShiftCache bool
	isOpenShiftOnce  sync.Once
)

// Default timeout for Kubernetes API operations
const defaultK8sTimeout = 10 * time.Second

// Retry configuration constants
const (
	projectRetryAttempts     = 5
	projectRetryInitialDelay = 200 * time.Millisecond
	projectRetryMaxDelay     = 2 * time.Second
)

// Kubernetes namespace name validation pattern
var namespaceNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// validateProjectName validates a project/namespace name according to Kubernetes naming rules
func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	if len(name) > 63 {
		return fmt.Errorf("project name must be 63 characters or less")
	}
	if !namespaceNamePattern.MatchString(name) {
		return fmt.Errorf("project name must be lowercase alphanumeric with hyphens (cannot start or end with hyphen)")
	}
	// Reserved namespaces
	reservedNames := map[string]bool{
		"default": true, "kube-system": true, "kube-public": true, "kube-node-lease": true,
		"openshift": true, "openshift-infra": true, "openshift-node": true,
	}
	if reservedNames[name] {
		return fmt.Errorf("project name '%s' is reserved and cannot be used", name)
	}
	return nil
}

// sanitizeForK8sName converts a user subject to a valid Kubernetes resource name
func sanitizeForK8sName(subject string) string {
	// Remove system:serviceaccount: prefix if present
	subject = strings.TrimPrefix(subject, "system:serviceaccount:")

	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	sanitized := reg.ReplaceAllString(strings.ToLower(subject), "-")

	// Remove leading/trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// Ensure it doesn't exceed 63 chars (leave room for prefix)
	if len(sanitized) > 40 {
		sanitized = sanitized[:40]
	}

	return sanitized
}

// isOpenShiftCluster detects if we're running on OpenShift by checking for the project.openshift.io API group
// Results are cached using sync.Once for thread-safe, race-free initialization
func isOpenShiftCluster() bool {
	isOpenShiftOnce.Do(func() {
		if K8sClientProjects == nil {
			log.Printf("K8s client not initialized, assuming vanilla Kubernetes")
			isOpenShiftCache = false
			return
		}

		// Try to list API groups and look for project.openshift.io
		groups, err := K8sClientProjects.Discovery().ServerGroups()
		if err != nil {
			log.Printf("Failed to detect OpenShift (assuming vanilla Kubernetes): %v", err)
			isOpenShiftCache = false
			return
		}

		for _, group := range groups.Groups {
			if group.Name == "project.openshift.io" {
				log.Printf("Detected OpenShift cluster")
				isOpenShiftCache = true
				return
			}
		}

		log.Printf("Detected vanilla Kubernetes cluster")
		isOpenShiftCache = false
	})
	return isOpenShiftCache
}

// GetClusterInfo handles GET /cluster-info
// Returns information about the cluster type (OpenShift vs vanilla Kubernetes)
// This endpoint does not require authentication as it's public cluster information
func GetClusterInfo(c *gin.Context) {
	isOpenShift := isOpenShiftCluster()

	c.JSON(http.StatusOK, gin.H{
		"isOpenShift": isOpenShift,
	})
}

// ListProjects handles GET /projects
// Uses user's token to list Projects (OpenShift) or Namespaces (k8s) with label selector
// Kubernetes RBAC and OpenShift automatically filter to only show accessible projects
func ListProjects(c *gin.Context) {
	reqK8s, reqDyn := GetK8sClientsForRequest(c)

	if reqK8s == nil || reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	isOpenShift := isOpenShiftCluster()
	projects := []types.AmbientProject{}

	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sTimeout)
	defer cancel()

	if isOpenShift {
		// OpenShift: List Projects with label selector (user's token)
		projGvr := GetOpenShiftProjectResource()
		var dynClient dynamic.Interface = reqDyn

		list, err := dynClient.Resource(projGvr).List(ctx, v1.ListOptions{
			LabelSelector: "ambient-code.io/managed=true",
		})
		if err != nil {
			log.Printf("Failed to list OpenShift Projects: %v", err)
			if errors.IsForbidden(err) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions to list projects"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
			}
			return
		}

		for _, item := range list.Items {
			projects = append(projects, projectFromUnstructured(&item, true))
		}
	} else {
		// Kubernetes: List Namespaces with label selector (user's token)
		nsList, err := reqK8s.CoreV1().Namespaces().List(ctx, v1.ListOptions{
			LabelSelector: "ambient-code.io/managed=true",
		})
		if err != nil {
			log.Printf("Failed to list Namespaces: %v", err)
			if errors.IsForbidden(err) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions to list namespaces"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
			}
			return
		}

		for _, ns := range nsList.Items {
			projects = append(projects, projectFromNamespace(&ns, false))
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": projects})
}

// projectFromUnstructured converts an unstructured OpenShift Project to AmbientProject
func projectFromUnstructured(item *unstructured.Unstructured, isOpenShift bool) types.AmbientProject {
	meta, ok := item.Object["metadata"].(map[string]interface{})
	if !ok || meta == nil {
		log.Printf("Warning: malformed metadata for project %s", item.GetName())
		meta = make(map[string]interface{})
	}

	name := item.GetName()

	labels := map[string]string{}
	annotations := map[string]string{}

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

	displayName := annotations["openshift.io/display-name"]
	description := annotations["openshift.io/description"]
	created := item.GetCreationTimestamp().Time

	status := "Active"
	if st, ok := item.Object["status"].(map[string]interface{}); ok {
		if phase, ok := st["phase"].(string); ok {
			status = phase
		}
	}

	return types.AmbientProject{
		Name:              name,
		DisplayName:       displayName,
		Description:       description,
		Labels:            labels,
		Annotations:       annotations,
		CreationTimestamp: created.Format(time.RFC3339),
		Status:            status,
		IsOpenShift:       isOpenShift,
	}
}

// projectFromNamespace converts a Kubernetes Namespace to AmbientProject
func projectFromNamespace(ns *corev1.Namespace, isOpenShift bool) types.AmbientProject {
	status := "Active"
	if ns.Status.Phase != corev1.NamespaceActive {
		status = string(ns.Status.Phase)
	}

	// For k8s, displayName and description are empty
	return types.AmbientProject{
		Name:              ns.Name,
		DisplayName:       "",
		Description:       "",
		Labels:            ns.Labels,
		Annotations:       ns.Annotations,
		CreationTimestamp: ns.CreationTimestamp.Format(time.RFC3339),
		Status:            status,
		IsOpenShift:       isOpenShift,
	}
}

// CreateProject handles POST /projects
// Unified approach for both Kubernetes and OpenShift:
// 1. Creates namespace using backend SA (both platforms)
// 2. Assigns ambient-project-admin ClusterRole to creator via RoleBinding (both platforms)
//
// The ClusterRole is namespace-scoped via the RoleBinding, giving the user admin access
// only to their specific project namespace.
func CreateProject(c *gin.Context) {
	reqK8s, _ := GetK8sClientsForRequest(c)

	// Validate that user authentication succeeded
	if reqK8s == nil {
		log.Printf("CreateProject: Invalid or missing authentication token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	var req types.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate project name
	if err := validateProjectName(req.Name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract user identity from token
	userSubject, err := getUserSubjectFromContext(c)
	if err != nil {
		log.Printf("CreateProject: Failed to extract user subject: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	isOpenShift := isOpenShiftCluster()

	// Create namespace using backend SA (users don't have cluster-level permissions)
	ns := &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: req.Name,
			Labels: map[string]string{
				"ambient-code.io/managed": "true",
			},
			Annotations: map[string]string{},
		},
	}

	// Add OpenShift-specific annotations if on OpenShift
	if isOpenShift {
		// Use displayName if provided, otherwise use name
		displayName := req.DisplayName
		if displayName == "" {
			displayName = req.Name
		}
		ns.Annotations["openshift.io/display-name"] = displayName
		if req.Description != "" {
			ns.Annotations["openshift.io/description"] = req.Description
		}
		ns.Annotations["openshift.io/requester"] = userSubject
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	createdNs, err := K8sClientProjects.CoreV1().Namespaces().Create(ctx, ns, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create namespace %s: %v", req.Name, err)
		if errors.IsAlreadyExists(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Project already exists"})
		} else if errors.IsForbidden(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions to create project"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		}
		return
	}

	// Assign ambient-project-admin ClusterRole to the creator
	// Use deterministic name based on user to avoid conflicts with multiple admins
	roleBindingName := fmt.Sprintf("ambient-admin-%s", sanitizeForK8sName(userSubject))

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: req.Name,
			Labels: map[string]string{
				"ambient-code.io/role": "admin",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "ambient-project-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     getUserSubjectKind(userSubject),
				Name:     getUserSubjectName(userSubject),
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
	}

	// Add namespace for ServiceAccount subjects
	if getUserSubjectKind(userSubject) == "ServiceAccount" {
		roleBinding.Subjects[0].Namespace = getUserSubjectNamespace(userSubject)
		roleBinding.Subjects[0].APIGroup = ""
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	_, err = K8sClientProjects.RbacV1().RoleBindings(req.Name).Create(ctx2, roleBinding, v1.CreateOptions{})
	if err != nil {
		log.Printf("ERROR: Created namespace %s but failed to assign admin role: %v", req.Name, err)

		// ROLLBACK: Delete the namespace since role binding failed
		// Without the role binding, the user won't have access to their project
		ctx3, cancel3 := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel3()

		deleteErr := K8sClientProjects.CoreV1().Namespaces().Delete(ctx3, req.Name, v1.DeleteOptions{})
		if deleteErr != nil {
			log.Printf("CRITICAL: Failed to rollback namespace %s after role binding failure: %v", req.Name, deleteErr)

			// Label the namespace as orphaned for manual cleanup
			patch := []byte(`{"metadata":{"labels":{"ambient-code.io/orphaned":"true","ambient-code.io/orphan-reason":"role-binding-failed"}}}`)
			ctx4, cancel4 := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel4()

			_, labelErr := K8sClientProjects.CoreV1().Namespaces().Patch(
				ctx4, req.Name, k8stypes.MergePatchType, patch, v1.PatchOptions{},
			)
			if labelErr != nil {
				log.Printf("CRITICAL: Failed to label orphaned namespace %s: %v", req.Name, labelErr)
			} else {
				log.Printf("Labeled orphaned namespace %s for manual cleanup", req.Name)
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project permissions"})
		return
	}

	// On OpenShift: Update the Project resource with display metadata
	// Use retry logic as OpenShift needs time to create the Project resource from the namespace
	// Use backend SA dynamic client (users don't have permission to update Project resources)
	if isOpenShift && DynamicClientProjects != nil {
		projGvr := GetOpenShiftProjectResource()

		// Retry getting and updating the Project resource (OpenShift creates it asynchronously)
		retryErr := RetryWithBackoff(projectRetryAttempts, projectRetryInitialDelay, projectRetryMaxDelay, func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Get the Project resource (using backend SA)
			projObj, err := DynamicClientProjects.Resource(projGvr).Get(ctx, req.Name, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get Project resource: %w", err)
			}

			// Update Project annotations with display metadata
			meta, ok := projObj.Object["metadata"].(map[string]interface{})
			if !ok || meta == nil {
				meta = map[string]interface{}{}
				projObj.Object["metadata"] = meta
			}
			anns, ok := meta["annotations"].(map[string]interface{})
			if !ok || anns == nil {
				anns = map[string]interface{}{}
				meta["annotations"] = anns
			}

			// Use displayName if provided, otherwise use name
			displayName := req.DisplayName
			if displayName == "" {
				displayName = req.Name
			}
			anns["openshift.io/display-name"] = displayName
			if req.Description != "" {
				anns["openshift.io/description"] = req.Description
			}
			anns["openshift.io/requester"] = userSubject

			ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel2()

			// Update using backend SA (users don't have Project update permission)
			_, err = DynamicClientProjects.Resource(projGvr).Update(ctx2, projObj, v1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update Project annotations: %w", err)
			}

			return nil
		})

		if retryErr != nil {
			log.Printf("WARNING: Failed to update Project resource for %s after retries: %v", req.Name, retryErr)
		} else {
			log.Printf("Successfully updated Project resource with display metadata for %s", req.Name)
		}
	}

	// Build response
	responseDisplayName := ""
	if isOpenShift {
		responseDisplayName = req.DisplayName
		if responseDisplayName == "" {
			responseDisplayName = req.Name
		}
	}

	project := types.AmbientProject{
		Name:              createdNs.Name,
		DisplayName:       responseDisplayName,
		Description:       req.Description,
		Labels:            createdNs.Labels,
		Annotations:       createdNs.Annotations,
		CreationTimestamp: createdNs.CreationTimestamp.Format(time.RFC3339),
		Status:            "Active",
		IsOpenShift:       isOpenShift,
	}

	c.JSON(http.StatusCreated, project)
}

// GetProject handles GET /projects/:projectName
// On OpenShift: Returns OpenShift Project details
// On Kubernetes: Returns Namespace details
func GetProject(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)

	if reqK8s == nil || reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	isOpenShift := isOpenShiftCluster()

	if isOpenShift {
		// OpenShift: Get Project
		projGvr := GetOpenShiftProjectResource()

		ctx, cancel := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel()

		projObj, err := reqDyn.Resource(projGvr).Get(ctx, projectName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}
			if errors.IsUnauthorized(err) || errors.IsForbidden(err) {
				log.Printf("User forbidden to access OpenShift Project %s: %v", projectName, err)
				c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to access project"})
				return
			}
			log.Printf("Failed to get OpenShift Project %s: %v", projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
			return
		}

		project := projectFromUnstructured(projObj, true)

		// Validate it's an Ambient-managed project
		if project.Labels["ambient-code.io/managed"] != "true" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not an Ambient project"})
			return
		}

		c.JSON(http.StatusOK, project)
	} else {
		// Kubernetes: Get Namespace
		ctx, cancel := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel()

		ns, err := reqK8s.CoreV1().Namespaces().Get(ctx, projectName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}
			if errors.IsUnauthorized(err) || errors.IsForbidden(err) {
				log.Printf("User forbidden to access Namespace %s: %v", projectName, err)
				c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to access project"})
				return
			}
			log.Printf("Failed to get Namespace %s: %v", projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
			return
		}

		// Validate it's an Ambient-managed namespace
		if ns.Labels["ambient-code.io/managed"] != "true" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not an Ambient project"})
			return
		}

		project := projectFromNamespace(ns, false)
		c.JSON(http.StatusOK, project)
	}
}

// UpdateProject handles PUT /projects/:projectName
// On OpenShift: Updates display name and description via Project annotations
// On Kubernetes: No-op (returns success but doesn't update anything as k8s namespaces don't have displayName/description)
func UpdateProject(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)

	if reqK8s == nil || reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" && req.Name != projectName {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project name in URL does not match request body"})
		return
	}

	isOpenShift := isOpenShiftCluster()

	if isOpenShift {
		// OpenShift: Update Project annotations
		projGvr := GetOpenShiftProjectResource()

		ctx, cancel := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel()

		projObj, err := reqDyn.Resource(projGvr).Get(ctx, projectName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}
			log.Printf("Failed to get OpenShift Project %s: %v", projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
			return
		}

		// Validate it's an Ambient-managed project
		project := projectFromUnstructured(projObj, true)
		if project.Labels["ambient-code.io/managed"] != "true" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not an Ambient project"})
			return
		}

		// Update annotations
		meta, ok := projObj.Object["metadata"].(map[string]interface{})
		if !ok || meta == nil {
			meta = map[string]interface{}{}
			projObj.Object["metadata"] = meta
		}
		anns, ok := meta["annotations"].(map[string]interface{})
		if !ok || anns == nil {
			anns = map[string]interface{}{}
			meta["annotations"] = anns
		}

		if req.DisplayName != "" {
			anns["openshift.io/display-name"] = req.DisplayName
		}
		if req.Description != "" {
			anns["openshift.io/description"] = req.Description
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel2()

		_, err = reqDyn.Resource(projGvr).Update(ctx2, projObj, v1.UpdateOptions{})
		if err != nil {
			log.Printf("Failed to update OpenShift Project %s: %v", projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
			return
		}

		// Read back and return
		ctx3, cancel3 := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel3()

		projObj, _ = reqDyn.Resource(projGvr).Get(ctx3, projectName, v1.GetOptions{})
		updatedProject := projectFromUnstructured(projObj, true)
		c.JSON(http.StatusOK, updatedProject)
	} else {
		// Kubernetes: Just verify the namespace exists and return it
		// Display name and description are not supported on vanilla k8s
		ctx, cancel := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel()

		ns, err := reqK8s.CoreV1().Namespaces().Get(ctx, projectName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}
			log.Printf("Failed to get Namespace %s: %v", projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
			return
		}

		// Validate it's an Ambient-managed namespace
		if ns.Labels["ambient-code.io/managed"] != "true" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not an Ambient project"})
			return
		}

		project := projectFromNamespace(ns, false)
		c.JSON(http.StatusOK, project)
	}
}

// DeleteProject handles DELETE /projects/:projectName
// On OpenShift: Deletes the Project resource using user's credentials (user has permission as project admin)
// On Kubernetes: Verifies user has ambient-project-admin role, then uses backend SA to delete namespace
//
//	(namespace deletion is cluster-scoped, so regular users can't delete directly)
func DeleteProject(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)

	if reqK8s == nil || reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	isOpenShift := isOpenShiftCluster()

	if isOpenShift {
		// OpenShift: Delete Project resource using user's credentials
		projGvr := GetOpenShiftProjectResource()
		var dynClient dynamic.Interface = reqDyn

		ctx, cancel := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel()

		projObj, err := dynClient.Resource(projGvr).Get(ctx, projectName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}
			if errors.IsUnauthorized(err) || errors.IsForbidden(err) {
				log.Printf("User forbidden to delete OpenShift Project %s: %v", projectName, err)
				c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to access project"})
				return
			}
			log.Printf("Failed to get OpenShift Project %s: %v", projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
			return
		}

		// Validate it's an Ambient-managed project
		project := projectFromUnstructured(projObj, true)
		if project.Labels["ambient-code.io/managed"] != "true" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not an Ambient project"})
			return
		}

		// Delete the Project using user's credentials (OpenShift will cascade delete the namespace)
		ctx2, cancel2 := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel2()

		err = dynClient.Resource(projGvr).Delete(ctx2, projectName, v1.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}
			if errors.IsForbidden(err) {
				log.Printf("User forbidden to delete OpenShift Project %s: %v", projectName, err)
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions to delete project"})
				return
			}
			log.Printf("Failed to delete OpenShift Project %s: %v", projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
			return
		}
	} else {
		// Kubernetes: Verify namespace exists and is Ambient-managed
		ctx, cancel := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel()

		ns, err := K8sClientProjects.CoreV1().Namespaces().Get(ctx, projectName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}
			log.Printf("Failed to get namespace %s: %v", projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
			return
		}

		// Validate it's an Ambient-managed namespace
		if ns.Labels["ambient-code.io/managed"] != "true" {
			log.Printf("SECURITY: User attempted to delete non-managed namespace: %s", projectName)
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not an Ambient project"})
			return
		}

		// Verify user has ambient-project-admin role binding in this namespace
		userSubject, err := getUserSubjectFromContext(c)
		if err != nil {
			log.Printf("DeleteProject: Failed to extract user subject: %v", err)
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions to delete project"})
			return
		}

		hasAdminAccess, err := checkUserHasAdminRoleBinding(projectName, userSubject)
		if err != nil {
			log.Printf("DeleteProject: Failed to check role binding for %s in %s: %v", userSubject, projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
			return
		}

		if !hasAdminAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions to delete project"})
			return
		}

		// Delete the namespace using backend SA (after verifying user has admin role)
		// On vanilla Kubernetes, regular users can't delete namespaces directly (cluster-scoped resource).
		// We verify the user has the ambient-project-admin role binding, then use backend SA to perform deletion.

		// Defense-in-depth: Double-check namespace is still Ambient-managed before deletion
		ctx2, cancel2 := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel2()

		verifyNs, verifyErr := K8sClientProjects.CoreV1().Namespaces().Get(ctx2, projectName, v1.GetOptions{})
		if verifyErr != nil {
			log.Printf("Failed to verify namespace %s before deletion: %v", projectName, verifyErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify project"})
			return
		}
		if verifyNs.Labels["ambient-code.io/managed"] != "true" {
			log.Printf("SECURITY: Namespace %s lost managed label, aborting deletion", projectName)
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete non-managed namespace"})
			return
		}

		ctx3, cancel3 := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel3()

		err = K8sClientProjects.CoreV1().Namespaces().Delete(ctx3, projectName, v1.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}
			log.Printf("Failed to delete namespace %s: %v", projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
			return
		}
	}

	c.Status(http.StatusNoContent)
}

// checkUserHasAdminRoleBinding verifies if a user has the ambient-project-admin role binding in a namespace
// Uses direct GET for efficiency instead of listing all role bindings
func checkUserHasAdminRoleBinding(namespace, userSubject string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get the specific role binding we create (user-specific name)
	roleBindingName := fmt.Sprintf("ambient-admin-%s", sanitizeForK8sName(userSubject))
	rb, err := K8sClientProjects.RbacV1().RoleBindings(namespace).Get(ctx, roleBindingName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Role binding doesn't exist, check if there are any other role bindings granting admin
			return checkUserHasAdminRoleBindingFallback(namespace, userSubject)
		}
		return false, err
	}

	// Verify this role binding grants ambient-project-admin
	if rb.RoleRef.Kind != "ClusterRole" || rb.RoleRef.Name != "ambient-project-admin" {
		return checkUserHasAdminRoleBindingFallback(namespace, userSubject)
	}

	userKind := getUserSubjectKind(userSubject)
	userName := getUserSubjectName(userSubject)
	userNs := getUserSubjectNamespace(userSubject)

	// Check if user is in the subjects list
	for _, subject := range rb.Subjects {
		if subject.Kind == userKind && subject.Name == userName {
			// For ServiceAccount, also check namespace
			if userKind == "ServiceAccount" {
				if subject.Namespace == userNs {
					return true, nil
				}
			} else {
				return true, nil
			}
		}
	}

	// User not in this role binding, check others
	return checkUserHasAdminRoleBindingFallback(namespace, userSubject)
}

// checkUserHasAdminRoleBindingFallback checks all role bindings (slower fallback)
func checkUserHasAdminRoleBindingFallback(namespace, userSubject string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// List all RoleBindings in the namespace
	roleBindings, err := K8sClientProjects.RbacV1().RoleBindings(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return false, err
	}

	userKind := getUserSubjectKind(userSubject)
	userName := getUserSubjectName(userSubject)
	userNs := getUserSubjectNamespace(userSubject)

	// Check if any RoleBinding grants ambient-project-admin to this user
	for _, rb := range roleBindings.Items {
		if rb.RoleRef.Kind == "ClusterRole" && rb.RoleRef.Name == "ambient-project-admin" {
			for _, subject := range rb.Subjects {
				if subject.Kind == userKind && subject.Name == userName {
					// For ServiceAccount, also check namespace
					if userKind == "ServiceAccount" {
						if subject.Namespace == userNs {
							return true, nil
						}
					} else {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// getUserSubjectFromContext extracts the user subject from the JWT token in the request
// Returns subject in format like "user@example.com" or "system:serviceaccount:namespace:name"
func getUserSubjectFromContext(c *gin.Context) (string, error) {
	// Try to extract from ServiceAccount first
	ns, saName, ok := ExtractServiceAccountFromAuth(c)
	if ok {
		return fmt.Sprintf("system:serviceaccount:%s:%s", ns, saName), nil
	}

	// Otherwise try to get from context (set by middleware)
	if userName, exists := c.Get("userName"); exists && userName != nil {
		return fmt.Sprintf("%v", userName), nil
	}
	if userID, exists := c.Get("userID"); exists && userID != nil {
		return fmt.Sprintf("%v", userID), nil
	}

	return "", fmt.Errorf("no user subject found in token")
}

// getUserSubjectKind returns "ServiceAccount" or "User" based on the subject format
func getUserSubjectKind(subject string) string {
	if len(subject) > 22 && subject[:22] == "system:serviceaccount:" {
		return "ServiceAccount"
	}
	return "User"
}

// getUserSubjectName returns the name part of the subject
// For ServiceAccount: "system:serviceaccount:namespace:name" -> "name"
// For User: returns the subject as-is
func getUserSubjectName(subject string) string {
	if getUserSubjectKind(subject) == "ServiceAccount" {
		parts := strings.Split(subject, ":")
		if len(parts) >= 4 {
			return parts[3]
		}
	}
	return subject
}

// getUserSubjectNamespace returns the namespace for ServiceAccount subjects
// For ServiceAccount: "system:serviceaccount:namespace:name" -> "namespace"
// For User: returns empty string
func getUserSubjectNamespace(subject string) string {
	if getUserSubjectKind(subject) == "ServiceAccount" {
		parts := strings.Split(subject, ":")
		if len(parts) >= 3 {
			return parts[2]
		}
	}
	return ""
}
