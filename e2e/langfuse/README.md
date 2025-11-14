# Langfuse Configuration for Ambient Code

This directory contains scripts and documentation for integrating Langfuse (LLM observability platform) with the Ambient Code platform.

## Overview

Langfuse provides observability for AI applications by tracking:
- LLM traces (prompts, responses, tokens)
- Performance metrics (latency, cost)
- Analytics and dashboards

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

**All observability configuration is now managed via the WorkspaceSettings UI in the frontend.**

1. **Access WorkspaceSettings** for your project:
   - Navigate to your workspace
   - Go to Settings tab
   - Expand "Observability (Langfuse + OpenTelemetry)" section

2. **Configure Langfuse** (pre-populated with defaults):
   - `LANGFUSE_ENABLED`: `true` (already set)
   - `LANGFUSE_HOST`: `http://langfuse-web.langfuse.svc.cluster.local:3000` (already set)
   - `LANGFUSE_PUBLIC_KEY`: Add your `pk-lf-...` key
   - `LANGFUSE_SECRET_KEY`: Add your `sk-lf-...` key

3. **Save Integration Secrets** - Saves all observability keys to `ambient-non-vertex-integrations` secret

### Using Scripts (Legacy)

The `configure-ambient-code.sh` script still works but creates separate secrets that require manual namespace management:

```bash
# Create .env.langfuse-keys file
cat > .env.langfuse-keys <<EOF
LANGFUSE_PUBLIC_KEY=pk-lf-your-public-key-here
LANGFUSE_SECRET_KEY=sk-lf-your-secret-key-here
EOF

# Run configuration script
./configure-ambient-code.sh
```

**Recommendation**: Use WorkspaceSettings UI instead for better multi-project support.

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
│       • LANGFUSE_PUBLIC_KEY             │
│       • LANGFUSE_SECRET_KEY             │
│       • LANGFUSE_HOST                   │
│       • LANGFUSE_ENABLED                │
│       ↓ Langfuse SDK                    │
│  • Traces sent to Langfuse ────────────┼──→ Langfuse
└─────────────────────────────────────────┘
```

The operator automatically injects all keys from `ambient-non-vertex-integrations` into runner pods using `EnvFrom`.

## Configuration Details

All observability environment variables are stored in the `ambient-non-vertex-integrations` secret:

### Langfuse Keys (Required for Traces)

```yaml
LANGFUSE_PUBLIC_KEY: "pk-lf-..."
LANGFUSE_SECRET_KEY: "sk-lf-..."
```

### Langfuse Configuration (Pre-configured)

```yaml
LANGFUSE_ENABLED: "true"
LANGFUSE_HOST: "http://langfuse-web.langfuse.svc.cluster.local:3000"
```

**Note**: The internal cluster URL is used because runner pods connect from inside the cluster.

## Trace Hierarchy

The runner creates rich Langfuse traces with:

1. **Session Span** - Main span for the entire Claude session
   - Input: Original prompt
   - Output: Final results with cost/token metrics
   - Metadata: session ID, namespace, timestamp

2. **Tool Spans** - Child spans for each tool execution (Read, Write, Bash, etc.)
   - Input: Tool parameters
   - Output: Tool results (truncated to 500 chars)
   - Metadata: tool name, tool ID, turn number

3. **Generation Spans** - Claude's text responses
   - Input: Turn number
   - Output: Claude's text (truncated to 1000 chars)
   - Metadata: Model name, turn number

## Viewing Traces

1. Open Langfuse UI: https://langfuse-langfuse.apps.<your-cluster>
2. Navigate to your project
3. View traces by session ID
4. Drill down into spans to see tool calls and generations

## Updating API Keys

To update your Langfuse API keys:

1. Go to WorkspaceSettings → Settings → Observability
2. Update `LANGFUSE_PUBLIC_KEY` and `LANGFUSE_SECRET_KEY`
3. Click "Save Integration Secrets"
4. New sessions will use the updated keys immediately

## Troubleshooting

### No traces appearing in Langfuse

1. **Check API keys are set**:
   ```bash
   oc get secret ambient-non-vertex-integrations -n ambient-code -o yaml | grep LANGFUSE
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

3. **Check runner logs for Langfuse initialization**:
   ```bash
   oc logs <runner-pod> -c ambient-code-runner | grep -i langfuse
   ```

   Should see: `Langfuse tracing enabled for session`

### Traces show errors

Check that `LANGFUSE_HOST` points to the correct service:
- Internal URL (from pods): `http://langfuse-web.langfuse.svc.cluster.local:3000`
- External URL (from browser): `https://langfuse-langfuse.apps.<your-cluster>`

Runner pods should use the **internal** URL.

## Security Notes

- **Never commit** API keys to version control
- API keys have **full access** to your Langfuse project
- Kubernetes Secrets are **base64 encoded** (not encrypted at rest by default)
- For production, consider using **Sealed Secrets** or **External Secrets Operator**

## Multi-Project Setup

Each project namespace can have its own `ambient-non-vertex-integrations` secret with different Langfuse keys for per-project isolation and cost tracking.

## References

- Langfuse Documentation: https://langfuse.com/docs
- Langfuse Python SDK: https://langfuse.com/docs/sdk/python
- Ambient Code Observability: See `components/runners/claude-code-runner/wrapper.py`
