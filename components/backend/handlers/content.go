package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ambient-code-backend/git"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/gin-gonic/gin"
)

// StateBaseDir is the base directory for content storage
// Set by main during initialization
var StateBaseDir string

// MaxResultFileSize is the maximum size for result files to prevent memory issues
const MaxResultFileSize = 10 * 1024 * 1024 // 10MB

// MaxGlobMatches limits the number of files that can be matched to prevent resource exhaustion
const MaxGlobMatches = 100

// Git operation functions - set by main package during initialization
// These are set to the actual implementations from git package
var (
	GitPushRepo           func(ctx context.Context, repoDir, commitMessage, outputRepoURL, branch, githubToken string) (string, error)
	GitAbandonRepo        func(ctx context.Context, repoDir string) error
	GitDiffRepo           func(ctx context.Context, repoDir string) (*git.DiffSummary, error)
	GitCheckMergeStatus   func(ctx context.Context, repoDir, branch string) (*git.MergeStatus, error)
	GitPullRepo           func(ctx context.Context, repoDir, branch string) error
	GitPushToRepo         func(ctx context.Context, repoDir, branch, commitMessage string) error
	GitCreateBranch       func(ctx context.Context, repoDir, branchName string) error
	GitListRemoteBranches func(ctx context.Context, repoDir string) ([]string, error)
)

// ContentGitPush handles POST /content/github/push in CONTENT_SERVICE_MODE
func ContentGitPush(c *gin.Context) {
	var body struct {
		RepoPath      string `json:"repoPath"`
		CommitMessage string `json:"commitMessage"`
		OutputRepoURL string `json:"outputRepoUrl"`
		Branch        string `json:"branch"`
	}
	_ = c.BindJSON(&body)
	log.Printf("contentGitPush: request received repoPath=%q outputRepoUrl=%q branch=%q commitLen=%d", body.RepoPath, body.OutputRepoURL, body.Branch, len(strings.TrimSpace(body.CommitMessage)))

	// Require explicit output repo URL and branch from caller
	if strings.TrimSpace(body.OutputRepoURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing outputRepoUrl"})
		return
	}
	if strings.TrimSpace(body.Branch) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing branch"})
		return
	}

	repoDir := filepath.Clean(filepath.Join(StateBaseDir, body.RepoPath))
	if body.RepoPath == "" {
		repoDir = StateBaseDir
	}

	// Basic safety: repoDir must be under StateBaseDir
	if !strings.HasPrefix(repoDir+string(os.PathSeparator), StateBaseDir+string(os.PathSeparator)) && repoDir != StateBaseDir {
		log.Printf("contentGitPush: invalid repoPath resolved=%q stateBaseDir=%q", repoDir, StateBaseDir)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repoPath"})
		return
	}

	log.Printf("contentGitPush: using repoDir=%q (stateBaseDir=%q)", repoDir, StateBaseDir)

	// Optional GitHub token provided by backend via internal header
	gitHubToken := strings.TrimSpace(c.GetHeader("X-GitHub-Token"))
	log.Printf("contentGitPush: tokenHeaderPresent=%t url.host.redacted=%t branch=%q", gitHubToken != "", strings.HasPrefix(body.OutputRepoURL, "https://"), body.Branch)

	// Call refactored git push function
	out, err := GitPushRepo(c.Request.Context(), repoDir, body.CommitMessage, body.OutputRepoURL, body.Branch, gitHubToken)
	if err != nil {
		if out == "" {
			// No changes to commit
			c.JSON(http.StatusOK, gin.H{"ok": true, "message": "no changes"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "push failed", "stderr": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "stdout": out})
}

// ContentGitAbandon handles POST /content/github/abandon
func ContentGitAbandon(c *gin.Context) {
	var body struct {
		RepoPath string `json:"repoPath"`
	}
	_ = c.BindJSON(&body)
	log.Printf("contentGitAbandon: request repoPath=%q", body.RepoPath)

	repoDir := filepath.Clean(filepath.Join(StateBaseDir, body.RepoPath))
	if body.RepoPath == "" {
		repoDir = StateBaseDir
	}

	if !strings.HasPrefix(repoDir+string(os.PathSeparator), StateBaseDir+string(os.PathSeparator)) && repoDir != StateBaseDir {
		log.Printf("contentGitAbandon: invalid repoPath resolved=%q base=%q", repoDir, StateBaseDir)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repoPath"})
		return
	}

	log.Printf("contentGitAbandon: using repoDir=%q", repoDir)

	if err := GitAbandonRepo(c.Request.Context(), repoDir); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ContentGitDiff handles GET /content/github/diff
func ContentGitDiff(c *gin.Context) {
	repoPath := strings.TrimSpace(c.Query("repoPath"))
	if repoPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing repoPath"})
		return
	}

	repoDir := filepath.Clean(filepath.Join(StateBaseDir, repoPath))
	if !strings.HasPrefix(repoDir+string(os.PathSeparator), StateBaseDir+string(os.PathSeparator)) && repoDir != StateBaseDir {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repoPath"})
		return
	}

	log.Printf("contentGitDiff: repoPath=%q repoDir=%q", repoPath, repoDir)

	summary, err := GitDiffRepo(c.Request.Context(), repoDir)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"files": gin.H{
				"added":   0,
				"removed": 0,
			},
			"total_added":   0,
			"total_removed": 0,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files": gin.H{
			"added":   summary.FilesAdded,
			"removed": summary.FilesRemoved,
		},
		"total_added":   summary.TotalAdded,
		"total_removed": summary.TotalRemoved,
	})
}

// ContentGitStatus handles GET /content/git-status?path=
func ContentGitStatus(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	abs := filepath.Join(StateBaseDir, path)

	// Check if directory exists
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		c.JSON(http.StatusOK, gin.H{
			"initialized": false,
			"hasChanges":  false,
		})
		return
	}

	// Check if git repo exists
	gitDir := filepath.Join(abs, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"initialized": false,
			"hasChanges":  false,
		})
		return
	}

	// Get git status using existing git package
	summary, err := GitDiffRepo(c.Request.Context(), abs)
	if err != nil {
		log.Printf("ContentGitStatus: git diff failed: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"initialized": true,
			"hasChanges":  false,
		})
		return
	}

	hasChanges := summary.FilesAdded > 0 || summary.FilesRemoved > 0 || summary.TotalAdded > 0 || summary.TotalRemoved > 0

	c.JSON(http.StatusOK, gin.H{
		"initialized":      true,
		"hasChanges":       hasChanges,
		"filesAdded":       summary.FilesAdded,
		"filesRemoved":     summary.FilesRemoved,
		"uncommittedFiles": summary.FilesAdded + summary.FilesRemoved,
		"totalAdded":       summary.TotalAdded,
		"totalRemoved":     summary.TotalRemoved,
	})
}

// ContentGitConfigureRemote handles POST /content/git-configure-remote
// Body: { path: string, remoteURL: string, branch: string }
func ContentGitConfigureRemote(c *gin.Context) {
	var body struct {
		Path      string `json:"path"`
		RemoteURL string `json:"remoteUrl"`
		Branch    string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	path := filepath.Clean("/" + body.Path)
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	abs := filepath.Join(StateBaseDir, path)

	// Check if directory exists
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "directory not found"})
		return
	}

	// Initialize git if not already
	gitDir := filepath.Join(abs, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		if err := git.InitRepo(c.Request.Context(), abs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize git"})
			return
		}
		log.Printf("Initialized git repository at %s", abs)
	}

	// Get GitHub token and inject into URL for authentication
	remoteURL := body.RemoteURL
	gitHubToken := strings.TrimSpace(c.GetHeader("X-GitHub-Token"))
	if gitHubToken != "" {
		if authenticatedURL, err := git.InjectGitHubToken(remoteURL, gitHubToken); err == nil {
			remoteURL = authenticatedURL
			log.Printf("Injected GitHub token into remote URL")
		}
	}

	// Configure remote with authenticated URL
	if err := git.ConfigureRemote(c.Request.Context(), abs, "origin", remoteURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to configure remote"})
		return
	}

	log.Printf("Configured remote for %s: %s", abs, body.RemoteURL)

	// Fetch from remote so merge status can be checked
	// This is best-effort - don't fail if fetch fails
	branch := body.Branch
	if branch == "" {
		branch = "main"
	}
	cmd := exec.CommandContext(c.Request.Context(), "git", "fetch", "origin", branch)
	cmd.Dir = abs
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Initial fetch after configure remote failed (non-fatal): %v (output: %s)", err, string(out))
	} else {
		log.Printf("Fetched origin/%s after configuring remote", branch)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "remote configured",
		"remote":  body.RemoteURL,
		"branch":  body.Branch,
	})
}

// ContentGitSync handles POST /content/git-sync
// Body: { path: string, message: string, branch: string }
func ContentGitSync(c *gin.Context) {
	var body struct {
		Path    string `json:"path"`
		Message string `json:"message"`
		Branch  string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	path := filepath.Clean("/" + body.Path)
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	abs := filepath.Join(StateBaseDir, path)

	// Check if git repo exists
	gitDir := filepath.Join(abs, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "git repository not initialized"})
		return
	}

	// Perform git sync operations
	if err := git.SyncRepo(c.Request.Context(), abs, body.Message, body.Branch); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Synchronized git repository at %s to branch %s", abs, body.Branch)
	c.JSON(http.StatusOK, gin.H{
		"message": "synchronized successfully",
		"branch":  body.Branch,
	})
}

// ContentWrite handles POST /content/write when running in CONTENT_SERVICE_MODE
func ContentWrite(c *gin.Context) {
	var req struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ContentWrite: bind JSON failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Printf("ContentWrite: path=%q contentLen=%d encoding=%q StateBaseDir=%q", req.Path, len(req.Content), req.Encoding, StateBaseDir)

	path := filepath.Clean("/" + strings.TrimSpace(req.Path))
	if path == "/" || strings.Contains(path, "..") {
		log.Printf("ContentWrite: invalid path rejected: path=%q", path)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	abs := filepath.Join(StateBaseDir, path)
	log.Printf("ContentWrite: absolute path=%q", abs)

	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		log.Printf("ContentWrite: mkdir failed for %q: %v", filepath.Dir(abs), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create directory"})
		return
	}
	var data []byte
	if strings.EqualFold(req.Encoding, "base64") {
		b, err := base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			log.Printf("ContentWrite: base64 decode failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 content"})
			return
		}
		data = b
	} else {
		data = []byte(req.Content)
	}
	if err := os.WriteFile(abs, data, 0644); err != nil {
		log.Printf("ContentWrite: write failed for %q: %v", abs, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file"})
		return
	}
	log.Printf("ContentWrite: successfully wrote %d bytes to %q", len(data), abs)
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// ContentRead handles GET /content/file?path=
func ContentRead(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	log.Printf("ContentRead: requested path=%q StateBaseDir=%q", c.Query("path"), StateBaseDir)
	log.Printf("ContentRead: cleaned path=%q", path)

	if path == "/" || strings.Contains(path, "..") {
		log.Printf("ContentRead: invalid path rejected: path=%q", path)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	abs := filepath.Join(StateBaseDir, path)
	log.Printf("ContentRead: absolute path=%q", abs)

	b, err := os.ReadFile(abs)
	if err != nil {
		log.Printf("ContentRead: read failed for %q: %v", abs, err)
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "read failed"})
		}
		return
	}
	log.Printf("ContentRead: successfully read %d bytes from %q", len(b), abs)
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// ContentList handles GET /content/list?path=
func ContentList(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	log.Printf("ContentList: requested path=%q", c.Query("path"))
	log.Printf("ContentList: cleaned path=%q", path)
	log.Printf("ContentList: StateBaseDir=%q", StateBaseDir)

	if path == "/" || strings.Contains(path, "..") {
		log.Printf("ContentList: invalid path rejected: path=%q", path)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	abs := filepath.Join(StateBaseDir, path)
	log.Printf("ContentList: absolute path=%q", abs)

	info, err := os.Stat(abs)
	if err != nil {
		log.Printf("ContentList: stat failed for %q: %v", abs, err)
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "stat failed"})
		}
		return
	}
	if !info.IsDir() {
		// If it's a file, return single entry metadata
		c.JSON(http.StatusOK, gin.H{"items": []gin.H{{
			"name":       filepath.Base(abs),
			"path":       path,
			"isDir":      false,
			"size":       info.Size(),
			"modifiedAt": info.ModTime().UTC().Format(time.RFC3339),
		}}})
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "readdir failed"})
		return
	}
	items := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		items = append(items, gin.H{
			"name":       e.Name(),
			"path":       filepath.Join(path, e.Name()),
			"isDir":      e.IsDir(),
			"size":       info.Size(),
			"modifiedAt": info.ModTime().UTC().Format(time.RFC3339),
		})
	}
	log.Printf("ContentList: returning %d items for path=%q", len(items), path)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ContentWorkflowMetadata handles GET /content/workflow-metadata?session=
// Parses .claude/commands/*.md and .claude/agents/*.md files from active workflow
func ContentWorkflowMetadata(c *gin.Context) {
	sessionName := c.Query("session")
	if sessionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing session parameter"})
		return
	}

	log.Printf("ContentWorkflowMetadata: session=%q", sessionName)

	// Find active workflow directory (no workflow name provided, will search)
	workflowDir := findActiveWorkflowDir(sessionName, "")
	if workflowDir == "" {
		log.Printf("ContentWorkflowMetadata: no active workflow found for session=%q", sessionName)
		c.JSON(http.StatusOK, gin.H{
			"commands": []interface{}{},
			"agents":   []interface{}{},
			"config":   gin.H{"artifactsDir": "artifacts"}, // Default platform folder when no workflow
		})
		return
	}

	log.Printf("ContentWorkflowMetadata: found workflow at %q", workflowDir)

	// Parse ambient.json configuration
	ambientConfig := parseAmbientConfig(workflowDir)

	// Parse commands from .claude/commands/*.md
	commandsDir := filepath.Join(workflowDir, ".claude", "commands")
	commands := []map[string]interface{}{}

	if files, err := os.ReadDir(commandsDir); err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
				filePath := filepath.Join(commandsDir, file.Name())
				metadata := parseFrontmatter(filePath)
				commandName := strings.TrimSuffix(file.Name(), ".md")

				displayName := metadata["displayName"]
				if displayName == "" {
					displayName = commandName
				}

				// Extract short command (last segment after final dot)
				shortCommand := commandName
				if lastDot := strings.LastIndex(commandName, "."); lastDot != -1 {
					shortCommand = commandName[lastDot+1:]
				}

				commands = append(commands, map[string]interface{}{
					"id":           commandName,
					"name":         displayName,
					"description":  metadata["description"],
					"slashCommand": "/" + shortCommand,
					"icon":         metadata["icon"],
				})
			}
		}
		log.Printf("ContentWorkflowMetadata: found %d commands", len(commands))
	} else {
		log.Printf("ContentWorkflowMetadata: commands directory not found or unreadable: %v", err)
	}

	// Parse agents from .claude/agents/*.md
	agentsDir := filepath.Join(workflowDir, ".claude", "agents")
	agents := []map[string]interface{}{}

	if files, err := os.ReadDir(agentsDir); err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
				filePath := filepath.Join(agentsDir, file.Name())
				metadata := parseFrontmatter(filePath)
				agentID := strings.TrimSuffix(file.Name(), ".md")

				agents = append(agents, map[string]interface{}{
					"id":          agentID,
					"name":        metadata["name"],
					"description": metadata["description"],
					"tools":       metadata["tools"],
				})
			}
		}
		log.Printf("ContentWorkflowMetadata: found %d agents", len(agents))
	} else {
		log.Printf("ContentWorkflowMetadata: agents directory not found or unreadable: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"commands": commands,
		"agents":   agents,
		"config": gin.H{
			"name":         ambientConfig.Name,
			"description":  ambientConfig.Description,
			"systemPrompt": ambientConfig.SystemPrompt,
			"artifactsDir": ambientConfig.ArtifactsDir,
		},
	})
}

// parseFrontmatter extracts YAML frontmatter from a markdown file
func parseFrontmatter(filePath string) map[string]string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("parseFrontmatter: failed to read %q: %v", filePath, err)
		return map[string]string{}
	}

	str := string(content)
	if !strings.HasPrefix(str, "---\n") {
		return map[string]string{}
	}

	// Find end of frontmatter
	endIdx := strings.Index(str[4:], "\n---")
	if endIdx == -1 {
		return map[string]string{}
	}

	frontmatter := str[4 : 4+endIdx]
	result := map[string]string{}

	// Simple key: value parsing
	for _, line := range strings.Split(frontmatter, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			result[key] = value
		}
	}

	return result
}

// AmbientConfig represents the ambient.json configuration
type AmbientConfig struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	SystemPrompt string            `json:"systemPrompt"`
	ArtifactsDir string            `json:"artifactsDir"`
	Results      map[string]string `json:"results,omitempty"` // displayName -> glob pattern
}

// parseAmbientConfig reads and parses ambient.json from workflow directory
// Returns default config if file doesn't exist (not an error)
// For custom workflows without ambient.json, returns empty artifactsDir (root directory)
// allowing them to manage their own structure
func parseAmbientConfig(workflowDir string) *AmbientConfig {
	configPath := filepath.Join(workflowDir, ".ambient", "ambient.json")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("parseAmbientConfig: no ambient.json found at %q, using defaults", configPath)
		return &AmbientConfig{
			ArtifactsDir: "", // Empty string means root (custom workflows manage their own structure)
		}
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("parseAmbientConfig: failed to read %q: %v", configPath, err)
		return &AmbientConfig{ArtifactsDir: ""}
	}

	// Parse JSON
	var config AmbientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("parseAmbientConfig: failed to parse JSON from %q: %v", configPath, err)
		return &AmbientConfig{ArtifactsDir: ""}
	}

	log.Printf("parseAmbientConfig: loaded config: name=%q artifactsDir=%q", config.Name, config.ArtifactsDir)
	return &config
}

// ResultFile represents a workflow result file
type ResultFile struct {
	DisplayName string `json:"displayName"`
	Path        string `json:"path"` // Relative path from workspace
	Exists      bool   `json:"exists"`
	Content     string `json:"content,omitempty"`
	Error       string `json:"error,omitempty"`
}

// listArtifactsFiles lists all files in the artifacts directory
func listArtifactsFiles(artifactsDir string) []ResultFile {
	results := []ResultFile{}

	// Check if artifacts directory exists
	if _, err := os.Stat(artifactsDir); os.IsNotExist(err) {
		log.Printf("listArtifactsFiles: artifacts directory %q does not exist", artifactsDir)
		return results
	}

	// Walk the artifacts directory recursively
	err := filepath.Walk(artifactsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("listArtifactsFiles: error accessing %q: %v", path, err)
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from artifacts directory
		relPath, err := filepath.Rel(artifactsDir, path)
		if err != nil {
			log.Printf("listArtifactsFiles: failed to get relative path for %q: %v", path, err)
			return nil
		}

		// Use filename as display name
		displayName := filepath.Base(relPath)

		result := ResultFile{
			DisplayName: displayName,
			Path:        relPath,
			Exists:      true,
		}

		// Check file size before reading
		if info.Size() > MaxResultFileSize {
			result.Error = fmt.Sprintf("File too large (%d bytes, max %d)", info.Size(), MaxResultFileSize)
			results = append(results, result)
			return nil
		}

		// Read file content
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			result.Error = fmt.Sprintf("Failed to read: %v", readErr)
		} else {
			result.Content = string(content)
		}

		results = append(results, result)
		return nil
	})

	if err != nil {
		log.Printf("listArtifactsFiles: error walking artifacts directory %q: %v", artifactsDir, err)
	}

	// Sort results by path for consistent order
	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	return results
}

// ContentWorkflowResults handles GET /content/workflow-results?session=&workflow=
func ContentWorkflowResults(c *gin.Context) {
	sessionName := c.Query("session")
	if sessionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing session parameter"})
		return
	}

	// Get workflow name from query parameter (if provided from CR)
	workflowName := c.Query("workflow")
	workflowDir := findActiveWorkflowDir(sessionName, workflowName)
	if workflowDir == "" {
		// No workflow found - return files from artifacts folder at root
		workspaceBase := filepath.Join(StateBaseDir, "sessions", sessionName, "workspace")
		artifactsDir := filepath.Join(workspaceBase, "artifacts")
		log.Printf("ContentWorkflowResults: no workflow found, listing artifacts from %q", artifactsDir)
		results := listArtifactsFiles(artifactsDir)
		c.JSON(http.StatusOK, gin.H{"results": results})
		return
	}

	ambientConfig := parseAmbientConfig(workflowDir)
	if len(ambientConfig.Results) == 0 {
		c.JSON(http.StatusOK, gin.H{"results": []ResultFile{}})
		return
	}

	workspaceBase := filepath.Join(StateBaseDir, "sessions", sessionName, "workspace")
	results := []ResultFile{}

	// Sort keys to ensure consistent order (maps are unordered in Go)
	displayNames := make([]string, 0, len(ambientConfig.Results))
	for displayName := range ambientConfig.Results {
		displayNames = append(displayNames, displayName)
	}
	sort.Strings(displayNames)

	for _, displayName := range displayNames {
		pattern := ambientConfig.Results[displayName]
		matches, err := findMatchingFiles(workspaceBase, pattern)

		if err != nil {
			results = append(results, ResultFile{
				DisplayName: displayName,
				Path:        pattern,
				Exists:      false,
				Error:       fmt.Sprintf("Pattern error: %v", err),
			})
			continue
		}

		if len(matches) == 0 {
			results = append(results, ResultFile{
				DisplayName: displayName,
				Path:        pattern,
				Exists:      false,
			})
		} else {
			// Sort matches for consistent order
			sort.Strings(matches)

			for _, matchedPath := range matches {
				relPath, _ := filepath.Rel(workspaceBase, matchedPath)

				result := ResultFile{
					DisplayName: displayName,
					Path:        relPath,
					Exists:      true,
				}

				// Check file size before reading
				fileInfo, statErr := os.Stat(matchedPath)
				if statErr != nil {
					result.Error = fmt.Sprintf("Failed to stat file: %v", statErr)
					results = append(results, result)
					continue
				}

				if fileInfo.Size() > MaxResultFileSize {
					result.Error = fmt.Sprintf("File too large (%d bytes, max %d)", fileInfo.Size(), MaxResultFileSize)
					results = append(results, result)
					continue
				}

				// Read file content
				content, readErr := os.ReadFile(matchedPath)
				if readErr != nil {
					result.Error = fmt.Sprintf("Failed to read: %v", readErr)
				} else {
					result.Content = string(content)
				}

				results = append(results, result)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// findMatchingFiles finds files matching a glob pattern with ** support for recursive matching
// Returns matched files and an error if validation fails or too many matches found
func findMatchingFiles(baseDir, pattern string) ([]string, error) {
	// Validate baseDir is absolute and exists
	if !filepath.IsAbs(baseDir) {
		return nil, fmt.Errorf("baseDir must be absolute path")
	}

	baseInfo, err := os.Stat(baseDir)
	if err != nil {
		return nil, fmt.Errorf("baseDir does not exist: %w", err)
	}
	if !baseInfo.IsDir() {
		return nil, fmt.Errorf("baseDir is not a directory")
	}

	// Use doublestar for glob matching with ** support
	fsys := os.DirFS(baseDir)
	matches, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		return nil, fmt.Errorf("glob pattern error: %w", err)
	}

	// Enforce match limit to prevent resource exhaustion
	if len(matches) > MaxGlobMatches {
		log.Printf("findMatchingFiles: pattern %q matched %d files, limiting to %d", pattern, len(matches), MaxGlobMatches)
		matches = matches[:MaxGlobMatches]
	}

	// Convert relative paths to absolute paths and validate they stay within baseDir
	var absolutePaths []string
	baseDirAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve baseDir: %w", err)
	}

	for _, match := range matches {
		// Join and clean the path
		absPath := filepath.Join(baseDirAbs, match)
		absPath = filepath.Clean(absPath)

		// Security: Ensure resolved path stays within baseDir (prevent directory traversal)
		relPath, err := filepath.Rel(baseDirAbs, absPath)
		if err != nil {
			log.Printf("findMatchingFiles: failed to compute relative path for %q: %v", absPath, err)
			continue
		}

		// Check for directory traversal attempts (paths like "../" or starting with "../")
		if strings.HasPrefix(relPath, "..") {
			log.Printf("findMatchingFiles: rejected path traversal attempt: %q", absPath)
			continue
		}

		absolutePaths = append(absolutePaths, absPath)
	}

	return absolutePaths, nil
}

// findActiveWorkflowDir finds the active workflow directory for a session
// If workflowName is provided, it uses that directly; otherwise searches for it
func findActiveWorkflowDir(sessionName, workflowName string) string {
	// Workflows are stored at {StateBaseDir}/sessions/{session-name}/workspace/workflows/{workflow-name}
	// The runner creates this nested structure
	workflowsBase := filepath.Join(StateBaseDir, "sessions", sessionName, "workspace", "workflows")

	// If workflow name is provided, use it directly
	if workflowName != "" {
		workflowPath := filepath.Join(workflowsBase, workflowName)
		// Verify it exists and has either .claude or .ambient/ambient.json
		claudeDir := filepath.Join(workflowPath, ".claude")
		ambientConfig := filepath.Join(workflowPath, ".ambient", "ambient.json")

		if stat, err := os.Stat(claudeDir); err == nil && stat.IsDir() {
			return workflowPath
		}
		if stat, err := os.Stat(ambientConfig); err == nil && !stat.IsDir() {
			log.Printf("findActiveWorkflowDir: found workflow via ambient.json: %q", workflowPath)
			return workflowPath
		}
		// If direct path doesn't work, fall through to search
		log.Printf("findActiveWorkflowDir: workflow %q not found at expected path, searching...", workflowName)
	}

	// Search for workflow directory (fallback when workflowName not provided)
	entries, err := os.ReadDir(workflowsBase)
	if err != nil {
		log.Printf("findActiveWorkflowDir: failed to read workflows directory %q: %v", workflowsBase, err)
		return ""
	}

	// Find first directory that has .claude subdirectory OR .ambient/ambient.json (excluding temp clones)
	// Check for .ambient/ambient.json as fallback for temp content pods when main runner isn't running
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "default" && !strings.HasSuffix(entry.Name(), "-clone-temp") {
			workflowPath := filepath.Join(workflowsBase, entry.Name())

			// Check for .claude subdirectory (preferred, indicates active runner)
			claudeDir := filepath.Join(workflowPath, ".claude")
			if stat, err := os.Stat(claudeDir); err == nil && stat.IsDir() {
				return workflowPath
			}

			// Fallback: check for .ambient/ambient.json (works from temp content pod)
			ambientConfig := filepath.Join(workflowPath, ".ambient", "ambient.json")
			if stat, err := os.Stat(ambientConfig); err == nil && !stat.IsDir() {
				log.Printf("findActiveWorkflowDir: found workflow via ambient.json: %q", workflowPath)
				return workflowPath
			}
		}
	}

	return ""
}

// ContentGitMergeStatus handles GET /content/git-merge-status?path=&branch=
func ContentGitMergeStatus(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	branch := strings.TrimSpace(c.Query("branch"))

	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	if branch == "" {
		branch = "main"
	}

	abs := filepath.Join(StateBaseDir, path)

	// Check if git repo exists
	gitDir := filepath.Join(abs, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"canMergeClean":      true,
			"localChanges":       0,
			"remoteCommitsAhead": 0,
			"conflictingFiles":   []string{},
			"remoteBranchExists": false,
		})
		return
	}

	status, err := GitCheckMergeStatus(c.Request.Context(), abs, branch)
	if err != nil {
		log.Printf("ContentGitMergeStatus: check failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ContentGitPull handles POST /content/git-pull
// Body: { path: string, branch: string }
func ContentGitPull(c *gin.Context) {
	var body struct {
		Path   string `json:"path"`
		Branch string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	path := filepath.Clean("/" + body.Path)
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	if body.Branch == "" {
		body.Branch = "main"
	}

	abs := filepath.Join(StateBaseDir, path)

	if err := GitPullRepo(c.Request.Context(), abs, body.Branch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Pulled changes from origin/%s in %s", body.Branch, abs)
	c.JSON(http.StatusOK, gin.H{"message": "pulled successfully", "branch": body.Branch})
}

// ContentGitPushToBranch handles POST /content/git-push
// Body: { path: string, branch: string, message: string }
func ContentGitPushToBranch(c *gin.Context) {
	var body struct {
		Path    string `json:"path"`
		Branch  string `json:"branch"`
		Message string `json:"message"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	path := filepath.Clean("/" + body.Path)
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	if body.Branch == "" {
		body.Branch = "main"
	}

	if body.Message == "" {
		body.Message = "Session artifacts update"
	}

	abs := filepath.Join(StateBaseDir, path)

	if err := GitPushToRepo(c.Request.Context(), abs, body.Branch, body.Message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Pushed changes to origin/%s in %s", body.Branch, abs)
	c.JSON(http.StatusOK, gin.H{"message": "pushed successfully", "branch": body.Branch})
}

// ContentGitCreateBranch handles POST /content/git-create-branch
// Body: { path: string, branchName: string }
func ContentGitCreateBranch(c *gin.Context) {
	var body struct {
		Path       string `json:"path"`
		BranchName string `json:"branchName"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	path := filepath.Clean("/" + body.Path)
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	if body.BranchName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "branchName is required"})
		return
	}

	abs := filepath.Join(StateBaseDir, path)

	if err := GitCreateBranch(c.Request.Context(), abs, body.BranchName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Created branch %s in %s", body.BranchName, abs)
	c.JSON(http.StatusOK, gin.H{"message": "branch created", "branchName": body.BranchName})
}

// ContentGitListBranches handles GET /content/git-list-branches?path=
func ContentGitListBranches(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))

	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	abs := filepath.Join(StateBaseDir, path)

	branches, err := GitListRemoteBranches(c.Request.Context(), abs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"branches": branches})
}
