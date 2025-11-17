# Spec vs Runtime State Synchronization

## Problem Statement

When a session is running, users can make dynamic changes:
- Switch active workflow
- Add/remove repositories
- Update the prompt
- Change LLM settings

**Questions:**
1. Should these changes update the CR spec?
2. Who applies these changes - backend or operator?
3. Should updating spec restart the session?
4. How do we keep CR in sync with what actually happened in the runner?

## Kubernetes Principles

### Spec vs Status vs Annotations

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: session-123
  annotations:
    # Runtime state - transient, doesn't affect desired state
    ambient-code.io/sdk-session-id: "abc-def-123"
    ambient-code.io/runner-progress: '{"message": "Processing..."}'
    ambient-code.io/repos-added-at-runtime: '["new-repo"]'
    
spec:
  # Desired state - what we WANT the session to do
  prompt: "Initial prompt"
  repos: [...]
  activeWorkflow:
    gitUrl: "..."
  # Should spec changes restart the session? YES (in most cases)

status:
  # Observed state - what we SEE happening
  phase: Running
  conditions: [...]
  # Derived from actual pod/job state by operator
```

### Key Principle: Spec is Immutable After Creation (Mostly)

**Rationale:**
- CR spec represents the **initial configuration** for a session
- Changing spec during execution creates ambiguity: "Is this a new session or modified old one?"
- Kubernetes pattern: spec changes → reconciliation → pod restart

**Exception:** Interactive sessions can have **runtime state** that differs from spec

## Architecture: Three Types of Changes

### Type 1: Initial Configuration (Spec) - Immutable

**Examples:**
- Initial prompt
- Initial repos
- LLM settings (model, temperature)
- Timeout
- Initial workflow

**Rules:**
- ✅ Can be edited **before** session starts (phase=Pending)
- ❌ Cannot be edited while running (operator ignores changes)
- ✅ Editing while running → creates **new session** (clone pattern)

**Implementation:**

```go
func (r *SessionReconciler) reconcileSession(ctx context.Context, session *unstructured.Unstructured) (ctrl.Result, error) {
    // Check if spec changed during execution
    observedGen := getObservedGeneration(session)
    currentGen := session.GetGeneration()
    
    if observedGen > 0 && currentGen > observedGen {
        currentPhase := getPhase(session)
        
        // If running, spec changes should stop the session and require restart
        if currentPhase == "Running" || currentPhase == "Creating" {
            r.updateCondition(ctx, session, ConditionTypeReady, metav1.ConditionFalse, 
                "SpecChanged", 
                "Spec was modified during execution - session must be restarted manually")
            r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
                "SpecModified", 
                "Cannot apply spec changes to running session")
            
            // Stop the job
            r.deleteJob(ctx, session)
            
            return ctrl.Result{}, nil // User must restart
        }
    }
    
    // Update observedGeneration after processing
    r.updateStatus(ctx, session, map[string]interface{}{
        "observedGeneration": currentGen,
    })
    
    // ... continue with normal reconciliation
}
```

**User Experience:**
```bash
# User edits running session
kubectl edit agenticsession session-123
# Change spec.prompt = "New prompt"

# Operator detects change and stops session
kubectl get agenticsession session-123
# Status: Failed, reason: SpecModified, message: "Cannot apply spec changes to running session"

# User must explicitly restart
curl -X POST /api/projects/myproject/agentic-sessions/session-123/start
```

### Type 2: Runtime Additions (Annotations) - Dynamic

**Examples:**
- Adding repos mid-session
- Switching workflows mid-session
- Recording SDK session ID

**Rules:**
- ✅ Stored in annotations (not spec)
- ✅ Applied by runner without restart
- ✅ Runner restarts SDK internally to pick up changes
- ✅ Preserved across pod restarts (annotations persist)

**Data Flow:**

```
┌──────────────┐
│ User (UI)    │
│  "Add repo"  │
└──────┬───────┘
       │
       │ WebSocket message
       ▼
┌──────────────────────────────────────────────────────┐
│ Backend                                              │
│  1. Validates user has permission                    │
│  2. Updates CR annotation with new repo              │
│  3. Forwards message to runner via WebSocket         │
└──────────────────────────────────────────────────────┘
       │
       │ annotation update                │ WebSocket
       ▼                                  ▼
┌──────────────────┐              ┌──────────────────┐
│ AgenticSession   │              │ Runner Pod       │
│ annotations:     │              │  1. Clone repo   │
│   repos-runtime: │              │  2. Update env   │
│   '["new-repo"]' │◄─────────────│  3. Restart SDK  │
└──────────────────┘   writes     └──────────────────┘
       │
       │ observes
       ▼
┌──────────────────┐
│ Operator         │
│  - Sees annotation │
│  - Updates condition: │
│    RuntimeReposAdded │
└──────────────────┘
```

**Implementation:**

> **Runtime guard rails:** The backend now enforces that runtime repo/workflow mutations are only accepted when the session is both interactive and currently in the `Running` phase. Calls made outside that window receive `409 Conflict`, allowing the UI to surface actionable errors instead of letting the operator chase impossible updates.

```go
// Backend: Handle runtime repo addition
func AddRepoToSession(c *gin.Context) {
    project := c.GetString("project")
    sessionName := c.Param("sessionName")
    
    var req struct {
        URL    string `json:"url" binding:"required"`
        Branch string `json:"branch"`
        Name   string `json:"name" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    gvr := GetAgenticSessionV1Alpha1Resource()
    
    // Get current session
    session, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
        return
    }
    
    // Ensure session is interactive and running
    spec, _ := session.Object["spec"].(map[string]interface{})
    interactive, _ := spec["interactive"].(bool)
    if !interactive {
        c.JSON(http.StatusConflict, gin.H{"error": "Can only add repos to interactive sessions"})
        return
    }
    
    status, _ := session.Object["status"].(map[string]interface{})
    phase, _ := status["phase"].(string)
    if phase != "Running" {
        c.JSON(http.StatusConflict, gin.H{"error": "Session must be running to add repos"})
        return
    }
    
    // Read current runtime repos from annotation
    annotations := session.GetAnnotations()
    if annotations == nil {
        annotations = make(map[string]string)
    }
    
    runtimeReposJSON := annotations["ambient-code.io/repos-added-at-runtime"]
    var runtimeRepos []map[string]string
    if runtimeReposJSON != "" {
        json.Unmarshal([]byte(runtimeReposJSON), &runtimeRepos)
    }
    
    // Add new repo
    runtimeRepos = append(runtimeRepos, map[string]string{
        "url":    req.URL,
        "branch": req.Branch,
        "name":   req.Name,
    })
    
    // Update annotation
    newJSON, _ := json.Marshal(runtimeRepos)
    annotations["ambient-code.io/repos-added-at-runtime"] = string(newJSON)
    session.SetAnnotations(annotations)
    
    // Patch the session (uses user token)
    _, err = reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), session, v1.UpdateOptions{})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
        return
    }
    
    // Forward to runner via WebSocket
    SendMessageToSession(project, sessionName, map[string]interface{}{
        "type": "repo_added",
        "payload": map[string]string{
            "url":    req.URL,
            "branch": req.Branch,
            "name":   req.Name,
        },
    })
    
    c.JSON(http.StatusOK, gin.H{"message": "Repo added successfully"})
}
```

**Runner: Reads annotations on restart**

```python
async def _prepare_workspace(self):
    """Clone input repos, including any added at runtime."""
    
    # Get base repos from spec
    repos_cfg = self._get_repos_config()
    
    # Get additional repos from annotation (added at runtime)
    runtime_repos = await self._get_runtime_repos_from_annotation()
    
    # Merge them
    all_repos = repos_cfg + runtime_repos
    
    # Clone all repos
    for repo in all_repos:
        # ... clone logic
```

**Operator: Updates condition based on annotation**

```go
func (r *SessionReconciler) reconcileSession(ctx context.Context, session *unstructured.Unstructured) (ctrl.Result, error) {
    // Check for runtime repos addition
    annotations := session.GetAnnotations()
    if runtimeReposJSON, ok := annotations["ambient-code.io/repos-added-at-runtime"]; ok && runtimeReposJSON != "" {
        var runtimeRepos []map[string]string
        json.Unmarshal([]byte(runtimeReposJSON), &runtimeRepos)
        
        if len(runtimeRepos) > 0 {
            r.updateCondition(ctx, session, "RuntimeReposAdded", metav1.ConditionTrue, 
                "ReposModified", 
                fmt.Sprintf("%d repos added at runtime", len(runtimeRepos)))
        }
    }
    
    // ... rest of reconciliation
}
```

### Type 3: Interactive Prompts (WebSocket) - Ephemeral

**Examples:**
- User sends chat message
- User sends interrupt
- User sends workflow switch command

**Rules:**
- ❌ NOT stored in CR at all
- ✅ Sent directly to runner via WebSocket
- ✅ Runner handles immediately
- ✅ Messages stored in backend (not CR)

**Data Flow:**

```
┌──────────────┐
│ User (UI)    │
│  Types chat  │
└──────┬───────┘
       │
       │ WebSocket
       ▼
┌──────────────────────────────────────────────────────┐
│ Backend WebSocket Handler                            │
│  - Validates session is interactive                  │
│  - Stores message in backend (optional)              │
│  - Forwards to runner pod                            │
│  - Does NOT update CR                                │
└──────┬───────────────────────────────────────────────┘
       │
       │ WebSocket
       ▼
┌──────────────────┐
│ Runner Pod       │
│  - Receives msg  │
│  - Queues it     │
│  - Sends to SDK  │
│  - NO CR update  │
└──────────────────┘
```

**No operator involvement** - these are runtime interactions, not desired state.

## Decision Matrix: Where Does Each Change Go?

| Change Type | Where Stored | Who Applies | Restart Needed | Example |
|-------------|-------------|-------------|----------------|---------|
| Initial prompt | `spec.prompt` | Operator (job creation) | N/A (creation time) | "Build a web app" |
| Initial repos | `spec.repos` | Operator (job creation) | N/A (creation time) | `[{url: "..."}]` |
| Initial workflow | `spec.activeWorkflow` | Operator (job creation) | N/A (creation time) | `{gitUrl: "..."}` |
| **Runtime repo add** | `annotations["repos-added-at-runtime"]` | Runner (clones) + Backend (annotation) | SDK restart only | User clicks "Add repo" |
| **Runtime workflow switch** | `annotations["workflow-switched-to"]` + env var | Runner (clones) + Backend (annotation) | SDK restart only | User selects new workflow |
| Chat message | Backend storage (not CR) | Runner (SDK) | No | "Fix the bug in auth.py" |
| Interrupt | Ephemeral (WebSocket) | Runner (SDK) | No | User clicks interrupt |
| **Edit prompt (running)** | ❌ Rejected or new session | N/A | Yes | Editing spec while running |
| Edit LLM settings (running) | ❌ Rejected or new session | N/A | Yes | Changing model mid-session |

## Implementation: Unified Approach

### Backend API Endpoints

```go
// Endpoints for interactive sessions
router.POST("/api/projects/:project/agentic-sessions/:sessionName/repos", AddRepoToSession)
router.DELETE("/api/projects/:project/agentic-sessions/:sessionName/repos/:repoName", RemoveRepoFromSession)
router.POST("/api/projects/:project/agentic-sessions/:sessionName/workflow", SwitchWorkflow)
router.POST("/api/projects/:project/agentic-sessions/:sessionName/messages", SendChatMessage)

// Endpoints for spec changes (only allowed when stopped)
router.PUT("/api/projects/:project/agentic-sessions/:sessionName", UpdateSessionSpec) // Validates phase != Running
```

### Backend: Validate State Before Allowing Changes

```go
func UpdateSessionSpec(c *gin.Context) {
    // ... get session ...
    
    status, _ := session.Object["status"].(map[string]interface{})
    phase, _ := status["phase"].(string)
    
    // Only allow spec changes for stopped sessions
    if phase == "Running" || phase == "Creating" {
        c.JSON(http.StatusConflict, gin.H{
            "error": "Cannot modify spec while session is running",
            "action": "Stop the session first, or create a new session with updated settings",
        })
        return
    }
    
    // Proceed with spec update
    // ...
}
```

### Operator: Handle observedGeneration

```go
func (r *SessionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    session := &unstructured.Unstructured{}
    if err := r.Get(ctx, req.NamespacedName, session); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    
    currentGen := session.GetGeneration()
    observedGen := getObservedGeneration(session)
    
    // First time reconciling this session
    if observedGen == 0 {
        r.updateStatus(ctx, session, map[string]interface{}{
            "observedGeneration": currentGen,
        })
        return r.reconcileSession(ctx, session)
    }
    
    // Spec changed since last reconciliation
    if currentGen > observedGen {
        phase := getPhase(session)
        
        // If running, stop and require manual restart
        if phase == "Running" || phase == "Creating" {
            log.Printf("Spec changed during execution (gen %d→%d), stopping session %s", 
                observedGen, currentGen, session.GetName())
            
            r.updateCondition(ctx, session, ConditionTypeFailed, metav1.ConditionTrue, 
                "SpecModified", 
                "Spec was modified during execution - session stopped")
            
            r.deleteJob(ctx, session)
            
            r.updateStatus(ctx, session, map[string]interface{}{
                "observedGeneration": currentGen,
                "completionTime": metav1.Now(),
            })
            
            return ctrl.Result{}, nil
        }
        
        // If pending/stopped, apply new spec
        log.Printf("Applying spec changes for session %s (gen %d→%d)", 
            session.GetName(), observedGen, currentGen)
        
        r.updateStatus(ctx, session, map[string]interface{}{
            "observedGeneration": currentGen,
        })
    }
    
    return r.reconcileSession(ctx, session)
}
```

### Runner: Sync on Startup from Annotations

```python
async def _prepare_workspace(self):
    """Prepare workspace with all repos (initial + runtime additions)."""
    
    # 1. Get initial repos from spec (REPOS_JSON env var)
    initial_repos = self._get_repos_config()
    
    # 2. Get runtime-added repos from annotation
    runtime_repos = await self._fetch_runtime_repos_annotation()
    
    # 3. Merge them (deduplicate by name)
    all_repos = self._merge_repos(initial_repos, runtime_repos)
    
    # 4. Clone all repos
    for repo in all_repos:
        await self._clone_or_update_repo(repo)
    
    logging.info(f"Workspace prepared with {len(all_repos)} repos")

async def _fetch_runtime_repos_annotation(self) -> list[dict]:
    """Fetch repos that were added at runtime from CR annotation."""
    try:
        # Build annotation URL from status URL
        status_url = self._compute_status_url()
        # ... transform to GET session URL ...
        
        resp = await self._http_get(url, headers={'Authorization': f'Bearer {bot_token}'})
        data = json.loads(resp)
        
        annotations = data.get('metadata', {}).get('annotations', {})
        repos_json = annotations.get('ambient-code.io/repos-added-at-runtime', '[]')
        
        runtime_repos = json.loads(repos_json)
        logging.info(f"Found {len(runtime_repos)} runtime-added repos")
        return runtime_repos
    except Exception as e:
        logging.warning(f"Failed to fetch runtime repos: {e}")
        return []
```

## UI Behavior

### When Session is Stopped

```typescript
// All spec fields editable
<SessionForm 
  editable={session.status?.phase !== 'Running'} 
  onSave={async (updates) => {
    // PUT /api/projects/:project/agentic-sessions/:sessionName
    await updateSessionSpec(projectName, sessionName, updates)
  }}
/>
```

### When Session is Running (Interactive)

```typescript
// Runtime actions available
<InteractiveSession session={session}>
  <Button onClick={() => addRepo(repoConfig)}>
    Add Repository
  </Button>
  
  <Button onClick={() => switchWorkflow(workflowConfig)}>
    Switch Workflow
  </Button>
  
  <ChatInput onSend={(msg) => sendChatMessage(msg)} />
</InteractiveSession>

// Spec fields LOCKED
<SessionForm editable={false} showWarning="Cannot edit spec while running" />
```

### When User Tries to Edit Running Session

```typescript
async function handleEditSpec() {
  try {
    await updateSessionSpec(projectName, sessionName, updates)
  } catch (error) {
    if (error.status === 409) {
      // Show dialog
      showDialog({
        title: "Session is Running",
        message: "Cannot modify session configuration while running.",
        options: [
          {
            label: "Stop and Edit",
            action: async () => {
              await stopSession(projectName, sessionName)
              await updateSessionSpec(projectName, sessionName, updates)
            }
          },
          {
            label: "Create New Session",
            action: () => {
              navigate(`/projects/${projectName}/sessions/new`, {
                state: { cloneFrom: sessionName, updates }
              })
            }
          },
          {
            label: "Cancel",
            action: () => {}
          }
        ]
      })
    }
  }
}
```

## Benefits of This Approach

✅ **Clear semantics**
- Spec = initial desired state
- Annotations = runtime modifications
- Status = observed state

✅ **No ambiguity**
- Editing running session → explicit user choice (stop or clone)
- Runtime additions → stored in annotations, survive restarts

✅ **Kubernetes-native**
- observedGeneration tracks spec changes
- Operator reconciles on generation change
- Follows standard controller patterns

✅ **Recoverable**
- Pod crashes → restart with annotations intact
- All runtime state preserved in CR

✅ **Debuggable**
- Can see what was added at runtime vs initial
- Condition shows "RuntimeReposAdded"
- Annotations show history

## Summary

| Question | Answer |
|----------|--------|
| "Should spec changes update running session?" | **No** - stop and restart, or clone to new session |
| "Who applies runtime changes?" | **Runner** (execution) + **Backend** (annotation write) |
| "How to keep CR in sync?" | **Annotations** store runtime state, **observedGeneration** tracks spec changes |
| "What if pod restarts?" | **Annotations persist** - runner reads them on startup |
| "Can user edit prompt mid-session?" | **No** - must stop session or create new one |
| "Can user add repos mid-session?" | **Yes** - via annotation + WebSocket to runner |

