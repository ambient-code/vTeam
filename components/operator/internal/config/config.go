// Package config provides Kubernetes client initialization and configuration management for the operator.
package config

import (
	"context"
	"fmt"
	"log"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Package-level variables (exported for use by handlers and services)
var (
	K8sClient     kubernetes.Interface
	DynamicClient dynamic.Interface
)

// Config holds the operator configuration
type Config struct {
	Namespace              string
	BackendNamespace       string
	AmbientCodeRunnerImage string
	ContentServiceImage    string
	ImagePullPolicy        corev1.PullPolicy
}

// InitK8sClients initializes the Kubernetes clients
func InitK8sClients() error {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	if config, err = rest.InClusterConfig(); err != nil {
		// If in-cluster config fails, try kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
		}

		if config, err = clientcmd.BuildConfigFromFlags("", kubeconfig); err != nil {
			return fmt.Errorf("failed to create Kubernetes config: %v", err)
		}
	}

	// Create standard Kubernetes client
	K8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Create dynamic client for custom resources
	DynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %v", err)
	}

	return nil
}

// DiscoverFrontendURL attempts to discover the frontend external URL from OpenShift Route or Kubernetes Ingress
// Returns empty string if not found (MCP OAuth callbacks will not work)
func DiscoverFrontendURL(namespace string) string {
	ctx := context.TODO()

	// Try OpenShift Route (most common in vTeam deployments)
	routeGVR := schema.GroupVersionResource{
		Group:    "route.openshift.io",
		Version:  "v1",
		Resource: "routes",
	}

	route, err := DynamicClient.Resource(routeGVR).Namespace(namespace).Get(ctx, "frontend", metav1.GetOptions{})
	if err == nil {
		if spec, found, _ := unstructured.NestedMap(route.Object, "spec"); found {
			if host, ok := spec["host"].(string); ok && host != "" {
				// Check TLS
				scheme := "http"
				if tls, found, _ := unstructured.NestedMap(spec, "tls"); found && tls != nil {
					scheme = "https"
				}
				url := fmt.Sprintf("%s://%s", scheme, host)
				log.Printf("Discovered frontend URL from OpenShift Route: %s", url)
				return url
			}
		}
	}

	// Try Kubernetes Ingress as fallback
	ingressGVR := schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "ingresses",
	}

	ingress, err := DynamicClient.Resource(ingressGVR).Namespace(namespace).Get(ctx, "frontend", metav1.GetOptions{})
	if err == nil {
		if spec, found, _ := unstructured.NestedMap(ingress.Object, "spec"); found {
			if rules, found, _ := unstructured.NestedSlice(spec, "rules"); found && len(rules) > 0 {
				if rule, ok := rules[0].(map[string]interface{}); ok {
					if host, ok := rule["host"].(string); ok && host != "" {
						scheme := "http"
						if tls, found, _ := unstructured.NestedSlice(spec, "tls"); found && len(tls) > 0 {
							scheme = "https"
						}
						url := fmt.Sprintf("%s://%s", scheme, host)
						log.Printf("Discovered frontend URL from Kubernetes Ingress: %s", url)
						return url
					}
				}
			}
		}
	}

	log.Printf("Warning: Could not discover frontend Route or Ingress in namespace %s - MCP OAuth will not work", namespace)
	return ""
}

// LoadConfig loads the operator configuration from environment variables
func LoadConfig() *Config {
	// Get namespace from environment or use default
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// Get backend namespace from environment or use operator namespace
	backendNamespace := os.Getenv("BACKEND_NAMESPACE")
	if backendNamespace == "" {
		backendNamespace = namespace // Default to same namespace as operator
	}

	// Get ambient-code runner image from environment or use default
	ambientCodeRunnerImage := os.Getenv("AMBIENT_CODE_RUNNER_IMAGE")
	if ambientCodeRunnerImage == "" {
		ambientCodeRunnerImage = "quay.io/ambient_code/vteam_claude_runner:latest"
	}

	// Image for per-namespace content service (defaults to backend image)
	contentServiceImage := os.Getenv("CONTENT_SERVICE_IMAGE")
	if contentServiceImage == "" {
		contentServiceImage = "quay.io/ambient_code/vteam_backend:latest"
	}

	// Get image pull policy from environment or use default
	imagePullPolicyStr := os.Getenv("IMAGE_PULL_POLICY")
	if imagePullPolicyStr == "" {
		imagePullPolicyStr = "Always"
	}
	imagePullPolicy := corev1.PullPolicy(imagePullPolicyStr)

	return &Config{
		Namespace:              namespace,
		BackendNamespace:       backendNamespace,
		AmbientCodeRunnerImage: ambientCodeRunnerImage,
		ContentServiceImage:    contentServiceImage,
		ImagePullPolicy:        imagePullPolicy,
	}
}
