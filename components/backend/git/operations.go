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
}

// PerformRepoSeeding performs the actual seeding operations
// wf parameter should implement the Workflow interface
func PerformRepoSeeding(ctx context.Context, wf Workflow, githubToken, agentURL, agentBranch, agentPath, specKitVersion, specKitTemplate string) error {
	umbrellaRepo := wf.GetUmbrellaRepo()
	if umbrellaRepo == nil {
		return fmt.Errorf("workflow has no umbrella repo")
	}

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

	// Clone umbrella repo with authentication
	log.Printf("Cloning umbrella repo: %s", umbrellaRepo.GetURL())
	authenticatedURL, err := InjectGitHubToken(umbrellaRepo.GetURL(), githubToken)
	if err != nil {
		return fmt.Errorf("failed to prepare umbrella repo URL: %w", err)
	}

	umbrellaArgs := []string{"clone", "--depth", "1"}
	if branch := umbrellaRepo.GetBranch(); branch != nil && strings.TrimSpace(*branch) != "" {
		umbrellaArgs = append(umbrellaArgs, "--branch", strings.TrimSpace(*branch))
	}
	umbrellaArgs = append(umbrellaArgs, authenticatedURL, umbrellaDir)

	cmd := exec.CommandContext(ctx, "git", umbrellaArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone umbrella repo: %w (output: %s)", err, string(out))
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

	// Download and extract spec-kit template
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

	// Extract spec-kit files
	specKitFilesAdded := 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}

		rel := strings.TrimPrefix(f.Name, "./")
		rel = strings.ReplaceAll(rel, "\\", "/")
		for strings.Contains(rel, "../") {
			rel = strings.ReplaceAll(rel, "../", "")
		}

		targetPath := filepath.Join(umbrellaDir, rel)

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

		if err := os.WriteFile(targetPath, content, 0644); err != nil {
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
		return fmt.Errorf("failed to clone agent source: %w (output: %s)", err, string(out))
	}

	// Copy agent markdown files to .claude/agents/
	agentSourcePath := filepath.Join(agentSrcDir, agentPath)
	claudeDir := filepath.Join(umbrellaDir, ".claude")
	claudeAgentsDir := filepath.Join(claudeDir, "agents")
	if err := os.MkdirAll(claudeAgentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude/agents directory: %w", err)
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
		return fmt.Errorf("failed to copy agents: %w", err)
	}
	log.Printf("Copied %d agent files", agentsCopied)

	// Commit and push changes
	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %w (output: %s)", err, string(out))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "diff", "--cached", "--quiet")
	if err := cmd.Run(); err == nil {
		log.Printf("No changes to commit for seeding")
		return nil
	}

	cmd = exec.CommandContext(ctx, "git", "-C", umbrellaDir, "commit", "-m", "chore: seed workspace with spec-kit and agents")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %w (output: %s)", err, string(out))
	}

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
			if resp.StatusCode == 200 {
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
			} else if resp.StatusCode == 403 {
				log.Printf("gitPushRepo: GitHub API /user returned 403 (token lacks 'read:user' scope, using fallback identity)")
			} else {
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

	// Get numstat for modified files (working tree vs HEAD)
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
		}
	}

	log.Printf("gitDiffRepo: total_added=%d total_removed=%d",
		summary.TotalAdded, summary.TotalRemoved)
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
