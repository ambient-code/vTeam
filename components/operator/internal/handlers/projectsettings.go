package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/types"
)

// WatchProjectSettings watches for ProjectSettings resources and reconciles them
func WatchProjectSettings() {
	gvr := types.GetProjectSettingsResource()

	for {
		// Watch across all namespaces for ProjectSettings
		watcher, err := config.DynamicClient.Resource(gvr).Watch(context.TODO(), v1.ListOptions{})
		if err != nil {
			log.Printf("Failed to create ProjectSettings watcher: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Watching for ProjectSettings events...")

		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added, watch.Modified:
				obj := event.Object.(*unstructured.Unstructured)

				// Add small delay to avoid race conditions
				time.Sleep(100 * time.Millisecond)

				if err := handleProjectSettingsEvent(obj); err != nil {
					log.Printf("Error handling ProjectSettings event: %v", err)
				}
			case watch.Deleted:
				obj := event.Object.(*unstructured.Unstructured)
				settingsName := obj.GetName()
				settingsNamespace := obj.GetNamespace()
				log.Printf("ProjectSettings %s/%s deleted", settingsNamespace, settingsName)
			case watch.Error:
				obj := event.Object.(*unstructured.Unstructured)
				log.Printf("Watch error for ProjectSettings: %v", obj)
			}
		}

		log.Println("ProjectSettings watch channel closed, restarting...")
		watcher.Stop()
		time.Sleep(2 * time.Second)
	}
}

func createDefaultProjectSettings(namespaceName string) error {
	gvr := types.GetProjectSettingsResource()

	// Check if ProjectSettings already exists in this namespace (singleton named 'projectsettings')
	_, err := config.DynamicClient.Resource(gvr).Namespace(namespaceName).Get(context.TODO(), "projectsettings", v1.GetOptions{})
	if err == nil {
		log.Printf("ProjectSettings already exists in namespace %s", namespaceName)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("error checking existing ProjectSettings: %v", err)
	}

	// Create default ProjectSettings (minimal: only groupAccess)
	defaultSettings := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "ProjectSettings",
			"metadata": map[string]interface{}{
				// Enforce singleton: fixed name 'projectsettings'
				"name":      "projectsettings",
				"namespace": namespaceName,
			},
			"spec": map[string]interface{}{
				"groupAccess": []interface{}{},
			},
		},
	}

	_, err = config.DynamicClient.Resource(gvr).Namespace(namespaceName).Create(context.TODO(), defaultSettings, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create default ProjectSettings: %v", err)
	}

	log.Printf("Created default ProjectSettings for namespace %s", namespaceName)
	return nil
}

func handleProjectSettingsEvent(obj *unstructured.Unstructured) error {
	name := obj.GetName()
	namespace := obj.GetNamespace()

	// Verify the resource still exists before processing
	gvr := types.GetProjectSettingsResource()
	currentObj, err := config.DynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("ProjectSettings %s/%s no longer exists, skipping processing", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to verify ProjectSettings %s/%s exists: %v", namespace, name, err)
	}

	log.Printf("Reconciling ProjectSettings %s/%s", namespace, name)
	return reconcileProjectSettings(currentObj)
}

func reconcileProjectSettings(obj *unstructured.Unstructured) error {
	namespace := obj.GetNamespace()
	name := obj.GetName()

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")

	// Reconcile group access (RoleBindings)
	groupBindingsCreated := 0
	if groupAccess, found, _ := unstructured.NestedSlice(spec, "groupAccess"); found {
		for _, accessInterface := range groupAccess {
			access := accessInterface.(map[string]interface{})
			groupName, _, _ := unstructured.NestedString(access, "groupName")
			role, _, _ := unstructured.NestedString(access, "role")
			if groupName != "" && role != "" {
				if err := ensureRoleBinding(namespace, groupName, role); err != nil {
					log.Printf("Error creating RoleBinding for group %s in namespace %s: %v", groupName, namespace, err)
					continue
				}
				groupBindingsCreated++
			}
		}
	}

	// Update status with reconciliation results (only fields defined in CRD)
	statusUpdate := map[string]interface{}{
		"groupBindingsCreated": groupBindingsCreated,
	}

	return updateProjectSettingsStatus(namespace, name, statusUpdate)
}

func ensureRoleBinding(namespace, groupName, role string) error {
	// Map role to ClusterRole used for ambient project access
	roleName := mapRoleToKubernetesRole(role)
	rbName := fmt.Sprintf("%s-%s", groupName, role)

	// Check if RoleBinding already exists
	_, err := config.K8sClient.RbacV1().RoleBindings(namespace).Get(context.TODO(), rbName, v1.GetOptions{})
	if err == nil {
		log.Printf("RoleBinding %s already exists in namespace %s", rbName, namespace)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("error checking existing RoleBinding: %v", err)
	}

	// Create RoleBinding
	rb := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      rbName,
			Namespace: namespace,
			Labels: map[string]string{
				"ambient-code.io/managed": "true",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "Group",
				Name:     groupName,
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
	}

	_, err = config.K8sClient.RbacV1().RoleBindings(namespace).Create(context.TODO(), rb, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create RoleBinding: %v", err)
	}

	log.Printf("Created RoleBinding %s for group %s in namespace %s", rbName, groupName, namespace)
	return nil
}

func mapRoleToKubernetesRole(role string) string {
	switch strings.ToLower(role) {
	case "admin":
		return "ambient-project-admin"
	case "edit":
		return "ambient-project-edit"
	case "view":
		return "ambient-project-view"
	default:
		return "ambient-project-view" // Default to view role
	}
}

func updateProjectSettingsStatus(namespace, name string, statusUpdate map[string]interface{}) error {
	gvr := types.GetProjectSettingsResource()

	// Get current resource
	obj, err := config.DynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("ProjectSettings %s/%s no longer exists, skipping status update", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to get ProjectSettings %s/%s: %v", namespace, name, err)
	}

	// Update status
	if obj.Object["status"] == nil {
		obj.Object["status"] = make(map[string]interface{})
	}

	status := obj.Object["status"].(map[string]interface{})
	for key, value := range statusUpdate {
		status[key] = value
	}

	// Update the resource
	_, err = config.DynamicClient.Resource(gvr).Namespace(namespace).UpdateStatus(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("ProjectSettings %s/%s was deleted during status update, skipping", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to update ProjectSettings status: %v", err)
	}

	return nil
}
