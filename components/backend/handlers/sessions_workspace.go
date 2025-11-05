// Package handlers provides HTTP handlers for the backend API.
// This file contains workspace file operations for agentic sessions.
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListSessionWorkspace lists files and directories in a session's workspace.
// GET /api/projects/:projectName/agentic-sessions/:sessionName/workspace
func ListSessionWorkspace(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		log.Printf("ListSessionWorkspace: project is empty, session=%s", session)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	rel := strings.TrimSpace(c.Query("path"))
	// Build absolute workspace path using plain session (no url.PathEscape to match FS paths)
	absPath := "/sessions/" + session + "/workspace"
	if rel != "" {
		absPath += "/" + rel
	}

	// Call per-job service or temp service for completed sessions
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// Try temp service first (for completed sessions), then regular service
	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			// Temp service doesn't exist, use regular service
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	u := fmt.Sprintf("%s/content/list?path=%s", endpoint, url.QueryEscape(absPath))
	log.Printf("ListSessionWorkspace: project=%s session=%s endpoint=%s", project, session, endpoint)
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ListSessionWorkspace: content service request failed: %v", err)
		// Soften error to 200 with empty list so UI doesn't spam
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	// If content service returns 404, check if it's because workspace doesn't exist yet
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("ListSessionWorkspace: workspace not found (may not be created yet by runner)")
		// Return empty list instead of error for better UX during session startup
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), b)
}

// GetSessionWorkspaceFile retrieves a specific file from a session's workspace.
// GET /api/projects/:projectName/agentic-sessions/:sessionName/workspace/*path
func GetSessionWorkspaceFile(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		log.Printf("GetSessionWorkspaceFile: project is empty, session=%s", session)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	sub := strings.TrimPrefix(c.Param("path"), "/")
	absPath := "/sessions/" + session + "/workspace/" + sub
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// Try temp service first (for completed sessions), then regular service
	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	u := fmt.Sprintf("%s/content/file?path=%s", endpoint, url.QueryEscape(absPath))
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), b)
}

// PutSessionWorkspaceFile writes or updates a file in a session's workspace.
// PUT /api/projects/:projectName/agentic-sessions/:sessionName/workspace/*path
func PutSessionWorkspaceFile(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		log.Printf("PutSessionWorkspaceFile: project is empty, session=%s", session)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}
	sub := strings.TrimPrefix(c.Param("path"), "/")
	absPath := "/sessions/" + session + "/workspace/" + sub
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// Try temp service first (for completed sessions), then regular service
	serviceName := fmt.Sprintf("temp-content-%s", session)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s != nil {
		if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
			// Temp service doesn't exist, use regular service
			serviceName = fmt.Sprintf("ambient-content-%s", session)
		}
	} else {
		serviceName = fmt.Sprintf("ambient-content-%s", session)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	log.Printf("PutSessionWorkspaceFile: using service %s for session %s", serviceName, session)
	payload, _ := io.ReadAll(c.Request.Body)
	wreq := struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}{Path: absPath, Content: string(payload), Encoding: "utf8"}
	b, _ := json.Marshal(wreq)
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint+"/content/write", strings.NewReader(string(b)))
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), rb)
}
