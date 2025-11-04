# Existing Runner Pattern Analysis

## Claude Code Runner Structure

### Directory Layout
```
components/runners/
├── runner-shell/                    # Standardized framework for all runners
│   ├── runner_shell/
│   │   ├── __init__.py
│   │   └── core/
│   │       ├── __init__.py
│   │       ├── protocol.py          # Message types and formats
│   │       ├── context.py           # Runner context (session info, env vars)
│   │       ├── transport_ws.py      # WebSocket transport implementation
│   │       └── shell.py             # Main orchestrator
│   ├── pyproject.toml
│   └── README.md
│
└── claude-code-runner/              # Claude-specific implementation
    ├── wrapper.py                   # Main entry point (1468 lines)
    ├── pyproject.toml               # Dependencies: anthropic, claude-agent-sdk
    ├── Dockerfile                   # Container image definition
    └── uv.lock                       # Locked dependencies
```

### Key Components

**1. Runner Shell Framework (Reusable)**
- **Location**: `components/runners/runner-shell/`
- **Purpose**: Standardized foundation for all runner implementations
- **Core Responsibilities**:
  - WebSocket connection management with backend
  - Message protocol (send/receive)
  - Session lifecycle management
  - Error handling and disconnection

**2. Claude Code Runner Adapter**
- **Location**: `components/runners/claude-code-runner/wrapper.py`
- **Purpose**: Bridges Claude Code CLI/SDK with runner-shell framework
- **Core Responsibilities**:
  - Initializes RunnerContext with session/workspace info
  - Executes Claude Code SDK with multi-repo support
  - Manages workspace setup (clone, checkout, git config)
  - Handles result storage and CR status updates
  - Processes WebSocket messages and sends logs/results

---

## Execution Flow

### 1. Session Creation (Backend → Operator)
```
User creates session via API
    ↓
Backend (handlers/sessions.go) creates AgenticSession CR
    - Sets spec: prompt, repos, interactive, timeout, LLM settings
    - Sets environment variables
    - Sets annotations with runner token secret
    ↓
Operator watches for AgenticSession CR creation
```

### 2. Job Spawning (Operator)
**File**: `components/operator/internal/handlers/sessions.go:handleAgenticSessionEvent()`

```
Operator detects AgenticSession with phase="Pending"
    ↓
Creates Kubernetes Job with:
    - InitContainer: Sets up workspace directory structure
    - Two main containers:
        a) ambient-content: Content service (keeps pod alive)
        b) ambient-code-runner: Actual runner execution
    ↓
Mounts PVC at /workspace/sessions/{sessionName}/workspace
    ↓
Sets environment variables on runner container
    ↓
Creates Job → Pod starts running
```

### 3. Runner Pod Initialization
**File**: `components/runners/claude-code-runner/wrapper.py:main()`

```
Runner pod starts
    ↓
main() function executes:
    1. Sets up logging (PYTHONUNBUFFERED)
    2. Creates ClaudeCodeAdapter instance
    3. Creates RunnerShell with:
       - session_id (from SESSION_ID env var)
       - workspace_path (from WORKSPACE_PATH env var)
       - websocket_url (from WEBSOCKET_URL env var)
       - adapter instance
    ↓
Links adapter to shell (adapter.shell = shell)
    ↓
Calls shell.start()
```

### 4. WebSocket Connection & Initialization
**File**: `runner_shell/core/shell.py:start()`

```
shell.start():
    ↓
    1. Calls transport.connect()
       - Reads BOT_TOKEN from env
       - Connects WebSocket with Bearer token header
       - Starts receive loop for incoming messages
    ↓
    2. Sends "session.started" system message
    ↓
    3. Calls adapter.initialize(context)
       - Prepares workspace (clone/reset repos)
       - Validates prerequisite files
       - Logs progress to frontend via WebSocket
    ↓
    4. Calls adapter.run()
       - Executes main logic (Claude SDK)
       - Streams results via WebSocket
    ↓
    5. Sends "session.completed" on success or error
    ↓
    6. Calls stop() → disconnects WebSocket
```

### 5. Workspace Preparation
**File**: `wrapper.py:_prepare_workspace()`

**Multi-repo flow**:
```
For each repo in REPOS_JSON:
    1. Determine if reusing from parent session
    2. Clone or reset to specified branch
    3. Configure git user (GIT_USER_NAME, GIT_USER_EMAIL)
    4. Set up output remote if provided
    5. Add GitHub token to URLs for authentication
```

**Legacy single-repo flow**:
```
If INPUT_REPO_URL set:
    1. Clone to workspace root with INPUT_BRANCH
    2. Configure git user and output remote
    3. Set up OUTPUT_REPO_URL if provided
```

### 6. Claude SDK Execution
**File**: `wrapper.py:_run_claude_agent_sdk()`

```
Initialize ClaudeAgentOptions:
    - cwd: Main repo directory (from MAIN_REPO_NAME or MAIN_REPO_INDEX)
    - additional_dirs: Other repos
    - permission_mode: "acceptEdits"
    - allowed_tools: ["Read", "Write", "Bash", "Glob", "Grep", "Edit", ...MCP tools]
    - mcp_servers: Loaded from .mcp.json if present
    - model, temperature, max_tokens: From LLM_* env vars
    - resume: From parent session's SDK session ID (if continuation)
    ↓
Stream response from ClaudeSDKClient:
    - Capture AssistantMessages (text, tool uses, thinking)
    - Send via shell._send_message() → WebSocket
    - Track turn count
    - Process ToolResultBlocks
    - Capture ResultMessage with summary
    ↓
Handle interactive mode:
    - If INTERACTIVE=true:
        - Send WAITING_FOR_INPUT message
        - Wait for user messages from _incoming_queue
        - Process each message and loop
    - Wait for end_session/terminate signal
```

### 7. Result Storage & CR Status Update
**File**: `wrapper.py:run()` (lines 95-146)

```
After SDK execution completes:
    ↓
    1. Optional auto-push on completion (AUTO_PUSH_ON_COMPLETE)
    ↓
    2. Update CR status with BLOCKING call:
       POST /api/projects/{project}/agentic-sessions/{session}/status
       Payload:
       {
           "phase": "Completed" | "Failed",
           "completionTime": UTC ISO timestamp,
           "message": Summary or error message,
           "is_error": boolean,
           "num_turns": integer,
           "session_id": session UUID,
           "subtype": result type from SDK,
           "result": stdout excerpt (first 10KB)
       }
    ↓
    3. BLOCKING flag ensures update completes before container exits
    4. Best-effort failure handling if update fails
```

### 8. Result Pushing (Optional)
**File**: `wrapper.py:_push_results_if_any()`

**Multi-repo push**:
```
For each repo with changes:
    1. Check git status (--porcelain)
    2. Get OUTPUT_REPO_URL from config
    3. Stage and commit changes: "Session {id}: update"
    4. Checkout branch: OUTPUT_BRANCH or "sessions/{session-id}"
    5. Push to output remote
    6. If CREATE_PR=true:
       - Create PR from fork → upstream
       - Post PR URL to WebSocket
```

### 9. Status Monitoring (Operator)
**File**: `components/operator/internal/handlers/sessions.go:monitorJob()`

```
Operator monitors Job/Pod status in background:
    ↓
    1. Check if session CR still exists
    2. Monitor Job status and Pod conditions
    3. Detect container failures (ImagePullBackOff, CrashLoopBackOff)
    4. Track runner container termination
    ↓
    If runner exits with success (code 0):
        → Update CR: phase=Completed
    If runner exits with failure:
        → Update CR: phase=Failed + error message
    If wrapper already set status:
        → Accept and clean up Job/Pod immediately
    ↓
    Clean up Job and per-job Service (keep PVC for restart)
```

---

## Reusable Patterns

### 1. Job Template Structure

**Location**: `components/operator/internal/handlers/sessions.go:handleAgenticSessionEvent()`

**Template Characteristics**:
- **RestartPolicy**: Never (batch jobs don't restart)
- **BackoffLimit**: 3 (retry up to 3 times)
- **ActiveDeadlineSeconds**: 14400 (4 hour safety timeout)
- **TTLSecondsAfterFinished**: 600 (auto-cleanup after 10 min)

**Container Structure**:
```go
Containers: []corev1.Container{
    {
        Name:            "ambient-content",      // Must stay alive
        Image:           appConfig.ContentServiceImage,
        ImagePullPolicy: appConfig.ImagePullPolicy,
        Env: []corev1.EnvVar{
            {Name: "CONTENT_SERVICE_MODE", Value: "true"},
            {Name: "STATE_BASE_DIR", Value: "/workspace"},
        },
        ReadinessProbe: httpGet /health,
        VolumeMounts: [{Name: "workspace", MountPath: "/workspace"}},
    },
    {
        Name:            "ambient-code-runner",  // Actual work
        Image:           appConfig.AmbientCodeRunnerImage,
        SecurityContext: {AllowPrivilegeEscalation: false, Drop: ALL},
        Env:             [/* See below */],
        VolumeMounts:    [{workspace}, {.claude session state}],
    },
}
```

**PVC Management**:
- Create: `ambient-workspace-{sessionName}`
- OwnerReferences: Point to AgenticSession CR (for auto-cleanup)
- For continuation sessions: Reuse parent's PVC
- MountPath: `/workspace/sessions/{sessionName}/workspace`

**InitContainer**:
- Image: `registry.access.redhat.com/ubi8/ubi-minimal:latest`
- Creates workspace directory structure
- Sets permissions: `chmod 777`

### 2. Environment Variables

**Category 1: Session Identification**
```go
{Name: "SESSION_ID", Value: sessionName},
{Name: "AGENTIC_SESSION_NAME", Value: sessionName},
{Name: "AGENTIC_SESSION_NAMESPACE", Value: sessionNamespace},
{Name: "WORKSPACE_PATH", Value: "/workspace/sessions/{name}/workspace"},
```

**Category 2: WebSocket & API**
```go
{Name: "WEBSOCKET_URL", 
 Value: "ws://backend-service.{namespace}.svc.cluster.local:8080/api/projects/{project}/sessions/{session}/ws"},
{Name: "BACKEND_API_URL", 
 Value: "http://backend-service.{namespace}.svc.cluster.local:8080/api"},
{Name: "BOT_TOKEN", ValueFrom: SecretKeyRef},  // From ambient-runner-token-{sessionName}
```

**Category 3: Execution Configuration**
```go
{Name: "PROMPT", Value: spec.prompt},
{Name: "INTERACTIVE", Value: "true|false"},  // From spec.interactive
{Name: "TIMEOUT", Value: "300"},
{Name: "AUTO_PUSH_ON_COMPLETE", Value: "false"},
{Name: "DEBUG", Value: "true"},
```

**Category 4: LLM Settings**
```go
{Name: "LLM_MODEL", Value: model},           // e.g., "sonnet", "haiku"
{Name: "LLM_TEMPERATURE", Value: "0.70"},
{Name: "LLM_MAX_TOKENS", Value: "4000"},
```

**Category 5: Git Configuration**
```go
{Name: "INPUT_REPO_URL", Value: repo.input.url},
{Name: "INPUT_BRANCH", Value: repo.input.branch},
{Name: "OUTPUT_REPO_URL", Value: repo.output.url},
{Name: "OUTPUT_BRANCH", Value: repo.output.branch},
{Name: "REPOS_JSON", Value: "[{name, input, output}...]"},  // Multi-repo
{Name: "MAIN_REPO_INDEX", Value: "0"},
{Name: "GIT_USER_NAME", Value: "Ambient Code Bot"},
{Name: "GIT_USER_EMAIL", Value: "bot@ambient-code.local"},
{Name: "GITHUB_TOKEN", ValueFrom: SecretKeyRef},  // Optional
```

**Category 6: Optional Features**
```go
{Name: "CREATE_PR", Value: "false"},
{Name: "PARENT_SESSION_ID", Value: parentSessionID},  // For continuation
{Name: "MCP_CONFIG_PATH", Value: "/path/to/.mcp.json"},
{Name: "ANTHROPIC_API_KEY", ValueFrom: SecretKeyRef},
```

**EnvFrom** (import all keys from runner secret):
```go
EnvFrom: []corev1.EnvFromSource{
    {SecretRef: {Name: runnerSecretsName}},
}
```

### 3. Runner Initialization Pattern

**Framework Entry Point**: `runner_shell/core/shell.py:RunnerShell`

```python
class RunnerShell:
    def __init__(
        self,
        session_id: str,
        workspace_path: str,
        websocket_url: str,
        adapter: Any,  # Runner-specific adapter (e.g., ClaudeCodeAdapter)
    ):
        # Initialize components
        self.transport = WebSocketTransport(websocket_url)
        self.context = RunnerContext(
            session_id=session_id,
            workspace_path=workspace_path,
        )
        self.adapter = adapter
        
    async def start(self):
        # 1. Connect WebSocket
        await self.transport.connect()
        
        # 2. Send session.started
        await self._send_message(MessageType.SYSTEM_MESSAGE, "session.started")
        
        # 3. Initialize adapter
        await self.adapter.initialize(self.context)
        
        # 4. Run adapter main loop
        result = await self.adapter.run()
        
        # 5. Send session.completed
        await self._send_message(MessageType.SYSTEM_MESSAGE, "session.completed")
```

**Runner Context**: `runner_shell/core/context.py:RunnerContext`

```python
@dataclass
class RunnerContext:
    session_id: str
    workspace_path: str
    environment: Dict[str, str]  # Merged with os.environ
    metadata: Dict[str, Any]
    
    def get_env(self, key: str, default: Optional[str] = None) -> Optional[str]:
        """Get environment variable with fallback"""
    
    def get_metadata(self, key: str, default: Any = None) -> Any:
        """Get metadata value"""
```

**Adapter Pattern** (required interface):

```python
class RunnerAdapter:
    async def initialize(self, context: RunnerContext):
        """Called by shell to initialize adapter"""
        self.context = context
        # Set up configuration, workspace, etc.
    
    async def run(self):
        """Main execution loop - returns result dict"""
        # Do actual work
        return {"success": True, ...}
    
    async def handle_message(self, message: dict):
        """Optional: handle incoming WebSocket messages"""
```

### 4. WebSocket Messaging Pattern

**Transport**: `runner_shell/core/transport_ws.py:WebSocketTransport`

**Connection Setup**:
```python
async def connect(self):
    token = os.getenv("BOT_TOKEN", "").strip()
    headers = {"Authorization": f"Bearer {token}"} if token else {}
    
    self.websocket = await websockets.connect(
        self.url,
        extra_headers=[(k, v) for k, v in headers.items()],
        ping_interval=None  # Backend sends pings every 30s
    )
    self._recv_task = asyncio.create_task(self._receive_loop())
```

**Message Protocol**: `runner_shell/core/protocol.py`

```python
class MessageType(str, Enum):
    SYSTEM_MESSAGE = "system.message"
    AGENT_MESSAGE = "agent.message"
    USER_MESSAGE = "user.message"
    MESSAGE_PARTIAL = "message.partial"
    AGENT_RUNNING = "agent.running"
    WAITING_FOR_INPUT = "agent.waiting"

class Message(BaseModel):
    seq: int                          # Monotonic sequence
    type: MessageType                 # Message type
    timestamp: str                    # UTC ISO format
    payload: Any                      # Content
    partial: Optional[PartialInfo]    # For fragmented messages
```

**Sending Messages**:
```python
await self.shell._send_message(
    MessageType.AGENT_MESSAGE,
    {"tool": "Bash", "input": {...}, "id": "tool-123"}
)

# Generates: {seq: 1, type: "agent.message", timestamp: "...", payload: {...}}
```

**Receiving Messages**:
```python
async def _receive_loop(self):
    while self.running:
        message = await self.websocket.recv()
        data = json.loads(message)
        if self.receive_handler:
            await self.receive_handler(data)
```

**Backend Broadcast** (WebSocket Hub):
- File: `components/backend/websocket/hub.go` and `handlers.go`
- Routes:
  - POST `/api/projects/{project}/sessions/{sessionId}/ws` (WebSocket upgrade)
  - POST `/api/projects/{project}/sessions/{sessionId}/messages` (Send to session)
  - GET `/api/projects/{project}/sessions/{sessionId}/messages` (Retrieve from S3)
- Hub manages connections: register, unregister, broadcast

### 5. Result Storage & Status Update

**CR Status Endpoint**: `PUT /api/projects/{project}/agentic-sessions/{sessionName}/status`

**Request Body** (from wrapper.py line 106-115):
```json
{
    "phase": "Completed|Failed|Running",
    "completionTime": "2025-01-15T10:30:45.123Z",
    "message": "Summary or error",
    "is_error": false,
    "num_turns": 5,
    "session_id": "agentic-session-1234567890",
    "subtype": "success|error|partial",
    "duration_ms": 45000,
    "result": "First 10KB of output"
}
```

**Backend Handler** (handlers/sessions.go:UpdateSessionStatus):
1. Authenticates with user token
2. Gets current CR
3. Filters allowed fields (predefined whitelist)
4. Merges into status subresource
5. Calls UpdateStatus on CR
6. Returns success to runner

**Status Update Flow**:
```
Runner (wrapper.py)
    ↓ HTTP PUT (blocking)
Backend Handler (UpdateSessionStatus)
    ↓ User token validation
    ↓ Get AgenticSession CR
    ↓ Update status subresource
    ↓ Return 200 OK
    ↓ Runner continues/exits
Operator (monitorJob)
    ↓ Detects status change
    ↓ Cleans up Job/Pod
    ↓ Keeps PVC
Frontend UI
    ↓ Watches CR status via API/WebSocket
    ↓ Updates UI with results
```

### 6. Error Handling Patterns

**Workspace Preparation Errors**:
```python
# If prerequisites missing (wrapper.py line 594-607)
error_message = "❌ spec.md not found. Please run /speckit.specify first"
await self._send_log(error_message)
await self._update_cr_status({
    "phase": "Failed",
    "message": error_msg,
    "is_error": True,
}, blocking=True)
raise RuntimeError(error_msg)
```

**Runner Execution Errors**:
```python
try:
    result = await self._run_claude_agent_sdk(prompt)
except Exception as e:
    logging.error(f"Failed to run Claude Code SDK: {e}")
    await self._update_cr_status({
        "phase": "Failed",
        "message": f"Runner failed: {e}",
        "is_error": True,
    }, blocking=True)
    return {"success": False, "error": str(e)}
```

**Best-Effort Updates**:
- Non-critical status updates: async (`blocking=False`)
- Final status updates (Completed/Failed): blocking (`blocking=True`)
- Errors during update: logged but don't halt execution

### 7. Secret Management

**Token Storage**: Kubernetes Secret `ambient-runner-token-{sessionName}`

**Keys**:
- `k8s-token`: Kubernetes SA token for runner (used as BOT_TOKEN)

**Injection Methods**:
1. **Via ValueFrom** (secure):
```go
{Name: "BOT_TOKEN", ValueFrom: &corev1.EnvVarSource{
    SecretKeyRef: &corev1.SecretKeySelector{
        LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
        Key: "k8s-token",
    }
}}
```

2. **Via EnvFrom** (all keys):
```go
EnvFrom: []corev1.EnvFromSource{
    {SecretRef: {Name: runnerSecretsName}},
}
```

**Usage in Runner**:
```python
bot = (os.getenv('BOT_TOKEN') or '').strip()
req.add_header('Authorization', f'Bearer {bot}')
```

### 8. Session Continuation Pattern

**Enabling Continuation**:
- Create new session with `parentSessionID` in request
- Backend creates CR with PARENT_SESSION_ID env var
- Operator reuses parent's PVC instead of creating new

**Runner Behavior** (wrapper.py):
```python
parent_session_id = self.context.get_env('PARENT_SESSION_ID', '').strip()
reusing_workspace = bool(parent_session_id)

if reusing_workspace:
    # Preserve local changes from previous session
    # Don't fetch, reset, or checkout - keep all changes
    logging.info("Preserving existing workspace state")
    
# SDK handles resumption internally if resume ID provided:
sdk_resume_id = await self._get_sdk_session_id(parent_session_id)
if sdk_resume_id:
    options.resume = sdk_resume_id
    options.fork_session = False
```

**SDK Session Storage**:
- Location: `/app/.claude` (mounted as PVC subpath)
- Contains: Session state for resume functionality
- Captured: From SDK's SystemMessage with subtype='init'
- Stored: In CR annotations as `ambient-code.io/sdk-session-id`

---

## Recommendations for LangGraph Runner

### What to Reuse

1. **Runner-Shell Framework** (100% reuse)
   - WebSocket transport implementation
   - Message protocol and types
   - Context and lifecycle management
   - No modifications needed

2. **Job Template Pattern**
   - PVC mounting strategy
   - Environment variable structure
   - Security contexts and capabilities
   - Token injection via secrets
   - Adapt: Change runner image, remove Claude-specific env vars

3. **Operator Integration**
   - Status update mechanism
   - Job monitoring logic
   - Error state detection
   - Container failure handling
   - No changes to CLAUDE.md requirements

4. **WebSocket Messaging**
   - Message types (AGENT_MESSAGE, SYSTEM_MESSAGE, etc.)
   - Protocol structure
   - Backend hub broadcasting
   - Adapt: Map LangGraph events to message types

5. **Secret & Token Management**
   - Token injection pattern
   - BOT_TOKEN environment variable
   - Authorization header construction
   - No changes needed

### What to Customize

1. **Adapter Implementation** (New)
   - Class: `LangGraphAdapter` (like `ClaudeCodeAdapter`)
   - Initialize: Load LangGraph workflow definition
   - Run: Execute workflow with streaming
   - Handle: Process workflow events/callbacks
   - Return: Results structured for status update

2. **Environment Variables** (Subset)
   - Keep: SESSION_ID, WORKSPACE_PATH, WEBSOCKET_URL, BOT_TOKEN
   - Keep: INTERACTIVE, TIMEOUT, DEBUG
   - Remove: LLM_MODEL, LLM_TEMPERATURE, LLM_MAX_TOKENS (framework-specific)
   - Add: WORKFLOW_ID, WORKFLOW_CONFIG, LANGGRAPH_API_KEY, etc.

3. **Dockerfile**
   - Python version: 3.11+ (same)
   - Base image: python:3.11-slim (same)
   - Dependencies: Replace `anthropic` & `claude-agent-sdk` with `langgraph`
   - Include: runner-shell package (same pattern)

4. **Workspace Setup**
   - Determine if LangGraph needs repo cloning
   - If yes: Reuse _prepare_workspace() pattern
   - If no: Simplify to just directory creation

5. **Result Format**
   - Structure results for status update
   - Map workflow outputs to CR status fields
   - Consider: num_turns, duration, result summary

### File Structure Template

```
components/runners/
├── runner-shell/                    # (unchanged)
│   ├── runner_shell/
│   │   └── core/
│   │       ├── protocol.py
│   │       ├── context.py
│   │       ├── transport_ws.py
│   │       └── shell.py
│   └── pyproject.toml
│
└── langgraph-runner/                # New runner
    ├── wrapper.py                   # LangGraphAdapter + main()
    ├── pyproject.toml               # Dependencies: langgraph, aiohttp, etc.
    ├── Dockerfile                   # Container image
    └── README.md
```

### Backend/Operator: No Changes Needed

The existing backend and operator code is framework-agnostic:
- Job creation works for any runner image
- Environment variables pass through to any runner
- Status update endpoint accepts any CR status
- WebSocket messaging is runner-agnostic
- Operator monitoring is generic (watches pod status)

---

## Code References

### Critical Files

**Backend**:
- `components/backend/handlers/sessions.go:CreateSession()` (line 280) - Session creation
- `components/backend/handlers/sessions.go:UpdateSessionStatus()` (line 1570) - Status updates
- `components/backend/routes.go` (line 67-70) - WebSocket routes
- `components/backend/websocket/handlers.go:HandleSessionWebSocket()` (line 27) - WS handler

**Operator**:
- `components/operator/internal/handlers/sessions.go:handleAgenticSessionEvent()` (line 86) - Event handler
- `components/operator/internal/handlers/sessions.go:createJobForSession()` (line 167) - Job creation
- `components/operator/internal/handlers/sessions.go:monitorJob()` (line 749) - Status monitoring
- `components/operator/internal/handlers/sessions.go:updateAgenticSessionStatus()` (line 988) - Status update

**Runner Shell**:
- `components/runners/runner-shell/runner_shell/core/shell.py` (line 15) - RunnerShell class
- `components/runners/runner-shell/runner_shell/core/transport_ws.py` (line 18) - WebSocketTransport
- `components/runners/runner-shell/runner_shell/core/protocol.py` (line 10) - Message types

**Claude Code Runner**:
- `components/runners/claude-code-runner/wrapper.py:main()` (line 1425) - Entry point
- `components/runners/claude-code-runner/wrapper.py:ClaudeCodeAdapter.initialize()` (line 34) - Initialization
- `components/runners/claude-code-runner/wrapper.py:ClaudeCodeAdapter.run()` (line 43) - Main execution
- `components/runners/claude-code-runner/wrapper.py:_prepare_workspace()` (line 434) - Workspace setup
- `components/runners/claude-code-runner/wrapper.py:_run_claude_agent_sdk()` (line 152) - SDK execution
- `components/runners/claude-code-runner/wrapper.py:_update_cr_status()` (line 954) - Status update
- `components/runners/claude-code-runner/Dockerfile` - Container build

### Line Counts

- Claude wrapper: 1,468 lines
  - SDK execution: ~280 lines
  - Workspace prep: ~130 lines
  - Status updates: ~170 lines
  - Result pushing: ~160 lines
  - WebSocket/CR ops: ~200 lines
  - Utilities: ~528 lines

---

## Key Insights

1. **Framework Separation**: Runner-shell provides 100% of infrastructure; adapter only implements business logic

2. **Workspace Persistence**: PVC-based, supports continuation sessions by reusing parent's workspace

3. **Status Flow**: Blocking final updates (Completed/Failed) ensure CR is updated before container exits

4. **WebSocket Design**: Adapter sends messages via `shell._send_message()` without directly touching WebSocket

5. **Error Handling**: Best-effort approach - failures don't block execution if non-critical

6. **Token Security**: BOT_TOKEN injected via secrets, never logged; used for Authorization headers

7. **Multi-Repo Support**: Configured via REPOS_JSON; runner manages all repos independently

8. **Interactive Mode**: Enabled by INTERACTIVE env var; runner waits for user messages via queue

9. **Session Continuation**: Reuses PVC + workspace state; SDK provides its own resume mechanism

10. **Monitoring**: Operator monitors Job status independently of runner; accepts status updates as they arrive
