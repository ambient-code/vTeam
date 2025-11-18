# GitLab Personal Access Token Setup Guide

This guide provides step-by-step instructions for creating a GitLab Personal Access Token (PAT) for use with vTeam.

## Overview

**What is a Personal Access Token?**
A Personal Access Token (PAT) is a secure credential that allows vTeam to access your GitLab repositories on your behalf without needing your password.

**Why do I need one?**
vTeam uses your PAT to:
- Validate your access to repositories
- Clone repositories for AgenticSessions
- Commit and push changes to your GitLab repositories

**Security Note**: Your token is stored securely in Kubernetes Secrets and never logged in plaintext.

---

## Creating a GitLab Personal Access Token

### For GitLab.com

#### Step 1: Log In to GitLab

1. Open your browser and navigate to: **https://gitlab.com**
2. Sign in with your GitLab credentials
3. If you don't have an account, click "Register" to create one (free for public and private repositories)

---

#### Step 2: Navigate to Access Tokens Page

**Option A - Via Profile Menu**:
1. Click your **profile icon** in the top-right corner
2. Select **"Preferences"** from the dropdown menu
3. In the left sidebar, click **"Access Tokens"**

**Option B - Direct Link**:
1. Navigate directly to: https://gitlab.com/-/profile/personal_access_tokens

---

#### Step 3: Create New Token

On the Personal Access Tokens page, you'll see a form to create a new token:

**1. Token Name**
- Enter: `vTeam Integration` (or any descriptive name)
- This helps you identify the token later

**2. Expiration Date**
- **Recommended**: Set 90 days from today
- **Maximum**: GitLab allows up to 1 year
- **Important**: You'll need to create a new token and reconnect vTeam before expiration

**3. Select Scopes** (IMPORTANT - must select all of these):

Check the following scopes:

- ✅ **`api`** - Full API access
  - *Required*: Allows vTeam to access GitLab API endpoints
  - Grants read and write access to repositories, merge requests, etc.

- ✅ **`read_api`** - Read API
  - *Required*: Allows read-only access to API
  - Used for validation and repository browsing

- ✅ **`read_user`** - Read user information
  - *Required*: Allows vTeam to verify your identity
  - Used to get your GitLab username and user ID

- ✅ **`write_repository`** - Write to repository
  - *Required*: Allows vTeam to push changes
  - Essential for AgenticSessions to commit and push code

**DO NOT SELECT** (not needed, grants excessive privileges):
- ❌ `sudo` - Admin-level access
- ❌ `admin_mode` - Administrative operations
- ❌ `create_runner` - Register CI runners
- ❌ `manage_runner` - Manage CI runners

**4. Click "Create personal access token"**

---

#### Step 4: Copy Your Token

**CRITICAL STEP** - This is your only chance to copy the token!

1. After clicking "Create", GitLab will display your new token
2. The token starts with **`glpat-`** followed by random characters
   - Example: `glpat-xyz123abc456def789ghi012`

3. **Copy the entire token** to your clipboard
   - Click the copy icon next to the token
   - Or select all text and copy manually

4. **Save the token securely**:
   - Paste into a password manager (recommended)
   - Or save to a secure text file temporarily
   - **DO NOT** share the token or commit it to git

**Warning**: GitLab will NOT show this token again. If you lose it, you must create a new token.

---

### For Self-Hosted GitLab

The process is identical to GitLab.com, with these differences:

#### Step 1: Access Your GitLab Instance

1. Navigate to your organization's GitLab URL
   - Example: `https://gitlab.company.com`
2. Sign in with your corporate credentials

#### Step 2: Navigate to Access Tokens

The location depends on your GitLab version:

**GitLab 14.0+**:
- Click profile icon → Preferences → Access Tokens

**GitLab 13.x**:
- Click profile icon → Settings → Access Tokens

**Direct URL**:
- `https://gitlab.company.com/-/profile/personal_access_tokens`
- (Replace `gitlab.company.com` with your instance)

#### Step 3: Create Token (Same as GitLab.com)

Follow Steps 3-4 from the GitLab.com instructions above.

**Important Notes for Self-Hosted**:
- Expiration policy may be enforced by your administrator
- Some scopes may be restricted by your admin
- Contact your GitLab administrator if you encounter permission issues
- Your instance may use different authentication (LDAP, SAML, etc.) but PAT creation is the same

---

## Verifying Your Token

Before using the token with vTeam, verify it works:

### Using cURL (Command Line)

```bash
# Test token validity
curl -H "Authorization: Bearer glpat-your-token-here" \
  https://gitlab.com/api/v4/user

# For self-hosted:
curl -H "Authorization: Bearer glpat-your-token-here" \
  https://gitlab.company.com/api/v4/user
```

**Expected Response** (200 OK):
```json
{
  "id": 123456,
  "username": "yourname",
  "name": "Your Name",
  "state": "active",
  "avatar_url": "...",
  "web_url": "https://gitlab.com/yourname"
}
```

**Error Responses**:

**401 Unauthorized**:
```json
{
  "message": "401 Unauthorized"
}
```
- Token is invalid or expired
- Create a new token

**403 Forbidden**:
```json
{
  "message": "403 Forbidden"
}
```
- Token lacks required scopes
- Recreate token with `api`, `read_api`, `read_user`, `write_repository`

---

### Verify Token Scopes

```bash
# Check token scopes (GitLab API)
curl -H "Authorization: Bearer glpat-your-token-here" \
  https://gitlab.com/api/v4/personal_access_tokens/self
```

**Expected Response**:
```json
{
  "id": 123456,
  "name": "vTeam Integration",
  "revoked": false,
  "created_at": "2025-11-05T10:00:00.000Z",
  "scopes": ["api", "read_api", "read_user", "write_repository"],
  "user_id": 789,
  "active": true,
  "expires_at": "2026-02-05"
}
```

**Verify**:
- `"revoked": false` - Token is active
- `"active": true` - Token is not expired
- `"scopes"` includes all required: `api`, `read_api`, `read_user`, `write_repository`

---

### Verify Repository Access

Test access to a specific repository:

```bash
# Replace owner/repo with your repository
curl -H "Authorization: Bearer glpat-your-token-here" \
  https://gitlab.com/api/v4/projects/owner%2Frepo

# Example:
curl -H "Authorization: Bearer glpat-xxx" \
  https://gitlab.com/api/v4/projects/myteam%2Fmyproject
```

**Expected Response** (200 OK):
```json
{
  "id": 12345,
  "name": "myproject",
  "path": "myproject",
  "path_with_namespace": "myteam/myproject",
  "permissions": {
    "project_access": {
      "access_level": 30,
      "notification_level": 3
    }
  }
}
```

**Access Levels**:
- `10` = Guest (❌ cannot push)
- `20` = Reporter (❌ cannot push)
- `30` = Developer (✅ can push)
- `40` = Maintainer (✅ can push)
- `50` = Owner (✅ can push)

**Minimum Required**: `30` (Developer) for AgenticSessions

---

## Using Your Token with vTeam

Once you have your token, connect it to vTeam:

### Via vTeam UI

1. Navigate to **Settings** → **Integrations**
2. Find **GitLab** section
3. Click **"Connect GitLab"** button
4. Paste your token in the **Personal Access Token** field
5. (Optional) For self-hosted: Enter **Instance URL**
   - Example: `https://gitlab.company.com`
6. Click **"Connect"**
7. Wait for success confirmation

### Via API (Command Line)

**For GitLab.com**:
```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-vteam-auth-token>" \
  -d '{
    "personalAccessToken": "glpat-your-gitlab-token-here",
    "instanceUrl": ""
  }'
```

**For Self-Hosted GitLab**:
```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-vteam-auth-token>" \
  -d '{
    "personalAccessToken": "glpat-your-gitlab-token-here",
    "instanceUrl": "https://gitlab.company.com"
  }'
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

## Token Management

### Viewing Your Tokens

**In GitLab**:
1. Navigate to: https://gitlab.com/-/profile/personal_access_tokens
2. Scroll down to **"Active Personal Access Tokens"**
3. You'll see a table with all your tokens:
   - Token name
   - Scopes
   - Created date
   - Last used date
   - Expiration date

**Note**: GitLab shows when a token was last used, helping you identify unused tokens.

---

### Revoking a Token

**When to Revoke**:
- Token compromised or accidentally exposed
- Token no longer needed
- Replacing with new token (after rotating)

**How to Revoke**:
1. Navigate to: https://gitlab.com/-/profile/personal_access_tokens
2. Find the token in the **"Active Personal Access Tokens"** table
3. Click the **"Revoke"** button next to the token
4. Confirm revocation

**Important**:
- Revoked tokens CANNOT be un-revoked
- Any application using the token will immediately lose access
- If you revoked the wrong token, create a new one

---

### Rotating Tokens (Recommended Every 90 Days)

Token rotation improves security by limiting exposure if a token is compromised.

**Rotation Process**:

1. **Create New Token**:
   - Follow steps above to create new token
   - Use same name with date: `vTeam Integration (Nov 2025)`
   - Select same scopes

2. **Test New Token**:
   ```bash
   curl -H "Authorization: Bearer glpat-new-token" \
     https://gitlab.com/api/v4/user
   ```

3. **Update vTeam**:
   - Disconnect current GitLab connection in vTeam
   - Reconnect with new token

4. **Verify vTeam Works**:
   - Check connection status in vTeam
   - Test with a simple AgenticSession

5. **Revoke Old Token**:
   - Go to GitLab Access Tokens page
   - Revoke the old token

**Set a Reminder**: Add calendar reminder 7 days before token expiration.

---

## Troubleshooting

### Token Not Working with vTeam

**Problem**: vTeam shows "Invalid token" error

**Solutions**:
1. **Verify token copied correctly**:
   - No extra spaces before/after
   - Entire token including `glpat-` prefix
   - Check for line breaks if copy-pasted from email

2. **Check token hasn't expired**:
   - Go to GitLab Access Tokens page
   - Check expiration date
   - Create new token if expired

3. **Verify token is active**:
   ```bash
   curl -H "Authorization: Bearer glpat-xxx" \
     https://gitlab.com/api/v4/personal_access_tokens/self
   ```
   - Check `"active": true` and `"revoked": false`

4. **For self-hosted**: Verify instance URL is correct
   - Must include `https://`
   - No trailing slash
   - Example: `https://gitlab.company.com`

---

### Insufficient Permissions Error

**Problem**: vTeam shows "Insufficient permissions" when pushing

**Solutions**:
1. **Check token scopes**:
   ```bash
   curl -H "Authorization: Bearer glpat-xxx" \
     https://gitlab.com/api/v4/personal_access_tokens/self
   ```

2. **Verify all required scopes**:
   - ✅ `api`
   - ✅ `read_api`
   - ✅ `read_user`
   - ✅ `write_repository` ← Often missing!

3. **Recreate token with correct scopes**:
   - Create new token with all scopes
   - Update vTeam connection
   - Revoke old token

4. **Check repository access**:
   - Verify you're at least Developer on the repository
   - For private repos: Check you're a member

---

### Rate Limit Exceeded

**Problem**: "Rate limit exceeded" error

**Cause**: GitLab.com limits:
- 300 requests per minute per user
- 10,000 requests per hour per user

**Solutions**:
1. **Wait**: Limits reset after the time window (1 minute or 1 hour)
2. **Check for loops**: Ensure no automated processes hammering API
3. **For self-hosted**: Contact admin about rate limit configuration

---

### Token Revoked Unexpectedly

**Possible Causes**:
1. **You revoked it**: Check GitLab audit log
2. **Admin revoked it**: Self-hosted instances allow admin token revocation
3. **Token expired**: Check expiration date
4. **Account issue**: Account suspended or password changed on some GitLab versions

**Solutions**:
- Create new token
- Contact GitLab admin (for self-hosted)
- Check GitLab account status

---

## Security Best Practices

### DO ✅

1. **Set Expiration Dates**
   - Always set an expiration (max 90 days recommended)
   - Prevents perpetual access if token compromised

2. **Use Minimum Required Scopes**
   - Only select: `api`, `read_api`, `read_user`, `write_repository`
   - Avoid `sudo` and `admin_mode`

3. **Store Tokens Securely**
   - Use password manager (1Password, LastPass, etc.)
   - Or secure corporate vault
   - Never in git repositories

4. **Rotate Regularly**
   - Every 90 days recommended
   - Immediately if compromised

5. **Use Separate Tokens**
   - Different token for vTeam vs other applications
   - Easier to identify in audit logs
   - Can revoke individually

6. **Monitor Last Used Date**
   - Check GitLab Access Tokens page monthly
   - Revoke unused tokens

### DON'T ❌

1. **Never Commit Tokens to Git**
   ```bash
   # BAD - token exposed in git history!
   git commit -m "Added token glpat-xxx to config"
   ```

2. **Never Share Tokens**
   - Each user should have their own token
   - Team members need individual vTeam connections

3. **Never Use Sudo Scope**
   - Grants excessive admin privileges
   - Not needed for vTeam

4. **Never Set "No Expiration"**
   - Security risk if token leaks
   - Always set expiration date

5. **Never Log Tokens**
   - Don't print tokens in application logs
   - Don't include in error messages
   - vTeam automatically redacts tokens

6. **Never Hardcode Tokens**
   ```python
   # BAD - token in source code!
   gitlab_token = "glpat-xyz123abc456"
   ```

---

## FAQ

**Q: How long should my token's expiration be?**
A: **90 days** is recommended. This balances security (shorter is better) with convenience (longer reduces rotation overhead).

**Q: What if I lose my token?**
A: Create a new token and update vTeam. You cannot retrieve a lost token - GitLab only shows it once during creation.

**Q: Can I use the same token for multiple vTeam projects?**
A: Yes, one token works for all vTeam projects under your user account.

**Q: Can multiple team members share one token?**
A: **No**. Each person should create their own token and connect individually to vTeam. This ensures proper audit trails.

**Q: What's the difference between `api` and `write_repository` scopes?**
A: `api` grants full API access (read + write). `write_repository` specifically grants push access to git repositories. Both are needed.

**Q: Do I need to create a new token for each repository?**
A: No. One token works for all repositories you have access to.

**Q: What happens when my token expires?**
A: AgenticSessions will fail with "Authentication failed" error. Create a new token and reconnect to vTeam.

**Q: Can I extend a token's expiration date?**
A: No. You must create a new token with a new expiration date.

**Q: How do I know if my token was compromised?**
A: Check "Last Used" date in GitLab. If it shows activity you didn't perform, revoke immediately and create new token.

**Q: Can administrators see my token?**
A: No. GitLab doesn't show token values to anyone, including admins. However, admins can revoke tokens on self-hosted instances.

**Q: What's the difference between Personal Access Token and Deploy Token?**
A: Personal Access Tokens are tied to your user account. Deploy Tokens are scoped to specific projects and have limited permissions. vTeam requires Personal Access Tokens.

**Q: Can I use OAuth instead of PAT?**
A: Not currently. vTeam only supports Personal Access Token authentication for GitLab.

---

## Additional Resources

**GitLab Official Documentation**:
- [Personal Access Tokens](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)
- [GitLab API Authentication](https://docs.gitlab.com/ee/api/index.html#authentication)
- [Token Security](https://docs.gitlab.com/ee/security/token_overview.html)

**vTeam Documentation**:
- [GitLab Integration Guide](./gitlab-integration.md)
- [Self-Hosted GitLab Configuration](./gitlab-self-hosted.md)
- [Troubleshooting Guide](./gitlab-integration.md#troubleshooting)

**Security Resources**:
- [GitLab Security Best Practices](https://docs.gitlab.com/ee/security/)
- [OWASP Token Management](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)

---

## Support

Need help with token creation?

**For GitLab.com Issues**:
- GitLab Support: https://about.gitlab.com/support/
- GitLab Forum: https://forum.gitlab.com/

**For Self-Hosted GitLab**:
- Contact your GitLab administrator
- Check your organization's GitLab documentation

**For vTeam Integration Issues**:
- vTeam GitHub Issues: https://github.com/natifridman/vTeam/issues
- vTeam Documentation: [Main README](../README.md)

---

## Quick Reference

**Required Token Scopes**:
```
✅ api
✅ read_api
✅ read_user
✅ write_repository
```

**Token Format**:
```
glpat-xxxxxxxxxxxxxxxx
```

**Test Token**:
```bash
curl -H "Authorization: Bearer glpat-xxx" \
  https://gitlab.com/api/v4/user
```

**Connect to vTeam**:
```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <vteam-token>" \
  -d '{"personalAccessToken":"glpat-xxx","instanceUrl":""}'
```

**Check vTeam Connection**:
```bash
curl -X GET http://vteam-backend:8080/api/auth/gitlab/status \
  -H "Authorization: Bearer <vteam-token>"
```
