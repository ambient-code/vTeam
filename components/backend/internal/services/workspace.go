package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ContentListItem represents a file or directory entry
type ContentListItem struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
}

// WriteProjectContentFile writes file content to the per-namespace content service
// using the caller's Authorization token. The path must be absolute (starts with "/").
func WriteProjectContentFile(c *gin.Context, project string, absPath string, data []byte) error {
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		// Fallback to X-Forwarded-Access-Token if present
		token = c.GetHeader("X-Forwarded-Access-Token")
	}
	if !strings.HasPrefix(absPath, "/") {
		absPath = "/" + absPath
	}
	base := os.Getenv("CONTENT_SERVICE_BASE")
	if base == "" {
		base = "http://ambient-content.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, project)
	type writeReq struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	reqBody := writeReq{Path: absPath, Content: string(data), Encoding: "utf8"}
	b, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint+"/content/write", strings.NewReader(string(b)))
	if strings.TrimSpace(token) != "" {
		httpReq.Header.Set("Authorization", token)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("content write failed: status %d", resp.StatusCode)
	}
	return nil
}

// ReadProjectContentFile reads file content from the per-namespace content service
// using the caller's Authorization token. The path must be absolute (starts with "/").
func ReadProjectContentFile(c *gin.Context, project string, absPath string) ([]byte, error) {
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		// Fallback to X-Forwarded-Access-Token if present
		token = c.GetHeader("X-Forwarded-Access-Token")
	}
	if !strings.HasPrefix(absPath, "/") {
		absPath = "/" + absPath
	}
	base := os.Getenv("CONTENT_SERVICE_BASE")
	if base == "" {
		base = "http://ambient-content.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, project)
	// Normalize any accidental double slashes in path parameter
	cleanedPath := "/" + strings.TrimLeft(absPath, "/")
	u := fmt.Sprintf("%s/content/file?path=%s", endpoint, url.QueryEscape(cleanedPath))
	httpReq, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if strings.TrimSpace(token) != "" {
		httpReq.Header.Set("Authorization", token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("content read failed: status %d", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

// ListProjectContent lists directory entries from the per-namespace content service
func ListProjectContent(c *gin.Context, project string, absPath string) ([]ContentListItem, error) {
	token := c.GetHeader("Authorization")
	if !strings.HasPrefix(absPath, "/") {
		absPath = "/" + absPath
	}
	base := os.Getenv("CONTENT_SERVICE_BASE")
	if base == "" {
		base = "http://ambient-content.%s.svc:8080"
	}
	endpoint := fmt.Sprintf(base, project)
	u := fmt.Sprintf("%s/content/list?path=%s", endpoint, url.QueryEscape(absPath))
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list failed: status %d", resp.StatusCode)
	}
	var out struct {
		Items []ContentListItem `json:"items"`
	}
	b, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}