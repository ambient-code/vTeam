package main

import (
	"archive/zip"
	"bytes"
	"context"
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

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ProjectSettings represents the project configuration
type ProjectSettings struct {
	RunnerSecret string
}

// getProjectSettings retrieves the ProjectSettings CR for a project using the provided dynamic client
func getProjectSettings(ctx context.Context, dynClient dynamic.Interface, projectName string) (*ProjectSettings, error) {
	if dynClient == nil {
		return &ProjectSettings{}, nil
	}

	gvr := getProjectSettingsResource()
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

// getGitHubToken tries to get a GitHub token from GitHub App first, then falls back to project runner secret
func getGitHubToken(ctx context.Context, k8sClient *kubernetes.Clientset, dynClient dynamic.Interface, project, userID string) (string, error) {
	// Try GitHub App first
	installation, err := getGitHubInstallation(ctx, userID)
	if err == nil && githubTokenManager != nil {
		// GitHub App is available - mint token
		token, _, err := githubTokenManager.MintInstallationTokenForHost(ctx, installation.InstallationID, installation.Host)
		if err == nil && token != "" {
			return token, nil
		}
		log.Printf("Failed to mint GitHub App token for user %s: %v", userID, err)
	}

	// Fall back to project runner secret GIT_TOKEN
	settings, err := getProjectSettings(ctx, dynClient, project)

	// Default to "ambient-runner-secrets" if not configured (same as updateRunnerSecrets handler)
	secretName := "ambient-runner-secrets"
	if err != nil {
		log.Printf("Failed to get ProjectSettings for %s (using default secret name): %v", project, err)
	} else if settings != nil && settings.RunnerSecret != "" {
		secretName = settings.RunnerSecret
	}

	// Use user-scoped client to read secret in their namespace
	secret, err := k8sClient.CoreV1().Secrets(project).Get(ctx, secretName, v1.GetOptions{})
	if err == nil && secret.Data != nil {
		if token, ok := secret.Data["GIT_TOKEN"]; ok && len(token) > 0 {
			log.Printf("Using GIT_TOKEN from project runner secret %s for git operations", secretName)
			return string(token), nil
		}
		log.Printf("Secret %s/%s exists but has no GIT_TOKEN key", project, secretName)
	} else if err != nil {
		log.Printf("Failed to get runner secret %s/%s: %v", project, secretName, err)
	}

	return "", fmt.Errorf("no GitHub credentials available. Either connect GitHub App or configure GIT_TOKEN in project runner secret")
}

// checkRepoSeeding checks if a repo has been seeded by verifying .claude/ and .specify/ exist
func checkRepoSeeding(ctx context.Context, repoURL string, branch *string, githubToken string) (bool, map[string]interface{}, error) {
	// Parse repo URL to get owner/repo
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		return false, nil, err
	}

	branchName := "main"
	if branch != nil && strings.TrimSpace(*branch) != "" {
		branchName = strings.TrimSpace(*branch)
	}

	// Check for .claude directory
	claudeExists, err := checkGitHubPathExists(ctx, owner, repo, branchName, ".claude", githubToken)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check .claude: %w", err)
	}

	// Check for .specify directory (from spec-kit)
	specifyExists, err := checkGitHubPathExists(ctx, owner, repo, branchName, ".specify", githubToken)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check .specify: %w", err)
	}

	details := map[string]interface{}{
		"claudeExists":  claudeExists,
		"specifyExists": specifyExists,
	}

	isSeeded := claudeExists && specifyExists
	return isSeeded, details, nil
}

// parseGitHubURL extracts owner and repo from a GitHub URL
func parseGitHubURL(gitURL string) (owner, repo string, err error) {
	// Handle both https://github.com/owner/repo and git@github.com:owner/repo
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

// checkGitHubPathExists checks if a path exists in a GitHub repo
func checkGitHubPathExists(ctx context.Context, owner, repo, branch, path, token string) (bool, error) {
	// Use GitHub Contents API
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

	// 200 = exists, 404 = doesn't exist
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	// Other error
	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("GitHub API error: %s (body: %s)", resp.Status, string(body))
}

// performRepoSeeding performs the actual seeding operations
func performRepoSeeding(ctx context.Context, wf *RFEWorkflow, githubToken, agentURL, agentBranch, agentPath, specKitVersion, specKitTemplate string) error {
	// Create temp directories
	umbrellaDir, err := os.MkdirTemp("", "umbrella-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for umbrella repo: %w", err)
	}
	defer os.RemoveAll(umbrellaDir)

	agentSrcDir, err := os.MkdirTemp("", "agents-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for agent source: %w", err)
	}
	defer os.RemoveAll(agentSrcDir)

	// 1. Clone umbrella repo with authentication
	log.Printf("Cloning umbrella repo: %s", wf.UmbrellaRepo.URL)
	authenticatedURL, err := injectGitHubToken(wf.UmbrellaRepo.URL, githubToken)
	if err != nil {
		return fmt.Errorf("failed to prepare umbrella repo URL: %w", err)
	}
	umbrellaArgs := []string{"clone", "--depth", "1"}
	if wf.UmbrellaRepo.Branch != nil && strings.TrimSpace(*wf.UmbrellaRepo.Branch) != "" {
		umbrellaArgs = append(umbrellaArgs, "--branch", strings.TrimSpace(*wf.UmbrellaRepo.Branch))
	}
	umbrellaArgs = append(umbrellaArgs, authenticatedURL, umbrellaDir)
	cmd := exec.CommandContext(ctx, "git", umbrellaArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone umbrella repo: %w (output: %s)", err, string(out))
	}

	// Configure git user for commits
	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "config", "user.email", "vteam-bot@ambient-code.io")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Warning: failed to set git user.email: %v (output: %s)", err, string(out))
	}
	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "config", "user.name", "vTeam Bot")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Warning: failed to set git user.name: %v (output: %s)", err, string(out))
	}

	// 2. Download and extract spec-kit template
	log.Printf("Downloading spec-kit template: %s", specKitVersion)
	specKitURL := fmt.Sprintf("https://github.com/github/spec-kit/releases/download/%s/%s-%s.zip", specKitVersion, specKitTemplate, specKitVersion)
	resp, err := http.Get(specKitURL)
	if err != nil {
		return fmt.Errorf("failed to download spec-kit: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("spec-kit download failed with status: %s", resp.Status)
	}

	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read spec-kit zip: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("failed to open spec-kit zip: %w", err)
	}

	// Extract spec-kit files to umbrella repo (skip if exists)
	specKitFilesAdded := 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// Normalize path
		rel := strings.TrimPrefix(f.Name, "./")
		rel = strings.ReplaceAll(rel, "\\", "/")
		for strings.Contains(rel, "../") {
			rel = strings.ReplaceAll(rel, "../", "")
		}

		targetPath := filepath.Join(umbrellaDir, rel)

		// Skip if file already exists
		if _, err := os.Stat(targetPath); err == nil {
			continue
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			log.Printf("Failed to create dir for %s: %v", rel, err)
			continue
		}

		// Extract file
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

		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			log.Printf("Failed to write %s: %v", targetPath, err)
			continue
		}
		specKitFilesAdded++
	}
	log.Printf("Extracted %d spec-kit files", specKitFilesAdded)

	// 3. Clone agent source repo
	log.Printf("Cloning agent source: %s", agentURL)
	agentArgs := []string{"clone", "--depth", "1"}
	if agentBranch != "" {
		agentArgs = append(agentArgs, "--branch", agentBranch)
	}
	agentArgs = append(agentArgs, agentURL, agentSrcDir)
	cmd = exec.CommandContext(ctx, "git", agentArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone agent source: %w (output: %s)", err, string(out))
	}

	// 4. Copy agent markdown files to .claude/
	agentSourcePath := filepath.Join(agentSrcDir, agentPath)
	claudeDir := filepath.Join(umbrellaDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
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

		targetPath := filepath.Join(claudeDir, d.Name())
		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			log.Printf("Failed to write agent file %s: %v", targetPath, err)
			return nil
		}
		agentsCopied++
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to copy agents: %w", err)
	}
	log.Printf("Copied %d agent files", agentsCopied)

	// 5. Commit and push changes
	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %w (output: %s)", err, string(out))
	}

	// Check if there are changes to commit
	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "diff", "--cached", "--quiet")
	if err := cmd.Run(); err == nil {
		// No changes to commit
		log.Printf("No changes to commit for seeding")
		return nil
	}

	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "commit", "-m", "chore: seed workspace with spec-kit and agents")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %w (output: %s)", err, string(out))
	}

	// Configure git to use the authenticated URL for push
	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "remote", "set-url", "origin", authenticatedURL)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set remote URL: %w (output: %s)", err, string(out))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "push")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %w (output: %s)", err, string(out))
	}

	log.Printf("Successfully seeded umbrella repo")
	return nil
}

// injectGitHubToken injects a GitHub token into a git URL for authentication
func injectGitHubToken(gitURL, token string) (string, error) {
	// Parse the URL
	u, err := url.Parse(gitURL)
	if err != nil {
		return "", fmt.Errorf("invalid git URL: %w", err)
	}

	// Only inject token for https URLs
	if u.Scheme != "https" {
		return gitURL, nil // Return as-is for ssh or other schemes
	}

	// Set credentials: GitHub App tokens use x-access-token as username
	u.User = url.UserPassword("x-access-token", token)
	return u.String(), nil
}
