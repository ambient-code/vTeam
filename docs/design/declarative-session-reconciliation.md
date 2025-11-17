# Declarative Session Reconciliation (The Right Way)

## The Kubernetes Pattern You Described

You're proposing the **correct declarative pattern**:

```
User wants: "Add repo2"
    ↓
Backend: Updates spec.repos (adds repo2)  ← Declares desired state
    ↓
CR generation increments
    ↓
Operator: "Spec says repo2 should exist, status says it doesn't"
    ↓
Operator: Calls content service to clone repo2
    ↓
Operator: Updates status.observedRepos (now includes repo2)
```

This is **declarative** (what should exist) vs **imperative** (do this action now).

## Architecture: Spec = Desired State, Status = Observed State

### The CR Structure

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: session-123
  
spec:
  # ============================================
  # DESIRED STATE - What should be present
  # ============================================
  
  # Initial prompt - only used on first SDK invocation
  initialPrompt: "Build a web app"
  
  # Repos that SHOULD be cloned in workspace
  repos:
    - url: "https://github.com/org/repo1"
      branch: "main"
    - url: "https://github.com/org/repo2"  # User adds this
      branch: "feature"
  
  # Workflow that SHOULD be active
  activeWorkflow:
    gitUrl: "https://github.com/org/workflow-speckit"
    branch: "main"
    path: ""  # Optional subdirectory
  
  # LLM settings for the session
  llmSettings:
    model: "sonnet"
    temperature: 0.7
    maxTokens: 4000
  
  # Session behavior
  interactive: true
  timeout: 3600

status:
  # ============================================
  # OBSERVED STATE - What actually exists
  # ============================================
  
  observedGeneration: 5  # Which spec version we last processed
  phase: Running
  
  # Conditions (detailed status)
  conditions:
    - type: Ready
      status: "True"
    - type: RunnerStarted
      status: "True"
  
  # What repos are actually cloned
  reconciledRepos:
    - url: "https://github.com/org/repo1"
      branch: "main"
      clonedAt: "2025-11-15T12:00:00Z"
      status: Ready
    - url: "https://github.com/org/repo2"
      branch: "feature"
      clonedAt: "2025-11-15T12:30:00Z"
      status: Ready
  
  # What workflow is actually active
  reconciledWorkflow:
    gitUrl: "https://github.com/org/workflow-speckit"
    branch: "main"
    appliedAt: "2025-11-15T12:00:00Z"
    status: Active
  
  # SDK session details
  sdkSessionId: "abc-def-123"
  sdkRestartCount: 2  # How many times SDK was restarted
```

### Key Insight: `initialPrompt` vs Runtime Chat

You're right - **initial prompt should only be used once**!

```yaml
spec:
  initialPrompt: "Build a web app"  # Used only on FIRST SDK invocation
  # NOT used on continuation sessions
  # NOT used after workflow switches (workflow has its own startupPrompt)
```

**Chat messages** are different - they're runtime interactions:
- Not stored in spec (they're not desired state)
- Not stored in status (they're not observed state)
- Stored in **backend message log** or **PVC files**

## Reconciliation Flow: Operator as the Brain

### 1. User Adds a Repo

```typescript
// UI: User clicks "Add Repository"
await fetch('/api/projects/myproject/agentic-sessions/session-123', {
  method: 'PATCH',
  body: JSON.stringify({
    spec: {
      repos: [
        ...existingRepos,
        { url: "https://github.com/org/new-repo", branch: "main" }
      ]
    }
  })
})
```

**Backend:**
```go
func PatchSessionSpec(c *gin.Context) {
    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    
    // Get current session
    session, err := reqDyn.Resource(gvr).Namespace(project).Get(...)
    
    // Validate user can update this session (RBAC check)
    // ...
    
    // Apply patch to spec
    var patch map[string]interface{}
    c.ShouldBindJSON(&patch)
    
    spec := session.Object["spec"].(map[string]interface{})
    
    // Merge patch into spec (strategic merge)
    if specPatch, ok := patch["spec"].(map[string]interface{}); ok {
        for k, v := range specPatch {
            spec[k] = v
        }
    }
    
    // Update the CR (this increments .metadata.generation)
    updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), session, v1.UpdateOptions{})
    
    c.JSON(http.StatusOK, updated)
}
```

**Operator Reconciliation:**
```go
func (r *SessionReconciler) reconcileSession(ctx context.Context, session *unstructured.Unstructured) (ctrl.Result, error) {
    name := session.GetName()
    namespace := session.GetNamespace()
    
    // Get desired state from spec
    spec, _ := unstructured.NestedMap(session.Object, "spec")
    desiredRepos := getReposFromSpec(spec)
    desiredWorkflow := getWorkflowFromSpec(spec)
    
    // Get observed state from status
    status, _ := unstructured.NestedMap(session.Object, "status")
    reconciledRepos := getReposFromStatus(status)
    reconciledWorkflow := getWorkflowFromStatus(status)
    
    // Check if session is running
    phase := getPhase(session)
    if phase != "Running" {
        // Can't reconcile repos if not running
        return ctrl.Result{}, nil
    }
    
    // ============================================
    // RECONCILE REPOS
    // ============================================
    
    // Find repos to ADD (in desired but not in observed)
    for _, desired := range desiredRepos {
        found := false
        for _, observed := range reconciledRepos {
            if observed.URL == desired.URL {
                found = true
                
                // Check if branch changed
                if observed.Branch != desired.Branch {
                    log.Printf("Repo %s branch changed: %s → %s", desired.URL, observed.Branch, desired.Branch)
                    // Tell content service to update branch
                    if err := r.updateRepoBranch(ctx, namespace, name, desired); err != nil {
                        log.Printf("Failed to update repo branch: %v", err)
                        continue
                    }
                    // Update status
                    r.updateRepoInStatus(ctx, session, desired, "Ready")
                }
                break
            }
        }
        
        if !found {
            log.Printf("Repo %s not yet cloned, requesting clone", desired.URL)
            
            // Call content service to clone repo
            if err := r.cloneRepo(ctx, namespace, name, desired); err != nil {
                log.Printf("Failed to clone repo: %v", err)
                r.updateRepoInStatus(ctx, session, desired, "Cloning")
                continue
            }
            
            // Update status to reflect repo was cloned
            r.addRepoToStatus(ctx, session, desired)
        }
    }
    
    // Find repos to REMOVE (in observed but not in desired)
    for _, observed := range reconciledRepos {
        found := false
        for _, desired := range desiredRepos {
            if observed.URL == desired.URL {
                found = true
                break
            }
        }
        
        if !found {
            log.Printf("Repo %s should be removed", observed.URL)
            
            // Call content service to remove repo directory
            if err := r.removeRepo(ctx, namespace, name, observed); err != nil {
                log.Printf("Failed to remove repo: %v", err)
                continue
            }
            
            // Update status to remove repo
            r.removeRepoFromStatus(ctx, session, observed)
        }
    }
    
    // ============================================
    // RECONCILE WORKFLOW
    // ============================================
    
    // Check if workflow changed
    if desiredWorkflow != nil {
        needsWorkflowSwitch := false
        
        if reconciledWorkflow == nil {
            // No workflow active, but one is desired
            needsWorkflowSwitch = true
        } else if reconciledWorkflow.GitURL != desiredWorkflow.GitURL || 
                  reconciledWorkflow.Branch != desiredWorkflow.Branch {
            // Different workflow desired
            needsWorkflowSwitch = true
        }
        
        if needsWorkflowSwitch {
            log.Printf("Workflow needs to be switched to %s", desiredWorkflow.GitURL)
            
            // Call content service to clone workflow
            if err := r.cloneWorkflow(ctx, namespace, name, desiredWorkflow); err != nil {
                log.Printf("Failed to clone workflow: %v", err)
                r.updateCondition(ctx, session, "WorkflowReconciled", metav1.ConditionFalse, 
                    "CloneFailed", fmt.Sprintf("Failed to clone workflow: %v", err))
                return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
            }
            
            // Request SDK restart to pick up new workflow
            if err := r.restartSDK(ctx, namespace, name); err != nil {
                log.Printf("Failed to restart SDK: %v", err)
                return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
            }
            
            // Update status
            r.updateWorkflowInStatus(ctx, session, desiredWorkflow)
            r.updateStatus(ctx, session, map[string]interface{}{
                "sdkRestartCount": (getSDKRestartCount(status) + 1),
            })
        }
    }
    
    // Update observedGeneration to indicate we processed this spec version
    r.updateStatus(ctx, session, map[string]interface{}{
        "observedGeneration": session.GetGeneration(),
    })
    
    return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}
```

### 2. Operator Calls Content Service

The **content service** (running in the same pod as the runner) provides APIs to mutate the workspace:

```go
// Operator calls content service HTTP API
func (r *SessionReconciler) cloneRepo(ctx context.Context, namespace, sessionName string, repo RepoConfig) error {
    // Build content service URL
    svcName := fmt.Sprintf("ambient-content-%s", sessionName)
    url := fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/repos/clone", svcName, namespace)
    
    body, _ := json.Marshal(map[string]interface{}{
        "url":    repo.URL,
        "branch": repo.Branch,
        "name":   repo.Name,
    })
    
    req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("content service call failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("content service returned %d: %s", resp.StatusCode, string(body))
    }
    
    log.Printf("Successfully requested repo clone via content service")
    return nil
}

func (r *SessionReconciler) restartSDK(ctx context.Context, namespace, sessionName string) error {
    svcName := fmt.Sprintf("ambient-content-%s", sessionName)
    url := fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/sdk/restart", svcName, namespace)
    
    req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("SDK restart request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("SDK restart returned %d", resp.StatusCode)
    }
    
    log.Printf("Successfully requested SDK restart")
    return nil
}
```

### 3. Content Service Implementation

The **content service** needs new endpoints to handle reconciliation:

```python
# New file: components/runners/claude-code-runner/content_service.py

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import asyncio
import subprocess
from pathlib import Path

app = FastAPI()

class CloneRepoRequest(BaseModel):
    url: str
    branch: str
    name: str

class UpdateBranchRequest(BaseModel):
    name: str
    branch: str

@app.post("/repos/clone")
async def clone_repo(req: CloneRepoRequest):
    """Clone a repository into the workspace."""
    workspace = Path(os.getenv("WORKSPACE_PATH", "/workspace"))
    repo_dir = workspace / req.name
    
    if repo_dir.exists():
        return {"status": "already_exists", "path": str(repo_dir)}
    
    try:
        # Get GitHub token
        token = os.getenv("GITHUB_TOKEN", "")
        clone_url = _add_token_to_url(req.url, token) if token else req.url
        
        # Clone the repo
        proc = await asyncio.create_subprocess_exec(
            "git", "clone", "--branch", req.branch, "--single-branch", 
            clone_url, str(repo_dir),
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        await proc.communicate()
        
        if proc.returncode != 0:
            raise HTTPException(status_code=500, detail="Clone failed")
        
        # Configure git identity
        await _run_cmd(["git", "config", "user.name", "Ambient Bot"], cwd=repo_dir)
        await _run_cmd(["git", "config", "user.email", "bot@ambient.local"], cwd=repo_dir)
        
        # Signal runner that repo was added
        await _signal_runner("repo_added", {"name": req.name})
        
        return {"status": "cloned", "path": str(repo_dir)}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/repos/{name}/update-branch")
async def update_repo_branch(name: str, req: UpdateBranchRequest):
    """Update a repo to a different branch."""
    workspace = Path(os.getenv("WORKSPACE_PATH", "/workspace"))
    repo_dir = workspace / name
    
    if not repo_dir.exists():
        raise HTTPException(status_code=404, detail="Repo not found")
    
    try:
        # Fetch and checkout new branch
        await _run_cmd(["git", "fetch", "origin", req.branch], cwd=repo_dir)
        await _run_cmd(["git", "checkout", req.branch], cwd=repo_dir)
        await _run_cmd(["git", "reset", "--hard", f"origin/{req.branch}"], cwd=repo_dir)
        
        return {"status": "updated", "branch": req.branch}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.delete("/repos/{name}")
async def remove_repo(name: str):
    """Remove a repository from the workspace."""
    workspace = Path(os.getenv("WORKSPACE_PATH", "/workspace"))
    repo_dir = workspace / name
    
    if not repo_dir.exists():
        return {"status": "not_found"}
    
    try:
        shutil.rmtree(repo_dir)
        
        # Signal runner that repo was removed
        await _signal_runner("repo_removed", {"name": name})
        
        return {"status": "removed"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/workflows/clone")
async def clone_workflow(req: CloneRepoRequest):
    """Clone a workflow into workspace/workflows/."""
    workspace = Path(os.getenv("WORKSPACE_PATH", "/workspace"))
    workflow_dir = workspace / "workflows" / req.name
    
    if workflow_dir.exists():
        return {"status": "already_exists", "path": str(workflow_dir)}
    
    try:
        token = os.getenv("GITHUB_TOKEN", "")
        clone_url = _add_token_to_url(req.url, token) if token else req.url
        
        # Clone to temp directory
        temp_dir = workflow_dir.parent / f"{req.name}-temp"
        proc = await asyncio.create_subprocess_exec(
            "git", "clone", "--branch", req.branch, "--single-branch",
            clone_url, str(temp_dir),
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        await proc.communicate()
        
        if proc.returncode != 0:
            raise HTTPException(status_code=500, detail="Clone failed")
        
        # Move to final location
        temp_dir.rename(workflow_dir)
        
        return {"status": "cloned", "path": str(workflow_dir)}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/sdk/restart")
async def restart_sdk():
    """Signal the runner to restart the SDK with new configuration."""
    try:
        # Set flag that wrapper.py checks
        flag_file = Path("/workspace/.sdk-restart-requested")
        flag_file.write_text("restart")
        
        # Send signal to runner process (wrapper.py)
        await _signal_runner("sdk_restart", {})
        
        return {"status": "restart_requested"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

async def _signal_runner(signal_type: str, payload: dict):
    """Send signal to runner wrapper via shared file or queue."""
    signals_dir = Path("/workspace/.signals")
    signals_dir.mkdir(exist_ok=True)
    
    signal_file = signals_dir / f"{signal_type}_{int(time.time())}.json"
    signal_file.write_text(json.dumps(payload))

async def _run_cmd(cmd, cwd):
    proc = await asyncio.create_subprocess_exec(
        *cmd,
        cwd=cwd,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE
    )
    stdout, stderr = await proc.communicate()
    if proc.returncode != 0:
        raise RuntimeError(f"Command failed: {stderr.decode()}")
```

### 4. Runner Wrapper Checks for Signals

```python
# In wrapper.py
async def _run_claude_agent_sdk(self, prompt: str):
    # ... existing SDK setup ...
    
    async with ClaudeSDKClient(options=options) as client:
        # ... existing startup logic ...
        
        if interactive:
            # Interactive loop
            while True:
                # Check for signals from content service
                if await self._check_sdk_restart_signal():
                    logging.info("SDK restart signal received, breaking loop")
                    self._restart_requested = True
                    break
                
                # ... existing message handling ...
```

## Benefits of This Pattern

✅ **Declarative**: User declares "I want repo2" not "clone repo2 now"
✅ **Kubernetes-Native**: Spec = desired, Status = observed, Operator = reconciler
✅ **Crash-Safe**: If operator crashes mid-reconciliation, it resumes on restart
✅ **Observable**: Status shows exactly what's been reconciled
✅ **Auditable**: Can see spec changes in CR history
✅ **Idempotent**: Operator can reconcile multiple times safely

## Handling Different Actions

### User Adds Repo

```
spec.repos: [repo1, repo2]  ← User adds repo2
status.reconciledRepos: [repo1]  ← Not yet reconciled
    ↓
Operator: "repo2 missing, clone it"
    ↓
Calls content service: POST /repos/clone
    ↓
status.reconciledRepos: [repo1, repo2]  ← Now reconciled
```

### User Switches Workflow

```
spec.activeWorkflow: workflow-B  ← User switches
status.reconciledWorkflow: workflow-A  ← Currently active
    ↓
Operator: "Workflow changed, switch it"
    ↓
Calls content service: POST /workflows/clone (workflow-B)
Calls content service: POST /sdk/restart
    ↓
status.reconciledWorkflow: workflow-B  ← Now reconciled
status.sdkRestartCount: 3  ← Incremented
```

### User Restarts Session

**Option A: Via status update** (your annotation idea):
```yaml
metadata:
  annotations:
    ambient-code.io/requested-action: "restart"  ← User clicks restart
```

Operator sees annotation → resets status to Pending → triggers new Job

**Option B: Via DELETE + CREATE pattern** (more Kubernetes-native):
```bash
# Backend deletes old Job, sets status to Stopped
DELETE /api/projects/myproject/agentic-sessions/session-123/job

# Backend resets status to Pending
PATCH /api/projects/myproject/agentic-sessions/session-123/status
{"phase": "Pending"}

# Operator sees Pending → creates new Job
```

## Answer to Your Question

> "Is this too much on the operator?"

**No! This is EXACTLY what operators are for!**

The operator's job is to **reconcile desired state (spec) with observed state (status)**.

**What operator should do:**
- ✅ Watch spec changes
- ✅ Call content service to make changes
- ✅ Update status to reflect what was done
- ✅ Retry on failures
- ✅ Handle race conditions

**What operator should NOT do:**
- ❌ Business logic (that's in runner/content service)
- ❌ Direct file system manipulation
- ❌ Auth/RBAC (that's in backend)

## Implementation Priority

1. **Phase 1**: Add content service HTTP endpoints
2. **Phase 2**: Update operator to call content service
3. **Phase 3**: Update backend to allow spec patches
4. **Phase 4**: Update UI to patch spec instead of WebSocket

This is the **right** pattern - let's build it!

