package handlers

import (
	"context"
	"log"
	"net/http"

	"ambient-code-backend/internal/middleware"

	"github.com/gin-gonic/gin"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PermissionItem struct {
	SubjectType string `json:"subjectType"`
	SubjectName string `json:"subjectName"`
	Role        string `json:"role"`
	Namespace   string `json:"namespace,omitempty"`
}

// ListProjectPermissions lists all permissions (RoleBindings) in the project namespace
func ListProjectPermissions(c *gin.Context) {
	project := c.GetString("project")
	reqK8s, _ := middleware.GetK8sClientsForRequest(c)

	// List RoleBindings in the project namespace
	roleBindings, err := reqK8s.RbacV1().RoleBindings(project).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list RoleBindings in %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list permissions"})
		return
	}

	// List ClusterRoleBindings that reference this namespace
	clusterRoleBindings, err := reqK8s.RbacV1().ClusterRoleBindings().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list ClusterRoleBindings: %v", err)
		// Continue with just RoleBindings
		clusterRoleBindings = &rbacv1.ClusterRoleBindingList{}
	}

	permissions := []PermissionItem{}

	// Process RoleBindings
	for _, rb := range roleBindings.Items {
		for _, subject := range rb.Subjects {
			if subject.Namespace == "" || subject.Namespace == project {
				permissions = append(permissions, PermissionItem{
					SubjectType: subject.Kind,
					SubjectName: subject.Name,
					Role:        rb.RoleRef.Name,
					Namespace:   project,
				})
			}
		}
	}

	// Process ClusterRoleBindings that might affect this namespace
	for _, crb := range clusterRoleBindings.Items {
		for _, subject := range crb.Subjects {
			if subject.Namespace == project {
				permissions = append(permissions, PermissionItem{
					SubjectType: subject.Kind,
					SubjectName: subject.Name,
					Role:        crb.RoleRef.Name,
					Namespace:   "",
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

// AddProjectPermission adds a permission (creates a RoleBinding) in the project namespace
func AddProjectPermission(c *gin.Context) {
	project := c.GetString("project")
	reqK8s, _ := middleware.GetK8sClientsForRequest(c)

	var req struct {
		SubjectType string `json:"subjectType" binding:"required"` // User, Group, ServiceAccount
		SubjectName string `json:"subjectName" binding:"required"`
		Role        string `json:"role" binding:"required"`        // Role or ClusterRole name
		RoleType    string `json:"roleType,omitempty"`             // "Role" or "ClusterRole"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Default to ClusterRole if not specified
	roleType := req.RoleType
	if roleType == "" {
		roleType = "ClusterRole"
	}

	// Create RoleBinding name based on subject and role
	bindingName := req.SubjectName + "-" + req.Role + "-binding"

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      bindingName,
			Namespace: project,
			Labels: map[string]string{
				"app":                       "ambient-permissions",
				"ambient-code.io/subject":   req.SubjectName,
				"ambient-code.io/role":      req.Role,
				"ambient-code.io/managed":   "true",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      req.SubjectType,
				Name:      req.SubjectName,
				Namespace: project,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     roleType,
			Name:     req.Role,
		},
	}

	// For ServiceAccount subjects, ensure namespace is set
	if req.SubjectType == "ServiceAccount" {
		roleBinding.Subjects[0].Namespace = project
	} else {
		// For User and Group, namespace should be empty
		roleBinding.Subjects[0].Namespace = ""
	}

	_, err := reqK8s.RbacV1().RoleBindings(project).Create(context.TODO(), roleBinding, v1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Permission already exists"})
			return
		}
		log.Printf("Failed to create RoleBinding in %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create permission"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Permission created successfully"})
}

// RemoveProjectPermission removes a permission (deletes a RoleBinding) from the project namespace
func RemoveProjectPermission(c *gin.Context) {
	project := c.GetString("project")
	subjectType := c.Param("subjectType")
	subjectName := c.Param("subjectName")
	reqK8s, _ := middleware.GetK8sClientsForRequest(c)

	// List all RoleBindings and find ones that match the subject
	roleBindings, err := reqK8s.RbacV1().RoleBindings(project).List(context.TODO(), v1.ListOptions{
		LabelSelector: "ambient-code.io/subject=" + subjectName,
	})
	if err != nil {
		log.Printf("Failed to list RoleBindings for subject %s in %s: %v", subjectName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find permissions"})
		return
	}

	deletedCount := 0
	for _, rb := range roleBindings.Items {
		// Check if any subject matches
		for _, subject := range rb.Subjects {
			if subject.Kind == subjectType && subject.Name == subjectName {
				err := reqK8s.RbacV1().RoleBindings(project).Delete(context.TODO(), rb.Name, v1.DeleteOptions{})
				if err != nil && !errors.IsNotFound(err) {
					log.Printf("Failed to delete RoleBinding %s in %s: %v", rb.Name, project, err)
				} else {
					deletedCount++
				}
				break
			}
		}
	}

	if deletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No permissions found for subject"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permissions removed successfully", "deletedCount": deletedCount})
}