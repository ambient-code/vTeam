package handlers

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ambient-code-backend/git"

	"github.com/gin-gonic/gin"
)

// StateBaseDir is the base directory for content storage
// Set by main during initialization
var StateBaseDir string

// Git operation functions - set by main package during initialization
// These are set to the actual implementations from git package
var (
	GitPushRepo    func(ctx context.Context, repoDir, commitMessage, outputRepoURL, branch, githubToken string) (string, error)
	GitAbandonRepo func(ctx context.Context, repoDir string) error
	GitDiffRepo    func(ctx context.Context, repoDir string) (*git.DiffSummary, error)
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
		c.JSON(http.StatusOK, gin.H{"added": 0, "modified": 0, "deleted": 0, "renamed": 0, "untracked": 0})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"added":     summary.Added,
		"modified":  summary.Modified,
		"deleted":   summary.Deleted,
		"renamed":   summary.Renamed,
		"untracked": summary.Untracked,
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	path := filepath.Clean("/" + strings.TrimSpace(req.Path))
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	abs := filepath.Join(StateBaseDir, path)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create directory"})
		return
	}
	var data []byte
	if strings.EqualFold(req.Encoding, "base64") {
		b, err := base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 content"})
			return
		}
		data = b
	} else {
		data = []byte(req.Content)
	}
	if err := ioutil.WriteFile(abs, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// ContentRead handles GET /content/file?path=
func ContentRead(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	abs := filepath.Join(StateBaseDir, path)
	b, err := ioutil.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "read failed"})
		}
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// ContentList handles GET /content/list?path=
func ContentList(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	if path == "/" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	abs := filepath.Join(StateBaseDir, path)
	info, err := os.Stat(abs)
	if err != nil {
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
	entries, err := ioutil.ReadDir(abs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "readdir failed"})
		return
	}
	items := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		items = append(items, gin.H{
			"name":       e.Name(),
			"path":       filepath.Join(path, e.Name()),
			"isDir":      e.IsDir(),
			"size":       e.Size(),
			"modifiedAt": e.ModTime().UTC().Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
