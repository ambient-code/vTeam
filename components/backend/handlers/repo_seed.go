package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ambient-code-backend/git"
	"ambient-code-backend/types"
)

// SeedingStatus represents the status of repository seeding
type SeedingStatus struct {
	Required      bool     `json:"required"`
	MissingDirs   []string `json:"missingDirs,omitempty"`
	MissingFiles  []string `json:"missingFiles,omitempty"`
	InProgress    bool     `json:"inProgress"`
	LastSeeded    *string  `json:"lastSeeded,omitempty"`
	Error         string   `json:"error,omitempty"`
	CompletedAt   *string  `json:"completedAt,omitempty"`
	RepositoryURL string   `json:"repositoryUrl"`
}

// SeedRequest represents a request to seed a repository
type SeedRequest struct {
	RepositoryURL string `json:"repositoryUrl" binding:"required"`
	Branch        string `json:"branch"`
	Force         bool   `json:"force"` // Force re-seed even if structure exists
}

// SeedResponse represents the response from a seeding operation
type SeedResponse struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	SeededDirs    []string `json:"seededDirs,omitempty"`
	SeededFiles   []string `json:"seededFiles,omitempty"`
	CommitSHA     string   `json:"commitSha,omitempty"`
	Error         string   `json:"error,omitempty"`
	RepositoryURL string   `json:"repositoryUrl"`
}

// RequiredClaudeStructure defines the required .claude/ directory structure
var RequiredClaudeStructure = map[string][]string{
	".claude": {},
	".claude/commands": {
		"README.md",
	},
}

// ClaudeTemplates contains default template content for .claude/ files
var ClaudeTemplates = map[string]string{
	".claude/README.md": "# Claude Code Configuration\n\n" +
		"This directory contains configuration for Claude Code integration.\n\n" +
		"## Structure\n\n" +
		"- `commands/` - Custom slash commands for this project\n" +
		"- `settings.local.json` - Local Claude Code settings (not committed)\n\n" +
		"## Documentation\n\n" +
		"For more information, see the [Claude Code documentation](https://docs.claude.com/claude-code).\n",

	".claude/commands/README.md": "# Custom Commands\n\n" +
		"Add custom slash commands for your project here.\n\n" +
		"Each command is a markdown file that defines:\n" +
		"- Command name (from filename)\n" +
		"- Command description\n" +
		"- Prompt template\n\n" +
		"## Example\n\n" +
		"Create `analyze.md`:\n\n" +
		"```markdown\n" +
		"Analyze the codebase and provide insights about:\n" +
		"- Architecture patterns\n" +
		"- Code quality issues\n" +
		"- Potential improvements\n" +
		"```\n\n" +
		"Then use with `/analyze` in Claude Code.\n",
	".claude/settings.local.json": `{
  "permissions": {
    "allow": [],
    "deny": [],
    "ask": []
  }
}
`,
	".claude/.gitignore": `settings.local.json
*.log
`,
}

// DetectMissingStructure checks if a repository is missing required .claude/ structure
func DetectMissingStructure(ctx context.Context, repoPath string) (*SeedingStatus, error) {
	status := &SeedingStatus{
		Required:      false,
		MissingDirs:   []string{},
		MissingFiles:  []string{},
		InProgress:    false,
		RepositoryURL: "",
	}

	// Check each required directory
	for dir, files := range RequiredClaudeStructure {
		dirPath := filepath.Join(repoPath, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			status.Required = true
			status.MissingDirs = append(status.MissingDirs, dir)
		} else {
			// Directory exists, check required files
			for _, file := range files {
				filePath := filepath.Join(dirPath, file)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					status.Required = true
					status.MissingFiles = append(status.MissingFiles, filepath.Join(dir, file))
				}
			}
		}
	}

	return status, nil
}

// SeedRepository creates the .claude/ directory structure in a repository
func SeedRepository(ctx context.Context, repoPath, repoURL, branch, userEmail, userName string) (*SeedResponse, error) {
	response := &SeedResponse{
		Success:       false,
		SeededDirs:    []string{},
		SeededFiles:   []string{},
		RepositoryURL: repoURL,
	}

	// Create required directories
	for dir := range RequiredClaudeStructure {
		dirPath := filepath.Join(repoPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			response.Error = fmt.Sprintf("Failed to create directory %s: %v", dir, err)
			return response, err
		}
		response.SeededDirs = append(response.SeededDirs, dir)
	}

	// Copy template files
	for templatePath, content := range ClaudeTemplates {
		filePath := filepath.Join(repoPath, templatePath)

		// Check if file already exists
		if _, err := os.Stat(filePath); err == nil {
			// File exists, skip
			continue
		}

		// Create parent directory if needed
		parentDir := filepath.Dir(filePath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			response.Error = fmt.Sprintf("Failed to create parent directory for %s: %v", templatePath, err)
			return response, err
		}

		// Write template content
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			response.Error = fmt.Sprintf("Failed to write template file %s: %v", templatePath, err)
			return response, err
		}
		response.SeededFiles = append(response.SeededFiles, templatePath)
	}

	// Commit changes
	commitMsg := "chore: initialize .claude/ directory structure\n\nAdd Claude Code configuration for AI-assisted development.\n\n🤖 Seeded by vTeam Ambient Code Platform"

	// Configure git user if provided
	if userEmail != "" && userName != "" {
		gitConfig := exec.CommandContext(ctx, "git", "-C", repoPath, "config", "user.email", userEmail)
		if err := gitConfig.Run(); err != nil {
			response.Error = fmt.Sprintf("Failed to configure git user email: %v", err)
			return response, err
		}

		gitConfig = exec.CommandContext(ctx, "git", "-C", repoPath, "config", "user.name", userName)
		if err := gitConfig.Run(); err != nil {
			response.Error = fmt.Sprintf("Failed to configure git user name: %v", err)
			return response, err
		}
	}

	// Add files to git
	gitAdd := exec.CommandContext(ctx, "git", "-C", repoPath, "add", ".claude/")
	if err := gitAdd.Run(); err != nil {
		response.Error = fmt.Sprintf("Failed to add files to git: %v", err)
		return response, err
	}

	// Commit
	gitCommit := exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", commitMsg)
	if output, err := gitCommit.CombinedOutput(); err != nil {
		// Check if error is because there's nothing to commit
		if strings.Contains(string(output), "nothing to commit") {
			response.Message = "Claude structure already exists, nothing to seed"
			response.Success = true
			return response, nil
		}
		response.Error = fmt.Sprintf("Failed to commit changes: %v - %s", err, string(output))
		return response, err
	}

	// Get commit SHA
	gitRev := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "HEAD")
	if output, err := gitRev.Output(); err == nil {
		response.CommitSHA = strings.TrimSpace(string(output))
	}

	response.Success = true
	response.Message = fmt.Sprintf("Successfully seeded .claude/ structure with %d directories and %d files",
		len(response.SeededDirs), len(response.SeededFiles))

	return response, nil
}

// GetRepoSeedStatus handles GET /projects/:project/repo/seed-status
func GetRepoSeedStatus(c *gin.Context) {
	project := c.Param("projectName")
	repoURL := c.Query("repo")

	if repoURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo query parameter required"})
		return
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := GetK8sClientsForRequestRepo(c)

	// Detect provider
	provider := types.DetectProvider(repoURL)

	// Clone repository temporarily to check structure
	tmpDir, err := os.MkdirTemp("", "seed-check-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create temp directory: %v", err)})
		return
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Printf("Warning: failed to cleanup temp directory %s: %v", tmpDir, err)
		}
	}()

	// Get appropriate token
	var token string
	switch provider {
	case types.ProviderGitLab:
		token, err = git.GetGitLabToken(c.Request.Context(), reqK8s, project, userID.(string))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
	case types.ProviderGitHub:
		token, err = GetGitHubTokenRepo(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported repository provider"})
		return
	}

	// Clone repository
	authURL, err := git.InjectGitToken(repoURL, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to prepare repository URL: %v", err)})
		return
	}

	gitClone := exec.CommandContext(c.Request.Context(), "git", "clone", "--depth", "1", authURL, tmpDir)
	if output, err := gitClone.CombinedOutput(); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to clone repository: %v - %s", err, string(output))})
		return
	}

	// Detect missing structure
	status, err := DetectMissingStructure(c.Request.Context(), tmpDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to detect structure: %v", err)})
		return
	}

	status.RepositoryURL = repoURL
	c.JSON(http.StatusOK, status)
}

// SeedRepositoryEndpoint handles POST /projects/:project/repo/seed
func SeedRepositoryEndpoint(c *gin.Context) {
	project := c.Param("projectName")

	var req SeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	if req.Branch == "" {
		req.Branch = "main"
	}

	userID, _ := c.Get("userID")
	reqK8s, reqDyn := GetK8sClientsForRequestRepo(c)

	// Detect provider
	provider := types.DetectProvider(req.RepositoryURL)

	// Get appropriate token
	var token string
	var err error
	switch provider {
	case types.ProviderGitLab:
		token, err = git.GetGitLabToken(c.Request.Context(), reqK8s, project, userID.(string))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":       err.Error(),
				"remediation": "Connect your GitLab account via /auth/gitlab/connect",
			})
			return
		}
	case types.ProviderGitHub:
		token, err = GetGitHubTokenRepo(c.Request.Context(), reqK8s, reqDyn, project, userID.(string))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":       err.Error(),
				"remediation": "Ensure GitHub App is installed or configure GIT_TOKEN in project runner secret",
			})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported repository provider"})
		return
	}

	// Clone repository
	tmpDir, err := os.MkdirTemp("", "repo-seed-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create temp directory: %v", err)})
		return
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Printf("Warning: failed to cleanup temp directory %s: %v", tmpDir, err)
		}
	}()

	authURL, err := git.InjectGitToken(req.RepositoryURL, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to prepare repository URL: %v", err)})
		return
	}

	gitClone := exec.CommandContext(c.Request.Context(), "git", "clone", "--branch", req.Branch, authURL, tmpDir)
	if output, err := gitClone.CombinedOutput(); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":       fmt.Sprintf("Failed to clone repository: %v", err),
			"details":     string(output),
			"remediation": "Verify repository URL and branch name, ensure token has read/write access",
		})
		return
	}

	// Check if seeding is needed
	status, err := DetectMissingStructure(c.Request.Context(), tmpDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to detect structure: %v", err)})
		return
	}

	if !status.Required && !req.Force {
		c.JSON(http.StatusOK, SeedResponse{
			Success:       true,
			Message:       "Repository already has .claude/ structure, no seeding needed",
			RepositoryURL: req.RepositoryURL,
		})
		return
	}

	// Get user info for git commits (use a default if not available)
	userEmail := "ambient-bot@vteam.ambient-code"
	userName := "vTeam Ambient Bot"

	// Seed repository
	response, err := SeedRepository(c.Request.Context(), tmpDir, req.RepositoryURL, req.Branch, userEmail, userName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":       response.Error,
			"remediation": "Check repository permissions and try again",
		})
		return
	}

	// Push changes back to remote
	gitPush := exec.CommandContext(c.Request.Context(), "git", "-C", tmpDir, "push", "origin", req.Branch)
	if output, err := gitPush.CombinedOutput(); err != nil {
		// Check for permission errors
		outputStr := string(output)
		if strings.Contains(outputStr, "403") || strings.Contains(outputStr, "Permission denied") {
			remediation := "Ensure your token has write access to the repository"
			if provider == types.ProviderGitLab {
				remediation = "Ensure your GitLab PAT has 'write_repository' scope"
			}
			c.JSON(http.StatusForbidden, gin.H{
				"error":       "Failed to push changes: permission denied",
				"details":     outputStr,
				"remediation": remediation,
			})
			return
		}

		c.JSON(http.StatusBadGateway, gin.H{
			"error":       fmt.Sprintf("Failed to push changes: %v", err),
			"details":     outputStr,
			"remediation": "Check repository permissions and network connectivity",
		})
		return
	}

	// Add timestamp
	now := time.Now().Format(time.RFC3339)
	response.Success = true
	if response.Message == "" {
		response.Message = fmt.Sprintf("Successfully seeded and pushed .claude/ structure at %s", now)
	}

	c.JSON(http.StatusOK, response)
}
