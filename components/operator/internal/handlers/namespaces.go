package handlers

import (
	"context"
	"log"
	"time"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/services"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

// WatchNamespaces watches for managed namespace events
func WatchNamespaces() {
	for {
		watcher, err := config.K8sClient.CoreV1().Namespaces().Watch(context.TODO(), v1.ListOptions{
			LabelSelector: "ambient-code.io/managed=true",
		})
		if err != nil {
			log.Printf("Failed to create namespace watcher: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Watching for managed namespaces...")

		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				namespace := event.Object.(*corev1.Namespace)
				log.Printf("Detected new managed namespace: %s", namespace.Name)

				// Auto-create ProjectSettings for this namespace
				if err := createDefaultProjectSettings(namespace.Name); err != nil {
					log.Printf("Error creating default ProjectSettings for namespace %s: %v", namespace.Name, err)
				}

				// Ensure shared workspace PVC exists
				if err := services.EnsureProjectWorkspacePVC(namespace.Name); err != nil {
					log.Printf("Failed to ensure workspace PVC in %s: %v", namespace.Name, err)
				}
			case watch.Error:
				obj := event.Object.(*unstructured.Unstructured)
				log.Printf("Watch error for namespaces: %v", obj)
			}
		}

		log.Println("Namespace watch channel closed, restarting...")
		watcher.Stop()
		time.Sleep(2 * time.Second)
	}
}
