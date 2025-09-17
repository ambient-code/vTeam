# Git Integration Guide

The vTeam platform provides comprehensive Git integration for agentic sessions, enabling secure access to private repositories, automatic repository cloning, and full Git workflow capabilities including commits and pushes.

## Overview

Git integration allows your agentic sessions to:
- **Clone private repositories** using SSH keys or access tokens
- **Configure Git user identity** for commits and pushes
- **Work with multiple repositories** simultaneously
- **Create branches and push changes** back to remote repositories
- **Use spek-kit with proper Git workflow** for spec-driven development

## Configuration Options

### User Configuration

Configure the Git user identity for commits:

```yaml
apiVersion: vteam.ambient-code/v1
kind: AgenticSession
metadata:
  name: my-session
spec:
  prompt: "Your agentic task here"
  gitConfig:
    user:
      name: "Your Name"
      email: "your.email@company.com"
```

### Authentication Methods

#### Option 1: SSH Key Authentication (Recommended)

Best for private repositories and most secure:

```yaml
gitConfig:
  authentication:
    sshKeySecret: "my-ssh-key-secret"
    knownHostsSecret: "my-known-hosts-secret"  # Optional but recommended
```

#### Option 2: Personal Access Token

Good for HTTPS-based authentication:

```yaml
gitConfig:
  authentication:
    tokenSecret: "my-github-token-secret"
```

### Repository Configuration

Clone and work with specific repositories:

```yaml
gitConfig:
  repositories:
    - url: "git@github.com:company/repo1.git"
      branch: "main"                    # Optional, defaults to "main"
      clonePath: "project1"            # Optional, defaults to repo name
    - url: "https://github.com/company/repo2.git"
      branch: "develop"
      clonePath: "project2"
```

## Secret Setup

### SSH Key Authentication

#### 1. Create SSH Key Secret

```bash
# Create SSH key secret with your private key
oc create secret generic my-ssh-key \
  --from-file=id_rsa=$HOME/.ssh/id_rsa \
  -n your-namespace

# Alternatively, create from literal if you have the key content
oc create secret generic my-ssh-key \
  --from-literal=id_rsa="$(cat $HOME/.ssh/id_rsa)" \
  -n your-namespace
```

#### 2. Create Known Hosts Secret (Recommended)

```bash
# Create known_hosts secret for SSH host verification
oc create secret generic my-known-hosts \
  --from-file=known_hosts=$HOME/.ssh/known_hosts \
  -n your-namespace

# Or create a minimal known_hosts for GitHub/GitLab
cat > /tmp/known_hosts << EOF
github.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7hr3oQpqHPQ==
gitlab.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDVz3j6rKqE==
EOF

oc create secret generic my-known-hosts \
  --from-file=known_hosts=/tmp/known_hosts \
  -n your-namespace
```

#### 3. Verification

```bash
# Verify secrets were created
oc get secrets | grep ssh
oc get secret my-ssh-key -o yaml
```

### Personal Access Token Authentication

#### 1. Create Token Secret

```bash
# GitHub Personal Access Token
oc create secret generic my-github-token \
  --from-literal=token="ghp_your_github_token_here" \
  -n your-namespace

# GitLab Personal Access Token
oc create secret generic my-gitlab-token \
  --from-literal=token="glpat-your_gitlab_token_here" \
  -n your-namespace
```

#### 2. Token Permissions

Ensure your token has the necessary permissions:

**GitHub:**
- `repo` (for private repositories)
- `workflow` (if working with GitHub Actions)

**GitLab:**
- `read_repository`
- `write_repository` (for pushing changes)

## Complete Examples

### Example 1: Basic Git Configuration

```yaml
apiVersion: vteam.ambient-code/v1
kind: AgenticSession
metadata:
  name: basic-git-session
spec:
  prompt: "/specify Create a REST API for user management"
  gitConfig:
    user:
      name: "Jane Developer"
      email: "jane@company.com"
```

### Example 2: Private Repository with SSH

```yaml
apiVersion: vteam.ambient-code/v1
kind: AgenticSession
metadata:
  name: private-repo-session
spec:
  prompt: "/plan Implement OAuth2 authentication using existing patterns"
  gitConfig:
    user:
      name: "Jane Developer"
      email: "jane@company.com"
    authentication:
      sshKeySecret: "company-ssh-key"
      knownHostsSecret: "company-known-hosts"
    repositories:
      - url: "git@github.com:company/auth-service.git"
        branch: "main"
        clonePath: "auth-service"
```

### Example 3: Multiple Repositories with Token

```yaml
apiVersion: vteam.ambient-code/v1
kind: AgenticSession
metadata:
  name: multi-repo-analysis
spec:
  prompt: "/tasks Analyze microservices architecture across our services"
  gitConfig:
    user:
      name: "Architecture Team"
      email: "architecture@company.com"
    authentication:
      tokenSecret: "github-analysis-token"
    repositories:
      - url: "https://github.com/company/user-service.git"
        branch: "main"
        clonePath: "users"
      - url: "https://github.com/company/order-service.git"
        branch: "main"
        clonePath: "orders"
      - url: "https://github.com/company/shared-libraries.git"
        branch: "develop"
        clonePath: "shared"
```

### Example 4: Spek-kit with Git Push Workflow

```yaml
apiVersion: vteam.ambient-code/v1
kind: AgenticSession
metadata:
  name: feature-development
spec:
  prompt: "/specify Build a payment processing API with webhook support"
  gitConfig:
    user:
      name: "Payment Team"
      email: "payments@company.com"
    authentication:
      sshKeySecret: "payment-team-ssh"
      knownHostsSecret: "company-known-hosts"
    repositories:
      - url: "git@github.com:company/payment-service.git"
        branch: "main"
        clonePath: "payment-api"
```

## Git Workflow in Agentic Sessions

When Git is configured, your agentic sessions can:

### 1. **Automatic Repository Cloning**
- Repositories are cloned to `/tmp/git-workspace/`
- Each repository is available at the specified `clonePath`
- Claude Code can analyze existing code and patterns

### 2. **Spek-kit Integration**
- Generated specifications are created in proper Git context
- Branches can be created for new features
- Changes can be committed and pushed automatically

### 3. **Code Generation and Commits**
- Generated code can be committed with proper Git identity
- Branch creation for feature development
- Automatic pushing of changes to remote repositories

## Environment Variables Available

When Git is configured, these environment variables are available in your agentic session:

```bash
GIT_USER_NAME="Your Name"
GIT_USER_EMAIL="your@email.com"
GIT_REPOSITORIES='[{"url":"...","branch":"main","clonePath":"..."}]'
```

## Security Best Practices

### SSH Key Management
- **Use dedicated deploy keys** for repository access
- **Rotate SSH keys regularly**
- **Limit key permissions** to specific repositories
- **Never commit private keys** to repositories

### Token Management
- **Use fine-grained personal access tokens** when available
- **Set appropriate token expiration**
- **Limit token scope** to minimum required permissions
- **Regularly audit and rotate tokens**

### Secret Management
- **Use Kubernetes RBAC** to control secret access
- **Regularly rotate secrets**
- **Monitor secret usage** in audit logs
- **Use separate secrets** for different environments

## Troubleshooting

### Common Issues

#### SSH Key Not Working
```bash
# Verify secret exists and has correct key
oc get secret my-ssh-key -o jsonpath='{.data.id_rsa}' | base64 -d | head -1
# Should show: -----BEGIN PRIVATE KEY----- or -----BEGIN RSA PRIVATE KEY-----

# Check SSH key permissions in pod
oc exec -it <pod-name> -- ls -la /tmp/.ssh/
```

#### Token Authentication Failing
```bash
# Verify token secret
oc get secret my-github-token -o jsonpath='{.data.token}' | base64 -d

# Test token manually
curl -H "Authorization: token YOUR_TOKEN" https://api.github.com/user
```

#### Repository Clone Failing
```bash
# Check pod logs for specific error
oc logs <agentic-session-job-pod>

# Common issues:
# - Wrong repository URL
# - Insufficient permissions
# - Network connectivity
# - SSH key not configured for repository
```

### Debug Commands

```bash
# View agentic session status
oc get agenticsession my-session -o yaml

# Check job logs
oc logs -l agentic-session=my-session

# Debug inside running pod
oc exec -it <pod-name> -- bash
git config --list
ssh -T git@github.com  # Test SSH connection
```

## Integration with Deploy Script

The Git configuration works seamlessly with the existing deployment:

```bash
# Deploy with custom images and Git support
NAMESPACE=my-namespace \
DEFAULT_RUNNER_IMAGE=quay.io/myorg/vteam:claude-runner-git \
IMAGE_PULL_POLICY=Always \
./deploy.sh
```

## Next Steps

1. **Set up your Git secrets** using the commands above
2. **Test with a simple AgenticSession** to verify configuration
3. **Use spek-kit commands** to generate specifications with Git workflow
4. **Monitor logs** to ensure Git operations are working correctly

For additional support, check the agentic session logs and verify your secret configuration.