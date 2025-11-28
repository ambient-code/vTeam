# ADR-0006: Agent Injection Architecture

**Status**: Proposed
**Date**: 2025-11-24 (Updated: 2025-11-26)
**Related**: [ADR-0001 Kubernetes Native Architecture](0001-kubernetes-native-architecture.md)

## Context

Users want to run agentic sessions in their own container environments. The platform must support arbitrary user-provided base images while injecting our agent code to orchestrate Claude API interactions.

**Critical architectural principle**:
```
User's image = workspace environment (their choice)
Agent = our code (separate container)
```

The agent must be **separate** from the user's base image. We cannot force users to extend our base image or install our dependencies in their environment.

## Decision

**Always use separate agent and workspace containers with kubectl exec for command execution.**

Every AgenticSession runs with this architecture:

1. **Workspace container**: User's environment running `sleep infinity`
2. **Agent container**: Our Claude Code runner that executes commands via kubectl exec

This provides complete separation between the agent runtime and user's environment, avoiding dependency conflicts and supporting any user image including distroless containers.

### Cascading Configuration

Workspace configuration cascades from platform defaults through project settings to session-specific overrides:

```
Platform Default → ProjectSettings → AgenticSession
```

| Level | Configuration | Purpose |
|-------|--------------|---------|
| Platform | Default workspace image in operator config | Fallback for all sessions |
| Project | `ProjectSettings.spec.workspacePodTemplate` | Team-wide defaults |
| Session | `AgenticSession.spec.workspacePodTemplate` | Per-session customization |

### Pod Architecture

```yaml
spec:
  shareProcessNamespace: true
  serviceAccountName: ambient-runner  # Has pods/exec permission
  containers:
  - name: workspace
    # Image from cascading config (platform → project → session)
    image: <resolved-workspace-image>
    command: ["sleep", "infinity"]
    workingDir: /workspace/sessions/<session>/workspace
    # Additional spec from workspacePodTemplate
    volumeMounts:
    - name: workspace-pvc
      mountPath: /workspace

  - name: ambient-code-runner
    image: quay.io/ambient_code/vteam_claude_runner:latest
    env:
    - name: WORKSPACE_CONTAINER
      value: workspace
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
    - name: POD_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
    volumeMounts:
    - name: workspace-pvc
      mountPath: /workspace

  - name: ambient-content
    image: quay.io/ambient_code/vteam_backend:latest
    # Content service sidecar
```

### Tool Replacement via MCP

The agent uses MCP (Model Context Protocol) to provide command execution in the workspace container:

**Agent tool configuration** (wrapper.py):
```python
# Always use MCP for workspace command execution
allowed_tools = ["Read", "Write", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]
disallowed_tools = ["Bash", "BashOutput", "KillShell"]

# Add MCP workspace exec tool
mcp_servers["workspace"] = {
    "type": "http",
    "url": f"http://localhost:{mcp_port}/mcp"
}
allowed_tools.append("mcp__workspace")

options = ClaudeAgentOptions(
    allowed_tools=allowed_tools,
    disallowed_tools=disallowed_tools,
    mcp_servers=mcp_servers,
    # ...
)
```

### MCP Server Implementation

The MCP server runs in the agent container and provides command execution via kubectl:

```python
# mcp_servers/workspace_exec.py
from mcp.server.fastmcp import FastMCP
import subprocess
import os

mcp = FastMCP("workspace")

@mcp.tool()
def exec(command: str, workdir: str = None, timeout: int = 300) -> str:
    """Execute a command in the workspace container.

    Args:
        command: Shell command to execute
        workdir: Working directory (default: session workspace)
        timeout: Command timeout in seconds
    """
    pod_name = os.environ["POD_NAME"]
    namespace = os.environ["POD_NAMESPACE"]
    container = os.environ.get("WORKSPACE_CONTAINER", "workspace")

    if workdir:
        full_cmd = f"cd {workdir} && {command}"
    else:
        full_cmd = command

    result = subprocess.run(
        ["kubectl", "exec", "-n", namespace, pod_name,
         "-c", container, "--", "sh", "-c", full_cmd],
        capture_output=True,
        text=True,
        timeout=timeout
    )

    output = result.stdout
    if result.stderr:
        output += f"\n[stderr]: {result.stderr}"
    if result.returncode != 0:
        output += f"\n[exit code: {result.returncode}]"

    return output
```

### RBAC Requirements

The agent container needs permission to exec into the workspace container:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ambient-runner
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: ambient-runner-exec
rules:
- apiGroups: [""]
  resources: ["pods/exec"]
  verbs: ["create"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ambient-runner-exec
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: ambient-runner-exec
subjects:
- kind: ServiceAccount
  name: ambient-runner
```

The operator automatically creates these resources in each session namespace via `ensureRunnerServiceAccount()`.

## Consequences

### Positive

- **Complete separation**: User's container is pristine, no agent code injected
- **Universal compatibility**: Works with any user image (distroless, Alpine/musl, minimal)
- **No dependency conflicts**: Agent runtime is isolated from user environment
- **User's tools preserved**: User's container runs their tools natively
- **Easy debugging**: Can exec into workspace container independently
- **Flexible configuration**: Cascading defaults allow team-wide and per-session customization

### Negative

- **Requires RBAC**: Pod needs exec permissions (security consideration)
- **Cross-container overhead**: kubectl exec has latency vs direct execution
- **Complexity**: Two containers instead of one, MCP server coordination
- **Service account token**: Pod must mount SA token for kubectl API access

### Neutral

- **Shared workspace volume**: Both containers access `/workspace` PVC
- **Process namespace sharing**: Enabled for cross-container visibility

## User Configuration

### Platform Default

Set in operator deployment:

```yaml
# Operator ConfigMap or environment
PLATFORM_DEFAULT_WORKSPACE_IMAGE: "registry.access.redhat.com/ubi9/ubi:latest"
```

### ProjectSettings CRD

Project-level workspace configuration:

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: ProjectSettings
metadata:
  name: default
  namespace: my-project
spec:
  # Default workspace image for all sessions in this project
  workspaceImage: python:3.11

  # Optional: Full pod template for workspace container
  workspacePodTemplate:
    spec:
      containers:
      - name: workspace
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2"
        env:
        - name: PYTHONUNBUFFERED
          value: "1"
```

### AgenticSession CRD

Session-level overrides:

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: my-session
spec:
  prompt: "Analyze the data in data.csv"

  # Override workspace image for this session
  workspaceImage: jupyter/scipy-notebook:latest

  # Optional: Additional pod template customization
  workspacePodTemplate:
    spec:
      containers:
      - name: workspace
        resources:
          limits:
            nvidia.com/gpu: "1"
        volumeMounts:
        - name: datasets
          mountPath: /data
      volumes:
      - name: datasets
        persistentVolumeClaim:
          claimName: shared-datasets

  repos:
  - name: my-repo
    input:
      url: https://github.com/myorg/myrepo
      branch: main
```

### Configuration Resolution

The operator merges configuration in order:

1. Start with platform default image
2. Apply ProjectSettings.workspaceImage (if set)
3. Apply ProjectSettings.workspacePodTemplate (if set)
4. Apply AgenticSession.workspaceImage (if set)
5. Apply AgenticSession.workspacePodTemplate (if set)

Pod template merging uses strategic merge patch semantics.

### Examples

**Python Data Science**:
```yaml
spec:
  workspaceImage: jupyter/scipy-notebook:latest
  prompt: "Analyze sales data"
```

**Rust Development with extra memory**:
```yaml
spec:
  workspaceImage: rust:1.75
  workspacePodTemplate:
    spec:
      containers:
      - name: workspace
        resources:
          limits:
            memory: "8Gi"
  prompt: "Build the project in release mode"
```

**GPU-enabled ML workspace**:
```yaml
spec:
  workspaceImage: pytorch/pytorch:2.1.0-cuda12.1-cudnn8-runtime
  workspacePodTemplate:
    spec:
      containers:
      - name: workspace
        resources:
          limits:
            nvidia.com/gpu: "1"
  prompt: "Train the model"
```

**Distroless Python** (minimal attack surface):
```yaml
spec:
  workspaceImage: gcr.io/distroless/python3
  prompt: "Run the analysis script"
```

## Alternatives Considered

### Agent Injection via Image Volume Mount

Agent binary mounted directly from container image using Kubernetes image volumes (v1.31+).

```yaml
spec:
  containers:
  - name: workspace
    image: <user-provided>
    command: ["/agent/runner"]  # Agent as entrypoint
    volumeMounts:
    - name: agent-binary
      mountPath: /agent
      readOnly: true

  volumes:
  - name: agent-binary
    image:
      reference: quay.io/ambient_code/agent:latest
      pullPolicy: IfNotPresent
```

**Why rejected**:
1. **Kubernetes 1.31+ required**: Image volumes are a newer feature
2. **libc compatibility issues**: Static binary must match user's libc (glibc vs musl)
3. **Distroless incompatible**: No shell to run agent binary
4. **Complex binary building**: Requires PyInstaller/Nuitka to create standalone binary
5. **User sees agent process**: Agent runs in user's container

### Init Container Copy

For Kubernetes < 1.31, copy agent binary via init container:

```yaml
initContainers:
- name: inject-agent
  image: quay.io/ambient_code/agent-binary:latest
  command: ["sh", "-c", "cp -r /agent /shared/"]
  volumeMounts:
  - name: agent-bin
    mountPath: /shared

containers:
- name: workspace
  image: <user-provided>
  command: ["/shared/agent/runner"]
  volumeMounts:
  - name: agent-bin
    mountPath: /shared/agent
```

**Why rejected**: Same libc/distroless issues as image volume approach.

### Process Namespace Sharing without kubectl

Use `shareProcessNamespace: true` with `nsenter` for command execution:

```python
def exec_via_nsenter(command: str, workspace_pid: int) -> str:
    return subprocess.run(
        ["nsenter", "-t", str(workspace_pid), "-m", "-p", "--",
         "sh", "-c", command],
        capture_output=True
    )
```

**Why rejected**:
- Requires `SYS_PTRACE` capability (security concern)
- More complex than kubectl exec
- Harder to debug

## Implementation References

### Files Modified

- **wrapper.py**: Tool configuration, MCP server startup
- **mcp_servers/workspace_exec.py**: MCP server providing exec tool
- **operator/internal/handlers/sessions.go**:
  - Pod spec generation with workspace container
  - RBAC setup via `ensureRunnerServiceAccount()`
  - Configuration cascading logic

### Environment Variables

| Variable | Description |
|----------|-------------|
| `WORKSPACE_CONTAINER` | Name of workspace container (default: "workspace") |
| `POD_NAME` | Injected by downward API |
| `POD_NAMESPACE` | Injected by downward API |
| `PLATFORM_DEFAULT_WORKSPACE_IMAGE` | Operator-level default image |

### CRD Schema Updates

```yaml
# ProjectSettings additions
spec:
  workspaceImage:
    type: string
    description: Default workspace image for sessions in this project
  workspacePodTemplate:
    type: object
    x-kubernetes-preserve-unknown-fields: true
    description: Pod template for workspace container customization

# AgenticSession additions
spec:
  workspaceImage:
    type: string
    description: Override workspace image for this session
  workspacePodTemplate:
    type: object
    x-kubernetes-preserve-unknown-fields: true
    description: Pod template for workspace container customization
```

## References

- [Kubernetes Pod Exec API](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podexecptions-v1-core)
- [MCP Protocol](https://modelcontextprotocol.io/)
- [Share Process Namespace](https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace/)
- [Strategic Merge Patch](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#use-a-strategic-merge-patch-to-update-a-deployment)
