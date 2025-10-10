package handlers

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Dependencies injected from main package
var (
	BaseKubeConfig *rest.Config
	K8sClientMw    *kubernetes.Clientset
)

// Helper functions and types
var (
	BoolPtr   = func(b bool) *bool { return &b }
	StringPtr = func(s string) *string { return &s }
)

// ContentListItem represents a content list item for file browsing
type ContentListItem struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
}

// GetK8sClientsForRequest returns K8s typed and dynamic clients using the caller's token when provided.
// It supports both Authorization: Bearer and X-Forwarded-Access-Token and NEVER falls back to the backend service account.
// Returns nil, nil if no valid user token is provided - all API operations require user authentication.
func GetK8sClientsForRequest(c *gin.Context) (*kubernetes.Clientset, dynamic.Interface) {
	// Prefer Authorization header (Bearer <token>)
	rawAuth := c.GetHeader("Authorization")
	rawFwd := c.GetHeader("X-Forwarded-Access-Token")
	tokenSource := "none"
	token := rawAuth

	if token != "" {
		tokenSource = "authorization"
		parts := strings.SplitN(token, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			token = strings.TrimSpace(parts[1])
		} else {
			token = strings.TrimSpace(token)
		}
	}
	// Fallback to X-Forwarded-Access-Token
	if token == "" {
		if rawFwd != "" {
			tokenSource = "x-forwarded-access-token"
		}
		token = rawFwd
	}

	// Debug: basic auth header state (do not log token)
	hasAuthHeader := strings.TrimSpace(rawAuth) != ""
	hasFwdToken := strings.TrimSpace(rawFwd) != ""

	if token != "" && BaseKubeConfig != nil {
		cfg := *BaseKubeConfig
		cfg.BearerToken = token
		// Ensure we do NOT fall back to the in-cluster SA token or other auth providers
		cfg.BearerTokenFile = ""
		cfg.AuthProvider = nil
		cfg.ExecProvider = nil
		cfg.Username = ""
		cfg.Password = ""

		kc, err1 := kubernetes.NewForConfig(&cfg)
		dc, err2 := dynamic.NewForConfig(&cfg)

		if err1 == nil && err2 == nil {

			// Best-effort update last-used for service account tokens
			updateAccessKeyLastUsedAnnotation(c)
			return kc, dc
		}
		// Token provided but client build failed – treat as invalid token
		log.Printf("Failed to build user-scoped k8s clients (source=%s tokenLen=%d) typedErr=%v dynamicErr=%v for %s", tokenSource, len(token), err1, err2, c.FullPath())
		return nil, nil
	} else {
		// No token provided
		log.Printf("No user token found for %s (hasAuthHeader=%t hasFwdToken=%t)", c.FullPath(), hasAuthHeader, hasFwdToken)
		return nil, nil
	}
}

// updateAccessKeyLastUsedAnnotation attempts to update the ServiceAccount's last-used annotation
// when the incoming token is a ServiceAccount JWT. Uses the backend service account client strictly
// for this telemetry update and only for SAs labeled app=ambient-access-key. Best-effort; errors ignored.
func updateAccessKeyLastUsedAnnotation(c *gin.Context) {
	// Parse Authorization header
	rawAuth := c.GetHeader("Authorization")
	parts := strings.SplitN(rawAuth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return
	}

	// Decode JWT payload (second segment)
	segs := strings.Split(token, ".")
	if len(segs) < 2 {
		return
	}
	payloadB64 := segs[1]
	// JWT uses base64url without padding; add padding if necessary
	if m := len(payloadB64) % 4; m != 0 {
		payloadB64 += strings.Repeat("=", 4-m)
	}
	data, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}
	// Expect sub like: system:serviceaccount:<namespace>:<sa-name>
	sub, _ := payload["sub"].(string)
	const prefix = "system:serviceaccount:"
	if !strings.HasPrefix(sub, prefix) {
		return
	}
	rest := strings.TrimPrefix(sub, prefix)
	parts2 := strings.SplitN(rest, ":", 2)
	if len(parts2) != 2 {
		return
	}
	ns := parts2[0]
	saName := parts2[1]

	// Backend client must exist
	if K8sClientMw == nil {
		return
	}

	// Ensure the SA is an Ambient access key (label check) before writing
	saObj, err := K8sClientMw.CoreV1().ServiceAccounts(ns).Get(c.Request.Context(), saName, v1.GetOptions{})
	if err != nil {
		return
	}
	if saObj.Labels == nil || saObj.Labels["app"] != "ambient-access-key" {
		return
	}

	// Patch the annotation
	now := time.Now().Format(time.RFC3339)
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				"ambient-code.io/last-used-at": now,
			},
		},
	}
	b, err := json.Marshal(patch)
	if err != nil {
		return
	}
	_, err = K8sClientMw.CoreV1().ServiceAccounts(ns).Patch(c.Request.Context(), saName, types.MergePatchType, b, v1.PatchOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to update last-used annotation for SA %s/%s: %v", ns, saName, err)
	}
}

// ExtractServiceAccountFromAuth extracts namespace and ServiceAccount name from the Authorization Bearer JWT 'sub' claim
// Returns (namespace, saName, true) when a SA subject is present, otherwise ("","",false)
func ExtractServiceAccountFromAuth(c *gin.Context) (string, string, bool) {
	rawAuth := c.GetHeader("Authorization")
	parts := strings.SplitN(rawAuth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", "", false
	}
	segs := strings.Split(token, ".")
	if len(segs) < 2 {
		return "", "", false
	}
	payloadB64 := segs[1]
	if m := len(payloadB64) % 4; m != 0 {
		payloadB64 += strings.Repeat("=", 4-m)
	}
	data, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", "", false
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", "", false
	}
	sub, _ := payload["sub"].(string)
	const prefix = "system:serviceaccount:"
	if !strings.HasPrefix(sub, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(sub, prefix)
	parts2 := strings.SplitN(rest, ":", 2)
	if len(parts2) != 2 {
		return "", "", false
	}
	return parts2[0], parts2[1], true
}

// ValidateProjectContext is middleware for project context validation
func ValidateProjectContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow token via query parameter for websocket/agent callers
		if c.GetHeader("Authorization") == "" && c.GetHeader("X-Forwarded-Access-Token") == "" {
			if qp := strings.TrimSpace(c.Query("token")); qp != "" {
				c.Request.Header.Set("Authorization", "Bearer "+qp)
			}
		}
		// Require user/API key token; do not fall back to service account
		if c.GetHeader("Authorization") == "" && c.GetHeader("X-Forwarded-Access-Token") == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
			c.Abort()
			return
		}
		reqK8s, _ := GetK8sClientsForRequest(c)
		if reqK8s == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
			c.Abort()
			return
		}
		// Prefer project from route param; fallback to header for backward compatibility
		projectHeader := c.Param("projectName")
		if projectHeader == "" {
			projectHeader = c.GetHeader("X-OpenShift-Project")
		}
		if projectHeader == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Project is required in path /api/projects/:projectName or X-OpenShift-Project header"})
			c.Abort()
			return
		}

		// Ensure the caller has at least list permission on agenticsessions in the namespace
		ssar := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authv1.ResourceAttributes{
					Group:     "vteam.ambient-code",
					Resource:  "agenticsessions",
					Verb:      "list",
					Namespace: projectHeader,
				},
			},
		}
		res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(c.Request.Context(), ssar, v1.CreateOptions{})
		if err != nil {
			log.Printf("validateProjectContext: SSAR failed for %s: %v", projectHeader, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to perform access review"})
			c.Abort()
			return
		}
		if !res.Status.Allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to access project"})
			c.Abort()
			return
		}

		// Store project in context for handlers
		c.Set("project", projectHeader)
		c.Next()
	}
}
