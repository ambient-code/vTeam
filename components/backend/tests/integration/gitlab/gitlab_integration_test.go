package gitlab_test

import (
	"context"
	"os"
	"testing"
	"time"

	"ambient-code-backend/git"
	"ambient-code-backend/gitlab"
	"ambient-code-backend/handlers"
	"ambient-code-backend/k8s"
	"ambient-code-backend/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestGitLabIntegrationEnd2End tests the complete GitLab integration workflow
// This test validates the full user journey from connecting GitLab to pushing code
func TestGitLabIntegrationEnd2End(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run")
	}

	// Require GitLab credentials from environment
	gitlabToken := os.Getenv("GITLAB_TEST_TOKEN")
	gitlabURL := os.Getenv("GITLAB_TEST_REPO_URL")

	if gitlabToken == "" || gitlabURL == "" {
		t.Skip("Skipping GitLab integration test: GITLAB_TEST_TOKEN and GITLAB_TEST_REPO_URL must be set")
	}

	ctx := context.Background()
	testNamespace := "vteam-backend-test"
	testUserID := "test-user-123"

	// Create fake Kubernetes client
	clientset := fake.NewSimpleClientset()

	t.Run("Phase 1: Connect GitLab Account", func(t *testing.T) {
		// Test token validation
		t.Run("Validate GitLab Token", func(t *testing.T) {
			result, err := gitlab.ValidateGitLabToken(ctx, gitlabToken, "https://gitlab.com")
			require.NoError(t, err, "Token validation should succeed")
			assert.True(t, result.Valid, "Token should be valid")
			assert.NotEmpty(t, result.Username, "Username should be populated")
			assert.NotEmpty(t, result.GitLabUserID, "GitLab user ID should be populated")
		})

		// Test token storage
		t.Run("Store GitLab Token in Kubernetes Secret", func(t *testing.T) {
			err := k8s.StoreGitLabToken(ctx, clientset, testNamespace, testUserID, gitlabToken)
			require.NoError(t, err, "Token storage should succeed")

			// Verify token stored
			secret, err := clientset.CoreV1().Secrets(testNamespace).Get(ctx, "gitlab-user-tokens", metav1.GetOptions{})
			require.NoError(t, err, "Secret should be created")
			assert.Contains(t, secret.Data, testUserID, "Secret should contain user's token")

			// Verify token can be retrieved
			retrievedToken, err := k8s.GetGitLabToken(ctx, clientset, testNamespace, testUserID)
			require.NoError(t, err, "Token retrieval should succeed")
			assert.Equal(t, gitlabToken, retrievedToken, "Retrieved token should match stored token")
		})

		// Test connection management
		t.Run("Store GitLab Connection Metadata", func(t *testing.T) {
			connMgr := gitlab.NewConnectionManager(clientset, testNamespace)

			connection, err := connMgr.StoreGitLabConnection(ctx, testUserID, gitlabToken, "https://gitlab.com")
			require.NoError(t, err, "Connection storage should succeed")
			assert.Equal(t, testUserID, connection.UserID)
			assert.NotEmpty(t, connection.Username)
			assert.Equal(t, "https://gitlab.com", connection.InstanceURL)

			// Verify connection can be retrieved
			retrievedConn, err := connMgr.GetGitLabConnection(ctx, testUserID)
			require.NoError(t, err, "Connection retrieval should succeed")
			assert.Equal(t, connection.Username, retrievedConn.Username)
			assert.Equal(t, connection.GitLabUserID, retrievedConn.GitLabUserID)
		})
	})

	t.Run("Phase 2: Repository Configuration", func(t *testing.T) {
		// Test provider detection
		t.Run("Detect GitLab Provider from URL", func(t *testing.T) {
			provider := types.DetectProvider(gitlabURL)
			assert.Equal(t, types.ProviderGitLab, provider, "Provider should be detected as GitLab")
		})

		// Test URL normalization
		t.Run("Normalize GitLab URL", func(t *testing.T) {
			normalized, err := gitlab.NormalizeGitLabURL(gitlabURL)
			require.NoError(t, err, "URL normalization should succeed")
			assert.Contains(t, normalized, "https://", "Normalized URL should use HTTPS")
			assert.Contains(t, normalized, ".git", "Normalized URL should have .git suffix")
		})

		// Test repository validation
		t.Run("Validate GitLab Repository Access", func(t *testing.T) {
			err := handlers.ValidateGitLabRepository(ctx, gitlabURL, gitlabToken)
			require.NoError(t, err, "Repository validation should succeed")
		})

		// Test repository info extraction
		t.Run("Extract Repository Information", func(t *testing.T) {
			info, err := handlers.GetRepositoryInfo(gitlabURL)
			require.NoError(t, err, "Repository info extraction should succeed")
			assert.Equal(t, types.ProviderGitLab, info.Provider)
			assert.NotEmpty(t, info.Owner, "Owner should be extracted")
			assert.NotEmpty(t, info.Repo, "Repo name should be extracted")
			assert.Equal(t, "https://gitlab.com/api/v4", info.APIURL, "API URL should be constructed correctly")
		})
	})

	t.Run("Phase 3: Git Operations", func(t *testing.T) {
		// Test token retrieval for git operations
		t.Run("Retrieve GitLab Token for Git Operations", func(t *testing.T) {
			token, err := git.GetGitLabToken(ctx, clientset, "test-project", testUserID)
			require.NoError(t, err, "Token retrieval should succeed")
			assert.Equal(t, gitlabToken, token, "Retrieved token should match")
		})

		// Test token injection
		t.Run("Inject Token into GitLab URL", func(t *testing.T) {
			authenticatedURL, err := git.InjectGitLabToken(gitlabURL, gitlabToken)
			require.NoError(t, err, "Token injection should succeed")
			assert.Contains(t, authenticatedURL, "oauth2:", "URL should contain oauth2 authentication")
			assert.NotContains(t, authenticatedURL, gitlabToken, "Raw token should not be visible in URL")
		})

		// Test branch URL construction
		t.Run("Construct GitLab Branch URL", func(t *testing.T) {
			branchURL, err := git.ConstructGitLabBranchURL(gitlabURL, "main")
			require.NoError(t, err, "Branch URL construction should succeed")
			assert.Contains(t, branchURL, "/-/tree/main", "Branch URL should have GitLab tree format")
		})
	})

	t.Run("Phase 4: Error Handling", func(t *testing.T) {
		// Test invalid token detection
		t.Run("Detect Invalid Token", func(t *testing.T) {
			invalidToken := "glpat-invalid-token-123"
			_, err := gitlab.ValidateGitLabToken(ctx, invalidToken, "https://gitlab.com")
			assert.Error(t, err, "Invalid token should fail validation")
		})

		// Test push error detection
		t.Run("Parse GitLab Push Errors", func(t *testing.T) {
			// Test 403 Forbidden
			err := git.DetectPushError(gitlabURL, "remote: HTTP Basic: Access denied. The provided password or token is incorrect or your account has 2FA enabled", "")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Insufficient permissions", "Should detect permission error")
			assert.Contains(t, err.Error(), "write_repository", "Should mention required scope")

			// Test 401 Unauthorized
			err = git.DetectPushError(gitlabURL, "fatal: Authentication failed", "")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Authentication failed", "Should detect auth error")
		})
	})

	t.Run("Phase 5: Token Security", func(t *testing.T) {
		// Test token redaction in logs
		t.Run("Redact Tokens in Log Messages", func(t *testing.T) {
			logMsg := "Cloning https://oauth2:" + gitlabToken + "@gitlab.com/owner/repo.git"
			redacted := gitlab.RedactToken(logMsg)
			assert.NotContains(t, redacted, gitlabToken, "Token should be redacted")
			assert.Contains(t, redacted, gitlab.TokenRedactionPlaceholder, "Should contain redaction placeholder")
		})

		// Test URL redaction
		t.Run("Redact URLs with Tokens", func(t *testing.T) {
			urlWithToken := "https://oauth2:" + gitlabToken + "@gitlab.com/owner/repo.git"
			redactedURL := gitlab.RedactURL(urlWithToken)
			assert.NotContains(t, redactedURL, gitlabToken, "Token should be redacted from URL")
			assert.Contains(t, redactedURL, gitlab.TokenRedactionPlaceholder, "Should contain redaction placeholder")
		})
	})

	t.Run("Phase 6: Cleanup", func(t *testing.T) {
		// Test token deletion
		t.Run("Delete GitLab Token", func(t *testing.T) {
			err := k8s.DeleteGitLabToken(ctx, clientset, testNamespace, testUserID)
			require.NoError(t, err, "Token deletion should succeed")

			// Verify token deleted
			_, err = k8s.GetGitLabToken(ctx, clientset, testNamespace, testUserID)
			assert.Error(t, err, "Token should not exist after deletion")
		})

		// Test connection deletion
		t.Run("Delete GitLab Connection", func(t *testing.T) {
			connMgr := gitlab.NewConnectionManager(clientset, testNamespace)
			err := connMgr.DeleteGitLabConnection(ctx, testUserID)
			require.NoError(t, err, "Connection deletion should succeed")

			// Verify connection deleted
			conn, err := connMgr.GetGitLabConnection(ctx, testUserID)
			assert.Error(t, err, "Connection should not exist after deletion")
			assert.Nil(t, conn, "Connection should be nil")
		})
	})
}

// TestGitLabSelfHostedIntegration tests self-hosted GitLab instance integration
func TestGitLabSelfHostedIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run")
	}

	// Require self-hosted GitLab credentials
	token := os.Getenv("GITLAB_SELFHOSTED_TOKEN")
	instanceURL := os.Getenv("GITLAB_SELFHOSTED_URL")
	repoURL := os.Getenv("GITLAB_SELFHOSTED_REPO_URL")

	if token == "" || instanceURL == "" || repoURL == "" {
		t.Skip("Skipping self-hosted GitLab test: GITLAB_SELFHOSTED_TOKEN, GITLAB_SELFHOSTED_URL, and GITLAB_SELFHOSTED_REPO_URL must be set")
	}

	ctx := context.Background()

	t.Run("Validate Self-Hosted Instance", func(t *testing.T) {
		result, err := gitlab.ValidateGitLabToken(ctx, token, instanceURL)
		require.NoError(t, err, "Self-hosted token validation should succeed")
		assert.True(t, result.Valid, "Token should be valid")
	})

	t.Run("Detect Self-Hosted Instance", func(t *testing.T) {
		parsed, err := gitlab.ParseGitLabURL(repoURL)
		require.NoError(t, err, "URL parsing should succeed")

		isSeflHosted := gitlab.IsGitLabSelfHosted(parsed.Host)
		assert.True(t, isSeflHosted, "Should detect as self-hosted instance")
	})

	t.Run("Construct Self-Hosted API URL", func(t *testing.T) {
		parsed, err := gitlab.ParseGitLabURL(repoURL)
		require.NoError(t, err)

		apiURL := gitlab.ConstructAPIURL(parsed.Host)
		assert.Contains(t, apiURL, parsed.Host, "API URL should contain instance host")
		assert.Contains(t, apiURL, "/api/v4", "API URL should have /api/v4 path")
	})
}

// TestGitLabProviderDetection tests provider detection for various URL formats
func TestGitLabProviderDetection(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected types.ProviderType
	}{
		{
			name:     "GitLab.com HTTPS",
			url:      "https://gitlab.com/owner/repo.git",
			expected: types.ProviderGitLab,
		},
		{
			name:     "GitLab.com HTTPS without .git",
			url:      "https://gitlab.com/owner/repo",
			expected: types.ProviderGitLab,
		},
		{
			name:     "GitLab.com SSH",
			url:      "git@gitlab.com:owner/repo.git",
			expected: types.ProviderGitLab,
		},
		{
			name:     "Self-hosted GitLab HTTPS",
			url:      "https://gitlab.company.com/group/project.git",
			expected: types.ProviderGitLab,
		},
		{
			name:     "Self-hosted GitLab SSH",
			url:      "git@gitlab.company.com:group/project.git",
			expected: types.ProviderGitLab,
		},
		{
			name:     "GitHub.com HTTPS",
			url:      "https://github.com/owner/repo.git",
			expected: types.ProviderGitHub,
		},
		{
			name:     "GitHub.com SSH",
			url:      "git@github.com:owner/repo.git",
			expected: types.ProviderGitHub,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detected := types.DetectProvider(tc.url)
			assert.Equal(t, tc.expected, detected, "Provider detection failed for %s", tc.url)
		})
	}
}

// TestGitLabURLNormalization tests URL normalization for various formats
func TestGitLabURLNormalization(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    string
		shouldError bool
	}{
		{
			name:     "HTTPS with .git",
			input:    "https://gitlab.com/owner/repo.git",
			expected: "https://gitlab.com/owner/repo.git",
		},
		{
			name:     "HTTPS without .git",
			input:    "https://gitlab.com/owner/repo",
			expected: "https://gitlab.com/owner/repo.git",
		},
		{
			name:     "SSH format",
			input:    "git@gitlab.com:owner/repo.git",
			expected: "https://gitlab.com/owner/repo.git",
		},
		{
			name:     "Self-hosted HTTPS",
			input:    "https://gitlab.company.com/group/project",
			expected: "https://gitlab.company.com/group/project.git",
		},
		{
			name:     "Self-hosted SSH",
			input:    "git@gitlab.company.com:group/project.git",
			expected: "https://gitlab.company.com/group/project.git",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalized, err := gitlab.NormalizeGitLabURL(tc.input)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, normalized)
			}
		})
	}
}

// TestGitLabClientLogging tests that API calls are properly logged with request IDs
func TestGitLabClientLogging(t *testing.T) {
	// Skip if no real GitLab access
	token := os.Getenv("GITLAB_TEST_TOKEN")
	if token == "" {
		t.Skip("Skipping: GITLAB_TEST_TOKEN not set")
	}

	ctx := context.Background()
	client := gitlab.NewClient("https://gitlab.com/api/v4", token)

	// Make a simple API request
	resp, err := client.GetCurrentUser(ctx)
	require.NoError(t, err, "API call should succeed")
	assert.NotNil(t, resp, "Response should not be nil")

	// Note: Actual log verification would require capturing log output
	// This test validates the happy path executes without errors
}

// BenchmarkGitLabTokenValidation benchmarks token validation performance
func BenchmarkGitLabTokenValidation(b *testing.B) {
	token := os.Getenv("GITLAB_TEST_TOKEN")
	if token == "" {
		b.Skip("Skipping: GITLAB_TEST_TOKEN not set")
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gitlab.ValidateGitLabToken(ctx, token, "https://gitlab.com")
	}
}

// BenchmarkProviderDetection benchmarks provider detection performance
func BenchmarkProviderDetection(b *testing.B) {
	url := "https://gitlab.com/owner/repo.git"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = types.DetectProvider(url)
	}
}
