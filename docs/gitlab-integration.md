# GitLab Integration for vTeam

vTeam now supports GitLab repositories alongside GitHub, enabling you to use your GitLab projects with AgenticSessions. This guide covers everything you need to know about using GitLab with vTeam.

## Overview

**What's Supported:**
- ✅ GitLab.com (public SaaS)
- ✅ Self-hosted GitLab instances (Community & Enterprise editions)
- ✅ Personal Access Token (PAT) authentication
- ✅ HTTPS and SSH URL formats
- ✅ Public and private repositories
- ✅ Clone, commit, and push operations
- ✅ Multi-repository projects (mix GitHub and GitLab)

**Requirements:**
- GitLab Personal Access Token with appropriate scopes
- Repository with write access (for AgenticSessions)
- vTeam backend v1.1.0 or higher

---

## Quick Start

### 1. Create GitLab Personal Access Token

1. **Log in to GitLab**: https://gitlab.com (or your self-hosted instance)

2. **Navigate to Access Tokens**:
   - Click your profile icon (top right)
   - Select "Preferences" → "Access Tokens"
   - Or visit: https://gitlab.com/-/profile/personal_access_tokens

3. **Create Token**:
   - **Token name**: `vTeam Integration`
   - **Expiration**: Set 90+ days from now
   - **Select scopes**:
     - ✅ `api` - Full API access (required)
     - ✅ `read_api` - Read API access
     - ✅ `read_user` - Read user information
     - ✅ `write_repository` - Push to repositories

4. **Copy Token**: Save the token starting with `glpat-...` securely

**Detailed instructions**: See [GitLab PAT Setup Guide](./gitlab-token-setup.md)

---

### 2. Connect GitLab Account to vTeam

**Via vTeam UI** (if available):
1. Navigate to Settings → Integrations
2. Click "Connect GitLab"
3. Paste your Personal Access Token
4. (Optional) For self-hosted: Enter instance URL (e.g., `https://gitlab.company.com`)
5. Click "Connect"

**Via API**:
```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-vteam-auth-token>" \
  -d '{
    "personalAccessToken": "glpat-your-token-here",
    "instanceUrl": ""
  }'
```

**For self-hosted GitLab**, include the instance URL:
```json
{
  "personalAccessToken": "glpat-your-token-here",
  "instanceUrl": "https://gitlab.company.com"
}
```

**Success Response**:
```json
{
  "userId": "user-123",
  "gitlabUserId": "456789",
  "username": "yourname",
  "instanceUrl": "https://gitlab.com",
  "connected": true,
  "message": "GitLab account connected successfully"
}
```

---

### 3. Configure Project with GitLab Repository

**Option A: Via vTeam UI**
1. Open your vTeam project
2. Navigate to Project Settings
3. Under "Repositories", click "Add Repository"
4. Enter GitLab repository URL:
   - HTTPS: `https://gitlab.com/owner/repo.git`
   - SSH: `git@gitlab.com:owner/repo.git`
5. Enter default branch (e.g., `main`)
6. Save settings

**Option B: Via Kubernetes**

Edit your ProjectSettings custom resource:
```bash
kubectl edit projectsettings -n <your-project-namespace>
```

Add GitLab repository to spec:
```yaml
apiVersion: ambient-code.io/v1
kind: ProjectSettings
metadata:
  name: projectsettings
  namespace: my-project
spec:
  repositories:
    - url: "https://gitlab.com/myteam/myrepo.git"
      branch: "main"
      provider: "gitlab"  # Auto-detected, optional
```

**Multiple Repositories** (GitHub + GitLab):
```yaml
spec:
  repositories:
    - url: "https://github.com/myteam/frontend.git"
      branch: "main"
    - url: "https://gitlab.com/myteam/backend.git"
      branch: "develop"
```

---

### 4. Create AgenticSession with GitLab Repository

Once your GitLab account is connected and repository configured, create sessions normally:

**Example AgenticSession CR**:
```yaml
apiVersion: ambient-code.io/v1alpha1
kind: AgenticSession
metadata:
  name: add-feature-x
  namespace: my-project
spec:
  description: "Add feature X to the backend service"
  outputRepo:
    url: "https://gitlab.com/myteam/backend.git"
    branch: "feature/add-feature-x"
```

**What Happens**:
1. Session pod starts with your task description
2. Repository clones using your GitLab PAT (automatic authentication)
3. Claude Code agent makes changes
4. Changes committed to local repository
5. Branch pushed to GitLab with your commits
6. Completion notification includes GitLab branch link

**Completion Notification**:
```
AgenticSession completed successfully!

View changes in GitLab:
https://gitlab.com/myteam/backend/-/tree/feature/add-feature-x
```

---

## Supported URL Formats

vTeam automatically detects and normalizes GitLab URLs:

### HTTPS URLs (Recommended)
```
✅ https://gitlab.com/owner/repo.git
✅ https://gitlab.com/owner/repo
✅ https://gitlab.company.com/group/project.git
✅ https://gitlab.company.com/group/subgroup/project.git
```

### SSH URLs
```
✅ git@gitlab.com:owner/repo.git
✅ git@gitlab.company.com:group/project.git
```

**Provider Auto-Detection**:
- URLs containing `gitlab.com` → Detected as GitLab.com
- URLs containing other hosts with `gitlab` pattern → Detected as self-hosted GitLab
- Provider field is optional in ProjectSettings (auto-detected from URL)

---

## Repository Access Validation

vTeam validates your access to GitLab repositories before allowing operations:

**Validation Checks**:
1. ✅ Token is valid and not expired
2. ✅ User has access to repository
3. ✅ Token has sufficient permissions (read + write)
4. ✅ Repository exists and is accessible

**When Validation Occurs**:
- When connecting GitLab account (token validation)
- When configuring project repository (repository access check)
- Before starting AgenticSession (pre-flight validation)

**Common Validation Errors**:

| Error | Cause | Solution |
|-------|-------|----------|
| Invalid token | Token expired or revoked | Reconnect with new PAT |
| Insufficient permissions | Token lacks `write_repository` | Recreate token with required scopes |
| Repository not found | Private repo or no access | Verify URL and repository permissions |
| Rate limit exceeded | Too many API calls | Wait a few minutes, then retry |

---

## Managing GitLab Connection

### Check Connection Status

**Via API**:
```bash
curl -X GET http://vteam-backend:8080/api/auth/gitlab/status \
  -H "Authorization: Bearer <your-vteam-token>"
```

**Response (Connected)**:
```json
{
  "connected": true,
  "username": "yourname",
  "instanceUrl": "https://gitlab.com",
  "gitlabUserId": "456789"
}
```

**Response (Not Connected)**:
```json
{
  "connected": false
}
```

### Disconnect GitLab Account

**Via API**:
```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/disconnect \
  -H "Authorization: Bearer <your-vteam-token>"
```

This removes:
- Your GitLab PAT from vTeam secrets
- Connection metadata
- Access to GitLab repositories (AgenticSessions will fail)

**Note**: Your repositories and GitLab account are not affected.

### Update GitLab Token

To update your token (when expired or scopes changed):

1. **Disconnect** current account
2. **Create new token** in GitLab with updated scopes
3. **Reconnect** with new token

---

## Self-Hosted GitLab

vTeam fully supports self-hosted GitLab instances (Community and Enterprise editions).

### Requirements

- GitLab instance accessible from vTeam backend pods
- Personal Access Token from your GitLab instance
- Network connectivity to GitLab API (default: port 443)

### Configuration

When connecting, provide your instance URL:

```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <vteam-token>" \
  -d '{
    "personalAccessToken": "glpat-xxx",
    "instanceUrl": "https://gitlab.company.com"
  }'
```

**Instance URL Format**:
- ✅ Include `https://` protocol
- ✅ No trailing slash
- ❌ Don't include `/api/v4` path

**Examples**:
```
✅ https://gitlab.company.com
✅ https://git.myorg.io
❌ gitlab.company.com (missing protocol)
❌ https://gitlab.company.com/ (trailing slash)
❌ https://gitlab.company.com/api/v4 (includes API path)
```

### API URL Construction

vTeam automatically constructs the correct API URL:

| Instance URL | API URL |
|--------------|---------|
| `https://gitlab.com` | `https://gitlab.com/api/v4` |
| `https://gitlab.company.com` | `https://gitlab.company.com/api/v4` |
| `https://git.myorg.io` | `https://git.myorg.io/api/v4` |

### Troubleshooting Self-Hosted

**Issue**: Connection fails with "connection refused"

**Solutions**:
1. Verify instance URL is correct and accessible
2. Check network connectivity from backend pods:
   ```bash
   kubectl exec -it <backend-pod> -n vteam-backend -- curl https://gitlab.company.com
   ```
3. Verify SSL certificate is valid (or configure trust for self-signed)
4. Check firewall rules allow traffic from Kubernetes cluster

**Issue**: Token validation fails

**Solutions**:
1. Verify token is from correct GitLab instance (not GitLab.com)
2. Check token hasn't expired
3. Verify admin hasn't revoked token
4. Test token manually:
   ```bash
   curl -H "Authorization: Bearer glpat-xxx" \
     https://gitlab.company.com/api/v4/user
   ```

**Detailed guide**: See [Self-Hosted GitLab Configuration](./gitlab-self-hosted.md)

---

## Security & Best Practices

### Token Security

**How Tokens Are Stored**:
- ✅ Stored in Kubernetes Secrets (encrypted at rest)
- ✅ Never logged in plaintext
- ✅ Redacted in all log output (`glpat-***`)
- ✅ Not exposed in API responses
- ✅ Injected into git URLs only in memory

**Token Redaction Examples**:
```
# In logs, you'll see:
[GitLab] Using token glpat-*** for user john
[GitLab] Cloning https://oauth2:***@gitlab.com/team/repo.git

# Never:
[GitLab] Using token glpat-abc123xyz456
```

### Token Scopes

**Minimum Required Scopes**:
- `api` - Full API access
- `read_repository` - Clone repositories
- `write_repository` - Push changes

**Recommended Scopes**:
- `api` - Covers all operations
- `read_api` - Read-only API access
- `read_user` - User information
- `write_repository` - Push to repos

**Avoid**:
- `sudo` - Not needed, grants excessive privileges
- `admin_mode` - Not needed

### Token Rotation

**Recommendation**: Rotate tokens every 90 days

**Process**:
1. Create new token in GitLab with same scopes
2. Test new token works (curl to GitLab API)
3. Disconnect vTeam GitLab connection
4. Reconnect with new token
5. Revoke old token in GitLab

### Repository Permissions

**Required Permissions**:
- Read access for cloning
- Write access for pushing changes
- For private repos: Must be member with at least Developer role

**GitLab Role Requirements**:
| Role | Can Clone | Can Push | Recommended For |
|------|-----------|----------|-----------------|
| Guest | ❌ | ❌ | Not supported |
| Reporter | ✅ | ❌ | Read-only use cases |
| Developer | ✅ | ✅ | ✅ AgenticSessions |
| Maintainer | ✅ | ✅ | ✅ AgenticSessions |
| Owner | ✅ | ✅ | ✅ AgenticSessions |

---

## Troubleshooting

### Connection Issues

**Problem**: "Invalid token" error when connecting

**Solutions**:
1. Verify token is copied correctly (no extra spaces)
2. Check token hasn't expired in GitLab
3. For self-hosted: Ensure `instanceUrl` is correct
4. Test token manually:
   ```bash
   curl -H "Authorization: Bearer glpat-xxx" \
     https://gitlab.com/api/v4/user
   ```

**Problem**: "Insufficient permissions" error

**Solutions**:
1. Verify token has all required scopes:
   - `api` ✅
   - `read_repository` ✅
   - `write_repository` ✅
2. Recreate token with correct scopes
3. Reconnect vTeam with new token

---

### Repository Configuration Issues

**Problem**: Provider not auto-detected

**Solutions**:
1. Verify URL contains `gitlab.com` or matches GitLab pattern
2. Manually specify provider in ProjectSettings:
   ```yaml
   spec:
     repositories:
       - url: "https://gitlab.company.com/team/repo.git"
         provider: "gitlab"
   ```

**Problem**: Repository validation fails

**Solutions**:
1. Check you're connected to GitLab (`/auth/gitlab/status`)
2. Verify you have access to repository (try cloning manually)
3. For private repos: Ensure you're a member with Developer+ role
4. Check repository URL is correct

---

### AgenticSession Issues

**Problem**: Session fails to clone repository

**Solutions**:
1. Verify GitLab account is connected
2. Check token stored in secret:
   ```bash
   kubectl get secret gitlab-user-tokens -n vteam-backend
   ```
3. Verify repository URL is correct
4. Check session logs:
   ```bash
   kubectl logs <session-pod> -n <project-namespace>
   ```

**Problem**: Clone succeeds but push fails (403 Forbidden)

**Solutions**:
1. Token lacks `write_repository` scope → Recreate token
2. You don't have push access → Contact repo owner
3. Branch is protected → Use different branch or update protection rules

**Error Message**:
```
GitLab push failed: Insufficient permissions. Ensure your GitLab token
has 'write_repository' scope. You can update your token by reconnecting
your GitLab account with the required permissions.
```

**Problem**: Self-hosted GitLab URL not working

**Solutions**:
1. Verify instance URL format (must include `https://`)
2. Check API URL construction in logs
3. Test connectivity from backend pod
4. Verify SSL certificate is valid

---

## Limits & Quotas

### GitLab.com Rate Limits

**API Rate Limits** (GitLab.com):
- 300 requests per minute per user
- 10,000 requests per hour per user

**How vTeam Handles Rate Limits**:
- Errors returned with clear message
- Recommended wait time provided
- No automatic retry (to avoid making it worse)

**Error Message**:
```
GitLab API rate limit exceeded. Please wait a few minutes before
retrying. GitLab.com allows 300 requests per minute.
```

### Self-Hosted Rate Limits

Self-hosted instances may have different limits configured by admins. Check with your GitLab administrator.

---

## Mixing GitHub and GitLab

vTeam supports projects with both GitHub and GitLab repositories.

**Example Multi-Provider Project**:
```yaml
spec:
  repositories:
    - url: "https://github.com/team/frontend.git"
      branch: "main"
      provider: "github"

    - url: "https://gitlab.com/team/backend.git"
      branch: "develop"
      provider: "gitlab"

    - url: "https://gitlab.company.com/team/infrastructure.git"
      branch: "main"
      provider: "gitlab"
```

**How It Works**:
- Provider auto-detected from URL
- Correct authentication method used automatically:
  - GitHub: Uses GitHub App or GIT_TOKEN
  - GitLab: Uses GitLab PAT
- Each repo operates independently
- Errors are provider-specific and clear

**AgenticSession with Multiple Repos**:
- Session can work with multiple repos in one task
- Each repo cloned with appropriate authentication
- Changes pushed to correct providers

---

## API Reference

### Connect GitLab Account

```http
POST /api/auth/gitlab/connect
Content-Type: application/json
Authorization: Bearer <vteam-token>

{
  "personalAccessToken": "glpat-xxx",
  "instanceUrl": "https://gitlab.com"  # Optional, defaults to gitlab.com
}
```

**Response (200 OK)**:
```json
{
  "userId": "user-123",
  "gitlabUserId": "456789",
  "username": "yourname",
  "instanceUrl": "https://gitlab.com",
  "connected": true,
  "message": "GitLab account connected successfully"
}
```

---

### Get Connection Status

```http
GET /api/auth/gitlab/status
Authorization: Bearer <vteam-token>
```

**Response (200 OK - Connected)**:
```json
{
  "connected": true,
  "username": "yourname",
  "instanceUrl": "https://gitlab.com",
  "gitlabUserId": "456789"
}
```

**Response (200 OK - Not Connected)**:
```json
{
  "connected": false
}
```

---

### Disconnect GitLab Account

```http
POST /api/auth/gitlab/disconnect
Authorization: Bearer <vteam-token>
```

**Response (200 OK)**:
```json
{
  "message": "GitLab account disconnected successfully",
  "connected": false
}
```

---

## FAQ

**Q: Can I use the same token for multiple vTeam users?**
A: No. Each vTeam user should connect their own GitLab account with their own PAT. This ensures:
- Audit trail shows real user
- Correct permissions enforcement
- Individual token rotation

**Q: What happens if my token expires?**
A: AgenticSessions will fail with "Authentication failed" error. Reconnect with a new token.

**Q: Can I use SSH URLs?**
A: Yes, vTeam accepts SSH URLs and automatically converts them to HTTPS for authentication.

**Q: Do I need to configure SSH keys?**
A: No. vTeam uses HTTPS + Personal Access Token authentication exclusively.

**Q: Can I use Deploy Tokens instead of PATs?**
A: Not currently. Only Personal Access Tokens are supported.

**Q: Does vTeam support GitLab Groups/Subgroups?**
A: Yes. URLs like `https://gitlab.com/group/subgroup/project.git` work correctly.

**Q: What if I don't have a GitLab account?**
A: Create one at https://gitlab.com - it's free for public and private repositories.

**Q: Can I use vTeam with GitLab Enterprise?**
A: Yes. Self-hosted GitLab Enterprise Edition is fully supported.

**Q: How do I know if my token has the right scopes?**
A: Test it:
```bash
curl -H "Authorization: Bearer glpat-xxx" \
  https://gitlab.com/api/v4/personal_access_tokens/self
```

**Q: Will GitHub stop working after I add GitLab?**
A: No. GitHub and GitLab integrations are independent and work side-by-side.

---

## Support & Resources

**Documentation**:
- [GitLab PAT Setup Guide](./gitlab-token-setup.md)
- [Self-Hosted GitLab Configuration](./gitlab-self-hosted.md)
- [GitLab Testing Procedures](./gitlab-testing-procedures.md)

**GitLab Resources**:
- [Personal Access Tokens Documentation](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)
- [GitLab API Documentation](https://docs.gitlab.com/ee/api/)
- [GitLab Permissions](https://docs.gitlab.com/ee/user/permissions.html)

**Troubleshooting**:
- Check backend logs: `kubectl logs -l app=vteam-backend -n vteam-backend`
- Check session logs: `kubectl logs <session-pod> -n <project-namespace>`
- Verify GitLab status: https://status.gitlab.com (for GitLab.com)

**Getting Help**:
- vTeam GitHub Issues: [Create an issue](https://github.com/natifridman/vTeam/issues)
- vTeam Documentation: [Main README](../README.md)

---

## Changelog

**v1.1.0** - 2025-11-05
- ✨ Initial GitLab integration release
- ✅ GitLab.com support
- ✅ Self-hosted GitLab support
- ✅ Personal Access Token authentication
- ✅ AgenticSession clone/commit/push operations
- ✅ Multi-provider support (GitHub + GitLab)
