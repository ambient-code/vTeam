package resources

import (
	"context"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GitConfig represents git configuration data
type GitConfig struct {
	UserName    string            `json:"userName"`
	UserEmail   string            `json:"userEmail"`
	SSHKeyPath  string            `json:"sshKeyPath,omitempty"`
	TokenPath   string            `json:"tokenPath,omitempty"`
	GlobalConfig map[string]string `json:"globalConfig,omitempty"`
}

// ConfigMapsReconciler handles reconciliation of ConfigMaps, particularly git configuration
type ConfigMapsReconciler struct {
	client     *kubernetes.Clientset
	operatorNS string
}

// NewConfigMapsReconciler creates a new ConfigMaps reconciler
func NewConfigMapsReconciler(client *kubernetes.Clientset, operatorNS string) *ConfigMapsReconciler {
	return &ConfigMapsReconciler{
		client:     client,
		operatorNS: operatorNS,
	}
}

// ReconcileGitConfig ensures git configuration ConfigMap exists in the target namespace
func (r *ConfigMapsReconciler) ReconcileGitConfig(targetNamespace string, gitConfig GitConfig) error {
	configMapName := GitConfigMapName

	// Check if ConfigMap already exists
	existingCM, err := r.client.CoreV1().ConfigMaps(targetNamespace).Get(context.TODO(), configMapName, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error checking if %s ConfigMap exists in namespace %s: %v", GitConfigMapName, targetNamespace, err)
	}

	// Prepare ConfigMap data
	data := map[string]string{
		"user.name":  gitConfig.UserName,
		"user.email": gitConfig.UserEmail,
	}

	// Add global config if provided
	if gitConfig.GlobalConfig != nil {
		for key, value := range gitConfig.GlobalConfig {
			data[key] = value
		}
	}

	// Add SSH key path if provided
	if gitConfig.SSHKeyPath != "" {
		data["ssh.keyPath"] = gitConfig.SSHKeyPath
	}

	// Add token path if provided
	if gitConfig.TokenPath != "" {
		data["token.path"] = gitConfig.TokenPath
	}

	if existingCM != nil {
		// Update existing ConfigMap
		existingCM.Data = data
		_, err = r.client.CoreV1().ConfigMaps(targetNamespace).Update(context.TODO(), existingCM, v1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update %s ConfigMap in namespace %s: %v", GitConfigMapName, targetNamespace, err)
		}
		log.Printf("Updated %s ConfigMap in namespace %s", GitConfigMapName, targetNamespace)
	} else {
		// Create new ConfigMap
		configMap := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      configMapName,
				Namespace: targetNamespace,
				Labels: map[string]string{
					ManagedLabelKey:    ManagedLabelValue,
					ConfigTypeLabelKey: GitConfigLabelValue,
				},
			},
			Data: data,
		}

		_, err = r.client.CoreV1().ConfigMaps(targetNamespace).Create(context.TODO(), configMap, v1.CreateOptions{})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				log.Printf("%s ConfigMap already exists in namespace %s", GitConfigMapName, targetNamespace)
				return nil
			}
			return fmt.Errorf("failed to create %s ConfigMap in namespace %s: %v", GitConfigMapName, targetNamespace, err)
		}
		log.Printf("Created %s ConfigMap in namespace %s", GitConfigMapName, targetNamespace)
	}

	return nil
}

// ReconcileDefaultGitConfig creates a default git configuration for a namespace
func (r *ConfigMapsReconciler) ReconcileDefaultGitConfig(targetNamespace string) error {
	defaultConfig := GitConfig{
		UserName:  "Ambient Code Runner",
		UserEmail: "runner@ambient-code.io",
		GlobalConfig: map[string]string{
			"init.defaultBranch": "main",
			"pull.rebase":        "false",
			"safe.directory":     "*",
		},
	}

	return r.ReconcileGitConfig(targetNamespace, defaultConfig)
}

// CopyGitConfigFromNamespace copies git configuration from source namespace to target namespace
func (r *ConfigMapsReconciler) CopyGitConfigFromNamespace(sourceNamespace, targetNamespace string) error {
	// Skip copying to the same namespace
	if sourceNamespace == targetNamespace {
		return nil
	}

	// Get source ConfigMap
	sourceConfigMap, err := r.client.CoreV1().ConfigMaps(sourceNamespace).Get(context.TODO(), GitConfigMapName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Source %s ConfigMap not found in namespace %s, creating default", GitConfigMapName, sourceNamespace)
			return r.ReconcileDefaultGitConfig(targetNamespace)
		}
		return fmt.Errorf("failed to get source %s ConfigMap from namespace %s: %v", GitConfigMapName, sourceNamespace, err)
	}

	// Extract git config from source
	gitConfig := GitConfig{
		UserName:     sourceConfigMap.Data["user.name"],
		UserEmail:    sourceConfigMap.Data["user.email"],
		SSHKeyPath:   sourceConfigMap.Data["ssh.keyPath"],
		TokenPath:    sourceConfigMap.Data["token.path"],
		GlobalConfig: make(map[string]string),
	}

	// Copy all other config items
	for key, value := range sourceConfigMap.Data {
		if key != "user.name" && key != "user.email" && key != "ssh.keyPath" && key != "token.path" {
			gitConfig.GlobalConfig[key] = value
		}
	}

	return r.ReconcileGitConfig(targetNamespace, gitConfig)
}

// ReconcileRunnerConfig ensures runner-specific configuration exists in the target namespace
func (r *ConfigMapsReconciler) ReconcileRunnerConfig(targetNamespace string, config map[string]string) error {
	configMapName := RunnerConfigMapName

	// Check if ConfigMap already exists
	existingCM, err := r.client.CoreV1().ConfigMaps(targetNamespace).Get(context.TODO(), configMapName, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error checking if %s ConfigMap exists in namespace %s: %v", RunnerConfigMapName, targetNamespace, err)
	}

	if existingCM != nil {
		// Update existing ConfigMap
		existingCM.Data = config
		_, err = r.client.CoreV1().ConfigMaps(targetNamespace).Update(context.TODO(), existingCM, v1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update %s ConfigMap in namespace %s: %v", RunnerConfigMapName, targetNamespace, err)
		}
		log.Printf("Updated %s ConfigMap in namespace %s", RunnerConfigMapName, targetNamespace)
	} else {
		// Create new ConfigMap
		configMap := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      configMapName,
				Namespace: targetNamespace,
				Labels: map[string]string{
					ManagedLabelKey:    ManagedLabelValue,
					ConfigTypeLabelKey: RunnerConfigLabelValue,
				},
			},
			Data: config,
		}

		_, err = r.client.CoreV1().ConfigMaps(targetNamespace).Create(context.TODO(), configMap, v1.CreateOptions{})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				log.Printf("%s ConfigMap already exists in namespace %s", RunnerConfigMapName, targetNamespace)
				return nil
			}
			return fmt.Errorf("failed to create %s ConfigMap in namespace %s: %v", RunnerConfigMapName, targetNamespace, err)
		}
		log.Printf("Created %s ConfigMap in namespace %s", RunnerConfigMapName, targetNamespace)
	}

	return nil
}

// ListManagedConfigMaps returns a list of ConfigMaps managed by this reconciler in the given namespace
func (r *ConfigMapsReconciler) ListManagedConfigMaps(namespace string) ([]corev1.ConfigMap, error) {
	configMaps, err := r.client.CoreV1().ConfigMaps(namespace).List(context.TODO(), v1.ListOptions{
		LabelSelector: ManagedLabelKey + "=" + ManagedLabelValue,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list managed ConfigMaps in namespace %s: %v", namespace, err)
	}

	return configMaps.Items, nil
}

// DeleteManagedConfigMaps removes all managed ConfigMaps from the given namespace
func (r *ConfigMapsReconciler) DeleteManagedConfigMaps(namespace string) error {
	configMaps, err := r.ListManagedConfigMaps(namespace)
	if err != nil {
		return fmt.Errorf("failed to list managed ConfigMaps for deletion: %v", err)
	}

	for _, cm := range configMaps {
		err := r.client.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), cm.Name, v1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Printf("Failed to delete managed ConfigMap %s in namespace %s: %v", cm.Name, namespace, err)
		} else {
			log.Printf("Deleted managed ConfigMap %s from namespace %s", cm.Name, namespace)
		}
	}

	return nil
}

// GetGitConfig retrieves the git configuration from a namespace
func (r *ConfigMapsReconciler) GetGitConfig(namespace string) (*GitConfig, error) {
	configMap, err := r.client.CoreV1().ConfigMaps(namespace).Get(context.TODO(), GitConfigMapName, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get %s ConfigMap from namespace %s: %v", GitConfigMapName, namespace, err)
	}

	gitConfig := &GitConfig{
		UserName:     configMap.Data["user.name"],
		UserEmail:    configMap.Data["user.email"],
		SSHKeyPath:   configMap.Data["ssh.keyPath"],
		TokenPath:    configMap.Data["token.path"],
		GlobalConfig: make(map[string]string),
	}

	// Extract all other config items
	for key, value := range configMap.Data {
		if key != "user.name" && key != "user.email" && key != "ssh.keyPath" && key != "token.path" {
			gitConfig.GlobalConfig[key] = value
		}
	}

	return gitConfig, nil
}