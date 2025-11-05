package types

import (
	"fmt"
	"strings"
)

// GetProviderSpecificGuidance returns remediation guidance for provider-specific errors
func GetProviderSpecificGuidance(provider ProviderType, errorType string) string {
	switch provider {
	case ProviderGitHub:
		switch errorType {
		case "auth":
			return "Ensure your GitHub App is installed and has access to the repository, or configure a GitHub PAT in the project runner secret"
		case "permissions":
			return "Ensure the GitHub App or PAT has write access to the repository"
		case "not_found":
			return "Verify the repository URL is correct and you have access to it on GitHub"
		default:
			return "Check your GitHub repository configuration and try again"
		}
	case ProviderGitLab:
		switch errorType {
		case "auth":
			return "Connect your GitLab account with a valid Personal Access Token via /auth/gitlab/connect"
		case "permissions":
			return "Ensure your GitLab PAT has 'api', 'read_repository', and 'write_repository' scopes"
		case "not_found":
			return "Verify the repository URL is correct and you have access to it on GitLab"
		default:
			return "Check your GitLab repository configuration and try again"
		}
	default:
		return "Check your repository configuration and try again"
	}
}

// FormatMixedProviderError formats an error message for mixed-provider scenarios
func FormatMixedProviderError(results []ProviderResult) string {
	failedProviders := []string{}
	successfulProviders := []string{}

	for _, result := range results {
		if result.Success {
			successfulProviders = append(successfulProviders, string(result.Provider))
		} else {
			failedProviders = append(failedProviders, string(result.Provider))
		}
	}

	if len(failedProviders) == 0 {
		return "All repository operations completed successfully"
	} else if len(failedProviders) == len(results) {
		return "All repository operations failed. Check your provider configurations and credentials"
	} else {
		return fmt.Sprintf("Some repository operations failed (%s). Successful: %s",
			strings.Join(failedProviders, ", "),
			strings.Join(successfulProviders, ", "))
	}
}

// CreateProviderResult creates a ProviderResult from an operation outcome
func CreateProviderResult(provider ProviderType, repoURL string, err error) ProviderResult {
	result := ProviderResult{
		Provider: provider,
		RepoURL:  repoURL,
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
	}

	return result
}

// AggregateProviderResults creates a MixedProviderSessionResult from multiple provider results
func AggregateProviderResults(results []ProviderResult) MixedProviderSessionResult {
	allSuccess := true
	for _, result := range results {
		if !result.Success {
			allSuccess = false
			break
		}
	}

	return MixedProviderSessionResult{
		OverallSuccess: allSuccess,
		Results:        results,
		Message:        FormatMixedProviderError(results),
	}
}
