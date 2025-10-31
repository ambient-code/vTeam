package handlers

import (
	"context"
	"fmt"
	"strings"

	"ambient-code-backend/types"
	"k8s.io/client-go/dynamic"
)

// ValidateReposAgainstProjectSettings validates that all repos in the request exist in ProjectSettings
func ValidateReposAgainstProjectSettings(ctx context.Context, dynClient dynamic.Interface, namespace string, umbrellaRepo *types.GitRepository, supportingRepos []types.GitRepository) error {
	settings, err := GetProjectSettings(ctx, dynClient, namespace)
	if err != nil {
		return err
	}

	if len(settings.Repos) == 0 {
		return fmt.Errorf("no repos defined in ProjectSettings for project %s. Please configure repos in ProjectSettings first.", namespace)
	}

	// Create a map of normalized URLs from ProjectSettings
	allowedRepos := make(map[string]types.ProjectRepo)
	for _, repo := range settings.Repos {
		normalizedURL := normalizeRepoURL(repo.URL)
		allowedRepos[normalizedURL] = repo
	}

	// Validate umbrella repo
	if umbrellaRepo != nil && umbrellaRepo.URL != "" {
		normalizedURL := normalizeRepoURL(umbrellaRepo.URL)
		if _, exists := allowedRepos[normalizedURL]; !exists {
			return fmt.Errorf("umbrella repo URL '%s' is not defined in ProjectSettings. Please add this repo to ProjectSettings first. Available repos: %s",
				umbrellaRepo.URL, formatAvailableRepos(settings.Repos))
		}
	}

	// Validate supporting repos
	for i, repo := range supportingRepos {
		if repo.URL == "" {
			continue
		}
		normalizedURL := normalizeRepoURL(repo.URL)
		if _, exists := allowedRepos[normalizedURL]; !exists {
			return fmt.Errorf("supporting repo #%d URL '%s' is not defined in ProjectSettings. Please add this repo to ProjectSettings first. Available repos: %s",
				i+1, repo.URL, formatAvailableRepos(settings.Repos))
		}
	}

	return nil
}

// ValidateSessionReposAgainstProjectSettings validates that all repos in an AgenticSession exist in ProjectSettings
func ValidateSessionReposAgainstProjectSettings(ctx context.Context, dynClient dynamic.Interface, namespace string, repos []types.SessionRepoMapping) error {
	settings, err := GetProjectSettings(ctx, dynClient, namespace)
	if err != nil {
		return err
	}

	if len(settings.Repos) == 0 {
		return fmt.Errorf("no repos defined in ProjectSettings for project %s. Please configure repos in ProjectSettings first.", namespace)
	}

	// Create a map of normalized URLs from ProjectSettings
	allowedRepos := make(map[string]types.ProjectRepo)
	for _, repo := range settings.Repos {
		normalizedURL := normalizeRepoURL(repo.URL)
		allowedRepos[normalizedURL] = repo
	}

	// Validate each repo's input URL
	for i, repoMapping := range repos {
		if repoMapping.Input.URL == "" {
			continue
		}
		normalizedURL := normalizeRepoURL(repoMapping.Input.URL)
		if _, exists := allowedRepos[normalizedURL]; !exists {
			return fmt.Errorf("repo #%d input URL '%s' is not defined in ProjectSettings. Please add this repo to ProjectSettings first. Available repos: %s",
				i+1, repoMapping.Input.URL, formatAvailableRepos(settings.Repos))
		}

		// Note: We don't validate output URLs as they may be forks (different URLs)
	}

	return nil
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

// formatAvailableRepos formats the list of available repos for error messages
func formatAvailableRepos(repos []types.ProjectRepo) string {
	if len(repos) == 0 {
		return "(none configured)"
	}
	var repoList []string
	for _, repo := range repos {
		repoList = append(repoList, fmt.Sprintf("%s (%s)", repo.Name, repo.URL))
	}
	return strings.Join(repoList, ", ")
}
