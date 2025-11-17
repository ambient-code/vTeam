# Operator-Centric Session Architecture: Complete Migration Plan

## Executive Summary

**Goal:** Migrate from mixed backend/operator/runner status updates to a **single source of truth** (operator) using Kubernetes Conditions pattern.

**Timeline:** 3-4 weeks (4 phases)

**Breaking Changes:** Yes - deprecated endpoints removed, runner loses CR write access

**Benefits:** No stuck sessions, automatic error detection, better observability, cleaner architecture

---

## Table of Contents

1. [Problem Analysis](#problem-analysis)
2. [Target Architecture](#target-architecture)
3. [Migration Phases](#migration-phases)
4. [Implementation Details](#implementation-details)
5. [Testing Strategy](#testing-strategy)
6. [Breaking Changes & User Impact](#breaking-changes--user-impact)

---

## Problem Analysis

### Current Issues

**1. Status Update Chaos**
- Backend updates status (StopSession, UpdateSessionStatus)
- Operator updates status (monitorJob goroutine)
- Runner updates status (wrapper.py lines 66-148)
- Race conditions: Who owns the final status?

**2. Stuck Sessions**
```yaml
status:
  phase: "Running"  # But actually...
  message: "Agent is running"
```

**Reality:**
- Job timed out 30 minutes ago (no detection)
- ImagePullBackOff for 2 hours (no auto-fail)
- SA token expired after 1 hour (runner can't update status)
- Can't tell what's actually wrong

**3. Poor Observability**
```yaml
status:
  phase: "Failed"
  message: "Something went wrong"
  is_error: true
```

No details on:
- What failed? (PVC? Secret? Image? SDK?)
- When did it fail?
- Is it transient or permanent?
- What was the timeline?

**4. Security Issues**
- Runner has CR write permissions (elevated)
- Backend uses service account for user operations (confused deputy)
- Temp pod spawning requires cluster-admin-like permissions

**5. Unclear Spec Semantics**
```yaml
spec:
  prompt: "Build a web app"  # Used once? Always? Who knows?
  repos: [...]  # Can I edit this while running?
```

---

## Target Architecture

### Responsibility Model

```
┌──────────────────────────────────────────────────┐
│ BACKEND (API Gateway)                            │
│ - Validates requests                             │
│ - Enforces RBAC                                  │
│ - Updates spec (declarative desired state)       │
│ - Proxies to content service                     │
│ - NEVER updates status during reconciliation    │
└──────────────────────────────────────────────────┘
         │
         │ Updates spec
         ▼
┌──────────────────────────────────────────────────┐
│ AGENTICSESSION CR (Source of Truth)              │
│ spec: Desired state                              │
│ status: Observed state (operator only)           │
└──────────────────────────────────────────────────┘
         │
         │ Watches
         ▼
┌──────────────────────────────────────────────────┐
│ OPERATOR (Reconciler)                            │
│ - Watches CR changes (generation increments)     │
│ - Compares spec vs status                        │
│ - Calls content service to reconcile             │
│ - Updates status with conditions                 │
│ - Handles timeouts, failures, token refresh      │
│ - ONLY component that writes status              │
└──────────────────────────────────────────────────┘
         │
         │ HTTP calls
         ▼
┌──────────────────────────────────────────────────┐
│ CONTENT SERVICE (Workspace Mutator)              │
│ - Runs in Job pod (main container)               │
│ - Provides HTTP API for workspace ops            │
│ - Clones/removes repos                           │
│ - Switches workflows                             │
│ - Restarts SDK                                   │
│ - Git operations                                 │
└──────────────────────────────────────────────────┘
         │
         │ Signals
         ▼
┌──────────────────────────────────────────────────┐
│ RUNNER (Execution Only)                          │
│ - Executes Claude Code SDK                      │
│ - NO CR status writes                           │
│ - Exits with semantic exit codes                │
│ - Sends WebSocket messages (UI only)            │
└──────────────────────────────────────────────────┘
```

### Conditions-Based Status

```yaml
status:
  # High-level summary
  phase: Running  # Derived from conditions
  observedGeneration: 5
  
  # Timestamps
  startTime: "2025-11-15T12:00:00Z"
  completionTime: null
  
  # Infrastructure tracking
  jobName: session-123-job
  runnerPodName: session-123-job-abc
  
  # Reconciliation state
  reconciledRepos:
    - url: "repo1"
      name: "repo1"
      branch: "main"
      status: Ready
      clonedAt: "..."
  
  reconciledWorkflow:
    gitUrl: "workflow-speckit"
    branch: "main"
    status: Active
    appliedAt: "..."
  
  sdkRestartCount: 2
  
  # Detailed conditions (Kubernetes standard)
  conditions:
    - type: PVCReady
      status: "True"
      reason: Bound
      message: "PVC is bound and ready"
      lastTransitionTime: "..."
      observedGeneration: 5
    
    - type: SecretsReady
      status: "True"
      reason: AllSecretsFound
      message: "All required secrets present"
      lastTransitionTime: "..."
    
    - type: JobCreated
      status: "True"
      reason: Created
      message: "Job created successfully"
      lastTransitionTime: "..."
    
    - type: PodScheduled
      status: "True"
      reason: Scheduled
      message: "Pod scheduled on node worker-1"
      lastTransitionTime: "..."
    
    - type: RunnerStarted
      status: "True"
      reason: ContainerRunning
      message: "Runner container is active"
      lastTransitionTime: "..."
    
    - type: ReposReconciled
      status: "True"
      reason: AllReposReady
      message: "All repos cloned successfully"
      lastTransitionTime: "..."
    
    - type: WorkflowReconciled
      status: "True"
      reason: WorkflowActive
      message: "Workflow is active"
      lastTransitionTime: "..."
    
    - type: Ready
      status: "True"
      reason: SessionRunning
      message: "Session running normally"
      lastTransitionTime: "..."
```

---

## Migration Phases

### Phase 1: Foundation (Week 1)

**Update CRD**
- Add conditions array
- Add observedGeneration
- Add reconciledRepos, reconciledWorkflow
- Add startTime, completionTime (re-add what was removed)
- Add sdkRestartCount
- Rename spec.prompt → spec.initialPrompt
- Remove is_error, message (replaced by conditions)

**Remove Deprecated Backend Endpoints**
- DELETE `PUT /sessions/:id/status` (only operator updates status)
- DELETE `POST /sessions/:id/spawn-content-pod`
- DELETE `GET /sessions/:id/content-pod-status`
- DELETE `DELETE /sessions/:id/content-pod`

**Add Validation**
- `PUT /sessions/:id` - Reject with 409 if phase=Running
- Document breaking changes

**No behavior changes yet** - just API cleanup

---

### Phase 2: Operator Reconciliation (Week 2)

**Implement Condition-Based Reconciliation**

Replace `handleAgenticSessionEvent()` with proper reconciliation:

```go
func (r *SessionReconciler) Reconcile(ctx, session) (ctrl.Result, error) {
    // 1. Check for deletion
    if !session.GetDeletionTimestamp().IsZero() {
        return r.handleDeletion(ctx, session)
    }
    
    // 2. Get current phase
    phase := getPhase(session)
    
    // 3. Handle terminal phases
    if phase == "Stopped" {
        return r.handleStopped(ctx, session)  // Cleanup
    }
    if phase == "Completed" || phase == "Failed" {
        return ctrl.Result{}, nil  // No-op
    }
    
    // 4. Main reconciliation
    return r.reconcileSession(ctx, session)
}

func (r *SessionReconciler) reconcileSession(ctx, session) (ctrl.Result, error) {
    // Check observedGeneration
    currentGen := session.GetGeneration()
    observedGen := getObservedGeneration(session)
    
    // Step 1: Ensure fresh token (< 45min old)
    if err := r.ensureFreshToken(ctx, session); err != nil {
        r.updateCondition(session, "Ready", False, "TokenRefreshFailed", err.Error())
        return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
    }
    
    // Step 2: Ensure PVC exists and is bound
    pvcReady, err := r.ensurePVC(ctx, session)
    if !pvcReady {
        r.updateCondition(session, "PVCReady", False, "Provisioning", "PVC provisioning")
        return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
    }
    r.updateCondition(session, "PVCReady", True, "Bound", "PVC is ready")
    
    // Step 3: Verify secrets exist
    secretsReady, missing, err := r.verifySecrets(ctx, session)
    if !secretsReady {
        r.updateCondition(session, "SecretsReady", False, "SecretNotFound", fmt.Sprintf("Secret '%s' not found", missing))
        return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
    }
    r.updateCondition(session, "SecretsReady", True, "AllSecretsFound", "All secrets present")
    
    // Step 4: Ensure Job exists
    job, err := r.ensureJob(ctx, session)
    if err != nil {
        r.updateCondition(session, "JobCreated", False, "CreationFailed", err.Error())
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }
    if job == nil {
        // Just created, give it time
        r.updateCondition(session, "JobCreated", True, "Created", "Job created")
        return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
    }
    
    // Step 5: Check for timeout
    if err := r.checkJobTimeout(ctx, session, job); err != nil {
        return ctrl.Result{}, nil  // Terminal state
    }
    
    // Step 6: Monitor pod status
    pod := r.getPodForJob(ctx, job)
    if pod == nil {
        r.updateCondition(session, "PodScheduled", False, "PodPending", "Waiting for pod")
        return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
    }
    
    // Check pod scheduling
    if pod.Spec.NodeName != "" {
        r.updateCondition(session, "PodScheduled", True, "Scheduled", fmt.Sprintf("Scheduled on %s", pod.Spec.NodeName))
    }
    
    // Step 7: Check runner container status
    runnerCS := getContainerStatus(pod, "ambient-code-runner")
    if runnerCS.State.Running != nil {
        r.updateCondition(session, "RunnerStarted", True, "ContainerRunning", "Runner active")
        r.updateCondition(session, "Ready", True, "SessionRunning", "Running normally")
        if getStartTime(session) == nil {
            r.setStartTime(session)
        }
        return ctrl.Result{RequeueAfter: 5 * time.Second}, nil  // Keep monitoring
    }
    
    if runnerCS.State.Waiting != nil {
        return r.handleContainerWaiting(ctx, session, runnerCS)
    }
    
    if runnerCS.State.Terminated != nil {
        return r.handleContainerTerminated(ctx, session, runnerCS)
    }
    
    // Step 8: Update observedGeneration
    r.updateStatus(session, map[string]interface{}{
        "observedGeneration": currentGen,
    })
    
    return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}
```

**Replace monitorJob goroutine** with reconciliation loop (no more goroutines)

**Add token refresh logic**

```go
func (r *SessionReconciler) ensureFreshToken(ctx, session) error {
    secretName := fmt.Sprintf("ambient-runner-token-%s", session.GetName())
    secret := r.K8sClient.Secrets(namespace).Get(secretName)
    
    age := time.Since(secret.CreationTimestamp.Time)
    if age > 45*time.Minute {
        log.Printf("Token is %v old, refreshing", age)
        
        // Delete old secret
        r.K8sClient.Secrets(namespace).Delete(secretName)
        
        // Mint fresh token
        return r.provisionRunnerToken(ctx, session)
    }
    return nil
}
```

**Add failure detection**

```go
func (r *SessionReconciler) handleContainerWaiting(ctx, session, cs) (ctrl.Result, error) {
    waiting := cs.State.Waiting
    
    // Detect permanent errors
    permanentErrors := map[string]bool{
        "ImagePullBackOff":           true,
        "ErrImagePull":              true,
        "InvalidImageName":          true,
        "CreateContainerConfigError": true,
    }
    
    if waiting.Reason == "CrashLoopBackOff" && cs.RestartCount > 3 {
        permanentErrors["CrashLoopBackOff"] = true
    }
    
    if permanentErrors[waiting.Reason] {
        // Permanent failure - mark as Failed
        r.updateCondition(session, "RunnerStarted", False, waiting.Reason, waiting.Message)
        r.updateCondition(session, "Failed", True, waiting.Reason, fmt.Sprintf("Container failed: %s", waiting.Message))
        r.updateCondition(session, "Ready", False, "SessionFailed", waiting.Message)
        r.setCompletionTime(session)
        r.deleteJob(ctx, session)
        return ctrl.Result{}, nil  // Terminal
    }
    
    // Transient error - keep retrying
    r.updateCondition(session, "RunnerStarted", False, waiting.Reason, waiting.Message)
    return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}
```

---

### Phase 3: Declarative Actions (Week 3)

**Migrate Add/Remove Repo to Spec Updates**

Backend:
```go
// BEFORE: Sends WebSocket
func AddRepo(c *gin.Context) {
    SendMessageToSession(project, sessionName, {
        "type": "repo_added",
        "payload": {...},
    })
}

// AFTER: Updates spec
func AddRepo(c *gin.Context) {
    session := getSession(...)
    
    // Validate running + interactive
    if phase != "Running" || !interactive {
        c.JSON(400, gin.H{"error": "Can only add repos to running interactive sessions"})
        return
    }
    
    // Add to spec.repos
    spec["repos"] = append(spec["repos"], newRepo)
    reqDyn.Update(session)  // Generation increments
    
    c.JSON(200, gin.H{"message": "Repo will be cloned by operator"})
}
```

Operator:
```go
func (r *SessionReconciler) reconcileRepos(ctx, session) error {
    desired := getReposFromSpec(session)
    reconciled := getReposFromStatus(session)
    
    // Clone missing repos
    for _, repo := range desired {
        if !contains(reconciled, repo) {
            log.Printf("Cloning missing repo: %s", repo.Name)
            // Keep temp pod pattern for now - no content service yet
            if err := r.cloneRepoViaTempPod(ctx, session, repo); err != nil {
                r.updateCondition(session, "ReposReconciled", False, "CloneFailed", err.Error())
                return err
            }
            r.addRepoToStatus(session, repo)
        }
    }
    
    // Remove extra repos
    for _, repo := range reconciled {
        if !contains(desired, repo) {
            log.Printf("Removing extra repo: %s", repo.Name)
            if err := r.removeRepoViaTempPod(ctx, session, repo); err != nil {
                continue
            }
            r.removeRepoFromStatus(session, repo)
        }
    }
    
    r.updateCondition(session, "ReposReconciled", True, "AllReposReady", fmt.Sprintf("%d repos ready", len(desired)))
    return nil
}
```

**Migrate Switch Workflow to Spec Updates**

Same pattern as repos.

**Simplify Stop Action**

Backend:
```go
// BEFORE: Deletes Job, Pods, updates status
func StopSession(c *gin.Context) {
    reqK8s.Jobs(project).Delete(jobName)
    reqK8s.Pods(project).DeleteCollection(...)
    DynamicClient.UpdateStatus(...)
}

// AFTER: Just update status
func StopSession(c *gin.Context) {
    // Validate user permission to update session
    reqDyn := GetK8sClientsForRequest(c)
    session := reqDyn.Get(...)
    
    // Update status to Stopped (using backend SA)
    DynamicClient.UpdateStatus(session, map[string]interface{}{
        "phase": "Stopped",
        "message": "User requested stop",
    })
    
    c.JSON(200, gin.H{"message": "Session will be stopped"})
}
```

Operator:
```go
// Handle Stopped phase
if phase == "Stopped" {
    r.updateCondition(session, "Ready", False, "UserStopped", "User stopped session")
    r.deleteJob(ctx, session)
    r.deletePods(ctx, session)
    r.deleteContentPod(ctx, session)
    // Keep PVC for restart
    return ctrl.Result{}, nil
}
```

---

### Phase 4: Runner Hardening (Week 4)

**Remove Status Updates from wrapper.py**

```python
# BEFORE: Direct CR status updates
async def run(self):
    await self._update_cr_status({"phase": "Running"})  # DELETE
    
    result = await self._run_claude_agent_sdk(prompt)
    
    if result.get("success"):
        await self._update_cr_status({"phase": "Completed", ...}, blocking=True)  # DELETE
    else:
        await self._update_cr_status({"phase": "Failed", ...}, blocking=True)  # DELETE

# AFTER: Just exit with proper codes
async def run(self):
    try:
        result = await self._run_claude_agent_sdk(prompt)
        
        if result.get("success"):
            logging.info("Session completed successfully")
            sys.exit(0)  # Operator detects and updates status
        else:
            logging.error(f"Session failed: {result.get('error')}")
            sys.exit(1)  # Operator detects and updates status
    except Exception as e:
        logging.error(f"Fatal error: {e}")
        sys.exit(1)
```

**Runner reports progress via annotations** (for observability):

```python
# Keep annotation updates for progress tracking
async def _report_progress(self, message: str):
    """Report progress via annotation (operator reads for observability)."""
    try:
        await self._update_cr_annotation("ambient-code.io/runner-progress", json.dumps({
            "message": message,
            "timestamp": self._utc_iso(),
        }))
    except Exception:
        pass  # Non-critical, ignore failures
```

**Operator maps exit codes to conditions:**

```go
func (r *SessionReconciler) handleContainerTerminated(ctx, session, cs) (ctrl.Result, error) {
    term := cs.State.Terminated
    
    switch term.ExitCode {
    case 0:
        // Success
        r.updateCondition(session, "Completed", True, "Success", "Runner completed")
        r.updateCondition(session, "Ready", False, "SessionCompleted", "Session finished")
        
    case 1:
        // SDK error
        r.updateCondition(session, "Failed", True, "SDKError", fmt.Sprintf("Runner error: %s", term.Message))
        r.updateCondition(session, "Ready", False, "SessionFailed", term.Message)
        
    case 2:
        // Prerequisite validation failed
        r.updateCondition(session, "Failed", True, "PrerequisiteFailed", "Required files missing")
        r.updateCondition(session, "Ready", False, "ValidationFailed", term.Message)
        
    case 143:
        // SIGTERM - user stop (already handled by Stopped phase)
        log.Printf("Runner received SIGTERM")
    }
    
    r.setCompletionTime(session)
    r.setSpecField(session, "interactive", true)  // Allow restart
    r.deleteJob(ctx, session)
    
    return ctrl.Result{}, nil
}
```

**Update Runner RBAC** - Remove status write:

```yaml
# BEFORE
rules:
  - apiGroups: ["vteam.ambient-code"]
    resources: ["agenticsessions/status"]
    verbs: ["get", "update", "patch"]

# AFTER
rules:
  - apiGroups: ["vteam.ambient-code"]
    resources: ["agenticsessions"]
    verbs: ["get", "patch"]  # Only for annotations
```

---

## Implementation Details

### 1. Updated CRD Schema

```yaml
# components/manifests/base/crds/agenticsessions-crd.yaml
spec:
  properties:
    initialPrompt:
      type: string
      description: "Initial prompt - used only on first SDK invocation for brand new sessions"
    
    repos:
      type: array
      items:
        type: object
        required: [url]
        properties:
          url:
            type: string
          branch:
            type: string
            default: main
          name:
            type: string
    
    activeWorkflow:
      type: object
      properties:
        gitUrl:
          type: string
        branch:
          type: string
          default: main
        path:
          type: string

status:
  properties:
    # Reconciliation tracking
    observedGeneration:
      type: integer
      format: int64
    
    # High-level summary
    phase:
      type: string
      enum: [Pending, Creating, Running, Completed, Failed, Stopped]
    
    # Timestamps
    startTime:
      type: string
      format: date-time
    
    completionTime:
      type: string
      format: date-time
    
    # Infrastructure references
    jobName:
      type: string
    
    runnerPodName:
      type: string
    
    # Reconciliation state
    reconciledRepos:
      type: array
      items:
        type: object
        properties:
          url:
            type: string
          branch:
            type: string
          name:
            type: string
          status:
            type: string
            enum: [Cloning, Ready, Failed]
          clonedAt:
            type: string
            format: date-time
    
    reconciledWorkflow:
      type: object
      properties:
        gitUrl:
          type: string
        branch:
          type: string
        status:
          type: string
          enum: [Cloning, Active, Failed]
        appliedAt:
          type: string
          format: date-time
    
    sdkSessionId:
      type: string
      description: "SDK's internal session UUID for resumption"
    
    sdkRestartCount:
      type: integer
      description: "How many times SDK was restarted during this session"
    
    # Kubernetes standard conditions
    conditions:
      type: array
      items:
        type: object
        required: [type, status]
        properties:
          type:
            type: string
          status:
            type: string
            enum: ["True", "False", "Unknown"]
          reason:
            type: string
          message:
            type: string
          lastTransitionTime:
            type: string
            format: date-time
          observedGeneration:
            type: integer
            format: int64
```

### 2. Condition Types

```go
const (
    ConditionTypeReady             = "Ready"
    ConditionTypePVCReady          = "PVCReady"
    ConditionTypeSecretsReady      = "SecretsReady"
    ConditionTypeJobCreated        = "JobCreated"
    ConditionTypePodScheduled      = "PodScheduled"
    ConditionTypeRunnerStarted     = "RunnerStarted"
    ConditionTypeReposReconciled   = "ReposReconciled"
    ConditionTypeWorkflowReconciled = "WorkflowReconciled"
    ConditionTypeCompleted         = "Completed"
    ConditionTypeFailed            = "Failed"
)
```

### 3. Helper Functions

```go
func (r *SessionReconciler) updateCondition(
    ctx context.Context,
    session *unstructured.Unstructured,
    conditionType string,
    status metav1.ConditionStatus,
    reason string,
    message string,
) error {
    conditions := getConditions(session)
    
    // Find existing condition
    found := false
    for i := range conditions {
        if conditions[i].Type == conditionType {
            if conditions[i].Status != status {
                conditions[i].Status = status
                conditions[i].Reason = reason
                conditions[i].Message = message
                conditions[i].LastTransitionTime = metav1.Now()
            }
            found = true
            break
        }
    }
    
    if !found {
        conditions = append(conditions, metav1.Condition{
            Type:               conditionType,
            Status:             status,
            Reason:             reason,
            Message:            message,
            LastTransitionTime: metav1.Now(),
            ObservedGeneration: session.GetGeneration(),
        })
    }
    
    return r.updateStatusFields(ctx, session, map[string]interface{}{
        "conditions": conditions,
        "phase":      r.derivePhase(conditions),
    })
}

func (r *SessionReconciler) derivePhase(conditions []metav1.Condition) string {
    // Check terminal conditions first
    if getConditionStatus(conditions, ConditionTypeFailed) == metav1.ConditionTrue {
        return "Failed"
    }
    if getConditionStatus(conditions, ConditionTypeCompleted) == metav1.ConditionTrue {
        return "Completed"
    }
    
    // Check running
    if getConditionStatus(conditions, ConditionTypeRunnerStarted) == metav1.ConditionTrue {
        return "Running"
    }
    
    // Check creating
    if getConditionStatus(conditions, ConditionTypeJobCreated) == metav1.ConditionTrue {
        return "Creating"
    }
    
    return "Pending"
}
```

### 4. Backend Changes

**File:** `components/backend/handlers/sessions.go`

Remove functions:
- `UpdateSessionStatus()` - Delete entirely
- `SpawnContentPod()` - Delete entirely
- `GetContentPodStatus()` - Delete entirely
- `DeleteContentPod()` - Delete entirely

Update functions:
```go
// Add validation
func UpdateSession(c *gin.Context) {
    session := getSession(...)
    phase := getPhase(session)
    
    // NEW: Reject if running
    if phase == "Running" || phase == "Creating" {
        c.JSON(409, gin.H{
            "error": "Cannot modify spec while session is running",
            "suggestion": "Stop the session first or create a new session",
        })
        return
    }
    
    // Update spec (only if stopped)
    spec["initialPrompt"] = req.Prompt  // Renamed from "prompt"
    // ...
}

// Simplify to spec update
func AddRepo(c *gin.Context) {
    session := getSession(...)
    
    if phase != "Running" {
        c.JSON(400, gin.H{"error": "Can only add repos to running sessions"})
        return
    }
    
    spec["repos"] = append(spec["repos"], newRepo)
    reqDyn.Update(session)  // Operator reconciles
    
    c.JSON(200, gin.H{"message": "Repo will be added"})
}

// Simplify to status update
func StopSession(c *gin.Context) {
    // Just update status, operator handles cleanup
    DynamicClient.UpdateStatus(session, map[string]interface{}{
        "phase": "Stopped",
        "message": "User requested stop",
    })
    
    c.JSON(200, gin.H{"message": "Session will be stopped"})
}
```

**File:** `components/backend/routes.go`

Remove routes:
```go
// DELETE THESE
projectGroup.PUT("/agentic-sessions/:sessionName/status", handlers.UpdateSessionStatus)
projectGroup.POST("/agentic-sessions/:sessionName/spawn-content-pod", handlers.SpawnContentPod)
projectGroup.GET("/agentic-sessions/:sessionName/content-pod-status", handlers.GetContentPodStatus)
projectGroup.DELETE("/agentic-sessions/:sessionName/content-pod", handlers.DeleteContentPod)
```

### 5. Operator Changes

**File:** `components/operator/internal/handlers/sessions.go`

Replace entire file with new reconciliation pattern:

Key changes:
- Delete `handleAgenticSessionEvent()` function
- Delete `monitorJob()` goroutine
- Add `Reconcile()` with proper controller-runtime pattern
- Add `reconcileRepos()` for spec.repos reconciliation
- Add `reconcileWorkflow()` for spec.activeWorkflow reconciliation
- Add `ensureFreshToken()` for token refresh
- Add condition management helpers
- Add failure detection logic

### 6. Runner Changes

**File:** `components/runners/claude-code-runner/wrapper.py`

Remove status updates:
```python
# DELETE these function calls (lines 66-72, 114-148, 842-850)
await self._update_cr_status({"phase": "Running", ...})
await self._update_cr_status({"phase": "Completed", ...})
await self._update_cr_status({"phase": "Failed", ...})

# DELETE the entire function
async def _update_cr_status(self, fields: dict, blocking: bool = False):
    # DELETE ENTIRE FUNCTION (lines 1385-1418)
```

Add exit codes:
```python
async def run(self):
    try:
        result = await self._run_claude_agent_sdk(prompt)
        
        if result.get("success"):
            logging.info("Session completed successfully")
            sys.exit(0)  # NEW: Operator maps to Completed
        else:
            logging.error(f"Session failed: {result.get('error')}")
            sys.exit(1)  # NEW: Operator maps to Failed (SDKError)
            
    except Exception as e:
        logging.error(f"Fatal error: {e}")
        sys.exit(1)

# In _validate_prerequisites()
if not found:
    logging.error(error_message)
    sys.exit(2)  # NEW: Operator maps to Failed (PrerequisiteFailed)
```

Keep annotation updates:
```python
# KEEP THIS - annotations for observability
async def _update_cr_annotation(self, key: str, value: str):
    # Keep this function
    
# Used for:
await self._update_cr_annotation("ambient-code.io/sdk-session-id", sdk_session_id)
await self._update_cr_annotation("ambient-code.io/runner-progress", progress_json)
```

Change environment variable:
```python
# Read renamed field
initial_prompt = self.context.get_env("INITIAL_PROMPT", "")  # Was "PROMPT"
```

### 7. Type Updates

**File:** `components/backend/types/session.go`

```go
type AgenticSessionSpec struct {
    InitialPrompt        string             `json:"initialPrompt,omitempty"`  // RENAMED
    Interactive          bool               `json:"interactive,omitempty"`
    DisplayName          string             `json:"displayName"`
    LLMSettings          LLMSettings        `json:"llmSettings"`
    Timeout              int                `json:"timeout"`
    UserContext          *UserContext       `json:"userContext,omitempty"`
    EnvironmentVariables map[string]string  `json:"environmentVariables,omitempty"`
    Project              string             `json:"project,omitempty"`
    Repos                []SimpleRepo       `json:"repos,omitempty"`
    ActiveWorkflow       *WorkflowSelection `json:"activeWorkflow,omitempty"`
}

type AgenticSessionStatus struct {
    // Reconciliation
    ObservedGeneration int64 `json:"observedGeneration,omitempty"`
    
    // Summary
    Phase string `json:"phase,omitempty"`
    
    // Timestamps
    StartTime      *string `json:"startTime,omitempty"`
    CompletionTime *string `json:"completionTime,omitempty"`
    
    // Infrastructure
    JobName       string `json:"jobName,omitempty"`
    RunnerPodName string `json:"runnerPodName,omitempty"`
    
    // Reconciliation state
    ReconciledRepos     []ReconciledRepo     `json:"reconciledRepos,omitempty"`
    ReconciledWorkflow  *ReconciledWorkflow  `json:"reconciledWorkflow,omitempty"`
    SDKSessionID        string               `json:"sdkSessionId,omitempty"`
    SDKRestartCount     int                  `json:"sdkRestartCount,omitempty"`
    
    // Conditions
    Conditions []Condition `json:"conditions,omitempty"`
}

type ReconciledRepo struct {
    URL      string  `json:"url"`
    Branch   string  `json:"branch"`
    Name     string  `json:"name"`
    Status   string  `json:"status"` // Cloning, Ready, Failed
    ClonedAt *string `json:"clonedAt,omitempty"`
}

type ReconciledWorkflow struct {
    GitURL    string  `json:"gitUrl"`
    Branch    string  `json:"branch"`
    Status    string  `json:"status"` // Cloning, Active, Failed
    AppliedAt *string `json:"appliedAt,omitempty"`
}

type Condition struct {
    Type               string  `json:"type"`
    Status             string  `json:"status"` // True, False, Unknown
    Reason             string  `json:"reason"`
    Message            string  `json:"message"`
    LastTransitionTime string  `json:"lastTransitionTime"`
    ObservedGeneration int64   `json:"observedGeneration,omitempty"`
}
```

### 8. Frontend Updates

**File:** `components/frontend/src/types/agentic-session.ts`

```typescript
export type AgenticSessionSpec = {
    initialPrompt?: string;  // RENAMED from 'prompt'
    llmSettings: LLMSettings;
    timeout: number;
    displayName?: string;
    project?: string;
    interactive?: boolean;
    repos?: SessionRepo[];
    activeWorkflow?: {
        gitUrl: string;
        branch: string;
        path?: string;
    };
};

export type ReconciledRepo = {
    url: string;
    branch: string;
    name: string;
    status: "Cloning" | "Ready" | "Failed";
    clonedAt?: string;
};

export type ReconciledWorkflow = {
    gitUrl: string;
    branch: string;
    status: "Cloning" | "Active" | "Failed";
    appliedAt?: string;
};

export type Condition = {
    type: string;
    status: "True" | "False" | "Unknown";
    reason: string;
    message: string;
    lastTransitionTime: string;
    observedGeneration?: number;
};

export type AgenticSessionStatus = {
    observedGeneration?: number;
    phase: AgenticSessionPhase;
    startTime?: string;
    completionTime?: string;
    jobName?: string;
    runnerPodName?: string;
    reconciledRepos?: ReconciledRepo[];
    reconciledWorkflow?: ReconciledWorkflow;
    sdkSessionId?: string;
    sdkRestartCount?: number;
    conditions?: Condition[];
};
```

Update UI components:
- Display conditions in session detail view
- Show condition timeline
- Lock spec fields when phase=Running
- Show reconciliation status for repos/workflows

---

## Testing Strategy

### Unit Tests

**Operator:**
```go
func TestReconcileRepos(t *testing.T) {
    tests := []struct {
        name            string
        specRepos       []Repo
        statusRepos     []ReconciledRepo
        expectedClone   []string
        expectedRemove  []string
    }{
        {
            name: "add new repo",
            specRepos: []Repo{{URL: "repo1"}, {URL: "repo2"}},
            statusRepos: []ReconciledRepo{{URL: "repo1"}},
            expectedClone: []string{"repo2"},
            expectedRemove: []string{},
        },
        // ... more test cases
    }
}
```

**Backend:**
```go
func TestUpdateSessionRejectsRunning(t *testing.T) {
    // Create running session
    // Attempt to update spec
    // Expect 409 Conflict
}
```

### Integration Tests

1. **Happy path**: Create → Run → Complete
2. **Timeout**: Job exceeds activeDeadlineSeconds
3. **ImagePullBackOff**: Bad image in spec
4. **Secret missing**: Required secret not found
5. **Token expiration**: Wait 46 minutes, verify auto-refresh
6. **Add repo (running)**: Update spec → operator clones
7. **Remove repo (running)**: Update spec → operator removes
8. **Switch workflow**: Update spec → operator switches
9. **Edit spec (running)**: Expect 409 error
10. **Stop session**: Verify operator cleans up
11. **Restart session**: Verify PVC reused

### E2E Tests

Update Cypress tests:
```typescript
it('should reject spec updates on running session', () => {
  // Create and start session
  // Attempt to update initialPrompt
  // Expect 409 error
  // Stop session
  // Update initialPrompt should succeed
})

it('should add repo dynamically', () => {
  // Create running session
  // Add repo via API
  // Wait for status.reconciledRepos to include new repo
  // Verify condition: ReposReconciled=True
})
```

---

## Breaking Changes & User Impact

### API Changes

**Removed Endpoints (4):**
```
❌ PUT /api/projects/:project/agentic-sessions/:session/status
   Impact: Runners and scripts that update status will fail
   Migration: Remove any direct status updates

❌ POST /api/projects/:project/agentic-sessions/:session/spawn-content-pod
❌ GET /api/projects/:project/agentic-sessions/:session/content-pod-status
❌ DELETE /api/projects/:project/agentic-sessions/:session/content-pod
   Impact: Temp pod management no longer exposed
   Migration: Use workspace endpoints directly (automatic)
```

**Modified Endpoints (4):**
```
⚠️ PUT /api/projects/:project/agentic-sessions/:session
   Before: Allows updates anytime
   After: Returns 409 if phase=Running
   Impact: Users must stop session before editing spec
   Migration: UI shows locked fields, user must stop first

⚠️ POST /api/projects/:project/agentic-sessions/:session/repos
   Before: Sends WebSocket to runner
   After: Updates spec.repos, operator reconciles
   Impact: Slightly slower (2s vs instant), but more reliable
   Migration: None (API contract unchanged)

⚠️ DELETE /api/projects/:project/agentic-sessions/:session/repos/:name
   Before: Sends WebSocket to runner
   After: Updates spec.repos, operator reconciles
   Migration: None (API contract unchanged)

⚠️ POST /api/projects/:project/agentic-sessions/:session/workflow
   Before: Sends WebSocket to runner
   After: Updates spec.activeWorkflow, operator reconciles
   Migration: None (API contract unchanged)
```

**Field Renames (1):**
```
⚠️ spec.prompt → spec.initialPrompt
   Impact: Old sessions won't have initialPrompt set
   Migration: Add migration in backend to copy prompt → initialPrompt on read
```

### Status Structure Changes

**Removed Fields:**
```yaml
# Old
status:
  message: "..."     # ❌ Removed (use conditions instead)
  is_error: false    # ❌ Removed (check Failed condition)

# New
status:
  conditions:
    - type: Failed
      status: "True"
      message: "Detailed error message"
```

**Added Fields:**
```yaml
status:
  observedGeneration: 5     # NEW
  startTime: "..."          # RE-ADDED (was removed in simplification)
  completionTime: "..."     # RE-ADDED
  jobName: "..."            # NEW (useful for debugging)
  runnerPodName: "..."      # NEW
  reconciledRepos: [...]    # NEW
  reconciledWorkflow: {...} # NEW
  sdkSessionId: "..."       # NEW
  sdkRestartCount: 2        # NEW
  conditions: [...]         # NEW
```

### UI Impact

**Before:**
```typescript
// All fields editable anytime
<Input value={spec.prompt} onChange={...} />
```

**After:**
```typescript
// Fields locked when running
<Input 
  value={spec.initialPrompt}
  disabled={session.status?.phase === 'Running'}
  onChange={...} 
/>

{session.status?.phase === 'Running' && (
  <Alert>
    Cannot edit while running. Stop session first or create a new session.
  </Alert>
)}
```

**New UI Features:**
- Condition timeline view
- Reconciliation status for repos/workflows
- Better error messages from conditions

### kubectl Users Impact

**Before:**
```bash
kubectl get agenticsession session-123 -o yaml
# Output:
status:
  phase: "Running"
  message: "Agent is running"
```

**After:**
```bash
kubectl get agenticsession session-123 -o yaml
# Output:
status:
  phase: Running
  observedGeneration: 5
  conditions:
    - type: Ready
      status: "True"
      reason: SessionRunning
      message: "Session running normally"
    - type: PVCReady
      status: "True"
      reason: Bound
    # ... more conditions
  reconciledRepos:
    - url: "repo1"
      status: Ready
```

Users get **much more detail** about what's actually happening!

### Migration Notes for Users

**Document this in release notes:**

```markdown
## Breaking Changes in v1.X.0

### Removed API Endpoints

The following endpoints have been removed:
- `PUT /api/projects/:project/agentic-sessions/:session/status`
- `POST /api/projects/:project/agentic-sessions/:session/spawn-content-pod`
- `GET /api/projects/:project/agentic-sessions/:session/content-pod-status`
- `DELETE /api/projects/:project/agentic-sessions/:session/content-pod`

If you have scripts that call these endpoints, please remove them.

### Spec Updates Rejected for Running Sessions

You can no longer edit session spec while the session is running.

**Before:** `PUT /sessions/:id` with new prompt → spec updated

**After:** `PUT /sessions/:id` with new prompt → 409 Conflict

**Migration:** Stop the session first, then update spec, or create a new session.

### Field Renames

- `spec.prompt` → `spec.initialPrompt`

Old sessions will continue to work (automatic migration).

### Enhanced Status

Sessions now report detailed status via Conditions:

```bash
kubectl describe agenticsession session-123
```

Shows detailed timeline of what happened.

### Improved Error Detection

Sessions no longer get stuck! The operator automatically:
- Detects timeouts
- Detects image pull failures
- Detects pod evictions
- Refreshes expired tokens
- Marks sessions as Failed with specific reasons
```

---

## Rollback Plan

### If Critical Issues Found

**Phase 1 Rollback:**
```bash
# Revert CRD
kubectl apply -f components/manifests/base/crds/agenticsessions-crd-old.yaml

# Redeploy old backend (has removed endpoints)
# Need to restore old code
```

**Phase 2 Rollback:**
```bash
# Deploy old operator
kubectl apply -f components/manifests/base/operator-deployment-old.yaml

# Old reconciliation logic active
```

**Phase 3 Rollback:**
```bash
# Revert backend spec update logic
# Restore WebSocket-based repo/workflow changes
```

**Phase 4 Rollback:**
```bash
# Restore runner status updates
# Grant CR write permissions back to runner
```

### Monitoring During Migration

Watch for:
- Sessions stuck in Pending (operator not reconciling)
- 409 errors from UI (users trying to edit running sessions)
- Condition updates not happening (operator not watching)
- Token refresh failures (SA creation issues)

---

## Success Criteria

After migration is complete:

✅ **No stuck sessions** - All failure modes auto-detected within 30 seconds  
✅ **Token refresh works** - Sessions can run > 1 hour without auth failures  
✅ **Clear error messages** - Conditions show exactly what failed  
✅ **Spec is declarative** - Users declare desired state, operator reconciles  
✅ **Audit trail** - Condition history shows complete timeline  
✅ **Better security** - Runner has no CR write access  
✅ **Cleaner code** - Backend is simpler (3-4 removed functions)  
✅ **Operator owns lifecycle** - Single source of truth  

---

## Implementation Checklist

### Phase 1: Foundation (Week 1)
- [ ] Update CRD schema (add conditions, observedGeneration, etc.)
- [ ] Apply CRD to cluster
- [ ] Update backend types (AgenticSessionStatus struct)
- [ ] Remove UpdateSessionStatus endpoint
- [ ] Remove temp pod endpoints (3 total)
- [ ] Add validation to UpdateSession (reject if Running)
- [ ] Update routes.go (remove 4 routes)
- [ ] Test: Create/get/delete sessions still work
- [ ] Test: Update session when stopped works
- [ ] Test: Update session when running returns 409

### Phase 2: Operator Reconciliation (Week 2)
- [ ] Create operator helpers (updateCondition, derivePhase, etc.)
- [ ] Implement Reconcile() function
- [ ] Implement reconcileSession() with all steps
- [ ] Implement ensureFreshToken() with 45min refresh
- [ ] Implement failure detection (ImagePullBackOff, timeout, etc.)
- [ ] Replace handleAgenticSessionEvent() with Reconcile()
- [ ] Remove monitorJob() goroutine
- [ ] Test: Timeout detection works
- [ ] Test: ImagePullBackOff auto-fails
- [ ] Test: Token refresh at 46 minutes
- [ ] Test: Conditions update correctly

### Phase 3: Declarative Actions (Week 3)
- [ ] Implement reconcileRepos() in operator
- [ ] Implement reconcileWorkflow() in operator
- [ ] Implement cloneRepoViaTempPod() helper
- [ ] Implement removeRepoViaTempPod() helper
- [ ] Update AddRepo() to update spec instead of WebSocket
- [ ] Update RemoveRepo() to update spec instead of WebSocket
- [ ] Update SelectWorkflow() to update spec instead of WebSocket
- [ ] Update StopSession() to just update status
- [ ] Test: Add repo → spec updated → operator clones
- [ ] Test: Remove repo → spec updated → operator removes
- [ ] Test: Switch workflow → spec updated → operator switches
- [ ] Test: Stop session → operator cleans up

### Phase 4: Runner Hardening (Week 4)
- [ ] Remove _update_cr_status() function from wrapper.py
- [ ] Remove all calls to _update_cr_status()
- [ ] Update run() to exit with proper codes (0, 1, 2)
- [ ] Update operator to map exit codes to conditions
- [ ] Update runner Role (remove status write permission)
- [ ] Update operator Job creation (set INITIAL_PROMPT env var)
- [ ] Update parseSpec() to handle initialPrompt
- [ ] Test: Runner exit 0 → Completed
- [ ] Test: Runner exit 1 → Failed (SDKError)
- [ ] Test: Runner exit 2 → Failed (PrerequisiteFailed)
- [ ] Test: Runner has no CR write access

### Phase 5: Documentation & Polish (Week 5)
- [ ] Update user documentation
- [ ] Create migration guide
- [ ] Document breaking changes
- [ ] Update API reference
- [ ] Add condition reference documentation
- [ ] Create troubleshooting guide using conditions
- [ ] Performance testing (reconciliation frequency tuning)
- [ ] Update frontend to show conditions
- [ ] Add condition timeline view
- [ ] Release notes

---

## File Changes Summary

**Files to Modify (11):**
1. `components/manifests/base/crds/agenticsessions-crd.yaml` - Update schema
2. `components/backend/types/session.go` - Update types
3. `components/backend/handlers/sessions.go` - Remove 4 functions, update 4 functions
4. `components/backend/routes.go` - Remove 4 routes
5. `components/operator/internal/handlers/sessions.go` - Complete rewrite
6. `components/operator/internal/handlers/helpers.go` - Add condition helpers (new file)
7. `components/runners/claude-code-runner/wrapper.py` - Remove status updates
8. `components/backend/handlers/helpers.go` - Update parseSpec for initialPrompt
9. `components/frontend/src/types/agentic-session.ts` - Update types
10. `components/frontend/src/components/session-detail.tsx` - Show conditions
11. `docs/api/sessions.md` - Update API documentation

**Estimated Changes:**
- Add: ~800 lines (operator reconciliation, condition helpers)
- Remove: ~400 lines (backend functions, runner status updates, monitorJob)
- Modify: ~200 lines (validation, type renames)
- Net: +200 lines

This consolidates everything into one actionable plan. Ready to proceed with implementation?
