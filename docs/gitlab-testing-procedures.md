# GitLab Integration Testing Procedures

## Quick Start Guide for Testing GitLab Support

This guide provides step-by-step instructions for manually testing the GitLab integration in vTeam.

---

## Prerequisites

### 1. GitLab Personal Access Token Setup

1. **Log in to GitLab**:
   - For GitLab.com: https://gitlab.com
   - For self-hosted: https://your-gitlab-instance.com

2. **Navigate to Access Tokens**:
   - Click your profile icon (top right)
   - Select "Preferences"
   - Click "Access Tokens" in left sidebar
   - Or direct link: https://gitlab.com/-/profile/personal_access_tokens

3. **Create New Token**:
   - **Token name**: `vTeam Integration Test`
   - **Expiration date**: Set 30+ days from now
   - **Scopes** (select ALL of these):
     - ✅ `api` - Full API access
     - ✅ `read_api` - Read API
     - ✅ `read_user` - Read user information
     - ✅ `write_repository` - Push to repositories

4. **Copy Token**:
   - Click "Create personal access token"
   - **IMPORTANT**: Copy the token immediately (starts with `glpat-`)
   - Store securely - you won't be able to see it again

**Example Token**: `glpat-xyz123abc456def789` (yours will be different)

---

### 2. Test Repository Setup

1. **Create Test Repository** (GitLab.com):
   - Go to https://gitlab.com/projects/new
   - Project name: `vteam-test-repo`
   - Visibility: Private or Public (your choice)
   - Initialize with README: ✅
   - Click "Create project"

2. **Note Repository URL**:
   - Clone button → Copy HTTPS URL
   - Example: `https://gitlab.com/yourusername/vteam-test-repo.git`

3. **Verify Access**:
   ```bash
   git clone https://oauth2:<your-token>@gitlab.com/yourusername/vteam-test-repo.git
   ```
   - Should clone successfully
   - Delete cloned folder after verification

---

### 3. vTeam Environment Setup

1. **Verify Backend Running**:
   ```bash
   kubectl get pods -n vteam-backend
   ```
   - Should show backend pod in Running state

2. **Get Backend URL**:
   ```bash
   # Get service URL (adjust for your environment)
   kubectl get svc -n vteam-backend
   ```
   - Note the backend API URL (e.g., `http://vteam-backend.vteam-backend.svc.cluster.local:8080`)

3. **Get User Auth Token**:
   - Log in to vTeam UI
   - Open browser developer console
   - Find auth token in localStorage or cookies
   - Or use test user token if available

---

## Test Procedures

### Test 1: Connect GitLab Account

**Objective**: Verify user can connect their GitLab account to vTeam

**Steps**:

1. **Send Connect Request**:
   ```bash
   curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer <your-vteam-token>" \
     -d '{
       "personalAccessToken": "glpat-your-actual-token-here",
       "instanceUrl": ""
     }'
   ```

2. **Expected Response** (200 OK):
   ```json
   {
     "userId": "user-123",
     "gitlabUserId": "789456",
     "username": "yourusername",
     "instanceUrl": "https://gitlab.com",
     "connected": true,
     "message": "GitLab account connected successfully"
   }
   ```

3. **Verify in Kubernetes**:
   ```bash
   # Check secret created
   kubectl get secret gitlab-user-tokens -n vteam-backend -o yaml

   # Check configmap created
   kubectl get configmap gitlab-connections -n vteam-backend -o yaml
   ```

**Success Criteria**:
- ✅ HTTP 200 response received
- ✅ Response includes your GitLab username
- ✅ Secret `gitlab-user-tokens` exists
- ✅ ConfigMap `gitlab-connections` exists
- ✅ Your user ID appears in both resources

---

### Test 2: Check Connection Status

**Objective**: Verify connection status endpoint returns correct information

**Steps**:

1. **Send Status Request**:
   ```bash
   curl -X GET http://vteam-backend:8080/api/auth/gitlab/status \
     -H "Authorization: Bearer <your-vteam-token>"
   ```

2. **Expected Response** (200 OK):
   ```json
   {
     "connected": true,
     "username": "yourusername",
     "instanceUrl": "https://gitlab.com",
     "gitlabUserId": "789456"
   }
   ```

**Success Criteria**:
- ✅ HTTP 200 response
- ✅ `connected: true`
- ✅ Your GitLab username shown
- ✅ Correct instanceUrl

---

### Test 3: Configure Project with GitLab Repository

**Objective**: Add GitLab repository to vTeam project

**Steps**:

1. **Create or Select Project**:
   - Use existing vTeam project or create new one
   - Note project namespace (e.g., `my-project`)

2. **Update ProjectSettings CR**:
   ```bash
   kubectl edit projectsettings -n <project-namespace>
   ```

3. **Add GitLab Repository**:
   ```yaml
   spec:
     repositories:
       - url: "https://gitlab.com/yourusername/vteam-test-repo.git"
         branch: "main"
   ```

4. **Save and Verify**:
   ```bash
   kubectl get projectsettings -n <project-namespace> -o yaml
   ```

**Success Criteria**:
- ✅ ProjectSettings updated successfully
- ✅ Repository appears in spec
- ✅ Provider auto-detected as `gitlab`

---

### Test 4: Create AgenticSession with GitLab Repository

**Objective**: Verify session can clone, commit, and push to GitLab

**Steps**:

1. **Create AgenticSession CR**:
   ```bash
   kubectl apply -f - <<EOF
   apiVersion: ambient-code.io/v1alpha1
   kind: AgenticSession
   metadata:
     name: test-gitlab-session
     namespace: <project-namespace>
   spec:
     description: "Test GitLab integration by adding a comment to README"
     outputRepo:
       url: "https://gitlab.com/yourusername/vteam-test-repo.git"
       branch: "test-branch"
   EOF
   ```

2. **Monitor Session**:
   ```bash
   # Watch session status
   kubectl get agenticsession test-gitlab-session -n <project-namespace> -w

   # View session logs
   kubectl logs -l agenticsession=test-gitlab-session -n <project-namespace> -f
   ```

3. **Check for Key Log Messages**:
   - "Cloning GitLab repository"
   - "Using GitLab token for user"
   - "Push succeeded"
   - GitLab branch URL in completion notification

4. **Verify in GitLab UI**:
   - Open repository in GitLab: https://gitlab.com/yourusername/vteam-test-repo
   - Click "Branches" dropdown
   - Find `test-branch`
   - Verify commits appear from session

**Success Criteria**:
- ✅ Session pod starts successfully
- ✅ Repository clones without errors
- ✅ Changes committed locally
- ✅ Push to GitLab succeeds
- ✅ Branch visible in GitLab UI
- ✅ Completion notification includes GitLab URL format:
  - `https://gitlab.com/yourusername/vteam-test-repo/-/tree/test-branch`

---

### Test 5: Test Error Handling - Insufficient Permissions

**Objective**: Verify user-friendly error when token lacks write access

**Steps**:

1. **Create Read-Only Token**:
   - GitLab → Access Tokens
   - Create new token with ONLY these scopes:
     - ✅ `read_api`
     - ✅ `read_user`
   - **DO NOT** select `write_repository`

2. **Connect with Read-Only Token**:
   ```bash
   curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer <your-vteam-token>" \
     -d '{
       "personalAccessToken": "glpat-readonly-token-here",
       "instanceUrl": ""
     }'
   ```

3. **Create AgenticSession** (same as Test 4)

4. **Observe Push Failure**:
   - Clone should succeed
   - Commit should succeed
   - Push should FAIL with user-friendly error

**Expected Error Message**:
```
GitLab push failed: Insufficient permissions. Ensure your GitLab token has 'write_repository' scope. You can update your token by reconnecting your GitLab account with the required permissions
```

**Success Criteria**:
- ✅ Error message is user-friendly (no stack traces)
- ✅ Error mentions `write_repository` scope
- ✅ Error includes remediation guidance
- ✅ Session status shows failure reason

---

### Test 6: Token Security - Verify Redaction

**Objective**: Ensure tokens never appear in logs

**Steps**:

1. **Search Backend Logs**:
   ```bash
   # Should find NO raw tokens
   kubectl logs -l app=vteam-backend -n vteam-backend | grep "glpat-"

   # Should only find redacted tokens (with ***)
   kubectl logs -l app=vteam-backend -n vteam-backend | grep "oauth2:"
   ```

2. **Search Session Logs**:
   ```bash
   # Should find NO raw tokens
   kubectl logs -l agenticsession=test-gitlab-session -n <project-namespace> | grep "glpat-"

   # Git URLs should be redacted
   kubectl logs -l agenticsession=test-gitlab-session -n <project-namespace> | grep "https://" | grep "gitlab"
   ```

**Success Criteria**:
- ✅ No raw tokens in backend logs
- ✅ No raw tokens in session logs
- ✅ Git URLs show `oauth2:***@` instead of `oauth2:<token>@`
- ✅ API calls show `Bearer ***` instead of `Bearer <token>`

---

### Test 7: Disconnect GitLab Account

**Objective**: Verify user can safely disconnect GitLab

**Steps**:

1. **Send Disconnect Request**:
   ```bash
   curl -X POST http://vteam-backend:8080/api/auth/gitlab/disconnect \
     -H "Authorization: Bearer <your-vteam-token>"
   ```

2. **Expected Response** (200 OK):
   ```json
   {
     "message": "GitLab account disconnected successfully",
     "connected": false
   }
   ```

3. **Verify Removal**:
   ```bash
   # Check token removed from secret
   kubectl get secret gitlab-user-tokens -n vteam-backend -o json | \
     jq '.data | keys'

   # Check connection removed from configmap
   kubectl get configmap gitlab-connections -n vteam-backend -o json | \
     jq '.data | keys'
   ```

4. **Verify Status Shows Disconnected**:
   ```bash
   curl -X GET http://vteam-backend:8080/api/auth/gitlab/status \
     -H "Authorization: Bearer <your-vteam-token>"
   ```

   Expected: `{"connected": false}`

**Success Criteria**:
- ✅ HTTP 200 response
- ✅ Token removed from Secret
- ✅ Connection removed from ConfigMap
- ✅ Status endpoint returns `connected: false`

---

### Test 8: Self-Hosted GitLab (Optional)

**Objective**: Verify self-hosted GitLab instances work

**Prerequisites**:
- Access to self-hosted GitLab instance
- Repository on self-hosted instance
- PAT from self-hosted instance

**Steps**:

1. **Connect with Instance URL**:
   ```bash
   curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer <your-vteam-token>" \
     -d '{
       "personalAccessToken": "glpat-self-hosted-token",
       "instanceUrl": "https://gitlab.example.com"
     }'
   ```

2. **Verify Response**:
   - Check `instanceUrl` matches your self-hosted URL
   - Not `https://gitlab.com`

3. **Create AgenticSession with Self-Hosted Repo**:
   ```yaml
   spec:
     outputRepo:
       url: "https://gitlab.example.com/group/project.git"
       branch: "test-branch"
   ```

4. **Verify Operations**:
   - Clone uses self-hosted URL
   - API calls go to `https://gitlab.example.com/api/v4`
   - Push succeeds to self-hosted instance
   - Completion URL uses self-hosted domain

**Success Criteria**:
- ✅ Connection succeeds with custom instanceUrl
- ✅ Self-hosted API URL constructed correctly
- ✅ Clone/push work with self-hosted instance
- ✅ Completion notification shows self-hosted URL

---

### Test 9: Regression - GitHub Still Works

**Objective**: Verify GitHub functionality unaffected by GitLab changes

**Steps**:

1. **Connect GitHub Account** (if not already):
   - Use existing GitHub App integration
   - Or configure GitHub PAT in runner secrets

2. **Create AgenticSession with GitHub Repo**:
   ```yaml
   spec:
     outputRepo:
       url: "https://github.com/username/repo.git"
       branch: "test-branch"
   ```

3. **Verify GitHub Operations**:
   - Clone uses `x-access-token` authentication
   - Push succeeds to GitHub
   - Completion URL uses GitHub format: `https://github.com/username/repo/tree/test-branch`

**Success Criteria**:
- ✅ GitHub sessions work identically to before GitLab support
- ✅ GitHub authentication unchanged
- ✅ No errors related to provider detection
- ✅ GitHub and GitLab can coexist in same backend instance

---

## Troubleshooting Guide

### Issue: Connection Fails with "Invalid Token"

**Symptoms**:
- HTTP 500 response
- Error: "GitLab token validation failed"

**Solutions**:
1. Verify token is copied correctly (no extra spaces)
2. Check token hasn't expired in GitLab
3. Verify token has required scopes:
   ```bash
   curl -H "Authorization: Bearer <your-token>" \
     https://gitlab.com/api/v4/personal_access_tokens/self
   ```
4. Check backend logs:
   ```bash
   kubectl logs -l app=vteam-backend -n vteam-backend | grep -i "gitlab"
   ```

---

### Issue: Session Clone Fails

**Symptoms**:
- Session pod starts but clone fails
- Error: "no GitLab credentials available"

**Solutions**:
1. Verify GitLab account connected:
   ```bash
   curl -X GET http://vteam-backend:8080/api/auth/gitlab/status \
     -H "Authorization: Bearer <token>"
   ```
2. Check token exists in Secret:
   ```bash
   kubectl get secret gitlab-user-tokens -n vteam-backend -o yaml
   ```
3. Verify namespace is correct (`vteam-backend`)
4. Check session logs for detailed error:
   ```bash
   kubectl logs <session-pod> -n <project-namespace>
   ```

---

### Issue: Push Fails with 403 Forbidden

**Symptoms**:
- Clone and commit succeed
- Push fails with "Insufficient permissions"

**Solutions**:
1. Verify token has `write_repository` scope:
   - GitLab → Access Tokens → View your token
   - Check scopes list
2. Regenerate token with correct scopes if needed
3. Reconnect account:
   ```bash
   # Disconnect
   curl -X POST http://vteam-backend:8080/api/auth/gitlab/disconnect \
     -H "Authorization: Bearer <token>"

   # Reconnect with new token
   curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer <token>" \
     -d '{"personalAccessToken": "glpat-new-token", "instanceUrl": ""}'
   ```

---

### Issue: Self-Hosted Instance Not Detected

**Symptoms**:
- Self-hosted GitLab treated as GitLab.com
- API calls fail with 404

**Solutions**:
1. Ensure `instanceUrl` provided when connecting:
   ```json
   {
     "personalAccessToken": "glpat-...",
     "instanceUrl": "https://gitlab.example.com"  // REQUIRED
   }
   ```
2. Verify instance URL format:
   - Must include `https://`
   - No trailing slash
   - No `/api/v4` path
3. Check repository URL includes correct host:
   - ✅ `https://gitlab.example.com/group/project.git`
   - ❌ `https://gitlab.com/group/project.git`

---

### Issue: Tokens Visible in Logs

**Symptoms**:
- Raw tokens appear in kubectl logs output

**CRITICAL SECURITY ISSUE**:
1. Immediately report this issue
2. Rotate all affected tokens in GitLab
3. Check backend logs for redaction failures:
   ```bash
   kubectl logs -l app=vteam-backend -n vteam-backend | grep -E "(glpat-|oauth2:)" | grep -v "***"
   ```

---

## Test Results Checklist

After completing all tests, verify:

**Connection Management**:
- [ ] Connect with valid token works
- [ ] Connect with invalid token shows error
- [ ] Status endpoint accurate (connected/disconnected)
- [ ] Disconnect removes credentials
- [ ] Self-hosted instance works (if tested)

**Repository Operations**:
- [ ] Provider detection works (HTTPS, SSH)
- [ ] Repository validation works
- [ ] ProjectSettings accepts GitLab URLs

**AgenticSession**:
- [ ] Clone succeeds with GitLab repo
- [ ] Commit creates changes locally
- [ ] Push succeeds to GitLab
- [ ] Completion notification shows GitLab URL
- [ ] Changes visible in GitLab UI

**Error Handling**:
- [ ] Insufficient permissions shows user-friendly error
- [ ] Invalid token shows clear error message
- [ ] All errors include remediation guidance

**Security**:
- [ ] Tokens stored in Kubernetes Secrets
- [ ] Tokens redacted in all logs
- [ ] No plaintext tokens in API responses

**Regression**:
- [ ] GitHub functionality unchanged
- [ ] Existing projects work correctly
- [ ] No performance degradation

---

## Quick Reference Commands

### Backend Logs
```bash
kubectl logs -l app=vteam-backend -n vteam-backend -f
```

### Session Logs
```bash
kubectl logs -l agenticsession=<session-name> -n <project-namespace> -f
```

### Check Secrets
```bash
kubectl get secret gitlab-user-tokens -n vteam-backend -o yaml
```

### Check ConfigMaps
```bash
kubectl get configmap gitlab-connections -n vteam-backend -o yaml
```

### GitLab API Test
```bash
# Test your token manually
curl -H "Authorization: Bearer glpat-..." \
  https://gitlab.com/api/v4/user
```

### Clean Up Test Resources
```bash
# Delete test session
kubectl delete agenticsession test-gitlab-session -n <project-namespace>

# Disconnect GitLab
curl -X POST http://vteam-backend:8080/api/auth/gitlab/disconnect \
  -H "Authorization: Bearer <token>"
```

---

## Next Steps

After successful testing:
1. Document any issues found
2. Create bug reports for failures
3. Update test plan with additional scenarios discovered
4. Prepare for production deployment

For production deployment:
- Review security checklist
- Plan token rotation strategy
- Configure monitoring/alerting
- Prepare user documentation
- Train support team on GitLab integration

---

## Support Resources

- **GitLab API Docs**: https://docs.gitlab.com/ee/api/
- **GitLab PAT Docs**: https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html
- **vTeam GitLab Test Plan**: `/docs/gitlab-integration-test-plan.md`
- **GitLab Integration Spec**: `specs/001-gitlab-support/spec.md`
