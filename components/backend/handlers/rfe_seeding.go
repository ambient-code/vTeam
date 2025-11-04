// Package handlers provides HTTP handlers for the backend API.
// This file contains seeding operations for RFE workflows.
package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SeedRequest holds the request body for seeding an RFE workflow
type SeedRequest struct {
	AgentSourceURL    string `json:"agentSourceUrl,omitempty"`
	AgentSourceBranch string `json:"agentSourceBranch,omitempty"`
	AgentSourcePath   string `json:"agentSourcePath,omitempty"`
	SpecKitRepo       string `json:"specKitRepo,omitempty"`
	SpecKitVersion    string `json:"specKitVersion,omitempty"`
	SpecKitTemplate   string `json:"specKitTemplate,omitempty"`
}

// SeedProjectRFEWorkflow seeds the umbrella repo with spec-kit and agents via direct git operations
func SeedProjectRFEWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	// Get the workflow
	gvr := GetRFEWorkflowResource()
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil || reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	item, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), id, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		} else if errors.IsForbidden(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this workflow"})
		} else {
			log.Printf("Failed to get workflow %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve workflow"})
		}
		return
	}
	wf := RfeFromUnstructured(item)
	if wf == nil || wf.UmbrellaRepo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No spec repo configured"})
		return
	}

	// Ensure we have a branch name
	if wf.BranchName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workflow missing branch name"})
		return
	}

	// Get user ID from forwarded identity middleware
	userID, _ := c.Get("userID")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User identity required"})
		return
	}

	githubToken, err := GetGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Read request body for optional agent source and spec-kit settings
	var req SeedRequest
	_ = c.ShouldBindJSON(&req)

	// Defaults
	agentURL := req.AgentSourceURL
	if agentURL == "" {
		agentURL = "https://github.com/ambient-code/vTeam.git"
	}
	agentBranch := req.AgentSourceBranch
	if agentBranch == "" {
		agentBranch = "main"
	}
	agentPath := req.AgentSourcePath
	if agentPath == "" {
		agentPath = "agents"
	}
	// Spec-kit configuration: request body > environment variables > hardcoded defaults
	specKitRepo := req.SpecKitRepo
	if specKitRepo == "" {
		if envRepo := strings.TrimSpace(os.Getenv("SPEC_KIT_REPO")); envRepo != "" {
			specKitRepo = envRepo
		} else {
			specKitRepo = "github/spec-kit"
		}
	}
	specKitVersion := req.SpecKitVersion
	if specKitVersion == "" {
		if envVersion := strings.TrimSpace(os.Getenv("SPEC_KIT_VERSION")); envVersion != "" {
			specKitVersion = envVersion
		} else {
			specKitVersion = "main"
		}
	}
	specKitTemplate := req.SpecKitTemplate
	if specKitTemplate == "" {
		if envTemplate := strings.TrimSpace(os.Getenv("SPEC_KIT_TEMPLATE")); envTemplate != "" {
			specKitTemplate = envTemplate
		} else {
			specKitTemplate = "spec-kit-template-claude-sh"
		}
	}

	// Perform seeding operations with platform-managed branch
	branchExisted, seedErr := PerformRepoSeeding(c.Request.Context(), wf, wf.BranchName, githubToken, agentURL, agentBranch, agentPath, specKitRepo, specKitVersion, specKitTemplate)

	if seedErr != nil {
		log.Printf("Failed to seed RFE workflow %s in project %s: %v", id, project, seedErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": seedErr.Error()})
		return
	}

	message := "Repository seeded successfully"
	if branchExisted {
		message = fmt.Sprintf("Repository seeded successfully. Note: Branch '%s' already existed and will be modified by this RFE.", wf.BranchName)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "completed",
		"message":       message,
		"branchName":    wf.BranchName,
		"branchExisted": branchExisted,
	})
}

// CheckProjectRFEWorkflowSeeding checks if the umbrella repo is seeded by querying GitHub API
func CheckProjectRFEWorkflowSeeding(c *gin.Context) {
	project := c.Param("projectName")
	id := c.Param("id")

	// Get the workflow
	gvr := GetRFEWorkflowResource()
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil || reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	item, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), id, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		} else if errors.IsForbidden(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this workflow"})
		} else {
			log.Printf("Failed to get workflow %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve workflow"})
		}
		return
	}
	wf := RfeFromUnstructured(item)
	if wf == nil || wf.UmbrellaRepo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No spec repo configured"})
		return
	}

	// Get user ID from forwarded identity middleware
	userID, _ := c.Get("userID")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User identity required"})
		return
	}

	githubToken, err := GetGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if umbrella repo is seeded - use the generated feature branch, not the base branch
	branchToCheck := wf.UmbrellaRepo.Branch
	if wf.BranchName != "" {
		branchToCheck = &wf.BranchName
	}
	umbrellaSeeded, umbrellaDetails, err := CheckRepoSeeding(c.Request.Context(), wf.UmbrellaRepo.URL, branchToCheck, githubToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if all supporting repos have the feature branch
	supportingReposStatus := []map[string]interface{}{}
	allSupportingReposSeeded := true

	for _, supportingRepo := range wf.SupportingRepos {
		branchExists, err := CheckBranchExists(c.Request.Context(), supportingRepo.URL, wf.BranchName, githubToken)
		if err != nil {
			log.Printf("Warning: failed to check branch in supporting repo %s: %v", supportingRepo.URL, err)
			allSupportingReposSeeded = false
			supportingReposStatus = append(supportingReposStatus, map[string]interface{}{
				"repoURL":      supportingRepo.URL,
				"branchExists": false,
				"error":        err.Error(),
			})
			continue
		}

		if !branchExists {
			allSupportingReposSeeded = false
		}

		supportingReposStatus = append(supportingReposStatus, map[string]interface{}{
			"repoURL":      supportingRepo.URL,
			"branchExists": branchExists,
		})
	}

	// Overall seeding is complete only if umbrella repo is seeded AND all supporting repos have the branch
	isFullySeeded := umbrellaSeeded && allSupportingReposSeeded

	c.JSON(http.StatusOK, gin.H{
		"isSeeded": isFullySeeded,
		"specRepo": gin.H{
			"isSeeded": umbrellaSeeded,
			"details":  umbrellaDetails,
		},
		"supportingRepos": supportingReposStatus,
	})
}
