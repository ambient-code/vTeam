// Package handlers provides HTTP handlers for the backend API.
// This file contains type definitions, package variables, constants, and validation functions for project management.
package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Package-level variables for project handlers (set from main package)
var (
	// GetOpenShiftProjectResource returns the GVR for OpenShift Project resources
	GetOpenShiftProjectResource func() schema.GroupVersionResource
	// K8sClientProjects is the backend service account client used for namespace operations
	// that require elevated permissions (e.g., creating namespaces, assigning roles)
	K8sClientProjects *kubernetes.Clientset
	// DynamicClientProjects is the backend SA dynamic client for OpenShift Project operations
	DynamicClientProjects dynamic.Interface
)

var (
	isOpenShiftCache bool
	isOpenShiftOnce  sync.Once
)

// Default timeout for Kubernetes API operations
const defaultK8sTimeout = 10 * time.Second

// Retry configuration constants
const (
	projectRetryAttempts     = 5
	projectRetryInitialDelay = 200 * time.Millisecond
	projectRetryMaxDelay     = 2 * time.Second
)

// Kubernetes namespace name validation pattern
var namespaceNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// validateProjectName validates a project/namespace name according to Kubernetes naming rules
func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	if len(name) > 63 {
		return fmt.Errorf("project name must be 63 characters or less")
	}
	if !namespaceNamePattern.MatchString(name) {
		return fmt.Errorf("project name must be lowercase alphanumeric with hyphens (cannot start or end with hyphen)")
	}
	// Reserved namespaces
	reservedNames := map[string]bool{
		"default": true, "kube-system": true, "kube-public": true, "kube-node-lease": true,
		"openshift": true, "openshift-infra": true, "openshift-node": true,
	}
	if reservedNames[name] {
		return fmt.Errorf("project name '%s' is reserved and cannot be used", name)
	}
	return nil
}

// sanitizeForK8sName converts a user subject to a valid Kubernetes resource name
func sanitizeForK8sName(subject string) string {
	// Remove system:serviceaccount: prefix if present
	subject = strings.TrimPrefix(subject, "system:serviceaccount:")

	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	sanitized := reg.ReplaceAllString(strings.ToLower(subject), "-")

	// Remove leading/trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// Ensure it doesn't exceed 63 chars (leave room for prefix)
	if len(sanitized) > 40 {
		sanitized = sanitized[:40]
	}

	return sanitized
}

// isOpenShiftCluster detects if we're running on OpenShift by checking for the project.openshift.io API group
// Results are cached using sync.Once for thread-safe, race-free initialization
func isOpenShiftCluster() bool {
	isOpenShiftOnce.Do(func() {
		if K8sClientProjects == nil {
			log.Printf("K8s client not initialized, assuming vanilla Kubernetes")
			isOpenShiftCache = false
			return
		}

		// Try to list API groups and look for project.openshift.io
		groups, err := K8sClientProjects.Discovery().ServerGroups()
		if err != nil {
			log.Printf("Failed to detect OpenShift (assuming vanilla Kubernetes): %v", err)
			isOpenShiftCache = false
			return
		}

		for _, group := range groups.Groups {
			if group.Name == "project.openshift.io" {
				log.Printf("Detected OpenShift cluster")
				isOpenShiftCache = true
				return
			}
		}

		log.Printf("Detected vanilla Kubernetes cluster")
		isOpenShiftCache = false
	})
	return isOpenShiftCache
}
