# Research: LangGraph Workflow Integration

**Feature**: LangGraph Workflow Integration
**Date**: 2025-11-04
**Research Phase**: Phase 0 - Resolve Technical Unknowns

This document consolidates research findings for all unknowns identified in the Technical Context section of plan.md.

---

## Table of Contents

1. [LangGraph Checkpoint Persistence with PostgreSQL](#1-langgraph-checkpoint-persistence-with-postgresql)
2. [JSON Schema Form Generation](#2-json-schema-form-generation)
3. [Existing Runner Pattern Analysis](#3-existing-runner-pattern-analysis)
4. [WebSocket Real-Time Messaging](#4-websocket-real-time-messaging)
5. [PostgreSQL JSONB Best Practices](#5-postgresql-jsonb-best-practices)

---

## 1. LangGraph Checkpoint Persistence with PostgreSQL

### Decision

Use the official **`langgraph-checkpoint-postgres`** library with **`AsyncPostgresSaver`** for checkpoint persistence.

### Rationale

1. **Official Support & Production-Ready**: Official LangGraph implementation optimized for production use in LangSmith with advanced storage efficiency
2. **Native Async Support**: Full Python asyncio support via `psycopg3` for non-blocking I/O with concurrent sessions
3. **Automatic Schema Management**: Handles database schema creation and migrations automatically via `.setup()` method

### Implementation Details

**Dependencies:**
```
langgraph-checkpoint-postgres>=3.0.0
psycopg[binary,pool]>=3.0
```

**Database Schema (4 tables auto-created):**

1. **checkpoints** - Main checkpoint data with JSONB storage
   - Primary key: (thread_id, checkpoint_ns, checkpoint_id)
   - Index on thread_id for fast lookups

2. **checkpoint_blobs** - Large channel values stored as BYTEA
   - Separate storage for efficiency
   - Linked by thread_id and version

3. **checkpoint_writes** - Intermediate writes during execution
   - Enables recovery from partial failures

4. **checkpoint_migrations** - Schema versioning for upgrades

**Python Implementation:**
```python
from langgraph.checkpoint.postgres.aio import AsyncPostgresSaver
from psycopg_pool import AsyncConnectionPool

DB_URI = "postgresql://user:password@postgres-host:5432/ambient_code"

async def init_checkpointer():
    pool = AsyncConnectionPool(
        conninfo=DB_URI,
        kwargs={
            "autocommit": True,
            "row_factory": dict_row
        }
    )

    checkpointer = AsyncPostgresSaver(pool)
    await checkpointer.setup()  # Create tables
    return checkpointer

# Use with LangGraph
graph = graph_builder.compile(checkpointer=checkpointer)

# Execute with thread management
config = {
    "configurable": {
        "thread_id": f"project-{project_name}:session-{session_name}",
        "checkpoint_ns": ""
    }
}
result = await graph.ainvoke(initial_state, config)
```

**Thread ID Strategy (Multi-Tenant Isolation):**
```python
# Project-scoped session
thread_id = f"project-{project_name}:session-{session_name}"

# Private user session
thread_id_private = f"project-{project_name}:user-{user_id}:session-{session_name}"
```

**Serialization Format:**
- Primitive values: Inlined in JSONB
- Complex objects: Stored as BYTEA with PickleCheckpointSerializer
- Custom serializers supported via SerializerProtocol

**Performance Optimizations:**
- Incremental storage (only changed values)
- Blob separation for large values
- Concurrent index creation
- Connection pooling
- Stateless design (singleton-safe)

**Checkpoint Retention:**
```python
# Manual cleanup (no built-in retention)
async def cleanup_old_checkpoints(project_name: str, days: int = 30):
    thread_pattern = f"project-{project_name}:%"
    # Delete from checkpoint_writes, checkpoint_blobs, checkpoints
    # WHERE created_at < NOW() - INTERVAL 'N days'
```

**Recommended Indexes:**
```sql
-- Project-based queries
CREATE INDEX checkpoints_thread_id_prefix_idx
ON checkpoints(thread_id text_pattern_ops)
WHERE thread_id LIKE 'project-%';

-- Time-based cleanup
CREATE INDEX checkpoints_metadata_timestamp_idx
ON checkpoints USING GIN (metadata)
WHERE metadata ? 'created_at';

-- Parent checkpoint lookups
CREATE INDEX checkpoints_parent_checkpoint_id_idx
ON checkpoints(parent_checkpoint_id)
WHERE parent_checkpoint_id IS NOT NULL;
```

### Alternatives Considered

- **SQLite Checkpointer**: Not suitable for production (file-based, no concurrent access)
- **In-Memory Checkpointer**: State lost on restart (not durable)
- **Redis Checkpointer**: Fast but higher memory cost, less mature
- **MongoDB Checkpointer**: Additional database dependency
- **Custom Implementation**: High development/maintenance burden

**Selected PostgreSQL** because: Platform already uses PostgreSQL, official implementation, strong durability, mature async support, no additional infrastructure.

### References

- [langgraph-checkpoint-postgres PyPI](https://pypi.org/project/langgraph-checkpoint-postgres/)
- [LangGraph Persistence Docs](https://docs.langchain.com/oss/python/langgraph/persistence)
- [PostgresSaver API Docs](https://api.python.langchain.com/en/latest/checkpoint/langchain_postgres.checkpoint.PostgresSaver.html)
- [GitHub Source](https://github.com/langchain-ai/langgraph/tree/main/libs/checkpoint-postgres)

---

## 2. JSON Schema Form Generation

### Decision

**Custom approach: JSON Schema → Zod (runtime) → React Hook Form + Shadcn UI**

Use `zod-from-json-schema` for runtime schema conversion with existing tech stack.

### Rationale

1. **Perfect Stack Alignment**: Leverages existing dependencies (react-hook-form, zod, @hookform/resolvers, Shadcn UI) already in project
2. **Zero Additional Heavy Dependencies**: Only adds `zod-from-json-schema` (~50KB) vs. 300KB+ for react-jsonschema-form
3. **Validation Consistency**: Backend validates JSON Schema (Go), frontend converts same schema to Zod - single source of truth
4. **Type Safety & Developer Experience**: Zod provides TypeScript inference, team already familiar with React Hook Form
5. **Flexibility**: Full control over field rendering, custom validation, layout customization

### Implementation Approach

**Dependencies:**
```bash
npm install zod-from-json-schema  # Only new dependency
```

**Schema Conversion Utility** (`src/lib/json-schema-to-form.ts`):
```typescript
import { parseSchema } from 'zod-from-json-schema';

export function convertJsonSchemaToZod(jsonSchema: Record<string, unknown>) {
  return parseSchema(jsonSchema);
}
```

**Dynamic Form Field Mapping:**
- `string` → `<Input>` from `@/components/ui/input`
- `string` with `enum` → `<Select>` from `@/components/ui/select`
- `number`/`integer` → `<Input type="number">`
- `boolean` → `<Checkbox>` from `@/components/ui/checkbox`
- `object` → Nested `<Fieldset>` with recursive rendering
- `array` → Dynamic list with add/remove controls

**Validation Strategy:**

*Client-Side:*
```typescript
const zodSchema = convertJsonSchemaToZod(workflowDefinition.spec.inputSchema);

const form = useForm({
  resolver: zodResolver(zodSchema),
  defaultValues: {}
});
```

*Server-Side:*
- Backend validates against original JSON Schema
- Return 400 with structured errors on validation failure
- Frontend displays errors via `form.setError()`

**Example Form Component:**
```typescript
export function WorkflowInputForm({ workflowDefinition, onSubmit }: Props) {
  const zodSchema = useMemo(
    () => convertJsonSchemaToZod(workflowDefinition.spec.inputSchema),
    [workflowDefinition.spec.inputSchema]
  );

  const form = useForm({
    resolver: zodResolver(zodSchema),
    defaultValues: {}
  });

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <DynamicFormFields
          schema={workflowDefinition.spec.inputSchema}
          control={form.control}
        />
        <Button type="submit" disabled={form.formState.isSubmitting}>
          {form.formState.isSubmitting && <Loader2 />}
          {form.formState.isSubmitting ? 'Creating...' : 'Create Session'}
        </Button>
      </form>
    </Form>
  );
}
```

**Customization Capabilities:**
- Field-level customization via `x-component` extension
- Layout configuration (grid/flex)
- Conditional fields via JSON Schema `if/then/else`
- Custom error messages via `errorMessage` extension

### Alternatives Considered

| Library | Pros | Cons | Why Not Selected |
|---------|------|------|-----------------|
| **react-jsonschema-form** | Battle-tested, comprehensive | 300KB+ bundle, custom theme needed, conflicts with zero-`any` standard | Heavy dependency, theming complexity |
| **uniforms** | Multi-schema support | 100KB+ bundle, learning curve, new state management | Introduces new paradigm, bundle overhead |
| **nextjs-shadcn-dynamic-form** | Ready-made, uses our stack | Custom format (not JSON Schema), less mature | Requires format conversion, limited features |
| **AJV + @hookform/resolvers** | Direct JSON Schema validation | No TypeScript inference, less customization | Loses Zod benefits, inconsistent approach |

### References

- [zod-from-json-schema](https://www.npmjs.com/package/zod-from-json-schema) - Runtime conversion library
- [React Hook Form](https://react-hook-form.com/) - Form library (existing)
- [Shadcn UI Forms Guide](https://ui.shadcn.com/docs/forms/react-hook-form) - Integration guide
- [Building Advanced React Forms](https://wasp.sh/blog/2025/01/22/advanced-react-hook-form-zod-shadcn) - 2025 guide

---

## 3. Existing Runner Pattern Analysis

### Overview

The platform has a comprehensive runner framework implemented in **claude-code-runner** that is 100% reusable for LangGraph runner. Detailed analysis documented in:
- `RUNNER_PATTERN_ANALYSIS.md` (795 lines)
- `LANGGRAPH_RUNNER_CHECKLIST.md` (451 lines)
- `ANALYSIS_INDEX.md` (287 lines)

### Key Finding

**Backend and Operator code is completely framework-agnostic** - no changes required to support LangGraph runners!

### Execution Flow (9 Steps)

1. **User Creates Session** → Backend API receives request
2. **Backend Creates AgenticSession CR** → Stored in Kubernetes etcd
3. **Operator Watches CR** → Detects new session
4. **Operator Spawns Job** → Kubernetes Job with runner pod
5. **Pod Initializes** → Runner authenticates, loads config
6. **Runner Executes** → Runs Claude Code CLI (or LangGraph workflow)
7. **WebSocket Streaming** → Real-time progress to frontend
8. **Status Updates** → Backend API receives status via HTTP
9. **Cleanup** → Job deleted, results stored in CR

### 100% Reusable Patterns

**1. Runner Shell Framework**
- `ResilientWebSocketTransport` class for WebSocket communication
- `AgentMessenger` for message formatting and routing
- Token/secret management
- **Location**: `components/runners/claude-code-runner/src/claude_code_runner/`

**2. Kubernetes Job Template**
- Environment variables (BACKEND_URL, WEBSOCKET_URL, etc.)
- PVC mounting for workspace
- Security context (non-privileged)
- Resource limits
- **Location**: `components/operator/internal/handlers/sessions.go:125-294`

**3. Backend Status Update API**
- POST `/api/v1/agentic-sessions/{namespace}/{name}/status`
- PATCH `/api/v1/agentic-sessions/{namespace}/{name}`
- **Location**: `components/backend/handlers/sessions.go:604-662`

**4. WebSocket Message Protocol**
```json
{
  "type": "agent_message",
  "timestamp": "2025-11-04T12:00:00Z",
  "data": {
    "message": "Processing...",
    "metadata": {}
  }
}
```

### Must Customize for LangGraph

**1. Core Execution Logic**
- Replace `ClaudeCodeAdapter` with `LangGraphAdapter`
- Implement workflow loading from container registry
- Integrate checkpoint manager
- **Pattern**: Same interface, different implementation

**2. Dependencies**
```python
# pyproject.toml
langgraph>=0.2.0
langgraph-checkpoint-postgres>=3.0.0
psycopg[binary,pool]>=3.0
anthropic>=0.68.0  # If using Claude models
```

**3. Environment Variables**
```python
# Existing (reuse)
BACKEND_URL, WEBSOCKET_URL, TOKEN, SESSION_NAME, NAMESPACE

# New (LangGraph-specific)
WORKFLOW_IMAGE, POSTGRES_CONNECTION_STRING, CHECKPOINT_RETENTION_DAYS
```

**4. Dockerfile**
```dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY pyproject.toml ./
RUN pip install -e .

COPY src/ ./src/

CMD ["python", "-m", "langgraph_runner"]
```

### LangGraphAdapter Skeleton

```python
class LangGraphAdapter:
    async def initialize(self):
        # Initialize checkpointer
        self.checkpointer = await init_checkpointer()

        # Load workflow from registry
        self.graph = await load_workflow(os.environ["WORKFLOW_IMAGE"])

    async def execute(self, initial_state: dict):
        config = {
            "configurable": {
                "thread_id": f"project-{self.namespace}:session-{self.session_name}",
                "checkpoint_ns": ""
            }
        }

        async for event in self.graph.astream(initial_state, config):
            await self.messenger.send_message(event)

    async def handle_interrupt(self):
        # Handle human-in-the-loop pauses
        user_input = await self.messenger.wait_for_user_input()
        await self.graph.ainvoke(user_input, config)
```

### Backend/Operator: Zero Changes Required

**Why**: Existing code is runner-agnostic:
- Job creation uses generic Job template
- Status updates via standard HTTP API
- WebSocket routing by session ID (not runner type)
- CR status fields are generic (phase, results, error)

**Only Addition**: WorkflowDefinition CRD and watch handler (new operator handler, doesn't modify existing)

### References

- **RUNNER_PATTERN_ANALYSIS.md**: Complete architectural breakdown
- **LANGGRAPH_RUNNER_CHECKLIST.md**: Step-by-step implementation guide
- **ANALYSIS_INDEX.md**: Navigation and quick reference
- **Source Files**:
  - `components/runners/claude-code-runner/src/claude_code_runner/__main__.py:23-115`
  - `components/operator/internal/handlers/sessions.go:125-294`
  - `components/backend/handlers/sessions.go:227-476`

---

## 4. WebSocket Real-Time Messaging

### Decision

Extend the existing **gorilla/websocket** implementation with session-type routing for workflow sessions.

### Current Implementation

**Backend WebSocket Server** (`components/backend/websocket/websocket_messaging.go`):
- Handles `/ws` endpoint with session-based routing
- Maintains connection pool by session name
- Message broadcasting to all clients of a session
- Automatic connection cleanup on disconnect

**Message Flow:**
1. Runner → Backend HTTP API → WebSocket Server → Frontend Client
2. User Input → Frontend → WebSocket → Backend → Runner (via polling or callback)

**Connection Authentication:**
- Token passed via query parameter: `/ws?sessionName=xyz&token=abc`
- Validated before upgrade to WebSocket

### Extension Strategy for Workflow Sessions

**No Changes to WebSocket Server**: Existing implementation is session-type agnostic

**Frontend Changes:**
1. Use same `/ws` endpoint with workflow session name
2. Message type differentiation handled in frontend:
```typescript
// Existing (Claude Code sessions)
ws.send(JSON.stringify({
  type: 'user_message',
  data: { message: userInput }
}));

// New (Workflow sessions) - same protocol
ws.send(JSON.stringify({
  type: 'user_input',  // Different type
  data: { response: userApproval }
}));
```

**Runner Changes:**
```python
# LangGraph runner publishes to same WebSocket transport
await messenger.send_message({
    "type": "workflow_progress",
    "data": {
        "step": "processing",
        "message": "Analyzing data..."
    }
})
```

### Message Protocol (Standardized)

**Agent → UI:**
```json
{
  "type": "agent_message",  # or "workflow_progress", "workflow_waiting_for_input"
  "timestamp": "2025-11-04T12:00:00Z",
  "data": {
    "message": "Processing complete",
    "metadata": {
      "step_name": "analyze",
      "progress": 0.75
    }
  }
}
```

**UI → Agent:**
```json
{
  "type": "user_input",
  "data": {
    "response": "approved",
    "metadata": {}
  }
}
```

### Human-in-the-Loop Pattern

**Workflow Pauses:**
1. LangGraph runner detects interrupt point
2. Sends `workflow_waiting_for_input` message via WebSocket
3. Updates session status to `waiting_for_input` via HTTP API
4. Frontend displays prompt to user
5. User responds via WebSocket
6. Runner receives input, resumes execution

**Implementation:**
```python
# In LangGraphAdapter
async def handle_interrupt(self, state: dict):
    # Checkpoint automatically saved by LangGraph
    await self.messenger.send_message({
        "type": "workflow_waiting_for_input",
        "data": {
            "prompt": "Approve outlier removal?",
            "options": ["approve", "reject"]
        }
    })

    # Wait for user input
    user_input = await self.messenger.wait_for_user_input()

    # Resume with input
    return {"user_approval": user_input["response"]}
```

### Connection Lifecycle

**Connection States:**
- `connecting`: Initial connection attempt
- `connected`: WebSocket open and authenticated
- `disconnected`: Connection lost (temporary)
- `closed`: Connection terminated (session complete)

**Reconnection Strategy:**
```typescript
// Frontend
const connectWebSocket = () => {
  const ws = new WebSocket(`wss://${BACKEND}/ws?sessionName=${name}&token=${token}`);

  ws.onclose = () => {
    if (sessionStatus !== 'completed') {
      // Retry with exponential backoff
      setTimeout(() => connectWebSocket(), Math.min(retries * 1000, 30000));
    }
  };
};
```

### Performance Considerations

**Existing Optimizations:**
- Connection pooling by session (multiple clients per session)
- Message buffering for disconnected clients (limited buffer)
- Heartbeat/ping for keep-alive

**Workflow-Specific:**
- Large output data stored in database (not sent via WebSocket)
- Progress messages throttled (max 1 per second)
- Checkpoint state not streamed (only status updates)

### References

- **Backend WebSocket**: `components/backend/websocket/websocket_messaging.go`
- **Runner Messenger**: `components/runners/claude-code-runner/src/claude_code_runner/agent_messenger.py`
- **Frontend WebSocket**: `components/frontend/src/hooks/use-websocket.ts` (if exists)

---

## 5. PostgreSQL JSONB Best Practices

### Decision

Use **JSONB columns** for semi-structured workflow input/output with **GIN indexes** for query performance.

### Schema Design

**WorkflowSession Table:**
```sql
CREATE TABLE workflow_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_name TEXT NOT NULL,
    session_name TEXT NOT NULL,
    workflow_name TEXT NOT NULL,
    input_data JSONB NOT NULL,  -- User-submitted input
    output_data JSONB,           -- Workflow results
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_by TEXT NOT NULL,
    error_message TEXT,

    UNIQUE(project_name, session_name)
);
```

**Rationale for JSONB:**
- Flexible schema (each workflow has different input/output structure)
- Efficient storage (binary format, compressed)
- Query support (JSON operators, indexing)
- Type safety (validation at application layer)

### Indexing Strategy

**1. Composite Index for Lookups:**
```sql
CREATE INDEX workflow_sessions_project_session_idx
ON workflow_sessions(project_name, session_name);
```

**2. GIN Index for JSONB Queries:**
```sql
-- For queries like: WHERE input_data @> '{"key": "value"}'
CREATE INDEX workflow_sessions_input_data_gin_idx
ON workflow_sessions USING GIN (input_data);

-- For path-specific queries
CREATE INDEX workflow_sessions_output_data_gin_idx
ON workflow_sessions USING GIN (output_data jsonb_path_ops);
```

**3. Partial Index for Active Sessions:**
```sql
CREATE INDEX workflow_sessions_active_idx
ON workflow_sessions(project_name, status)
WHERE status IN ('pending', 'running', 'waiting_for_input');
```

**4. Time-Based Index for Cleanup:**
```sql
CREATE INDEX workflow_sessions_completed_at_idx
ON workflow_sessions(completed_at)
WHERE completed_at IS NOT NULL;
```

### Query Patterns

**1. List Sessions by Project:**
```sql
SELECT id, session_name, workflow_name, status, created_at
FROM workflow_sessions
WHERE project_name = $1
ORDER BY created_at DESC
LIMIT 100;
```

**2. Get Session with Input/Output:**
```sql
SELECT *
FROM workflow_sessions
WHERE project_name = $1 AND session_name = $2;
```

**3. Search by Input Parameter:**
```sql
-- Example: Find sessions with specific CSV file
SELECT id, session_name, created_at
FROM workflow_sessions
WHERE project_name = $1
  AND input_data @> '{"file_path": "/data/sales.csv"}';
```

**4. Partial Output Update:**
```sql
-- Update specific field in output_data
UPDATE workflow_sessions
SET output_data = jsonb_set(
    COALESCE(output_data, '{}'),
    '{interim_results}',
    $1::jsonb
)
WHERE id = $2;
```

### Size Limits

**100MB Limit Enforcement:**
```sql
-- Add constraint
ALTER TABLE workflow_sessions
ADD CONSTRAINT output_data_size_check
CHECK (pg_column_size(output_data) < 104857600);  -- 100MB
```

**Application Layer Validation:**
```go
// Backend validation before insert
func validateOutputSize(output map[string]interface{}) error {
    data, _ := json.Marshal(output)
    if len(data) > 100*1024*1024 {
        return errors.New("output data exceeds 100MB limit")
    }
    return nil
}
```

**Handling Large Outputs:**
```python
# Workflow runner stores large files externally
if len(results) > 90 * 1024 * 1024:  # 90MB (safety margin)
    # Upload to object storage
    url = await upload_to_s3(results)
    output_data = {"result_url": url, "size": len(results)}
else:
    output_data = results
```

### JSONB Performance Best Practices

**1. Avoid Deep Nesting:**
```json
// ❌ Bad: Deep nesting hurts query performance
{
  "results": {
    "analysis": {
      "data": {
        "metrics": {
          "value": 123
        }
      }
    }
  }
}

// ✅ Good: Flat structure
{
  "analysis_metrics_value": 123,
  "analysis_timestamp": "2025-11-04T12:00:00Z"
}
```

**2. Use jsonb_path_ops for Key-Only Queries:**
```sql
-- Faster for @> queries, but doesn't support ? operator
CREATE INDEX workflow_sessions_input_path_idx
ON workflow_sessions USING GIN (input_data jsonb_path_ops);
```

**3. Extract Frequently Queried Fields:**
```sql
-- If often querying by workflow_name in input_data, extract to column
ALTER TABLE workflow_sessions
ADD COLUMN workflow_version TEXT GENERATED ALWAYS AS (input_data->>'version') STORED;

CREATE INDEX workflow_sessions_version_idx ON workflow_sessions(workflow_version);
```

**4. Use JSONB Operators Efficiently:**
```sql
-- Fast (uses GIN index)
WHERE input_data @> '{"key": "value"}'

-- Slow (no index)
WHERE input_data->>'key' = 'value'

-- Better (extract + B-tree index)
WHERE (input_data->>'key')::text = 'value'
```

### Migration Strategy

**Initial Schema:**
```sql
-- Migration 001_create_workflow_sessions.sql
CREATE TABLE workflow_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_name TEXT NOT NULL,
    session_name TEXT NOT NULL,
    workflow_name TEXT NOT NULL,
    input_data JSONB NOT NULL,
    output_data JSONB,
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_by TEXT NOT NULL,
    error_message TEXT,

    CONSTRAINT workflow_sessions_project_session_key
        UNIQUE(project_name, session_name),
    CONSTRAINT output_data_size_check
        CHECK (pg_column_size(output_data) < 104857600)
);

-- Indexes
CREATE INDEX workflow_sessions_project_session_idx
    ON workflow_sessions(project_name, session_name);
CREATE INDEX workflow_sessions_input_data_gin_idx
    ON workflow_sessions USING GIN (input_data);
CREATE INDEX workflow_sessions_active_idx
    ON workflow_sessions(project_name, status)
    WHERE status IN ('pending', 'running', 'waiting_for_input');
```

### Backup and Maintenance

**Regular VACUUM:**
```sql
-- JSONB updates can create bloat
VACUUM ANALYZE workflow_sessions;
```

**Partition by Time (Future Optimization):**
```sql
-- If table grows large (millions of rows)
CREATE TABLE workflow_sessions_2025_11 PARTITION OF workflow_sessions
FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
```

### References

- [PostgreSQL JSONB Documentation](https://www.postgresql.org/docs/current/datatype-json.html)
- [JSONB Indexing Best Practices](https://www.postgresql.org/docs/current/datatype-json.html#JSON-INDEXING)
- [GIN vs GiST Indexes](https://www.postgresql.org/docs/current/textsearch-indexes.html)
- [JSONB Performance Tips](https://www.enterprisedb.com/postgres-tutorials/how-use-jsonb-effectively-postgresql)

---

## Research Summary

All technical unknowns identified in plan.md have been resolved:

✅ **LangGraph Integration**: Use official `langgraph-checkpoint-postgres` with AsyncPostgresSaver
✅ **Form Generation**: Custom approach with zod-from-json-schema + existing stack
✅ **Runner Pattern**: 100% reusable framework from claude-code-runner
✅ **WebSocket Messaging**: Extend existing gorilla/websocket implementation
✅ **PostgreSQL JSONB**: Use JSONB with GIN indexes, 100MB limit, external storage for large outputs

**No blockers identified**. All patterns and libraries are production-ready and compatible with the Ambient Code Platform architecture.

**Next Phase**: Phase 1 - Design & Contracts (data-model.md, API contracts, quickstart.md)
