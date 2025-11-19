# Session Initialization & Continuation Flow

## The initialPrompt Problem

You identified a key issue: **`prompt` is only used once on first SDK invocation**, but currently lives in `spec.prompt` which implies it's always relevant.

### Solution: Rename to `initialPrompt`

```yaml
spec:
  initialPrompt: "Build a web app"  # Only used on FIRST SDK invocation
  repos: [...]
  activeWorkflow: {...}
  llmSettings: {...}
  timeout: 3600
  interactive: true
```

**Rules:**
1. ✅ Used only on **brand new session** (no parent)
2. ❌ **NOT used** on continuation sessions (parent exists)
3. ❌ **NOT used** after workflow switches (workflow has its own `startupPrompt`)
4. ✅ Stored in CR for auditability ("What was the original request?")

---

## Session Initialization Decision Tree

### Runner Startup Logic

```python
async def run(self):
    """Run the Claude Code CLI session."""
    
    # 1. Determine if this is a continuation
    parent_session_id = self.context.get_env('PARENT_SESSION_ID', '').strip()
    is_continuation = bool(parent_session_id)
    
    # 2. Prepare workspace
    await self._prepare_workspace()  # Clone repos from spec
    
    # 3. Initialize workflow if set
    await self._initialize_workflow_if_set()  # Clone workflow from spec
    
    # 4. Determine what prompt to use
    prompt_to_use = self._determine_initial_prompt(is_continuation)
    
    # 5. Run SDK
    async with ClaudeSDKClient(options=options) as client:
        if prompt_to_use:
            await client.query(prompt_to_use)
        
        if interactive:
            # Enter chat loop
            while True:
                msg = await self._incoming_queue.get()
                # ...

def _determine_initial_prompt(self, is_continuation: bool) -> str | None:
    """Determine what prompt to send on SDK startup."""
    
    # CASE 1: Continuation session
    if is_continuation:
        logging.info("Continuation session - SDK will resume, no initial prompt needed")
        return None  # SDK resume handles context
    
    # CASE 2: Workflow with startupPrompt
    active_workflow_url = os.getenv('ACTIVE_WORKFLOW_GIT_URL', '').strip()
    if active_workflow_url:
        ambient_config = self._load_ambient_config(workflow_dir)
        if ambient_config.get('startupPrompt'):
            logging.info("Using workflow's startupPrompt")
            return ambient_config['startupPrompt']  # e.g., "Hi! I'm SpecKit. What would you like to build?"
    
    # CASE 3: New session with initialPrompt
    initial_prompt = self.context.get_env('INITIAL_PROMPT', '').strip()
    if initial_prompt:
        logging.info("Using spec.initialPrompt for first invocation")
        return initial_prompt  # e.g., "Build a web app"
    
    # CASE 4: No prompt - Claude greets based on system prompt
    logging.info("No initial prompt - Claude will greet based on system prompt")
    return None
```

---

## Workspace Preparation Flow

### Case 1: Brand New Session

```python
async def _prepare_workspace(self):
    parent_session_id = self.context.get_env('PARENT_SESSION_ID', '').strip()
    workspace = Path(self.context.workspace_path)
    
    if parent_session_id:
        # CONTINUATION - skip, handled below
        return
    
    # Brand new session - clone everything fresh
    repos_from_spec = self._get_repos_from_spec()
    
    for repo in repos_from_spec:
        repo_dir = workspace / repo['name']
        
        if repo_dir.exists():
            # Workspace was reused (PVC from deleted session) - reset to clean state
            await self._run_cmd(["git", "fetch", "origin", repo['branch']], cwd=repo_dir)
            await self._run_cmd(["git", "reset", "--hard", f"origin/{repo['branch']}"], cwd=repo_dir)
            logging.info(f"Reset {repo['name']} to clean state")
        else:
            # Fresh clone
            await self._run_cmd(["git", "clone", "--branch", repo['branch'], 
                               repo['url'], str(repo_dir)])
            logging.info(f"Cloned {repo['name']}")

def _get_repos_from_spec(self) -> list[dict]:
    """Get repos from spec (REPOS_JSON env var set by operator from spec.repos)."""
    repos_json = os.getenv('REPOS_JSON', '').strip()
    if not repos_json:
        return []
    return json.loads(repos_json)
```

### Case 2: Continuation Session

```python
async def _prepare_workspace(self):
    parent_session_id = self.context.get_env('PARENT_SESSION_ID', '').strip()
    
    if not parent_session_id:
        # Brand new - handled above
        return
    
    # CONTINUATION - preserve everything, only sync spec changes
    logging.info(f"Continuation from {parent_session_id} - preserving workspace state")
    
    repos_from_spec = self._get_repos_from_spec()
    workspace = Path(self.context.workspace_path)
    
    for repo in repos_from_spec:
        repo_dir = workspace / repo['name']
        
        if repo_dir.exists():
            # Repo exists from parent session
            # PRESERVE all local changes (commits, branches, uncommitted work)
            logging.info(f"Preserving {repo['name']} from parent session")
            
            # Only update remote URL in case credentials changed
            await self._run_cmd(["git", "remote", "set-url", "origin", repo['url']], 
                               cwd=repo_dir, ignore_errors=True)
        else:
            # Repo was added to spec since parent session - clone it
            logging.info(f"Repo {repo['name']} added to spec, cloning")
            await self._run_cmd(["git", "clone", "--branch", repo['branch'], 
                               repo['url'], str(repo_dir)])
```

### Case 3: Runtime Repo Addition (Operator-Driven)

```python
# Content service receives HTTP call from operator
@app.post("/repos/clone")
async def clone_repo(req: CloneRepoRequest):
    workspace = Path(os.getenv("WORKSPACE_PATH"))
    repo_dir = workspace / req.name
    
    if repo_dir.exists():
        return {"status": "already_exists"}
    
    # Clone the repo
    await git_clone(req.url, req.branch, repo_dir)
    
    # Signal runner that repo was added (write to signal file)
    signal_file = Path("/workspace/.signals/repo_added.json")
    signal_file.write_text(json.dumps({
        "name": req.name,
        "url": req.url,
        "timestamp": datetime.utcnow().isoformat()
    }))
    
    return {"status": "cloned", "path": str(repo_dir)}

# Runner checks for signals periodically
async def _check_for_operator_signals(self):
    """Check for signals from operator via content service."""
    signals_dir = Path("/workspace/.signals")
    if not signals_dir.exists():
        return None
    
    for signal_file in signals_dir.glob("*.json"):
        try:
            signal = json.loads(signal_file.read_text())
            signal_type = signal_file.stem.split('_')[0]  # "repo", "workflow", etc.
            
            # Delete signal file
            signal_file.unlink()
            
            return (signal_type, signal)
        except Exception as e:
            logging.error(f"Failed to process signal: {e}")
    
    return None

# In interactive loop
while True:
    # Check for operator signals
    signal = await self._check_for_operator_signals()
    if signal:
        signal_type, payload = signal
        
        if signal_type == "repo":
            # Operator cloned a new repo - restart SDK to add it to add_dirs
            logging.info(f"Operator added repo {payload['name']}, restarting SDK")
            self._restart_requested = True
            break
        elif signal_type == "workflow":
            # Operator switched workflow - restart SDK with new CWD
            logging.info(f"Operator switched workflow, restarting SDK")
            self._restart_requested = True
            break
    
    # ... rest of interactive loop
```

---

## Complete Flow: User Adds Repo Mid-Session

### Step-by-Step Sequence

```
T=0s: User clicks "Add Repository" in UI
      └─> POST /api/projects/myproject/agentic-sessions/session-123/repos
          Body: {"url": "https://github.com/org/repo2", "branch": "main", "name": "repo2"}

T=0.1s: Backend validates and updates CR
        ┌─────────────────────────────────────┐
        │ Backend: AddRepo()                  │
        │  1. Get session CR                  │
        │  2. Check phase = Running ✓         │
        │  3. Check interactive = true ✓      │
        │  4. Add to spec.repos[]             │
        │  5. UPDATE CR                       │
        │  6. Return 200 OK                   │
        └─────────────────────────────────────┘

T=0.2s: CR updated, generation increments
        ┌─────────────────────────────────────┐
        │ AgenticSession CR                   │
        │  metadata:                          │
        │    generation: 5 (was 4)            │
        │  spec:                              │
        │    repos:                           │
        │      - {url: repo1, name: repo1}    │
        │      - {url: repo2, name: repo2} ← NEW │
        │  status:                            │
        │    observedGeneration: 4 (stale)    │
        │    reconciledRepos:                 │
        │      - {url: repo1, status: Ready}  │
        └─────────────────────────────────────┘

T=0.3s: Operator receives watch event
        ┌─────────────────────────────────────┐
        │ Operator: Reconcile()               │
        │  1. currentGen (5) > observedGen (4)│
        │  2. Compare spec vs status repos    │
        │  3. repo2 missing from status       │
        │  4. POST to content service:        │
        │     /repos/clone                    │
        └─────────────────────────────────────┘

T=1.0s: Content service clones repo
        ┌─────────────────────────────────────┐
        │ Content Service                     │
        │  1. git clone repo2 → /workspace/repo2 │
        │  2. git config user.name/email      │
        │  3. Write signal file:              │
        │     /workspace/.signals/repo_added.json │
        │  4. Return 200 OK                   │
        └─────────────────────────────────────┘

T=1.1s: Operator updates status
        ┌─────────────────────────────────────┐
        │ Operator                            │
        │  1. Update status.reconciledRepos   │
        │     add repo2 with status: Ready    │
        │  2. Update observedGeneration = 5   │
        │  3. POST to content service:        │
        │     /sdk/restart                    │
        └─────────────────────────────────────┘

T=1.5s: Runner detects signal
        ┌─────────────────────────────────────┐
        │ Runner: wrapper.py                  │
        │  1. Check signals directory         │
        │  2. Read repo_added.json            │
        │  3. Set _restart_requested = True   │
        │  4. Break from SDK interactive loop │
        └─────────────────────────────────────┘

T=2.0s: SDK restarts with new config
        ┌─────────────────────────────────────┐
        │ Runner: _run_claude_agent_sdk()     │
        │  1. Read updated REPOS_JSON         │
        │  2. Build add_dirs list             │
        │     - /workspace/repo1              │
        │     - /workspace/repo2 ← NEW        │
        │  3. Create new SDK client           │
        │     with updated add_dirs           │
        │  4. SDK can now access repo2        │
        │  5. Send log: "✅ Repo added"        │
        └─────────────────────────────────────┘

T=2.5s: User sees confirmation
        UI shows: "Repository repo2 added and available to Claude"
```

**Timeline:** ~2.5 seconds from click to SDK restart (fast!)

---

## initialPrompt vs Chat Messages

### Data Model

```go
type AgenticSessionSpec struct {
    // Used ONCE on first SDK invocation for brand new session
    InitialPrompt string `json:"initialPrompt,omitempty"`
    
    // Repos to clone into workspace
    Repos []SimpleRepo `json:"repos,omitempty"`
    
    // Workflow to activate
    ActiveWorkflow *WorkflowSelection `json:"activeWorkflow,omitempty"`
    
    // LLM configuration
    LLMSettings LLMSettings `json:"llmSettings"`
    
    // Session behavior
    Interactive bool   `json:"interactive,omitempty"`
    Timeout     int    `json:"timeout"`
}
```

**Chat messages** are NOT in the CR - they're stored in:
- Option 1: Backend database/storage
- Option 2: PVC files (current approach via WebSocket streaming)
- Option 3: Not stored at all (ephemeral)

---

## Complete Session Lifecycle

### Scenario 1: New Session → Run → Complete

```
T=0: User creates session
     POST /sessions
     Body: {
       initialPrompt: "Build a TODO app",
       repos: [{url: "repo1", branch: "main"}],
       activeWorkflow: null,
       interactive: false
     }

T=1: Operator reconciles
     - Creates PVC
     - Creates Job with env:
       INITIAL_PROMPT="Build a TODO app"
       REPOS_JSON='[{"url":"repo1","branch":"main"}]'

T=2: Runner starts
     - Clones repo1
     - No workflow, no parent → use initialPrompt
     - await client.query("Build a TODO app")
     - SDK executes prompt
     - SDK finishes
     - Exit code 0

T=120: Operator sees exit code 0
       - Update condition: Completed=True
       - Update status.phase = "Completed"
       - Delete Job
       - Set spec.interactive = true (allow restart)
```

### Scenario 2: Restart Completed Session

```
T=0: User clicks "Restart"
     POST /sessions/session-123/start

T=1: Backend updates status
     - status.phase = "Pending"

T=2: Operator reconciles
     - Detects parent_session_id annotation
     - Reuses PVC (has repo1 with changes)
     - Creates Job with env:
       PARENT_SESSION_ID="session-123"
       REPOS_JSON='[{"url":"repo1","branch":"main"}]'

T=3: Runner starts
     - Sees parent_session_id → is_continuation = true
     - Preserves repo1 (don't reset)
     - Fetches SDK session ID from parent CR
     - await client with options.resume = sdk_session_id
     - SDK resumes with full context
     - NO initialPrompt sent (SDK has full history)
     - Enters interactive mode

T=5: User sends chat message
     "Add authentication"
     - Goes via WebSocket
     - Runner forwards to SDK
     - NOT stored in CR
```

### Scenario 3: Switch Workflow Mid-Session

```
T=0: User selects workflow "SpecKit"
     POST /sessions/session-123/workflow
     Body: {
       gitUrl: "https://github.com/ambient-code/workflow-speckit",
       branch: "main"
     }

T=1: Backend updates spec
     - spec.activeWorkflow = {...}
     - CR generation: 4 → 5

T=2: Operator reconciles
     - Compares spec.activeWorkflow vs status.reconciledWorkflow
     - Different! Need to switch
     - POST /workflows/clone to content service

T=3: Content service clones workflow
     - git clone to /workspace/workflows/workflow-speckit
     - Write signal: /workspace/.signals/workflow_added.json
     - Return 200 OK

T=4: Operator updates status
     - status.reconciledWorkflow = {...}
     - status.observedGeneration = 5
     - POST /sdk/restart to content service

T=5: Content service signals runner
     - Write flag: /workspace/.sdk-restart-requested

T=6: Runner detects signal
     - Check signal in interactive loop
     - Break from SDK loop
     - _restart_requested = True

T=7: SDK restarts with new CWD
     - options.cwd = "/workspace/workflows/workflow-speckit"
     - options.add_dirs = ["/workspace/repo1", "/workspace/artifacts"]
     - SDK reinitializes
     - Send workflow's startupPrompt:
       "Hi! I'm SpecKit. What would you like to build?"
     - User sees workflow greeting
```

---

## Environment Variable Contract

### Operator → Runner (via Job env vars)

```bash
# Session identity
SESSION_ID=session-123
AGENTIC_SESSION_NAME=session-123
AGENTIC_SESSION_NAMESPACE=myproject

# Paths
WORKSPACE_PATH=/workspace/sessions/session-123/workspace
ARTIFACTS_DIR=_artifacts

# Initial configuration (from spec)
INITIAL_PROMPT="Build a web app"
REPOS_JSON='[{"url":"repo1","branch":"main","name":"repo1"}]'
ACTIVE_WORKFLOW_GIT_URL="https://github.com/org/workflow"
ACTIVE_WORKFLOW_BRANCH="main"
ACTIVE_WORKFLOW_PATH=""

# LLM settings (from spec)
LLM_MODEL="sonnet"
LLM_TEMPERATURE="0.7"
LLM_MAX_TOKENS="4000"

# Session behavior (from spec)
INTERACTIVE="true"
TIMEOUT="3600"
AUTO_PUSH_ON_COMPLETE="false"

# Continuation (if restarting)
PARENT_SESSION_ID="session-122"  # If continuing

# Authentication
BOT_TOKEN="<k8s-sa-token>"  # For backend API calls
ANTHROPIC_API_KEY="<key>"  # Or Vertex creds

# Backend URLs
BACKEND_API_URL="http://backend-service.ambient-code.svc:8080/api"
WEBSOCKET_URL="ws://backend-service.ambient-code.svc:8080/api/projects/myproject/sessions/session-123/ws"
```

---

## Operator Reconciliation: What to Check

```go
func (r *SessionReconciler) reconcileSession(ctx, session) (ctrl.Result, error) {
    // ================================================
    // 1. Check generation (did spec change?)
    // ================================================
    currentGen := session.GetGeneration()
    observedGen := getObservedGeneration(session)
    
    if currentGen > observedGen {
        log.Printf("Spec changed: gen %d→%d", observedGen, currentGen)
        // Proceed with reconciliation of new desired state
    }
    
    // ================================================
    // 2. Reconcile repos (spec vs status)
    // ================================================
    desiredRepos := getReposFromSpec(session)      // From spec.repos
    reconciledRepos := getReposFromStatus(session) // From status.reconciledRepos
    
    // Clone missing repos
    for _, repo := range desiredRepos {
        if !containsRepo(reconciledRepos, repo) {
            r.callContentService(session, "/repos/clone", repo)
            r.addRepoToStatus(session, repo)
        }
    }
    
    // Remove extra repos
    for _, repo := range reconciledRepos {
        if !containsRepo(desiredRepos, repo) {
            r.callContentService(session, fmt.Sprintf("/repos/%s", repo.Name), nil, "DELETE")
            r.removeRepoFromStatus(session, repo)
        }
    }
    
    // ================================================
    // 3. Reconcile workflow (spec vs status)
    // ================================================
    desiredWorkflow := getWorkflowFromSpec(session)      // From spec.activeWorkflow
    reconciledWorkflow := getWorkflowFromStatus(session) // From status.reconciledWorkflow
    
    if desiredWorkflow != reconciledWorkflow {
        r.callContentService(session, "/workflows/clone", desiredWorkflow)
        r.updateWorkflowInStatus(session, desiredWorkflow)
        r.callContentService(session, "/sdk/restart", nil)
        r.incrementSDKRestartCount(session)
    }
    
    // ================================================
    // 4. Reconcile infrastructure (PVC, secrets, token)
    // ================================================
    r.ensurePVC() → update PVCReady condition
    r.ensureSecrets() → update SecretsReady condition
    r.ensureFreshToken() → refresh if > 45min
    
    // ================================================
    // 5. Reconcile Job (create if missing, monitor if exists)
    // ================================================
    job := r.getJob()
    if job == nil {
        r.createJob() → update JobCreated condition
    } else {
        r.monitorJob(job) → update PodScheduled, RunnerStarted conditions
        r.checkTimeout(job) → update Failed condition if timeout
    }
    
    // ================================================
    // 6. Update observedGeneration
    // ================================================
    r.updateStatus(session, map[string]interface{}{
        "observedGeneration": currentGen,
    })
    
    return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}
```

---

## Status Structure After Migration

```yaml
status:
  # High-level summary (derived from conditions)
  phase: Running
  
  # Reconciliation tracking
  observedGeneration: 5
  
  # Timestamps
  startTime: "2025-11-15T12:00:00Z"
  completionTime: null
  
  # Infrastructure references
  jobName: session-123-job
  runnerPodName: session-123-job-abc123
  pvcName: ambient-workspace-session-123
  
  # What repos are actually cloned
  reconciledRepos:
    - url: "https://github.com/org/repo1"
      branch: "main"
      name: "repo1"
      clonedAt: "2025-11-15T12:00:05Z"
      status: Ready
    - url: "https://github.com/org/repo2"
      branch: "main"
      name: "repo2"
      clonedAt: "2025-11-15T12:30:00Z"
      status: Ready
  
  # What workflow is actually active
  reconciledWorkflow:
    gitUrl: "https://github.com/org/workflow-speckit"
    branch: "main"
    appliedAt: "2025-11-15T12:00:10Z"
    status: Active
  
  # SDK details
  sdkSessionId: "abc-def-123-456"  # UUID from SDK
  sdkRestartCount: 2  # How many times SDK was restarted
  
  # Detailed status (Kubernetes standard)
  conditions:
    - type: PVCReady
      status: "True"
      reason: Bound
      message: "PVC ambient-workspace-session-123 is bound"
      lastTransitionTime: "2025-11-15T12:00:01Z"
      observedGeneration: 5
    
    - type: SecretsReady
      status: "True"
      reason: AllSecretsFound
      message: "All required secrets are present"
      lastTransitionTime: "2025-11-15T12:00:02Z"
      observedGeneration: 5
    
    - type: JobCreated
      status: "True"
      reason: Created
      message: "Job session-123-job created"
      lastTransitionTime: "2025-11-15T12:00:03Z"
      observedGeneration: 5
    
    - type: PodScheduled
      status: "True"
      reason: Scheduled
      message: "Pod scheduled on node worker-1"
      lastTransitionTime: "2025-11-15T12:00:05Z"
      observedGeneration: 5
    
    - type: RunnerStarted
      status: "True"
      reason: ContainerRunning
      message: "Runner container is active"
      lastTransitionTime: "2025-11-15T12:00:15Z"
      observedGeneration: 5
    
    - type: ReposReconciled
      status: "True"
      reason: AllReposReady
      message: "All 2 repos cloned successfully"
      lastTransitionTime: "2025-11-15T12:30:05Z"
      observedGeneration: 5
    
    - type: WorkflowReconciled
      status: "True"
      reason: WorkflowActive
      message: "Workflow workflow-speckit is active"
      lastTransitionTime: "2025-11-15T12:00:10Z"
      observedGeneration: 5
    
    - type: Ready
      status: "True"
      reason: SessionRunning
      message: "Session is running normally"
      lastTransitionTime: "2025-11-15T12:00:15Z"
      observedGeneration: 5
```

This gives you **complete observability** - you can see:
- ✅ Which spec version was processed (observedGeneration)
- ✅ What repos are cloned and when
- ✅ What workflow is active
- ✅ When each condition became true
- ✅ Current phase at a glance
- ✅ Complete timeline of session lifecycle

Ready to implement! Which phase should we start with?

