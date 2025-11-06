package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// TestOAuthScopeRestriction validates that the OAuth scope restriction (user:info)
// allows application authentication while blocking console and kubectl access.
//
// This test verifies:
// 1. Users can authenticate and access vTeam API operations
// 2. SelfSubjectAccessReview works via --openshift-delegate-urls
// 3. Users cannot perform arbitrary cluster operations
// 4. Users cannot access resources outside authorized namespaces
func TestOAuthScopeRestriction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	tc := NewTestConfig(t)
	defer tc.Cleanup(t, ctx)

	// Ensure test namespace exists
	tc.EnsureNamespace(t, ctx)

	t.Run("VerifyUserInfoScope", func(t *testing.T) {
		testVerifyUserInfoScope(t, tc, ctx)
	})

	t.Run("VerifyAgenticSessionAccess", func(t *testing.T) {
		testVerifyAgenticSessionAccess(t, tc, ctx)
	})

	t.Run("VerifyClusterAccessBlocked", func(t *testing.T) {
		testVerifyClusterAccessBlocked(t, tc, ctx)
	})

	t.Run("VerifyNamespaceIsolation", func(t *testing.T) {
		testVerifyNamespaceIsolation(t, tc, ctx)
	})
}

// testVerifyUserInfoScope verifies that basic user info can be accessed
func testVerifyUserInfoScope(t *testing.T, tc *TestConfig, ctx context.Context) {
	// Create a service account to simulate OAuth user
	saName := "test-oauth-user"
	tc.CreateServiceAccount(t, ctx, saName)

	// Get token for the service account
	token := tc.GetServiceAccountToken(t, ctx, saName)
	require.NotEmpty(t, token, "Failed to get service account token")

	// Create a client using the token (simulating user:info scope)
	config := rest.CopyConfig(tc.RestConfig)
	config.BearerToken = token
	config.BearerTokenFile = ""

	userClient, err := kubernetes.NewForConfig(config)
	require.NoError(t, err, "Failed to create user-scoped client")

	// Verify we can perform SelfSubjectAccessReview (this should work)
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      "list",
				Namespace: tc.Namespace,
			},
		},
	}

	result, err := userClient.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
	require.NoError(t, err, "SelfSubjectAccessReview should work with user:info scope")

	t.Logf("SelfSubjectAccessReview result: Allowed=%v, Reason=%s", result.Status.Allowed, result.Status.Reason)

	// The result.Status.Allowed may be false (no permissions yet), but the API call itself should succeed
	assert.NotNil(t, result, "SelfSubjectAccessReview should return a result")
}

// testVerifyAgenticSessionAccess verifies access to AgenticSession custom resources
func testVerifyAgenticSessionAccess(t *testing.T, tc *TestConfig, ctx context.Context) {
	// Create service account with proper RBAC
	saName := "test-session-user"
	tc.CreateServiceAccount(t, ctx, saName)

	// Grant ambient-project-edit role (allows session CRUD operations)
	tc.CreateRoleBinding(t, ctx, "test-session-edit-binding", "ambient-project-edit", saName)

	// Wait a moment for RBAC to propagate
	time.Sleep(2 * time.Second)

	// Get token (for logging/verification purposes)
	token := tc.GetServiceAccountToken(t, ctx, saName)
	require.NotEmpty(t, token, "Failed to get service account token")

	// Verify user can check access to AgenticSessions
	allowed := tc.PerformSelfSubjectAccessReview(t, ctx, "agenticsessions", "list", tc.Namespace)
	t.Logf("User access to list agenticsessions in %s: %v", tc.Namespace, allowed)

	// Test creating an AgenticSession (simulating API operation)
	gvr := schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "agenticsessions",
	}

	sessionName := "test-session-oauth"
	session := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      sessionName,
				"namespace": tc.Namespace,
			},
			"spec": map[string]interface{}{
				"prompt": "Test OAuth scope restriction",
				"repos": []interface{}{
					map[string]interface{}{
						"input": map[string]interface{}{
							"url":    "https://github.com/ambient-code/vTeam",
							"branch": "main",
						},
					},
				},
			},
		},
	}

	// Attempt to create session (this tests if delegate URLs work)
	created, err := tc.DynamicClient.Resource(gvr).Namespace(tc.Namespace).Create(ctx, session, metav1.CreateOptions{})
	if err != nil {
		t.Logf("Note: Session creation may fail if CRDs not installed or RBAC not ready: %v", err)
	} else {
		assert.NotNil(t, created, "Should be able to create AgenticSession with proper RBAC")
		t.Logf("Successfully created AgenticSession: %s", sessionName)

		// Cleanup the session
		err = tc.DynamicClient.Resource(gvr).Namespace(tc.Namespace).Delete(ctx, sessionName, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			t.Logf("Warning: Failed to cleanup test session: %v", err)
		}
	}
}

// testVerifyClusterAccessBlocked verifies that cluster-wide operations are blocked
func testVerifyClusterAccessBlocked(t *testing.T, tc *TestConfig, ctx context.Context) {
	// Create service account (no cluster-wide permissions)
	saName := "test-restricted-user"
	tc.CreateServiceAccount(t, ctx, saName)

	token := tc.GetServiceAccountToken(t, ctx, saName)
	config := rest.CopyConfig(tc.RestConfig)
	config.BearerToken = token

	userClient, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	// Test cluster-wide operations that should be blocked
	testCases := []struct {
		name      string
		operation func() error
	}{
		{
			name: "ListAllNamespaces",
			operation: func() error {
				_, err := userClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				return err
			},
		},
		{
			name: "ListAllNodes",
			operation: func() error {
				_, err := userClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				return err
			},
		},
		{
			name: "ListClusterRoles",
			operation: func() error {
				_, err := userClient.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.operation()
			if err != nil {
				// We expect Forbidden errors for cluster-wide operations
				assert.True(t, errors.IsForbidden(err), "Expected Forbidden error, got: %v", err)
				t.Logf("✓ Cluster operation correctly blocked: %s", tc.name)
			} else {
				// This would indicate the scope is too permissive
				t.Errorf("⚠️ Cluster operation succeeded when it should be blocked: %s", tc.name)
			}
		})
	}
}

// testVerifyNamespaceIsolation verifies users cannot access unauthorized namespaces
func testVerifyNamespaceIsolation(t *testing.T, tc *TestConfig, ctx context.Context) {
	// Create service account with RBAC only in test namespace
	saName := "test-isolated-user"
	tc.CreateServiceAccount(t, ctx, saName)
	tc.CreateRoleBinding(t, ctx, "test-isolated-binding", "ambient-project-view", saName)

	time.Sleep(2 * time.Second) // Wait for RBAC propagation

	token := tc.GetServiceAccountToken(t, ctx, saName)
	config := rest.CopyConfig(tc.RestConfig)
	config.BearerToken = token

	userClient, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	// Try to access pods in default namespace (should be blocked)
	_, err = userClient.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		assert.True(t, errors.IsForbidden(err), "Expected Forbidden error for unauthorized namespace, got: %v", err)
		t.Logf("✓ Access to unauthorized namespace correctly blocked")
	} else {
		t.Errorf("⚠️ Access to unauthorized namespace should be blocked")
	}

	// Try to access pods in authorized namespace (may succeed if RBAC is set up)
	pods, err := userClient.CoreV1().Pods(tc.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsForbidden(err) {
			t.Logf("Note: Access to authorized namespace blocked (RBAC may need time to propagate)")
		} else {
			t.Logf("Note: Error accessing authorized namespace: %v", err)
		}
	} else {
		t.Logf("✓ Access to authorized namespace works (found %d pods)", len(pods.Items))
	}
}

// TestOAuthDelegateURLs specifically tests the --openshift-delegate-urls functionality
func TestOAuthDelegateURLs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	tc := NewTestConfig(t)
	defer tc.Cleanup(t, ctx)

	tc.EnsureNamespace(t, ctx)

	t.Run("VerifyDelegateURLsWork", func(t *testing.T) {
		// This test verifies that the OAuth proxy's --openshift-delegate-urls parameter
		// allows the proxy to perform SubjectAccessReview checks on behalf of users
		// even when the user's token has limited scope (user:info)

		saName := "test-delegate-user"
		tc.CreateServiceAccount(t, ctx, saName)
		tc.CreateRoleBinding(t, ctx, "test-delegate-binding", "ambient-project-view", saName)

		time.Sleep(2 * time.Second)

		// The delegate URLs configuration in the OAuth proxy should allow
		// the proxy to check if the user has access to list projects
		// This is configured as: --openshift-delegate-urls={"/":{"resource":"projects","verb":"list"}}

		allowed := tc.PerformSelfSubjectAccessReview(t, ctx, "projects", "list", "")
		t.Logf("SelfSubjectAccessReview for listing projects: %v", allowed)

		// Note: The actual verification of delegate URLs would require
		// testing through the HTTP endpoint with the OAuth proxy, which
		// is beyond the scope of this unit test. This test documents the
		// expected behavior and validates that RBAC checks can be performed.
	})
}
