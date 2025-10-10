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
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Package-level variables for project handlers (set from main package)
var (
	GetOpenShiftProjectResource func() schema.GroupVersionResource
)

// ListProjects handles GET /projects
func ListProjects(c *gin.Context) {
	_, reqDyn := GetK8sClientsForRequest(c)

	// List OpenShift Projects the user can see; filter to Ambient-managed
	projGvr := GetOpenShiftProjectResource()
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

	var projects []types.AmbientProject
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

		project := types.AmbientProject{
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

// CreateProject handles POST /projects
func CreateProject(c *gin.Context) {
	reqK8s, _ := GetK8sClientsForRequest(c)
	var req types.CreateProjectRequest
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

	project := types.AmbientProject{
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

// GetProject handles GET /projects/:projectName
func GetProject(c *gin.Context) {
	projectName := c.Param("projectName")
	_, reqDyn := GetK8sClientsForRequest(c)

	// Read OpenShift Project (user context) and validate Ambient label
	projGvr := GetOpenShiftProjectResource()
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

	project := types.AmbientProject{
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

// DeleteProject handles DELETE /projects/:projectName
func DeleteProject(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)
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

// UpdateProject handles PUT /projects/:projectName
// Update basic project metadata (annotations)
func UpdateProject(c *gin.Context) {
	projectName := c.Param("projectName")
	_, reqDyn := GetK8sClientsForRequest(c)

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
	projGvr := GetOpenShiftProjectResource()
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

	project := types.AmbientProject{
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
