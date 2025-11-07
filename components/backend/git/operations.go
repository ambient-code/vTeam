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

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Package-level dependencies (set from main package)
var (
	GetProjectSettingsResource func() schema.GroupVersionResource
	GetGitHubInstallation      func(context.Context, string) (interface{}, error)
	GitHubTokenManager         interface{} // *GitHubTokenManager from main package
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

// getProjectSettings retrieves the ProjectSettings CR for a project using the provided dynamic client
func getProjectSettings(ctx context.Context, dynClient dynamic.Interface, projectName string) (*ProjectSettings, error) {
	if dynClient == nil {
		return &ProjectSettings{}, nil
	}

	if GetProjectSettingsResource == nil {
		return &ProjectSettings{}, nil
	}

	gvr := GetProjectSettingsResource()
	obj, err := dynClient.Resource(gvr).Namespace(projectName).Get(ctx, "projectsettings", v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return &ProjectSettings{}, nil
		}
		return nil, fmt.Errorf("failed to get ProjectSettings: %w", err)
	}

	settings := &ProjectSettings{}
	if obj != nil {
		if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
			if v, ok := spec["runnerSecretsName"].(string); ok {
				settings.RunnerSecret = strings.TrimSpace(v)
			}
		}
	}

	return settings, nil
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

	// Fall back to project runner secret GIT_TOKEN
	if k8sClient == nil {
		log.Printf("Cannot read runner secret: k8s client is nil")
		return "", fmt.Errorf("no GitHub credentials available. Either connect GitHub App or configure GIT_TOKEN in project runner secret")
	}

	settings, err := getProjectSettings(ctx, dynClient, project)

	// Default to "ambient-runner-secrets" if not configured
	secretName := "ambient-runner-secrets"
	if err != nil {
		log.Printf("Failed to get ProjectSettings for %s (using default secret name): %v", project, err)
	} else if settings != nil && settings.RunnerSecret != "" {
		secretName = settings.RunnerSecret
	}

	log.Printf("Attempting to read GIT_TOKEN from secret %s/%s", project, secretName)

	secret, err := k8sClient.CoreV1().Secrets(project).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		log.Printf("Failed to get runner secret %s/%s: %v", project, secretName, err)
		return "", fmt.Errorf("no GitHub credentials available. Either connect GitHub App or configure GIT_TOKEN in project runner secret")
	}

	if secret.Data == nil {
		log.Printf("Secret %s/%s exists but Data is nil", project, secretName)
		return "", fmt.Errorf("no GitHub credentials available. Either connect GitHub App or configure GIT_TOKEN in project runner secret")
	}

	token, ok := secret.Data["GIT_TOKEN"]
	if !ok {
		log.Printf("Secret %s/%s exists but has no GIT_TOKEN key (available keys: %v)", project, secretName, getSecretKeys(secret.Data))
		return "", fmt.Errorf("no GitHub credentials available. Either connect GitHub App or configure GIT_TOKEN in project runner secret")
	}

	if len(token) == 0 {
		log.Printf("Secret %s/%s has GIT_TOKEN key but value is empty", project, secretName)
		return "", fmt.Errorf("no GitHub credentials available. Either connect GitHub App or configure GIT_TOKEN in project runner secret")
	}

	log.Printf("Using GIT_TOKEN from project runner secret %s/%s", project, secretName)
	return string(token), nil
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
func CheckRepoSeeding(ctx context.Context, repoURL string, branch *string, githubToken string) (bool, map[string]interface{}, error) {
	owner, repo, err := ParseGitHubURL(repoURL)
	if err != nil {
		return false, nil, err
	}

	branchName := "main"
	if branch != nil && strings.TrimSpace(*branch) != "" {
		branchName = strings.TrimSpace(*branch)
	}

	claudeExists, err := checkGitHubPathExists(ctx, owner, repo, branchName, ".claude", githubToken)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check .claude: %w", err)
	}

	// Check for .claude/commands directory (spec-kit slash commands)
	claudeCommandsExists, err := checkGitHubPathExists(ctx, owner, repo, branchName, ".claude/commands", githubToken)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check .claude/commands: %w", err)
	}

	// Check for .claude/agents directory
	claudeAgentsExists, err := checkGitHubPathExists(ctx, owner, repo, branchName, ".claude/agents", githubToken)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check .claude/agents: %w", err)
	}

	// Check for .specify directory (from spec-kit)
	specifyExists, err := checkGitHubPathExists(ctx, owner, repo, branchName, ".specify", githubToken)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check .specify: %w", err)
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
func PerformRepoSeeding(ctx context.Context, wf Workflow, branchName, githubToken, agentURL, agentBranch, agentPath, specKitRepo, specKitVersion, specKitTemplate string) (bool, error) {
	umbrellaRepo := wf.GetUmbrellaRepo()
	if umbrellaRepo == nil {
		return false, fmt.Errorf("workflow has no spec repo")
	}

	if branchName == "" {
		return false, fmt.Errorf("branchName is required")
	}

	// Validate push access to spec repo before starting
	log.Printf("Validating push access to spec repo: %s", umbrellaRepo.GetURL())
	if err := validatePushAccess(ctx, umbrellaRepo.GetURL(), githubToken); err != nil {
		return false, fmt.Errorf("spec repo access validation failed: %w", err)
	}

	// Validate push access to all supporting repos before starting
	supportingRepos := wf.GetSupportingRepos()
	if len(supportingRepos) > 0 {
		log.Printf("Validating push access to %d supporting repos", len(supportingRepos))
		for i, repo := range supportingRepos {
			if err := validatePushAccess(ctx, repo.GetURL(), githubToken); err != nil {
				return false, fmt.Errorf("supporting repo #%d (%s) access validation failed: %w", i+1, repo.GetURL(), err)
			}
		}
	}

	umbrellaDir, err := os.MkdirTemp("", "umbrella-*")
	if err != nil {
		return false, fmt.Errorf("failed to create temp dir for spec repo: %w", err)
	}
	defer os.RemoveAll(umbrellaDir)

	agentSrcDir, err := os.MkdirTemp("", "agents-*")
	if err != nil {
		return false, fmt.Errorf("failed to create temp dir for agent source: %w", err)
	}
	defer os.RemoveAll(agentSrcDir)

	// Clone umbrella repo with authentication
	log.Printf("Cloning umbrella repo: %s", umbrellaRepo.GetURL())
	authenticatedURL, err := InjectGitHubToken(umbrellaRepo.GetURL(), githubToken)
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
	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "ls-remote", "--heads", "origin", branchName)
	lsRemoteOut, lsRemoteErr := cmd.CombinedOutput()
	branchExistsRemotely := lsRemoteErr == nil && strings.TrimSpace(string(lsRemoteOut)) != ""

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
	// Note: we already validated push access to all repos above, so any failure here is unexpected
	if len(supportingRepos) > 0 {
		log.Printf("Creating feature branch %s in %d supporting repos", branchName, len(supportingRepos))
		for i, repo := range supportingRepos {
			if err := createBranchInRepo(ctx, repo, branchName, githubToken); err != nil {
				return false, fmt.Errorf("failed to create branch in supporting repo #%d (%s): %w", i+1, repo.GetURL(), err)
			}
		}
	}

	return branchExistsRemotely, nil
}

// InjectGitHubToken injects a GitHub token into a git URL for authentication
func InjectGitHubToken(gitURL, token string) (string, error) {
	u, err := url.Parse(gitURL)
	if err != nil {
		return "", fmt.Errorf("invalid git URL: %w", err)
	}

	if u.Scheme != "https" {
		return gitURL, nil
	}

	u.User = url.UserPassword("x-access-token", token)
	return u.String(), nil
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
				var ghUser struct {
					Login string `json:"login"`
					Name  string `json:"name"`
					Email string `json:"email"`
				}
				if json.Unmarshal([]byte(fmt.Sprintf("%v", resp.Body)), &ghUser) == nil {
					if gitUserName == "" && ghUser.Name != "" {
						gitUserName = ghUser.Name
					} else if gitUserName == "" && ghUser.Login != "" {
						gitUserName = ghUser.Login
					}
					if gitUserEmail == "" && ghUser.Email != "" {
						gitUserEmail = ghUser.Email
					}
					log.Printf("gitPushRepo: fetched GitHub user name=%q email=%q", gitUserName, gitUserEmail)
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
		return "", fmt.Errorf("push failed: %s", errOut)
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

// validatePushAccess checks if the user has push access to a repository via GitHub API
func validatePushAccess(ctx context.Context, repoURL, githubToken string) error {
	owner, repo, err := ParseGitHubURL(repoURL)
	if err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}

	// Use GitHub API to check repository permissions
	log.Printf("Validating push access to %s with token (len=%d)", repoURL, len(githubToken))
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

	log.Printf("Validated push access to %s", repoURL)
	return nil
}

// createBranchInRepo creates a feature branch in a supporting repository
// Follows the same pattern as umbrella repo seeding but without adding files
// Note: This function assumes push access has already been validated by the caller
func createBranchInRepo(ctx context.Context, repo GitRepo, branchName, githubToken string) error {
	repoURL := repo.GetURL()
	if repoURL == "" {
		return fmt.Errorf("repository URL is empty")
	}

	repoDir, err := os.MkdirTemp("", "supporting-repo-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(repoDir)

	authenticatedURL, err := InjectGitHubToken(repoURL, githubToken)
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

	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "push", "-u", "origin", branchName)
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
