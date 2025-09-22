# Local Dev One-Shot (Option A) - CURRENT LIMITATIONS

## Status: ⚠️ PARTIALLY WORKING

The backend is currently **OpenShift-specific** and hardcoded to use OpenShift Project resources (`project.openshift.io/v1`). This prevents full functionality on Kind/vanilla Kubernetes.

## What Works
- ✅ Backend health endpoint (`/health`)
- ✅ Frontend UI loads
- ✅ Kind cluster + CRDs setup
- ✅ Namespace creation and labeling

## What Doesn't Work
- ❌ `/api/projects` endpoint (crashes with nil pointer - tries to list OpenShift Projects)
- ❌ All project-scoped APIs (depend on OpenShift Project validation)
- ❌ Full e2e workflow testing

## Root Cause
Backend hardcoded assumptions:
1. Uses `project.openshift.io/v1` instead of `v1/namespaces`
2. Requires user tokens (no service account fallback)
3. OpenShift-specific RBAC patterns

## Usage (Limited)

### Start
```bash
make dev-start
```
- Creates Kind cluster `ambient-agentic`
- Applies CRDs
- Starts backend on :8080 (health works, APIs crash)
- Starts frontend on :3000 (loads but can't call backend APIs)

### Test
```bash
make dev-test
```
- ✅ Backend health
- ✅ Frontend reachable  
- ❌ Projects API (500 error)

### Stop
```bash
make dev-stop
```

## Next Steps (Backend Changes Needed)
1. Add cluster detection (OpenShift vs vanilla K8s)
2. Use `v1/namespaces` when not on OpenShift
3. Add optional service account fallback for local dev
4. Make RBAC patterns cluster-agnostic

## Workaround for Now
Use the full OpenShift deployment method instead:
```bash
cd components/manifests
cp env.example .env
# Edit .env with your Anthropic key
./deploy.sh
```

This will work on OpenShift or with OpenShift Local (CRC).