package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"ambient-code-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// resolveWorkspaceAbsPath normalizes a workspace-relative or absolute path to the
// absolute workspace path for a given session.
func resolveWorkspaceAbsPath(sessionName string, relOrAbs string) string {
	base := fmt.Sprintf("/sessions/%s/workspace", sessionName)
	trimmed := strings.TrimSpace(relOrAbs)
	if trimmed == "" || trimmed == "/" {
		return base
	}
	cleaned := "/" + strings.TrimLeft(trimmed, "/")
	if cleaned == base || strings.HasPrefix(cleaned, base+"/") {
		return cleaned
	}
	// Join under base for any other relative path
	return filepath.Join(base, strings.TrimPrefix(cleaned, "/"))
}

// GetSessionWorkspace lists the workspace contents for an agentic session
// Lists the contents of a session's workspace by delegating to the per-project content service
func GetSessionWorkspace(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")

	// Optional subpath within the workspace to list
	rel := strings.TrimSpace(c.Query("path"))
	absPath := resolveWorkspaceAbsPath(sessionName, rel)

	items, err := services.ListProjectContent(c, project, absPath)
	if err == nil {
		// If content/list returns exactly this file (non-dir), serve file bytes
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			b, ferr := services.ReadProjectContentFile(c, project, absPath)
			if ferr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read workspace file"})
				return
			}
			c.Data(http.StatusOK, "application/octet-stream", b)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}
	// Fallback: try file read directly
	b, ferr := services.ReadProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// GetSessionWorkspaceFile reads a specific file from the session workspace
// Reads a file from a session's workspace by delegating to the per-project content service
func GetSessionWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	pathParam := c.Param("path")

	absPath := resolveWorkspaceAbsPath(sessionName, pathParam)

	// Try directory listing first to determine type
	items, err := services.ListProjectContent(c, project, absPath)
	if err == nil {
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			// It's a file
			b, ferr := services.ReadProjectContentFile(c, project, absPath)
			if ferr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read workspace file"})
				return
			}
			c.Data(http.StatusOK, "application/octet-stream", b)
			return
		}
		// It's a directory
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}
	// Fallback to file read
	b, ferr := services.ReadProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// PutSessionWorkspaceFile writes a file to the session workspace
// Writes a file into a session's workspace via the per-project content service
func PutSessionWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	pathParam := c.Param("path")

	absPath := resolveWorkspaceAbsPath(sessionName, pathParam)

	// Read raw request body and forward as-is (treat as text/binary pass-through)
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if err := services.WriteProjectContentFile(c, project, absPath, data); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to write workspace file"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// resolveWorkflowWorkspaceAbsPath normalizes a workspace-relative or absolute path to the
// absolute workspace path for a given RFE workflow.
func resolveWorkflowWorkspaceAbsPath(workflowID string, relOrAbs string) string {
	base := fmt.Sprintf("/rfe-workflows/%s/workspace", workflowID)
	trimmed := strings.TrimSpace(relOrAbs)
	if trimmed == "" || trimmed == "/" {
		return base
	}
	cleaned := "/" + strings.TrimLeft(trimmed, "/")
	if cleaned == base || strings.HasPrefix(cleaned, base+"/") {
		return cleaned
	}
	// Join under base for any other relative path
	return filepath.Join(base, strings.TrimPrefix(cleaned, "/"))
}

// GetRFEWorkflowWorkspace lists the workspace contents for an RFE workflow
// Lists the contents of a workflow's workspace by delegating to the per-project content service
func GetRFEWorkflowWorkspace(c *gin.Context) {
	project := c.GetString("project")
	workflowID := c.Param("id")

	// Optional subpath within the workspace to list
	rel := strings.TrimSpace(c.Query("path"))
	absPath := resolveWorkflowWorkspaceAbsPath(workflowID, rel)

	items, err := services.ListProjectContent(c, project, absPath)
	if err == nil {
		// If content/list returns exactly this file (non-dir), serve file bytes
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			b, ferr := services.ReadProjectContentFile(c, project, absPath)
			if ferr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read workspace file"})
				return
			}
			c.Data(http.StatusOK, "application/octet-stream", b)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}
	// Fallback: try file read directly
	b, ferr := services.ReadProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// GetRFEWorkflowWorkspaceFile reads a specific file from the RFE workflow workspace
// Reads a file from a workflow's workspace by delegating to the per-project content service
func GetRFEWorkflowWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	workflowID := c.Param("id")
	pathParam := c.Param("path")

	absPath := resolveWorkflowWorkspaceAbsPath(workflowID, pathParam)

	// Try directory listing first to determine type
	items, err := services.ListProjectContent(c, project, absPath)
	if err == nil {
		if len(items) == 1 && strings.TrimRight(items[0].Path, "/") == absPath && !items[0].IsDir {
			// It's a file
			b, ferr := services.ReadProjectContentFile(c, project, absPath)
			if ferr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read workspace file"})
				return
			}
			c.Data(http.StatusOK, "application/octet-stream", b)
			return
		}
		// It's a directory
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}
	// Fallback to file read
	b, ferr := services.ReadProjectContentFile(c, project, absPath)
	if ferr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to access workspace"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// PutRFEWorkflowWorkspaceFile writes a file to the RFE workflow workspace
// Writes a file into a workflow's workspace via the per-project content service
func PutRFEWorkflowWorkspaceFile(c *gin.Context) {
	project := c.GetString("project")
	workflowID := c.Param("id")
	pathParam := c.Param("path")

	absPath := resolveWorkflowWorkspaceAbsPath(workflowID, pathParam)

	// Read raw request body and forward as-is (treat as text/binary pass-through)
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if err := services.WriteProjectContentFile(c, project, absPath, data); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to write workspace file"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// PublishWorkflowFileToJira publishes a workflow file to Jira and records linkage
func PublishWorkflowFileToJira(c *gin.Context) {
	// TODO: Implement Jira integration
	// For now, return not implemented
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Jira integration not implemented yet"})
}

// GetWorkflowJira gets Jira linkage information for a workflow
func GetWorkflowJira(c *gin.Context) {
	// TODO: Implement Jira integration
	// For now, return empty linkage
	c.JSON(http.StatusOK, gin.H{"jira": nil})
}
