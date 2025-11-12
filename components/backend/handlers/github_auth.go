package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Package-level variables for GitHub auth (set from main package)
var (
	K8sClient          *kubernetes.Clientset
	Namespace          string
	GithubTokenManager GithubTokenManagerInterface
)

// GithubTokenManagerInterface defines the interface for GitHub token management
type GithubTokenManagerInterface interface {
	GenerateJWT() (string, error)
}

// GitHubAppInstallation represents a GitHub App installation for a user
type GitHubAppInstallation struct {
	UserID         string    `json:"userId"`
	GitHubUserID   string    `json:"githubUserId"`
	InstallationID int64     `json:"installationId"`
	Host           string    `json:"host"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// GetInstallationID implements the interface for git package
func (g *GitHubAppInstallation) GetInstallationID() int64 {
	return g.InstallationID
}

// GetHost implements the interface for git package
func (g *GitHubAppInstallation) GetHost() string {
	return g.Host
}

// helper: resolve GitHub API base URL from host
func githubAPIBaseURL(host string) string {
	if host == "" || host == "github.com" {
		return "https://api.github.com"
	}
	// GitHub Enterprise default
	return fmt.Sprintf("https://%s/api/v3", host)
}

// doGitHubRequest executes an HTTP request to the GitHub API
func doGitHubRequest(ctx context.Context, method string, url string, authHeader string, accept string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if accept == "" {
		accept = "application/vnd.github+json"
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "vTeam-Backend")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	// Optional If-None-Match can be set by callers via context
	if v := ctx.Value("ifNoneMatch"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			req.Header.Set("If-None-Match", s)
		}
	}
	client := &http.Client{Timeout: 15 * time.Second}
	return client.Do(req)
}

// ===== OAuth during installation (user verification) =====

// signState signs a payload with HMAC SHA-256
func signState(secret string, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// HandleGitHubUserOAuthCallback handles GET /auth/github/user/callback
func HandleGitHubUserOAuthCallback(c *gin.Context) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	stateSecret := os.Getenv("GITHUB_STATE_SECRET")
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" || strings.TrimSpace(stateSecret) == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth not configured"})
		return
	}
	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}
	// Defaults when no state provided
	var retB64 string
	var instID int64
	// Validate state if present
	if state != "" {
		raw, err := base64.RawURLEncoding.DecodeString(state)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}
		parts := strings.SplitN(string(raw), ".", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}
		payload, sig := parts[0], parts[1]
		if signState(stateSecret, payload) != sig {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad state signature"})
			return
		}
		fields := strings.Split(payload, ":")
		if len(fields) != 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad state payload"})
			return
		}
		userInState := fields[0]
		ts := fields[1]
		retB64 = fields[3]
		instB64 := fields[4]
		if sec, err := strconv.ParseInt(ts, 10, 64); err == nil {
			if time.Since(time.Unix(sec, 0)) > 10*time.Minute {
				c.JSON(http.StatusBadRequest, gin.H{"error": "state expired"})
				return
			}
		}
		// Confirm current session user matches state user
		userID, _ := c.Get("userID")
		if userID == nil || userInState != userID.(string) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user mismatch"})
			return
		}
		// Decode installation id from state
		instBytes, _ := base64.RawURLEncoding.DecodeString(instB64)
		instStr := string(instBytes)
		instID, _ = strconv.ParseInt(instStr, 10, 64)
	} else {
		// No state (install started outside our UI). Require user session and read installation_id from query.
		userID, _ := c.Get("userID")
		if userID == nil || strings.TrimSpace(userID.(string)) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
			return
		}
		instStr := c.Query("installation_id")
		var err error
		instID, err = strconv.ParseInt(instStr, 10, 64)
		if err != nil || instID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid installation id"})
			return
		}
	}
	// Exchange code → user token
	token, err := exchangeOAuthCodeForUserToken(clientID, clientSecret, code)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "oauth exchange failed"})
		return
	}
	// Verify ownership: GET /user/installations includes the installation
	owns, login, err := userOwnsInstallation(token, instID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "verification failed"})
		return
	}
	if !owns {
		c.JSON(http.StatusForbidden, gin.H{"error": "installation not owned by user"})
		return
	}
	// Store mapping
	installation := GitHubAppInstallation{
		UserID:         c.GetString("userID"),
		GitHubUserID:   login,
		InstallationID: instID,
		Host:           "github.com",
		UpdatedAt:      time.Now(),
	}
	if err := storeGitHubInstallation(c.Request.Context(), "", &installation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store installation"})
		return
	}
	// Redirect back to return_to if present
	retURL := "/integrations"
	if retB64 != "" {
		if b, err := base64.RawURLEncoding.DecodeString(retB64); err == nil {
			retURL = string(b)
		}
	}
	if retURL == "" {
		retURL = "/integrations"
	}
	c.Redirect(http.StatusFound, retURL)
}

func exchangeOAuthCodeForUserToken(clientID, clientSecret, code string) (string, error) {
	reqBody := strings.NewReader(fmt.Sprintf("client_id=%s&client_secret=%s&code=%s", clientID, clientSecret, code))
	req, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", reqBody)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("empty token")
	}
	return parsed.AccessToken, nil
}

func userOwnsInstallation(userToken string, installationID int64) (bool, string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user/installations", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "token "+userToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	var data struct {
		Installations []struct {
			ID      int64 `json:"id"`
			Account struct {
				Login string `json:"login"`
			} `json:"account"`
		} `json:"installations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, "", err
	}
	for _, inst := range data.Installations {
		if inst.ID == installationID {
			return true, inst.Account.Login, nil
		}
	}
	return false, "", nil
}

// storeGitHubInstallation persists the GitHub App installation mapping
func storeGitHubInstallation(ctx context.Context, projectName string, installation *GitHubAppInstallation) error {
	if installation == nil || installation.UserID == "" {
		return fmt.Errorf("invalid installation payload")
	}
	// Cluster-scoped by server namespace; ignore projectName for storage
	const cmName = "github-app-installations"
	for i := 0; i < 3; i++ { // retry on conflict
		cm, err := K8sClient.CoreV1().ConfigMaps(Namespace).Get(ctx, cmName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// create
				cm = &corev1.ConfigMap{ObjectMeta: v1.ObjectMeta{Name: cmName, Namespace: Namespace}, Data: map[string]string{}}
				if _, cerr := K8sClient.CoreV1().ConfigMaps(Namespace).Create(ctx, cm, v1.CreateOptions{}); cerr != nil && !errors.IsAlreadyExists(cerr) {
					return fmt.Errorf("failed to create ConfigMap: %w", cerr)
				}
				// fetch again to get resourceVersion
				cm, err = K8sClient.CoreV1().ConfigMaps(Namespace).Get(ctx, cmName, v1.GetOptions{})
				if err != nil {
					return fmt.Errorf("failed to fetch ConfigMap after create: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get ConfigMap: %w", err)
			}
		}
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		b, err := json.Marshal(installation)
		if err != nil {
			return fmt.Errorf("failed to marshal installation: %w", err)
		}
		cm.Data[installation.UserID] = string(b)
		if _, uerr := K8sClient.CoreV1().ConfigMaps(Namespace).Update(ctx, cm, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue // retry
			}
			return fmt.Errorf("failed to update ConfigMap: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update ConfigMap after retries")
}

// GetGitHubInstallation retrieves GitHub App installation for a user
func GetGitHubInstallation(ctx context.Context, userID string) (*GitHubAppInstallation, error) {
	const cmName = "github-app-installations"
	cm, err := K8sClient.CoreV1().ConfigMaps(Namespace).Get(ctx, cmName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("installation not found")
		}
		return nil, fmt.Errorf("failed to read ConfigMap: %w", err)
	}
	if cm.Data == nil {
		return nil, fmt.Errorf("installation not found")
	}
	raw, ok := cm.Data[userID]
	if !ok || raw == "" {
		return nil, fmt.Errorf("installation not found")
	}
	var inst GitHubAppInstallation
	if err := json.Unmarshal([]byte(raw), &inst); err != nil {
		return nil, fmt.Errorf("failed to decode installation: %w", err)
	}
	return &inst, nil
}

// deleteGitHubInstallation removes the user mapping from ConfigMap
func deleteGitHubInstallation(ctx context.Context, userID string) error {
	const cmName = "github-app-installations"
	cm, err := K8sClient.CoreV1().ConfigMaps(Namespace).Get(ctx, cmName, v1.GetOptions{})
	if err != nil {
		return err
	}
	if cm.Data == nil {
		return nil
	}
	delete(cm.Data, userID)
	_, uerr := K8sClient.CoreV1().ConfigMaps(Namespace).Update(ctx, cm, v1.UpdateOptions{})
	return uerr
}

// ===== Global, non-project-scoped endpoints =====

// LinkGitHubInstallationGlobal handles POST /auth/github/install
// Links the current SSO user to a GitHub App installation ID.
func LinkGitHubInstallationGlobal(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil || strings.TrimSpace(userID.(string)) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
		return
	}
	var req struct {
		InstallationID int64 `json:"installationId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	installation := GitHubAppInstallation{
		UserID:         userID.(string),
		InstallationID: req.InstallationID,
		Host:           "github.com",
		UpdatedAt:      time.Now(),
	}
	// Best-effort: enrich with GitHub account login for the installation
	if GithubTokenManager != nil {
		if jwt, err := GithubTokenManager.GenerateJWT(); err == nil {
			api := githubAPIBaseURL(installation.Host)
			url := fmt.Sprintf("%s/app/installations/%d", api, req.InstallationID)
			resp, err := doGitHubRequest(c.Request.Context(), http.MethodGet, url, "Bearer "+jwt, "", nil)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					var instObj map[string]interface{}
					if err := json.NewDecoder(resp.Body).Decode(&instObj); err == nil {
						if acct, ok := instObj["account"].(map[string]interface{}); ok {
							if login, ok := acct["login"].(string); ok {
								installation.GitHubUserID = login
							}
						}
					}
				}
			}
		}
	}
	if err := storeGitHubInstallation(c.Request.Context(), "", &installation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store installation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "GitHub App installation linked successfully", "installationId": req.InstallationID})
}

// GetGitHubStatusGlobal handles GET /auth/github/status
func GetGitHubStatusGlobal(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil || strings.TrimSpace(userID.(string)) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
		return
	}
	inst, err := GetGitHubInstallation(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"installed": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"installed":      true,
		"installationId": inst.InstallationID,
		"host":           inst.Host,
		"githubUserId":   inst.GitHubUserID,
		"userId":         inst.UserID,
		"updatedAt":      inst.UpdatedAt.Format(time.RFC3339),
	})
}

// DisconnectGitHubGlobal handles POST /auth/github/disconnect
func DisconnectGitHubGlobal(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil || strings.TrimSpace(userID.(string)) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
		return
	}
	if err := deleteGitHubInstallation(c.Request.Context(), userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlink installation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "GitHub account disconnected"})
}
