# OpenTelemetry Configuration for Ambient Code

This directory contains documentation for enabling OpenTelemetry instrumentation in the Claude Code runner.

## Overview

The Claude Code runner has custom OpenTelemetry instrumentation that captures:
- **Session lifecycle** (start, end, duration)
- **Tool executions** (Read, Write, Bash, Glob, Grep, Edit, etc.) with timing
- **Tool results** and error states
- **Cost and token metrics** (via final span attributes)

This complements Langfuse (which captures LLM-specific observability like prompts, responses, and generations).

## Prerequisites

1. **OTEL Collector** deployed in your cluster
   - Tempo, Jaeger, or other OTLP-compatible backend
   - Accepting traces on port 4317 (gRPC) or 4318 (HTTP)

2. **OpenShift CLI** (`oc`) installed and logged in

## Quick Start

### Using WorkspaceSettings UI (Recommended)

**All observability configuration is now managed via the WorkspaceSettings UI in the frontend.**

1. **Access WorkspaceSettings** for your project:
   - Navigate to your workspace
   - Go to Settings tab
   - Expand "Observability (Langfuse + OpenTelemetry)" section

2. **Configure OpenTelemetry** (pre-populated with defaults):
   - `OTEL_EXPORTER_OTLP_ENDPOINT`: `http://otel-collector-collector.observability-hub.svc.cluster.local:4317` (already set)
   - `OTEL_SERVICE_NAME`: `claude-code-runner` (overridden to `claude-{session-id}` at runtime)
   - `OTEL_EXPORTER_OTLP_PROTOCOL`: `grpc` (already set)

3. **Save Integration Secrets** - Saves all observability keys to `ambient-non-vertex-integrations` secret

**Note**: The service name is automatically set to `claude-{session-id}` at runtime for better trace isolation.

## How It Works

```
┌─────────────────────────────────────────┐
│  Project Namespace (e.g., ambient-code) │
├─────────────────────────────────────────┤
│  WorkspaceSettings UI                   │
│       ↓ creates/updates                 │
│  • ambient-non-vertex-integrations      │
│       (Secret with observability keys)  │
│                                         │
│  • vteam-operator                       │
│       ↓ injects into spawned jobs       │
│  • claude-runner-job-xyz                │
│       ↓ uses env vars                   │
│       • OTEL_EXPORTER_OTLP_ENDPOINT     │
│       • OTEL_SERVICE_NAME (overridden)  │
│       • OTEL_EXPORTER_OTLP_PROTOCOL     │
│       ↓ OpenTelemetry SDK               │
│  • Traces sent to collector ───────────┼──→ OTEL Collector → Tempo/Jaeger
└─────────────────────────────────────────┘
```

The operator automatically injects all keys from `ambient-non-vertex-integrations` into runner pods using `EnvFrom`.

## Configuration Details

All observability environment variables are stored in the `ambient-non-vertex-integrations` secret:

### OpenTelemetry Configuration (Pre-configured)

```yaml
OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector-collector.observability-hub.svc.cluster.local:4317"
OTEL_SERVICE_NAME: "claude-code-runner"  # Runtime override: claude-{session-id}
OTEL_EXPORTER_OTLP_PROTOCOL: "grpc"
```

### Protocol Options

- **gRPC** (port 4317): Default, more efficient
- **HTTP** (port 4318): Alternative, use if gRPC unavailable

Update the endpoint port in WorkspaceSettings based on your collector configuration.

## Span Structure

The runner creates OpenTelemetry spans with:

1. **Session Span** - Main span for the entire Claude session
   - Attributes:
     - `session.id`: AgenticSession name
     - `namespace`: Kubernetes namespace
     - `prompt.length`: Length of initial prompt
   - Final attributes (on completion):
     - `claude_code.cost.usage`: Total cost in USD
     - `claude_code.token.usage`: Total tokens used
     - `claude_code.session.turns`: Number of turns
     - `claude_code.session.duration_ms`: Session duration
     - `claude_code.session.subtype`: Success/error type

2. **Tool Decision Events** - Events added to session span for each tool use
   - Attributes:
     - `tool.name`: Tool being used (Read, Write, Bash, etc.)
     - `tool.id`: Unique tool use ID

3. **Tool Result Events** - Events added for each tool result
   - Attributes:
     - `tool.use_id`: Matching tool use ID
     - `tool.is_error`: Whether tool execution failed

## Viewing Traces

### Grafana + Tempo

1. Open Grafana
2. Navigate to Explore
3. Select Tempo data source
4. Query: `{service.name=~"claude-.*"}`  (finds all Claude sessions)
   - Or specific session: `{service.name="claude-langfuse-test"}`

### Jaeger

1. Open Jaeger UI
2. Service dropdown will show: `claude-{session-id}`
3. Find traces by session ID

### Example Trace Attributes

```
service.name: claude-langfuse-test
session.id: langfuse-test
namespace: ambient-code
prompt.length: 82
claude_code.cost.usage: 0.00045
claude_code.token.usage: 1234
claude_code.session.turns: 15
claude_code.session.duration_ms: 42301
claude_code.session.subtype: success
```

## OTEL vs Langfuse

| Observability Type | Use OTEL | Use Langfuse |
|--------------------|----------|--------------|
| Session lifecycle & timing | ✅ | ✅ |
| Cost & token metrics | ✅ | ✅ |
| Tool decision events | ✅ | ❌ |
| Distributed trace correlation | ✅ | ❌ |
| Tool spans with I/O | ❌ | ✅ |
| Prompt/response content | ❌ | ✅ |
| Generation spans | ❌ | ✅ |
| Model parameters | ❌ | ✅ |

**Recommendation**: Use **both** for complete observability.
- **OTEL**: Session-level metrics, distributed tracing across services
- **Langfuse**: Detailed LLM observability with full context

## Troubleshooting

### Traces not appearing

1. **Check OTEL keys are set**:
   ```bash
   oc get secret ambient-non-vertex-integrations -n ambient-code -o yaml | grep OTEL
   ```

2. **Verify runner pod has env vars**:
   ```bash
   oc get pod <runner-pod> -n ambient-code -o yaml | grep -A 5 envFrom
   ```

   Should show:
   ```yaml
   envFrom:
   - secretRef:
       name: ambient-non-vertex-integrations
   ```

3. **Check runner logs for OTEL initialization**:
   ```bash
   oc logs <runner-pod> -c ambient-code-runner | grep -i otel
   ```

   Should see: `OpenTelemetry tracing enabled (endpoint: ...)`

4. **Verify endpoint is reachable from pods**:
   ```bash
   oc run -it --rm debug --image=curlimages/curl --restart=Never -- \
     curl -v http://otel-collector-collector.observability-hub.svc.cluster.local:4317
   ```

### Service name not showing as claude-{session-id}

Check runner logs for dynamic service name override:
```bash
oc logs <runner-pod> | grep "service.name"
```

Should see service name set to `claude-{session-id}`.

### gRPC connection errors

If seeing gRPC errors, try switching to HTTP protocol:

1. Go to WorkspaceSettings → Settings → Observability
2. Change `OTEL_EXPORTER_OTLP_ENDPOINT` port from `4317` to `4318`
3. Change `OTEL_EXPORTER_OTLP_PROTOCOL` from `grpc` to `http`
4. Save Integration Secrets

## Updating Configuration

To update OpenTelemetry settings:

1. Go to WorkspaceSettings → Settings → Observability
2. Update OTEL environment variables as needed
3. Click "Save Integration Secrets"
4. New sessions will use the updated configuration immediately

## Security Notes

- OTEL traces may contain **session IDs** and **namespace names**
- Ensure your OTEL collector has appropriate **access controls**
- Consider **data retention policies** for trace storage
- Use Tempo's **tenant isolation** for multi-project security

## Multi-Project Setup

Each project namespace can have its own `ambient-non-vertex-integrations` secret with different OTEL endpoints for per-project trace isolation.

## References

- OpenTelemetry Spec: https://opentelemetry.io/docs/specs/otel/
- OTLP Protocol: https://opentelemetry.io/docs/specs/otlp/
- Tempo Documentation: https://grafana.com/docs/tempo/
- Jaeger Documentation: https://www.jaegertracing.io/docs/
- Ambient Code Observability: See `components/runners/claude-code-runner/wrapper.py`
