package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"ambient-code-backend/internal/middleware"
	"ambient-code-backend/internal/services"
	"ambient-code-backend/internal/types"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListProjectRFEWorkflows lists all RFE workflows for a project
func ListProjectRFEWorkflows(c *gin.Context) {
	project := c.Param("projectName")
	var workflows []types.RFEWorkflow
	// Prefer CRD list with request-scoped client; fallback to file scan if unavailable or fails
	gvr := types.GetRFEWorkflowResource()
	_, reqDyn := middleware.GetK8sClientsForRequest(c)
	if reqDyn != nil {
		if list, err := reqDyn.Resource(gvr).Namespace(project).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("project=%s", project)}); err == nil {
			for _, item := range list.Items {
				wf := services.RFEFromUnstructured(&item)
				if wf == nil {
					continue
				}
				workflows = append(workflows, *wf)
			}
		}
	}
	if workflows == nil {
		workflows = []types.RFEWorkflow{}
	}
	// Return slim summaries: omit artifacts/agentSessions/phaseResults/status/currentPhase
	summaries := make([]map[string]interface{}, 0, len(workflows))
	for _, w := range workflows {
		item := map[string]interface{}{
			"id":            w.ID,
			"title":         w.Title,
			"description":   w.Description,
			"project":       w.Project,
			"workspacePath": w.WorkspacePath,
			"createdAt":     w.CreatedAt,
			"updatedAt":     w.UpdatedAt,
		}
		if len(w.Repositories) > 0 {
			repos := make([]map[string]interface{}, 0, len(w.Repositories))
			for _, r := range w.Repositories {
				rm := map[string]interface{}{"url": r.URL}
				if r.Branch != nil {
					rm["branch"] = *r.Branch
				}
				if r.ClonePath != nil {
					rm["clonePath"] = *r.ClonePath
				}
				repos = append(repos, rm)
			}
			item["repositories"] = repos
		}
		summaries = append(summaries, item)
	}
	c.JSON(http.StatusOK, gin.H{"workflows": summaries})
}

// CreateProjectRFEWorkflow creates a new RFE workflow
func CreateProjectRFEWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	var req types.CreateRFEWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed: " + err.Error()})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	workflowID := fmt.Sprintf("rfe-%d", time.Now().Unix())
	workflow := &types.RFEWorkflow{
		ID:            workflowID,
		Title:         req.Title,
		Description:   req.Description,
		Repositories:  req.Repositories,
		WorkspacePath: req.WorkspacePath,
		Project:       project,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_, reqDyn := middleware.GetK8sClientsForRequest(c)
	if err := services.UpsertProjectRFEWorkflowCR(reqDyn, workflow); err != nil {
		log.Printf("⚠️ Failed to upsert RFEWorkflow CR: %v", err)
	}

	// Initialize workspace structure and clone repositories
	workspaceRoot := resolveWorkflowWorkspaceAbsPath(workflowID, "")

	// Initialize Spec Kit template into workspace (version via SPEC_KIT_VERSION)
	if err := initSpecKitInWorkspace(c, project, workspaceRoot); err != nil {
		log.Printf("spec-kit init failed for %s/%s: %v", project, workflowID, err)
	}

	// Clone repositories into workspace
	for _, r := range workflow.Repositories {
		if err := cloneRepositoryToWorkspace(c, project, r, workspaceRoot); err != nil {
			log.Printf("repo clone failed for %s: %v", r.URL, err)
			// Continue with other repositories even if one fails
		}
	}

	c.JSON(http.StatusCreated, workflow)
}

// GetProjectRFEWorkflow gets a specific RFE workflow
func GetProjectRFEWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")
	// Try CRD with request-scoped client first
	gvr := types.GetRFEWorkflowResource()
	_, reqDyn := middleware.GetK8sClientsForRequest(c)
	if reqDyn != nil {
		obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), id, v1.GetOptions{})
		if err == nil {
			workflow := services.RFEFromUnstructured(obj)
			if workflow != nil {
				c.JSON(http.StatusOK, workflow)
				return
			}
		} else if !errors.IsNotFound(err) {
			log.Printf("Failed to get RFEWorkflow %s/%s: %v", project, id, err)
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "RFE workflow not found"})
}

// GetProjectRFEWorkflowSummary gets a summary of an RFE workflow
func GetProjectRFEWorkflowSummary(c *gin.Context) {
	// For now, just return the same as GetProjectRFEWorkflow
	// TODO: Implement actual summary logic
	GetProjectRFEWorkflow(c)
}

// DeleteProjectRFEWorkflow deletes an RFE workflow
func DeleteProjectRFEWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	gvr := types.GetRFEWorkflowResource()
	_, reqDyn := middleware.GetK8sClientsForRequest(c)
	if reqDyn != nil {
		err := reqDyn.Resource(gvr).Namespace(project).Delete(context.TODO(), id, v1.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "RFE workflow not found"})
				return
			}
			log.Printf("Failed to delete RFEWorkflow %s/%s: %v", project, id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete RFE workflow"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "RFE workflow deleted successfully"})
}

// ListProjectRFEWorkflowSessions lists all sessions linked to an RFE workflow by label selector
func ListProjectRFEWorkflowSessions(c *gin.Context) {
	project := c.Param("projectName")
	workflowId := c.Param("id")

	gvr := types.GetAgenticSessionV1Alpha1Resource()
	selector := fmt.Sprintf("rfe-workflow=%s,project=%s", workflowId, project)
	_, reqDyn := middleware.GetK8sClientsForRequest(c)
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}

	list, err := reqDyn.Resource(gvr).Namespace(project).List(context.TODO(), v1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Printf("Failed to list sessions for workflow %s/%s: %v", project, workflowId, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions", "details": err.Error()})
		return
	}

	// Return full session objects for UI
	sessions := make([]map[string]interface{}, 0, len(list.Items))
	for _, item := range list.Items {
		sessions = append(sessions, item.Object)
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// AddProjectRFEWorkflowSession adds a session to an RFE workflow by adding labels
func AddProjectRFEWorkflowSession(c *gin.Context) {
	project := c.Param("projectName")
	workflowId := c.Param("id")

	var req struct {
		SessionName string `json:"sessionName" binding:"required"`
		Phase       string `json:"phase,omitempty"` // Optional RFE phase
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	gvr := types.GetAgenticSessionV1Alpha1Resource()
	_, reqDyn := middleware.GetK8sClientsForRequest(c)
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}

	// Get the session to add labels to
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), req.SessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to fetch session %s/%s: %v", project, req.SessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch session", "details": err.Error()})
		return
	}

	// Add labels to link session to workflow
	meta, _ := obj.Object["metadata"].(map[string]interface{})
	labels, _ := meta["labels"].(map[string]interface{})
	if labels == nil {
		labels = map[string]interface{}{}
		meta["labels"] = labels
	}
	labels["project"] = project
	labels["rfe-workflow"] = workflowId
	if req.Phase != "" {
		labels["rfe-phase"] = req.Phase
	}

	// Inherit git repositories from the RFE workflow
	rfeGvr := types.GetRFEWorkflowResource()
	rfeObj, err := reqDyn.Resource(rfeGvr).Namespace(project).Get(context.TODO(), workflowId, v1.GetOptions{})
	if err != nil {
		log.Printf("Warning: Failed to fetch RFE workflow %s/%s for git config inheritance: %v", project, workflowId, err)
	} else {
		// Extract repositories from RFE workflow spec
		if rfeSpec, ok := rfeObj.Object["spec"].(map[string]interface{}); ok {
			if rfeRepos, ok := rfeSpec["repositories"].([]interface{}); ok && len(rfeRepos) > 0 {
				// Ensure session has a spec section
				if obj.Object["spec"] == nil {
					obj.Object["spec"] = map[string]interface{}{}
				}
				sessionSpec := obj.Object["spec"].(map[string]interface{})

				// Create or update gitConfig
				gitConfig := map[string]interface{}{}
				if existingGitConfig, ok := sessionSpec["gitConfig"].(map[string]interface{}); ok {
					gitConfig = existingGitConfig
				}

				// Convert RFE repositories to session gitConfig format
				sessionRepos := make([]map[string]interface{}, len(rfeRepos))
				for i, repo := range rfeRepos {
					if repoMap, ok := repo.(map[string]interface{}); ok {
						sessionRepo := map[string]interface{}{}
						if url, ok := repoMap["url"].(string); ok {
							sessionRepo["url"] = url
						}
						if branch, ok := repoMap["branch"].(string); ok {
							sessionRepo["branch"] = branch
						} else {
							sessionRepo["branch"] = "main" // default branch
						}
						if clonePath, ok := repoMap["clonePath"].(string); ok {
							sessionRepo["clonePath"] = clonePath
						}
						sessionRepos[i] = sessionRepo
					}
				}

				gitConfig["repositories"] = sessionRepos
				sessionSpec["gitConfig"] = gitConfig
				log.Printf("Inherited %d git repositories from RFE workflow %s to session %s", len(sessionRepos), workflowId, req.SessionName)
			}
		}
	}

	// Update the session resource
	_, err = reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update session labels %s/%s: %v", project, req.SessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session labels", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session linked to RFE workflow successfully", "session": req.SessionName, "rfe": workflowId})
}

// RemoveProjectRFEWorkflowSession removes a session from an RFE workflow by removing labels
func RemoveProjectRFEWorkflowSession(c *gin.Context) {
	project := c.Param("projectName")
	workflowId := c.Param("id")
	sessionName := c.Param("sessionName")

	gvr := types.GetAgenticSessionV1Alpha1Resource()
	_, reqDyn := middleware.GetK8sClientsForRequest(c)
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid user token"})
		return
	}

	// Get the session to remove labels from
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to fetch session %s/%s: %v", project, sessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch session", "details": err.Error()})
		return
	}

	// Remove RFE workflow labels
	meta, _ := obj.Object["metadata"].(map[string]interface{})
	labels, _ := meta["labels"].(map[string]interface{})
	if labels != nil {
		delete(labels, "rfe-workflow")
		delete(labels, "rfe-phase")
	}

	// Update the session resource
	_, err = reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update session labels %s/%s: %v", project, sessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session labels", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session unlinked from RFE workflow successfully", "session": sessionName, "rfe": workflowId})
}

// cloneRepositoryToWorkspace clones a repository into the workflow workspace
func cloneRepositoryToWorkspace(c *gin.Context, project string, repo types.GitRepository, workspaceRoot string) error {
	// Determine target directory
	targetDir := ""
	if repo.ClonePath != nil && strings.TrimSpace(*repo.ClonePath) != "" {
		targetDir = *repo.ClonePath
	} else {
		name := filepath.Base(strings.TrimSuffix(strings.TrimSuffix(repo.URL, ".git"), "/"))
		targetDir = filepath.Join("repos", name)
	}
	absTarget := filepath.Join(workspaceRoot, targetDir)

	// Ensure target directory exists in content service
	if err := services.WriteProjectContentFile(c, project, filepath.Join(absTarget, ".keep"), []byte("")); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Perform shallow clone to a temp dir on backend container filesystem
	tmpDir, err := os.MkdirTemp("", "clone-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use git CLI for shallow clone
	args := []string{"clone", "--depth", "1"}
	if repo.Branch != nil && strings.TrimSpace(*repo.Branch) != "" {
		args = append(args, "--branch", strings.TrimSpace(*repo.Branch))
	}
	args = append(args, repo.URL, tmpDir)

	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %v, output: %s", err, string(out))
	}

	// Walk cloned files and write each to content service (skip .git directory)
	return filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		rel, err := filepath.Rel(tmpDir, path)
		if err != nil {
			return nil
		}

		unixRel := strings.ReplaceAll(rel, "\\", "/")
		// Skip git metadata and root
		if unixRel == "." || strings.HasPrefix(unixRel, ".git/") || unixRel == ".git" {
			return nil
		}

		if d.IsDir() {
			// Ensure directory exists by placing a marker (harmless if overwritten later)
			_ = services.WriteProjectContentFile(c, project, filepath.Join(absTarget, unixRel, ".keep"), []byte(""))
			return nil
		}

		// File: read and write
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Continue on read errors
		}

		if err := services.WriteProjectContentFile(c, project, filepath.Join(absTarget, unixRel), data); err != nil {
			log.Printf("repo write failed: %s -> %s: %v", path, filepath.Join(absTarget, unixRel), err)
		}

		return nil
	})
}

// initSpecKitInWorkspace downloads a Spec Kit template zip and writes its contents into the workflow workspace
// SPEC_KIT_VERSION env var controls version tag (e.g., v0.0.50). Template assumed: spec-kit-template-claude-sh-<ver>.zip
func initSpecKitInWorkspace(c *gin.Context, project, workspaceRoot string) error {
	version := strings.TrimSpace(os.Getenv("SPEC_KIT_VERSION"))
	if version == "" {
		version = "v0.0.50"
	}
	tmplName := strings.TrimSpace(os.Getenv("SPEC_KIT_TEMPLATE_NAME"))
	if tmplName == "" {
		tmplName = "spec-kit-template-claude-sh"
	}
	url := fmt.Sprintf("https://github.com/github/spec-kit/releases/download/%s/%s-%s.zip", version, tmplName, version)

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download spec-kit template failed: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	// Extract files
	total := len(zr.File)
	var filesWritten, skippedDirs, openErrors, readErrors, writeErrors int
	log.Printf("initSpecKitInWorkspace: extracting spec-kit template: %d entries", total)
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			skippedDirs++
			log.Printf("spec-kit: skipping directory: %s", f.Name)
			continue
		}
		rc, err := f.Open()
		if err != nil {
			openErrors++
			log.Printf("spec-kit: open failed: %s: %v", f.Name, err)
			continue
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			readErrors++
			log.Printf("spec-kit: read failed: %s: %v", f.Name, err)
			continue
		}
		// Normalize path: keep leading dots intact; only trim explicit "./" prefix
		rel := f.Name
		origRel := rel
		rel = strings.TrimPrefix(rel, "./")
		// Ensure we do not write outside workspace
		rel = strings.ReplaceAll(rel, "\\", "/")
		for strings.Contains(rel, "../") {
			rel = strings.ReplaceAll(rel, "../", "")
		}
		if rel != origRel {
			log.Printf("spec-kit: normalized path %q -> %q", origRel, rel)
		}
		target := filepath.Join(workspaceRoot, rel)
		if err := services.WriteProjectContentFile(c, project, target, b); err != nil {
			writeErrors++
			log.Printf("write spec-kit file failed: %s: %v", target, err)
		} else {
			filesWritten++
			log.Printf("spec-kit: wrote %s (%d bytes)", target, len(b))
		}
	}
	log.Printf("initSpecKitInWorkspace: extraction summary: written=%d, skipped_dirs=%d, open_errors=%d, read_errors=%d, write_errors=%d", filesWritten, skippedDirs, openErrors, readErrors, writeErrors)
	return nil
}