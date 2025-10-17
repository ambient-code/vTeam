package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	authnv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Role constants for Ambient RBAC
const (
	AmbientRoleAdmin = "ambient-project-admin"
	AmbientRoleEdit  = "ambient-project-edit"
	AmbientRoleView  = "ambient-project-view"
)

// sanitizeName converts input to a Kubernetes-safe name (lowercase alphanumeric with dashes, max 63 chars)
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

// PermissionAssignment represents a user or group permission
type PermissionAssignment struct {
	SubjectType string `json:"subjectType"`
	SubjectName string `json:"subjectName"`
	Role        string `json:"role"`
}

// ListProjectPermissions handles GET /api/projects/:projectName/permissions
func ListProjectPermissions(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)

	// Prefer new label, but also include legacy group-access for backward-compat listing
	rbsAll, err := reqK8s.RbacV1().RoleBindings(projectName).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list RoleBindings in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list permissions"})
		return
	}

	validRoles := map[string]string{
		AmbientRoleAdmin: "admin",
		AmbientRoleEdit:  "edit",
		AmbientRoleView:  "view",
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

// AddProjectPermission handles POST /api/projects/:projectName/permissions
func AddProjectPermission(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)

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
		roleRefName = AmbientRoleAdmin
	case "edit":
		roleRefName = AmbientRoleEdit
	case "view":
		roleRefName = AmbientRoleView
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

// RemoveProjectPermission handles DELETE /api/projects/:projectName/permissions/:subjectType/:subjectName
func RemoveProjectPermission(c *gin.Context) {
	projectName := c.Param("projectName")
	subjectType := strings.ToLower(c.Param("subjectType"))
	subjectName := c.Param("subjectName")
	reqK8s, _ := GetK8sClientsForRequest(c)

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

// ListProjectKeys handles GET /api/projects/:projectName/keys
// Lists access keys (ServiceAccounts with label app=ambient-access-key)
func ListProjectKeys(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)

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
				case AmbientRoleAdmin:
					role = "admin"
				case AmbientRoleEdit:
					role = "edit"
				case AmbientRoleView:
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
			ki.CreatedAt = t.Format(time.RFC3339)
		}
		if lu := sa.Annotations["ambient-code.io/last-used-at"]; lu != "" {
			ki.LastUsedAt = lu
		}
		items = append(items, ki)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// CreateProjectKey handles POST /api/projects/:projectName/keys
// Creates a new access key (ServiceAccount with token and RoleBinding)
func CreateProjectKey(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)

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
		roleRefName = AmbientRoleAdmin
	case "edit":
		roleRefName = AmbientRoleEdit
	case "view":
		roleRefName = AmbientRoleView
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

	// Issue a one-time JWT token for this ServiceAccount (no audience; used as API key)
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

// DeleteProjectKey handles DELETE /api/projects/:projectName/keys/:keyId
// Deletes an access key (ServiceAccount and associated RoleBindings)
func DeleteProjectKey(c *gin.Context) {
	projectName := c.Param("projectName")
	keyID := c.Param("keyId")
	reqK8s, _ := GetK8sClientsForRequest(c)

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
