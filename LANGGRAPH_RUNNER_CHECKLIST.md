# LangGraph Runner Implementation Checklist

## Quick Reference: What to Reuse vs. Customize

### 100% Reusable (No Changes)

- [x] **Runner Shell Framework** (`components/runners/runner-shell/`)
  - WebSocket transport with authentication
  - Message protocol (system, agent, user messages)
  - Session lifecycle management (start/stop)
  - No modifications needed

- [x] **Backend API Patterns**
  - Status update endpoint: `PUT /api/projects/{project}/agentic-sessions/{session}/status`
  - CR creation mechanism
  - Token authentication via BOT_TOKEN
  - No backend changes required

- [x] **Operator Job Monitoring**
  - Job template structure
  - PVC management and mounting
  - Container status tracking
  - Error detection and cleanup
  - No operator changes required

- [x] **WebSocket Messaging**
  - Message types: SYSTEM_MESSAGE, AGENT_MESSAGE, etc.
  - Protocol structure with seq, timestamp, payload
  - Backend hub broadcasting
  - Adapter can send via `self.shell._send_message()`

- [x] **Token & Secret Management**
  - BOT_TOKEN injection via Kubernetes Secret
  - Authorization header construction
  - Token redaction in logs
  - No changes needed

### Must Customize

1. **LangGraph Adapter** (New file: `wrapper.py`)
   - Create `LangGraphAdapter` class matching interface:
     ```python
     async def initialize(self, context: RunnerContext)
     async def run(self) -> dict
     async def handle_message(self, message: dict)  # Optional
     ```
   - Load and execute LangGraph workflow
   - Map workflow events to message types
   - Stream progress/results via `self.shell._send_message()`
   - Return result dict with status

2. **Environment Variables** (Subset)
   - Required (from base): SESSION_ID, WORKSPACE_PATH, WEBSOCKET_URL, BOT_TOKEN, DEBUG
   - Remove: LLM_MODEL, LLM_TEMPERATURE, LLM_MAX_TOKENS, ANTHROPIC_API_KEY
   - Add as needed: LANGGRAPH_API_KEY, WORKFLOW_ID, WORKFLOW_CONFIG, etc.

3. **Dockerfile**
   - Base: `python:3.11-slim` (same)
   - Dependencies: Replace `anthropic` + `claude-agent-sdk` with `langgraph`
   - Include: `runner-shell` package installation (same pattern)
   - ENV: RUNNER_TYPE, HOME, SHELL, TERM (same)

4. **pyproject.toml**
   - Dependencies: `langgraph`, `aiohttp`, `websockets`, `pydantic`, `pyjwt`
   - Python: `>=3.11`
   - Name: `langgraph-runner`
   - Version: `0.1.0`

---

## Implementation Steps

### Step 1: Create Directory Structure
```bash
mkdir -p components/runners/langgraph-runner
```

### Step 2: Copy Template Files
```bash
# Copy runner-shell (unchanged)
# Already exists at: components/runners/runner-shell/

# Create new runner directory
touch components/runners/langgraph-runner/{wrapper.py,Dockerfile,pyproject.toml,README.md}
```

### Step 3: Implement LangGraphAdapter

**Template** (wrapper.py):
```python
import asyncio
import os
import sys
import logging
from pathlib import Path

sys.path.insert(0, '/app/runner-shell')

from runner_shell.core.shell import RunnerShell
from runner_shell.core.protocol import MessageType, SessionStatus
from runner_shell.core.context import RunnerContext

class LangGraphAdapter:
    def __init__(self):
        self.context = None
        self.shell = None
        
    async def initialize(self, context: RunnerContext):
        """Initialize LangGraph adapter with session context."""
        self.context = context
        logging.info(f"LangGraph adapter initialized for session {context.session_id}")
        # Load workflow, config, etc.
        
    async def run(self):
        """Execute LangGraph workflow and stream results."""
        try:
            # 1. Get workflow definition
            # 2. Initialize LangGraph with config
            # 3. Execute workflow
            # 4. Stream events via self.shell._send_message()
            # 5. Return result
            return {
                "success": True,
                "result": {...},
                "returnCode": 0
            }
        except Exception as e:
            logging.error(f"Workflow execution failed: {e}")
            return {
                "success": False,
                "error": str(e)
            }

async def main():
    """Main entry point for LangGraph runner."""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    session_id = os.getenv('SESSION_ID', 'test-session')
    workspace_path = os.getenv('WORKSPACE_PATH', '/workspace')
    websocket_url = os.getenv('WEBSOCKET_URL', 'ws://backend:8080/session/ws')
    
    Path(workspace_path).mkdir(parents=True, exist_ok=True)
    
    adapter = LangGraphAdapter()
    shell = RunnerShell(
        session_id=session_id,
        workspace_path=workspace_path,
        websocket_url=websocket_url,
        adapter=adapter,
    )
    
    adapter.shell = shell
    
    try:
        await shell.start()
        return 0
    except Exception as e:
        logging.error(f"LangGraph runner failed: {e}")
        return 1

if __name__ == '__main__':
    exit(asyncio.run(main()))
```

### Step 4: Create Dockerfile

```dockerfile
FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy and install runner-shell package
COPY runner-shell /app/runner-shell
RUN cd /app/runner-shell && pip install --no-cache-dir .

# Copy langgraph-runner specific files
COPY langgraph-runner /app/langgraph-runner

# Install runner wrapper as a package (pulls dependencies like langgraph)
RUN pip install --no-cache-dir /app/langgraph-runner \
    && pip install --no-cache-dir aiofiles

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV RUNNER_TYPE=langgraph
ENV HOME=/app
ENV SHELL=/bin/bash
ENV TERM=xterm-256color

# OpenShift compatibility
RUN chmod -R g=u /app && chmod -R g=u /usr/local && chmod g=u /etc/passwd

# Default command
CMD ["python", "/app/langgraph-runner/wrapper.py"]
```

### Step 5: Create pyproject.toml

```toml
[project]
name = "langgraph-runner"
version = "0.1.0"
description = "Runner for LangGraph workflow execution in Ambient Code platform"
readme = "README.md"
requires-python = ">=3.11"
authors = [
  { name = "Ambient Code" }
]
dependencies = [
  "requests>=2.31.0",
  "aiohttp>=3.8.0",
  "pyjwt>=2.8.0",
  "langgraph>=0.0.30",  # Adjust version as needed
]

[tool.uv]
dev-dependencies = []

[build-system]
requires = ["setuptools>=61.0"]
build-backend = "setuptools.build_meta"
```

### Step 6: Create README.md

```markdown
# LangGraph Runner

Executes LangGraph workflows in the Ambient Code platform.

## Configuration

### Environment Variables

Required:
- `SESSION_ID` - Session identifier
- `WORKSPACE_PATH` - Workspace directory path
- `WEBSOCKET_URL` - Backend WebSocket URL
- `BOT_TOKEN` - Kubernetes SA token for authorization

Optional:
- `LANGGRAPH_API_KEY` - API key for LangGraph
- `WORKFLOW_ID` - Workflow definition ID
- `WORKFLOW_CONFIG` - Workflow configuration (JSON)
- `DEBUG` - Debug logging enabled
- `INTERACTIVE` - Interactive mode enabled

### Building

```bash
make build-langgraph-runner
```

### Testing

```bash
python -m pytest tests/
```
```

### Step 7: Update Operator Configuration

**File**: `components/operator/internal/config/config.go`

Add environment variable (if not already present):
```go
const (
    // ... existing constants
    LANGGRAPH_RUNNER_IMAGE = "quay.io/ambient_code/vteam_langgraph_runner:latest"
)
```

Then update the config loading:
```go
type Config struct {
    AmbientCodeRunnerImage string  // Existing Claude runner
    LangGraphRunnerImage   string  // New LangGraph runner
    // ...
}

func LoadConfig() *Config {
    return &Config{
        AmbientCodeRunnerImage: os.Getenv("AMBIENT_CODE_RUNNER_IMAGE"),
        LangGraphRunnerImage:   os.Getenv("LANGGRAPH_RUNNER_IMAGE"),
        // ...
    }
}
```

### Step 8: Update Backend to Support Multiple Runners

**File**: `components/backend/handlers/sessions.go`

In `CreateSession()`, add runner type selection:
```go
// In CreateAgenticSessionRequest type
type CreateAgenticSessionRequest struct {
    Prompt        string `json:"prompt"`
    // ... existing fields
    RunnerType    string `json:"runnerType"`  // "claude" or "langgraph"
    // ... rest of fields
}

// In CreateSession handler
if req.RunnerType == "langgraph" {
    session["spec"].(map[string]interface{})["runnerType"] = "langgraph"
}
```

**File**: `components/operator/internal/handlers/sessions.go`

In `handleAgenticSessionEvent()`, select runner image:
```go
runnerImage := appConfig.AmbientCodeRunnerImage  // Default: Claude

// Check if runnerType specified
if runnerType, ok := spec["runnerType"].(string); ok && runnerType == "langgraph" {
    runnerImage = appConfig.LangGraphRunnerImage
}

// Use runnerImage in container spec
job.Spec.Template.Spec.Containers[1].Image = runnerImage
```

---

## Testing Checklist

- [ ] LangGraphAdapter implements required interface
- [ ] WebSocket connection succeeds with BOT_TOKEN
- [ ] Session messages appear in frontend
- [ ] Workflow execution streaming works
- [ ] CR status updates occur (phase, completionTime, etc.)
- [ ] Error cases handled gracefully
- [ ] Container exits cleanly
- [ ] Job cleanup works (keep PVC)
- [ ] Multi-repo support works (if needed)
- [ ] Interactive mode works (if needed)

---

## Debugging Tips

### View Runner Logs
```bash
kubectl logs -f pod/agentic-session-XXXXX-job-0 -c ambient-code-runner -n project-name
```

### Check CR Status
```bash
kubectl get agenticsessions agentic-session-XXXXX -n project-name -o yaml
```

### Verify WebSocket Connection
```bash
# Check if pod can reach backend WebSocket
kubectl exec -it pod/agentic-session-XXXXX-job-0 -c ambient-code-runner -n project-name -- \
  python -c "import websockets; asyncio.run(websockets.connect('ws://backend-service:8080/...'))"
```

### Check BOT_TOKEN
```bash
# Verify secret exists
kubectl get secret ambient-runner-token-agentic-session-XXXXX -n project-name

# Check token is injected
kubectl exec -it pod/agentic-session-XXXXX-job-0 -c ambient-code-runner -n project-name -- \
  env | grep BOT_TOKEN
```

---

## Key Differences from Claude Runner

| Aspect | Claude | LangGraph |
|--------|--------|-----------|
| **Main SDK** | `claude-agent-sdk` | `langgraph` |
| **Execution** | Streaming SDK responses | Workflow graph execution |
| **Tool Use** | SDK manages tools | Must implement tools |
| **State** | PVC `.claude` directory | Workflow state management |
| **Continuation** | SDK resume capability | Workflow-specific |
| **Results** | result.message from SDK | Workflow final output |

---

## File Locations

**New Files to Create**:
- `/workspace/sessions/.../components/runners/langgraph-runner/wrapper.py`
- `/workspace/sessions/.../components/runners/langgraph-runner/Dockerfile`
- `/workspace/sessions/.../components/runners/langgraph-runner/pyproject.toml`
- `/workspace/sessions/.../components/runners/langgraph-runner/README.md`

**Files to Modify** (Optional, for runner selection):
- `components/backend/handlers/sessions.go` (add runnerType field)
- `components/operator/internal/handlers/sessions.go` (select runner image)
- `components/operator/internal/config/config.go` (add LangGraphRunnerImage)

**No Changes Needed**:
- `components/backend/websocket/` (router-agnostic)
- `components/operator/internal/handlers/sessions.go` (job monitoring generic)
- `components/runners/runner-shell/` (framework reusable)

---

## Deployment

1. Build image:
   ```bash
   make build-langgraph-runner REGISTRY=quay.io/your-username
   ```

2. Push image:
   ```bash
   make push-langgraph-runner REGISTRY=quay.io/your-username
   ```

3. Update operator config:
   ```bash
   export LANGGRAPH_RUNNER_IMAGE="quay.io/your-username/vteam_langgraph_runner:latest"
   make deploy
   ```

4. Test:
   ```bash
   # Create session with runnerType: "langgraph"
   curl -X POST http://backend:8080/api/projects/test/agentic-sessions \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"prompt":"test", "runnerType":"langgraph"}'
   ```

---

## References

- [RUNNER_PATTERN_ANALYSIS.md](./RUNNER_PATTERN_ANALYSIS.md) - Detailed analysis
- [claude-code-runner](./components/runners/claude-code-runner/wrapper.py) - Reference implementation
- [runner-shell](./components/runners/runner-shell/) - Framework documentation
- LangGraph docs: https://langchain-ai.github.io/langgraph/
