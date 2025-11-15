# Unified Observability with Langfuse

This directory contains scripts and documentation for implementing **unified LLM and system observability** using Langfuse as the single backend for both Langfuse SDK traces and OpenTelemetry traces.

## Overview

**Unified Observability** means all traces (both Langfuse SDK spans AND OpenTelemetry spans) are sent to Langfuse and appear in the same trace hierarchy:

- **Langfuse SDK spans**: LLM-specific observability (prompts, responses, generations, tool executions with full I/O)
- **OpenTelemetry spans**: System-level distributed tracing (session lifecycle, performance metrics, costs)

Both span types automatically nest together in Langfuse because **Langfuse SDK v3 is OpenTelemetry-native**.

## Prerequisites

1. **Langfuse deployed** on your cluster
   - Run: `../scripts/deploy-langfuse-openshift.sh`
   - Verify: `oc get pods -n langfuse`

2. **Langfuse API keys** generated
   - Access Langfuse UI: https://langfuse-langfuse.apps.<your-cluster>
   - Create organization and project
   - Generate API keys: Settings → API Keys

3. **OpenShift CLI** (`oc`) installed and logged in

## Quick Start

### Using WorkspaceSettings UI (Recommended)

**All observability configuration is managed via the WorkspaceSettings UI.**

1. **Access WorkspaceSettings** for your project:
   - Navigate to your workspace
   - Go to Settings tab
   - Expand "Observability (Langfuse + OpenTelemetry)" section

2. **Configure Langfuse** (pre-populated with defaults):
   - `LANGFUSE_ENABLED`: `true` (already set)
   - `LANGFUSE_HOST`: `http://langfuse-web.langfuse.svc.cluster.local:3000` (already set)
   - `LANGFUSE_PUBLIC_KEY`: Add your `pk-lf-...` key
   - `LANGFUSE_SECRET_KEY`: Add your `sk-lf-...` key

3. **OpenTelemetry (Optional - Advanced Users Only)**:
   - `OTEL_EXPORTER_OTLP_ENDPOINT`: **Leave empty** (traces auto-sent to Langfuse)
   - `OTEL_SERVICE_NAME`: `claude-code-runner` (dynamically set to `claude-{session-id}`)

4. **Save Integration Secrets** - All keys saved to `ambient-non-vertex-integrations`

### Using Scripts (Legacy)

```bash
# Create .env.langfuse-keys file
cat > .env.langfuse-keys <<EOF
LANGFUSE_PUBLIC_KEY=pk-lf-your-public-key-here
LANGFUSE_SECRET_KEY=sk-lf-your-secret-key-here
EOF

# Run configuration script
./configure-ambient-code.sh
```

**Recommendation**: Use WorkspaceSettings UI for better multi-project support.

## How Unified Observability Works

```
┌─────────────────────────────────────────────────┐
│  Runner Pod                                     │
├─────────────────────────────────────────────────┤
│  Langfuse SDK v3 (OTEL-native)                 │
│       ↓ creates                                 │
│  • Session span (Langfuse API)                  │
│  • Tool spans (Read, Write, Bash...)           │
│  • Generation spans (Claude responses)          │
│                                                 │
│  OpenTelemetry SDK                              │
│       ↓ creates                                 │
│  • Session span (system-level)                  │
│  • Tool decision events                         │
│  • Performance metrics                          │
│                                                 │
│  Both span sources ──────────────────────────┐ │
└──────────────────────────────────────────────┼─┘
                                               ↓
                    Langfuse OTLP Endpoint
                /api/public/otel/v1/traces
                (HTTP/protobuf + Basic Auth)
                                               ↓
                         Langfuse Backend
                                               ↓
                    ┌─────────────────────────┐
                    │  Unified Trace View     │
                    ├─────────────────────────┤
                    │ claude_agent_session    │ ← Main span
                    │   ├─ tool_Read         │ ← Langfuse SDK
                    │   ├─ tool_Write        │ ← Langfuse SDK
                    │   ├─ claude_response   │ ← Langfuse SDK
                    │   └─ Performance data  │ ← OTEL SDK
                    └─────────────────────────┘
```

### Key Points

1. **Auto-derivation**: If `OTEL_EXPORTER_OTLP_ENDPOINT` is empty, runner derives endpoint from `LANGFUSE_HOST`
2. **Basic Auth**: Runner automatically adds `Authorization: Basic {base64(public_key:secret_key)}` header
3. **Protocol**: HTTP/protobuf (gRPC not supported by Langfuse)
4. **Context propagation**: OpenTelemetry automatically nests spans within same trace

## Trace Structure

### Langfuse SDK Spans

1. **Session Span** - Main span for entire Claude session
   - Input: Original prompt
   - Output: Final results with cost/token metrics
   - Metadata: session ID, namespace

2. **Tool Spans** - Child spans for each tool execution
   - Input: Tool parameters (full detail)
   - Output: Tool results (truncated to 500 chars)
   - Metadata: tool name, ID, turn number

3. **Generation Spans** - Claude's text responses
   - Input: Turn number
   - Output: Claude text (truncated to 1000 chars)
   - Metadata: Model, turn number

### OpenTelemetry Spans

1. **Session Span** - System-level span matching session
   - Attributes: session ID, namespace, prompt length
   - Final attributes: cost, tokens, turns, duration, subtype

2. **Tool Events** - Events on session span
   - `claude_code.tool_decision`: tool name, tool ID
   - `claude_code.tool_result`: tool use ID, error status

## Viewing Unified Traces

1. Open Langfuse UI: https://langfuse-langfuse.apps.<your-cluster>
2. Navigate to your project
3. View traces by session ID
4. Drill down to see:
   - **Langfuse spans**: Full tool I/O, generation content
   - **OTEL spans**: Performance metrics, costs, tokens
   - All in one unified hierarchy!

## Configuration Details

All environment variables are stored in the `ambient-non-vertex-integrations` secret:

### Required Keys

```yaml
LANGFUSE_PUBLIC_KEY: "pk-lf-..."
LANGFUSE_SECRET_KEY: "sk-lf-..."
```

### Pre-configured (Optional to Override)

```yaml
LANGFUSE_ENABLED: "true"
LANGFUSE_HOST: "http://langfuse-web.langfuse.svc.cluster.local:3000"
OTEL_SERVICE_NAME: "claude-code-runner"  # Dynamically set to claude-{session-id}
```

### Advanced (Usually Empty)

```yaml
OTEL_EXPORTER_OTLP_ENDPOINT: ""  # Leave empty to use Langfuse
```

## Updating Configuration

To update your observability settings:

1. Go to WorkspaceSettings → Settings → Observability
2. Update Langfuse keys or other settings
3. Click "Save Integration Secrets"
4. New sessions use updated config immediately

## Troubleshooting

### No traces appearing in Langfuse

1. **Check Langfuse API keys are set**:
   ```bash
   oc get secret ambient-non-vertex-integrations -n ambient-code -o yaml | grep LANGFUSE
   ```

2. **Check Langfuse is enabled**:
   ```bash
   oc get secret ambient-non-vertex-integrations -n ambient-code -o jsonpath='{.data.LANGFUSE_ENABLED}' | base64 -d
   # Should show: true
   ```

3. **Check runner pod has env vars**:
   ```bash
   oc get pod <runner-pod> -o yaml | grep -A 5 envFrom
   ```

   Should show:
   ```yaml
   envFrom:
   - secretRef:
       name: ambient-non-vertex-integrations
   ```

4. **Check runner logs for initialization**:
   ```bash
   oc logs <runner-pod> | grep -i "langfuse\|otel"
   ```

   Should see:
   ```
   INFO - Langfuse tracing enabled for session
   INFO - Using Langfuse OTLP endpoint (unified observability): http://...
   INFO - Added Langfuse Basic Auth to OTLP exporter
   INFO - OpenTelemetry tracing enabled (endpoint: http://...)
   INFO - OpenTelemetry spans flushed to collector
   ```

### Only seeing Langfuse SDK spans (no OTEL spans)

Check that OTEL endpoint was derived correctly:
```bash
oc logs <runner-pod> | grep "Using Langfuse OTLP endpoint"
```

If you see gRPC warnings, ensure `OTEL_EXPORTER_OTLP_ENDPOINT` is empty (not set to gRPC collector).

### Spans not combined in same trace

This shouldn't happen with Langfuse SDK v3 (OTEL-native). If you see separate traces:

1. Verify Langfuse SDK version:
   ```bash
   oc exec <runner-pod> -- pip show langfuse | grep Version
   # Should be 3.x or higher
   ```

2. Check for trace ID in logs - both systems should use same trace ID

## Alternative Backends (Tempo/Grafana/Jaeger)

**Important Limitation**: Langfuse SDK spans **cannot** be exported to Tempo/Grafana/Jaeger. The Langfuse SDK always sends traces to the Langfuse API endpoint, even though it's "OTEL-native".

**What you CAN do** (advanced users only):
- Set `OTEL_EXPORTER_OTLP_ENDPOINT` to send **OpenTelemetry SDK spans** to Tempo/Jaeger/Grafana
- This creates **two separate trace systems**:
  - Langfuse SDK spans → Langfuse UI (LLM-specific: tool I/O, generations, prompts)
  - OTEL SDK spans → Tempo/Grafana (system metrics: duration, tokens, cost)
- Manual correlation required via session ID or timestamps

**Recommendation**: Use unified Langfuse approach (default) for best experience. Only use separate backends if you have existing observability infrastructure that requires it.

## Security Notes

- **Never commit** API keys to version control
- API keys have **full access** to your Langfuse project
- Kubernetes Secrets are **base64 encoded** (not encrypted at rest by default)
- For production, consider using **Sealed Secrets** or **External Secrets Operator**
- Langfuse OTLP endpoint uses **Basic Auth** (keys sent in Authorization header)

## Multi-Project Setup

Each project namespace can have its own `ambient-non-vertex-integrations` secret with different Langfuse keys for per-project isolation and cost tracking.

## Why Unified Observability?

**Benefits over separate systems:**

✅ **Single source of truth**: All traces in one place
✅ **Automatic correlation**: Spans auto-nest via OTEL context propagation
✅ **Simpler setup**: Only Langfuse keys needed
✅ **Better UX**: Langfuse UI designed for LLM observability
✅ **Cost efficiency**: No separate OTEL Collector infrastructure
✅ **Richer insights**: Combine LLM-specific + system-level observability

## References

- Langfuse Documentation: https://langfuse.com/docs
- Langfuse OTLP Integration: https://langfuse.com/integrations/native/opentelemetry
- Langfuse Python SDK v3: https://langfuse.com/changelog/2025-05-23-otel-based-python-sdk
- Ambient Code Observability: See `components/runners/claude-code-runner/wrapper.py`
