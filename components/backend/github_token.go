package main

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GitHubTokenManager manages GitHub App installation tokens
type GitHubTokenManager struct {
	AppID      string
	PrivateKey *rsa.PrivateKey
	cacheMu    *sync.Mutex
	cache      map[int64]cachedInstallationToken
}

type cachedInstallationToken struct {
	token     string
	expiresAt time.Time
}

// NewGitHubTokenManager creates a new token manager
func NewGitHubTokenManager() (*GitHubTokenManager, error) {
	appID := os.Getenv("GITHUB_APP_ID")
	if appID == "" {
		// Return nil if GitHub App is not configured
		return nil, nil
	}

	// Require private key via env var GITHUB_PRIVATE_KEY (raw PEM or base64-encoded)
	raw := strings.TrimSpace(os.Getenv("GITHUB_PRIVATE_KEY"))
	if raw == "" {
		return nil, fmt.Errorf("GITHUB_PRIVATE_KEY not set")
	}
	// Support both raw PEM and base64-encoded PEM
	pemBytes := []byte(raw)
	if !strings.Contains(raw, "-----BEGIN") {
		decoded, decErr := base64.StdEncoding.DecodeString(raw)
		if decErr != nil {
			return nil, fmt.Errorf("failed to base64-decode GITHUB_PRIVATE_KEY: %w", decErr)
		}
		pemBytes = decoded
	}
	privateKey, perr := parsePrivateKeyPEM(pemBytes)
	if perr != nil {
		return nil, fmt.Errorf("failed to parse GITHUB_PRIVATE_KEY: %w", perr)
	}

	return &GitHubTokenManager{
		AppID:      appID,
		PrivateKey: privateKey,
		cacheMu:    &sync.Mutex{},
		cache:      map[int64]cachedInstallationToken{},
	}, nil
}

// loadPrivateKey loads the RSA private key from a PEM file
func parsePrivateKeyPEM(keyData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		var ok bool
		key, ok = keyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
	}

	return key, nil
}

// loadPrivateKey loads the RSA private key from a PEM file
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parsePrivateKeyPEM(keyData)
}

// GenerateJWT generates a JWT for GitHub App authentication
func (m *GitHubTokenManager) GenerateJWT() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(),
		"iss": m.AppID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.PrivateKey)
}

// MintInstallationToken creates a short-lived installation access token
func (m *GitHubTokenManager) MintInstallationToken(ctx context.Context, installationID int64) (string, time.Time, error) {
	if m == nil {
		return "", time.Time{}, fmt.Errorf("GitHub App not configured")
	}
	return m.MintInstallationTokenForHost(ctx, installationID, "github.com")
}

// MintInstallationTokenForHost mints an installation token against the specified GitHub API host
func (m *GitHubTokenManager) MintInstallationTokenForHost(ctx context.Context, installationID int64, host string) (string, time.Time, error) {
	if m == nil {
		return "", time.Time{}, fmt.Errorf("GitHub App not configured")
	}
	// Serve from cache if still valid (>3 minutes left)
	m.cacheMu.Lock()
	if entry, ok := m.cache[installationID]; ok {
		if time.Until(entry.expiresAt) > 3*time.Minute {
			token := entry.token
			exp := entry.expiresAt
			m.cacheMu.Unlock()
			return token, exp, nil
		}
	}
	m.cacheMu.Unlock()

	jwtToken, err := m.GenerateJWT()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate JWT: %w", err)
	}

	apiBase := githubAPIBaseURLLocal(host)
	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", apiBase, installationID)
	reqBody := bytes.NewBuffer([]byte("{}"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reqBody)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "vTeam-Backend")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to call GitHub: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", time.Time{}, fmt.Errorf("GitHub token mint failed: %s", string(body))
	}
	var parsed struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse token response: %w", err)
	}
	m.cacheMu.Lock()
	m.cache[installationID] = cachedInstallationToken{token: parsed.Token, expiresAt: parsed.ExpiresAt}
	m.cacheMu.Unlock()
	return parsed.Token, parsed.ExpiresAt, nil
}

// ValidateInstallationAccess checks if the installation has access to a repository
func (m *GitHubTokenManager) ValidateInstallationAccess(ctx context.Context, installationID int64, repo string) error {
	if m == nil {
		return fmt.Errorf("GitHub App not configured")
	}
	// Mint installation token (default host github.com)
	token, _, err := m.MintInstallationTokenForHost(ctx, installationID, "github.com")
	if err != nil {
		return fmt.Errorf("failed to mint installation token: %w", err)
	}

	// repo should be in form "owner/repo"; tolerate full URL and trim
	ownerRepo := repo
	if strings.HasPrefix(ownerRepo, "http://") || strings.HasPrefix(ownerRepo, "https://") {
		// Trim protocol and host
		// Examples: https://github.com/owner/repo(.git)?
		// Split by "/" and take last two segments
		parts := strings.Split(strings.TrimSuffix(ownerRepo, ".git"), "/")
		if len(parts) >= 2 {
			ownerRepo = parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
	}
	parts := strings.Split(ownerRepo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: expected owner/repo")
	}
	owner := parts[0]
	name := parts[1]

	apiBase := githubAPIBaseURLLocal("github.com")
	url := fmt.Sprintf("%s/repos/%s/%s", apiBase, owner, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "vTeam-Backend")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("installation does not have access to repository or repo not found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("unexpected GitHub response: %s", string(body))
	}
	return nil
}

// MintSessionToken creates a GitHub access token for a session
// Returns the token and expiry time to be injected as a Kubernetes Secret
func MintSessionToken(ctx context.Context, userID string) (string, time.Time, error) {
	if githubTokenManager == nil {
		return "", time.Time{}, fmt.Errorf("GitHub App not configured")
	}

	// Get user's GitHub installation
	installation, err := getGitHubInstallation(ctx, userID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get GitHub installation: %w", err)
	}

	// Mint short-lived token for the installation's host
	token, expiresAt, err := githubTokenManager.MintInstallationTokenForHost(ctx, installation.InstallationID, installation.Host)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to mint installation token: %w", err)
	}

	return token, expiresAt, nil
}

// local helper (dup of backend file) to avoid cross-file deps
func githubAPIBaseURLLocal(host string) string {
	if host == "" || host == "github.com" {
		return "https://api.github.com"
	}
	return fmt.Sprintf("https://%s/api/v3", host)
}
