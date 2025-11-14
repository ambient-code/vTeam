# OpenTelemetry Configuration for Ambient Code

This directory contains configuration for enabling Claude Code's built-in OpenTelemetry instrumentation.

## Overview

Claude Code CLI has built-in OTEL instrumentation that captures:
- **Tool executions** (Read, Write, Bash, Glob, Grep, Edit, etc.)
- **SDK lifecycle events** (session start/end, errors)
- **Tool performance** (latency per tool call)
- **File operations** and git commands
- **Error propagation** through the SDK stack

This complements Langfuse (which captures LLM-specific observability).

## Prerequisites

1. **OTEL Collector** deployed in your cluster
   - Tempo, Jaeger, or other OTLP-compatible backend
   - Accepting traces on port 4318 (OTLP/HTTP)

2. **OpenShift CLI** (`oc`) installed and logged in

## Quick Start

### 1. Update the ConfigMap with your collector endpoint

Edit `configmap.yaml` and update `OTEL_EXPORTER_OTLP_ENDPOINT`:

```yaml
data:
  OTEL_EXPORTER_OTLP_ENDPOINT: "http://your-otel-collector.your-namespace.svc.cluster.local:4318"
```

### 2. Apply the ConfigMap

```bash
cd e2e/otel
oc apply -f configmap.yaml
```

### 3. Verify configuration

```bash
oc get configmap otel-config -n ambient-code
```

That's it! The operator will automatically inject OTEL configuration into runner pods when this ConfigMap exists.

## Configuration Options

The ConfigMap supports standard OpenTelemetry environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | URL of your OTEL collector | **Required** |
| `OTEL_SERVICE_NAME` | Service name for traces | `claude-code-runner` |
| `OTEL_TRACES_EXPORTER` | Export protocol | `otlp` |
| `OTEL_RESOURCE_ATTRIBUTES` | Additional resource attributes | See configmap.yaml |
| `OTEL_TRACES_SAMPLER` | Sampling strategy | `always_on` (default) |
| `OTEL_TRACES_SAMPLER_ARG` | Sampler argument (e.g., "0.1" for 10%) | N/A |

### Example: Reduce sampling to 10%

```yaml
data:
  OTEL_TRACES_SAMPLER: "parentbased_traceidratio"
  OTEL_TRACES_SAMPLER_ARG: "0.1"
```

## How It Works

```
┌─────────────────────────────────────────┐
│  ambient-code namespace                 │
├─────────────────────────────────────────┤
│  • otel-config (ConfigMap)              │
│                                         │
│  • vteam-operator                       │
│       ↓ spawns                          │
│  • claude-runner-job-xyz                │
│       ↓ reads (via EnvFrom)             │
│       • OTEL_EXPORTER_OTLP_ENDPOINT     │
│       • OTEL_SERVICE_NAME               │
│       • OTEL_TRACES_EXPORTER            │
│       ↓ Claude Code SDK detects         │
│  • Built-in OTEL instrumentation ──────┼──→ OTEL Collector
└─────────────────────────────────────────┘
```

The operator injects this ConfigMap when it exists - no code changes needed!

## Viewing Traces

### Grafana + Tempo

1. Open Grafana
2. Navigate to Explore
3. Select Tempo data source
4. Query: `{service.name="claude-code-runner"}`

### Jaeger

1. Open Jaeger UI
2. Select service: `claude-code-runner`
3. Find traces by session ID

### Trace Attributes

Each trace includes these attributes:
- `service.name`: `claude-code-runner`
- `service.namespace`: `ambient-code`
- `session.id`: AgenticSession name
- `namespace`: Kubernetes namespace

## OTEL vs Langfuse

| Observability Type | Use OTEL | Use Langfuse |
|--------------------|----------|--------------|
| Tool call performance | ✅ | ❌ |
| File operation traces | ✅ | ❌ |
| Git command latency | ✅ | ❌ |
| SDK error details | ✅ | ❌ |
| Anthropic API calls | ❌ | ✅ |
| Token counts & costs | ❌ | ✅ |
| Prompt/response content | ❌ | ✅ |
| Model parameters | ❌ | ✅ |

**Recommendation**: Use **both** for complete observability.

## Troubleshooting

### Traces not appearing

1. **Check ConfigMap exists**:
   ```bash
   oc get configmap otel-config -n ambient-code
   ```

2. **Verify endpoint is reachable**:
   ```bash
   oc run -it --rm debug --image=curlimages/curl --restart=Never -- \
     curl -v http://your-otel-collector:4318/v1/traces
   ```

3. **Check runner pod env vars**:
   ```bash
   oc get pod <runner-pod> -o yaml | grep -A 5 envFrom
   ```

   Should show:
   ```yaml
   envFrom:
   - configMapRef:
       name: otel-config
   ```

4. **View runner logs**:
   ```bash
   oc logs <runner-pod> -c ambient-code-runner | grep -i otel
   ```

### High trace volume

Reduce sampling:

```yaml
data:
  OTEL_TRACES_SAMPLER: "parentbased_traceidratio"
  OTEL_TRACES_SAMPLER_ARG: "0.1"  # 10% sampling
```

## Disabling OTEL

To disable OTEL instrumentation:

```bash
oc delete configmap otel-config -n ambient-code
```

New runner pods will no longer export traces.

## Security Notes

- OTEL traces may contain **file paths** and **command arguments**
- Ensure your OTEL collector has appropriate **access controls**
- Consider **data retention policies** for trace storage
- Use **sampling** to reduce storage costs in production

## References

- Claude Code Monitoring: https://code.claude.com/docs/en/monitoring-usage
- OpenTelemetry Spec: https://opentelemetry.io/docs/specs/otel/
- OTLP Protocol: https://opentelemetry.io/docs/specs/otlp/
- Tempo Documentation: https://grafana.com/docs/tempo/
- Jaeger Documentation: https://www.jaegertracing.io/docs/
