// Package handlers provides HTTP handlers for the backend API.
// This file contains authentication and authorization functions for project management.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	authv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetClusterInfo handles GET /cluster-info
// Returns information about the cluster type (OpenShift vs vanilla Kubernetes)
// This endpoint does not require authentication as it's public cluster information
func GetClusterInfo(c *gin.Context) {
	isOpenShift := isOpenShiftCluster()

	c.JSON(http.StatusOK, gin.H{
		"isOpenShift": isOpenShift,
	})
}

// checkUserCanAccessNamespace uses SelfSubjectAccessReview to verify if user can access a namespace
// This is the proper Kubernetes-native way - lets RBAC engine determine access from ALL sources
// (RoleBindings, ClusterRoleBindings, groups, etc.)
func checkUserCanAccessNamespace(userClient *kubernetes.Clientset, namespace string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check if user can list agenticsessions in the namespace (a good proxy for project access)
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      "list",
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
			},
		},
	}

	result, err := userClient.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, v1.CreateOptions{})
	if err != nil {
		return false, err
	}

	return result.Status.Allowed, nil
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
