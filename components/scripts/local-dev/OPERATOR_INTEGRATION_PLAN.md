# Plan: Add Operator Build & Deployment to CRC Local Dev

## Overview
Integrate the vTeam operator into the `crc-start.sh` local development workflow, following the same patterns used for backend and frontend components.

## Current State Analysis

### What's Already Working
- ✅ Backend build and deployment via BuildConfig
- ✅ Frontend build and deployment via BuildConfig
- ✅ CRD application (agenticsessions, projectsettings)
- ✅ RBAC for backend service account
- ✅ Operator Dockerfile exists (`components/operator/Dockerfile`)
- ✅ Operator manifests exist (`components/manifests/operator-deployment.yaml`)

### What's Missing
- ❌ Operator BuildConfig for local builds
- ❌ Operator ImageStream
- ❌ Operator RBAC (ServiceAccount, ClusterRole, ClusterRoleBinding) adapted for local dev
- ❌ Operator deployment step in `crc-start.sh`
- ❌ Operator build step in `crc-start.sh`

## Implementation Plan

### 1. Create Operator BuildConfig Manifest
**File**: `components/scripts/local-dev/manifests/operator-build-config.yaml`

**Content**:
```yaml
---
apiVersion: image.openshift.io/v1
kind: ImageStream
metadata:
  name: vteam-operator
  labels:
    app: vteam-operator
---
apiVersion: build.openshift.io/v1
kind: BuildConfig
metadata:
  name: vteam-operator
  labels:
    app: vteam-operator
spec:
  source:
    type: Binary
  strategy:
    type: Docker
    dockerStrategy:
      dockerfilePath: Dockerfile
  output:
    to:
      kind: ImageStreamTag
      name: vteam-operator:latest
```

**Rationale**: Follows exact same pattern as backend/frontend in `build-configs.yaml`

### 2. Create Operator RBAC Manifest for Local Dev
**File**: `components/scripts/local-dev/manifests/operator-rbac.yaml`

**Content**:
```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: agentic-operator
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: agentic-operator-local
rules:
# AgenticSession custom resources
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions/status"]
  verbs: ["update"]
# ProjectSettings custom resources
- apiGroups: ["vteam.ambient-code"]
  resources: ["projectsettings"]
  verbs: ["get", "list", "watch", "create"]
- apiGroups: ["vteam.ambient-code"]
  resources: ["projectsettings/status"]
  verbs: ["update"]
# Namespaces (watch for managed namespaces)
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "watch"]
# Jobs (create and monitor)
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["get", "create"]
# Pods (for job logs)
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["list"]
- apiGroups: [""]
  resources: ["pods/log"]
  verbs: ["get"]
# PVCs (create workspace PVCs)
- apiGroups: [""]
  resources: ["persistentvolumeclaims"]
  verbs: ["get", "create"]
# Services and Deployments (for content service)
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get", "create"]
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["create"]
# RoleBindings (group access)
- apiGroups: ["rbac.authorization.k8s.io"]
  resources: ["rolebindings"]
  verbs: ["get", "create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: agentic-operator-local
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: agentic-operator-local
subjects:
- kind: ServiceAccount
  name: agentic-operator
  namespace: vteam-dev
```

**Rationale**: 
- Based on production `operator-clusterrole.yaml` but adapted for local namespace
- Uses same naming pattern as `backend-api-local` ClusterRole

### 3. Create Operator Deployment Manifest for Local Dev
**File**: `components/scripts/local-dev/manifests/operator-deployment.yaml`

**Content**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vteam-operator
  labels:
    app: vteam-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: vteam-operator
  template:
    metadata:
      labels:
        app: vteam-operator
    spec:
      serviceAccountName: agentic-operator
      containers:
      - name: operator
        image: image-registry.openshift-image-registry.svc:5000/vteam-dev/vteam-operator:latest
        imagePullPolicy: Always
        env:
        - name: NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: BACKEND_NAMESPACE
          value: "vteam-dev"
        - name: AMBIENT_CODE_RUNNER_IMAGE
          # For local dev, point to local registry or use external image
          value: "quay.io/ambient_code/vteam_claude_runner:latest"
        - name: CONTENT_SERVICE_IMAGE
          # Use locally built backend image for content service
          value: "image-registry.openshift-image-registry.svc:5000/vteam-dev/vteam-backend:latest"
        - name: IMAGE_PULL_POLICY
          value: "IfNotPresent"
        resources:
          requests:
            cpu: 50m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 256Mi
      restartPolicy: Always
```

**Rationale**:
- Uses local ImageStream reference (like backend/frontend deployments)
- Points to local backend image for content service
- Uses external runner image (can be built locally later if needed)
- Environment variables match local namespace

### 4. Update `crc-start.sh` Script

**Location**: Line 262-266 (after `apply_rbac()` function)

**Add new function**:
```bash
apply_operator_rbac() {
  log "Applying operator RBAC (service account and permissions)..."
  oc apply -f "${MANIFESTS_DIR}/operator-rbac.yaml" -n "$PROJECT_NAME"
}
```

**Location**: Line 286-293 (in `build_and_deploy()` function)

**Add operator build steps AFTER frontend build**:
```bash
  log "Building operator image..."
  oc start-build vteam-operator --from-dir="$OPERATOR_DIR" --wait -n "$PROJECT_NAME"
```

**Add operator deployment step AFTER frontend deployment**:
```bash
  log "Deploying operator..."
  oc apply -f "${MANIFESTS_DIR}/operator-deployment.yaml" -n "$PROJECT_NAME"
```

**Location**: Line 15 (add to configuration section)
```bash
OPERATOR_DIR="${REPO_ROOT}/components/operator"
```

**Location**: Line 286 (update BuildConfigs application)
```bash
build_and_deploy() {
  log "Creating BuildConfigs..."
  oc apply -f "${MANIFESTS_DIR}/build-configs.yaml" -n "$PROJECT_NAME"
  oc apply -f "${MANIFESTS_DIR}/operator-build-config.yaml" -n "$PROJECT_NAME"
  
  # Start builds
  log "Building backend image..."
  oc start-build vteam-backend --from-dir="$BACKEND_DIR" --wait -n "$PROJECT_NAME"
  
  log "Building frontend image..."  
  oc start-build vteam-frontend --from-dir="$FRONTEND_DIR" --wait -n "$PROJECT_NAME"
  
  log "Building operator image..."
  oc start-build vteam-operator --from-dir="$OPERATOR_DIR" --wait -n "$PROJECT_NAME"
  
  # Deploy services
  log "Deploying backend..."
  oc apply -f "${MANIFESTS_DIR}/backend-deployment.yaml" -n "$PROJECT_NAME"
  
  log "Deploying frontend..."
  oc apply -f "${MANIFESTS_DIR}/frontend-deployment.yaml" -n "$PROJECT_NAME"
  
  log "Deploying operator..."
  oc apply -f "${MANIFESTS_DIR}/operator-deployment.yaml" -n "$PROJECT_NAME"
}
```

**Location**: Line 305 (update wait_for_ready)
```bash
wait_for_ready() {
  log "Waiting for deployments to be ready..."
  oc rollout status deployment/vteam-backend --timeout=300s -n "$PROJECT_NAME"
  oc rollout status deployment/vteam-frontend --timeout=300s -n "$PROJECT_NAME"
  oc rollout status deployment/vteam-operator --timeout=300s -n "$PROJECT_NAME"
}
```

**Location**: Line 352 (update execution order)
```bash
ensure_project
apply_crds
apply_rbac
apply_operator_rbac  # ADD THIS LINE
build_and_deploy
wait_for_ready
show_results
```

### 5. Update `crc-test.sh` - Test-Driven Development Approach

Following TDD principles, **write these tests FIRST**, then implement operator integration to make them pass.

**Add operator test functions** (insert after line 188):

```bash
#########################
# Operator Tests
#########################
test_operator_deployment_exists() {
  oc get deployment vteam-operator -n "$PROJECT_NAME" >/dev/null 2>&1
}

test_operator_pod_running() {
  local operator_ready
  operator_ready=$(oc get deployment vteam-operator -n "$PROJECT_NAME" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
  [[ "$operator_ready" -gt 0 ]]
}

test_operator_service_account() {
  oc get serviceaccount agentic-operator -n "$PROJECT_NAME" >/dev/null 2>&1
}

test_operator_rbac_configured() {
  # Check ClusterRole exists
  oc get clusterrole agentic-operator-local >/dev/null 2>&1 &&
  # Check ClusterRoleBinding exists
  oc get clusterrolebinding agentic-operator-local >/dev/null 2>&1
}

test_operator_watching_sessions() {
  # Check operator logs for watcher initialization
  local operator_pod
  operator_pod=$(oc get pods -n "$PROJECT_NAME" -l app=vteam-operator -o name 2>/dev/null | head -n 1)
  
  [[ -n "$operator_pod" ]] || return 1
  
  # Look for log messages indicating watchers started
  oc logs "$operator_pod" -n "$PROJECT_NAME" --tail=100 2>/dev/null | \
    grep -q "Watching for AgenticSession events"
}

test_operator_workspace_pvc_created() {
  # Operator should create ambient-workspace PVC when namespace is labeled
  oc get pvc ambient-workspace -n "$PROJECT_NAME" >/dev/null 2>&1
}

test_operator_content_service_deployed() {
  # Operator should create ambient-content service
  oc get service ambient-content -n "$PROJECT_NAME" >/dev/null 2>&1 &&
  oc get deployment ambient-content -n "$PROJECT_NAME" >/dev/null 2>&1
}

test_operator_projectsettings_created() {
  # Operator should auto-create ProjectSettings singleton
  oc get projectsettings projectsettings -n "$PROJECT_NAME" >/dev/null 2>&1
}

test_operator_can_create_session_job() {
  # Create a test AgenticSession and verify operator creates a Job
  local test_session="test-session-$$"
  
  # Create test session
  cat <<EOF | oc apply -f - >/dev/null 2>&1
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: ${test_session}
  namespace: ${PROJECT_NAME}
spec:
  prompt: "echo 'test session'"
  timeout: 300
  interactive: false
  llmSettings:
    model: "claude-sonnet-4-20250514"
    temperature: 0.7
    maxTokens: 4096
EOF
  
  # Wait for operator to create job (up to 30 seconds)
  local timeout=30
  local elapsed=0
  local job_created=false
  
  while [[ $elapsed -lt $timeout ]]; do
    if oc get job "${test_session}-job" -n "$PROJECT_NAME" >/dev/null 2>&1; then
      job_created=true
      break
    fi
    sleep 2
    elapsed=$((elapsed + 2))
  done
  
  # Cleanup test session
  oc delete agenticsession "$test_session" -n "$PROJECT_NAME" >/dev/null 2>&1 || true
  
  [[ "$job_created" == "true" ]]
}

test_operator_updates_session_status() {
  # Create a test session and verify operator updates its status
  local test_session="test-status-$$"
  
  cat <<EOF | oc apply -f - >/dev/null 2>&1
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: ${test_session}
  namespace: ${PROJECT_NAME}
spec:
  prompt: "echo 'test'"
  timeout: 300
  interactive: false
  llmSettings:
    model: "claude-sonnet-4-20250514"
    temperature: 0.7
    maxTokens: 4096
EOF
  
  # Wait for status update (operator should set phase to at least "Creating")
  local timeout=30
  local elapsed=0
  local status_updated=false
  
  while [[ $elapsed -lt $timeout ]]; do
    local phase
    phase=$(oc get agenticsession "$test_session" -n "$PROJECT_NAME" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    
    if [[ -n "$phase" ]] && [[ "$phase" != "null" ]]; then
      status_updated=true
      break
    fi
    sleep 2
    elapsed=$((elapsed + 2))
  done
  
  # Cleanup
  oc delete agenticsession "$test_session" -n "$PROJECT_NAME" >/dev/null 2>&1 || true
  
  [[ "$status_updated" == "true" ]]
}

test_operator_handles_managed_namespace_label() {
  # Verify the vteam-dev namespace has the managed label
  local label
  label=$(oc get namespace "$PROJECT_NAME" -o jsonpath='{.metadata.labels.ambient-code\.io/managed}' 2>/dev/null || echo "")
  [[ "$label" == "true" ]]
}

test_operator_logs_no_errors() {
  # Check operator logs for critical errors (not warnings)
  local operator_pod
  operator_pod=$(oc get pods -n "$PROJECT_NAME" -l app=vteam-operator -o name 2>/dev/null | head -n 1)
  
  [[ -n "$operator_pod" ]] || return 1
  
  # Look for error patterns (excluding expected informational messages)
  local error_count
  error_count=$(oc logs "$operator_pod" -n "$PROJECT_NAME" --tail=200 2>/dev/null | \
    grep -iE "error|fatal|panic" | \
    grep -viE "watching for.*error|watch.*error.*restarting" | \
    wc -l || echo "0")
  
  [[ "$error_count" -eq 0 ]]
}
```

**Update test execution section** (replace lines 213-256 with):

```bash
#########################
# Execution
#########################
echo "Running CRC-based local development tests..."
echo ""

load_environment

# Infrastructure tests
run_test "CRC cluster is running" test_crc_status
run_test "OpenShift CLI authentication" test_oc_authentication  
run_test "OpenShift API accessible" test_openshift_api
run_test "Project '$PROJECT_NAME' exists" test_project_exists

# Resource tests
run_test "CRDs are applied" test_crds_applied
run_test "Service accounts exist" test_service_accounts
run_test "Namespace has managed label" test_operator_handles_managed_namespace_label

# Deployment tests
run_test "Deployments are ready" test_deployments_ready
run_test "Services exist" test_services_exist
run_test "Routes are configured" test_routes_exist

# Operator Infrastructure Tests
echo ""
log "Running Operator Infrastructure Tests..."
run_test "Operator deployment exists" test_operator_deployment_exists
run_test "Operator pod is running" test_operator_pod_running
run_test "Operator service account exists" test_operator_service_account
run_test "Operator RBAC configured" test_operator_rbac_configured

# Operator Functionality Tests
echo ""
log "Running Operator Functionality Tests..."
run_test "Operator watching AgenticSessions" test_operator_watching_sessions
run_test "Operator created workspace PVC" test_operator_workspace_pvc_created
run_test "Operator deployed content service" test_operator_content_service_deployed
run_test "Operator created ProjectSettings" test_operator_projectsettings_created
run_test "Operator logs show no critical errors" test_operator_logs_no_errors

# Operator Integration Tests (E2E)
echo ""
log "Running Operator End-to-End Tests..."
run_test "Operator creates Job from AgenticSession" test_operator_can_create_session_job
run_test "Operator updates AgenticSession status" test_operator_updates_session_status

# Health tests  
echo ""
log "Running Service Health Tests..."
run_test "Backend health endpoint" test_backend_health
run_test "Frontend is reachable" test_frontend_reachable

# API tests with authentication
run_test "Backend API with OpenShift token" test_backend_api_with_token

# Security tests
log "Skipping RBAC test - known issue with CRC permission model (admin/view permissions work correctly)"

# Optional console test (might be slow) - NOT counted in pass/fail
log "Testing OpenShift Console accessibility (optional)..."
if test_openshift_console_access 2>/dev/null; then
  success "PASS: OpenShift Console accessible"
else
  warn "OpenShift Console test failed (this is usually not critical in local dev)"
fi
```

## Testing Strategy - Test-Driven Development

### Phase 0: Write Tests FIRST (Red Phase)
**Duration: 30-45 minutes**

1. ✅ Update `crc-test.sh` with ALL operator test functions (above)
2. ✅ Run tests against current environment - EXPECT FAILURES
3. ✅ Document baseline: which tests fail and why
4. ✅ Commit failing tests to establish acceptance criteria

**Success Criteria**: 
- 13 new operator tests added to `crc-test.sh`
- All operator tests fail with clear error messages
- Test output clearly shows what's missing

### Phase 1: Implement Manifests (Green Phase - Part 1)
**Duration: 30 minutes**

1. Create `operator-build-config.yaml`
2. Create `operator-rbac.yaml`  
3. Create `operator-deployment.yaml`
4. Verify YAML syntax: `yamllint manifests/*.yaml`

**TDD Checkpoint**: Run `make dev-test` - expect infrastructure tests to pass, E2E tests still fail

### Phase 2: Update Script Integration (Green Phase - Part 2)
**Duration: 45 minutes**

1. Add `OPERATOR_DIR` variable to `crc-start.sh`
2. Add `apply_operator_rbac()` function
3. Update `build_and_deploy()` function
4. Update `wait_for_ready()` function
5. **CRITICAL**: Add namespace labeling in `ensure_project()` function:

```bash
ensure_project() {
  log "Ensuring OpenShift project '$PROJECT_NAME'..."
  
  if ! oc get project "$PROJECT_NAME" >/dev/null 2>&1; then
    oc new-project "$PROJECT_NAME" --display-name="vTeam Development"
  else
    oc project "$PROJECT_NAME"
  fi
  
  # Apply ambient-code labels for operator to recognize managed namespace
  oc label namespace "$PROJECT_NAME" ambient-code.io/managed=true --overwrite
  log "Namespace labeled as managed for operator"
}
```

6. Update execution flow to include operator steps

**TDD Checkpoint**: Run `make dev-test` - expect 8-10 operator tests to pass

### Phase 3: Verify End-to-End (Green Phase - Part 3)
**Duration: 1-2 hours**

1. Test on clean CRC environment: `make dev-clean && make dev-start`
2. Wait for all deployments to be ready
3. Run full test suite: `make dev-test`
4. Verify operator logs: `make dev-logs-operator`
5. Create manual test AgenticSession to verify Job creation
6. Check operator reconciliation of ProjectSettings

**TDD Checkpoint**: Run `make dev-test` - ALL operator tests should pass

### Phase 4: Refactor & Document
**Duration: 30 minutes**

1. Review operator logs for warnings or inefficiencies
2. Optimize resource requests/limits if needed
3. Update `README.md` with operator information
4. Add operator troubleshooting guide
5. Update Makefile with operator-specific targets:
   - `make dev-logs-operator`
   - `make dev-restart-operator`

**TDD Checkpoint**: Final run of `make dev-test` - 100% pass rate

## Test Coverage Matrix

| Category | Test Name | What It Validates | TDD Phase |
|----------|-----------|-------------------|-----------|
| **Infrastructure** | `test_operator_deployment_exists` | Deployment resource created | Phase 1 |
| **Infrastructure** | `test_operator_pod_running` | Pod is ready and healthy | Phase 2 |
| **Infrastructure** | `test_operator_service_account` | ServiceAccount exists | Phase 1 |
| **Infrastructure** | `test_operator_rbac_configured` | RBAC resources created | Phase 1 |
| **Infrastructure** | `test_operator_handles_managed_namespace_label` | Namespace properly labeled | Phase 2 |
| **Functionality** | `test_operator_watching_sessions` | Watchers initialized | Phase 2 |
| **Functionality** | `test_operator_workspace_pvc_created` | PVC auto-creation works | Phase 3 |
| **Functionality** | `test_operator_content_service_deployed` | Content service deployed | Phase 3 |
| **Functionality** | `test_operator_projectsettings_created` | ProjectSettings singleton created | Phase 3 |
| **Functionality** | `test_operator_logs_no_errors` | No critical errors in logs | Phase 2-3 |
| **E2E** | `test_operator_can_create_session_job` | Full session → job workflow | Phase 3 |
| **E2E** | `test_operator_updates_session_status` | Status reconciliation works | Phase 3 |

**Total New Tests**: 12 operator-specific tests  
**Total Assertions**: 25+ individual checks  
**Expected Pass Rate After Implementation**: 100%

## Benefits

1. **Complete Local Development**: All three core components (backend, frontend, operator) running locally
2. **Consistent Pattern**: Operator follows same build/deploy pattern as other components
3. **E2E Testing**: Can test full AgenticSession workflow locally
4. **Faster Iteration**: No need to push to external registry for operator changes
5. **Developer Experience**: Single `make dev-start` command builds everything

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Build time increases | Medium | Builds run in parallel where possible; operator is small Go binary |
| Resource constraints | Medium | Operator has minimal resource requests (50m CPU, 64Mi RAM) |
| CRD timing issues | Low | CRDs applied before operator starts |
| RBAC permission errors | Medium | Use tried-and-tested production RBAC rules |
| Image pull issues for runner | Low | Use external runner image initially; document local build option |

## Success Criteria

- ✅ `make dev-start` successfully builds and deploys operator
- ✅ Operator pod runs without errors
- ✅ Operator watches for AgenticSessions
- ✅ Operator can create Jobs for sessions
- ✅ Operator logs are accessible via `make dev-logs`
- ✅ No breaking changes to existing backend/frontend workflow


## Open Questions

1. Should we build the claude-runner locally too, or use external image?
   - **DECISION**: Use external image initially for simplicity
   
2. Do we need operator hot-reloading support like backend/frontend?
   - **DECISION**: KEEP IT SIMPLE. Hot reloading is out of scope for now. 

3. Should operator deployment be optional?
   - **DECISION**: HARD REQUIREMENT for a standard local dev instance for e2e testing.

### 6. Update Makefile - Add Operator-Specific Targets

**File**: `Makefile` (add after `dev-logs-frontend` target)

```makefile
dev-logs-operator: ## Show operator logs
	@oc logs -f deployment/vteam-operator -n vteam-dev

dev-restart-operator: ## Restart operator deployment
	@echo "Restarting operator..."
	@oc rollout restart deployment/vteam-operator -n vteam-dev
	@oc rollout status deployment/vteam-operator -n vteam-dev --timeout=60s

dev-operator-status: ## Show operator status and recent events
	@echo "Operator Deployment Status:"
	@oc get deployment vteam-operator -n vteam-dev
	@echo ""
	@echo "Operator Pod Status:"
	@oc get pods -n vteam-dev -l app=vteam-operator
	@echo ""
	@echo "Recent Operator Events:"
	@oc get events -n vteam-dev --field-selector involvedObject.kind=Deployment,involvedObject.name=vteam-operator --sort-by='.lastTimestamp' | tail -10

dev-test-operator: ## Run only operator tests
	@echo "Running operator-specific tests..."
	@bash components/scripts/local-dev/crc-test.sh 2>&1 | grep -A 1 "Operator"
```

## Pre-Implementation Checklist

Before starting implementation, ensure:

- [ ] CRC is installed and configured (`crc version`)
- [ ] Current local dev works (`make dev-start && make dev-test`)
- [ ] All existing tests pass (baseline established)
- [ ] Go toolchain available for operator build verification
- [ ] `yamllint` installed for manifest validation (`brew install yamllint` or `pip install yamllint`)
- [ ] Disk space available (operator adds ~500MB for build)
- [ ] Team consensus on TDD approach

## Implementation Workflow (TDD)

### Step 1: RED - Write Failing Tests (30 min)
```bash
# Commit current working state
git checkout -b feature/operator-local-dev
git add -A && git commit -m "Baseline: working local dev without operator"

# Add operator tests to crc-test.sh
# Edit: components/scripts/local-dev/crc-test.sh
# Copy all test functions from section 5 above

# Run tests - expect operator tests to FAIL
make dev-test

# Commit failing tests
git add components/scripts/local-dev/crc-test.sh
git commit -m "RED: Add operator tests (currently failing)"
```

### Step 2: GREEN - Implement Manifests (30 min)
```bash
# Create the three manifest files
# (Copy content from sections 1-3 above)

# Validate YAML
yamllint components/scripts/local-dev/manifests/*.yaml

# Commit manifests
git add components/scripts/local-dev/manifests/operator-*.yaml
git commit -m "GREEN: Add operator manifests"
```

### Step 3: GREEN - Update Scripts (45 min)
```bash
# Update crc-start.sh
# (Follow section 4 above)

# Update Makefile
# (Follow section 6 above)

# Test build and deploy
make dev-start

# Commit script updates
git add components/scripts/local-dev/crc-start.sh Makefile
git commit -m "GREEN: Integrate operator into dev-start workflow"
```

### Step 4: VERIFY - Run Tests (15 min)
```bash
# Run full test suite
make dev-test

# Check operator logs
make dev-logs-operator

# Verify all tests pass
# Expected: 12/12 operator tests passing
```

### Step 5: REFACTOR - Optimize & Document (30 min)
```bash
# Add operator documentation
# Update README with operator section

# Commit documentation
git add docs/ README.md
git commit -m "REFACTOR: Add operator documentation"

# Create PR
git push origin feature/operator-local-dev
```

## Expected Test Output (After Full Implementation)

```
Running CRC-based local development tests...

[09:15:23] Running test: CRC cluster is running
PASS: CRC cluster is running
[09:15:24] Running test: OpenShift CLI authentication
PASS: OpenShift CLI authentication
...

Running Operator Infrastructure Tests...
[09:16:10] Running test: Operator deployment exists
PASS: Operator deployment exists
[09:16:11] Running test: Operator pod is running
PASS: Operator pod is running
[09:16:12] Running test: Operator service account exists
PASS: Operator service account exists
[09:16:13] Running test: Operator RBAC configured
PASS: Operator RBAC configured

Running Operator Functionality Tests...
[09:16:15] Running test: Operator watching AgenticSessions
PASS: Operator watching AgenticSessions
[09:16:16] Running test: Operator created workspace PVC
PASS: Operator created workspace PVC
[09:16:17] Running test: Operator deployed content service
PASS: Operator deployed content service
[09:16:18] Running test: Operator created ProjectSettings
PASS: Operator created ProjectSettings
[09:16:19] Running test: Operator logs show no critical errors
PASS: Operator logs show no critical errors

Running Operator End-to-End Tests...
[09:16:21] Running test: Operator creates Job from AgenticSession
PASS: Operator creates Job from AgenticSession
[09:16:35] Running test: Operator updates AgenticSession status
PASS: Operator updates AgenticSession status

=========================================
Test Results: 24/24 passed
=========================================
All tests passed! vTeam local development environment is healthy.
```

## Next Steps

### Immediate (Today)
1. ✅ Review this plan with team
2. ✅ Validate TDD approach consensus
3. ✅ Run pre-implementation checklist

### Implementation (Next Session)
5. Follow TDD workflow steps 1-5
6. Create PR when all tests pass

### Follow-up (Future)
7. Add operator hot-reloading support (if needed)
8. Build claude-runner locally (optional)
9. Add operator performance metrics
10. Document common operator troubleshooting scenarios

