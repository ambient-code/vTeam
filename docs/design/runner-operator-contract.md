# Runner-Operator Contract: Status Updates

## Problem Statement

Currently, the runner (`wrapper.py`) directly updates the AgenticSession CR status, which creates:
- Race conditions with operator monitoring
- Security issues (runner needs elevated CR write permissions)
- Poor handling of timeouts, pod failures, stale tokens
- Violation of "operator as source of truth" principle

## Solution: Operator-Only Status Updates

### Architecture Pattern

```
┌─────────────────────────────────────────────────────────────┐
│                     OPERATOR (Source of Truth)              │
│  - Watches Job/Pod status                                   │
│  - Updates CR status via Conditions                         │
│  - Detects timeouts, failures, ImagePullBackOff            │
│  - Refreshes SA tokens when needed                         │
│  - Retries transient errors                                │
└─────────────────────────────────────────────────────────────┘
                             ▲
                             │ Observes
                             │
┌─────────────────────────────────────────────────────────────┐
│                    KUBERNETES JOB & POD                     │
│  - Job activeDeadlineSeconds handles timeout                │
│  - Pod containerStatuses show runner state                  │
│  - Pod conditions show scheduling issues                    │
└─────────────────────────────────────────────────────────────┘
                             ▲
                             │ Runs in
                             │
┌─────────────────────────────────────────────────────────────┐
│                    RUNNER (Execution Only)                  │
│  - Executes Claude Code SDK                                 │
│  - Writes progress to annotation (not status)               │
│  - Sends messages via WebSocket                             │
│  - Exits with proper exit code                              │
│  - NO CR status updates                                     │
└─────────────────────────────────────────────────────────────┘
```

## Runner Changes: Remove Status Updates

### What Runner Should DO

1. **Write progress to annotation** (not status):
```python
async def _report_progress(self, message: str):
    """Report progress via annotation (read by operator for observability)."""
    try:
        timestamp = self._utc_iso()
        annotation_key = "ambient-code.io/runner-progress"
        annotation_value = json.dumps({
            "message": message,
            "timestamp": timestamp
        })
        await self._update_cr_annotation(annotation_key, annotation_value)
    except Exception as e:
        logging.debug(f"Progress annotation update failed (non-critical): {e}")
```

2. **Exit with proper exit codes**:
```python
# Success
sys.exit(0)

# User-requested stop
sys.exit(143)  # SIGTERM

# SDK error
sys.exit(1)

# Prerequisite validation failed
sys.exit(2)
```

3. **Send real-time updates via WebSocket** (UI only, not CR):
```python
await self._send_log("Starting Claude Code session...")
```

### What Runner Should NOT DO

❌ **Remove these functions from wrapper.py**:
```python
# DELETE THIS - operator handles status
async def _update_cr_status(self, fields: dict, blocking: bool = False):
    # REMOVE ENTIRE FUNCTION
```

❌ **Remove all calls to `_update_cr_status()`**:
```python
# DELETE THESE
await self._update_cr_status({"phase": "Running", ...})
await self._update_cr_status({"phase": "Completed", ...})
await self._update_cr_status({"phase": "Failed", ...})
```

✅ **Keep annotation updates** (for observability):
```python
# KEEP THIS - annotations are metadata, not status
await self._update_cr_annotation("ambient-code.io/sdk-session-id", sdk_session_id)
await self._update_cr_annotation("ambient-code.io/runner-progress", progress_json)
```

## Operator Changes: Complete Reconciliation

### 1. Job Timeout Handling

**Use Kubernetes Job's built-in timeout**:

```go
// In operator when creating Job
job := &batchv1.Job{
    Spec: batchv1.JobSpec{
        // Kubernetes handles timeout - no manual monitoring needed
        ActiveDeadlineSeconds: int64Ptr(sessionTimeout),
        BackoffLimit:          int32Ptr(3),
        // ...
    },
}
```

**Detect timeout in reconciliation**:

```go
func (r *SessionReconciler) checkJobTimeout(ctx context.Context, session *unstructured.Unstructured, job *batchv1.Job) error {
    // Job exceeded ActiveDeadlineSeconds
    if job.Status.Failed > 0 {
        for _, cond := range job.Status.Conditions {
            if cond.Type == batchv1.JobFailed && cond.Reason == "DeadlineExceeded" {
                r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
                    "Timeout", 
                    fmt.Sprintf("Job exceeded timeout of %d seconds", *job.Spec.ActiveDeadlineSeconds))
                r.updateStatus(ctx, session, map[string]interface{}{
                    "completionTime": metav1.Now(),
                })
                // Cleanup
                r.deleteJob(ctx, session, job)
                return nil
            }
        }
    }
    return nil
}
```

### 2. Runner Container Exit Code Detection

**Map exit codes to conditions**:

```go
func (r *SessionReconciler) handleRunnerTermination(ctx context.Context, session *unstructured.Unstructured, cs *corev1.ContainerStatus) error {
    term := cs.State.Terminated
    
    switch term.ExitCode {
    case 0:
        // Success
        r.updateCondition(ctx, session, ConditionTypeCompleted, metav1.ConditionTrue, 
            "Success", "Runner completed successfully")
        r.updateCondition(ctx, session, ConditionTypeReady, metav1.ConditionFalse, 
            "SessionCompleted", "Session finished successfully")
        
    case 1:
        // SDK error
        r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
            "SDKError", fmt.Sprintf("Runner exited with error: %s", term.Message))
        r.updateCondition(ctx, session, ConditionTypeReady, metav1.ConditionFalse, 
            "SessionFailed", term.Message)
        
    case 2:
        // Prerequisite validation failed
        r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
            "PrerequisiteFailed", "Required prerequisite files missing")
        
    case 143:
        // SIGTERM - user requested stop (already handled by StopSession)
        log.Printf("Runner received SIGTERM (user stop)")
        
    default:
        // Unknown error
        r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
            "UnknownError", fmt.Sprintf("Runner exited with code %d: %s", term.ExitCode, term.Message))
    }
    
    // Always set completion time
    r.updateStatus(ctx, session, map[string]interface{}{
        "completionTime": metav1.Now(),
    })
    
    // Set interactive for restart
    r.setSpecField(ctx, session, "interactive", true)
    
    // Cleanup Job
    r.deleteJob(ctx, session, job)
    
    return nil
}
```

### 3. SA Token Refresh

**Monitor token age and recreate before expiration**:

```go
func (r *SessionReconciler) ensureFreshToken(ctx context.Context, session *unstructured.Unstructured) error {
    name := session.GetName()
    namespace := session.GetNamespace()
    
    // Get token secret
    secretName := fmt.Sprintf("ambient-runner-token-%s", name)
    secret, err := r.K8sClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
    if errors.IsNotFound(err) {
        // Secret missing - recreate it
        log.Printf("Token secret missing for session %s, recreating", name)
        return r.provisionRunnerToken(ctx, session)
    }
    if err != nil {
        return fmt.Errorf("failed to check token secret: %w", err)
    }
    
    // Check token age (ServiceAccount tokens expire after 1 hour by default)
    creationTime := secret.CreationTimestamp.Time
    age := time.Since(creationTime)
    
    // Refresh token if older than 45 minutes (15 min buffer)
    if age > 45*time.Minute {
        log.Printf("Token for session %s is %v old, refreshing", name, age)
        
        // Delete old secret
        err := r.K8sClient.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
        if err != nil && !errors.IsNotFound(err) {
            return fmt.Errorf("failed to delete old token: %w", err)
        }
        
        // Create fresh token
        return r.provisionRunnerToken(ctx, session)
    }
    
    return nil
}

func (r *SessionReconciler) provisionRunnerToken(ctx context.Context, session *unstructured.Unstructured) error {
    name := session.GetName()
    namespace := session.GetNamespace()
    saName := fmt.Sprintf("ambient-session-%s", name)
    
    // Mint fresh token
    tr := &authnv1.TokenRequest{
        Spec: authnv1.TokenRequestSpec{
            // Request token with 1 hour expiration
            ExpirationSeconds: int64Ptr(3600),
        },
    }
    tok, err := r.K8sClient.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, saName, tr, metav1.CreateOptions{})
    if err != nil {
        return fmt.Errorf("failed to mint token: %w", err)
    }
    
    // Store in secret
    secretName := fmt.Sprintf("ambient-runner-token-%s", name)
    secret := &corev1.Secret{
        ObjectMeta: metav1.ObjectMeta{
            Name:      secretName,
            Namespace: namespace,
            Labels:    map[string]string{"app": "ambient-runner-token"},
            OwnerReferences: []metav1.OwnerReference{{
                APIVersion: session.GetAPIVersion(),
                Kind:       session.GetKind(),
                Name:       session.GetName(),
                UID:        session.GetUID(),
                Controller: boolPtr(true),
            }},
        },
        Type: corev1.SecretTypeOpaque,
        StringData: map[string]string{
            "k8s-token": tok.Status.Token,
        },
    }
    
    _, err = r.K8sClient.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
    if errors.IsAlreadyExists(err) {
        // Update existing secret
        _, err = r.K8sClient.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
    }
    
    log.Printf("Provisioned fresh token for session %s (expires in 1h)", name)
    return err
}
```

### 4. Complete Reconciliation Loop with All Error Handling

```go
func (r *SessionReconciler) reconcileSession(ctx context.Context, session *unstructured.Unstructured) (ctrl.Result, error) {
    name := session.GetName()
    namespace := session.GetNamespace()
    
    // Step 1: Ensure token is fresh (refresh if > 45min old)
    if err := r.ensureFreshToken(ctx, session); err != nil {
        r.updateCondition(ctx, session, ConditionTypeReady, metav1.ConditionFalse, 
            "TokenRefreshFailed", fmt.Sprintf("Failed to refresh SA token: %v", err))
        return ctrl.Result{RequeueAfter: 30 * time.Second}, nil // Retry
    }
    
    // Step 2: Ensure PVC exists and is bound
    pvcReady, err := r.ensurePVC(ctx, session)
    if err != nil {
        r.updateCondition(ctx, session, ConditionTypePVCReady, metav1.ConditionFalse, 
            "ProvisionFailed", err.Error())
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }
    if !pvcReady {
        r.updateCondition(ctx, session, ConditionTypePVCReady, metav1.ConditionFalse, 
            "Provisioning", "PVC is being provisioned")
        return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
    }
    r.updateCondition(ctx, session, ConditionTypePVCReady, metav1.ConditionTrue, 
        "Bound", "PVC is bound and ready")
    
    // Step 3: Verify secrets exist
    secretsReady, missingSecret, err := r.verifySecrets(ctx, session)
    if err != nil {
        r.updateCondition(ctx, session, ConditionTypeSecretsReady, metav1.ConditionFalse, 
            "VerificationFailed", err.Error())
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }
    if !secretsReady {
        r.updateCondition(ctx, session, ConditionTypeSecretsReady, metav1.ConditionFalse, 
            "SecretNotFound", fmt.Sprintf("Secret '%s' not found", missingSecret))
        return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
    }
    r.updateCondition(ctx, session, ConditionTypeSecretsReady, metav1.ConditionTrue, 
        "AllSecretsFound", "All required secrets are present")
    
    // Step 4: Ensure Job exists
    jobName := fmt.Sprintf("%s-job", name)
    job, err := r.K8sClient.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
    if errors.IsNotFound(err) {
        // Create Job
        job, err = r.createJob(ctx, session)
        if err != nil {
            r.updateCondition(ctx, session, ConditionTypeJobCreated, metav1.ConditionFalse, 
                "CreationFailed", err.Error())
            return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
        }
        r.updateCondition(ctx, session, ConditionTypeJobCreated, metav1.ConditionTrue, 
            "Created", "Job created successfully")
        return ctrl.Result{RequeueAfter: 2 * time.Second}, nil // Let pod schedule
    }
    if err != nil {
        return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("failed to get job: %w", err)
    }
    
    // Step 5: Check for Job timeout
    if err := r.checkJobTimeout(ctx, session, job); err != nil {
        return ctrl.Result{}, err
    }
    
    // Step 6: Check Job failure (backoff limit exceeded)
    if job.Spec.BackoffLimit != nil && job.Status.Failed >= *job.Spec.BackoffLimit {
        r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
            "BackoffLimitExceeded", 
            fmt.Sprintf("Job failed after %d attempts", job.Status.Failed))
        r.updateStatus(ctx, session, map[string]interface{}{
            "completionTime": metav1.Now(),
        })
        r.deleteJob(ctx, session, job)
        return ctrl.Result{}, nil // Terminal
    }
    
    // Step 7: Monitor pod status
    pods, err := r.K8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
        LabelSelector: fmt.Sprintf("job-name=%s", jobName),
    })
    if err != nil {
        return ctrl.Result{RequeueAfter: 5 * time.Second}, err
    }
    
    if len(pods.Items) == 0 {
        // No pods yet - waiting for scheduler
        r.updateCondition(ctx, session, ConditionTypePodScheduled, metav1.ConditionFalse, 
            "PodPending", "Waiting for pod to be scheduled")
        return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
    }
    
    pod := pods.Items[0]
    
    // Check pod phase-level failures
    if pod.Status.Phase == corev1.PodFailed {
        failureMsg := fmt.Sprintf("Pod failed: %s - %s", pod.Status.Reason, pod.Status.Message)
        r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
            "PodFailed", failureMsg)
        r.updateStatus(ctx, session, map[string]interface{}{
            "completionTime": metav1.Now(),
        })
        r.deleteJob(ctx, session, job)
        return ctrl.Result{}, nil // Terminal
    }
    
    // Check pod scheduling
    if pod.Spec.NodeName != "" {
        r.updateCondition(ctx, session, ConditionTypePodScheduled, metav1.ConditionTrue, 
            "Scheduled", fmt.Sprintf("Pod scheduled on node %s", pod.Spec.NodeName))
    } else {
        // Check for scheduling issues
        for _, cond := range pod.Status.Conditions {
            if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
                r.updateCondition(ctx, session, ConditionTypePodScheduled, metav1.ConditionFalse, 
                    cond.Reason, cond.Message)
                return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
            }
        }
    }
    
    // Step 8: Check runner container status
    runnerCS := getContainerStatus(&pod, "ambient-code-runner")
    if runnerCS == nil {
        return ctrl.Result{RequeueAfter: 2 * time.Second}, nil // Container not ready
    }
    
    // Container running
    if runnerCS.State.Running != nil {
        r.updateCondition(ctx, session, ConditionTypeRunnerStarted, metav1.ConditionTrue, 
            "ContainerRunning", "Runner container is active")
        r.updateCondition(ctx, session, ConditionTypeReady, metav1.ConditionTrue, 
            "SessionRunning", "Session is running normally")
        
        // Set start time if not set
        if getStartTime(session) == nil {
            r.updateStatus(ctx, session, map[string]interface{}{
                "startTime": metav1.Now(),
            })
        }
        
        return ctrl.Result{RequeueAfter: 5 * time.Second}, nil // Keep monitoring
    }
    
    // Container waiting (check for errors)
    if runnerCS.State.Waiting != nil {
        waiting := runnerCS.State.Waiting
        isPermanentError := false
        
        switch waiting.Reason {
        case "ImagePullBackOff", "ErrImagePull", "InvalidImageName":
            isPermanentError = true
        case "CrashLoopBackOff":
            isPermanentError = runnerCS.RestartCount > 3 // Permanent after 3 retries
        case "CreateContainerConfigError":
            isPermanentError = true
        }
        
        if isPermanentError {
            r.updateCondition(ctx, session, ConditionTypeRunnerStarted, metav1.ConditionFalse, 
                waiting.Reason, waiting.Message)
            r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
                waiting.Reason, fmt.Sprintf("Runner container failed: %s", waiting.Message))
            r.updateCondition(ctx, session, ConditionTypeReady, metav1.ConditionFalse, 
                "SessionFailed", waiting.Message)
            r.updateStatus(ctx, session, map[string]interface{}{
                "completionTime": metav1.Now(),
            })
            r.deleteJob(ctx, session, job)
            return ctrl.Result{}, nil // Terminal
        } else {
            // Transient error - keep retrying
            r.updateCondition(ctx, session, ConditionTypeRunnerStarted, metav1.ConditionFalse, 
                waiting.Reason, waiting.Message)
            return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
        }
    }
    
    // Container terminated
    if runnerCS.State.Terminated != nil {
        return r.handleRunnerTermination(ctx, session, runnerCS)
    }
    
    return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}
```

## Migration Strategy

### Phase 1: Add Operator Capabilities (No Breaking Changes)
1. Update operator to handle all failure scenarios
2. Add token refresh logic
3. Add condition-based reconciliation
4. **Keep runner status updates for now** (backward compatible)

### Phase 2: Update Runner (Gradual Rollout)
1. Add exit code signaling
2. Add progress annotations
3. **Keep status updates temporarily** (log deprecation warnings)

### Phase 3: Remove Runner Status Updates
1. Remove `_update_cr_status()` from wrapper.py
2. Update runner RBAC to remove CR write permissions
3. Operator is now sole source of truth

### Phase 4: Validation
1. Test all failure scenarios (timeout, ImagePullBackOff, etc.)
2. Verify token refresh works
3. Monitor for race conditions
4. Verify UI shows correct status from conditions

## Benefits

✅ **No stuck sessions** - Operator detects and handles all failure modes
✅ **Better security** - Runner doesn't need CR write access
✅ **Token refresh** - Operator handles expiration automatically
✅ **Timeout handling** - Kubernetes Job handles it natively
✅ **Clearer debugging** - Conditions show exactly what failed
✅ **Separation of concerns** - Runner executes, operator manages lifecycle

## Testing Scenarios

1. **Happy path**: Session completes successfully → exit code 0
2. **Timeout**: Job exceeds ActiveDeadlineSeconds → Condition: Timeout
3. **Image pull error**: Bad image → Condition: ImagePullBackOff
4. **Secret missing**: Runner secrets not found → Condition: SecretsNotReady
5. **Stale token**: Token > 45min old → Auto-refreshed before job creation
6. **Pod eviction**: Node pressure → Condition: PodEvicted, Job retries
7. **Runner crash**: SDK error → exit code 1 → Condition: SDKError
8. **User stop**: DELETE job → exit code 143 → Condition: Stopped

