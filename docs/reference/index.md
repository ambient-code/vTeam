# Reference Documentation

This section provides comprehensive reference material for the Ambient Code Platform, including API documentation, Custom Resource specifications, and configuration details.

## Quick Reference

### **[Glossary](glossary.md)** 📖
Definitions of terms, concepts, and acronyms used throughout the Ambient Code Platform system and documentation.

## Custom Resources

The platform uses Kubernetes Custom Resource Definitions (CRDs) for declarative automation management.

### AgenticSession

The primary Custom Resource for AI-powered automation tasks.

**API Version**: `vteam.ambient-code/v1alpha1`
**Kind**: `AgenticSession`

**Key Spec Fields:**

- `prompt`: The task description for the AI agent (string, required)
- `repos`: Array of repository configurations for input/output (required)
  - `input`: Source repository configuration (url, branch, ref)
  - `output`: Target repository for changes (optional fork configuration)
- `interactive`: Boolean for chat mode vs headless execution (default: false)
- `timeout`: Maximum execution time in seconds (default: 3600)
- `model`: Claude model to use (e.g., "claude-sonnet-4")
- `mainRepoIndex`: Which repo is the Claude working directory (default: 0)

**Status Fields:**

- `phase`: Current state (Pending, Running, Completed, Failed, Error)
- `startTime`: When execution began (RFC3339 timestamp)
- `completionTime`: When execution finished (RFC3339 timestamp)
- `results`: Summary of session output
- `message`: Human-readable status message
- `repos`: Per-repository status (pushed or abandoned)

**Example AgenticSession:**

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: analyze-codebase
  namespace: my-project
spec:
  prompt: "Analyze this repository and generate comprehensive API documentation"
  repos:
    - input:
        url: https://github.com/myorg/myrepo
        branch: main
      output:
        targetBranch: docs-update
  interactive: false
  timeout: 3600
  model: "claude-sonnet-4"
```

### ProjectSettings

Namespace-scoped configuration for platform projects, managing API keys, access control, and default settings.

**API Version**: `vteam.ambient-code/v1alpha1`
**Kind**: `ProjectSettings`

**Key Spec Fields:**

- `groupAccess`: Array of group permissions for multi-user access
  - `groupName`: OpenShift group name
  - `role`: Access level (view, edit, admin)
- `runnerSecretsName`: Reference to Secret containing API keys (default: "runner-secrets")

**Example ProjectSettings with Secret:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: runner-secrets
  namespace: my-project
type: Opaque
stringData:
  ANTHROPIC_API_KEY: "sk-ant-api03-your-key-here"
---
apiVersion: vteam.ambient-code/v1alpha1
kind: ProjectSettings
metadata:
  name: projectsettings
  namespace: my-project
spec:
  groupAccess:
    - groupName: "developers"
      role: "edit"
    - groupName: "viewers"
      role: "view"
  runnerSecretsName: "runner-secrets"
```

### RFEWorkflow

Specialized Custom Resource for Request for Enhancement workflows using a 7-agent council process. This is an advanced feature for structured engineering refinement.

**API Version**: `vteam.ambient-code/v1alpha1`
**Kind**: `RFEWorkflow`

This is an advanced feature not covered in the standard user documentation. For implementation details, see the project's AGENTS.md file in the repository root.

## REST API Endpoints

The backend API provides HTTP endpoints for managing projects and sessions.

### Base URLs

- **Development**: `http://localhost:8080`
- **Production**: `https://vteam-backend.<apps-domain>`

### Authentication

The platform uses OpenShift OAuth for authentication. Include the user's bearer token in all requests:

```http
Authorization: Bearer <user-oauth-token>
Content-Type: application/json
```

### Projects API

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/projects` | List all accessible projects |
| POST | `/api/projects` | Create new project |
| GET | `/api/projects/:project` | Get project details |
| DELETE | `/api/projects/:project` | Delete project |

### Agentic Sessions API

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/projects/:project/agentic-sessions` | List sessions in project |
| POST | `/api/projects/:project/agentic-sessions` | Create new session |
| GET | `/api/projects/:project/agentic-sessions/:name` | Get session details |
| DELETE | `/api/projects/:project/agentic-sessions/:name` | Delete session |

### Project Settings API

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/projects/:project/settings` | Get project configuration |
| PUT | `/api/projects/:project/settings` | Update project settings |

### Health & Status

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/health` | Backend health check |

### Example: Creating an AgenticSession via API

```bash
curl -X POST \
  https://vteam-backend.apps.example.com/api/projects/my-project/agentic-sessions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "analyze-repo",
    "spec": {
      "prompt": "Analyze this codebase and suggest improvements",
      "repos": [
        {
          "input": {
            "url": "https://github.com/myorg/myrepo",
            "branch": "main"
          }
        }
      ],
      "interactive": false,
      "timeout": 3600
    }
  }'
```

## WebSocket API

Real-time session updates are available via WebSocket connection to the backend. This enables live status monitoring in the web interface.

**Connection URL**: `wss://vteam-backend.<apps-domain>/ws`

Messages are broadcasted when AgenticSession status changes (phase transitions, completion, errors).

## Error Handling

### Common HTTP Status Codes

| Code | Error | Description |
|------|-------|-------------|
| 400 | `Bad Request` | Invalid request format or missing required fields |
| 401 | `Unauthorized` | Missing or invalid bearer token |
| 403 | `Forbidden` | User lacks RBAC permissions for the operation |
| 404 | `Not Found` | Project or session does not exist |
| 500 | `Internal Server Error` | Backend processing failure |

### AgenticSession Error States

When an AgenticSession fails, the `status.phase` will be `Failed` or `Error`, with details in `status.message`:

```yaml
status:
  phase: Failed
  message: "Repository clone failed: authentication required"
  startTime: "2025-10-30T10:00:00Z"
  completionTime: "2025-10-30T10:01:15Z"
```

## Kubernetes Resources

When you create an AgenticSession, the platform automatically creates these Kubernetes resources:

- **Job**: Manages the pod lifecycle for session execution
- **Pod**: Runs the Claude Code runner container
- **PersistentVolumeClaim**: Provides workspace storage for repository clones
- **Secret**: Contains API keys (created by ProjectSettings)

All resources use **OwnerReferences** for automatic cleanup when the AgenticSession is deleted.

## Performance Considerations

### Expected Response Times

| Operation | Target Time | Notes |
|-----------|-------------|-------|
| Session Creation (API) | < 2 seconds | Creates CR, returns immediately |
| Job Pod Startup | 10-30 seconds | Image pull, volume mount |
| Simple Code Analysis | 2-5 minutes | Depends on repository size |
| Complex Refactoring | 10-30 minutes | Multiple file changes |

### System Limits

Default limits (configurable via ProjectSettings):

- **Session Timeout**: 3600 seconds (1 hour)
- **Concurrent Sessions**: Limited by namespace resource quotas
- **Repository Size**: No hard limit, but larger repos increase execution time
- **API Rate Limit**: Enforced by Anthropic API (typically 100 RPM)

## Version History

### Current Version: v2.0.0

**Major Features:**

- Kubernetes operator-based orchestration with Custom Resources
- Next.js frontend with Shadcn UI and React Query
- Multi-repository support for cross-repo analysis
- Interactive and headless execution modes
- Production-ready OpenShift deployment architecture

**Breaking Changes:**

- Complete architecture rewrite: moved from LlamaDeploy to Kubernetes operators
- API endpoints now use project-scoped pattern: `/api/projects/:project/*`
- Frontend migrated from @llamaindex/server to Next.js with Shadcn UI
- Authentication now uses OpenShift OAuth with user bearer tokens
- Configuration moved from files to Kubernetes Custom Resources (ProjectSettings)

## Support

### Getting Help

- **Documentation Issues**: [GitHub Issues](https://github.com/ambient-code/platform/issues)
- **API Questions**: [GitHub Discussions](https://github.com/ambient-code/platform/discussions)
- **Bug Reports**: Include system info, error messages, and reproduction steps

### Gathering System Information

To help with support requests, gather this information:

```bash
# Version info
git describe --tags

# Kubernetes cluster info
kubectl version
kubectl get pods -n ambient-code

# Component versions
kubectl get pods -n ambient-code -o jsonpath='{.items[*].spec.containers[*].image}'

# Check AgenticSession status
kubectl get agenticsessions -n <namespace> -o yaml

# View session logs
kubectl logs job/<session-name> -n <namespace>
```

## Additional Resources

- **User Guide**: [Getting Started](../user-guide/getting-started.md) for usage instructions
- **Labs**: [Hands-on exercises](../labs/index.md) for practical learning
- **Deployment Guides**: [OpenShift deployment](../OPENSHIFT_DEPLOY.md) for production setup
- **Contributing**: See the project's AGENTS.md file in the repository root for contributor guidelines and architecture details

---

This reference documentation is maintained alongside the codebase. Found an error or missing information? [Submit a pull request](https://github.com/ambient-code/platform/pulls) or [create an issue](https://github.com/ambient-code/platform/issues).
