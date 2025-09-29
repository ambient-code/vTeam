package handlers

import (
	"net/http"
	"strings"

	"ambient-code-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// ContentWrite handles content write requests for the content service mode
func ContentWrite(c *gin.Context) {
	var req struct {
		Path     string `json:"path" binding:"required"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate path is absolute
	if !strings.HasPrefix(req.Path, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path must be absolute (start with /)"})
		return
	}

	// Get project from middleware context or header
	project := c.GetString("project")
	if project == "" {
		// In content service mode, project might come from request context
		project = c.GetHeader("X-Project-Namespace")
		if project == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
			return
		}
	}

	// Write the content using the workspace service
	data := []byte(req.Content)
	if err := services.WriteProjectContentFile(c, project, req.Path, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write content: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Content written successfully"})
}

// ContentRead handles content read requests for the content service mode
func ContentRead(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path parameter required"})
		return
	}

	// Validate path is absolute
	if !strings.HasPrefix(path, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path must be absolute (start with /)"})
		return
	}

	// Get project from middleware context or header
	project := c.GetString("project")
	if project == "" {
		// In content service mode, project might come from request context
		project = c.GetHeader("X-Project-Namespace")
		if project == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
			return
		}
	}

	// Read the content using the workspace service
	data, err := services.ReadProjectContentFile(c, project, path)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read content: " + err.Error()})
		return
	}

	// Return the raw content
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// ContentList handles content list requests for the content service mode
func ContentList(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = "/"
	}

	// Validate path is absolute
	if !strings.HasPrefix(path, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path must be absolute (start with /)"})
		return
	}

	// Get project from middleware context or header
	project := c.GetString("project")
	if project == "" {
		// In content service mode, project might come from request context
		project = c.GetHeader("X-Project-Namespace")
		if project == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
			return
		}
	}

	// List the content using the workspace service
	items, err := services.ListProjectContent(c, project, path)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Directory not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list content: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}