# Session Status Redesign: Conditions-Based Architecture

## Problem Statement

The current `AgenticSession` status model is too simplistic:

```go
type AgenticSessionStatus struct {
    Phase    string `json:"phase,omitempty"`      // "Pending", "Running", "Completed", etc.
    Message  string `json:"message,omitempty"`    // Human-readable message
    IsError  bool   `json:"is_error,omitempty"`   // Error flag
}
```

**Issues:**
1. ❌ **Single phase can't represent multiple parallel concerns** (e.g., "PVC ready" + "Secrets missing" + "Job pending")
2. ❌ **No condition history** - can't tell what checks passed/failed over time
3. ❌ **Backend and Operator both update status** - race conditions and unclear ownership
4. ❌ **No observedGeneration** - can't tell if operator has processed the latest spec changes
5. ❌ **Sessions get "stuck"** - no way to distinguish between transient and permanent failures
6. ❌ **Poor debuggability** - single message string doesn't capture all context

## Kubernetes Best Practices: Conditions Pattern

### Standard K8s Status Structure

```go
type AgenticSessionStatus struct {
    // ObservedGeneration - which version of spec we've processed
    ObservedGeneration int64 `json:"observedGeneration,omitempty"`
    
    // Conditions - detailed, timestamped status checks (K8s standard)
    Conditions []metav1.Condition `json:"conditions,omitempty"`
    
    // Phase - high-level summary (derived from conditions)
    Phase string `json:"phase,omitempty"`
    
    // StartTime - when job started
    StartTime *metav1.Time `json:"startTime,omitempty"`
    
    // CompletionTime - when job finished
    CompletionTime *metav1.Time `json:"completionTime,omitempty"`
    
    // Additional fields for UI/metrics
    RunnerPodName string `json:"runnerPodName,omitempty"`
    JobName       string `json:"jobName,omitempty"`
}
```

### Conditions Types (Standard Pattern)

```go
const (
    // ConditionTypeReady - Overall readiness (rolling up all conditions)
    ConditionTypeReady = "Ready"
    
    // ConditionTypePVCReady - Workspace PVC is provisioned and bound
    ConditionTypePVCReady = "PVCReady"
    
    // ConditionTypeSecretsReady - Required secrets exist and are accessible
    ConditionTypeSecretsReady = "SecretsReady"
    
    // ConditionTypeJobCreated - Kubernetes Job has been created
    ConditionTypeJobCreated = "JobCreated"
    
    // ConditionTypePodScheduled - Pod has been scheduled to a node
    ConditionTypePodScheduled = "PodScheduled"
    
    // ConditionTypeRunnerStarted - Runner container is running
    ConditionTypeRunnerStarted = "RunnerStarted"
    
    // ConditionTypeCompleted - Session has completed successfully
    ConditionTypeCompleted = "Completed"
    
    // ConditionTypeFailed - Session has failed (permanent failure)
    ConditionTypeFailed = "Failed"
)
```

### Example Status with Conditions

**Healthy Running Session:**
```yaml
status:
  observedGeneration: 1
  phase: Running
  startTime: "2025-11-15T12:00:00Z"
  jobName: agentic-session-123456-job
  runnerPodName: agentic-session-123456-job-abc123
  conditions:
  - type: PVCReady
    status: "True"
    reason: Bound
    message: "PVC ambient-workspace-123456 is bound"
    lastTransitionTime: "2025-11-15T12:00:01Z"
  - type: SecretsReady
    status: "True"
    reason: AllSecretsFound
    message: "All required secrets are present"
    lastTransitionTime: "2025-11-15T12:00:02Z"
  - type: JobCreated
    status: "True"
    reason: Created
    message: "Job agentic-session-123456-job created"
    lastTransitionTime: "2025-11-15T12:00:03Z"
  - type: PodScheduled
    status: "True"
    reason: Scheduled
    message: "Pod scheduled on node worker-1"
    lastTransitionTime: "2025-11-15T12:00:05Z"
  - type: RunnerStarted
    status: "True"
    reason: ContainerRunning
    message: "Runner container started"
    lastTransitionTime: "2025-11-15T12:00:15Z"
  - type: Ready
    status: "True"
    reason: SessionRunning
    message: "Session is running normally"
    lastTransitionTime: "2025-11-15T12:00:15Z"
```

**Stuck Session (Secrets Missing):**
```yaml
status:
  observedGeneration: 1
  phase: Pending
  conditions:
  - type: PVCReady
    status: "True"
    reason: Bound
    message: "PVC ambient-workspace-123456 is bound"
    lastTransitionTime: "2025-11-15T12:00:01Z"
  - type: SecretsReady
    status: "False"
    reason: SecretNotFound
    message: "Secret 'ambient-runner-secrets' not found in namespace"
    lastTransitionTime: "2025-11-15T12:00:02Z"
  - type: JobCreated
    status: "False"
    reason: WaitingForSecrets
    message: "Cannot create job until secrets are ready"
    lastTransitionTime: "2025-11-15T12:00:02Z"
  - type: Ready
    status: "False"
    reason: SecretsNotReady
    message: "Waiting for required secrets"
    lastTransitionTime: "2025-11-15T12:00:02Z"
```

**Failed Session (Image Pull Error):**
```yaml
status:
  observedGeneration: 1
  phase: Failed
  startTime: "2025-11-15T12:00:00Z"
  completionTime: "2025-11-15T12:02:00Z"
  conditions:
  - type: PodScheduled
    status: "True"
    reason: Scheduled
    message: "Pod scheduled on node worker-1"
    lastTransitionTime: "2025-11-15T12:00:05Z"
  - type: RunnerStarted
    status: "False"
    reason: ImagePullBackOff
    message: "Failed to pull image: quay.io/ambient_code/vteam_claude_runner:latest"
    lastTransitionTime: "2025-11-15T12:02:00Z"
  - type: Failed
    status: "True"
    reason: ImagePullError
    message: "Cannot start runner container"
    lastTransitionTime: "2025-11-15T12:02:00Z"
  - type: Ready
    status: "False"
    reason: SessionFailed
    message: "Session failed: ImagePullBackOff"
    lastTransitionTime: "2025-11-15T12:02:00Z"
```

## Responsibility Split: Operator vs Backend

### ✅ OPERATOR Responsibilities (Source of Truth)

**The operator should be the ONLY component updating `status`**

1. **PVC Management**
   - Check PVC exists and is bound
   - Update condition: `PVCReady`

2. **Secret Verification**
   - Check all required secrets exist
   - Update condition: `SecretsReady`

3. **Job Lifecycle**
   - Create Job when all prerequisites met
   - Update condition: `JobCreated`
   - Monitor Job status
   - Delete Job on completion/failure

4. **Pod Monitoring**
   - Watch pod scheduling
   - Update condition: `PodScheduled`
   - Watch container states
   - Update condition: `RunnerStarted`
   - Detect ImagePullBackOff, CrashLoopBackOff, etc.
   - Update conditions with specific failure reasons

5. **Session Lifecycle State Machine**
   - Update `phase` based on conditions
   - Set `startTime` when runner starts
   - Set `completionTime` when finished
   - Update `observedGeneration` after processing spec changes

6. **Terminal State Handling**
   - Cleanup on Completed/Failed/Stopped
   - Set `interactive: true` for restart capability
   - Keep PVC for continuations

### ✅ BACKEND Responsibilities (API Gateway)

**The backend NEVER updates status directly** (except for explicit user actions)

1. **CRUD Operations on Spec**
   - Create sessions (validates, sets defaults)
   - Update spec fields (prompt, displayName, llmSettings)
   - Delete sessions

2. **User-Initiated State Changes**
   - **Stop**: Update status to `Stopped`, delete Job (uses user token)
   - **Start/Restart**: Reset status to `Pending` → triggers operator

3. **Token Provisioning**
   - Create ServiceAccount, Role, RoleBinding
   - Mint tokens, store in secrets

4. **API Endpoints for Runners**
   - Message streaming (WebSocket)
   - GitHub token minting on-demand
   - Content service proxy

5. **RBAC Enforcement**
   - Validate user permissions for all operations
   - Use user-scoped K8s clients

### ❌ BACKEND Should NOT:
- Monitor pods directly
- Update phase during reconciliation
- Detect container failures
- Track Job completion
- Update `observedGeneration`

### ❌ RUNNERS Should NOT:
- Update phase directly
- Update conditions
- They can only:
  - Send messages via WebSocket
  - Request tokens via backend API

## State Machine and Phase Transitions

### Valid Phases

```go
const (
    PhasePending    = "Pending"    // Waiting for operator to provision resources
    PhaseCreating   = "Creating"   // Operator creating Job/Service
    PhaseRunning    = "Running"    // Runner container actively executing
    PhaseCompleted  = "Completed"  // Successfully finished
    PhaseFailed     = "Failed"     // Permanent failure
    PhaseStopped    = "Stopped"    // User stopped the session
)
```

### Phase Determination (Operator Logic)

```go
func determinePhase(conditions []metav1.Condition) string {
    // Check terminal conditions first
    if getCondition(conditions, ConditionTypeFailed).Status == metav1.ConditionTrue {
        return PhaseFailed
    }
    if getCondition(conditions, ConditionTypeCompleted).Status == metav1.ConditionTrue {
        return PhaseCompleted
    }
    
    // Check running state
    runnerStarted := getCondition(conditions, ConditionTypeRunnerStarted)
    if runnerStarted.Status == metav1.ConditionTrue {
        return PhaseRunning
    }
    
    // Check creating state
    jobCreated := getCondition(conditions, ConditionTypeJobCreated)
    if jobCreated.Status == metav1.ConditionTrue {
        return PhaseCreating
    }
    
    // Default to Pending if nothing else matches
    return PhasePending
}
```

### Condition Update Pattern (Operator)

```go
func (r *SessionReconciler) updateCondition(
    ctx context.Context,
    session *unstructured.Unstructured,
    conditionType string,
    status metav1.ConditionStatus,
    reason string,
    message string,
) error {
    conditions, _ := getConditions(session)
    
    // Find existing condition
    found := false
    for i := range conditions {
        if conditions[i].Type == conditionType {
            // Only update if status changed
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
    
    // Add new condition if not found
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
    
    // Update status
    return r.updateStatus(ctx, session, map[string]interface{}{
        "conditions":          conditions,
        "phase":               determinePhase(conditions),
        "observedGeneration":  session.GetGeneration(),
    })
}
```

## Reconciliation Loop (Operator)

### Current Flow Problems

**Today:**
1. Backend creates CR with `status.phase = Pending`
2. Operator sees `Pending` → creates Job
3. Backend or Operator updates status → RACE CONDITION
4. monitorJob goroutine updates status → ANOTHER RACE
5. No retry on transient errors
6. No way to tell "waiting" vs "failed"

### Improved Reconciliation Loop

```go
func (r *SessionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch the AgenticSession
    session := &unstructured.Unstructured{}
    if err := r.Get(ctx, req.NamespacedName, session); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    
    // 2. Check if deletion is in progress
    if !session.GetDeletionTimestamp().IsZero() {
        return r.handleDeletion(ctx, session)
    }
    
    // 3. Get current phase from status
    currentPhase := getPhase(session)
    
    // 4. Handle terminal phases (no-op)
    if isTerminalPhase(currentPhase) {
        return ctrl.Result{}, nil
    }
    
    // 5. Handle Stopped phase (cleanup)
    if currentPhase == PhaseStopped {
        return r.handleStopped(ctx, session)
    }
    
    // 6. Main reconciliation logic
    return r.reconcileSession(ctx, session)
}

func (r *SessionReconciler) reconcileSession(ctx context.Context, session *unstructured.Unstructured) (ctrl.Result, error) {
    // Step 1: Ensure PVC exists
    pvcReady, err := r.ensurePVC(ctx, session)
    if err != nil {
        r.updateCondition(ctx, session, ConditionTypePVCReady, metav1.ConditionFalse, "ProvisionFailed", err.Error())
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil // Retry
    }
    if !pvcReady {
        r.updateCondition(ctx, session, ConditionTypePVCReady, metav1.ConditionFalse, "Provisioning", "PVC is being provisioned")
        return ctrl.Result{RequeueAfter: 5 * time.Second}, nil // Wait for PVC
    }
    r.updateCondition(ctx, session, ConditionTypePVCReady, metav1.ConditionTrue, "Bound", "PVC is bound and ready")
    
    // Step 2: Verify secrets exist
    secretsReady, missingSecret, err := r.verifySecrets(ctx, session)
    if err != nil {
        r.updateCondition(ctx, session, ConditionTypeSecretsReady, metav1.ConditionFalse, "VerificationFailed", err.Error())
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }
    if !secretsReady {
        r.updateCondition(ctx, session, ConditionTypeSecretsReady, metav1.ConditionFalse, "SecretNotFound", fmt.Sprintf("Secret '%s' not found", missingSecret))
        return ctrl.Result{RequeueAfter: 30 * time.Second}, nil // Retry less frequently for secrets
    }
    r.updateCondition(ctx, session, ConditionTypeSecretsReady, metav1.ConditionTrue, "AllSecretsFound", "All required secrets are present")
    
    // Step 3: Ensure Job exists
    jobExists, job, err := r.ensureJob(ctx, session)
    if err != nil {
        r.updateCondition(ctx, session, ConditionTypeJobCreated, metav1.ConditionFalse, "CreationFailed", err.Error())
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }
    if !jobExists {
        // Job was just created, give it time to schedule
        r.updateCondition(ctx, session, ConditionTypeJobCreated, metav1.ConditionTrue, "Created", "Job created successfully")
        return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
    }
    
    // Step 4: Monitor pod status
    pod, err := r.getPodForJob(ctx, job)
    if err != nil {
        return ctrl.Result{RequeueAfter: 5 * time.Second}, nil // Retry
    }
    
    // Check pod scheduling
    if pod.Spec.NodeName != "" {
        r.updateCondition(ctx, session, ConditionTypePodScheduled, metav1.ConditionTrue, "Scheduled", fmt.Sprintf("Pod scheduled on node %s", pod.Spec.NodeName))
    } else {
        // Check for scheduling issues
        for _, cond := range pod.Status.Conditions {
            if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
                r.updateCondition(ctx, session, ConditionTypePodScheduled, metav1.ConditionFalse, cond.Reason, cond.Message)
                return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
            }
        }
    }
    
    // Step 5: Check runner container status
    for _, cs := range pod.Status.ContainerStatuses {
        if cs.Name != "ambient-code-runner" {
            continue
        }
        
        // Container running
        if cs.State.Running != nil {
            r.updateCondition(ctx, session, ConditionTypeRunnerStarted, metav1.ConditionTrue, "ContainerRunning", "Runner container is active")
            // Update Ready condition
            r.updateCondition(ctx, session, ConditionTypeReady, metav1.ConditionTrue, "SessionRunning", "Session is running normally")
            // Record start time if not set
            if getStartTime(session) == nil {
                r.updateStatus(ctx, session, map[string]interface{}{
                    "startTime": metav1.Now(),
                })
            }
            return ctrl.Result{RequeueAfter: 5 * time.Second}, nil // Keep monitoring
        }
        
        // Container waiting (potential errors)
        if cs.State.Waiting != nil {
            waiting := cs.State.Waiting
            isPermanentError := false
            switch waiting.Reason {
            case "ImagePullBackOff", "ErrImagePull", "InvalidImageName":
                isPermanentError = true
            case "CrashLoopBackOff":
                isPermanentError = cs.RestartCount > 3 // Permanent after 3 retries
            }
            
            if isPermanentError {
                r.updateCondition(ctx, session, ConditionTypeRunnerStarted, metav1.ConditionFalse, waiting.Reason, waiting.Message)
                r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, waiting.Reason, fmt.Sprintf("Runner container failed: %s", waiting.Message))
                r.updateCondition(ctx, session, ConditionTypeReady, metav1.ConditionFalse, "SessionFailed", waiting.Message)
                r.updateStatus(ctx, session, map[string]interface{}{
                    "completionTime": metav1.Now(),
                })
                // Cleanup Job
                r.deleteJob(ctx, session, job)
                return ctrl.Result{}, nil // Terminal state, stop reconciling
            } else {
                // Transient error, keep retrying
                r.updateCondition(ctx, session, ConditionTypeRunnerStarted, metav1.ConditionFalse, waiting.Reason, waiting.Message)
                return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
            }
        }
        
        // Container terminated
        if cs.State.Terminated != nil {
            term := cs.State.Terminated
            if term.ExitCode == 0 {
                r.updateCondition(ctx, session, ConditionTypeCompleted, metav1.ConditionTrue, "Success", "Runner completed successfully")
                r.updateCondition(ctx, session, ConditionTypeReady, metav1.ConditionFalse, "SessionCompleted", "Session finished")
            } else {
                r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, "ExitCode", fmt.Sprintf("Runner exited with code %d: %s", term.ExitCode, term.Message))
                r.updateCondition(ctx, session, ConditionTypeReady, metav1.ConditionFalse, "SessionFailed", term.Message)
            }
            r.updateStatus(ctx, session, map[string]interface{}{
                "completionTime": metav1.Now(),
            })
            // Set interactive for restart
            r.setSpecField(ctx, session, "interactive", true)
            // Cleanup Job
            r.deleteJob(ctx, session, job)
            return ctrl.Result{}, nil // Terminal state
        }
    }
    
    return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}
```

## Benefits of This Approach

### ✅ Debuggability
- **Clear history**: Conditions show exactly what failed and when
- **Granular status**: Can see "PVC ready, secrets missing" instead of just "Pending"
- **Timestamps**: Know how long each stage took

### ✅ No More Stuck Sessions
- **Transient vs permanent failures**: Operator retries transient errors, fails fast on permanent ones
- **Clear error messages**: Conditions show exactly what's wrong (e.g., "ImagePullBackOff" vs "SecretNotFound")
- **Automatic retries**: RequeueAfter ensures operator keeps checking

### ✅ Better UI
- Can show detailed progress: ✅ PVC Ready → ✅ Secrets Ready → ⏳ Job Creating
- Can display condition history timeline
- Can show specific error guidance ("Secret 'X' is missing - create it in project settings")

### ✅ Separation of Concerns
- **Operator**: Infrastructure and lifecycle (PVCs, Jobs, Pods)
- **Backend**: API gateway and user operations
- **Runners**: Execution only, no status updates

### ✅ Standards Compliance
- Follows Kubernetes API conventions
- Compatible with standard K8s tooling (`kubectl describe` shows conditions)
- Easier for other developers to understand

## Migration Strategy

### Phase 1: Add Conditions (Backward Compatible)
1. Update CRD to add `conditions`, `observedGeneration`, `startTime`, `completionTime`
2. Keep existing `phase`, `message`, `is_error` fields
3. Operator writes BOTH old and new formats
4. Backend continues reading `phase`
5. **No breaking changes**

### Phase 2: Update Operator
1. Implement condition-based reconciliation loop
2. Replace `monitorJob` goroutine with proper reconciliation
3. Use controller-runtime's `Reconcile()` pattern
4. Add retry logic and exponential backoff
5. **Still backward compatible**

### Phase 3: Update Frontend
1. Display conditions in session detail view
2. Show progress indicators based on conditions
3. Add condition timeline/history view
4. **Still reads `phase` for compatibility**

### Phase 4: Remove Backend Status Updates
1. Backend stops calling `UpdateStatus` directly
2. Remove `UpdateSessionStatus` endpoint (runners shouldn't use it)
3. Backend only updates status for `Stop` action
4. **Operator is now source of truth**

### Phase 5: Deprecate Old Fields (Optional)
1. Mark `message`, `is_error` as deprecated in CRD
2. Frontend only uses conditions
3. Eventually remove old fields in v1alpha2/v1beta1

## Example CRD Update

```yaml
# components/manifests/base/crds/agenticsessions-crd.yaml
status:
  type: object
  properties:
    # New fields (Kubernetes standard)
    observedGeneration:
      type: integer
      format: int64
      description: "Generation of spec that was last processed"
    
    conditions:
      type: array
      items:
        type: object
        required: [type, status]
        properties:
          type:
            type: string
            description: "Condition type (Ready, PVCReady, JobCreated, etc.)"
          status:
            type: string
            enum: ["True", "False", "Unknown"]
          reason:
            type: string
            description: "Machine-readable reason code"
          message:
            type: string
            description: "Human-readable message"
          lastTransitionTime:
            type: string
            format: date-time
          observedGeneration:
            type: integer
            format: int64
    
    startTime:
      type: string
      format: date-time
    
    completionTime:
      type: string
      format: date-time
    
    jobName:
      type: string
    
    runnerPodName:
      type: string
    
    # Legacy fields (keep for backward compatibility)
    phase:
      type: string
      enum: [Pending, Creating, Running, Completed, Failed, Stopped]
    
    message:
      type: string
      description: "DEPRECATED: Use conditions instead"
    
    is_error:
      type: boolean
      description: "DEPRECATED: Check Failed condition instead"
```

## Next Steps

1. **Review this design** - Does this address your stuck state issues?
2. **Prioritize conditions** - Which ones are most important for your use cases?
3. **Choose migration approach** - Big bang or incremental?
4. **Implement Phase 1** - Add conditions to CRD (backward compatible)
5. **Update operator** - Implement condition-based reconciliation

Would you like me to start implementing Phase 1 (CRD update + condition helpers)?

