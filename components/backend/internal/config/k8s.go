package config

import (
	"fmt"
	"os"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	K8sClient      *kubernetes.Clientset
	BaseKubeConfig *rest.Config
)

// InitK8sClients initializes Kubernetes clients
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

	// Save base config for per-request impersonation/user-token clients
	BaseKubeConfig = config

	return nil
}

// GetK8sClientsForToken returns K8s clients configured with the provided token
func GetK8sClientsForToken(token string) (*kubernetes.Clientset, dynamic.Interface, error) {
	if BaseKubeConfig == nil {
		return nil, nil, fmt.Errorf("base kubernetes config not initialized")
	}

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

	if err1 != nil {
		return nil, nil, fmt.Errorf("failed to create typed client: %v", err1)
	}
	if err2 != nil {
		return nil, nil, fmt.Errorf("failed to create dynamic client: %v", err2)
	}

	return kc, dc, nil
}
