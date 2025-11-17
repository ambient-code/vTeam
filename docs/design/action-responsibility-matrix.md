# Action Responsibility Matrix: Before vs After

## Complete Action Audit (35 Session Actions)

### Legend
- 🔵 **No Change** - Already correct
- 🟢 **Spec Update** - Migrate to declarative spec updates
- 🟡 **Content Service** - Call content service instead of temp pods
- 🔴 **Remove Backend** - Operator should own this
- 🟣 **WebSocket** - Ephemeral, keep as-is

---

## 1. Session Lifecycle (10 actions)

| # | Action | Endpoint | Current | After | Migration |
|---|--------|----------|---------|-------|-----------|
| 1 | **List Sessions** | `GET /sessions` | Backend reads CR | Backend reads CR | 🔵 No change |
| 2 | **Create Session** | `POST /sessions` | Backend creates CR<br>Operator provisions | Backend creates CR<br>Operator provisions | 🔵 No change |
| 3 | **Get Session** | `GET /sessions/:id` | Backend reads CR | Backend reads CR | 🔵 No change |
| 4 | **Update Session Spec** | `PUT /sessions/:id` | Backend updates spec<br>(no validation) | Backend validates phase<br>Rejects if Running | 🟢 Add validation |
| 5 | **Patch Session** | `PATCH /sessions/:id` | Backend patches annotations | Backend patches annotations | 🔵 No change |
| 6 | **Delete Session** | `DELETE /sessions/:id` | Backend deletes CR<br>K8s GC cleans up | Backend deletes CR<br>K8s GC cleans up | 🔵 No change |
| 7 | **Clone Session** | `POST /sessions/:id/clone` | Backend creates new CR | Backend creates new CR | 🔵 No change |
| 8 | **Start Session** | `POST /sessions/:id/start` | Backend deletes pod<br>Backend updates status<br>Operator creates Job | Backend sets status=Pending<br>Operator cleans up + creates Job | 🟢 Simplify backend |
| 9 | **Stop Session** | `POST /sessions/:id/stop` | Backend deletes Job<br>Backend deletes pods<br>Backend updates status | Backend sets status=Stopped<br>Operator deletes Job/pods | 🔴 Move cleanup to operator |
| 10 | **Update Display Name** | `PUT /sessions/:id/displayname` | Backend updates spec | Backend updates spec | 🔵 No change |

---

## 2. Status & Monitoring (2 actions)

| # | Action | Endpoint | Current | After | Migration |
|---|--------|----------|---------|-------|-----------|
| 11 | **Update Status** | `PUT /sessions/:id/status` | Backend OR runner<br>updates status | **REMOVED**<br>Only operator updates | 🔴 Remove endpoint |
| 12 | **Get K8s Resources** | `GET /sessions/:id/k8s-resources` | Backend lists Job/Pods/PVC | Backend lists Job/Pods/PVC | 🔵 No change |

---

## 3. Runtime Modifications (3 actions)

| # | Action | Endpoint | Current | After | Migration |
|---|--------|----------|---------|-------|-----------|
| 13 | **Add Repository** | `POST /sessions/:id/repos` | Backend sends WebSocket<br>Runner clones repo | Backend updates spec.repos<br>Operator calls content service<br>Content service clones | 🟢 Declarative |
| 14 | **Remove Repository** | `DELETE /sessions/:id/repos/:name` | Backend sends WebSocket<br>Runner removes repo | Backend updates spec.repos<br>Operator calls content service<br>Content service removes | 🟢 Declarative |
| 15 | **Switch Workflow** | `POST /sessions/:id/workflow` | Backend sends WebSocket<br>Runner clones workflow | Backend updates spec.activeWorkflow<br>Operator calls content service<br>Content service clones + restarts SDK | 🟢 Declarative |

---

## 4. Workspace Access (3 actions)

| # | Action | Endpoint | Current | After | Migration |
|---|--------|----------|---------|-------|-----------|
| 16 | **List Workspace** | `GET /sessions/:id/workspace` | Backend spawns temp pod<br>Proxies to temp pod | Backend proxies to content service<br>(running with Job) | 🟡 Direct call |
| 17 | **Get Workspace File** | `GET /sessions/:id/workspace/*path` | Backend spawns temp pod<br>Proxies to temp pod | Backend proxies to content service | 🟡 Direct call |
| 18 | **Put Workspace File** | `PUT /sessions/:id/workspace/*path` | Backend spawns temp pod<br>Proxies to temp pod | Backend proxies to content service | 🟡 Direct call |

---

## 5. Git Operations (8 actions)

| # | Action | Endpoint | Current | After | Migration |
|---|--------|----------|---------|-------|-----------|
| 19 | **Git Status** | `GET /sessions/:id/git/status` | Backend spawns temp pod<br>Runs git status | Backend proxies to content service | 🟡 Direct call |
| 20 | **Git Push** | `POST /sessions/:id/git/push` | Backend spawns temp pod<br>Runs git push | Backend proxies to content service | 🟡 Direct call |
| 21 | **Git Pull** | `POST /sessions/:id/git/pull` | Backend spawns temp pod<br>Runs git pull | Backend proxies to content service | 🟡 Direct call |
| 22 | **Git Create Branch** | `POST /sessions/:id/git/create-branch` | Backend spawns temp pod<br>Runs git checkout -b | Backend proxies to content service | 🟡 Direct call |
| 23 | **Git List Branches** | `GET /sessions/:id/git/list-branches` | Backend spawns temp pod<br>Runs git branch | Backend proxies to content service | 🟡 Direct call |
| 24 | **Git Configure Remote** | `POST /sessions/:id/git/configure-remote` | Backend spawns temp pod<br>Runs git remote add | Backend proxies to content service | 🟡 Direct call |
| 25 | **Git Synchronize** | `POST /sessions/:id/git/synchronize` | Backend spawns temp pod<br>Runs git fetch/reset | Backend proxies to content service | 🟡 Direct call |
| 26 | **Git Merge Status** | `GET /sessions/:id/git/merge-status` | Backend spawns temp pod<br>Checks merge conflicts | Backend proxies to content service | 🟡 Direct call |

---

## 6. GitHub-Specific Operations (3 actions)

| # | Action | Endpoint | Current | After | Migration |
|---|--------|----------|---------|-------|-----------|
| 27 | **Mint GitHub Token** | `POST /sessions/:id/github/token` | Backend mints from App/PAT | Backend mints from App/PAT | 🔵 No change |
| 28 | **GitHub Push** | `POST /sessions/:id/github/push` | Backend spawns temp pod<br>Pushes to GitHub | Backend proxies to content service | 🟡 Direct call |
| 29 | **GitHub Diff** | `GET /sessions/:id/github/diff` | Backend spawns temp pod<br>Gets git diff | Backend proxies to content service | 🟡 Direct call |

---

## 7. Content Pod Management (3 actions - DEPRECATED)

| # | Action | Endpoint | Current | After | Migration |
|---|--------|----------|---------|-------|-----------|
| 30 | **Spawn Content Pod** | `POST /sessions/:id/spawn-content-pod` | Backend creates temp pod | **REMOVED** | 🔴 Delete endpoint |
| 31 | **Get Content Pod Status** | `GET /sessions/:id/content-pod-status` | Backend checks pod status | **REMOVED** | 🔴 Delete endpoint |
| 32 | **Delete Content Pod** | `DELETE /sessions/:id/content-pod` | Backend deletes temp pod | **REMOVED** | 🔴 Delete endpoint |

---

## 8. Workflow Helpers (2 actions)

| # | Action | Endpoint | Current | After | Migration |
|---|--------|----------|---------|-------|-----------|
| 33 | **Get Workflow Metadata** | `GET /sessions/:id/workflow/metadata` | Backend spawns temp pod<br>Reads ambient.json | Backend proxies to content service | 🟡 Direct call |
| 34 | **List OOTB Workflows** | `GET /workflows/ootb` | Backend queries GitHub | Backend queries GitHub | 🔵 No change |

---

## 9. WebSocket Actions (2 actions - Ephemeral)

| # | Action | Endpoint | Current | After | Migration |
|---|--------|----------|---------|-------|-----------|
| 35 | **Send Chat Message** | `POST /sessions/:id/messages` | Backend stores + forwards<br>via WebSocket | Backend stores + forwards<br>via WebSocket | 🟣 Keep as-is |
| 36 | **WebSocket Connect** | `GET /sessions/:id/ws` | Backend maintains WS<br>connection to runner | Backend maintains WS<br>connection to runner | 🟣 Keep as-is |

---

## Operator Actions (Hidden from API)

These are **operator-only** actions that happen during reconciliation:

| # | Action | Trigger | Current | After |
|---|--------|---------|---------|-------|
| 1 | **Create Job** | Phase = Pending | Operator creates Job | ✅ Same |
| 2 | **Monitor Job** | Job exists | `monitorJob()` goroutine | ✅ Reconcile loop (no goroutine) |
| 3 | **Update Status** | Pod state changes | Operator updates status | ✅ Same (using conditions) |
| 4 | **Cleanup Job** | Job completes | Operator deletes Job | ✅ Same |
| 5 | **Set Interactive** | Session completes | Operator updates spec | ✅ Same |
| 6 | **Provision PVC** | New session | Operator creates PVC | ✅ Same |
| 7 | **Copy Secrets** | New session | Operator copies secrets | ✅ Same |
| 8 | **Create Service** | New session | Operator creates Service | ✅ Same |
| 9 | **Refresh Token** | Token > 45min | ❌ Not implemented | ✅ NEW: Auto-refresh |
| 10 | **Reconcile Repos** | spec.repos changes | ❌ Not implemented | ✅ NEW: Clone/remove repos |
| 11 | **Reconcile Workflow** | spec.activeWorkflow changes | ❌ Not implemented | ✅ NEW: Switch workflow |
| 12 | **Handle Timeout** | Job deadline exceeded | ❌ Stuck | ✅ NEW: Auto-fail with condition |
| 13 | **Handle ImagePullBackOff** | Container waiting | ❌ Stuck | ✅ NEW: Auto-fail with condition |

---

## Implementation Roadmap

### Week 1: Foundation (Low Risk)

**Tasks:**
1. Update CRD to add conditions, observedGeneration
2. Remove `UpdateSessionStatus` endpoint
3. Remove temp pod endpoints (spawn/status/delete)
4. Add validation to `UpdateSession` (reject if Running)

**Testing:**
- Create/get/delete sessions still work
- UI doesn't break (reads same fields)

---

### Week 2: Content Service Integration

**Tasks:**
1. Add HTTP endpoints to content service:
   ```python
   POST   /repos/clone
   DELETE /repos/{name}
   POST   /workflows/clone
   POST   /sdk/restart
   GET    /workspace/list
   GET    /workspace/file
   PUT    /workspace/file
   GET    /repos/{name}/git/status
   POST   /repos/{name}/git/push
   POST   /repos/{name}/git/pull
   ```

2. Update backend to call content service:
   ```go
   // Replace all SpawnContentPod() + proxy patterns
   func ListSessionWorkspace(c *gin.Context) {
       url := fmt.Sprintf("http://ambient-content-%s.%s.svc:8080/workspace/list", 
                         sessionName, project)
       resp := http.Get(url)
       c.JSON(resp.StatusCode, resp.Body)
   }
   ```

**Testing:**
- File browsing works
- Git operations work via content service
- No temp pods created

---

### Week 3: Declarative Actions

**Tasks:**
1. Update backend endpoints:
   ```go
   // AddRepo: Update spec instead of WebSocket
   func AddRepo(c *gin.Context) {
       spec["repos"] = append(spec["repos"], newRepo)
       reqDyn.Update(session)  // Operator will reconcile
   }
   
   // SelectWorkflow: Update spec instead of WebSocket
   func SelectWorkflow(c *gin.Context) {
       spec["activeWorkflow"] = newWorkflow
       reqDyn.Update(session)  // Operator will reconcile
   }
   ```

2. Implement operator reconciliation:
   ```go
   func (r *SessionReconciler) reconcileRepos(session) {
       desired := getReposFromSpec(session)
       observed := getReposFromStatus(session)
       
       // Clone missing repos
       for _, repo := range desired {
           if !contains(observed, repo) {
               r.callContentService(session, "/repos/clone", repo)
               r.addRepoToStatus(session, repo)
               r.callContentService(session, "/sdk/restart", nil)
           }
       }
       
       // Remove extra repos
       for _, repo := range observed {
           if !contains(desired, repo) {
               r.callContentService(session, "/repos/"+repo.Name, nil, "DELETE")
               r.removeRepoFromStatus(session, repo)
               r.callContentService(session, "/sdk/restart", nil)
           }
       }
   }
   ```

**Testing:**
- Add repo via UI → spec updates → operator clones → SDK restarts
- Remove repo via UI → spec updates → operator removes → SDK restarts
- Switch workflow → spec updates → operator clones → SDK restarts with new CWD

---

### Week 4: Operator Hardening

**Tasks:**
1. Migrate stop action:
   ```go
   // Backend: Just update status
   func StopSession(c *gin.Context) {
       DynamicClient.UpdateStatus(session, {"phase": "Stopped"})
   }
   
   // Operator: Handle cleanup
   if phase == "Stopped" {
       r.deleteJob()
       r.deletePods()
   }
   ```

2. Remove runner status updates:
   ```python
   # wrapper.py: Remove all _update_cr_status() calls
   # Just exit with proper codes
   sys.exit(0)  # Success
   sys.exit(1)  # Error
   ```

3. Implement condition-based reconciliation:
   ```go
   func (r *SessionReconciler) reconcileSession(session) {
       r.ensurePVC() → update PVCReady condition
       r.ensureSecrets() → update SecretsReady condition
       r.ensureFreshToken() → refresh if > 45min
       r.ensureJob() → update JobCreated condition
       r.monitorPod() → update PodScheduled, RunnerStarted conditions
       r.checkTimeout() → update Failed condition
       r.reconcileRepos() → clone/remove as needed
       r.reconcileWorkflow() → switch if changed
   }
   ```

**Testing:**
- Stop session → operator cleans up
- Token expires → auto-refreshed
- Job times out → auto-failed with Timeout condition
- ImagePullBackOff → auto-failed with ImagePullBackOff condition
- Runner crash → auto-failed with SDKError condition

---

### Week 5: Polish & Documentation

**Tasks:**
1. Update UI to show conditions
2. Add condition timeline view
3. Update documentation
4. Performance tuning (reconciliation frequency)

---

## Detailed Migration for Key Actions

### Action: Add Repository (Runtime)

#### BEFORE (Imperative)

```
┌─────────┐
│ User UI │ clicks "Add Repo"
└────┬────┘
     │
     ▼
┌────────────────────────────────────────┐
│ Backend: AddRepo()                     │
│  1. No CR update                       │
│  2. Sends WebSocket message:           │
│     type: "repo_added"                 │
│     payload: {url, branch, name}       │
└────┬───────────────────────────────────┘
     │ WebSocket
     ▼
┌────────────────────────────────────────┐
│ Runner: wrapper.py                     │
│  1. Receives message                   │
│  2. Clones repo immediately            │
│  3. Updates env var REPOS_JSON         │
│  4. Requests SDK restart               │
└────────────────────────────────────────┘

PROBLEMS:
❌ No CR record of repo being added
❌ If runner crashes, change is lost
❌ Can't audit what repos were added
❌ Operator doesn't know about repo
```

#### AFTER (Declarative)

```
┌─────────┐
│ User UI │ clicks "Add Repo"
└────┬────┘
     │
     ▼
┌─────────────────────────────────────────────────────┐
│ Backend: AddRepo()                                  │
│  1. GET session CR                                  │
│  2. Validate phase = Running, interactive = true    │
│  3. Add repo to spec.repos[]                        │
│  4. UPDATE CR (generation increments)               │
│  5. Return 200 OK                                   │
└─────────────────────────────────────────────────────┘
     │
     │ CR updated
     ▼
┌─────────────────────────────────────────────────────┐
│ AgenticSession CR                                   │
│  metadata:                                          │
│    generation: 5  ◄── Incremented                  │
│  spec:                                              │
│    repos:                                           │
│      - {url: repo1, branch: main}                   │
│      - {url: repo2, branch: dev}  ◄── New!         │
│  status:                                            │
│    observedGeneration: 4  ◄── Out of sync          │
│    reconciledRepos:                                 │
│      - {url: repo1, status: Ready}                  │
│      # repo2 not here yet                           │
└─────────────────────────────────────────────────────┘
     │
     │ Operator watch event
     ▼
┌─────────────────────────────────────────────────────┐
│ Operator: reconcileRepos()                          │
│  1. Compare spec.repos vs status.reconciledRepos    │
│  2. Find repo2 is missing                           │
│  3. Call content service:                           │
│     POST /repos/clone {url: repo2, ...}             │
│  4. Wait for success                                │
│  5. Update status.reconciledRepos                   │
│  6. Call content service:                           │
│     POST /sdk/restart                               │
│  7. Update status.observedGeneration = 5            │
└─────┬───────────────────────────────────────────────┘
      │
      │ HTTP call
      ▼
┌─────────────────────────────────────────────────────┐
│ Content Service (running in pod)                    │
│  POST /repos/clone                                  │
│   1. git clone repo2 to /workspace/repo2            │
│   2. git config user.name/email                     │
│   3. Return 200 OK                                  │
│                                                      │
│  POST /sdk/restart                                  │
│   1. Set flag: /workspace/.sdk-restart-requested    │
│   2. Signal runner via queue                        │
└─────┬───────────────────────────────────────────────┘
      │
      │ signal
      ▼
┌─────────────────────────────────────────────────────┐
│ Runner: wrapper.py                                  │
│  1. Check restart flag in interactive loop          │
│  2. Break from SDK loop                             │
│  3. Re-initialize SDK with updated add_dirs         │
│  4. Continue session                                │
└─────────────────────────────────────────────────────┘

BENEFITS:
✅ CR is source of truth (spec.repos shows all repos)
✅ Crash-safe (if operator crashes, resumes on restart)
✅ Auditable (kubectl get session shows repos)
✅ Idempotent (operator can reconcile multiple times)
✅ Operator owns lifecycle (backend just validates)
```

---

### Action: Start/Restart Session

#### BEFORE

```
┌─────────┐
│ User UI │ clicks "Restart"
└────┬────┘
     │
     ▼
┌──────────────────────────────────────────────┐
│ Backend: StartSession()                      │
│  1. Get session CR                           │
│  2. Call ensureRunnerRolePermissions()       │
│  3. Delete temp-content pod using user token │
│  4. Update spec.interactive = true           │
│  5. Update status.phase = "Pending" (SA)     │
└──────────────────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────────────┐
│ Operator: handleAgenticSessionEvent()       │
│  1. See phase = Pending                      │
│  2. Ensure PVC exists                        │
│  3. Copy secrets                             │
│  4. Create new Job                           │
│  5. Update status.phase = "Creating"         │
│  6. Start monitorJob() goroutine             │
└──────────────────────────────────────────────┘

PROBLEMS:
❌ Backend doing operator work (deleting pods)
❌ Too many responsibilities in backend
❌ Race between backend and operator updates
```

#### AFTER

```
┌─────────┐
│ User UI │ clicks "Restart"
└────┬────┘
     │
     ▼
┌──────────────────────────────────────────────┐
│ Backend: StartSession()                      │
│  1. Validate current phase is terminal       │
│  2. Update status.phase = "Pending" (SA)     │
│  3. Return 200 OK                            │
└──────────────────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────────────┐
│ Operator: Reconcile()                        │
│  phase == "Pending":                         │
│   1. Delete old Job if exists                │
│   2. Delete old pods if exist                │
│   3. Delete temp content pod if exists       │
│   4. Ensure PVC exists                       │
│   5. Ensure fresh token (< 45min old)        │
│   6. Verify secrets exist                    │
│   7. Create new Job                          │
│   8. Update conditions:                      │
│      - PVCReady = True                       │
│      - SecretsReady = True                   │
│      - JobCreated = True                     │
│   9. Update status.phase = "Creating"        │
│  10. Requeue after 5s to monitor             │
└──────────────────────────────────────────────┘

BENEFITS:
✅ Backend is simple (one status update)
✅ Operator owns full lifecycle
✅ Automatic cleanup of old resources
✅ Token refresh built-in
✅ Clear separation of concerns
```

---

### Action: Stop Session

#### BEFORE

```
┌─────────┐
│ User UI │ clicks "Stop"
└────┬────┘
     │
     ▼
┌──────────────────────────────────────────────┐
│ Backend: StopSession()                       │
│  1. Get session CR                           │
│  2. Validate phase != Completed/Failed       │
│  3. Delete Job using user token              │
│  4. Delete pods using user token             │
│  5. Update spec.interactive = true (SA)      │
│  6. Update status.phase = "Stopped" (SA)     │
└──────────────────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────────────┐
│ Operator: monitorJob()                       │
│  1. Sees Job deleted                         │
│  2. Exits monitoring goroutine               │
│  OR                                          │
│  3. Sees pod terminated                      │
│  4. Tries to update status (race!)           │
└──────────────────────────────────────────────┘

PROBLEMS:
❌ Backend doing K8s resource manipulation
❌ Race condition with monitorJob()
❌ Backend needs both user token AND SA token
❌ User needs delete Job/Pods permissions
```

#### AFTER

```
┌─────────┐
│ User UI │ clicks "Stop"
└────┬────┘
     │
     ▼
┌──────────────────────────────────────────────┐
│ Backend: StopSession()                       │
│  1. Validate user has update permission      │
│  2. Update status.phase = "Stopped" (SA)     │
│  3. Return 200 OK                            │
└──────────────────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────────────┐
│ Operator: Reconcile()                        │
│  phase == "Stopped":                         │
│   1. Update condition:                       │
│      Ready = False, reason = UserStopped     │
│   2. Delete Job (foreground propagation)     │
│   3. Delete all pods (by label)              │
│   4. Delete content pod                      │
│   5. Delete ambient-vertex secret            │
│   6. Keep PVC (for potential restart)        │
│   7. Update spec.interactive = true          │
│   8. Return (terminal state, stop reconcile) │
└──────────────────────────────────────────────┘

BENEFITS:
✅ Backend is simple (one status update)
✅ Operator owns cleanup
✅ No race conditions
✅ User only needs session update permission
✅ All cleanup in one place
```

---

## Migration Complexity Analysis

| Migration Category | # Actions | Complexity | Risk | Estimated Time |
|-------------------|-----------|------------|------|----------------|
| 🔵 No Change | 12 | None | None | 0 days |
| 🟢 Spec Updates | 5 | Low | Low | 2-3 days |
| 🟡 Content Service | 10 | Medium | Medium | 5-7 days |
| 🔴 Remove Backend | 4 | Medium | Low | 2-3 days |
| 🟣 WebSocket (Keep) | 2 | None | None | 0 days |
| **Operator New Features** | 13 | High | Medium | 7-10 days |

**Total Estimated Time:** 3-4 weeks (including testing)

---

## Breaking Changes Checklist

### Backend API Changes

**Removed Endpoints:**
- ❌ `PUT /sessions/:id/status` (only operator updates status)
- ❌ `POST /sessions/:id/spawn-content-pod` (no temp pods)
- ❌ `GET /sessions/:id/content-pod-status` (no temp pods)
- ❌ `DELETE /sessions/:id/content-pod` (no temp pods)

**Modified Behavior:**
- ⚠️ `PUT /sessions/:id` - Rejects if phase = Running (409 Conflict)
- ⚠️ `POST /sessions/:id/repos` - Updates spec instead of WebSocket
- ⚠️ `DELETE /sessions/:id/repos/:name` - Updates spec instead of WebSocket
- ⚠️ `POST /sessions/:id/workflow` - Updates spec instead of WebSocket
- ⚠️ `POST /sessions/:id/stop` - Simplified to status update only

**No Change:**
- ✅ All other endpoints work identically

### Frontend Changes Required

```typescript
// BEFORE: Workspace access works on stopped sessions
const files = await listWorkspace(project, session)  // Spawns temp pod

// AFTER: Workspace access requires running session
if (session.status?.phase !== 'Running') {
  return <div>Start session to access workspace</div>
}
const files = await listWorkspace(project, session)  // Calls content service
```

### CRD Changes

**Added Fields:**
```yaml
status:
  observedGeneration: 1  # NEW
  conditions: []         # NEW
  reconciledRepos: []    # NEW
  reconciledWorkflow: {} # NEW
  sdkRestartCount: 0     # NEW
  startTime: "..."       # NEW (was removed in simplification)
  completionTime: "..."  # NEW (was removed in simplification)
```

**Modified Fields:**
```yaml
spec:
  initialPrompt: "..."  # RENAMED from 'prompt'
```

---

## Rollback Strategy

### If Phase 1 Fails (CRD + Validation)
```bash
# Revert CRD
kubectl apply -f old-crd.yaml

# Backend still works (ignores new fields)
```

### If Phase 2 Fails (Content Service)
```bash
# Feature flag to disable content service routing
ENABLE_CONTENT_SERVICE_PROXY=false

# Falls back to temp pod spawning
```

### If Phase 3 Fails (Declarative Actions)
```bash
# Feature flag to use old WebSocket pattern
ENABLE_DECLARATIVE_REPOS=false

# Backend sends WebSocket instead of spec updates
```

### If Phase 4 Fails (Operator Hardening)
```bash
# Feature flag to keep old monitoring
ENABLE_CONDITION_RECONCILIATION=false

# Uses old monitorJob() goroutine pattern
```

---

## Success Metrics

After migration:

✅ **No stuck sessions** - All failure modes auto-detected  
✅ **Token refresh works** - No auth failures after 1 hour  
✅ **Spec is source of truth** - `kubectl get session` shows complete state  
✅ **Audit trail** - Conditions show timeline of what happened  
✅ **Faster workspace access** - No temp pod spawning (0.5s → 50ms)  
✅ **Better security** - Runner has no CR write access  
✅ **Cleaner code** - Backend is simpler, operator owns lifecycle  

This is the complete migration plan! Ready to start implementing?

