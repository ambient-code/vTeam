# Missing Pieces for E2E MVP

## ✅ Fixed Critical Issues

### 1. ✅ Backend URL Construction Bug - FIXED
**Fix Applied**: Changed runner to use `{BACKEND_API_URL}/projects/...` (removed duplicate `/api`)

### 2. ✅ Postgres Secret Namespace Access - FIXED  
**Fix Applied**: Using explicit namespace in POSTGRES_HOST env var: `postgres-service.{namespace}.svc.cluster.local`
**Note**: Secret still needs to be accessible. Options:
- Copy secret to each project namespace (recommended for MVP)
- Or grant ServiceAccount permission to read secrets from `ambient-code` namespace

### 3. ⚠️ Pod Restart/Resume Logic - PARTIALLY FIXED
**Current State**: Resume endpoint handles pod restart (loads graph lazily)
**Missing**: Operator doesn't detect pod restart and call `/resume` automatically
**Workaround**: Can manually call `/resume` API, but operator should detect this

### 4. ✅ Ready Endpoint Fails Before Graph Load - FIXED
**Fix Applied**: `/ready` now returns ready=true even if graph not loaded (graph loads lazily)

### 5. ✅ Resume Logic Needs Checkpoint ID - FIXED
**Fix Applied**: Resume now calls `aget_state()` first to verify checkpoint, then updates state

## Remaining Critical Issues

### 3. Pod Restart Detection in Operator
**Location**: `components/operator/internal/handlers/sessions.go:startLangGraphWorkflow`
**Issue**: When pod restarts (backoff retry), operator should detect and call `/resume` with checkpoint_id
**Fix Needed**: 
- Check if session has `status.checkpointId` 
- If yes and pod restarted, call `/resume` instead of `/start`
- Get checkpoint_id from status before calling

### 2b. Postgres Secret Access Permissions
**Location**: ServiceAccount used by LangGraph runner pods
**Issue**: Pods need permission to read `postgres-secret` from `ambient-code` namespace
**Fix Needed**: Add RBAC to allow secret reading, OR copy secret to each project namespace

## Important Fixes (Should Fix)

### 6. Event Sequence Numbers
**Location**: `components/runners/langgraph-wrapper/runner/server.py:123`
**Issue**: Using timestamp as sequence, should use incrementing counter
**Fix**: Use atomic counter or database sequence (can defer for MVP)

### 7. Graph Compilation Logic
**Location**: `components/runners/langgraph-wrapper/runner/server.py:211-216`
**Issue**: May try to recompile already-compiled graphs
**Fix**: Check if graph already has checkpointer before recompiling (can defer)

### 8. ✅ Missing Inputs Parsing - FIXED
**Fix Applied**: Removed WORKFLOW_INPUTS env var, inputs only come from `/start` POST body

## Nice-to-Have (Can Defer)

### 9. Frontend UI
**Missing**: Complete UI for:
- Workflow registration form
- Workflow version listing
- Run creation form
- Run status monitoring
- Approval UI for interrupts

### 10. Example Workflow
**Missing**: Simple test LangGraph workflow to verify end-to-end:
- `app/workflow.py` with `build_app()` function
- Basic graph with 2-3 nodes
- One node with interrupt for approval testing

### 11. Error Handling
- Better error messages in runner server
- Retry logic for event emission failures
- Proper cleanup on pod termination

### 12. Testing
- Integration tests for workflow registration
- E2E test for workflow execution
- Pod restart recovery test

## Deployment Requirements

### 13. Base Image Build
**Action**: Build and push base wrapper image:
```bash
docker build -t quay.io/ambient_code/langgraph-wrapper:base components/runners/langgraph-wrapper/
docker push quay.io/ambient_code/langgraph-wrapper:base
```

### 14. Postgres Deployment
**Action**: Deploy Postgres manifests:
```bash
kubectl apply -k components/manifests/postgres/
```

### 15. Backend Dependencies
**Action**: Run `go mod tidy` and rebuild backend:
```bash
cd components/backend && go mod tidy && go build
```

### 16. Operator Dependencies  
**Action**: Rebuild operator (no new deps needed, HTTP client is stdlib)

