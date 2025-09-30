// RBAC Middleware Component
// C4 Architecture: Authorization and multi-tenancy enforcement
// Validates user permissions and enforces access control

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RBACMiddleware handles authorization and multi-tenancy
type RBACMiddleware struct {
	clientset kubernetes.Interface
}

// NewRBACMiddleware creates a new RBAC middleware instance
func NewRBACMiddleware(clientset kubernetes.Interface) *RBACMiddleware {
	return &RBACMiddleware{
		clientset: clientset,
	}
}

// Authorize returns a middleware function that validates permissions
func (r *RBACMiddleware) Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract bearer token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Parse bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token and extract user information
		user, err := r.validateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": fmt.Sprintf("Invalid token: %v", err),
			})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user", user.Username)
		c.Set("groups", user.Groups)
		c.Set("isAdmin", r.isAdmin(user.Groups))

		// Check resource-specific permissions
		if !r.checkResourcePermission(c, user) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions for this resource",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// User represents an authenticated user
type User struct {
	Username string
	Groups   []string
	Email    string
}

// validateToken validates the OAuth token with OpenShift/Kubernetes
func (r *RBACMiddleware) validateToken(token string) (*User, error) {
	// Create a TokenReview to validate the token
	tokenReview := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: token,
		},
	}

	// Submit the TokenReview to the API server
	result, err := r.clientset.AuthorizationV1().TokenReviews().Create(
		context.TODO(),
		tokenReview,
		metav1.CreateOptions{},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %v", err)
	}

	// Check if token is valid
	if !result.Status.Authenticated {
		return nil, fmt.Errorf("token authentication failed")
	}

	// Extract user information
	user := &User{
		Username: result.Status.User.Username,
		Groups:   result.Status.User.Groups,
	}

	// Extract email from extra info if available
	if extra := result.Status.User.Extra; extra != nil {
		if emails, ok := extra["email"]; ok && len(emails) > 0 {
			user.Email = emails[0]
		}
	}

	return user, nil
}

// checkResourcePermission checks if user has permission for the requested resource
func (r *RBACMiddleware) checkResourcePermission(c *gin.Context, user *User) bool {
	path := c.Request.URL.Path
	method := c.Request.Method

	// Parse resource from path
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
	if len(parts) == 0 {
		return true // Allow access to root API
	}

	// Determine resource type and action
	var resource, verb, namespace string

	switch parts[0] {
	case "projects":
		resource = "projects"
		namespace = "vteam-projects"
	case "health":
		return true // Always allow health checks
	default:
		if len(parts) >= 2 && parts[0] == "projects" {
			// Project-specific resources
			projectName := parts[1]
			namespace = fmt.Sprintf("vteam-%s", projectName)

			if len(parts) >= 3 {
				switch parts[2] {
				case "sessions":
					resource = "agenticsessions"
				case "secrets":
					resource = "secrets"
				default:
					resource = parts[2]
				}
			}
		}
	}

	// Map HTTP method to Kubernetes verb
	switch method {
	case "GET":
		verb = "get"
		if strings.HasSuffix(path, "s") || strings.Contains(path, "list") {
			verb = "list"
		}
	case "POST":
		verb = "create"
	case "PUT", "PATCH":
		verb = "update"
	case "DELETE":
		verb = "delete"
	default:
		return false
	}

	// Perform SubjectAccessReview
	sar := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User:   user.Username,
			Groups: user.Groups,
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     "vteam.io",
				Version:   "v1alpha1",
				Resource:  resource,
			},
		},
	}

	result, err := r.clientset.AuthorizationV1().SubjectAccessReviews().Create(
		context.TODO(),
		sar,
		metav1.CreateOptions{},
	)

	if err != nil {
		// Log error but deny access on failure
		fmt.Printf("Failed to check permissions: %v\n", err)
		return false
	}

	return result.Status.Allowed
}

// isAdmin checks if user belongs to admin groups
func (r *RBACMiddleware) isAdmin(groups []string) bool {
	adminGroups := []string{
		"system:masters",
		"cluster-admins",
		"vteam-admins",
	}

	for _, group := range groups {
		for _, adminGroup := range adminGroups {
			if group == adminGroup {
				return true
			}
		}
	}

	return false
}

// ProjectAccess checks if user has access to a specific project
func (r *RBACMiddleware) ProjectAccess(projectName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.GetString("user")
		groups, _ := c.Get("groups")
		isAdmin := c.GetBool("isAdmin")

		// Admins always have access
		if isAdmin {
			c.Next()
			return
		}

		// Check if user is project member
		namespace := fmt.Sprintf("vteam-%s", projectName)

		// Create a SubjectAccessReview for the project namespace
		sar := &authv1.SubjectAccessReview{
			Spec: authv1.SubjectAccessReviewSpec{
				User:   user,
				Groups: groups.([]string),
				ResourceAttributes: &authv1.ResourceAttributes{
					Namespace: namespace,
					Verb:      "get",
					Group:     "",
					Version:   "v1",
					Resource:  "pods",
				},
			},
		}

		result, err := r.clientset.AuthorizationV1().SubjectAccessReviews().Create(
			context.TODO(),
			sar,
			metav1.CreateOptions{},
		)

		if err != nil || !result.Status.Allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied to this project",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}