// Package handlers provides HTTP handlers for the backend API.
// This file contains type aliases, package variables, and validation functions for RFE workflows.
package handlers

import (
	"fmt"
	"strings"

	"ambient-code-backend/types"

	"context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Package-level variables for dependency injection (RFE-specific)
var (
	GetRFEWorkflowResource     func() schema.GroupVersionResource
	UpsertProjectRFEWorkflowCR func(dynamic.Interface, *types.RFEWorkflow) error
	PerformRepoSeeding         func(context.Context, *types.RFEWorkflow, string, string, string, string, string, string, string, string) (bool, error)
	CheckRepoSeeding           func(context.Context, string, *string, string) (bool, map[string]interface{}, error)
	CheckBranchExists          func(context.Context, string, string, string) (bool, error)
	RfeFromUnstructured        func(*unstructured.Unstructured) *types.RFEWorkflow
)

// Type aliases for RFE workflow types
type RFEWorkflow = types.RFEWorkflow
type CreateRFEWorkflowRequest = types.CreateRFEWorkflowRequest
type GitRepository = types.GitRepository
type WorkflowJiraLink = types.WorkflowJiraLink

// rfeLinkSessionRequest holds the request body for linking a session to an RFE workflow
type rfeLinkSessionRequest struct {
	ExistingName string `json:"existingName"`
	Phase        string `json:"phase"`
}

// normalizeRepoURL normalizes a repository URL for comparison
func normalizeRepoURL(repoURL string) string {
	normalized := strings.ToLower(strings.TrimSpace(repoURL))
	// Remove .git suffix
	normalized = strings.TrimSuffix(normalized, ".git")
	// Remove trailing slash
	normalized = strings.TrimSuffix(normalized, "/")
	return normalized
}

// validateUniqueRepositories checks that all repository URLs are unique
func validateUniqueRepositories(umbrellaRepo *GitRepository, supportingRepos []GitRepository) error {
	seen := make(map[string]bool)

	// Check umbrella repo
	if umbrellaRepo != nil && umbrellaRepo.URL != "" {
		normalized := normalizeRepoURL(umbrellaRepo.URL)
		seen[normalized] = true
	}

	// Check supporting repos
	for _, repo := range supportingRepos {
		if repo.URL == "" {
			continue
		}
		normalized := normalizeRepoURL(repo.URL)
		if seen[normalized] {
			return fmt.Errorf("duplicate repository URL detected: %s", repo.URL)
		}
		seen[normalized] = true
	}

	return nil
}
