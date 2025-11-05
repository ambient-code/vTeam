// Package handlers provides HTTP handlers for the backend API.
// This file contains CRUD operations for project management.
package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// ListProjects handles GET /projects
// Lists Namespaces (both platforms) using backend SA with label selector,
// then uses SubjectAccessReview to verify user access to each namespace
func ListProjects(c *gin.Context) {
	reqK8s, _ := GetK8sClientsForRequest(c)

	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	// List namespaces using backend SA (both platforms)
	if K8sClientProjects == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
		return
	}

	isOpenShift := isOpenShiftCluster()
	projects := []types.AmbientProject{}

	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sTimeout)
	defer cancel()

	nsList, err := K8sClientProjects.CoreV1().Namespaces().List(ctx, v1.ListOptions{
		LabelSelector: "ambient-code.io/managed=true",
	})
	if err != nil {
		log.Printf("Failed to list Namespaces: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
		return
	}

	// Filter to only namespaces where user has access
	// Use SubjectAccessReview - checks ALL RBAC sources (any RoleBinding, group, etc.)
	for _, ns := range nsList.Items {
		hasAccess, err := checkUserCanAccessNamespace(reqK8s, ns.Name)
		if err != nil {
			log.Printf("Failed to check access for namespace %s: %v", ns.Name, err)
			continue
		}

		if hasAccess {
			projects = append(projects, projectFromNamespace(&ns, isOpenShift))
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": projects})
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

	// Assign ambient-project-admin ClusterRole to the creator in the namespace
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
			meta, ok := GetMetadataMap(projObj)
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
// Returns Namespace details with OpenShift annotations if on OpenShift
func GetProject(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)

	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	isOpenShift := isOpenShiftCluster()

	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sTimeout)
	defer cancel()

	ns, err := reqK8s.CoreV1().Namespaces().Get(ctx, projectName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		if errors.IsForbidden(err) {
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
		log.Printf("SECURITY: User attempted to access non-managed namespace: %s", projectName)
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not an Ambient project"})
		return
	}

	project := projectFromNamespace(ns, isOpenShift)
	c.JSON(http.StatusOK, project)
}

// UpdateProject handles PUT /projects/:projectName
// On OpenShift: Updates namespace annotations for display name/description
// On Kubernetes: No-op (k8s namespaces don't have display metadata)
func UpdateProject(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)

	if reqK8s == nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sTimeout)
	defer cancel()

	// Get namespace using user's token (verifies access)
	ns, err := reqK8s.CoreV1().Namespaces().Get(ctx, projectName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		if errors.IsForbidden(err) {
			log.Printf("User forbidden to update Namespace %s: %v", projectName, err)
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to update project"})
			return
		}
		log.Printf("Failed to get Namespace %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
		return
	}

	// Validate it's an Ambient-managed namespace
	if ns.Labels["ambient-code.io/managed"] != "true" {
		log.Printf("SECURITY: User attempted to update non-managed namespace: %s", projectName)
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not an Ambient project"})
		return
	}

	// On OpenShift: Update namespace annotations (backend SA needed for namespace updates)
	if isOpenShift && K8sClientProjects != nil {
		if req.DisplayName != "" {
			if ns.Annotations == nil {
				ns.Annotations = make(map[string]string)
			}
			ns.Annotations["openshift.io/display-name"] = req.DisplayName
		}
		if req.Description != "" {
			if ns.Annotations == nil {
				ns.Annotations = make(map[string]string)
			}
			ns.Annotations["openshift.io/description"] = req.Description
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel2()

		// Update using backend SA (users can't update namespace annotations)
		_, err = K8sClientProjects.CoreV1().Namespaces().Update(ctx2, ns, v1.UpdateOptions{})
		if err != nil {
			log.Printf("Failed to update Namespace annotations for %s: %v", projectName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
			return
		}

		// Read back the updated namespace
		ctx3, cancel3 := context.WithTimeout(context.Background(), defaultK8sTimeout)
		defer cancel3()

		ns, _ = K8sClientProjects.CoreV1().Namespaces().Get(ctx3, projectName, v1.GetOptions{})
	}

	project := projectFromNamespace(ns, isOpenShift)
	c.JSON(http.StatusOK, project)
}

// DeleteProject handles DELETE /projects/:projectName
// Verifies user has access, then uses backend SA to delete namespace (both platforms)
// Namespace deletion is cluster-scoped, so regular users can't delete directly
func DeleteProject(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)

	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sTimeout)
	defer cancel()

	// Verify namespace exists and is Ambient-managed (using backend SA)
	if K8sClientProjects == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

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

	// Verify user has access (use SubjectAccessReview with user's token)
	hasAccess, err := checkUserCanAccessNamespace(reqK8s, projectName)
	if err != nil {
		log.Printf("DeleteProject: Failed to check access for %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
		return
	}

	if !hasAccess {
		log.Printf("User attempted to delete project %s without access", projectName)
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions to delete project"})
		return
	}

	// Delete the namespace using backend SA (after verifying user has access)
	ctx2, cancel2 := context.WithTimeout(context.Background(), defaultK8sTimeout)
	defer cancel2()

	err = K8sClientProjects.CoreV1().Namespaces().Delete(ctx2, projectName, v1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		log.Printf("Failed to delete namespace %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	c.Status(http.StatusNoContent)
}

// projectFromNamespace converts a Kubernetes Namespace to AmbientProject
// On OpenShift, extracts displayName and description from namespace annotations
func projectFromNamespace(ns *corev1.Namespace, isOpenShift bool) types.AmbientProject {
	status := "Active"
	if ns.Status.Phase != corev1.NamespaceActive {
		status = string(ns.Status.Phase)
	}

	displayName := ""
	description := ""

	// On OpenShift, extract display metadata from annotations
	if isOpenShift && ns.Annotations != nil {
		displayName = ns.Annotations["openshift.io/display-name"]
		description = ns.Annotations["openshift.io/description"]
	}

	return types.AmbientProject{
		Name:              ns.Name,
		DisplayName:       displayName,
		Description:       description,
		Labels:            ns.Labels,
		Annotations:       ns.Annotations,
		CreationTimestamp: ns.CreationTimestamp.Format(time.RFC3339),
		Status:            status,
		IsOpenShift:       isOpenShift,
	}
}
