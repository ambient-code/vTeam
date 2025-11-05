# GitLab Integration Test Plan

**Feature**: GitLab Support for vTeam
**Branch**: `feature/gitlab-support`
**Date**: 2025-11-05
**Status**: Ready for Testing

## Overview

This test plan validates the GitLab integration implemented in vTeam, covering User Story 1 (Configure GitLab Repository) and User Story 3 (Execute AgenticSession with GitLab).

## Prerequisites

### Test Environment Setup

1. **GitLab.com Account**
   - Active GitLab.com account
   - At least one test repository with write access

2. **Self-Hosted GitLab (Optional)**
   - Self-hosted GitLab instance (for advanced testing)
   - Test repository with write access

3. **GitLab Personal Access Token (PAT)**
   - Token with required scopes:
     - `api` (full API access)
     - `read_api` (read API)
     - `read_user` (read user info)
     - `write_repository` (push to repositories)
   - Create at: https://gitlab.com/-/profile/personal_access_tokens

4. **vTeam Environment**
   - vTeam backend running with Kubernetes access
   - Backend namespace: `vteam-backend`
   - kubectl access to backend namespace
   - Valid user authentication token

### Test Data

**GitLab.com Test Repository**:
```
URL: https://gitlab.com/<your-username>/<test-repo>.git
Example: https://gitlab.com/testuser/vteam-test-repo.git
```

**Self-Hosted Test Repository** (if applicable):
```
URL: https://gitlab.example.com/<owner>/<repo>.git
Example: https://gitlab.example.com/dev/integration-test.git
```

---

## Test Cases

### TC-001: GitLab Connection - Connect with Valid Token (GitLab.com)

**User Story**: US1 - Configure vTeam Project with GitLab Repository
**Priority**: P1 (Critical)

**Setup**:
1. Ensure user has no existing GitLab connection
2. Prepare valid GitLab PAT with required scopes

**Steps**:
1. Send POST request to `/auth/gitlab/connect`:
   ```json
   {
     "personalAccessToken": "<valid-token>",
     "instanceUrl": ""
   }
   ```

**Expected Results**:
- HTTP 200 OK response
- Response body contains:
  ```json
  {
    "userId": "<user-id>",
    "gitlabUserId": "<gitlab-user-id>",
    "username": "<gitlab-username>",
    "instanceUrl": "https://gitlab.com",
    "connected": true,
    "message": "GitLab account connected successfully"
  }
  ```
- Kubernetes Secret `gitlab-user-tokens` created in `vteam-backend` namespace
- Secret contains entry with key=`<user-id>`, value=`<token>`
- ConfigMap `gitlab-connections` created in `vteam-backend` namespace
- ConfigMap contains JSON entry with connection metadata

**Validation**:
```bash
# Check secret
kubectl get secret gitlab-user-tokens -n vteam-backend -o json | \
  jq '.data["<user-id>"]' | base64 -d

# Check configmap
kubectl get configmap gitlab-connections -n vteam-backend -o json | \
  jq '.data["<user-id>"]'
```

---

### TC-002: GitLab Connection - Connect with Invalid Token

**User Story**: US1
**Priority**: P1

**Setup**:
1. Prepare invalid GitLab PAT (expired or malformed)

**Steps**:
1. Send POST request to `/auth/gitlab/connect`:
   ```json
   {
     "personalAccessToken": "invalid-token-12345",
     "instanceUrl": ""
   }
   ```

**Expected Results**:
- HTTP 500 Internal Server Error response
- Response body contains error message indicating token validation failed
- Error message includes: "GitLab token validation failed" or "401 Unauthorized"
- No Secret or ConfigMap created

---

### TC-003: GitLab Connection - Connect with Self-Hosted Instance

**User Story**: US1
**Priority**: P2

**Setup**:
1. Prepare valid PAT for self-hosted GitLab instance
2. Note the instance URL (e.g., `https://gitlab.example.com`)

**Steps**:
1. Send POST request to `/auth/gitlab/connect`:
   ```json
   {
     "personalAccessToken": "<valid-token>",
     "instanceUrl": "https://gitlab.example.com"
   }
   ```

**Expected Results**:
- HTTP 200 OK response
- Response includes `"instanceUrl": "https://gitlab.example.com"`
- ConfigMap stores instanceURL correctly
- Self-hosted detection: `isGitlabSelfHosted: true` in repository metadata

---

### TC-004: GitLab Connection - Get Status (Connected)

**User Story**: US1
**Priority**: P1

**Setup**:
1. Complete TC-001 successfully (user connected)

**Steps**:
1. Send GET request to `/auth/gitlab/status`

**Expected Results**:
- HTTP 200 OK response
- Response body:
  ```json
  {
    "connected": true,
    "username": "<gitlab-username>",
    "instanceUrl": "https://gitlab.com",
    "gitlabUserId": "<gitlab-user-id>"
  }
  ```

---

### TC-005: GitLab Connection - Get Status (Not Connected)

**User Story**: US1
**Priority**: P1

**Setup**:
1. Ensure user has no GitLab connection

**Steps**:
1. Send GET request to `/auth/gitlab/status`

**Expected Results**:
- HTTP 200 OK response
- Response body:
  ```json
  {
    "connected": false
  }
  ```

---

### TC-006: GitLab Connection - Disconnect

**User Story**: US1
**Priority**: P1

**Setup**:
1. Complete TC-001 successfully (user connected)

**Steps**:
1. Send POST request to `/auth/gitlab/disconnect`

**Expected Results**:
- HTTP 200 OK response
- Response body:
  ```json
  {
    "message": "GitLab account disconnected successfully",
    "connected": false
  }
  ```
- Token removed from `gitlab-user-tokens` Secret
- Connection metadata removed from `gitlab-connections` ConfigMap

**Validation**:
```bash
# Verify token removed
kubectl get secret gitlab-user-tokens -n vteam-backend -o json | \
  jq '.data["<user-id>"]'  # Should return null

# Verify connection removed
kubectl get configmap gitlab-connections -n vteam-backend -o json | \
  jq '.data["<user-id>"]'  # Should return null
```

---

### TC-007: Repository Provider Detection - GitLab HTTPS URL

**User Story**: US1
**Priority**: P1

**Steps**:
1. Test provider detection with various GitLab HTTPS URLs:
   - `https://gitlab.com/owner/repo.git`
   - `https://gitlab.com/owner/repo`
   - `https://gitlab.example.com/group/project.git`

**Expected Results**:
- Provider detected as `gitlab`
- URL normalized to HTTPS format with `.git` suffix
- Self-hosted instances correctly identified

---

### TC-008: Repository Provider Detection - GitLab SSH URL

**User Story**: US1
**Priority**: P2

**Steps**:
1. Test provider detection with GitLab SSH URLs:
   - `git@gitlab.com:owner/repo.git`
   - `git@gitlab.example.com:group/project.git`

**Expected Results**:
- Provider detected as `gitlab`
- URL normalized to HTTPS format

---

### TC-009: Repository Configuration - Add GitLab Repository to Project

**User Story**: US1
**Priority**: P1

**Setup**:
1. Complete TC-001 (GitLab connected)
2. Create or use existing vTeam project

**Steps**:
1. Update ProjectSettings CR with GitLab repository:
   ```yaml
   spec:
     repositories:
       - url: "https://gitlab.com/testuser/vteam-test-repo.git"
         branch: "main"
   ```

**Expected Results**:
- ProjectSettings CR updated successfully
- Provider automatically detected as `gitlab`
- Repository validation succeeds (if connected)
- Provider field populated in repository entry

---

### TC-010: Repository Validation - Valid Repository with Valid Token

**User Story**: US1
**Priority**: P1

**Setup**:
1. Complete TC-001 (GitLab connected)
2. Use repository URL user has access to

**Steps**:
1. Call `ValidateProjectRepository` or configure project with GitLab repo

**Expected Results**:
- Validation succeeds
- Repository metadata returned:
  - `provider: "gitlab"`
  - `owner: "<owner>"`
  - `repo: "<repo>"`
  - `host: "gitlab.com"` or self-hosted host
  - `apiUrl: "https://gitlab.com/api/v4"` or self-hosted API URL

---

### TC-011: Repository Validation - Repository User Lacks Access

**User Story**: US1
**Priority**: P1

**Setup**:
1. Complete TC-001 (GitLab connected)
2. Use private repository URL user does NOT have access to

**Steps**:
1. Call `ValidateProjectRepository` with inaccessible repository

**Expected Results**:
- Validation fails
- Error message: "repository validation failed" or "404 Not Found"
- User-friendly error message explaining lack of access

---

### TC-012: AgenticSession - Clone GitLab Repository

**User Story**: US3 - Execute AgenticSession with GitLab Repository
**Priority**: P1

**Setup**:
1. Complete TC-001 (GitLab connected)
2. Complete TC-009 (Project configured with GitLab repo)
3. GitLab repository must exist and be accessible

**Steps**:
1. Create AgenticSession with GitLab repository
2. Monitor session logs for clone operation

**Expected Results**:
- Session pod starts successfully
- Clone operation uses oauth2:TOKEN@ authentication format
- Repository cloned successfully to session workspace
- Logs show: "Cloning GitLab repository: <repo-url>"

**Validation**:
```bash
# Check session logs
kubectl logs <session-pod> -n <project-namespace> | grep -i gitlab
```

---

### TC-013: AgenticSession - Commit and Push to GitLab Repository

**User Story**: US3
**Priority**: P1

**Setup**:
1. Complete TC-012 (Repository cloned)
2. AgenticSession makes file changes

**Steps**:
1. Wait for session to complete task
2. Verify commit created
3. Verify push to GitLab succeeds

**Expected Results**:
- Commit created locally with session changes
- Push to GitLab succeeds using oauth2:TOKEN@ authentication
- Changes visible in GitLab web UI
- Completion notification includes GitLab branch URL:
  - Format: `https://gitlab.com/<owner>/<repo>/-/tree/<branch>`

**Validation**:
```bash
# Check GitLab UI or API for pushed branch
curl -H "Authorization: Bearer <token>" \
  "https://gitlab.com/api/v4/projects/<project-id>/repository/branches/<branch>"
```

---

### TC-014: AgenticSession - Push Error (Insufficient Permissions)

**User Story**: US3
**Priority**: P1

**Setup**:
1. Complete TC-001 with token that has read-only access (no `write_repository` scope)
2. Complete TC-009 (Project configured)

**Steps**:
1. Create AgenticSession that attempts to push changes

**Expected Results**:
- Clone succeeds
- Commit succeeds
- Push fails with user-friendly error message
- Error message includes:
  - "GitLab push failed: Insufficient permissions"
  - "Ensure your GitLab token has 'write_repository' scope"
  - "You can update your token by reconnecting your GitLab account"

---

### TC-015: AgenticSession - Push Error (Invalid Token)

**User Story**: US3
**Priority**: P2

**Setup**:
1. Complete TC-001 (GitLab connected)
2. Manually invalidate token or wait for expiration
3. Complete TC-009 (Project configured)

**Steps**:
1. Create AgenticSession that attempts to push changes

**Expected Results**:
- Clone may fail or succeed (depending on timing)
- Push fails with authentication error
- Error message includes:
  - "GitLab push failed: Authentication failed"
  - "Your GitLab token may be invalid or expired"
  - "Please reconnect your GitLab account"

---

### TC-016: AgenticSession - Self-Hosted GitLab Instance

**User Story**: US3
**Priority**: P2

**Setup**:
1. Complete TC-003 (Self-hosted GitLab connected)
2. Configure project with self-hosted GitLab repository

**Steps**:
1. Create AgenticSession with self-hosted GitLab repo

**Expected Results**:
- Clone uses correct self-hosted instance URL
- Authentication works with self-hosted API
- Push succeeds to self-hosted instance
- Completion notification uses self-hosted URL:
  - Format: `https://gitlab.example.com/<owner>/<repo>/-/tree/<branch>`

---

### TC-017: Error Handling - User-Friendly Messages

**User Story**: US1, US3
**Priority**: P1

**Test Scenarios**:

| Scenario | Expected Error Message |
|----------|------------------------|
| Invalid token | "GitLab token validation failed" with 401 details |
| Insufficient permissions | "Ensure your GitLab token has 'write_repository' scope" |
| Repository not found | "Repository not found. Verify the repository URL" |
| Rate limit exceeded | "Rate limit exceeded. Please wait a few minutes" |
| Network error | "Unable to connect to gitlab.com. Check network connectivity" |

**Validation**:
- All error messages are user-friendly (no raw stack traces)
- Error messages include remediation guidance
- Tokens are redacted in all log output

---

### TC-018: Token Security - Redaction in Logs

**User Story**: US1, US3
**Priority**: P1 (Security Critical)

**Steps**:
1. Complete TC-001 (GitLab connected)
2. Trigger various operations (validate, clone, push)
3. Review all logs (backend, session pod)

**Expected Results**:
- Tokens never appear in plaintext in logs
- Token patterns redacted:
  - `glpat-...` → `glpat-***`
  - `Bearer <token>` → `Bearer ***`
  - `oauth2:TOKEN@` → `oauth2:***@`
- Git URLs in logs show redacted tokens

**Validation**:
```bash
# Search backend logs for tokens
kubectl logs <backend-pod> -n vteam-backend | grep -i "glpat-"  # Should find no matches
kubectl logs <backend-pod> -n vteam-backend | grep "oauth2:" | grep -v "***"  # Should find no matches

# Search session logs
kubectl logs <session-pod> -n <project> | grep -i "token" | grep -v "***"  # Should find no matches
```

---

### TC-019: Mixed Providers - GitHub and GitLab in Same Project

**User Story**: US4 (if implemented)
**Priority**: P3

**Setup**:
1. Connect both GitHub and GitLab accounts
2. Configure project with both providers:
   ```yaml
   spec:
     repositories:
       - url: "https://github.com/user/repo.git"
         provider: "github"
       - url: "https://gitlab.com/user/repo.git"
         provider: "gitlab"
   ```

**Steps**:
1. Create AgenticSession that uses both repositories

**Expected Results**:
- Both repositories cloned successfully
- Correct authentication used for each provider
- GitHub uses x-access-token, GitLab uses oauth2
- Both pushes succeed independently

---

## Regression Testing

### RT-001: GitHub Functionality Unaffected

**Priority**: P1 (Critical)

**Steps**:
1. Run existing GitHub integration tests
2. Verify GitHub App connections still work
3. Verify GitHub repository operations (clone, push) still work
4. Verify GitHub error handling unchanged

**Expected Results**:
- Zero GitHub functionality regression
- All existing GitHub tests pass
- GitHub-only projects work identically to before

---

### RT-002: Backward Compatibility - Existing Projects

**Priority**: P1

**Steps**:
1. Load existing ProjectSettings CRs (created before GitLab support)
2. Verify they continue to work

**Expected Results**:
- Existing projects without `provider` field work correctly
- Provider auto-detected for existing repositories
- No migration required for existing projects

---

## Performance Testing

### PT-001: Token Validation Performance

**Target**: < 200ms per validation (per SC-002)

**Steps**:
1. Measure time for GitLab token validation
2. Test with GitLab.com and self-hosted instances

**Expected Results**:
- Validation completes in < 200ms (95th percentile)
- Timeout set to 15 seconds (matches GitHub)

---

### PT-002: Repository Browsing Performance

**Target**: < 3s for browsing operations (per SC-002)

**Steps**:
1. List branches in large repository (100+ branches)
2. Browse large directory tree (1000+ files)

**Expected Results**:
- Operations complete in < 3s (95th percentile)
- Pagination works correctly for large result sets

---

## Security Testing

### ST-001: Token Storage Security

**Steps**:
1. Verify tokens stored in Kubernetes Secrets
2. Verify tokens encrypted at rest (K8s default)
3. Verify tokens never logged in plaintext
4. Verify tokens never exposed in API responses

**Expected Results**:
- All security requirements met
- No token leakage vectors found

---

### ST-002: Input Validation

**Steps**:
1. Test with malicious repository URLs:
   - Path traversal: `https://gitlab.com/../../etc/passwd`
   - Script injection: `https://gitlab.com/<script>alert()</script>`
2. Test with malformed tokens
3. Test with excessively long inputs

**Expected Results**:
- All malicious inputs rejected
- No code injection possible
- No path traversal vulnerabilities

---

## Manual Testing Checklist

### Setup Phase
- [ ] Deploy vTeam backend with GitLab support
- [ ] Verify backend namespace exists (`vteam-backend`)
- [ ] Create GitLab.com test account and repository
- [ ] Generate GitLab PAT with required scopes
- [ ] (Optional) Set up self-hosted GitLab instance

### User Story 1: Configure GitLab
- [ ] TC-001: Connect with valid token (GitLab.com)
- [ ] TC-002: Connect with invalid token
- [ ] TC-003: Connect with self-hosted instance
- [ ] TC-004: Get status (connected)
- [ ] TC-005: Get status (not connected)
- [ ] TC-006: Disconnect
- [ ] TC-007: Provider detection (HTTPS URLs)
- [ ] TC-008: Provider detection (SSH URLs)
- [ ] TC-009: Add GitLab repository to project
- [ ] TC-010: Validate repository with access
- [ ] TC-011: Validate repository without access

### User Story 3: AgenticSession
- [ ] TC-012: Clone GitLab repository
- [ ] TC-013: Commit and push to GitLab
- [ ] TC-014: Push error (insufficient permissions)
- [ ] TC-015: Push error (invalid token)
- [ ] TC-016: Self-hosted GitLab instance

### Error Handling & Security
- [ ] TC-017: User-friendly error messages
- [ ] TC-018: Token redaction in logs
- [ ] ST-001: Token storage security
- [ ] ST-002: Input validation

### Regression & Performance
- [ ] RT-001: GitHub functionality unaffected
- [ ] RT-002: Backward compatibility
- [ ] PT-001: Token validation performance
- [ ] PT-002: Repository browsing performance (if implemented)

---

## Test Results Template

```markdown
### Test Execution Report

**Date**: YYYY-MM-DD
**Tester**: <name>
**Environment**: <dev/staging/prod>

| Test Case | Status | Notes |
|-----------|--------|-------|
| TC-001 | ✅ Pass | |
| TC-002 | ✅ Pass | |
| ... | | |

**Summary**:
- Total Tests: X
- Passed: Y
- Failed: Z
- Blocked: W

**Issues Found**:
1. [Issue description]
2. ...

**Recommendations**:
1. [Recommendation]
2. ...
```

---

## Known Limitations

1. **User Story 2 (Repository Browsing)**: Not yet implemented
2. **User Story 4 (Mixed Providers)**: Basic support implemented, advanced scenarios untested
3. **User Story 5 (Repository Seeding)**: Not yet implemented
4. **GitLab Groups**: Nested group paths may need additional testing
5. **GitLab Subgroups**: URL parsing for subgroups (e.g., `group/subgroup/project`) needs validation

---

## References

- **Specification**: `specs/001-gitlab-support/spec.md`
- **Task List**: `specs/001-gitlab-support/tasks.md`
- **Implementation Files**:
  - `components/backend/gitlab/`
  - `components/backend/handlers/gitlab_auth.go`
  - `components/backend/handlers/repository.go`
  - `components/backend/git/operations.go`

---

## Support

For issues or questions during testing:
- Review backend logs: `kubectl logs -l app=vteam-backend -n vteam-backend`
- Review session logs: `kubectl logs <session-pod> -n <project-namespace>`
- Check GitLab API responses using curl with PAT
- Verify Kubernetes resources: Secrets and ConfigMaps in `vteam-backend` namespace
