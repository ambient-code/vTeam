# Observability with Langfuse

## Overview

**Langfuse** is the observability platform for the vTeam/Ambient Code Platform. All traces are sent to Langfuse for LLM-specific observability:

- **Session spans**: Full Claude session lifecycle with cost and token metrics
- **Generation spans**: Claude's text responses with prompt/completion tracking
- **Tool spans**: Tool executions (Read, Write, Bash, etc.) with full I/O capture

## Prerequisites

1. **Langfuse deployed** on your cluster
   - Run: `../scripts/deploy-langfuse.sh`
   - Verify: `oc get pods -n langfuse`

2. **Langfuse API keys** generated
   - Access Langfuse UI: https://langfuse-langfuse.apps.<your-cluster>
   - Create organization and project
   - Generate API keys: Settings → API Keys

3. **OpenShift CLI** (`oc`) or **Kubernetes CLI** (`kubectl`) installed and logged in

## Quick Start

### Platform-Wide Configuration (Platform Admin Only)

**IMPORTANT**: Langfuse observability is configured by **platform administrators**, not individual workspace users. All LANGFUSE_* configuration is stored in a single secret for consistency and security.

#### Step 1: Create the Secret

Create the `ambient-admin-langfuse-secret` secret with all Langfuse configuration:

```bash
kubectl create secret generic ambient-admin-langfuse-secret \
  --from-literal=LANGFUSE_PUBLIC_KEY=pk-lf-... \
  --from-literal=LANGFUSE_SECRET_KEY=sk-lf-... \
  --from-literal=LANGFUSE_HOST=http://langfuse-web.langfuse.svc.cluster.local:3000 \
  --from-literal=LANGFUSE_ENABLED=true \
  -n ambient-code
```

#### Step 2: Restart the Operator

```bash
kubectl rollout restart deployment ambient-operator -n ambient-code
```

#### Step 3: Verify

All new sessions will automatically have Langfuse observability enabled. Check runner pod logs:

```bash
kubectl logs -n <workspace-namespace> <runner-pod> -c ambient-code-runner | grep Langfuse
```

### Per-Workspace Configuration (Not Supported)

Per-workspace Langfuse configuration is **not supported**. Observability must be consistent across the platform for proper cost tracking and compliance. All LANGFUSE_* configuration is managed by platform administrators via the `ambient-admin-langfuse-secret` secret.

If you need workspace-specific tracking, use Langfuse's built-in tagging and filtering features instead.

## How Langfuse Observability Works

```
┌─────────────────────────────────────────────────┐
│  Runner Pod                                     │
├─────────────────────────────────────────────────┤
│  Langfuse SDK v3                                │
│       ↓ creates                                 │
│  • Session span (claude_agent_session)          │
│  • Tool spans (Read, Write, Bash...)           │
│  • Generation spans (Claude responses)          │
│                                                 │
│  All spans ────────────────────────────────────┐│
└─────────────────────────────────────────────────┼┘
                                                  ↓
                        Langfuse HTTP API
                    /api/public/ingestion
                    (HTTP/JSON with API keys)
                                                  ↓
                       Langfuse Backend
                                                  ↓
                  ┌─────────────────────────┐
                  │  Unified Trace View     │
                  ├─────────────────────────┤
                  │ claude_agent_session    │ ← Session span
                  │   ├─ tool_Read         │ ← Tool execution
                  │   ├─ tool_Write        │ ← Tool execution
                  │   ├─ claude_response   │ ← Claude generation
                  │   └─ Cost & tokens     │ ← Final metrics
                  └─────────────────────────┘
```

### Key Points

1. **Simple integration**: Runner uses Langfuse Python SDK (v3+) for direct HTTP API calls
2. **API Key auth**: Runner automatically receives all LANGFUSE_* config from `ambient-admin-langfuse-secret` secret
3. **Automatic nesting**: Child spans (tools, generations) attach to parent session span via SDK context

## Trace Structure

### Session Span (Root)

**Name**: `claude_agent_session`

- **Input**: Original prompt from user
- **Output**: Final results with cost/token metrics
- **Metadata**:
  - `session_id`: Kubernetes session name
  - `namespace`: Project namespace
  - `user_id`: User identifier (for multi-user tracking)
  - `user_name`: User display name

### Tool Spans (Children)

**Name**: `tool_{ToolName}` (e.g., `tool_Read`, `tool_Write`, `tool_Bash`)

- **Input**: Tool parameters (full detail)
- **Output**: Tool results (truncated to 500 chars)
- **Metadata**:
  - `tool_id`: Unique ID for this tool use
  - `tool_name`: Tool name (Read, Write, etc.)

### Generation Spans (Children)

**Name**: `claude_response`

- **Input**: Turn number
- **Output**: Claude's text response (truncated to 1000 chars)
- **Metadata**:
  - `model`: Model name (e.g., `claude-3-5-sonnet-20241022`)
  - `turn`: Turn number in conversation
- **Usage**: Token counts for cost tracking
  - `input`: Input tokens
  - `output`: Output tokens
  - `cache_read_input_tokens`: Cache hits (if applicable)
  - `cache_creation_input_tokens`: Cache writes (if applicable)

## Viewing Traces

1. Open Langfuse UI: https://langfuse-langfuse.apps.<your-cluster>
2. Navigate to your project
3. View traces by session ID or timestamp
4. Drill down to see:
   - **Full tool I/O**: Complete input/output for each tool execution
   - **Generation content**: Claude's responses with token breakdown
   - **Cost tracking**: Total cost and per-generation costs
   - **Session metrics**: Total tokens, turns, duration

## Configuration Details

### Platform Admin Secret (`ambient-admin-langfuse-secret`)

All Langfuse configuration is stored in a single secret managed by platform administrators:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ambient-admin-langfuse-secret
  namespace: ambient-code  # operator's namespace
type: Opaque
stringData:
  # Credentials (sensitive)
  LANGFUSE_PUBLIC_KEY: "pk-lf-..."
  LANGFUSE_SECRET_KEY: "sk-lf-..."

  # Configuration
  LANGFUSE_HOST: "http://langfuse-web.langfuse.svc.cluster.local:3000"
  LANGFUSE_ENABLED: "true"
```

**Note**: This secret is platform-wide and cannot be overridden per-workspace. For workspace-specific tracking, use Langfuse's tags and filters.

## Updating Configuration

### Platform-Wide Update (Platform Admin Only)

```bash
# Update the secret
kubectl create secret generic ambient-admin-langfuse-secret \
  --from-literal=LANGFUSE_PUBLIC_KEY=pk-lf-new-key \
  --from-literal=LANGFUSE_SECRET_KEY=sk-lf-new-key \
  --from-literal=LANGFUSE_HOST=http://langfuse-web.langfuse.svc.cluster.local:3000 \
  --from-literal=LANGFUSE_ENABLED=true \
  -n ambient-code \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart operator to pick up changes
kubectl rollout restart deployment ambient-operator -n ambient-code
```

### Per-Project Update

1. Go to WorkspaceSettings → Settings → Observability
2. Update Langfuse keys
3. Click "Save Integration Secrets"
4. New sessions use updated config immediately

## Multi-Project Setup

**Platform-wide configuration**: All projects share the same Langfuse instance and keys.

**Per-project configuration**: Each project namespace can have its own `ambient-non-vertex-integrations` secret with different Langfuse keys for per-project isolation and cost tracking.

## Why Langfuse?

**Benefits of Langfuse for AI observability:**

✅ **LLM-optimized**: Designed specifically for LLM observability with prompt/response tracking
✅ **Simple setup**: Only Langfuse keys needed, no additional infrastructure
✅ **Cost tracking**: Built-in token and cost calculation for Claude API usage
✅ **Rich insights**: Full tool I/O, generation content, and performance metrics
✅ **Easy debugging**: Trace view shows exact sequence of tool calls and responses
✅ **Multi-user support**: Track usage by user_id for cost allocation

## References

- Langfuse Documentation: https://langfuse.com/docs
- Langfuse Python SDK v3: https://langfuse.com/docs/sdk/python
- Ambient Code Observability: See `components/runners/claude-code-runner/observability.py`
