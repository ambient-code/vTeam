# RFE Operator

## Overview

The RFE (Requirements Feature Engineering) Operator manages AI-powered agentic sessions in Kubernetes. It watches for `AgenticSession` resources and creates corresponding jobs that run AI agents with access to git repositories and workspace storage.

## Architecture

- **Resources Package**: Modular reconcilers for secrets and ConfigMaps
- **Namespace Isolation**: Each RFE controller instance uses `rfe-controller-` prefixed resources
- **Multi-tenant**: Supports multiple managed namespaces with isolated workspaces

## Required Resources

### Secret: `rfe-controller-secrets`
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rfe-controller-secrets
data:
  ANTHROPIC_API_KEY: <base64-encoded-api-key>  # Required
  GITHUB_TOKEN: <base64-encoded-token>         # Optional - needed for private repos, push operations, API calls
  GIT_TOKEN: <base64-encoded-token>            # Optional - alternative for non-GitHub providers
  GIT_SSH_KEY: <base64-encoded-ssh-key>        # Optional - for SSH-based auth
```

**Note**: Git auth only required for private repositories and write operations. Public repo read access works without authentication.

### Managed Namespaces
Namespaces must have label: `ambient-code.io/managed=true`

## Deployment

1. **Build the operator:**
   ```bash
   go build -o operator main.go
   ```

2. **Set environment variables:**
   ```bash
   export NAMESPACE=default
   export SECRETS_TO_COPY=rfe-controller-secrets
   export AMBIENT_CODE_RUNNER_IMAGE=quay.io/ambient_code/vteam_claude_runner:latest
   ```

3. **Run the operator:**
   ```bash
   ./operator
   ```

## Watched Resources

- **AgenticSession**: Creates jobs for AI agent execution
- **ProjectSettings**: Manages per-namespace configuration
- **Namespaces**: Auto-provisions resources for managed namespaces

## Created Resources

Per managed namespace:
- `rfe-controller-secrets` (copied from source)
- `rfe-controller-git-config` ConfigMap
- `rfe-controller-workspace` PVC
- `rfe-controller-content` Service/Deployment

## Configuration

### Environment Variables
- `NAMESPACE`: Operator namespace (default: `default`)
- `SECRETS_SOURCE_NAMESPACE`: Source for secret copying (default: operator namespace)
- `SECRETS_TO_COPY`: Comma-separated secret names (default: `rfe-controller-secrets`)
- `AMBIENT_CODE_RUNNER_IMAGE`: Runner container image
- `CONTENT_SERVICE_IMAGE`: Content service image
- `IMAGE_PULL_POLICY`: Container image pull policy (default: `Always`)

### Custom Resources
- `AgenticSession` (vteam.ambient-code/v1alpha1)
- `ProjectSettings` (vteam.ambient-code/v1alpha1)

## Operation

1. Label namespace with `ambient-code.io/managed=true`
2. Operator copies secrets and creates default resources
3. Create `AgenticSession` resources in managed namespaces
4. Operator creates Kubernetes jobs that run AI agents
5. Jobs mount shared workspace and have access to git configuration