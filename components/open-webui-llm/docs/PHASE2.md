# Phase 2: Production Migration Plan

This document outlines the migration from Phase 1 (simple dev deployment) to Phase 2 (production-ready with OAuth and Claude service).

## Status: PLANNED (Not Yet Implemented)

Phase 2 is documented but not yet built. This serves as a design spec and migration guide for future implementation.

## What Phase 2 Adds

### Security & Authentication
- ✅ OAuth2 proxy with OpenShift OAuth or generic OIDC
- ✅ User authentication required for UI access
- ✅ API key rotation support
- ✅ Network policies (restrict egress)

### Claude Integration
- ✅ Long-running Claude Code service
- ✅ Multi-session management (one per user)
- ✅ Tool execution (code, bash, file operations)
- ✅ Amber agent persona integration

### Production Hardening
- ✅ Kubernetes Secrets (replace ConfigMaps)
- ✅ Resource quotas and limits
- ✅ High availability (multiple replicas)
- ✅ Monitoring and observability

### OpenShift Compatibility
- ✅ OpenShift Routes (instead of Ingress)
- ✅ SecurityContextConstraints compliance
- ✅ Service accounts with proper RBAC

## Architecture Changes

### Phase 1 Flow
```
User → Ingress → Open WebUI → LiteLLM → Anthropic API
```

### Phase 2 Flow
```
User → Route/Ingress → OAuth Proxy → Open WebUI → LiteLLM →
  ├→ Anthropic API (direct models)
  └→ Claude Service → Anthropic API (Amber sessions)
```

## Migration Strategy

### Option A: In-Place Migration (Recommended)

Upgrade existing Phase 1 deployment with minimal downtime.

**Steps**:
1. Backup Open WebUI data (PVC snapshot or export)
2. Create Phase 2 secrets (OAuth, Claude service)
3. Apply Phase 2 overlay (patches existing deployments)
4. Update DNS/Ingress (route to OAuth proxy)
5. Test authentication and Claude service
6. Rollback if issues (revert to Phase 1 overlay)

**Downtime**: ~5-10 minutes (during OAuth proxy deployment)

**Pros**:
- Preserves chat history and user data
- Faster migration (no new cluster setup)
- Easier rollback

**Cons**:
- Risk of breaking existing deployment
- Harder to test before migration

### Option B: Parallel Deployment (Safer)

Deploy Phase 2 to new namespace, test, then cutover.

**Steps**:
1. Deploy Phase 2 to `openwebui-prod` namespace
2. Export Phase 1 data (chat history, settings)
3. Import data into Phase 2
4. Test Phase 2 thoroughly
5. Update DNS to point to Phase 2
6. Deprecate Phase 1 after validation period

**Downtime**: None (parallel systems)

**Pros**:
- No risk to Phase 1
- Full testing before cutover
- Easy rollback (just revert DNS)

**Cons**:
- Requires double resources temporarily
- More complex data migration
- Manual cutover process

## Implementation Checklist

### Prerequisites

- [ ] Decide OAuth provider (OpenShift OAuth or generic OIDC)
- [ ] Obtain OAuth client credentials
- [ ] Allocate cluster resources (2x current for parallel deployment)
- [ ] Plan downtime window (if in-place migration)

### Step 1: Create Phase 2 Directory Structure

```bash
cd components/open-webui-llm

mkdir -p overlays/phase2-production/{claude-service,secrets}
```

### Step 2: Build Claude Service

**Files to create**:
- `overlays/phase2-production/claude-service/deployment.yaml`
- `overlays/phase2-production/claude-service/service.yaml`
- `overlays/phase2-production/claude-service/configmap.yaml`

**Claude Service Features**:
- FastAPI server with `/v1/chat/completions` endpoint
- Session management (create, resume, list)
- Streaming responses via SSE
- Tool execution (bash, code, file ops)
- Integration with Amber persona from `agents/amber.md`

**Example implementation** (pseudocode):
```python
# claude-service/main.py
from fastapi import FastAPI
from anthropic import Anthropic

app = FastAPI()
client = Anthropic(api_key=os.environ["ANTHROPIC_API_KEY"])

@app.post("/v1/chat/completions")
async def chat(request: ChatRequest):
    # Create or resume Claude session
    session = get_or_create_session(request.user_id)

    # Stream response
    async for chunk in client.messages.stream(...):
        yield format_openai_chunk(chunk)
```

### Step 3: Configure OAuth Proxy

**File**: `overlays/phase2-production/oauth-deployment-patch.yaml`

**Adds sidecar to Open WebUI**:
```yaml
containers:
- name: oauth-proxy
  image: quay.io/oauth2-proxy/oauth2-proxy:latest
  args:
  - --provider=oidc
  - --upstream=http://localhost:8080
  - --cookie-secret=$(COOKIE_SECRET)
  ...
```

**OAuth Provider Options**:

1. **OpenShift OAuth** (recommended for OpenShift):
   ```yaml
   --provider=openshift
   --login-url=https://oauth-openshift.apps.cluster/oauth/authorize
   ```

2. **Generic OIDC** (Google, Okta, etc.):
   ```yaml
   --provider=oidc
   --oidc-issuer-url=https://accounts.google.com
   --client-id=...
   --client-secret=...
   ```

### Step 4: Update LiteLLM Configuration

**File**: `base/litellm/configmap.yaml` (add Claude service route)

```yaml
model_list:
  # Existing direct routes
  - model_name: claude-sonnet-4-5
    litellm_params:
      model: anthropic/claude-sonnet-4-5-20250929
      api_key: os.environ/ANTHROPIC_API_KEY

  # NEW: Claude service route
  - model_name: claude-amber-session
    litellm_params:
      model: openai/gpt-3.5-turbo  # Proxy format
      api_base: http://claude-service:8001/v1
      api_key: internal-token
```

### Step 5: Create Secrets

**File**: `overlays/phase2-production/secrets/secrets.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: oauth-config
type: Opaque
stringData:
  cookie-secret: "<generated-random-32-bytes>"
  client-id: "openwebui-client"
  client-secret: "<oauth-client-secret>"
---
apiVersion: v1
kind: Secret
metadata:
  name: claude-service-secrets
type: Opaque
stringData:
  ANTHROPIC_API_KEY: "sk-ant-..."
  CLAUDE_SERVICE_KEY: "internal-token"
```

**Generate secrets**:
```bash
# Cookie secret (32 bytes)
openssl rand -base64 32

# OAuth client secret (if not provided by provider)
openssl rand -hex 32
```

### Step 6: Create Phase 2 Kustomization

**File**: `overlays/phase2-production/kustomization.yaml`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: openwebui  # Or openwebui-prod for parallel deployment

resources:
- ../../base
- claude-service/deployment.yaml
- claude-service/service.yaml
- claude-service/configmap.yaml
- ingress.yaml  # Or route.yaml for OpenShift
- secrets/secrets.yaml

patches:
- path: oauth-deployment-patch.yaml
  target:
    kind: Deployment
    name: openwebui
- path: oauth-service-patch.yaml
  target:
    kind: Service
    name: openwebui-service
```

### Step 7: Test Phase 2 Locally

**Build Claude service image**:
```bash
cd docker/claude-service
podman build -t localhost/claude-service:dev .
```

**Deploy to Kind** (for testing):
```bash
cd ../../overlays/phase2-production
kustomize build . | kubectl apply -f -
```

**Test checklist**:
- [ ] OAuth login redirects correctly
- [ ] Can access Open WebUI after auth
- [ ] Claude direct models still work
- [ ] Claude service endpoint is reachable
- [ ] Amber session model appears in dropdown
- [ ] Can create long-running session
- [ ] Session persists across pod restarts

### Step 8: Production Deployment

**For OpenShift**:
```bash
# Create OAuthClient (requires cluster-admin)
oc apply -f oauth-client.yaml

# Deploy Phase 2
cd components/open-webui-llm
kubectl apply -k overlays/phase2-production

# Update Route hostname (if needed)
oc patch route openwebui -n openwebui -p '{"spec":{"host":"openwebui.apps.cluster.example.com"}}'
```

**For Kind** (with Ingress):
```bash
kubectl apply -k overlays/phase2-production

# Update /etc/hosts
echo "127.0.0.1 openwebui.local" | sudo tee -a /etc/hosts
```

### Step 9: Data Migration

**Export Phase 1 data**:
```bash
# Backup PVC
kubectl exec -n openwebui deployment/openwebui -- \
  tar czf /tmp/backup.tar.gz /app/backend/data

kubectl cp openwebui/openwebui-xxxxx:/tmp/backup.tar.gz ./backup.tar.gz
```

**Import to Phase 2** (if using new namespace):
```bash
kubectl cp ./backup.tar.gz openwebui-prod/openwebui-xxxxx:/tmp/backup.tar.gz

kubectl exec -n openwebui-prod deployment/openwebui -- \
  tar xzf /tmp/backup.tar.gz -C /
```

### Step 10: Validation

**Smoke tests**:
```bash
# Test OAuth flow
curl -I https://openwebui.apps.cluster.example.com
# Should redirect to OAuth provider

# Test LiteLLM health
kubectl exec -n openwebui deployment/litellm -- curl localhost:4000/health

# Test Claude service health
kubectl exec -n openwebui deployment/claude-service -- curl localhost:8001/health

# Test end-to-end (requires valid OAuth token)
curl https://openwebui.apps.cluster.example.com/api/chat \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"model":"claude-amber-session", "messages":[...]}'
```

## Rollback Plan

### If Phase 2 Fails (In-Place Migration)

```bash
# Immediate rollback to Phase 1
kubectl apply -k overlays/phase1-kind

# Verify Phase 1 is working
make phase1-test

# Restore data if needed
kubectl cp ./backup.tar.gz openwebui/openwebui-xxxxx:/tmp/backup.tar.gz
kubectl exec -n openwebui deployment/openwebui -- \
  tar xzf /tmp/backup.tar.gz -C /
```

### If Phase 2 Fails (Parallel Deployment)

```bash
# Revert DNS to Phase 1
# (No kubectl changes needed, just update DNS/Route)

# Delete Phase 2 namespace
kubectl delete namespace openwebui-prod
```

## Technical Debt to Address

### Immediate (Before Production)

1. **External Secrets Operator**: Move from Kubernetes Secrets to Vault/AWS Secrets Manager
2. **Network Policies**: Restrict egress to only Anthropic API
3. **Resource Quotas**: Enforce limits per namespace
4. **Monitoring**: Add Prometheus metrics, Grafana dashboards

### Future Enhancements

5. **Multi-tenancy**: Namespace-per-team or namespace-per-user
6. **Rate Limiting**: Per-user API call limits
7. **Cost Tracking**: Track Anthropic API usage per user/team
8. **Audit Logging**: Log all chat sessions for compliance
9. **High Availability**: Multiple replicas with pod disruption budgets
10. **Auto-scaling**: HPA based on request volume

## Timeline Estimate

**Assuming one developer, part-time**:

- **Week 1-2**: Build Claude service (API, session management, tool execution)
- **Week 3**: OAuth integration and testing
- **Week 4**: Phase 2 overlay and Kustomize patches
- **Week 5**: Testing and documentation
- **Week 6**: Production deployment and validation

**Total**: 6-8 weeks (30-40 hours)

## Questions to Resolve

Before implementing Phase 2, decide:

1. **OAuth Provider**: OpenShift OAuth or generic OIDC?
2. **Session Storage**: PostgreSQL or file-based (PVC)?
3. **Claude Service Language**: Python (FastAPI) or Go?
4. **Deployment Strategy**: In-place or parallel?
5. **Namespace**: Reuse `openwebui` or create `openwebui-prod`?

## Next Steps

1. Review this plan with team
2. Create GitHub issues for each step
3. Build Claude service POC
4. Test OAuth locally
5. Create Phase 2 overlay structure
6. Schedule production deployment window

## References

- [OAuth2 Proxy Documentation](https://oauth2-proxy.github.io/oauth2-proxy/)
- [OpenShift OAuth](https://docs.openshift.com/container-platform/4.14/authentication/configuring-oauth-clients.html)
- [Claude API Documentation](https://docs.anthropic.com/claude/reference/getting-started-with-the-api)
- [Kustomize Overlays](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/)
