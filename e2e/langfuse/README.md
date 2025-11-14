# Langfuse Configuration for Ambient Code

This directory contains configuration files for integrating Langfuse (LLM observability platform) with the Ambient Code platform.

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

### 1. Create `.env.langfuse-keys` file

```bash
cd e2e/langfuse
cat > .env.langfuse-keys <<EOF
LANGFUSE_PUBLIC_KEY=pk-lf-your-public-key-here
LANGFUSE_SECRET_KEY=sk-lf-your-secret-key-here
EOF
```

**Important**: This file is `.gitignore`d and should NEVER be committed.

### 2. Run the configuration script

```bash
./configure-ambient-code.sh
```

This will:
- Validate your API keys
- Create Secret `langfuse-keys` in `ambient-code` namespace
- Create ConfigMap `langfuse-config` in `ambient-code` namespace

### 3. Verify configuration

```bash
oc get secret langfuse-keys -n ambient-code
oc get configmap langfuse-config -n ambient-code
```

## Files in this Directory

| File | Description | Safe to Commit? |
|------|-------------|----------------|
| `secret-template.yaml` | Template for Secret with placeholder variables | ✅ Yes |
| `configmap.yaml` | ConfigMap with Langfuse host URL | ✅ Yes |
| `configure-ambient-code.sh` | Script to create resources from templates | ✅ Yes |
| `.env.langfuse-keys` | Your actual API keys (created by you) | ❌ **NO** |
| `README.md` | This file | ✅ Yes |

## Configuration Details

### Secret: `langfuse-keys`

Contains sensitive API credentials:
```yaml
LANGFUSE_PUBLIC_KEY: "pk-lf-..."
LANGFUSE_SECRET_KEY: "sk-lf-..."
```

### ConfigMap: `langfuse-config`

Contains non-sensitive configuration:
```yaml
LANGFUSE_HOST: "http://langfuse-web.langfuse.svc.cluster.local:3000"
LANGFUSE_ENABLED: "true"
```

**Note**: We use the internal cluster URL because runner pods connect from inside the cluster.

## How It Works

```
┌─────────────────────────────────────────┐
│  ambient-code namespace                 │
├─────────────────────────────────────────┤
│  • langfuse-keys (Secret)               │
│  • langfuse-config (ConfigMap)          │
│                                         │
│  • vteam-operator                       │
│       ↓ spawns                          │
│  • claude-runner-job-xyz                │
│       ↓ reads (via EnvFrom)             │
│       • LANGFUSE_PUBLIC_KEY             │
│       • LANGFUSE_SECRET_KEY             │
│       • LANGFUSE_HOST                   │
│       • LANGFUSE_ENABLED                │
│       ↓ uses                            │
│  • Langfuse SDK sends traces ──────────┼──→ Langfuse
└─────────────────────────────────────────┘
```

The operator will inject these environment variables into Claude Code runner Job pods using `EnvFrom`.

## Next Steps

After running the configuration script:

1. **Update the operator** to inject Langfuse config into Job pods
   - Edit: `components/operator/internal/handlers/sessions.go`
   - Add `EnvFrom` for Secret and ConfigMap

2. **Update Claude Code runner** to use Langfuse SDK
   - Add dependency: `langfuse>=2.0.0` to `requirements.txt`
   - Instrument: Add Langfuse tracing in `main.py`

3. **Rebuild and redeploy** ambient-code components
   - Build: `make build-operator build-runner`
   - Deploy: `make deploy`

4. **Test** by creating an AgenticSession and viewing traces in Langfuse UI

## Updating API Keys

If you need to update your API keys:

1. Update `.env.langfuse-keys` with new keys
2. Run `./configure-ambient-code.sh` again
3. Restart any running Job pods (they'll pick up new secrets)

## Troubleshooting

### Script fails with "envsubst not found"

Install gettext:
```bash
brew install gettext
brew link --force gettext
```

### Script fails with "Not logged into OpenShift"

Log in to your cluster:
```bash
oc login https://api.your-cluster.com:6443
```

### Secrets not being picked up by pods

Check that the operator is configured to inject them:
```bash
oc get job <job-name> -n ambient-code -o yaml | grep -A 10 envFrom
```

You should see references to `langfuse-keys` and `langfuse-config`.

## Security Notes

- **Never commit** `.env.langfuse-keys` to version control
- API keys have **full access** to your Langfuse project
- For production, consider using **per-project API keys** (Phase 3)
- Kubernetes Secrets are **base64 encoded** (not encrypted at rest by default)

## Multi-Tenant Setup (Future)

For Phase 3 multi-tenancy:
- Each project namespace gets its own `langfuse-keys` Secret
- Operator injects from the namespace where the Job runs
- Enables per-project isolation and cost tracking

## References

- Langfuse Documentation: https://langfuse.com/docs
- Langfuse Python SDK: https://langfuse.com/docs/sdk/python
- Ambient Code Docs: `../../docs/deployment/langfuse-phase2-context.md`
