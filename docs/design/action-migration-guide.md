# Complete Action Migration Guide: Backend → Operator-Centric

## Current State Analysis

We have **35 session-related actions** spread across backend, operator, and runner.

**Problem:** Responsibility is unclear - some actions modify CR directly (backend), some through operator, some through runner.

**Solution:** Migrate to **declarative operator-centric pattern** where:
- Backend = API gateway (validation, RBAC, spec updates)
- Operator = Reconciler (makes observed state match desired state)
- Content Service = Workspace mutator (executes git/file operations)
- Runner = SDK executor (no CR writes)

---

## Migration Categories

### 🔵 Category A: Pure CRUD (No Migration Needed)
These already follow the pattern - backend validates + updates CR, operator reacts.

### 🟢 Category B: Migrate to Spec Updates
Currently use imperative actions (WebSocket, direct calls). Should update spec, operator reconciles.

### 🟡 Category C: Migrate to Content Service
Currently backend directly manipulates workspace. Should call content service instead.

### 🔴 Category D: Remove Backend Involvement
Currently backend does operator's job. Operator should own these entirely.

---

## Detailed Migration Plan

### 1. Session Lifecycle Actions

#### `POST /api/projects/:project/agentic-sessions` - Create Session
**Current:** Backend creates CR with spec + initial status  
**Migration:** 🔵 **No change needed** (already correct)

```go
// Backend: Validates + creates CR spec
func CreateSession(c *gin.Context) {
    // Validate request
    // Create CR with spec only
    obj := &unstructured.Unstructured{
        Object: map[string]interface{}{
            "spec": spec,
            "status": {"phase": "Pending"},  // Initial only
        },
    }
    reqDyn.Resource(gvr).Namespace(project).Create(obj)
}

// Operator: Sees Pending → provisions resources → creates Job
func (r *SessionReconciler) reconcileSession(session) {
    if phase == "Pending" {
        r.ensurePVC()
        r.ensureSecrets()
        r.createJob()
    }
}
```

**Responsibility:**
- ✅ Backend: Validation, RBAC, CR creation
- ✅ Operator: Resource provisioning, Job creation

---

#### `GET /api/projects/:project/agentic-sessions/:session` - Get Session
**Current:** Backend reads CR  
**Migration:** 🔵 **No change needed**

```go
// Backend: Simple CR read using user token
func GetSession(c *gin.Context) {
    reqDyn := GetK8sClientsForRequest(c)
    session := reqDyn.Resource(gvr).Namespace(project).Get(sessionName)
    c.JSON(200, session)
}
```

**Responsibility:**
- ✅ Backend: RBAC-enforced read

---

#### `DELETE /api/projects/:project/agentic-sessions/:session` - Delete Session
**Current:** Backend deletes CR, K8s GC handles cleanup  
**Migration:** 🔵 **No change needed**

```go
// Backend: Delete CR using user token
func DeleteSession(c *gin.Context) {
    reqDyn := GetK8sClientsForRequest(c)
    reqDyn.Resource(gvr).Namespace(project).Delete(sessionName)
    c.JSON(204, nil)
}

// Kubernetes GC: Deletes owned resources (Job, PVC, Secrets)
```

**Responsibility:**
- ✅ Backend: RBAC-enforced delete
- ✅ K8s GC: Cleanup via OwnerReferences

---

#### `POST /api/projects/:project/agentic-sessions/:session/start` - Start/Restart Session
**Current:** Backend deletes Job, updates status to Pending, operator sees Pending → creates new Job  
**Migration:** 🟢 **Simplify to spec update**

**BEFORE:**
```go
// Backend does too much
func StartSession(c *gin.Context) {
    // Delete temp pod
    reqK8s.CoreV1().Pods(project).Delete(tempPodName)
    // Update status to Pending
    DynamicClient.Resource(gvr).Namespace(project).UpdateStatus(...)
}
```

**AFTER:**
```go
// Backend: Just update spec.restartRequested or reset phase
func StartSession(c *gin.Context) {
    reqDyn := GetK8sClientsForRequest(c)
    session := reqDyn.Get(...)
    
    // Check if terminal phase
    currentPhase := getPhase(session)
    if currentPhase == "Completed" || currentPhase == "Failed" || currentPhase == "Stopped" {
        // Reset status to Pending (using backend SA - one-time write)
        if DynamicClient != nil {
            status := map[string]interface{}{"phase": "Pending", "message": "Restart requested"}
            DynamicClient.Resource(gvr).Namespace(project).UpdateStatus(...)
        }
        c.JSON(200, gin.H{"message": "Restart initiated"})
        return
    }
    
    c.JSON(400, gin.H{"error": "Can only restart completed/failed sessions"})
}

// Operator: Sees Pending → cleans up old job → creates new job
func (r *SessionReconciler) reconcileSession(session) {
    if phase == "Pending" {
        // Cleanup any leftover resources
        r.deleteJobIfExists()
        r.deleteContentPodIfExists()
        
        // Provision fresh resources
        r.ensurePVC()
        r.ensureFreshToken()
        r.createJob()
    }
}
```

**Responsibility:**
- ✅ Backend: Reset status to Pending
- ✅ Operator: Full reconciliation (cleanup + recreate)

---

#### `POST /api/projects/:project/agentic-sessions/:session/stop` - Stop Session
**Current:** Backend deletes Job + pods, updates status to Stopped  
**Migration:** 🔴 **Backend should only update status, operator handles cleanup**

**BEFORE:**
```go
// Backend does too much - manipulates pods directly
func StopSession(c *gin.Context) {
    // Delete job
    reqK8s.BatchV1().Jobs(project).Delete(jobName)
    // Delete pods
    reqK8s.CoreV1().Pods(project).DeleteCollection(...)
    // Update status
    DynamicClient.Resource(gvr).UpdateStatus(...)
}
```

**AFTER:**
```go
// Backend: Just update status, operator does the rest
func StopSession(c *gin.Context) {
    reqDyn := GetK8sClientsForRequest(c)
    
    // Verify user has permission to update this session
    // ...
    
    // Update status to Stopped (using backend SA)
    if DynamicClient != nil {
        status := map[string]interface{}{
            "phase": "Stopped",
            "message": "User requested stop",
            "completionTime": time.Now().Format(time.RFC3339),
        }
        DynamicClient.Resource(gvr).Namespace(project).UpdateStatus(...)
    }
    
    c.JSON(200, gin.H{"message": "Stop initiated"})
}

// Operator: Sees Stopped → cleans up resources
func (r *SessionReconciler) Reconcile(session) {
    phase := getPhase(session)
    
    if phase == "Stopped" {
        log.Printf("Session stopped by user, cleaning up")
        r.deleteJob()
        r.deletePods()
        r.deleteContentPod()
        // Keep PVC for potential restart
        return ctrl.Result{}, nil // Terminal state
    }
    
    return r.reconcileSession(session)
}
```

**Responsibility:**
- ✅ Backend: Set status to Stopped
- ✅ Operator: Cleanup Job, Pods, Content Pod

---

### 2. Spec Update Actions

#### `PUT /api/projects/:project/agentic-sessions/:session` - Update Session Spec
**Current:** Backend updates spec (prompt, llmSettings, timeout)  
**Migration:** 🟢 **Add validation for running sessions**

**BEFORE:**
```go
// No validation if session is running
func UpdateSession(c *gin.Context) {
    session := getSession(...)
    spec["prompt"] = req.Prompt  // Dangerous if running!
    reqDyn.Update(session)
}
```

**AFTER:**
```go
// Validate phase before allowing spec updates
func UpdateSession(c *gin.Context) {
    session := getSession(...)
    phase := getPhase(session)
    
    // Only allow spec updates for stopped sessions
    if phase == "Running" || phase == "Creating" {
        c.JSON(409, gin.H{
            "error": "Cannot update spec while session is running",
            "suggestion": "Stop the session first, or create a new session",
        })
        return
    }
    
    // OK to update spec when stopped
    spec := session.Object["spec"]
    spec["prompt"] = req.Prompt
    spec["llmSettings"] = req.LLMSettings
    spec["timeout"] = req.Timeout
    
    reqDyn.Update(session)  // Generation increments
    
    c.JSON(200, session)
}

// Operator: Detects generation change while running → stops session
func (r *SessionReconciler) Reconcile(session) {
    currentGen := session.GetGeneration()
    observedGen := getObservedGeneration(session)
    
    if currentGen > observedGen {
        phase := getPhase(session)
        if phase == "Running" {
            log.Printf("Spec changed during execution (gen %d→%d), stopping", observedGen, currentGen)
            r.updateCondition(session, ConditionTypeFailed, "SpecModified")
            r.deleteJob()
            return ctrl.Result{}, nil
        }
    }
    
    // Update observedGeneration after processing
    r.updateStatus(session, map[string]interface{}{
        "observedGeneration": currentGen,
    })
    
    return r.reconcileSession(session)
}
```

**Responsibility:**
- ✅ Backend: Validate phase, update spec if allowed
- ✅ Operator: Detect generation changes, stop if running

---

#### `PATCH /api/projects/:project/agentic-sessions/:session` - Patch Session
**Current:** Backend patches annotations  
**Migration:** 🔵 **No change needed** (annotations are OK to patch anytime)

```go
// Backend: Patch annotations (not spec)
func PatchSession(c *gin.Context) {
    session := getSession(...)
    
    // Apply patch to annotations only
    annotations := session.GetAnnotations()
    for k, v := range patchData["metadata"]["annotations"] {
        annotations[k] = v
    }
    session.SetAnnotations(annotations)
    
    reqDyn.Update(session)
    c.JSON(200, session)
}
```

**Responsibility:**
- ✅ Backend: Annotation updates (runtime metadata)

---

### 3. Runtime Modification Actions

#### `POST /api/projects/:project/agentic-sessions/:session/repos` - Add Repository
**Current:** Backend sends WebSocket message to runner  
**Migration:** 🟢 **Update spec.repos, operator reconciles**

**BEFORE:**
```go
// Imperative: Backend tells runner "do this now"
func AddRepo(c *gin.Context) {
    // Send WebSocket message
    SendMessageToSession(project, sessionName, {
        "type": "repo_added",
        "payload": {"url": url, "branch": branch},
    })
    c.JSON(200, gin.H{"message": "Repo added"})
}
```

**AFTER:**
```go
// Declarative: Backend updates spec, operator reconciles
func AddRepo(c *gin.Context) {
    reqDyn := GetK8sClientsForRequest(c)
    session := reqDyn.Get(...)
    
    // Validate session is running and interactive
    phase := getPhase(session)
    interactive := getInteractive(session)
    if phase != "Running" || !interactive {
        c.JSON(400, gin.H{"error": "Can only add repos to running interactive sessions"})
        return
    }
    
    // Update spec.repos (declare desired state)
    spec := session.Object["spec"]
    repos := spec["repos"].([]interface{})
    
    // Check if repo already exists
    for _, r := range repos {
        if r["url"] == req.URL {
            c.JSON(409, gin.H{"error": "Repo already exists"})
            return
        }
    }
    
    // Add new repo to spec
    repos = append(repos, map[string]interface{}{
        "url":    req.URL,
        "branch": req.Branch,
        "name":   req.Name,
    })
    spec["repos"] = repos
    
    // Update CR (generation increments)
    reqDyn.Update(session)
    
    c.JSON(200, gin.H{"message": "Repo will be cloned", "name": req.Name})
}

// Operator: Reconciles repos
func (r *SessionReconciler) reconcileRepos(session) {
    desiredRepos := getReposFromSpec(session)
    reconciledRepos := getReposFromStatus(session)
    
    // Find repos to clone
    for _, desired := range desiredRepos {
        if !contains(reconciledRepos, desired) {
            log.Printf("Repo %s needs to be cloned", desired.Name)
            
            // Call content service to clone repo
            err := r.callContentService(session, "/repos/clone", map[string]interface{}{
                "url":    desired.URL,
                "branch": desired.Branch,
                "name":   desired.Name,
            })
            
            if err != nil {
                r.updateCondition(session, "ReposReconciled", metav1.ConditionFalse, 
                    "CloneFailed", fmt.Sprintf("Failed to clone %s: %v", desired.Name, err))
                continue
            }
            
            // Update status to reflect repo was cloned
            r.addRepoToStatus(session, desired)
            
            // Request SDK restart to add repo to additional directories
            r.callContentService(session, "/sdk/restart", nil)
        }
    }
}
```

**Responsibility:**
- ✅ Backend: Validate, update spec.repos
- ✅ Operator: Call content service to clone, update status
- ✅ Content Service: Execute `git clone`

---

#### `DELETE /api/projects/:project/agentic-sessions/:session/repos/:repo` - Remove Repository
**Current:** Backend sends WebSocket message to runner  
**Migration:** 🟢 **Remove from spec.repos, operator reconciles**

**AFTER:**
```go
// Backend: Remove from spec
func RemoveRepo(c *gin.Context) {
    reqDyn := GetK8sClientsForRequest(c)
    session := reqDyn.Get(...)
    repoName := c.Param("repoName")
    
    // Validate session is running and interactive
    phase := getPhase(session)
    if phase != "Running" {
        c.JSON(400, gin.H{"error": "Can only remove repos from running sessions"})
        return
    }
    
    // Remove repo from spec.repos
    spec := session.Object["spec"]
    repos := spec["repos"].([]interface{})
    
    newRepos := []interface{}{}
    found := false
    for _, r := range repos {
        if r["name"] != repoName {
            newRepos = append(newRepos, r)
        } else {
            found = true
        }
    }
    
    if !found {
        c.JSON(404, gin.H{"error": "Repo not found"})
        return
    }
    
    spec["repos"] = newRepos
    reqDyn.Update(session)
    
    c.JSON(200, gin.H{"message": "Repo will be removed"})
}

// Operator: Reconciles removed repos
func (r *SessionReconciler) reconcileRepos(session) {
    desiredRepos := getReposFromSpec(session)
    reconciledRepos := getReposFromStatus(session)
    
    // Find repos to remove
    for _, reconciled := range reconciledRepos {
        if !contains(desiredRepos, reconciled) {
            log.Printf("Repo %s should be removed", reconciled.Name)
            
            // Call content service to remove repo directory
            r.callContentService(session, fmt.Sprintf("/repos/%s", reconciled.Name), nil, "DELETE")
            
            // Remove from status
            r.removeRepoFromStatus(session, reconciled)
            
            // Request SDK restart to update additional directories
            r.callContentService(session, "/sdk/restart", nil)
        }
    }
}
```

**Responsibility:**
- ✅ Backend: Validate, remove from spec.repos
- ✅ Operator: Call content service to remove, update status
- ✅ Content Service: Delete directory, restart SDK

---

#### `POST /api/projects/:project/agentic-sessions/:session/workflow` - Switch Workflow
**Current:** Backend sends WebSocket message to runner  
**Migration:** 🟢 **Update spec.activeWorkflow, operator reconciles**

**AFTER:**
```go
// Backend: Update spec.activeWorkflow
func SelectWorkflow(c *gin.Context) {
    reqDyn := GetK8sClientsForRequest(c)
    session := reqDyn.Get(...)
    
    // Validate session is running and interactive
    phase := getPhase(session)
    if phase != "Running" {
        c.JSON(400, gin.H{"error": "Can only switch workflow on running sessions"})
        return
    }
    
    // Update spec.activeWorkflow
    spec := session.Object["spec"]
    spec["activeWorkflow"] = map[string]interface{}{
        "gitUrl": req.GitURL,
        "branch": req.Branch,
        "path":   req.Path,
    }
    
    reqDyn.Update(session)
    
    c.JSON(200, gin.H{"message": "Workflow will be switched"})
}

// Operator: Reconciles workflow
func (r *SessionReconciler) reconcileWorkflow(session) {
    desiredWorkflow := getWorkflowFromSpec(session)
    reconciledWorkflow := getWorkflowFromStatus(session)
    
    if desiredWorkflow != reconciledWorkflow {
        log.Printf("Workflow needs to switch to %s", desiredWorkflow.GitURL)
        
        // Call content service to clone workflow
        err := r.callContentService(session, "/workflows/clone", map[string]interface{}{
            "url":    desiredWorkflow.GitURL,
            "branch": desiredWorkflow.Branch,
            "path":   desiredWorkflow.Path,
            "name":   deriveWorkflowName(desiredWorkflow.GitURL),
        })
        
        if err != nil {
            r.updateCondition(session, "WorkflowReconciled", metav1.ConditionFalse, 
                "CloneFailed", fmt.Sprintf("Failed to clone workflow: %v", err))
            return
        }
        
        // Request SDK restart to switch CWD
        r.callContentService(session, "/sdk/restart", nil)
        
        // Update status
        r.updateWorkflowInStatus(session, desiredWorkflow)
        r.incrementSDKRestartCount(session)
    }
}
```

**Responsibility:**
- ✅ Backend: Validate, update spec.activeWorkflow
- ✅ Operator: Clone workflow, restart SDK, update status
- ✅ Content Service: Clone workflow repo, restart SDK

---

### 4. Status Update Actions

#### `PUT /api/projects/:project/agentic-sessions/:session/status` - Update Status
**Current:** Backend OR runner can update status  
**Migration:** 🔴 **Remove entirely - only operator updates status**

**BEFORE:**
```go
// Backend endpoint allows arbitrary status updates
func UpdateSessionStatus(c *gin.Context) {
    // Anyone with token can update any status field
    DynamicClient.Resource(gvr).Namespace(project).UpdateStatus(statusUpdate)
}
```

**AFTER:**
```go
// REMOVE THIS ENDPOINT ENTIRELY
// Only operator should update status based on observed state
```

**In runner wrapper.py:**
```python
# BEFORE: Runner updates CR status
await self._update_cr_status({
    "phase": "Completed",
    "message": "...",
})

# AFTER: Runner just exits with proper code
sys.exit(0)  # Operator detects exit code and updates status
```

**Responsibility:**
- ❌ Backend: Removed
- ❌ Runner: Removed
- ✅ Operator: Only component that updates status

---

### 5. Workspace Access Actions

#### `GET /api/projects/:project/agentic-sessions/:session/workspace` - List Workspace
**Current:** Backend spawns temp content pod, proxies request  
**Migration:** 🟡 **Call content service directly (if pod exists)**

**BEFORE:**
```go
// Backend spawns temp pod every time
func ListSessionWorkspace(c *gin.Context) {
    // Spawn temp content pod
    SpawnContentPod(...)
    
    // Proxy to content pod
    proxyToContentPod(c, "/list")
}
```

**AFTER:**
```go
// Backend calls content service if session is running
func ListSessionWorkspace(c *gin.Context) {
    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    session := reqDyn.Get(...)
    
    phase := getPhase(session)
    
    if phase == "Running" {
        // Content service is running - call it directly
        svcName := fmt.Sprintf("ambient-content-%s", sessionName)
        url := fmt.Sprintf("http://%s.%s.svc:8080/workspace/list?path=%s", 
                          svcName, project, c.Query("path"))
        
        resp := http.Get(url)
        c.JSON(resp.StatusCode, resp.Body)
        return
    }
    
    // Session stopped - need temp pod for workspace access
    // OR: Return error telling user to start session first
    c.JSON(400, gin.H{
        "error": "Session is not running",
        "suggestion": "Start the session to access workspace",
    })
}
```

**Responsibility:**
- ✅ Backend: Route to content service
- ✅ Content Service: Read filesystem, return listing

---

#### `GET /api/projects/:project/agentic-sessions/:session/workspace/*path` - Get File
**Current:** Backend spawns temp pod, proxies request  
**Migration:** 🟡 **Same as above - call content service directly**

**Same pattern as ListSessionWorkspace**

---

#### `PUT /api/projects/:project/agentic-sessions/:session/workspace/*path` - Write File
**Current:** Backend spawns temp pod, proxies request  
**Migration:** 🟡 **Same as above - call content service directly**

**Same pattern as ListSessionWorkspace**

---

### 6. Git Operations

#### `GET /api/projects/:project/agentic-sessions/:session/git/status` - Git Status
**Current:** Backend spawns temp pod, runs git status  
**Migration:** 🟡 **Call content service directly**

**AFTER:**
```go
// Backend: Proxy to content service
func GetGitStatus(c *gin.Context) {
    session := getSession(...)
    repoName := c.Query("repo")
    
    // Call content service
    svcName := fmt.Sprintf("ambient-content-%s", sessionName)
    url := fmt.Sprintf("http://%s.%s.svc:8080/repos/%s/git/status", 
                      svcName, project, repoName)
    
    resp := http.Get(url)
    c.JSON(resp.StatusCode, resp.Body)
}
```

**Responsibility:**
- ✅ Backend: Proxy request
- ✅ Content Service: Execute `git status`

---

#### `POST /api/projects/:project/agentic-sessions/:session/git/push` - Git Push
**Current:** Backend spawns temp pod, runs git push  
**Migration:** 🟡 **Call content service directly**

**Same pattern as GetGitStatus**

---

#### `POST /api/projects/:project/agentic-sessions/:session/git/pull` - Git Pull
**Current:** Backend spawns temp pod, runs git pull  
**Migration:** 🟡 **Call content service directly**

**Same pattern as GetGitStatus**

---

#### `POST /api/projects/:project/agentic-sessions/:session/git/create-branch` - Create Branch
**Current:** Backend spawns temp pod, runs git checkout -b  
**Migration:** 🟡 **Call content service directly**

**Same pattern as GetGitStatus**

---

#### `GET /api/projects/:project/agentic-sessions/:session/git/list-branches` - List Branches
**Current:** Backend spawns temp pod, runs git branch  
**Migration:** 🟡 **Call content service directly**

**Same pattern as GetGitStatus**

---

### 7. Content Pod Management

#### `POST /api/projects/:project/agentic-sessions/:session/spawn-content-pod` - Spawn Temp Pod
**Current:** Backend creates temporary pod for workspace access  
**Migration:** 🔴 **Remove - content service runs with Job**

**BEFORE:**
```go
// Backend creates temp pod on demand
func SpawnContentPod(c *gin.Context) {
    // Create pod
    // Wait for ready
    // Return service URL
}
```

**AFTER:**
```go
// REMOVE THIS ENDPOINT
// Content service runs as main container in Job pod
// No need for temp pods
```

**Responsibility:**
- ❌ Backend: Removed
- ✅ Operator: Creates Job with content service as main container

---

#### `DELETE /api/projects/:project/agentic-sessions/:session/content-pod` - Delete Temp Pod
**Current:** Backend deletes temporary pod  
**Migration:** 🔴 **Remove - no temp pods**

**REMOVE THIS ENDPOINT**

---

#### `GET /api/projects/:project/agentic-sessions/:session/content-pod-status` - Get Pod Status
**Current:** Backend checks if temp pod is ready  
**Migration:** 🔴 **Remove - no temp pods**

**REMOVE THIS ENDPOINT**

---

### 8. GitHub Integration Actions

#### `POST /api/projects/:project/agentic-sessions/:session/github/token` - Mint GitHub Token
**Current:** Backend mints token from GitHub App or PAT  
**Migration:** 🔵 **No change needed** (backend is correct place)

```go
// Backend: Mints GitHub token (GitHub App or PAT)
func MintSessionGitHubToken(c *gin.Context) {
    // Get user ID from session spec
    userID := getUserFromSession(session)
    
    // Mint token (GitHub App or PAT fallback)
    token := GetGitHubToken(ctx, K8sClient, DynamicClient, project, userID)
    
    c.JSON(200, gin.H{"token": token})
}
```

**Responsibility:**
- ✅ Backend: GitHub App integration, token minting

---

#### `POST /api/projects/:project/agentic-sessions/:session/github/push` - Push to GitHub
**Current:** Backend spawns temp pod, pushes  
**Migration:** 🟡 **Call content service directly**

**Same pattern as git operations**

---

#### `GET /api/projects/:project/agentic-sessions/:session/github/diff` - Get Diff
**Current:** Backend spawns temp pod, gets diff  
**Migration:** 🟡 **Call content service directly**

**Same pattern as git operations**

---

#### `POST /api/projects/:project/agentic-sessions/:session/github/abandon` - Abandon Changes
**Current:** Backend spawns temp pod, runs git reset  
**Migration:** 🟡 **Call content service directly**

**Same pattern as git operations**

---

### 9. Session Cloning

#### `POST /api/projects/:project/agentic-sessions/:session/clone` - Clone Session
**Current:** Backend creates new CR with copied spec  
**Migration:** 🔵 **No change needed**

```go
// Backend: Creates new CR with spec from existing session
func CloneSession(c *gin.Context) {
    sourceSession := getSession(...)
    
    // Create new session with cloned spec
    newSession := map[string]interface{}{
        "apiVersion": "vteam.ambient-code/v1alpha1",
        "kind": "AgenticSession",
        "metadata": map[string]interface{}{
            "name": req.NewSessionName,
            "namespace": req.TargetProject,
        },
        "spec": sourceSession.Object["spec"],  // Clone spec
        "status": map[string]interface{}{
            "phase": "Pending",
        },
    }
    
    reqDyn.Resource(gvr).Namespace(req.TargetProject).Create(newSession)
}
```

**Responsibility:**
- ✅ Backend: Validate, create new CR
- ✅ Operator: Provision resources for new session

---

### 10. K8s Resource Inspection

#### `GET /api/projects/:project/agentic-sessions/:session/k8s-resources` - Get Resources
**Current:** Backend lists Job, Pods, PVC  
**Migration:** 🔵 **No change needed** (read-only, informational)

```go
// Backend: Lists K8s resources (read-only)
func GetSessionK8sResources(c *gin.Context) {
    reqK8s := GetK8sClientsForRequest(c)
    
    // Get Job
    job := reqK8s.BatchV1().Jobs(project).Get(jobName)
    
    // Get Pods
    pods := reqK8s.CoreV1().Pods(project).List(labelSelector)
    
    // Get PVC
    pvc := reqK8s.CoreV1().PersistentVolumeClaims(project).Get(pvcName)
    
    c.JSON(200, gin.H{
        "job": job,
        "pods": pods,
        "pvc": pvc,
    })
}
```

**Responsibility:**
- ✅ Backend: RBAC-enforced read of K8s resources

---

## Summary: Migration Categories

### 🔵 No Change Needed (12 actions)
- Create Session
- Get Session
- Delete Session
- Patch Session (annotations only)
- Clone Session
- Get K8s Resources
- List Sessions
- Mint GitHub Token
- Get Workflow Metadata
- List OOTB Workflows
- Update Session Display Name
- Configure Git Remote

### 🟢 Migrate to Spec Updates (5 actions)
- Update Session Spec (add validation)
- Add Repository (spec.repos)
- Remove Repository (spec.repos)
- Switch Workflow (spec.activeWorkflow)
- Start Session (simplify to status reset)

### 🟡 Migrate to Content Service (10 actions)
- List Workspace → call content service
- Get Workspace File → call content service
- Write Workspace File → call content service
- Git Status → call content service
- Git Push → call content service
- Git Pull → call content service
- Git Create Branch → call content service
- Git List Branches → call content service
- GitHub Push → call content service
- GitHub Diff → call content service

### 🔴 Remove Backend Involvement (4 actions)
- Stop Session → operator handles cleanup
- Update Status → operator only
- Spawn Content Pod → removed (runs with Job)
- Delete Content Pod → removed (no temp pods)

### 🟣 Special: WebSocket Actions (not endpoints)
- Send Chat Message → no change (ephemeral)
- Send Interrupt → no change (ephemeral)
- Get Messages → no change (from backend storage)

---

## Migration Priority

### Phase 1: Low Risk (Week 1)
- ✅ Update Session Spec (add validation)
- ✅ Remove UpdateSessionStatus endpoint
- ✅ Remove temp pod endpoints

### Phase 2: Content Service (Week 2)
- ✅ Add HTTP endpoints to content service
- ✅ Update backend to call content service instead of spawning pods
- ✅ Test all git operations

### Phase 3: Spec-Based Actions (Week 3)
- ✅ Migrate Add/Remove Repo to spec updates
- ✅ Migrate Switch Workflow to spec updates
- ✅ Implement operator reconciliation for repos/workflows
- ✅ Test dynamic repo/workflow changes

### Phase 4: Operator Hardening (Week 4)
- ✅ Migrate Stop Session cleanup to operator
- ✅ Remove runner status updates
- ✅ Implement full condition-based reconciliation
- ✅ Add token refresh logic

### Phase 5: Polish & Testing (Week 5)
- ✅ End-to-end testing of all actions
- ✅ Performance testing (reconciliation frequency)
- ✅ Documentation updates
- ✅ Migration guide for existing sessions

---

## Testing Matrix

| Action | Current Works? | After Migration | Test Case |
|--------|---------------|----------------|-----------|
| Create session | ✅ | ✅ | Session created with Pending phase |
| Start session | ✅ | ✅ | Job created, phase → Running |
| Stop session | ✅ | ✅ | Job deleted, phase → Stopped |
| Add repo (running) | ✅ | ✅ | Repo cloned, SDK restarted |
| Remove repo (running) | ✅ | ✅ | Repo removed, SDK restarted |
| Switch workflow | ✅ | ✅ | Workflow cloned, SDK restarted |
| Edit spec (running) | ❌ Allows | ✅ Rejects | 409 error, user must stop first |
| Git push | ✅ | ✅ | Changes pushed via content service |
| Get workspace file | ✅ | ✅ | File returned via content service |
| Session timeout | ❌ Stuck | ✅ Auto-fails | Condition: Timeout |
| Token expires | ❌ Stuck | ✅ Auto-refreshes | New token minted |
| ImagePullBackOff | ❌ Stuck | ✅ Auto-fails | Condition: ImagePullBackOff |

This is the complete migration plan! Which phase should we start implementing first?

