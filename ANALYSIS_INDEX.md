# Runner Analysis Documents

This directory contains comprehensive analysis of the existing Claude Code Runner implementation and recommendations for building a LangGraph runner.

## Documents

### 1. RUNNER_PATTERN_ANALYSIS.md (26 KB)
**Comprehensive technical analysis of the Claude Code Runner**

Content:
- Directory structure and component breakdown
- Complete 9-step execution flow with diagrams
- 8 reusable pattern sections with code examples:
  1. Job Template Structure
  2. Environment Variables (6 categories)
  3. Runner Initialization Pattern
  4. WebSocket Messaging Pattern
  5. Result Storage & Status Update
  6. Error Handling Patterns
  7. Secret Management
  8. Session Continuation Pattern
- Key insights and design decisions
- Code references with line numbers and locations

Use this document to understand:
- How the platform orchestrates runner execution
- Detailed flow from session creation to completion
- Patterns that can be reused without modification
- Critical implementation details and edge cases

### 2. LANGGRAPH_RUNNER_CHECKLIST.md (13 KB)
**Step-by-step implementation guide for LangGraph runner**

Content:
- Quick reference table: What to reuse vs. customize
- 8 implementation steps with templates:
  1. Create directory structure
  2. Copy template files
  3. Implement LangGraphAdapter (with skeleton code)
  4. Create Dockerfile (complete template)
  5. Create pyproject.toml (dependencies)
  6. Create README.md (documentation)
  7. Optional: Update operator configuration
  8. Optional: Update backend for runner selection
- Testing checklist (10 items)
- Debugging tips and commands
- Deployment steps
- Key differences table (Claude vs. LangGraph)

Use this document to:
- Quickly identify what needs to be built from scratch
- Find ready-to-use templates for new files
- Understand optional enhancements
- Debug common issues

## Quick Summary

### What's Fully Reusable (100%)
- **Runner Shell Framework** - WebSocket transport, message protocol, lifecycle
- **Kubernetes Job Template** - PVC mounting, env vars, security, monitoring
- **Backend Status API** - CR status updates, filtering, authentication
- **Token Management** - BOT_TOKEN injection, authorization, redaction

### What Must Be Customized
- **LangGraphAdapter** - New Python class implementing the adapter interface
- **Environment Variables** - Subset of existing + new LangGraph-specific ones
- **Dependencies** - Replace anthropic/claude-agent-sdk with langgraph
- **Optional: Backend/Operator** - Add runner type selection (not required)

### Key Realization
Backend and Operator code is **completely framework-agnostic**:
- Any runner image works in the job template
- Status update endpoint accepts any CR status
- WebSocket messaging doesn't depend on runner type
- Job monitoring is generic

This means:
1. LangGraph runner needs NO changes to backend or operator
2. Optional: Add runner type field to support multiple runners in one cluster
3. Focus implementation effort on LangGraphAdapter only

## File Locations

### New Files to Create
```
components/runners/langgraph-runner/
  ├── wrapper.py          # LangGraphAdapter implementation
  ├── Dockerfile          # Container image
  ├── pyproject.toml      # Python dependencies
  └── README.md           # Documentation
```

### Optional Backend Changes
```
components/backend/
  └── handlers/sessions.go         # Add runnerType field

components/operator/
  ├── internal/handlers/sessions.go  # Select runner image
  └── internal/config/config.go      # Add LangGraphRunnerImage
```

### No Changes Needed
```
components/backend/websocket/       # Routing is generic
components/operator/...             # Job monitoring is generic
components/runners/runner-shell/    # Framework reusable as-is
```

## Implementation Roadmap

### Phase 1: Core Implementation (Week 1)
1. Create directory structure
2. Implement LangGraphAdapter
3. Create Dockerfile and pyproject.toml
4. Test WebSocket connection
5. Test basic workflow execution

### Phase 2: Features (Week 2)
1. Add result streaming to frontend
2. Handle interactive mode (if needed)
3. Add error handling and logging
4. Test multi-repo support (if needed)

### Phase 3: Integration (Week 3)
1. Optional: Add runner type selection to backend
2. Optional: Update operator config
3. Build and push to registry
4. Integration testing
5. Documentation

### Phase 4: Enhancement (Week 4)
1. Performance optimization
2. Advanced features (continuation sessions, etc.)
3. Monitoring and metrics
4. Production hardening

## Quick Start

1. **Understand the existing patterns:**
   ```bash
   # Read the comprehensive analysis
   less RUNNER_PATTERN_ANALYSIS.md
   ```

2. **Follow the implementation guide:**
   ```bash
   # Review step-by-step instructions
   less LANGGRAPH_RUNNER_CHECKLIST.md
   ```

3. **Create the new runner:**
   ```bash
   mkdir -p components/runners/langgraph-runner
   # Copy templates from LANGGRAPH_RUNNER_CHECKLIST.md
   ```

4. **Test locally:**
   ```bash
   cd components/runners/langgraph-runner
   docker build -t langgraph-runner .
   docker run --rm -e SESSION_ID=test langgraph-runner
   ```

## Key Code References

### Backend
- Session creation: `components/backend/handlers/sessions.go:280`
- Status updates: `components/backend/handlers/sessions.go:1570`
- WebSocket routes: `components/backend/routes.go:67`

### Operator
- Event handling: `components/operator/internal/handlers/sessions.go:86`
- Job creation: `components/operator/internal/handlers/sessions.go:207`
- Status monitoring: `components/operator/internal/handlers/sessions.go:749`

### Runner Shell
- Shell orchestrator: `components/runners/runner-shell/runner_shell/core/shell.py:15`
- WebSocket transport: `components/runners/runner-shell/runner_shell/core/transport_ws.py:18`
- Message protocol: `components/runners/runner-shell/runner_shell/core/protocol.py:10`

### Claude Runner (Reference)
- Entry point: `components/runners/claude-code-runner/wrapper.py:1425`
- SDK execution: `components/runners/claude-code-runner/wrapper.py:152`
- Workspace prep: `components/runners/claude-code-runner/wrapper.py:434`
- Status updates: `components/runners/claude-code-runner/wrapper.py:954`

## Environment Variables

### Required (Always)
- `SESSION_ID` - Session identifier
- `WORKSPACE_PATH` - Working directory
- `WEBSOCKET_URL` - Backend connection
- `BOT_TOKEN` - Kubernetes token

### Execution Config
- `DEBUG` - Enable debug logging
- `INTERACTIVE` - Interactive mode flag
- `TIMEOUT` - Execution timeout (seconds)

### Framework-Specific (Keep from Claude)
- `BACKEND_API_URL` - Backend API base URL
- `PROJECT_NAME` - Kubernetes namespace
- `AGENTIC_SESSION_NAME` - Session CR name

### Framework-Specific (Remove)
- `LLM_MODEL` - Claude model selection
- `LLM_TEMPERATURE` - Model temperature
- `LLM_MAX_TOKENS` - Token limit
- `ANTHROPIC_API_KEY` - Claude API key

### Framework-Specific (Add for LangGraph)
- `LANGGRAPH_API_KEY` - LangGraph credentials
- `WORKFLOW_ID` - Workflow definition
- `WORKFLOW_CONFIG` - Workflow configuration (JSON)

## Architecture Diagram

```
Session Creation Flow
=====================

User → Backend API
         ↓
    Create AgenticSession CR
         ↓
    Operator watches CR
         ↓
    Create Kubernetes Job
         ├─ InitContainer (setup workspace)
         ├─ ambient-content (stays alive)
         └─ ambient-code-runner (actual work)
              ↓
         Runner pod starts
              ↓
         LangGraphAdapter.initialize()
              ├─ Load workflow
              ├─ Prepare workspace
              └─ Connect WebSocket
              ↓
         LangGraphAdapter.run()
              ├─ Execute workflow
              ├─ Stream events → WebSocket
              ├─ Update CR status (blocking)
              └─ Return result
              ↓
         Runner exits
              ↓
         Operator detects completion
              ├─ Clean up Job/Service
              └─ Keep PVC (for restart)
              ↓
         Frontend shows results
```

## Status Codes

Commonly used CR phase values:
- `Pending` - Job creation pending
- `Creating` - Job being created
- `Running` - Workflow executing
- `Completed` - Success (runner exited 0)
- `Failed` - Failure (runner exited non-zero)
- `Stopped` - User stopped session

## Common Issues & Solutions

See LANGGRAPH_RUNNER_CHECKLIST.md "Debugging Tips" section for:
- Viewing runner logs
- Checking CR status
- Verifying WebSocket connection
- Checking token injection
- Troubleshooting connection failures

## References

- CLAUDE.md - Project guidelines and standards
- RUNNER_PATTERN_ANALYSIS.md - Detailed technical analysis
- LANGGRAPH_RUNNER_CHECKLIST.md - Implementation guide
- claude-code-runner/wrapper.py - Reference implementation (1468 lines)
- runner-shell/core/ - Framework implementation

---

**Document Version**: 1.0
**Last Updated**: 2025-11-04
**Maintained By**: Ambient Code Team
