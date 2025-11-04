package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"ambient-code-backend/server"
	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	// TrustedRegistries is a comma-separated list of registry patterns (e.g., "quay.io/ambient_code/*,quay.io/myorg/*")
	TrustedRegistries string
)

func init() {
	TrustedRegistries = os.Getenv("TRUSTED_REGISTRIES")
	if TrustedRegistries == "" {
		TrustedRegistries = "quay.io/ambient_code/*"
	}
}

// validateImageDigest validates that the image digest is in the correct format
func validateImageDigest(imageDigest string) error {
	// Must contain @sha256: prefix
	if !strings.Contains(imageDigest, "@sha256:") {
		return fmt.Errorf("image digest must be in format 'registry.io/org/repo@sha256:...' (not a tag)")
	}

	// Basic format check: registry.io/org/repo@sha256:hexdigest
	digestPattern := regexp.MustCompile(`^[a-zA-Z0-9._-]+(/[a-zA-Z0-9._-]+)+@sha256:[a-f0-9]{64}$`)
	if !digestPattern.MatchString(imageDigest) {
		return fmt.Errorf("invalid image digest format")
	}

	return nil
}

// validateRegistryWhitelist checks if the image digest matches a trusted registry pattern
func validateRegistryWhitelist(imageDigest string) error {
	patterns := strings.Split(TrustedRegistries, ",")
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Convert glob pattern to regex
		// quay.io/ambient_code/* -> ^quay\.io/ambient_code/[^@]+
		// quay.io/myorg/* -> ^quay\.io/myorg/[^@]+
		regexPattern := strings.ReplaceAll(pattern, ".", "\\.")
		regexPattern = strings.ReplaceAll(regexPattern, "*", "[^@]+")
		regexPattern = "^" + regexPattern

		matched, err := regexp.MatchString(regexPattern, imageDigest)
		if err != nil {
			log.Printf("Error matching registry pattern %s: %v", pattern, err)
			continue
		}

		if matched {
			return nil // Found matching pattern
		}
	}

	return fmt.Errorf("image digest does not match any trusted registry pattern. Allowed: %s", TrustedRegistries)
}

// CreateWorkflow registers a new workflow
func CreateWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	userID, _ := c.Get("userID")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User identity required"})
		return
	}

	var req types.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate image digest format
	if err := validateImageDigest(req.ImageDigest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate registry whitelist
	if err := validateRegistryWhitelist(req.ImageDigest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate graphs
	if len(req.Graphs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one graph is required"})
		return
	}

	for _, graph := range req.Graphs {
		if graph.Name == "" || graph.Entry == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "graph name and entry are required"})
			return
		}
		// Validate entry format: module:function
		if !strings.Contains(graph.Entry, ":") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "graph entry must be in format 'module:function'"})
			return
		}
	}

	// Start transaction
	tx, err := server.DB.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workflow"})
		return
	}
	defer tx.Rollback()

	// Check if workflow already exists
	var existingID string
	err = tx.QueryRow("SELECT id FROM workflows WHERE project = $1 AND name = $2", project, req.Name).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("workflow '%s' already exists", req.Name)})
		return
	}
	if err != sql.ErrNoRows {
		log.Printf("Error checking existing workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check workflow existence"})
		return
	}

	// Create workflow
	workflowID := uuid.New().String()
	_, err = tx.Exec(
		"INSERT INTO workflows (id, name, owner, project, created_at) VALUES ($1, $2, $3, $4, $5)",
		workflowID, req.Name, userIDStr, project, time.Now(),
	)
	if err != nil {
		log.Printf("Failed to insert workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workflow"})
		return
	}

	// Create initial version (v1.0.0)
	versionID := uuid.New().String()
	graphsJSON, _ := json.Marshal(req.Graphs)
	var inputsSchemaJSON []byte
	if req.InputsSchema != nil {
		inputsSchemaJSON, _ = json.Marshal(req.InputsSchema)
	}

	_, err = tx.Exec(
		"INSERT INTO workflow_versions (id, workflow_id, version, image_digest, graphs, inputs_schema, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		versionID, workflowID, "v1.0.0", req.ImageDigest, graphsJSON, inputsSchemaJSON, time.Now(),
	)
	if err != nil {
		log.Printf("Failed to insert workflow version: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workflow version"})
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workflow"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":   workflowID,
		"name": req.Name,
	})
}

// ListWorkflows lists all workflows for a project
func ListWorkflows(c *gin.Context) {
	project := c.Param("projectName")

	rows, err := server.DB.Query(
		"SELECT id, name, owner, project, created_at FROM workflows WHERE project = $1 ORDER BY created_at DESC",
		project,
	)
	if err != nil {
		log.Printf("Failed to query workflows: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list workflows"})
		return
	}
	defer rows.Close()

	workflows := []types.Workflow{}
	for rows.Next() {
		var wf types.Workflow
		if err := rows.Scan(&wf.ID, &wf.Name, &wf.Owner, &wf.Project, &wf.CreatedAt); err != nil {
			log.Printf("Error scanning workflow: %v", err)
			continue
		}
		workflows = append(workflows, wf)
	}

	c.JSON(http.StatusOK, gin.H{"workflows": workflows})
}

// GetWorkflow gets a workflow with its versions
func GetWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	name := c.Param("name")

	var wf types.Workflow
	err := server.DB.QueryRow(
		"SELECT id, name, owner, project, created_at FROM workflows WHERE project = $1 AND name = $2",
		project, name,
	).Scan(&wf.ID, &wf.Name, &wf.Owner, &wf.Project, &wf.CreatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow"})
		return
	}

	// Get versions
	versionRows, err := server.DB.Query(
		"SELECT id, workflow_id, version, image_digest, graphs, inputs_schema, created_at FROM workflow_versions WHERE workflow_id = $1 ORDER BY created_at DESC",
		wf.ID,
	)
	if err != nil {
		log.Printf("Failed to query workflow versions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow versions"})
		return
	}
	defer versionRows.Close()

	versions := []types.WorkflowVersion{}
	for versionRows.Next() {
		var v types.WorkflowVersion
		var graphsJSON, inputsSchemaJSON []byte
		if err := versionRows.Scan(&v.ID, &v.WorkflowID, &v.Version, &v.ImageDigest, &graphsJSON, &inputsSchemaJSON, &v.CreatedAt); err != nil {
			log.Printf("Error scanning workflow version: %v", err)
			continue
		}

		if err := json.Unmarshal(graphsJSON, &v.Graphs); err != nil {
			log.Printf("Error unmarshaling graphs: %v", err)
			continue
		}

		if len(inputsSchemaJSON) > 0 {
			if err := json.Unmarshal(inputsSchemaJSON, &v.InputsSchema); err != nil {
				log.Printf("Error unmarshaling inputs schema: %v", err)
			}
		}

		versions = append(versions, v)
	}

	c.JSON(http.StatusOK, gin.H{
		"workflow":  wf,
		"versions": versions,
	})
}

// CreateWorkflowVersion adds a new version to an existing workflow
func CreateWorkflowVersion(c *gin.Context) {
	project := c.Param("projectName")
	name := c.Param("name")

	var req types.CreateWorkflowVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate image digest format
	if err := validateImageDigest(req.ImageDigest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate registry whitelist
	if err := validateRegistryWhitelist(req.ImageDigest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate graphs
	if len(req.Graphs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one graph is required"})
		return
	}

	for _, graph := range req.Graphs {
		if graph.Name == "" || graph.Entry == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "graph name and entry are required"})
			return
		}
		if !strings.Contains(graph.Entry, ":") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "graph entry must be in format 'module:function'"})
			return
		}
	}

	// Get workflow ID
	var workflowID string
	err := server.DB.QueryRow(
		"SELECT id FROM workflows WHERE project = $1 AND name = $2",
		project, name,
	).Scan(&workflowID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow"})
		return
	}

	// Check if version already exists
	var existingID string
	err = server.DB.QueryRow(
		"SELECT id FROM workflow_versions WHERE workflow_id = $1 AND version = $2",
		workflowID, req.Version,
	).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("version '%s' already exists", req.Version)})
		return
	}
	if err != sql.ErrNoRows {
		log.Printf("Error checking existing version: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check version existence"})
		return
	}

	// Create version
	versionID := uuid.New().String()
	graphsJSON, _ := json.Marshal(req.Graphs)
	var inputsSchemaJSON []byte
	if req.InputsSchema != nil {
		inputsSchemaJSON, _ = json.Marshal(req.InputsSchema)
	}

	_, err = server.DB.Exec(
		"INSERT INTO workflow_versions (id, workflow_id, version, image_digest, graphs, inputs_schema, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		versionID, workflowID, req.Version, req.ImageDigest, graphsJSON, inputsSchemaJSON, time.Now(),
	)
	if err != nil {
		log.Printf("Failed to insert workflow version: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workflow version"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      versionID,
		"version": req.Version,
	})
}

// GetWorkflowVersion gets a specific workflow version
func GetWorkflowVersion(c *gin.Context) {
	project := c.Param("projectName")
	name := c.Param("name")
	version := c.Param("version")

	// Get workflow ID
	var workflowID string
	err := server.DB.QueryRow(
		"SELECT id FROM workflows WHERE project = $1 AND name = $2",
		project, name,
	).Scan(&workflowID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow"})
		return
	}

	// Get version
	var v types.WorkflowVersion
	var graphsJSON, inputsSchemaJSON []byte
	err = server.DB.QueryRow(
		"SELECT id, workflow_id, version, image_digest, graphs, inputs_schema, created_at FROM workflow_versions WHERE workflow_id = $1 AND version = $2",
		workflowID, version,
	).Scan(&v.ID, &v.WorkflowID, &v.Version, &v.ImageDigest, &graphsJSON, &inputsSchemaJSON, &v.CreatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow version not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query workflow version: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow version"})
		return
	}

	if err := json.Unmarshal(graphsJSON, &v.Graphs); err != nil {
		log.Printf("Error unmarshaling graphs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse workflow graphs"})
		return
	}

	if len(inputsSchemaJSON) > 0 {
		if err := json.Unmarshal(inputsSchemaJSON, &v.InputsSchema); err != nil {
			log.Printf("Error unmarshaling inputs schema: %v", err)
		}
	}

	c.JSON(http.StatusOK, v)
}

// DeleteWorkflow deletes a workflow and all its versions
func DeleteWorkflow(c *gin.Context) {
	project := c.Param("projectName")
	name := c.Param("name")

	// Get workflow ID
	var workflowID string
	err := server.DB.QueryRow(
		"SELECT id FROM workflows WHERE project = $1 AND name = $2",
		project, name,
	).Scan(&workflowID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}
	if err != nil {
		log.Printf("Failed to query workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow"})
		return
	}

	// Delete workflow (CASCADE will delete versions)
	_, err = server.DB.Exec("DELETE FROM workflows WHERE id = $1", workflowID)
	if err != nil {
		log.Printf("Failed to delete workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete workflow"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workflow deleted successfully"})
}

