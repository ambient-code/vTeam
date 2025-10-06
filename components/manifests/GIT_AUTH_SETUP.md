# Git Authentication Setup

vTeam supports **two independent git authentication methods** that serve different purposes:

1. **GitHub App**: Backend OAuth login + Repository browser in UI
2. **Project-level Git Secrets**: Runner git operations (clone, commit, push)

You can use **either one or both** - the system gracefully handles all scenarios.

## Project-Level Git Authentication

This approach allows each project to have its own Git credentials, similar to how `ANTHROPIC_API_KEY` is configured.

### Setup: Using GitHub API Token

**1. Create a secret with a GitHub token:**

```bash
# Create secret with GitHub personal access token
oc create secret generic my-runner-secret \
  --from-literal=ANTHROPIC_API_KEY="your-anthropic-api-key" \
  --from-literal=GIT_USER_NAME="Your Name" \
  --from-literal=GIT_USER_EMAIL="your.email@example.com" \
  --from-literal=GIT_TOKEN="ghp_your_github_token" \
  -n your-project-namespace
```

**2. Reference the secret in your ProjectSettings:**

(Most users will access this from the frontend)

```yaml
apiVersion: vteam.ambient-code/v1
kind: ProjectSettings
metadata:
  name: my-project
  namespace: your-project-namespace
spec:
  runnerSecret: my-runner-secret
```

**3. Use HTTPS URLs in your AgenticSession:**

(Most users will access this from the frontend)

```yaml
spec:
  repos:
    - input:
        url: "https://github.com/your-org/your-repo.git"
        branch: "main"
```

The runner will automatically use your `GIT_TOKEN` for authentication.

---

## GitHub App Authentication (Optional - For Backend OAuth)

**Purpose**: Enables GitHub OAuth login and repository browsing in the UI

**Who configures it**: Platform administrators (cluster-wide)

**What it provides**:
- GitHub OAuth login for users
- Repository browser in the UI (`/auth/github/repos/...`)
- PR creation via backend API

**Setup**:

Edit `github-app-secret.yaml` with your GitHub App credentials:

```bash
# Fill in your GitHub App details
vim github-app-secret.yaml

# Apply to the cluster namespace
oc apply -f github-app-secret.yaml -n ambient-code
```

**What happens if NOT configured**:
- ✅ Backend starts normally (prints warning: "GitHub App not configured")
- ✅ Runner git operations still work (via project-level secrets)
- ❌ GitHub OAuth login unavailable
- ❌ Repository browser endpoints return "GitHub App not configured"
- ✅ Everything else works fine!

---

## Using Both Methods Together (Recommended)

**Best practice setup**:

1. **Platform admin**: Configure GitHub App for OAuth login
2. **Each user**: Create their own project-level git secret for runner operations

This provides:
- ✅ GitHub SSO login (via GitHub App)
- ✅ Repository browsing in UI (via GitHub App)
- ✅ Isolated git credentials per project (via project secrets)
- ✅ Different tokens per team/project
- ✅ No shared credentials

**Example workflow**:
```bash
# 1. User logs in via GitHub App OAuth
# 2. User creates their project with their own git secret
oc create secret generic my-runner-secret \
  --from-literal=ANTHROPIC_API_KEY="..." \
  --from-literal=GIT_TOKEN="ghp_your_project_token" \
  -n my-project

# 3. Runner uses the project's GIT_TOKEN for git operations
# 4. Backend uses GitHub App for UI features
```

---

## How It Works

1. **ProjectSettings CR**: References a secret name in `spec.runnerSecretsName`
2. **Operator**: Injects all secret keys as environment variables via `EnvFrom`
3. **Runner**: Checks `GIT_TOKEN` → `GITHUB_TOKEN` → (no auth)
4. **Backend**: Creates per-session secret with GitHub App token (if configured)

## Decision Matrix

| Setup | GitHub App | Project Secret | Git Clone Works? | OAuth Login? |
|-------|-----------|----------------|------------------|--------------|
| None | ❌ | ❌ | ❌ (public only) | ❌ |
| App Only | ✅ | ❌ | ✅ (if user linked) | ✅ |
| Secret Only | ❌ | ✅ | ✅ (always) | ❌ |
| Both | ✅ | ✅ | ✅ (prefers secret) | ✅ |

## Authentication Priority (Runner)

When cloning/pushing repos, the runner checks for credentials in this order:

1. **GIT_TOKEN** (from project runner secret) - Preferred for most deployments
2. **GITHUB_TOKEN** (from per-session secret, if GitHub App configured)
3. **No credentials** - Only works with public repos, no git pushing

**How it works:**
- Backend creates `ambient-runner-token-{sessionName}` secret with GitHub App installation token (if user linked GitHub)
- Operator must mount this secret and expose as `GITHUB_TOKEN` env var
- Runner prefers project-level `GIT_TOKEN` over per-session `GITHUB_TOKEN`
