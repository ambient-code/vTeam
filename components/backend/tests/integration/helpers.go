package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// TestConfig holds configuration for integration tests
type TestConfig struct {
	Namespace       string
	CleanupEnabled  bool
	K8sClient       *kubernetes.Clientset
	DynamicClient   dynamic.Interface
	RestConfig      *rest.Config
	ServiceAccounts []string // Track created SAs for cleanup
	RoleBindings    []string // Track created RoleBindings for cleanup
}

// NewTestConfig creates a new test configuration
func NewTestConfig(t *testing.T) *TestConfig {
	t.Helper()

	namespace := os.Getenv("TEST_NAMESPACE")
	if namespace == "" {
		namespace = "ambient-code-test"
	}

	cleanupEnabled := os.Getenv("CLEANUP_RESOURCES") != "false"

	config, err := GetK8sConfig()
	require.NoError(t, err, "Failed to get Kubernetes config")

	clientset, err := kubernetes.NewForConfig(config)
	require.NoError(t, err, "Failed to create Kubernetes clientset")

	dynamicClient, err := dynamic.NewForConfig(config)
	require.NoError(t, err, "Failed to create dynamic client")

	return &TestConfig{
		Namespace:       namespace,
		CleanupEnabled:  cleanupEnabled,
		K8sClient:       clientset,
		DynamicClient:   dynamicClient,
		RestConfig:      config,
		ServiceAccounts: []string{},
		RoleBindings:    []string{},
	}
}

// GetK8sConfig returns a Kubernetes REST config
func GetK8sConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfig = filepath.Join(homeDir, ".kube", "config")
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}

// EnsureNamespace ensures the test namespace exists
func (tc *TestConfig) EnsureNamespace(t *testing.T, ctx context.Context) {
	t.Helper()

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: tc.Namespace,
			Labels: map[string]string{
				"test": "oauth-scope-restriction",
			},
		},
	}

	_, err := tc.K8sClient.CoreV1().Namespaces().Get(ctx, tc.Namespace, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = tc.K8sClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		require.NoError(t, err, "Failed to create test namespace")
		t.Logf("Created test namespace: %s", tc.Namespace)
	} else {
		require.NoError(t, err, "Failed to check namespace existence")
		t.Logf("Using existing test namespace: %s", tc.Namespace)
	}
}

// CreateServiceAccount creates a service account for testing
func (tc *TestConfig) CreateServiceAccount(t *testing.T, ctx context.Context, name string) *corev1.ServiceAccount {
	t.Helper()

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: tc.Namespace,
			Labels: map[string]string{
				"test": "oauth-scope-restriction",
			},
		},
	}

	created, err := tc.K8sClient.CoreV1().ServiceAccounts(tc.Namespace).Create(ctx, sa, metav1.CreateOptions{})
	require.NoError(t, err, "Failed to create service account")

	tc.ServiceAccounts = append(tc.ServiceAccounts, name)
	t.Logf("Created service account: %s/%s", tc.Namespace, name)

	return created
}

// CreateRoleBinding creates a role binding for testing
func (tc *TestConfig) CreateRoleBinding(t *testing.T, ctx context.Context, name, role, saName string) *rbacv1.RoleBinding {
	t.Helper()

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: tc.Namespace,
			Labels: map[string]string{
				"test": "oauth-scope-restriction",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     role,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: tc.Namespace,
			},
		},
	}

	created, err := tc.K8sClient.RbacV1().RoleBindings(tc.Namespace).Create(ctx, rb, metav1.CreateOptions{})
	require.NoError(t, err, "Failed to create role binding")

	tc.RoleBindings = append(tc.RoleBindings, name)
	t.Logf("Created role binding: %s/%s", tc.Namespace, name)

	return created
}

// PerformSelfSubjectAccessReview performs a SelfSubjectAccessReview
func (tc *TestConfig) PerformSelfSubjectAccessReview(t *testing.T, ctx context.Context, resource, verb, namespace string) bool {
	t.Helper()

	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  resource,
				Verb:      verb,
				Namespace: namespace,
			},
		},
	}

	result, err := tc.K8sClient.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
	require.NoError(t, err, "Failed to perform SelfSubjectAccessReview")

	return result.Status.Allowed
}

// GetServiceAccountToken gets a token for a service account
func (tc *TestConfig) GetServiceAccountToken(t *testing.T, ctx context.Context, saName string) string {
	t.Helper()

	// Wait for service account to have a token
	sa, err := tc.K8sClient.CoreV1().ServiceAccounts(tc.Namespace).Get(ctx, saName, metav1.GetOptions{})
	require.NoError(t, err, "Failed to get service account")

	// In modern Kubernetes, we need to create a token request
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: int64Ptr(3600), // 1 hour
		},
	}

	result, err := tc.K8sClient.CoreV1().ServiceAccounts(tc.Namespace).CreateToken(
		ctx,
		sa.Name,
		tokenRequest,
		metav1.CreateOptions{},
	)
	require.NoError(t, err, "Failed to create token for service account")

	return result.Status.Token
}

// Cleanup removes all test resources
func (tc *TestConfig) Cleanup(t *testing.T, ctx context.Context) {
	t.Helper()

	if !tc.CleanupEnabled {
		t.Logf("Cleanup disabled, keeping test resources in namespace: %s", tc.Namespace)
		return
	}

	t.Logf("Cleaning up test resources in namespace: %s", tc.Namespace)

	// Delete RoleBindings
	for _, rbName := range tc.RoleBindings {
		err := tc.K8sClient.RbacV1().RoleBindings(tc.Namespace).Delete(ctx, rbName, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			t.Logf("Warning: Failed to delete role binding %s: %v", rbName, err)
		}
	}

	// Delete ServiceAccounts
	for _, saName := range tc.ServiceAccounts {
		err := tc.K8sClient.CoreV1().ServiceAccounts(tc.Namespace).Delete(ctx, saName, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			t.Logf("Warning: Failed to delete service account %s: %v", saName, err)
		}
	}

	t.Logf("Cleanup completed")
}

// Helper function to create int64 pointer
func int64Ptr(i int64) *int64 {
	return &i
}
