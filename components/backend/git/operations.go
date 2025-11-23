// Package git provides Git repository operations including cloning, forking, and PR creation.
package git

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"ambient-code-backend/gitlab"
	"ambient-code-backend/types"
)

// Package-level dependencies (set from main package)
var (
	GetProjectSettingsResource func() schema.GroupVersionResource
	GetGitHubInstallation      func(context.Context, string) (interface{}, error)
	GitHubTokenManager         interface{} // *GitHubTokenManager from main package
	GetBackendNamespace        func() string
)

// ProjectSettings represents the project configuration
type ProjectSettings struct {
	RunnerSecret string
}

// DiffSummary holds summary counts from git diff --numstat
type DiffSummary struct {
	TotalAdded   int `json:"total_added"`
	TotalRemoved int `json:"total_removed"`
	FilesAdded   int `json:"files_added"`
	FilesRemoved int `json:"files_removed"`
}

// GetGitHubToken tries to get a GitHub token from GitHub App first, then falls back to project runner secret
func GetGitHubToken(ctx context.Context, k8sClient *kubernetes.Clientset, dynClient dynamic.Interface, project, userID string) (string, error) {
	// Try GitHub App first if available
	if GetGitHubInstallation != nil && GitHubTokenManager != nil {
		installation, err := GetGitHubInstallation(ctx, userID)
		if err == nil && installation != nil {
			// Use reflection-like approach to call MintInstallationTokenForHost
			// This requires the caller to set up the proper interface/struct
			type githubInstallation interface {
				GetInstallationID() int64
				GetHost() string
			}
			type tokenManager interface {
				MintInstallationTokenForHost(context.Context, int64, string) (string, time.Time, error)
			}

			if inst, ok := installation.(githubInstallation); ok {
				if mgr, ok := GitHubTokenManager.(tokenManager); ok {
					token, _, err := mgr.MintInstallationTokenForHost(ctx, inst.GetInstallationID(), inst.GetHost())
					if err == nil && token != "" {
						log.Printf("Using GitHub App token for user %s", userID)
						return token, nil
					}
					log.Printf("Failed to mint GitHub App token for user %s: %v", userID, err)
				}
			}
		}
	}

	// Fall back to project integration secret GITHUB_TOKEN (hardcoded secret name)
	if k8sClient == nil {
		log.Printf("Cannot read integration secret: k8s client is nil")
		return "", fmt.Errorf("no GitHub credentials available. Either connect GitHub App or configure GITHUB_TOKEN in integration secrets")
	}

	const secretName = "ambient-non-vertex-integrations"

	log.Printf("Attempting to read GITHUB_TOKEN from secret %s/%s", project, secretName)

	secret, err := k8sClient.CoreV1().Secrets(project).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		log.Printf("Failed to get integration secret %s/%s: %v", project, secretName, err)
		return "", fmt.Errorf("no GitHub credentials available. Either connect GitHub App or configure GITHUB_TOKEN in integration secrets")
	}

	if secret.Data == nil {
		log.Printf("Secret %s/%s exists but Data is nil", project, secretName)
		return "", fmt.Errorf("no GitHub credentials available. Either connect GitHub App or configure GITHUB_TOKEN in integration secrets")
	}

	token, ok := secret.Data["GITHUB_TOKEN"]
	if !ok {
		log.Printf("Secret %s/%s exists but has no GITHUB_TOKEN key (available keys: %v)", project, secretName, getSecretKeys(secret.Data))
		return "", fmt.Errorf("no GitHub credentials available. Either connect GitHub App or configure GITHUB_TOKEN in integration secrets")
	}

	if len(token) == 0 {
		log.Printf("Secret %s/%s has GITHUB_TOKEN key but value is empty", project, secretName)
		return "", fmt.Errorf("no GitHub credentials available. Either connect GitHub App or configure GITHUB_TOKEN in integration secrets")
	}

	log.Printf("Using GITHUB_TOKEN from integration secret %s/%s", project, secretName)
	return string(token), nil
}

// GetGitLabToken retrieves a GitLab Personal Access Token for a user
func GetGitLabToken(ctx context.Context, k8sClient *kubernetes.Clientset, project, userID string) (string, error) {
	if k8sClient == nil {
		log.Printf("Cannot read GitLab token: k8s client is nil")
		return "", fmt.Errorf("no GitLab credentials available. Please connect your GitLab account")
	}

	// GitLab tokens are stored in the project namespace (multi-tenant isolation)
	// This matches the GitHub PAT pattern using ambient-non-vertex-integrations
	secret, err := k8sClient.CoreV1().Secrets(project).Get(ctx, "gitlab-user-tokens", v1.GetOptions{})
	if err != nil {
		log.Printf("Failed to get gitlab-user-tokens secret in %s: %v", project, err)
		return "", fmt.Errorf("no GitLab credentials available. Please connect your GitLab account in this project")
	}

	if secret.Data == nil {
		log.Printf("Secret gitlab-user-tokens exists but Data is nil")
		return "", fmt.Errorf("no GitLab credentials available. Please connect your GitLab account")
	}

	token, ok := secret.Data[userID]
	if !ok {
		log.Printf("Secret gitlab-user-tokens has no token for user %s", userID)
		return "", fmt.Errorf("no GitLab credentials available. Please connect your GitLab account")
	}

	if len(token) == 0 {
		log.Printf("Secret gitlab-user-tokens has token for user %s but value is empty", userID)
		return "", fmt.Errorf("no GitLab credentials available. Please connect your GitLab account")
	}

	log.Printf("Using GitLab token for user %s from gitlab-user-tokens secret", userID)
	return string(token), nil
}

// GetGitToken retrieves a Git token based on the repository provider
func GetGitToken(ctx context.Context, k8sClient *kubernetes.Clientset, dynClient dynamic.Interface, repoURL, project, userID string) (string, error) {
	provider := types.DetectProvider(repoURL)

	switch provider {
	case types.ProviderGitHub:
		return GetGitHubToken(ctx, k8sClient, dynClient, project, userID)
	case types.ProviderGitLab:
		return GetGitLabToken(ctx, k8sClient, project, userID)
	default:
		return "", fmt.Errorf("unsupported repository provider for URL: %s", repoURL)
	}
}

// getSecretKeys returns a list of keys from a secret's Data map for debugging
func getSecretKeys(data map[string][]byte) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}

// CheckRepoSeeding checks if a repo has been seeded by verifying .claude/commands/ and .specify/ exist
// Supports both GitHub and GitLab repositories
func CheckRepoSeeding(ctx context.Context, repoURL string, branch *string, token string) (bool, map[string]interface{}, error) {
	branchName := "main"
	if branch != nil && strings.TrimSpace(*branch) != "" {
		branchName = strings.TrimSpace(*branch)
	}

	provider := types.DetectProvider(repoURL)

	var claudeExists, claudeCommandsExists, claudeAgentsExists, specifyExists bool
	var err error

	switch provider {
	case types.ProviderGitHub:
		var owner, repo string
		owner, repo, err = ParseGitHubURL(repoURL)
		if err != nil {
			return false, nil, err
		}

		claudeExists, err = checkGitHubPathExists(ctx, owner, repo, branchName, ".claude", token)
		if err != nil {
			return false, nil, fmt.Errorf("failed to check .claude: %w", err)
		}

		claudeCommandsExists, err = checkGitHubPathExists(ctx, owner, repo, branchName, ".claude/commands", token)
		if err != nil {
			return false, nil, fmt.Errorf("failed to check .claude/commands: %w", err)
		}

		claudeAgentsExists, err = checkGitHubPathExists(ctx, owner, repo, branchName, ".claude/agents", token)
		if err != nil {
			return false, nil, fmt.Errorf("failed to check .claude/agents: %w", err)
		}

		specifyExists, err = checkGitHubPathExists(ctx, owner, repo, branchName, ".specify", token)
		if err != nil {
			return false, nil, fmt.Errorf("failed to check .specify: %w", err)
		}

	case types.ProviderGitLab:
		var parsed *types.ParsedGitLabRepo
		parsed, err = gitlab.ParseGitLabURL(repoURL)
		if err != nil {
			return false, nil, fmt.Errorf("invalid GitLab URL: %w", err)
		}

		client := gitlab.NewClient(parsed.APIURL, token)

		claudeExists, err = checkGitLabPathExists(ctx, client, parsed.ProjectID, branchName, ".claude")
		if err != nil {
			return false, nil, fmt.Errorf("failed to check .claude: %w", err)
		}

		claudeCommandsExists, err = checkGitLabPathExists(ctx, client, parsed.ProjectID, branchName, ".claude/commands")
		if err != nil {
			return false, nil, fmt.Errorf("failed to check .claude/commands: %w", err)
		}

		claudeAgentsExists, err = checkGitLabPathExists(ctx, client, parsed.ProjectID, branchName, ".claude/agents")
		if err != nil {
			return false, nil, fmt.Errorf("failed to check .claude/agents: %w", err)
		}

		specifyExists, err = checkGitLabPathExists(ctx, client, parsed.ProjectID, branchName, ".specify")
		if err != nil {
			return false, nil, fmt.Errorf("failed to check .specify: %w", err)
		}

	default:
		return false, nil, fmt.Errorf("unsupported repository provider for URL: %s", repoURL)
	}

	details := map[string]interface{}{
		"claudeExists":         claudeExists,
		"claudeCommandsExists": claudeCommandsExists,
		"claudeAgentsExists":   claudeAgentsExists,
		"specifyExists":        specifyExists,
	}

	// Repo is properly seeded if all critical components exist
	isSeeded := claudeCommandsExists && claudeAgentsExists && specifyExists
	return isSeeded, details, nil
}

// checkGitLabPathExists checks if a path exists in a GitLab repository
func checkGitLabPathExists(ctx context.Context, client *gitlab.Client, projectID, branch, path string) (bool, error) {
	// Try to get the tree for this path
	entries, err := client.GetAllTreeEntries(ctx, projectID, branch, path)
	if err != nil {
		// Check if it's a 404 error (path doesn't exist)
		if gitlabErr, ok := err.(*types.GitLabAPIError); ok {
			if gitlabErr.StatusCode == 404 {
				return false, nil
			}
		}
		return false, err
	}

	// Path exists if we got entries
	return len(entries) > 0 || entries != nil, nil
}

// ParseGitHubURL extracts owner and repo from a GitHub URL
func ParseGitHubURL(gitURL string) (owner, repo string, err error) {
	gitURL = strings.TrimSuffix(gitURL, ".git")

	if strings.Contains(gitURL, "github.com") {
		parts := strings.Split(gitURL, "github.com")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid GitHub URL")
		}
		path := strings.Trim(parts[1], "/:")
		pathParts := strings.Split(path, "/")
		if len(pathParts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub URL path")
		}
		return pathParts[0], pathParts[1], nil
	}

	return "", "", fmt.Errorf("not a GitHub URL")
}

// IsProtectedBranch checks if a branch name is a protected branch
// Protected branches: main, master, develop
func IsProtectedBranch(branchName string) bool {
	protected := []string{"main", "master", "develop"}
	normalized := strings.ToLower(strings.TrimSpace(branchName))
	for _, p := range protected {
		if normalized == p {
			return true
		}
	}
	return false
}

// ValidateBranchName validates a user-provided branch name
// Returns an error if the branch name is protected or invalid
func ValidateBranchName(branchName string) error {
	normalized := strings.TrimSpace(branchName)
	if normalized == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	if IsProtectedBranch(normalized) {
		return fmt.Errorf("'%s' is a protected branch name. Please use a different branch name", normalized)
	}
	return nil
}

// checkGitHubPathExists checks if a path exists in a GitHub repo
func checkGitHubPathExists(ctx context.Context, owner, repo, branch, path, token string) (bool, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		owner, repo, path, branch)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("GitHub API error: %s (body: %s)", resp.Status, string(body))
}

// GitRepo interface for repository information
type GitRepo interface {
	GetURL() string
	GetBranch() *string
}

// Workflow interface for RFE workflows
type Workflow interface {
	GetUmbrellaRepo() GitRepo
	GetSupportingRepos() []GitRepo
}

// PerformRepoSeeding performs the actual seeding operations
// wf parameter should implement the Workflow interface
// Returns: branchExisted (bool), error
func PerformRepoSeeding(ctx context.Context, wf Workflow, branchName, token, agentURL, agentBranch, agentPath, specKitRepo, specKitVersion, specKitTemplate string) (bool, error) {
	umbrellaRepo := wf.GetUmbrellaRepo()
	if umbrellaRepo == nil {
		return false, fmt.Errorf("workflow has no spec repo")
	}

	if branchName == "" {
		return false, fmt.Errorf("branchName is required")
	}

	// Validate push access to spec repo before starting
	log.Printf("Validating push access to spec repo: %s", umbrellaRepo.GetURL())
	if err := validatePushAccess(ctx, umbrellaRepo.GetURL(), token); err != nil {
		return false, fmt.Errorf("spec repo access validation failed: %w", err)
	}

	// Validate push access to all supporting repos before starting
	supportingRepos := wf.GetSupportingRepos()
	if len(supportingRepos) > 0 {
		log.Printf("Validating push access to %d supporting repos", len(supportingRepos))
		for i, repo := range supportingRepos {
			if err := validatePushAccess(ctx, repo.GetURL(), token); err != nil {
				return false, fmt.Errorf("supporting repo #%d (%s) access validation failed: %w", i+1, repo.GetURL(), err)
			}
		}
	}

	umbrellaDir, err := os.MkdirTemp("", "umbrella-*")
	if err != nil {
		return false, fmt.Errorf("failed to create temp dir for spec repo: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(umbrellaDir); err != nil {
			log.Printf("Warning: failed to cleanup temp directory %s: %v", umbrellaDir, err)
		}
	}()

	agentSrcDir, err := os.MkdirTemp("", "agents-*")
	if err != nil {
		return false, fmt.Errorf("failed to create temp dir for agent source: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(agentSrcDir); err != nil {
			log.Printf("Warning: failed to cleanup temp directory %s: %v", agentSrcDir, err)
		}
	}()

	// Clone umbrella repo with authentication
	log.Printf("Cloning umbrella repo: %s", umbrellaRepo.GetURL())
	authenticatedURL, err := InjectGitToken(umbrellaRepo.GetURL(), token)
	if err != nil {
		return false, fmt.Errorf("failed to prepare spec repo URL: %w", err)
	}

	// Clone base branch (the branch from which feature branch will be created)
	baseBranch := "main"
	if branch := umbrellaRepo.GetBranch(); branch != nil && strings.TrimSpace(*branch) != "" {
		baseBranch = strings.TrimSpace(*branch)
	}

	log.Printf("Verifying base branch '%s' exists before cloning", baseBranch)

	// Verify base branch exists before trying to clone
	verifyCmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", authenticatedURL, baseBranch)
	verifyOut, verifyErr := verifyCmd.CombinedOutput()
	if verifyErr != nil || strings.TrimSpace(string(verifyOut)) == "" {
		return false, fmt.Errorf("base branch '%s' does not exist in repository. Please ensure the base branch exists before seeding", baseBranch)
	}

	umbrellaArgs := []string{"clone", "--depth", "1", "--branch", baseBranch, authenticatedURL, umbrellaDir}

	cmd := exec.CommandContext(ctx, "git", umbrellaArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("failed to clone base branch '%s': %w (output: %s)", baseBranch, err, string(out))
	}

	// Configure git user
	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "config", "user.email", "vteam-bot@ambient-code.io")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Warning: failed to set git user.email: %v (output: %s)", err, string(out))
	}
	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "config", "user.name", "vTeam Bot")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Warning: failed to set git user.name: %v (output: %s)", err, string(out))
	}

	// Check if feature branch already exists remotely
	// Use authenticated URL directly to avoid issues with shallow clone remote setup
	cmd = exec.CommandContext(ctx, "git", "ls-remote", "--heads", authenticatedURL, fmt.Sprintf("refs/heads/%s", branchName))
	lsRemoteOut, lsRemoteErr := cmd.CombinedOutput()
	log.Printf("DEBUG: ls-remote for branch '%s': error=%v, output='%s'", branchName, lsRemoteErr, string(lsRemoteOut))

	// Check if branch exists by looking for actual git ref (ignoring warnings)
	// Valid output format: "<sha>\trefs/heads/<branch>"
	branchExistsRemotely := false
	if lsRemoteErr == nil {
		lines := strings.Split(string(lsRemoteOut), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Skip empty lines and warning messages
			if trimmed == "" || strings.HasPrefix(trimmed, "warning:") {
				continue
			}
			// Check if line contains the branch ref
			if strings.Contains(trimmed, fmt.Sprintf("refs/heads/%s", branchName)) {
				branchExistsRemotely = true
				break
			}
		}
	}
	log.Printf("DEBUG: branchExistsRemotely=%v", branchExistsRemotely)

	if branchExistsRemotely {
		// Branch exists - check it out instead of creating new
		log.Printf("⚠️  Branch '%s' already exists remotely - checking out existing branch", branchName)
		log.Printf("⚠️  This RFE will modify the existing branch '%s'", branchName)

		// Check if the branch is already checked out (happens when base branch == feature branch)
		if baseBranch == branchName {
			log.Printf("Feature branch '%s' is the same as base branch - already checked out", branchName)
		} else {
			// Fetch the specific branch with depth (works with shallow clones)
			// Format: git fetch --depth 1 origin <remote-branch>:<local-branch>
			cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "fetch", "--depth", "1", "origin", fmt.Sprintf("%s:%s", branchName, branchName))
			if out, err := cmd.CombinedOutput(); err != nil {
				return false, fmt.Errorf("failed to fetch existing branch %s: %w (output: %s)", branchName, err, string(out))
			}

			// Checkout the fetched branch
			cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "checkout", branchName)
			if out, err := cmd.CombinedOutput(); err != nil {
				return false, fmt.Errorf("failed to checkout existing branch %s: %w (output: %s)", branchName, err, string(out))
			}
		}
	} else {
		// Branch doesn't exist remotely
		// Check if we're already on the feature branch (happens when base branch == feature branch)
		if baseBranch == branchName {
			log.Printf("Feature branch '%s' is the same as base branch - already on this branch", branchName)
		} else {
			// Create new feature branch from the current base branch
			log.Printf("Creating new feature branch: %s", branchName)
			cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "checkout", "-b", branchName)
			if out, err := cmd.CombinedOutput(); err != nil {
				return false, fmt.Errorf("failed to create branch %s: %w (output: %s)", branchName, err, string(out))
			}
		}
	}

	// Download and extract spec-kit template
	log.Printf("Downloading spec-kit from repo: %s, version: %s", specKitRepo, specKitVersion)

	// Support both releases (vX.X.X) and branch archives (main, branch-name)
	var specKitURL string
	if strings.HasPrefix(specKitVersion, "v") {
		// It's a tagged release - use releases API
		specKitURL = fmt.Sprintf("https://github.com/%s/releases/download/%s/%s-%s.zip",
			specKitRepo, specKitVersion, specKitTemplate, specKitVersion)
		log.Printf("Downloading spec-kit release: %s", specKitURL)
	} else {
		// It's a branch name - use archive API
		specKitURL = fmt.Sprintf("https://github.com/%s/archive/refs/heads/%s.zip",
			specKitRepo, specKitVersion)
		log.Printf("Downloading spec-kit branch archive: %s", specKitURL)
	}

	resp, err := http.Get(specKitURL)
	if err != nil {
		return false, fmt.Errorf("failed to download spec-kit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("spec-kit download failed with status: %s", resp.Status)
	}

	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read spec-kit zip: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return false, fmt.Errorf("failed to open spec-kit zip: %w", err)
	}

	// Extract spec-kit files
	specKitFilesAdded := 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}

		rel := strings.TrimPrefix(f.Name, "./")
		rel = strings.ReplaceAll(rel, "\\", "/")

		// Strip archive prefix from branch downloads (e.g., "spec-kit-rh-vteam-flexible-branches/")
		// Branch archives have format: "repo-branch-name/file", releases have just "file"
		if strings.Contains(rel, "/") && !strings.HasPrefix(specKitVersion, "v") {
			parts := strings.SplitN(rel, "/", 2)
			if len(parts) == 2 {
				rel = parts[1] // Take everything after first "/"
			}
		}

		// Only extract files needed for umbrella repos (matching official spec-kit release template):
		// - templates/commands/ → .claude/commands/
		// - scripts/bash/ → .specify/scripts/bash/
		// - templates/*.md → .specify/templates/
		// - memory/ → .specify/memory/
		// Skip everything else (docs/, media/, root files, .github/, scripts/powershell/, etc.)

		var targetRel string
		if strings.HasPrefix(rel, "templates/commands/") {
			// Map templates/commands/*.md to .claude/commands/speckit.*.md
			cmdFile := strings.TrimPrefix(rel, "templates/commands/")
			if !strings.HasPrefix(cmdFile, "speckit.") {
				cmdFile = "speckit." + cmdFile
			}
			targetRel = ".claude/commands/" + cmdFile
		} else if strings.HasPrefix(rel, "scripts/bash/") {
			// Map scripts/bash/ to .specify/scripts/bash/
			targetRel = strings.Replace(rel, "scripts/bash/", ".specify/scripts/bash/", 1)
		} else if strings.HasPrefix(rel, "templates/") && strings.HasSuffix(rel, ".md") {
			// Map templates/*.md to .specify/templates/
			targetRel = strings.Replace(rel, "templates/", ".specify/templates/", 1)
		} else if strings.HasPrefix(rel, "memory/") {
			// Map memory/ to .specify/memory/
			targetRel = ".specify/" + rel
		} else {
			// Skip all other files (docs/, media/, root files, .github/, scripts/powershell/, etc.)
			continue
		}

		// Security: prevent path traversal
		for strings.Contains(targetRel, "../") {
			targetRel = strings.ReplaceAll(targetRel, "../", "")
		}

		targetPath := filepath.Join(umbrellaDir, targetRel)

		if _, err := os.Stat(targetPath); err == nil {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			log.Printf("Failed to create dir for %s: %v", rel, err)
			continue
		}

		rc, err := f.Open()
		if err != nil {
			log.Printf("Failed to open zip entry %s: %v", f.Name, err)
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			log.Printf("Failed to read zip entry %s: %v", f.Name, err)
			continue
		}

		// Preserve executable permissions for scripts
		fileMode := fs.FileMode(0644)
		if strings.HasPrefix(targetRel, ".specify/scripts/") {
			// Scripts need to be executable
			fileMode = 0755
		} else if f.Mode().Perm()&0111 != 0 {
			// Preserve executable bit from zip if it was set
			fileMode = 0755
		}

		if err := os.WriteFile(targetPath, content, fileMode); err != nil {
			log.Printf("Failed to write %s: %v", targetPath, err)
			continue
		}
		specKitFilesAdded++
	}
	log.Printf("Extracted %d spec-kit files", specKitFilesAdded)

	// Clone agent source repo
	log.Printf("Cloning agent source: %s", agentURL)
	agentArgs := []string{"clone", "--depth", "1"}
	if agentBranch != "" {
		agentArgs = append(agentArgs, "--branch", agentBranch)
	}
	agentArgs = append(agentArgs, agentURL, agentSrcDir)

	cmd = exec.CommandContext(ctx, "git", agentArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("failed to clone agent source: %w (output: %s)", err, string(out))
	}

	// Copy agent markdown files to .claude/agents/
	agentSourcePath := filepath.Join(agentSrcDir, agentPath)
	claudeDir := filepath.Join(umbrellaDir, ".claude")
	claudeAgentsDir := filepath.Join(claudeDir, "agents")
	if err := os.MkdirAll(claudeAgentsDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create .claude/agents directory: %w", err)
	}

	agentsCopied := 0
	err = filepath.WalkDir(agentSourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Failed to read agent file %s: %v", path, err)
			return nil
		}

		targetPath := filepath.Join(claudeAgentsDir, d.Name())
		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			log.Printf("Failed to write agent file %s: %v", targetPath, err)
			return nil
		}
		agentsCopied++
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("failed to copy agents: %w", err)
	}
	log.Printf("Copied %d agent files", agentsCopied)

	// Create specs directory for feature work
	specsDir := filepath.Join(umbrellaDir, "specs", branchName)
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create specs/%s directory: %w", branchName, err)
	}
	log.Printf("Created specs/%s directory", branchName)

	// Commit and push changes to feature branch
	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("git add failed: %w (output: %s)", err, string(out))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "diff", "--cached", "--quiet")
	if err := cmd.Run(); err == nil {
		log.Printf("No changes to commit for seeding, but will still push branch")
	} else {
		// Commit with branch-specific message
		commitMsg := fmt.Sprintf("chore: initialize %s with spec-kit and agents", branchName)
		cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "commit", "-m", commitMsg)
		if out, err := cmd.CombinedOutput(); err != nil {
			return false, fmt.Errorf("git commit failed: %w (output: %s)", err, string(out))
		}
	}

	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "remote", "set-url", "origin", authenticatedURL)
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("failed to set remote URL: %w (output: %s)", err, string(out))
	}

	// Push feature branch to origin
	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "push", "-u", "origin", branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("git push failed: %w (output: %s)", err, string(out))
	}

	log.Printf("Successfully seeded umbrella repo on branch %s", branchName)

	// Create feature branch in all supporting repos
	// Push access will be validated by the actual git operations - if they fail, we'll get a clear error
	if len(supportingRepos) > 0 {
		log.Printf("Creating feature branch %s in %d supporting repos", branchName, len(supportingRepos))
		for i, repo := range supportingRepos {
			if err := createBranchInRepo(ctx, repo, branchName, token); err != nil {
				return false, fmt.Errorf("failed to create branch in supporting repo #%d (%s): %w", i+1, repo.GetURL(), err)
			}
		}
	}

	return branchExistsRemotely, nil
}

// sanitizeURLForError removes credentials from a URL for safe error logging
func sanitizeURLForError(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		// If URL can't be parsed, just return a generic message
		return "[invalid URL format]"
	}
	// Remove any embedded credentials
	u.User = nil
	return u.String()
}

// InjectGitHubToken injects a GitHub token into a git URL for authentication
func InjectGitHubToken(gitURL, token string) (string, error) {
	u, err := url.Parse(gitURL)
	if err != nil {
		// Sanitize URL before including in error message
		return "", fmt.Errorf("invalid git URL (%s): %w", sanitizeURLForError(gitURL), err)
	}

	if u.Scheme != "https" {
		return gitURL, nil
	}

	u.User = url.UserPassword("x-access-token", token)
	return u.String(), nil
}

// InjectGitLabToken injects a GitLab token into a git URL for authentication
func InjectGitLabToken(gitURL, token string) (string, error) {
	u, err := url.Parse(gitURL)
	if err != nil {
		// Sanitize URL before including in error message
		return "", fmt.Errorf("invalid git URL (%s): %w", sanitizeURLForError(gitURL), err)
	}

	if u.Scheme != "https" {
		return gitURL, nil
	}

	// GitLab uses oauth2:token@ format
	u.User = url.UserPassword("oauth2", token)
	return u.String(), nil
}

// InjectGitToken injects a Git token into a URL based on the repository provider
func InjectGitToken(gitURL, token string) (string, error) {
	provider := types.DetectProvider(gitURL)

	switch provider {
	case types.ProviderGitHub:
		return InjectGitHubToken(gitURL, token)
	case types.ProviderGitLab:
		return InjectGitLabToken(gitURL, token)
	default:
		return "", fmt.Errorf("unsupported repository provider for URL: %s", gitURL)
	}
}

// DetectPushError analyzes git push error output and provides user-friendly error messages
func DetectPushError(repoURL, stderr, stdout string) error {
	provider := types.DetectProvider(repoURL)

	// Common error patterns
	stderrLower := strings.ToLower(stderr)
	stdoutLower := strings.ToLower(stdout)
	combined := stderrLower + " " + stdoutLower

	// Check for authentication/permission errors
	if strings.Contains(combined, "403") || strings.Contains(combined, "forbidden") {
		if provider == types.ProviderGitLab {
			return fmt.Errorf("GitLab push failed: Insufficient permissions. Ensure your GitLab token has 'write_repository' scope. You can update your token by reconnecting your GitLab account with the required permissions")
		} else if provider == types.ProviderGitHub {
			return fmt.Errorf("GitHub push failed: Insufficient permissions. Check that your GitHub App installation has write access to this repository")
		}
		return fmt.Errorf("Push failed: Insufficient permissions (403 Forbidden)")
	}

	// Check for authentication failures
	if strings.Contains(combined, "401") || strings.Contains(combined, "unauthorized") || strings.Contains(combined, "authentication failed") {
		if provider == types.ProviderGitLab {
			return fmt.Errorf("GitLab push failed: Authentication failed. Your GitLab token may be invalid or expired. Please reconnect your GitLab account")
		} else if provider == types.ProviderGitHub {
			return fmt.Errorf("GitHub push failed: Authentication failed. Check your GitHub App installation")
		}
		return fmt.Errorf("Push failed: Authentication failed (401 Unauthorized)")
	}

	// Check for network errors
	if strings.Contains(combined, "could not resolve host") || strings.Contains(combined, "connection refused") {
		return fmt.Errorf("Push failed: Unable to connect to %s. Check network connectivity", extractHostFromURL(repoURL))
	}

	// Check for rate limiting
	if strings.Contains(combined, "429") || strings.Contains(combined, "rate limit") {
		if provider == types.ProviderGitLab {
			return fmt.Errorf("GitLab push failed: Rate limit exceeded. Please wait a few minutes before retrying")
		}
		return fmt.Errorf("Push failed: API rate limit exceeded. Please wait before retrying")
	}

	// Check for repository not found
	if strings.Contains(combined, "404") || strings.Contains(combined, "not found") || strings.Contains(combined, "repository not found") {
		return fmt.Errorf("Push failed: Repository not found. Verify the repository URL: %s", repoURL)
	}

	// Return original error if no pattern matched
	errMsg := strings.TrimSpace(stderr)
	if errMsg == "" {
		errMsg = strings.TrimSpace(stdout)
	}
	if len(errMsg) > 500 {
		errMsg = errMsg[:500] + "..."
	}
	return fmt.Errorf("Push failed: %s", errMsg)
}

// extractHostFromURL extracts the host from a git URL for error messages
func extractHostFromURL(gitURL string) string {
	if strings.HasPrefix(gitURL, "git@") {
		// SSH format: git@host:owner/repo
		parts := strings.Split(gitURL, "@")
		if len(parts) > 1 {
			hostParts := strings.Split(parts[1], ":")
			if len(hostParts) > 0 {
				return hostParts[0]
			}
		}
	} else {
		// HTTPS format
		u, err := url.Parse(gitURL)
		if err == nil && u.Host != "" {
			return u.Host
		}
	}
	return "repository host"
}

// ConstructBranchURL constructs a web URL to view a branch based on the provider
func ConstructBranchURL(repoURL, branch string) (string, error) {
	provider := types.DetectProvider(repoURL)

	switch provider {
	case types.ProviderGitHub:
		return ConstructGitHubBranchURL(repoURL, branch)
	case types.ProviderGitLab:
		return ConstructGitLabBranchURL(repoURL, branch)
	default:
		return "", fmt.Errorf("unsupported provider for URL: %s", repoURL)
	}
}

// ConstructGitHubBranchURL constructs a GitHub web URL for a branch
func ConstructGitHubBranchURL(repoURL, branch string) (string, error) {
	owner, repo, err := ParseGitHubURL(repoURL)
	if err != nil {
		return "", err
	}

	// Clean repo name (remove .git if present)
	repo = strings.TrimSuffix(repo, ".git")

	return fmt.Sprintf("https://github.com/%s/%s/tree/%s", owner, repo, branch), nil
}

// ConstructGitLabBranchURL constructs a GitLab web URL for a branch
func ConstructGitLabBranchURL(repoURL, branch string) (string, error) {
	parsed, err := gitlab.ParseGitLabURL(repoURL)
	if err != nil {
		return "", err
	}

	// GitLab branch URL format: https://gitlab.com/owner/repo/-/tree/branch
	return fmt.Sprintf("https://%s/%s/%s/-/tree/%s", parsed.Host, parsed.Owner, parsed.Repo, branch), nil
}

// GetRepositoryWebURL returns the main web URL for a repository
func GetRepositoryWebURL(repoURL string) (string, error) {
	provider := types.DetectProvider(repoURL)

	switch provider {
	case types.ProviderGitHub:
		owner, repo, err := ParseGitHubURL(repoURL)
		if err != nil {
			return "", err
		}
		repo = strings.TrimSuffix(repo, ".git")
		return fmt.Sprintf("https://github.com/%s/%s", owner, repo), nil

	case types.ProviderGitLab:
		parsed, err := gitlab.ParseGitLabURL(repoURL)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://%s/%s/%s", parsed.Host, parsed.Owner, parsed.Repo), nil

	default:
		return "", fmt.Errorf("unsupported provider for URL: %s", repoURL)
	}
}

// DeriveRepoFolderFromURL extracts the repo folder from a Git URL
func DeriveRepoFolderFromURL(u string) string {
	s := strings.TrimSpace(u)
	if s == "" {
		return ""
	}

	if strings.HasPrefix(s, "git@") && strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		host := strings.TrimPrefix(parts[0], "git@")
		s = "https://" + host + "/" + parts[1]
	}

	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}

	if i := strings.Index(s, "/"); i >= 0 {
		s = s[i+1:]
	}

	segs := strings.Split(s, "/")
	if len(segs) == 0 {
		return ""
	}

	last := segs[len(segs)-1]
	last = strings.TrimSuffix(last, ".git")
	return strings.TrimSpace(last)
}

// PushRepo performs git add/commit/push operations on a repository directory
func PushRepo(ctx context.Context, repoDir, commitMessage, outputRepoURL, branch, githubToken string) (string, error) {
	if fi, err := os.Stat(repoDir); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("repo directory not found: %s", repoDir)
	}

	run := func(args ...string) (string, string, error) {
		start := time.Now()
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = repoDir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		dur := time.Since(start)
		log.Printf("gitPushRepo: exec dur=%s cmd=%q stderr.len=%d stdout.len=%d err=%v", dur, strings.Join(args, " "), len(stderr.Bytes()), len(stdout.Bytes()), err)
		return stdout.String(), stderr.String(), err
	}

	log.Printf("gitPushRepo: checking worktree status ...")
	if out, _, _ := run("git", "status", "--porcelain"); strings.TrimSpace(out) == "" {
		return "", nil
	}

	// Configure git user identity from GitHub API
	gitUserName := ""
	gitUserEmail := ""

	if githubToken != "" {
		req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
		req.Header.Set("Authorization", "token "+githubToken)
		req.Header.Set("Accept", "application/vnd.github+json")
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			switch resp.StatusCode {
			case 200:
				body, err := io.ReadAll(resp.Body)
				if err == nil {
					var ghUser struct {
						Login string `json:"login"`
						Name  string `json:"name"`
						Email string `json:"email"`
					}
					if err := json.Unmarshal(body, &ghUser); err == nil {
						if gitUserName == "" && ghUser.Name != "" {
							gitUserName = ghUser.Name
						} else if gitUserName == "" && ghUser.Login != "" {
							gitUserName = ghUser.Login
						}
						if gitUserEmail == "" && ghUser.Email != "" {
							gitUserEmail = ghUser.Email
						}
						log.Printf("gitPushRepo: fetched GitHub user name=%q email=%q", gitUserName, gitUserEmail)
					} else {
						log.Printf("Failed to parse GitHub user info: %v", err)
					}
				}
			case 403:
				log.Printf("gitPushRepo: GitHub API /user returned 403 (token lacks 'read:user' scope, using fallback identity)")
			default:
				log.Printf("gitPushRepo: GitHub API /user returned status %d", resp.StatusCode)
			}
		} else {
			log.Printf("gitPushRepo: failed to fetch GitHub user: %v", err)
		}
	}

	if gitUserName == "" {
		gitUserName = "Ambient Code Bot"
	}
	if gitUserEmail == "" {
		gitUserEmail = "bot@ambient-code.local"
	}
	run("git", "config", "user.name", gitUserName)
	run("git", "config", "user.email", gitUserEmail)
	log.Printf("gitPushRepo: configured git identity name=%q email=%q", gitUserName, gitUserEmail)

	// Stage and commit
	log.Printf("gitPushRepo: staging changes ...")
	_, _, _ = run("git", "add", "-A")

	cm := commitMessage
	if strings.TrimSpace(cm) == "" {
		cm = "Update from Ambient session"
	}

	log.Printf("gitPushRepo: committing changes ...")
	commitOut, commitErr, commitErrCode := run("git", "commit", "-m", cm)
	if commitErrCode != nil {
		log.Printf("gitPushRepo: commit failed (continuing): err=%v stderr=%q stdout=%q", commitErrCode, commitErr, commitOut)
	}

	// Determine target refspec
	ref := "HEAD"
	if branch == "auto" {
		cur, _, _ := run("git", "rev-parse", "--abbrev-ref", "HEAD")
		br := strings.TrimSpace(cur)
		if br == "" || br == "HEAD" {
			branch = "ambient-session"
			log.Printf("gitPushRepo: auto branch resolved to %q", branch)
		} else {
			branch = br
		}
	}
	if branch != "auto" {
		ref = "HEAD:" + branch
	}

	// Push with token authentication
	var pushArgs []string
	if githubToken != "" {
		cfg := fmt.Sprintf("url.https://x-access-token:%s@github.com/.insteadOf=https://github.com/", githubToken)
		pushArgs = []string{"git", "-c", cfg, "push", "-u", outputRepoURL, ref}
		log.Printf("gitPushRepo: running git push with token auth to %s %s", outputRepoURL, ref)
	} else {
		pushArgs = []string{"git", "push", "-u", outputRepoURL, ref}
		log.Printf("gitPushRepo: running git push %s %s in %s", outputRepoURL, ref, repoDir)
	}

	out, errOut, err := run(pushArgs...)
	if err != nil {
		serr := errOut
		if len(serr) > 2000 {
			serr = serr[:2000] + "..."
		}
		sout := out
		if len(sout) > 2000 {
			sout = sout[:2000] + "..."
		}
		log.Printf("gitPushRepo: push failed url=%q ref=%q err=%v stderr.snip=%q stdout.snip=%q", outputRepoURL, ref, err, serr, sout)
		// Use enhanced error detection for user-friendly messages
		return "", DetectPushError(outputRepoURL, errOut, out)
	}

	if len(out) > 2000 {
		out = out[:2000] + "..."
	}
	log.Printf("gitPushRepo: push ok url=%q ref=%q stdout.snip=%q", outputRepoURL, ref, out)
	return out, nil
}

// AbandonRepo discards all uncommitted changes in a repository directory
func AbandonRepo(ctx context.Context, repoDir string) error {
	if fi, err := os.Stat(repoDir); err != nil || !fi.IsDir() {
		return fmt.Errorf("repo directory not found: %s", repoDir)
	}

	run := func(args ...string) (string, string, error) {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = repoDir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		return stdout.String(), stderr.String(), err
	}

	log.Printf("gitAbandonRepo: git reset --hard in %s", repoDir)
	_, _, _ = run("git", "reset", "--hard")
	log.Printf("gitAbandonRepo: git clean -fd in %s", repoDir)
	_, _, _ = run("git", "clean", "-fd")
	return nil
}

// DiffRepo returns diff statistics comparing working directory to HEAD
func DiffRepo(ctx context.Context, repoDir string) (*DiffSummary, error) {
	// Validate repoDir exists
	if fi, err := os.Stat(repoDir); err != nil || !fi.IsDir() {
		return &DiffSummary{}, nil
	}

	run := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return stdout.String(), nil
	}

	summary := &DiffSummary{}

	// Get numstat for modified tracked files (working tree vs HEAD)
	numstatOut, err := run("git", "diff", "--numstat", "HEAD")
	if err == nil && strings.TrimSpace(numstatOut) != "" {
		lines := strings.Split(strings.TrimSpace(numstatOut), "\n")
		for _, ln := range lines {
			if ln == "" {
				continue
			}
			parts := strings.Fields(ln)
			if len(parts) < 3 {
				continue
			}
			added, removed := parts[0], parts[1]
			// Parse additions
			if added != "-" {
				var n int
				fmt.Sscanf(added, "%d", &n)
				summary.TotalAdded += n
			}
			// Parse deletions
			if removed != "-" {
				var n int
				fmt.Sscanf(removed, "%d", &n)
				summary.TotalRemoved += n
			}
			// If file was deleted (0 added, all removed), count as removed file
			if added == "0" && removed != "0" {
				summary.FilesRemoved++
			}
		}
	}

	// Get untracked files (new files not yet added to git)
	untrackedOut, err := run("git", "ls-files", "--others", "--exclude-standard")
	if err == nil && strings.TrimSpace(untrackedOut) != "" {
		untrackedFiles := strings.Split(strings.TrimSpace(untrackedOut), "\n")
		for _, filePath := range untrackedFiles {
			if filePath == "" {
				continue
			}
			// Count lines in the untracked file
			fullPath := filepath.Join(repoDir, filePath)
			if data, err := os.ReadFile(fullPath); err == nil {
				// Count lines (all lines in a new file are "added")
				lineCount := strings.Count(string(data), "\n")
				if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
					lineCount++ // Count last line if it doesn't end with newline
				}
				summary.TotalAdded += lineCount
				summary.FilesAdded++
			}
		}
	}

	log.Printf("gitDiffRepo: files_added=%d files_removed=%d total_added=%d total_removed=%d",
		summary.FilesAdded, summary.FilesRemoved, summary.TotalAdded, summary.TotalRemoved)
	return summary, nil
}

// ReadGitHubFile reads the content of a file from a GitHub repository
func ReadGitHubFile(ctx context.Context, owner, repo, branch, path, token string) ([]byte, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		owner, repo, path, branch)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s (body: %s)", resp.Status, string(body))
	}

	return io.ReadAll(resp.Body)
}

// CheckBranchExists checks if a branch exists in a GitHub repository
func CheckBranchExists(ctx context.Context, repoURL, branchName, githubToken string) (bool, error) {
	owner, repo, err := ParseGitHubURL(repoURL)
	if err != nil {
		return false, err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs/heads/%s",
		owner, repo, branchName)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("GitHub API error: %s (body: %s)", resp.Status, string(body))
}

// validatePushAccess checks if the user has push access to a repository (supports both GitHub and GitLab)
func validatePushAccess(ctx context.Context, repoURL, token string) error {
	provider := types.DetectProvider(repoURL)

	switch provider {
	case types.ProviderGitHub:
		return validateGitHubPushAccess(ctx, repoURL, token)
	case types.ProviderGitLab:
		return validateGitLabPushAccess(ctx, repoURL, token)
	default:
		return fmt.Errorf("unsupported repository provider for URL: %s", repoURL)
	}
}

// validateGitHubPushAccess checks if the user has push access to a GitHub repository
func validateGitHubPushAccess(ctx context.Context, repoURL, githubToken string) error {
	owner, repo, err := ParseGitHubURL(repoURL)
	if err != nil {
		return fmt.Errorf("invalid GitHub repository URL: %w", err)
	}

	// Use GitHub API to check repository permissions
	log.Printf("Validating push access to GitHub repo %s with token (len=%d)", repoURL, len(githubToken))
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check repository access: %w", err)
	}
	defer resp.Body.Close()

	// Read response body once
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("repository %s/%s not found or you don't have access to it", owner, repo)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		resetTime := resp.Header.Get("X-RateLimit-Reset")
		if resetTime != "" {
			return fmt.Errorf("GitHub API rate limit exceeded. Rate limit will reset at %s. Please try again later", resetTime)
		}
		return fmt.Errorf("GitHub API rate limit exceeded. Please try again later")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API error: %s (body: %s)", resp.Status, string(body))
	}

	// Parse response to check permissions
	var repoInfo struct {
		Permissions struct {
			Push bool `json:"push"`
		} `json:"permissions"`
	}

	if err := json.Unmarshal(body, &repoInfo); err != nil {
		return fmt.Errorf("failed to parse repository info: %w (body: %s)", err, string(body))
	}

	if !repoInfo.Permissions.Push {
		return fmt.Errorf("you don't have push access to %s. Please fork the repository or use a repository you have write access to", repoURL)
	}

	log.Printf("Validated push access to GitHub repo %s", repoURL)
	return nil
}

// validateGitLabPushAccess checks if the user has push access to a GitLab repository
func validateGitLabPushAccess(ctx context.Context, repoURL, gitlabToken string) error {
	parsed, err := gitlab.ParseGitLabURL(repoURL)
	if err != nil {
		return fmt.Errorf("invalid GitLab repository URL: %w", err)
	}

	// Use GitLab API to check repository permissions
	log.Printf("Validating push access to GitLab repo %s with token (len=%d)", repoURL, len(gitlabToken))

	// Get project details to check permissions
	// Note: parsed.ProjectID is already URL-encoded, don't double-encode it
	apiURL := fmt.Sprintf("%s/projects/%s", parsed.APIURL, parsed.ProjectID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+gitlabToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check repository access: %w", err)
	}
	defer resp.Body.Close()

	// Read response body once
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("repository %s/%s not found or you don't have access to it. Verify the repository URL and your GitLab token permissions", parsed.Owner, parsed.Repo)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("authentication failed for GitLab repository. Ensure your GitLab token has 'api' and 'write_repository' scopes")
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("GitLab API rate limit exceeded. Please wait a few minutes before retrying")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitLab API error: %s (body: %s)", resp.Status, string(body))
	}

	// Parse response to check permissions and ownership
	var projectInfo struct {
		Visibility string `json:"visibility"`
		Namespace  struct {
			Kind string `json:"kind"`
			Path string `json:"path"`
		} `json:"namespace"`
		Permissions struct {
			ProjectAccess *struct {
				AccessLevel int `json:"access_level"`
			} `json:"project_access"`
			GroupAccess *struct {
				AccessLevel int `json:"access_level"`
			} `json:"group_access"`
		} `json:"permissions"`
	}

	if err := json.Unmarshal(body, &projectInfo); err != nil {
		return fmt.Errorf("failed to parse project info: %w (body: %s)", err, string(body))
	}

	// For public repositories, GitLab may return null permissions
	// In this case, verify access by checking if we can get the authenticated user's info
	// and if the namespace matches
	if projectInfo.Permissions.ProjectAccess == nil && projectInfo.Permissions.GroupAccess == nil {
		log.Printf("GitLab repo %s has null permissions (likely public repo), verifying access via user info", repoURL)

		// Get authenticated user info to verify token and check namespace ownership
		userReq, err := http.NewRequestWithContext(ctx, "GET", parsed.APIURL+"/user", nil)
		if err != nil {
			return fmt.Errorf("failed to create user info request: %w", err)
		}
		userReq.Header.Set("Authorization", "Bearer "+gitlabToken)
		userReq.Header.Set("Accept", "application/json")

		userResp, err := http.DefaultClient.Do(userReq)
		if err != nil {
			return fmt.Errorf("failed to get user info: %w", err)
		}
		defer userResp.Body.Close()

		if userResp.StatusCode != http.StatusOK {
			return fmt.Errorf("unable to verify repository access. Token may not have sufficient permissions")
		}

		userBody, err := io.ReadAll(userResp.Body)
		if err != nil {
			return fmt.Errorf("failed to read user info: %w", err)
		}

		var userInfo struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal(userBody, &userInfo); err != nil {
			return fmt.Errorf("failed to parse user info: %w", err)
		}

		// For user namespaces, check if the authenticated user owns the namespace
		if projectInfo.Namespace.Kind == "user" && projectInfo.Namespace.Path == userInfo.Username {
			log.Printf("Validated push access to GitLab repo %s (owner: %s)", repoURL, userInfo.Username)
			return nil
		}

		// For public repos not owned by the user, we cannot guarantee push access
		// but if the token is valid and scoped correctly, assume access based on visibility
		if projectInfo.Visibility == "public" {
			log.Printf("Warning: GitLab repo %s is public but permissions are null. Assuming push access based on valid token", repoURL)
			return nil
		}

		return fmt.Errorf("unable to verify push access to %s. Repository may require explicit permissions", repoURL)
	}

	// GitLab access levels: 10=Guest, 20=Reporter, 30=Developer, 40=Maintainer, 50=Owner
	// Need at least Developer (30) to push
	hasAccess := false
	if projectInfo.Permissions.ProjectAccess != nil && projectInfo.Permissions.ProjectAccess.AccessLevel >= 30 {
		hasAccess = true
	}
	if projectInfo.Permissions.GroupAccess != nil && projectInfo.Permissions.GroupAccess.AccessLevel >= 30 {
		hasAccess = true
	}

	if !hasAccess {
		return fmt.Errorf("you don't have push access to %s. You need at least Developer (30) access level. Please check your permissions in GitLab", repoURL)
	}

	log.Printf("Validated push access to GitLab repo %s", repoURL)
	return nil
}

// createBranchInRepo creates a feature branch in a supporting repository
// Follows the same pattern as umbrella repo seeding but without adding files
// Note: This function assumes push access has already been validated by the caller
func createBranchInRepo(ctx context.Context, repo GitRepo, branchName, token string) error {
	repoURL := repo.GetURL()
	if repoURL == "" {
		return fmt.Errorf("repository URL is empty")
	}

	repoDir, err := os.MkdirTemp("", "supporting-repo-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(repoDir); err != nil {
			log.Printf("Warning: failed to cleanup temp directory %s: %v", repoDir, err)
		}
	}()

	authenticatedURL, err := InjectGitToken(repoURL, token)
	if err != nil {
		return fmt.Errorf("failed to prepare repo URL: %w", err)
	}

	baseBranch := "main"
	if branch := repo.GetBranch(); branch != nil && strings.TrimSpace(*branch) != "" {
		baseBranch = strings.TrimSpace(*branch)
	}

	log.Printf("Cloning supporting repo: %s (branch: %s)", repoURL, baseBranch)
	cloneArgs := []string{"clone", "--depth", "1", "--branch", baseBranch, authenticatedURL, repoDir}
	cmd := exec.CommandContext(ctx, "git", cloneArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repo: %w (output: %s)", err, string(out))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "config", "user.email", "vteam-bot@ambient-code.io")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Warning: failed to set git user.email: %v (output: %s)", err, string(out))
	}
	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "config", "user.name", "vTeam Bot")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Warning: failed to set git user.name: %v (output: %s)", err, string(out))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "ls-remote", "--heads", "origin", branchName)
	lsRemoteOut, lsRemoteErr := cmd.CombinedOutput()
	branchExistsRemotely := lsRemoteErr == nil && strings.TrimSpace(string(lsRemoteOut)) != ""

	if branchExistsRemotely {
		log.Printf("Branch '%s' already exists in %s, skipping", branchName, repoURL)
		return nil
	}

	log.Printf("Creating feature branch '%s' in %s", branchName, repoURL)
	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "checkout", "-b", branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create branch %s: %w (output: %s)", branchName, err, string(out))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "remote", "set-url", "origin", authenticatedURL)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set remote URL: %w (output: %s)", err, string(out))
	}

	// Push using HEAD:branchName refspec to ensure the newly created local branch is pushed
	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "push", "-u", "origin", fmt.Sprintf("HEAD:%s", branchName))
	if out, err := cmd.CombinedOutput(); err != nil {
		// Check if it's a permission error
		errMsg := string(out)
		if strings.Contains(errMsg, "Permission denied") || strings.Contains(errMsg, "403") || strings.Contains(errMsg, "not authorized") {
			return fmt.Errorf("permission denied: you don't have push access to %s. Please provide a repository you can push to", repoURL)
		}
		return fmt.Errorf("failed to push branch: %w (output: %s)", err, errMsg)
	}

	log.Printf("Successfully created and pushed branch '%s' in %s", branchName, repoURL)
	return nil
}

// InitRepo initializes a new git repository
func InitRepo(ctx context.Context, repoDir string) error {
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = repoDir

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to init git repo: %w (output: %s)", err, string(out))
	}

	// Configure default user if not set
	cmd = exec.CommandContext(ctx, "git", "config", "user.name", "Ambient Code Bot")
	cmd.Dir = repoDir
	_ = cmd.Run() // Best effort

	cmd = exec.CommandContext(ctx, "git", "config", "user.email", "bot@ambient-code.local")
	cmd.Dir = repoDir
	_ = cmd.Run() // Best effort

	return nil
}

// ConfigureRemote adds or updates a git remote
func ConfigureRemote(ctx context.Context, repoDir, remoteName, remoteURL string) error {
	// Try to remove existing remote first
	cmd := exec.CommandContext(ctx, "git", "remote", "remove", remoteName)
	cmd.Dir = repoDir
	_ = cmd.Run() // Ignore error if remote doesn't exist

	// Add the remote
	cmd = exec.CommandContext(ctx, "git", "remote", "add", remoteName, remoteURL)
	cmd.Dir = repoDir

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add remote: %w (output: %s)", err, string(out))
	}

	return nil
}

// MergeStatus contains information about merge conflict status
type MergeStatus struct {
	CanMergeClean      bool     `json:"canMergeClean"`
	LocalChanges       int      `json:"localChanges"`
	RemoteCommitsAhead int      `json:"remoteCommitsAhead"`
	ConflictingFiles   []string `json:"conflictingFiles"`
	RemoteBranchExists bool     `json:"remoteBranchExists"`
}

// CheckMergeStatus checks if local and remote can merge cleanly
func CheckMergeStatus(ctx context.Context, repoDir, branch string) (*MergeStatus, error) {
	if branch == "" {
		branch = "main"
	}

	status := &MergeStatus{
		ConflictingFiles: []string{},
	}

	run := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			return stdout.String(), err
		}
		return stdout.String(), nil
	}

	// Fetch remote branch
	_, err := run("git", "fetch", "origin", branch)
	if err != nil {
		// Remote branch doesn't exist yet
		status.RemoteBranchExists = false
		status.CanMergeClean = true
		return status, nil
	}
	status.RemoteBranchExists = true

	// Count local uncommitted changes
	statusOut, _ := run("git", "status", "--porcelain")
	status.LocalChanges = len(strings.Split(strings.TrimSpace(statusOut), "\n"))
	if strings.TrimSpace(statusOut) == "" {
		status.LocalChanges = 0
	}

	// Count commits on remote but not local
	countOut, _ := run("git", "rev-list", "--count", "HEAD..origin/"+branch)
	fmt.Sscanf(strings.TrimSpace(countOut), "%d", &status.RemoteCommitsAhead)

	// Test merge to detect conflicts (dry run)
	mergeBase, err := run("git", "merge-base", "HEAD", "origin/"+branch)
	if err != nil {
		// No common ancestor - unrelated histories
		// This is NOT a conflict - we can merge with --allow-unrelated-histories
		// which is already used in PullRepo and SyncRepo
		status.CanMergeClean = true
		status.ConflictingFiles = []string{}
		return status, nil
	}

	// Use git merge-tree to simulate merge without touching working directory
	mergeTreeOut, err := run("git", "merge-tree", strings.TrimSpace(mergeBase), "HEAD", "origin/"+branch)
	if err == nil && strings.TrimSpace(mergeTreeOut) != "" {
		// Check for conflict markers in output
		if strings.Contains(mergeTreeOut, "<<<<<<<") {
			status.CanMergeClean = false
			// Parse conflicting files from merge-tree output
			for _, line := range strings.Split(mergeTreeOut, "\n") {
				if strings.HasPrefix(line, "--- a/") || strings.HasPrefix(line, "+++ b/") {
					file := strings.TrimPrefix(strings.TrimPrefix(line, "--- a/"), "+++ b/")
					if file != "" && !contains(status.ConflictingFiles, file) {
						status.ConflictingFiles = append(status.ConflictingFiles, file)
					}
				}
			}
		} else {
			status.CanMergeClean = true
		}
	} else {
		status.CanMergeClean = true
	}

	return status, nil
}

// PullRepo pulls changes from remote branch
func PullRepo(ctx context.Context, repoDir, branch string) error {
	if branch == "" {
		branch = "main"
	}

	cmd := exec.CommandContext(ctx, "git", "pull", "--allow-unrelated-histories", "origin", branch)
	cmd.Dir = repoDir

	if out, err := cmd.CombinedOutput(); err != nil {
		outStr := string(out)
		if strings.Contains(outStr, "CONFLICT") {
			return fmt.Errorf("merge conflicts detected: %s", outStr)
		}
		return fmt.Errorf("failed to pull: %w (output: %s)", err, outStr)
	}

	log.Printf("Successfully pulled from origin/%s", branch)
	return nil
}

// PushToRepo pushes local commits to specified branch
func PushToRepo(ctx context.Context, repoDir, branch, commitMessage string) error {
	if branch == "" {
		branch = "main"
	}

	run := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout
		err := cmd.Run()
		return stdout.String(), err
	}

	// Ensure we're on the correct branch (create if needed)
	// This handles fresh git init repos that don't have a branch yet
	if _, err := run("git", "checkout", "-B", branch); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	// Stage all changes
	if _, err := run("git", "add", "."); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Commit if there are changes
	if out, err := run("git", "commit", "-m", commitMessage); err != nil {
		if !strings.Contains(out, "nothing to commit") {
			return fmt.Errorf("failed to commit: %w", err)
		}
	}

	// Push to branch
	if out, err := run("git", "push", "-u", "origin", branch); err != nil {
		return fmt.Errorf("failed to push: %w (output: %s)", err, out)
	}

	log.Printf("Successfully pushed to origin/%s", branch)
	return nil
}

// CreateBranch creates a new branch and pushes it to remote
func CreateBranch(ctx context.Context, repoDir, branchName string) error {
	run := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout
		err := cmd.Run()
		return stdout.String(), err
	}

	// Create and checkout new branch
	if _, err := run("git", "checkout", "-b", branchName); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Push to remote using HEAD:branchName refspec
	if out, err := run("git", "push", "-u", "origin", fmt.Sprintf("HEAD:%s", branchName)); err != nil {
		return fmt.Errorf("failed to push new branch: %w (output: %s)", err, out)
	}

	log.Printf("Successfully created and pushed branch %s", branchName)
	return nil
}

// ListRemoteBranches lists all branches in the remote repository
func ListRemoteBranches(ctx context.Context, repoDir string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", "origin")
	cmd.Dir = repoDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list remote branches: %w", err)
	}

	branches := []string{}
	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Format: "commit-hash refs/heads/branch-name"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			ref := parts[1]
			branchName := strings.TrimPrefix(ref, "refs/heads/")
			branches = append(branches, branchName)
		}
	}

	return branches, nil
}

// SyncRepo commits, pulls, and pushes changes
func SyncRepo(ctx context.Context, repoDir, commitMessage, branch string) error {
	if branch == "" {
		branch = "main"
	}

	// Stage all changes
	cmd := exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage changes: %w (output: %s)", err, string(out))
	}

	// Commit changes (only if there are changes)
	cmd = exec.CommandContext(ctx, "git", "commit", "-m", commitMessage)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		// Check if error is "nothing to commit"
		outStr := string(out)
		if !strings.Contains(outStr, "nothing to commit") && !strings.Contains(outStr, "no changes added") {
			return fmt.Errorf("failed to commit: %w (output: %s)", err, outStr)
		}
		// Nothing to commit is not an error
		log.Printf("SyncRepo: nothing to commit in %s", repoDir)
	}

	// Pull with rebase to sync with remote
	cmd = exec.CommandContext(ctx, "git", "pull", "--rebase", "origin", branch)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		outStr := string(out)
		// Check if it's just "no tracking information" (first push)
		if !strings.Contains(outStr, "no tracking information") && !strings.Contains(outStr, "couldn't find remote ref") {
			return fmt.Errorf("failed to pull: %w (output: %s)", err, outStr)
		}
		log.Printf("SyncRepo: pull skipped (no remote tracking): %s", outStr)
	}

	// Push to remote
	cmd = exec.CommandContext(ctx, "git", "push", "-u", "origin", branch)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		outStr := string(out)
		if strings.Contains(outStr, "Permission denied") || strings.Contains(outStr, "403") {
			return fmt.Errorf("permission denied: no push access to remote")
		}
		return fmt.Errorf("failed to push: %w (output: %s)", err, outStr)
	}

	log.Printf("Successfully synchronized %s to %s", repoDir, branch)
	return nil
}

// Helper function to check if string slice contains a value
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
