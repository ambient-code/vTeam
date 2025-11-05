# GitLab Integration Tests

This directory contains end-to-end integration tests for the GitLab integration functionality.

## Overview

The integration tests validate the complete GitLab workflow:
1. Token validation and storage
2. Connection management
3. Repository configuration and validation
4. Git operations (clone, push)
5. Error handling
6. Token security (redaction)
7. Self-hosted GitLab support

## Prerequisites

### Required Environment Variables

**For GitLab.com Tests**:
```bash
export INTEGRATION_TESTS=true
export GITLAB_TEST_TOKEN="glpat-your-token-here"
export GITLAB_TEST_REPO_URL="https://gitlab.com/yourusername/test-repo.git"
```

**For Self-Hosted GitLab Tests** (optional):
```bash
export GITLAB_SELFHOSTED_TOKEN="glpat-your-selfhosted-token"
export GITLAB_SELFHOSTED_URL="https://gitlab.company.com"
export GITLAB_SELFHOSTED_REPO_URL="https://gitlab.company.com/group/project.git"
```

### GitLab Setup

1. **Create Test Repository**:
   - Public or private repository on GitLab.com
   - You must have Developer+ access (to test push operations)

2. **Create Personal Access Token**:
   - Required scopes: `api`, `read_api`, `read_user`, `write_repository`
   - See: [GitLab PAT Setup Guide](../../../docs/gitlab-token-setup.md)

## Running Tests

### Run All Integration Tests

```bash
cd components/backend

# Set environment variables
export INTEGRATION_TESTS=true
export GITLAB_TEST_TOKEN="glpat-..."
export GITLAB_TEST_REPO_URL="https://gitlab.com/user/repo.git"

# Run tests
go test -v ./tests/integration/gitlab/...
```

### Run Specific Test

```bash
# Run only end-to-end test
go test -v ./tests/integration/gitlab -run TestGitLabIntegrationEnd2End

# Run only self-hosted tests
go test -v ./tests/integration/gitlab -run TestGitLabSelfHostedIntegration

# Run only provider detection tests (no GitLab access needed)
go test -v ./tests/integration/gitlab -run TestGitLabProviderDetection
```

### Run with Verbose Output

```bash
go test -v -count=1 ./tests/integration/gitlab/...
```

### Skip Integration Tests

Integration tests are automatically skipped unless `INTEGRATION_TESTS=true` is set:

```bash
# This will skip integration tests
go test ./tests/integration/gitlab/...
```

## Test Coverage

### TestGitLabIntegrationEnd2End

**Phases**:
1. **Phase 1: Connect GitLab Account**
   - Token validation
   - Token storage in Kubernetes Secret
   - Connection metadata storage

2. **Phase 2: Repository Configuration**
   - Provider detection
   - URL normalization
   - Repository validation
   - Repository info extraction

3. **Phase 3: Git Operations**
   - Token retrieval for git operations
   - Token injection into URLs
   - Branch URL construction

4. **Phase 4: Error Handling**
   - Invalid token detection
   - Push error parsing and user-friendly messages

5. **Phase 5: Token Security**
   - Token redaction in logs
   - URL redaction

6. **Phase 6: Cleanup**
   - Token deletion
   - Connection deletion

### TestGitLabSelfHostedIntegration

Tests self-hosted GitLab functionality:
- Instance validation
- Self-hosted detection
- API URL construction for custom domains

### TestGitLabProviderDetection

Tests provider detection for various URL formats (no GitLab access required):
- GitLab.com HTTPS and SSH URLs
- Self-hosted HTTPS and SSH URLs
- GitHub URLs (to verify no false positives)

### TestGitLabURLNormalization

Tests URL normalization (no GitLab access required):
- HTTPS URLs with/without .git suffix
- SSH to HTTPS conversion
- Self-hosted URL handling

### Benchmarks

- `BenchmarkGitLabTokenValidation`: Token validation performance
- `BenchmarkProviderDetection`: Provider detection performance

## Expected Results

### Success Criteria

All tests should pass with valid GitLab credentials:

```
=== RUN   TestGitLabIntegrationEnd2End
=== RUN   TestGitLabIntegrationEnd2End/Phase_1:_Connect_GitLab_Account
=== RUN   TestGitLabIntegrationEnd2End/Phase_1:_Connect_GitLab_Account/Validate_GitLab_Token
=== RUN   TestGitLabIntegrationEnd2End/Phase_1:_Connect_GitLab_Account/Store_GitLab_Token_in_Kubernetes_Secret
=== RUN   TestGitLabIntegrationEnd2End/Phase_1:_Connect_GitLab_Account/Store_GitLab_Connection_Metadata
=== RUN   TestGitLabIntegrationEnd2End/Phase_2:_Repository_Configuration
... (more tests)
--- PASS: TestGitLabIntegrationEnd2End (2.34s)
PASS
```

### Performance Expectations

- Token validation: < 200ms (per SC-002 from spec)
- Provider detection: < 1ms
- URL normalization: < 1ms

Run benchmarks to verify:
```bash
go test -bench=. ./tests/integration/gitlab/
```

## Troubleshooting

### "Skipping integration test"

**Cause**: `INTEGRATION_TESTS` environment variable not set

**Solution**:
```bash
export INTEGRATION_TESTS=true
```

### "GITLAB_TEST_TOKEN and GITLAB_TEST_REPO_URL must be set"

**Cause**: Required environment variables missing

**Solution**:
```bash
export GITLAB_TEST_TOKEN="glpat-your-token-here"
export GITLAB_TEST_REPO_URL="https://gitlab.com/user/repo.git"
```

### "Token validation should succeed" fails

**Possible Causes**:
1. Token expired
2. Token invalid or revoked
3. Token missing required scopes
4. Network connectivity issues

**Debug**:
```bash
# Test token manually
curl -H "Authorization: Bearer $GITLAB_TEST_TOKEN" \
  https://gitlab.com/api/v4/user
```

### "Repository validation should succeed" fails

**Possible Causes**:
1. Repository URL incorrect
2. Repository doesn't exist
3. You don't have access to repository
4. Token lacks `write_repository` scope

**Debug**:
```bash
# Test repository access manually
curl -H "Authorization: Bearer $GITLAB_TEST_TOKEN" \
  "https://gitlab.com/api/v4/projects/$(echo $GITLAB_TEST_REPO_URL | sed 's|https://gitlab.com/||' | sed 's|.git$||' | sed 's|/|%2F|')"
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  gitlab-integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run GitLab Integration Tests
        env:
          INTEGRATION_TESTS: true
          GITLAB_TEST_TOKEN: ${{ secrets.GITLAB_TEST_TOKEN }}
          GITLAB_TEST_REPO_URL: ${{ secrets.GITLAB_TEST_REPO_URL }}
        run: |
          cd components/backend
          go test -v ./tests/integration/gitlab/...
```

### GitLab CI Example

```yaml
integration-tests:
  stage: test
  image: golang:1.24
  variables:
    INTEGRATION_TESTS: "true"
    GITLAB_TEST_TOKEN: $GITLAB_TEST_TOKEN
    GITLAB_TEST_REPO_URL: $GITLAB_TEST_REPO_URL
  script:
    - cd components/backend
    - go test -v ./tests/integration/gitlab/...
```

## Security Notes

### Token Safety

- **Never commit tokens to git**
- Use environment variables or CI/CD secrets
- Rotate test tokens regularly
- Use separate token for testing (not production)

### Test Repository

- Use a dedicated test repository
- Don't use production repositories
- Can be public or private (tests work for both)
- Should be a repository you control

## Additional Tests

For comprehensive testing, also run:

### Unit Tests

```bash
cd components/backend
go test ./gitlab/... -v
go test ./types/... -v
go test ./handlers/... -v
```

### Regression Tests

Verify GitHub functionality still works:

```bash
cd components/backend
go test ./tests/integration/github/... -v
```

## References

- [GitLab Integration Guide](../../../docs/gitlab-integration.md)
- [GitLab API Documentation](https://docs.gitlab.com/ee/api/)
- [Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
