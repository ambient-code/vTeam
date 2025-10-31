package handlers

import (
	"context"
	"fmt"

	"ambient-code-backend/types"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// GetProjectSettings retrieves the ProjectSettings CR for a namespace
func GetProjectSettings(ctx context.Context, dynClient dynamic.Interface, namespace string) (*types.ProjectSettings, error) {
	gvr := GetProjectSettingsResource()
	obj, err := dynClient.Resource(gvr).Namespace(namespace).Get(ctx, "projectsettings", v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("ProjectSettings not found for project %s. Please create ProjectSettings first.", namespace)
		}
		return nil, fmt.Errorf("failed to get ProjectSettings: %w", err)
	}

	settings := &types.ProjectSettings{}

	// Extract spec
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return nil, fmt.Errorf("failed to extract ProjectSettings spec: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("ProjectSettings spec not found")
	}

	// Extract groupAccess
	if groupAccessRaw, found := spec["groupAccess"]; found {
		if groupAccessList, ok := groupAccessRaw.([]interface{}); ok {
			for _, item := range groupAccessList {
				if itemMap, ok := item.(map[string]interface{}); ok {
					ga := types.GroupAccess{}
					if groupName, ok := itemMap["groupName"].(string); ok {
						ga.GroupName = groupName
					}
					if role, ok := itemMap["role"].(string); ok {
						ga.Role = role
					}
					settings.GroupAccess = append(settings.GroupAccess, ga)
				}
			}
		}
	}

	// Extract runnerSecretsName
	if secretName, found, _ := unstructured.NestedString(spec, "runnerSecretsName"); found {
		settings.RunnerSecretsName = secretName
	}

	// Extract repos
	if reposRaw, found := spec["repos"]; found {
		if reposList, ok := reposRaw.([]interface{}); ok {
			for _, item := range reposList {
				if itemMap, ok := item.(map[string]interface{}); ok {
					repo := types.ProjectRepo{}
					if name, ok := itemMap["name"].(string); ok {
						repo.Name = name
					}
					if url, ok := itemMap["url"].(string); ok {
						repo.URL = url
					}
					if branch, ok := itemMap["defaultBranch"].(string); ok {
						repo.DefaultBranch = branch
					} else {
						repo.DefaultBranch = "main" // Default from CRD
					}
					settings.Repos = append(settings.Repos, repo)
				}
			}
		}
	}

	return settings, nil
}

// ValidateAndResolveRepoRefs validates that all repo references exist in ProjectSettings
// and returns the resolved repo objects
func ValidateAndResolveRepoRefs(ctx context.Context, dynClient dynamic.Interface, namespace string, repoRefs []string) ([]types.GitRepository, error) {
	if len(repoRefs) == 0 {
		return nil, fmt.Errorf("at least one repo reference is required")
	}

	settings, err := GetProjectSettings(ctx, dynClient, namespace)
	if err != nil {
		return nil, err
	}

	if len(settings.Repos) == 0 {
		return nil, fmt.Errorf("no repos defined in ProjectSettings for project %s. Please configure repos first.", namespace)
	}

	// Create a map for quick lookup
	repoMap := make(map[string]types.ProjectRepo)
	for _, repo := range settings.Repos {
		repoMap[repo.Name] = repo
	}

	// Validate and resolve each reference
	var resolved []types.GitRepository
	for _, ref := range repoRefs {
		projectRepo, exists := repoMap[ref]
		if !exists {
			return nil, fmt.Errorf("repo reference '%s' not found in ProjectSettings. Available repos: %v", ref, getRepoNames(settings.Repos))
		}

		// Convert ProjectRepo to GitRepository
		gitRepo := types.GitRepository{
			URL:    projectRepo.URL,
			Branch: &projectRepo.DefaultBranch,
		}
		resolved = append(resolved, gitRepo)
	}

	return resolved, nil
}

// ValidateAndResolveUmbrellaRepo validates umbrella repo reference and returns the resolved repo
func ValidateAndResolveUmbrellaRepo(ctx context.Context, dynClient dynamic.Interface, namespace string, umbrellaRepoRef string) (types.GitRepository, error) {
	settings, err := GetProjectSettings(ctx, dynClient, namespace)
	if err != nil {
		return types.GitRepository{}, err
	}

	if len(settings.Repos) == 0 {
		return types.GitRepository{}, fmt.Errorf("no repos defined in ProjectSettings for project %s. Please configure repos first.", namespace)
	}

	// Find the umbrella repo
	for _, repo := range settings.Repos {
		if repo.Name == umbrellaRepoRef {
			return types.GitRepository{
				URL:    repo.URL,
				Branch: &repo.DefaultBranch,
			}, nil
		}
	}

	return types.GitRepository{}, fmt.Errorf("umbrella repo reference '%s' not found in ProjectSettings. Available repos: %v", umbrellaRepoRef, getRepoNames(settings.Repos))
}

// getRepoNames extracts repo names for error messages
func getRepoNames(repos []types.ProjectRepo) []string {
	names := make([]string, len(repos))
	for i, repo := range repos {
		names[i] = repo.Name
	}
	return names
}

// buildProjectSettingsSpec builds the spec object for ProjectSettings CR
func buildProjectSettingsSpec(settings *types.ProjectSettings) map[string]interface{} {
	spec := make(map[string]interface{})

	// GroupAccess - only include if non-empty (field is optional in CRD)
	if len(settings.GroupAccess) > 0 {
		groupAccess := make([]map[string]interface{}, len(settings.GroupAccess))
		for i, ga := range settings.GroupAccess {
			groupAccess[i] = map[string]interface{}{
				"groupName": ga.GroupName,
				"role":      ga.Role,
			}
		}
		spec["groupAccess"] = groupAccess
	} else {
		// Explicitly set empty array to satisfy CRD schema
		spec["groupAccess"] = []interface{}{}
	}

	// RunnerSecretsName
	if settings.RunnerSecretsName != "" {
		spec["runnerSecretsName"] = settings.RunnerSecretsName
	}

	// Repos
	if len(settings.Repos) > 0 {
		repos := make([]map[string]interface{}, len(settings.Repos))
		for i, repo := range settings.Repos {
			repoObj := map[string]interface{}{
				"name": repo.Name,
				"url":  repo.URL,
			}
			if repo.DefaultBranch != "" {
				repoObj["defaultBranch"] = repo.DefaultBranch
			} else {
				repoObj["defaultBranch"] = "main" // Default
			}
			repos[i] = repoObj
		}
		spec["repos"] = repos
	}

	return spec
}

// validateUniqueRepoNames ensures all repo names are unique
func validateUniqueRepoNames(repos []types.ProjectRepo) error {
	seen := make(map[string]bool)
	for _, repo := range repos {
		if repo.Name == "" {
			return fmt.Errorf("repo name cannot be empty")
		}
		if seen[repo.Name] {
			return fmt.Errorf("duplicate repo name: %s", repo.Name)
		}
		seen[repo.Name] = true
	}
	return nil
}

// validateUniqueRepoURLs ensures all repo URLs are unique
func validateUniqueRepoURLs(repos []types.ProjectRepo) error {
	seen := make(map[string]bool)
	for _, repo := range repos {
		if repo.URL == "" {
			return fmt.Errorf("repo URL cannot be empty for repo: %s", repo.Name)
		}
		normalizedURL := normalizeRepoURL(repo.URL)
		if seen[normalizedURL] {
			return fmt.Errorf("duplicate repo URL: %s (repo: %s)", repo.URL, repo.Name)
		}
		seen[normalizedURL] = true
	}
	return nil
}
