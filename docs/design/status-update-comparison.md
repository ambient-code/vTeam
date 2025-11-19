# Status Update Architecture: Before vs After

## ❌ Current Architecture (Problems)

```
┌──────────────────────────────────────────────────────────────┐
│ Backend API                                                  │
│  - Creates session with status.phase = "Pending"            │
│  - StopSession: Updates status to "Stopped"                 │
│  - Updates status on errors                                 │
└──────────────────────────────────────────────────────────────┘
         │ (writes status)
         │
         ▼
┌──────────────────────────────────────────────────────────────┐
│ AgenticSession CR                                            │
│  status:                                                     │
│    phase: "Running"  ◄─── WHO SET THIS?                     │
│    message: "..."    ◄─── RUNNER? OPERATOR? BACKEND?       │
│    is_error: false                                          │
└──────────────────────────────────────────────────────────────┘
         ▲ (writes status)        ▲ (writes status)
         │                        │
         │                        │
┌────────┴────────┐      ┌────────┴─────────┐
│ Operator        │      │ Runner (wrapper) │
│  - monitorJob() │      │  - line 66-72:   │
│  - Updates      │      │    "Running"     │
│    status when  │      │  - line 114-134: │
│    pod exits    │      │    "Completed"   │
│  - Race with    │      │  - line 135-148: │
│    runner!      │      │    "Failed"      │
└─────────────────┘      └──────────────────┘

PROBLEMS:
❌ Three components updating same field (race condition)
❌ No way to tell "timeout" from "ImagePullBackOff" from "user stop"
❌ Runner needs elevated CR write permissions (security risk)
❌ Stale SA tokens break runner updates
❌ No automatic retry on transient errors
❌ Sessions get "stuck" - no component takes ownership
```

## ✅ Improved Architecture (With Conditions)

```
┌──────────────────────────────────────────────────────────────┐
│ Backend API (API Gateway Only)                              │
│  - Creates session (spec only, no status)                   │
│  - User actions: DELETE job (stop), reset status (start)    │
│  - NEVER updates status during reconciliation               │
└──────────────────────────────────────────────────────────────┘
         │
         │ creates
         ▼
┌──────────────────────────────────────────────────────────────┐
│ AgenticSession CR (Single Source of Truth)                  │
│  spec:                                                       │
│    prompt: "..."                                            │
│    repos: [...]                                             │
│  status: ◄── ONLY OPERATOR WRITES THIS                      │
│    observedGeneration: 1                                    │
│    phase: "Running"      ◄── Derived from conditions        │
│    startTime: "..."                                         │
│    conditions:                                              │
│    - type: PVCReady                                         │
│      status: "True"                                         │
│      reason: Bound                                          │
│    - type: SecretsReady                                     │
│      status: "True"                                         │
│      reason: AllSecretsFound                                │
│    - type: JobCreated                                       │
│      status: "True"                                         │
│      reason: Created                                        │
│    - type: PodScheduled                                     │
│      status: "True"                                         │
│      reason: Scheduled                                      │
│    - type: RunnerStarted                                    │
│      status: "True"                                         │
│      reason: ContainerRunning                               │
│    - type: Ready                                            │
│      status: "True"                                         │
│      reason: SessionRunning                                 │
└──────────────────────────────────────────────────────────────┘
         ▲
         │ (ONLY status updater)
         │
┌────────┴─────────────────────────────────────────────────────┐
│ Operator (Reconciliation Loop)                              │
│  reconcileSession():                                        │
│   1. Check token age → refresh if > 45min                   │
│   2. Ensure PVC exists → update PVCReady condition         │
│   3. Verify secrets → update SecretsReady condition        │
│   4. Ensure Job exists → update JobCreated condition       │
│   5. Monitor pod scheduling → update PodScheduled          │
│   6. Watch runner container:                               │
│      - Running? → RunnerStarted=True, Ready=True           │
│      - Waiting? → Check reason:                            │
│        • ImagePullBackOff → Failed=True (permanent)        │
│        • CrashLoopBackOff (3x) → Failed=True               │
│        • Other → keep retrying                             │
│      - Terminated? → Check exit code:                      │
│        • 0 → Completed=True                                │
│        • 1 → Failed=True, reason=SDKError                  │
│        • 2 → Failed=True, reason=PrerequisiteFailed        │
│        • 143 → Stopped by user                             │
│   7. Check Job timeout:                                    │
│      - ActiveDeadlineSeconds exceeded?                     │
│        → Failed=True, reason=Timeout                       │
│   8. Derive phase from conditions                          │
│   9. Requeue after 5s to keep monitoring                   │
└──────────────────────────────────────────────────────────────┘
         │
         │ observes (no writes)
         ▼
┌──────────────────────────────────────────────────────────────┐
│ Kubernetes Job & Pod                                         │
│  Job:                                                        │
│    activeDeadlineSeconds: 3600  ◄── Handles timeout         │
│    backoffLimit: 3              ◄── Retry policy            │
│  Pod:                                                        │
│    phase: Running                                           │
│    containerStatuses:                                       │
│    - name: ambient-code-runner                              │
│      state:                                                 │
│        running: {...}                                       │
│        waiting: {reason: "ImagePullBackOff"}  ◄── Detected │
│        terminated: {exitCode: 0}              ◄── Success   │
└──────────────────────────────────────────────────────────────┘
         │
         │ runs
         ▼
┌──────────────────────────────────────────────────────────────┐
│ Runner (Execution Only - NO CR Status Access)               │
│  - Executes Claude Code SDK                                 │
│  - Writes progress annotation (observability):              │
│    ambient-code.io/runner-progress: "Starting..."          │
│  - Sends messages via WebSocket (UI only)                   │
│  - Exits with semantic exit codes:                          │
│    • 0   = Success                                          │
│    • 1   = SDK error                                        │
│    • 2   = Prerequisite validation failed                   │
│    • 143 = SIGTERM (user stop)                              │
│  - NO _update_cr_status() function                          │
│  - NO CR write permissions in RBAC                          │
└──────────────────────────────────────────────────────────────┘

BENEFITS:
✅ Single source of truth (operator)
✅ Clear error attribution (conditions show exactly what failed)
✅ Automatic retry on transient errors
✅ Token refresh handled automatically
✅ Timeout handled by Kubernetes Job
✅ Better security (runner has no CR write access)
✅ No race conditions
✅ Easy debugging (condition history shows timeline)
```

## Specific Problem Solutions

### Problem 1: Job Timeout

**Before:**
```
Session stuck in "Running" forever because:
- Job times out but no component detects it
- monitorJob() only checks pod exit codes
- Runner can't update status (token expired)
```

**After:**
```go
// Operator detects timeout via Job.Status.Conditions
if job.Status.Failed > 0 {
    for _, cond := range job.Status.Conditions {
        if cond.Type == batchv1.JobFailed && cond.Reason == "DeadlineExceeded" {
            r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
                "Timeout", "Job exceeded 1 hour timeout")
            // Status automatically becomes "Failed"
        }
    }
}
```

**Result:** Session status updates to:
```yaml
status:
  phase: Failed
  conditions:
  - type: Failed
    status: "True"
    reason: Timeout
    message: "Job exceeded 1 hour timeout"
    lastTransitionTime: "2025-11-15T14:30:00Z"
```

### Problem 2: Stale SA Token

**Before:**
```
1. Backend creates token (expires in 1h)
2. After 1h, token is invalid
3. Runner tries to update CR status → 401 Unauthorized
4. Session stuck in "Running" (no status update happens)
5. User confused - can't tell if session is really running
```

**After:**
```go
// Operator checks token age every reconciliation loop
func (r *SessionReconciler) ensureFreshToken(ctx context.Context, session *unstructured.Unstructured) error {
    age := time.Since(secret.CreationTimestamp.Time)
    
    if age > 45*time.Minute {
        log.Printf("Token is %v old, refreshing", age)
        // Delete old secret, mint new token
        // Runner pod automatically gets new token via secret mount
        return r.provisionRunnerToken(ctx, session)
    }
    return nil
}
```

**Result:**
- Token refreshed every 45 minutes automatically
- Runner always has valid credentials (but doesn't need to update CR)
- No stuck sessions due to auth failures

### Problem 3: ImagePullBackOff

**Before:**
```
1. Operator creates Job with bad image
2. Pod enters ImagePullBackOff
3. monitorJob() sees pod.State.Waiting but doesn't mark as Failed
4. Session stuck in "Creating" forever
5. User can't tell what's wrong from status
```

**After:**
```go
// Operator detects ImagePullBackOff as permanent error
if runnerCS.State.Waiting != nil {
    waiting := runnerCS.State.Waiting
    
    switch waiting.Reason {
    case "ImagePullBackOff", "ErrImagePull":
        // Permanent error - mark as Failed immediately
        r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
            "ImagePullBackOff", 
            fmt.Sprintf("Cannot pull image: %s", waiting.Message))
        // Delete job (no point retrying)
        r.deleteJob(ctx, session, job)
    }
}
```

**Result:** Session status updates to:
```yaml
status:
  phase: Failed
  conditions:
  - type: RunnerStarted
    status: "False"
    reason: ImagePullBackOff
    message: "Failed to pull image: quay.io/ambient_code/vteam_claude_runner:bad-tag"
  - type: Failed
    status: "True"
    reason: ImagePullBackOff
    message: "Cannot pull image: manifest unknown"
  - type: Ready
    status: "False"
    reason: SessionFailed
```

### Problem 4: Runner Crashes During Execution

**Before:**
```
1. Runner starts successfully
2. SDK crashes with Python exception
3. Runner tries to update status to "Failed" → stale token → fails
4. Container exits with code 1
5. monitorJob() sees exit code 1 → updates to "Failed" (race!)
6. Status message is generic: "Runner failed"
```

**After:**
```python
# Runner: Just exit with proper code
sys.exit(1)  # SDK error

# Operator detects exit code and sets specific condition
if term.ExitCode == 1 {
    r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
        "SDKError", 
        fmt.Sprintf("Runner exited with error: %s", term.Message))
}
```

**Result:** Clear error in status:
```yaml
status:
  phase: Failed
  conditions:
  - type: Failed
    status: "True"
    reason: SDKError
    message: "Runner exited with error: ModuleNotFoundError: No module named 'claude_agent_sdk'"
```

### Problem 5: User Stops Session

**Before:**
```
1. User clicks "Stop" in UI
2. Backend calls StopSession() → deletes Job
3. Backend updates status to "Stopped" (uses backend SA)
4. Operator's monitorJob() still running
5. monitorJob() sees pod terminated → tries to update status to "Failed"
6. Race condition: Final status is unpredictable
```

**After:**
```go
// Backend: Just delete the Job and set status once
func StopSession(c *gin.Context) {
    // Delete Job (uses user token - enforces RBAC)
    reqK8s.BatchV1().Jobs(project).Delete(...)
    
    // Update status to Stopped (backend SA - one-time write)
    DynamicClient.Resource(gvr).Namespace(project).UpdateStatus(...)
}

// Operator: Detects Stopped phase and handles cleanup
if currentPhase == "Stopped" {
    r.deleteJobIfExists(ctx, session)
    r.updateCondition(ctx, session, ConditionTypeReady, metav1.ConditionFalse, 
        "SessionStopped", "User stopped the session")
    return ctrl.Result{}, nil  // No more reconciliation
}
```

**Result:**
- No race conditions
- Clear "Stopped" status preserved
- Operator respects user action

## Migration Checklist

### Step 1: Update CRD ✅
- [ ] Add `conditions[]` field
- [ ] Add `observedGeneration` field
- [ ] Keep old fields for backward compatibility

### Step 2: Update Operator ✅
- [ ] Implement condition-based reconciliation
- [ ] Add token refresh logic
- [ ] Handle all failure scenarios (timeout, ImagePullBackOff, etc.)
- [ ] Detect exit codes and map to conditions
- [ ] Test reconciliation loop thoroughly

### Step 3: Update Runner 🔄
- [ ] Remove `_update_cr_status()` function
- [ ] Exit with proper exit codes (0, 1, 2, 143)
- [ ] Keep annotation updates for observability
- [ ] Keep WebSocket messages for UI

### Step 4: Update RBAC ✅
- [ ] Remove CR status write from runner Role
- [ ] Runner only needs annotation write (main resource patch)

### Step 5: Update Frontend 🔄
- [ ] Display conditions in session detail view
- [ ] Show condition timeline
- [ ] Map conditions to user-friendly messages

### Step 6: Testing ✅
- [ ] Happy path (session completes)
- [ ] Timeout (job exceeds deadline)
- [ ] Image pull error
- [ ] Secret missing
- [ ] Stale token (> 1h)
- [ ] Runner crash
- [ ] User stop
- [ ] Pod eviction

## Rollback Plan

If issues are discovered:

1. **Phase 1 Issues** - Revert CRD changes
2. **Phase 2 Issues** - Disable new reconciliation logic (feature flag)
3. **Phase 3 Issues** - Re-enable runner status updates temporarily
4. **Phase 4 Issues** - Rollback RBAC changes

All phases are designed to be backward compatible until Phase 3 (runner changes).

