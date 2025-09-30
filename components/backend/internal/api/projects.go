// Projects API Component
// C4 Architecture: Multi-tenant project CRUD operations
// Manages Project Custom Resources in Kubernetes

package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// ProjectsAPI handles multi-tenant project CRUD operations
type ProjectsAPI struct {
	dynamicClient dynamic.Interface
	gvr          schema.GroupVersionResource
}

// NewProjectsAPI creates a new Projects API handler
func NewProjectsAPI(client dynamic.Interface) *ProjectsAPI {
	return &ProjectsAPI{
		dynamicClient: client,
		gvr: schema.GroupVersionResource{
			Group:    "vteam.io",
			Version:  "v1alpha1",
			Resource: "projects",
		},
	}
}

// Project represents a multi-tenant project
type Project struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Owner       string            `json:"owner"`
	Members     []string          `json:"members"`
	Labels      map[string]string `json:"labels"`
	Quotas      ResourceQuotas    `json:"quotas"`
}

// ResourceQuotas defines resource limits for a project
type ResourceQuotas struct {
	MaxSessions      int    `json:"maxSessions"`
	MaxCPU          string `json:"maxCpu"`
	MaxMemory       string `json:"maxMemory"`
	MaxStorage      string `json:"maxStorage"`
}

// ListProjects returns all projects accessible to the user
func (p *ProjectsAPI) ListProjects(c *gin.Context) {
	// Extract user from context (set by RBAC middleware)
	user := c.GetString("user")
	namespace := c.DefaultQuery("namespace", "vteam-projects")

	// List projects with label selector for multi-tenancy
	labelSelector := fmt.Sprintf("owner=%s", user)
	if c.Query("all") == "true" && c.GetBool("isAdmin") {
		labelSelector = ""
	}

	projects, err := p.dynamicClient.Resource(p.gvr).Namespace(namespace).List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector: labelSelector,
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to list projects: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, projects.Items)
}

// CreateProject creates a new project with namespace and RBAC
func (p *ProjectsAPI) CreateProject(c *gin.Context) {
	var project Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid project data: %v", err),
		})
		return
	}

	// Set owner from authenticated user
	project.Owner = c.GetString("user")

	// Create Project Custom Resource
	projectCR := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.io/v1alpha1",
			"kind":       "Project",
			"metadata": map[string]interface{}{
				"name": project.Name,
				"labels": map[string]interface{}{
					"owner": project.Owner,
					"app":   "vteam",
				},
			},
			"spec": map[string]interface{}{
				"description": project.Description,
				"owner":       project.Owner,
				"members":     project.Members,
				"quotas": map[string]interface{}{
					"maxSessions": project.Quotas.MaxSessions,
					"maxCpu":      project.Quotas.MaxCPU,
					"maxMemory":   project.Quotas.MaxMemory,
					"maxStorage":  project.Quotas.MaxStorage,
				},
			},
		},
	}

	// Create the Project CR
	created, err := p.dynamicClient.Resource(p.gvr).Namespace("vteam-projects").Create(
		context.TODO(),
		projectCR,
		metav1.CreateOptions{},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create project: %v", err),
		})
		return
	}

	c.JSON(http.StatusCreated, created.Object)
}

// GetProject retrieves a specific project
func (p *ProjectsAPI) GetProject(c *gin.Context) {
	projectName := c.Param("name")

	project, err := p.dynamicClient.Resource(p.gvr).Namespace("vteam-projects").Get(
		context.TODO(),
		projectName,
		metav1.GetOptions{},
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Project not found: %v", err),
		})
		return
	}

	// Check access permissions
	owner, _, _ := unstructured.NestedString(project.Object, "spec", "owner")
	members, _, _ := unstructured.NestedStringSlice(project.Object, "spec", "members")

	user := c.GetString("user")
	hasAccess := owner == user || c.GetBool("isAdmin")
	for _, member := range members {
		if member == user {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this project",
		})
		return
	}

	c.JSON(http.StatusOK, project.Object)
}

// UpdateProject updates an existing project
func (p *ProjectsAPI) UpdateProject(c *gin.Context) {
	projectName := c.Param("name")
	var updates map[string]interface{}

	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid update data: %v", err),
		})
		return
	}

	// Get existing project
	existing, err := p.dynamicClient.Resource(p.gvr).Namespace("vteam-projects").Get(
		context.TODO(),
		projectName,
		metav1.GetOptions{},
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Project not found",
		})
		return
	}

	// Check ownership
	owner, _, _ := unstructured.NestedString(existing.Object, "spec", "owner")
	if owner != c.GetString("user") && !c.GetBool("isAdmin") {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only project owner can update the project",
		})
		return
	}

	// Update spec fields
	if spec, ok := updates["spec"].(map[string]interface{}); ok {
		existing.Object["spec"] = spec
	}

	// Update the Project CR
	updated, err := p.dynamicClient.Resource(p.gvr).Namespace("vteam-projects").Update(
		context.TODO(),
		existing,
		metav1.UpdateOptions{},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update project: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, updated.Object)
}

// DeleteProject deletes a project and its resources
func (p *ProjectsAPI) DeleteProject(c *gin.Context) {
	projectName := c.Param("name")

	// Get project to check ownership
	project, err := p.dynamicClient.Resource(p.gvr).Namespace("vteam-projects").Get(
		context.TODO(),
		projectName,
		metav1.GetOptions{},
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Project not found",
		})
		return
	}

	// Check ownership
	owner, _, _ := unstructured.NestedString(project.Object, "spec", "owner")
	if owner != c.GetString("user") && !c.GetBool("isAdmin") {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only project owner can delete the project",
		})
		return
	}

	// Delete the Project CR (operator will handle namespace cleanup)
	err = p.dynamicClient.Resource(p.gvr).Namespace("vteam-projects").Delete(
		context.TODO(),
		projectName,
		metav1.DeleteOptions{},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to delete project: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Project %s deleted successfully", projectName),
	})
}