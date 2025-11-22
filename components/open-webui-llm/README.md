# Open WebUI + LiteLLM Deployment

A phased deployment of Open WebUI with LiteLLM proxy for chatting with Claude models, designed to work with the Ambient Code Platform's Kind cluster.

## Architecture

- **Phase 1**: Open WebUI → LiteLLM → Anthropic Claude API (simple proxy, no auth)
- **Phase 2** (Future): Long-running Claude service for Amber agent integration

## Quick Start (Phase 1)

### Prerequisites

1. **Kind cluster running** with nginx-ingress:
   ```bash
   cd ../../e2e
   ./scripts/setup-kind.sh
   # Or if using Podman: CONTAINER_ENGINE=podman ./scripts/setup-kind.sh
   ```

2. **Anthropic API key**: Get yours from [console.anthropic.com](https://console.anthropic.com)

### Deploy

1. **Configure API key**:
   ```bash
   cd overlays/phase1-kind

   # Edit secrets.yaml and replace sk-ant-YOUR-KEY-HERE with your actual key
   # Or use sed:
   sed -i.bak 's/sk-ant-YOUR-KEY-HERE/sk-ant-api01-YOUR-ACTUAL-KEY/g' secrets.yaml
   ```

2. **Deploy to Kind**:
   ```bash
   cd ../..  # Back to components/open-webui-llm/
   make phase1-deploy
   ```

3. **Wait for pods** (automatic, but you can check):
   ```bash
   make phase1-status
   ```

4. **Access Open WebUI**:
   - **Docker**: http://vteam.local/chat
   - **Podman**: http://vteam.local:8080/chat

### Usage

1. Open the URL in your browser
2. No login required (Phase 1 has auth disabled)
3. Select a model from the dropdown:
   - `claude-sonnet-4-5` (recommended)
   - `claude-sonnet-3-7`
   - `claude-haiku-3-5`
4. Start chatting!

## Management Commands

```bash
# View logs
make phase1-logs              # Open WebUI logs
make phase1-logs-litellm      # LiteLLM logs

# Check status
make phase1-status            # All resources

# Run health checks
make phase1-test              # Verify LiteLLM and Open WebUI connectivity

# Clean up
make phase1-clean             # Remove all resources
```

## Troubleshooting

### Pods not starting

```bash
# Check pod status
kubectl get pods -n openwebui

# View pod logs
kubectl logs -n openwebui deployment/openwebui
kubectl logs -n openwebui deployment/litellm

# Describe pod for events
kubectl describe pod -n openwebui -l app=openwebui
```

### LiteLLM errors

**"No API key provided"**:
- Check secrets.yaml has your actual Anthropic API key
- Verify secret was created: `kubectl get secret litellm-secrets -n openwebui -o yaml`

**"Model not found"**:
- Check LiteLLM config: `kubectl get cm litellm-config -n openwebui -o yaml`
- Verify model names match Anthropic's API

### Ingress not working

**Docker** (ports 80/443):
```bash
# Verify vteam.local resolves to 127.0.0.1
grep vteam.local /etc/hosts

# Test ingress
curl http://vteam.local/chat
```

**Podman** (ports 8080/8443):
```bash
# Use port 8080
curl http://vteam.local:8080/chat
```

**Fallback - Port forwarding**:
```bash
# Access via localhost instead
make phase1-port-forward
# Then open: http://localhost:8080
```

### PVC not binding

```bash
# Check PVC status
kubectl get pvc -n openwebui

# If pending, check storage class
kubectl get sc

# Kind should have 'standard' storage class by default
```

## Component Structure

```
.
├── base/                      # Shared base manifests
│   ├── namespace.yaml
│   ├── rbac.yaml             # ServiceAccounts
│   ├── litellm/              # LiteLLM proxy
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── configmap.yaml    # Model routing
│   ├── open-webui/           # Web UI
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── pvc.yaml          # Persistent storage
│   └── kustomization.yaml
│
├── overlays/
│   ├── phase1-kind/          # Phase 1: Simple deployment
│   │   ├── kustomization.yaml
│   │   ├── secrets.yaml      # API keys (edit this!)
│   │   ├── ingress.yaml      # Nginx ingress
│   │   └── pvc-patch.yaml    # Reduced storage for Kind
│   │
│   └── phase2-production/    # Phase 2: Future (OAuth, Claude service)
│       └── (planned)
│
├── docs/
│   ├── PHASE1.md             # Detailed Phase 1 guide
│   └── PHASE2.md             # Phase 2 migration plan
│
├── Makefile                  # Deployment automation
└── README.md                 # This file
```

## Data Flow

```
User Browser → vteam.local/chat → Nginx Ingress → Open WebUI Service
  → Open WebUI Pod → LiteLLM Service → LiteLLM Pod → Anthropic API
```

## Phase 2 (Future)

Phase 2 will add:
- **Authentication**: OAuth2 proxy for production use
- **Claude Service**: Long-running Claude Code sessions
- **Amber Integration**: Direct integration with Amber agent
- **Production deployment**: OpenShift Routes, proper RBAC

See `docs/PHASE2.md` for migration plan (coming soon).

## Files You May Need to Edit

- **`overlays/phase1-kind/secrets.yaml`**: Add your Anthropic API key here (required)
- **`base/litellm/configmap.yaml`**: Add more models or adjust LiteLLM settings
- **`base/open-webui/deployment.yaml`**: Change resource limits or add environment variables

## Clean Up

```bash
# Remove deployment but keep namespace
make phase1-clean

# Remove namespace too
kubectl delete namespace openwebui
```

## Next Steps

1. Try chatting with different Claude models
2. Explore Open WebUI settings (http://vteam.local/chat/settings)
3. Review LiteLLM logs to see API calls: `make phase1-logs-litellm`
4. Plan for Phase 2 migration (see `docs/PHASE2.md`)

## Support

- **Documentation**: See `docs/` directory
- **Issues**: Create an issue in the main repository
- **Logs**: Always check logs first: `make phase1-logs`
