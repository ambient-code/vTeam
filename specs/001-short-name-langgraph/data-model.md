# Data Model: LangGraph Workflow Integration

**Feature**: LangGraph Workflow Integration
**Date**: 2025-11-04
**Phase**: Phase 1 - Design & Contracts

This document defines all data entities, their relationships, validation rules, and state transitions for the LangGraph workflow integration feature.

---

## Table of Contents

1. [Entity Overview](#entity-overview)
2. [WorkflowDefinition (Custom Resource)](#1-workflowdefinition-custom-resource)
3. [WorkflowSession (Database Table)](#2-workflowsession-database-table)
4. [SessionMessage (Database Table)](#3-sessionmessage-database-table)
5. [Checkpoint (Database Tables)](#4-checkpoint-database-tables)
6. [Entity Relationships](#entity-relationships)
7. [State Transition Diagrams](#state-transition-diagrams)

---

## Entity Overview

| Entity | Storage | Scope | Purpose |
|--------|---------|-------|---------|
| **WorkflowDefinition** | Kubernetes (etcd) | Cluster | Registry of available workflows |
| **WorkflowSession** | PostgreSQL | Project | Execution instance of a workflow |
| **SessionMessage** | PostgreSQL | Project | Real-time progress messages |
| **Checkpoint** | PostgreSQL | Project | LangGraph state persistence |

**Storage Strategy Rationale:**
- **WorkflowDefinition**: Cluster-scoped, administrator-managed → Kubernetes CR (consistent with existing AgenticSession pattern)
- **WorkflowSession**: High-volume, query-intensive, project-scoped → PostgreSQL (better than CR for database-style queries)
- **Checkpoint**: LangGraph native integration → PostgreSQL (required by `langgraph-checkpoint-postgres`)

---

## 1. WorkflowDefinition (Custom Resource)

### Overview

Represents a registered workflow template available cluster-wide. Created by cluster administrators, used by all projects.

### Custom Resource Definition

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: workflowdefinitions.vteam.ambient-code
spec:
  group: vteam.ambient-code
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required: [name, displayName, containerImage, inputSchema]
              properties:
                name:
                  type: string
                  pattern: '^[a-z0-9]([-a-z0-9]*[a-z0-9])?$'
                  maxLength: 63
                displayName:
                  type: string
                  maxLength: 100
                description:
                  type: string
                  maxLength: 500
                containerImage:
                  type: string
                  pattern: '^[a-zA-Z0-9._-]+(/[a-zA-Z0-9._-]+)*:[a-zA-Z0-9._-]+$'
                inputSchema:
                  type: object
                  x-kubernetes-preserve-unknown-fields: true
                version:
                  type: string
                  pattern: '^v[0-9]+\.[0-9]+\.[0-9]+$'
                tags:
                  type: array
                  items:
                    type: string
            status:
              type: object
              properties:
                phase:
                  type: string
                  enum: [Active, Deprecated, Error]
                registeredAt:
                  type: string
                  format: date-time
                lastUpdated:
                  type: string
                  format: date-time
                activeSessions:
                  type: integer
  scope: Cluster
  names:
    plural: workflowdefinitions
    singular: workflowdefinition
    kind: WorkflowDefinition
    shortNames: [wfdef]
```

### Example Resource

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: WorkflowDefinition
metadata:
  name: csv-forecast-analyzer
spec:
  name: csv-forecast-analyzer
  displayName: "CSV Forecast Analyzer"
  description: "Analyze CSV data and generate forecasts with human-in-the-loop validation of statistical outliers"
  containerImage: "quay.io/ambient_code/workflows/csv-forecast-analyzer:v1.0.0"
  version: "v1.0.0"
  tags: ["data-analysis", "forecasting", "interactive"]
  inputSchema:
    type: object
    required: ["csv_url", "forecast_column"]
    properties:
      csv_url:
        type: string
        format: uri
        title: "CSV File URL"
        description: "URL to the CSV file to analyze"
      forecast_column:
        type: string
        title: "Column to Forecast"
        description: "Name of the column containing values to forecast"
      forecast_periods:
        type: integer
        minimum: 1
        maximum: 365
        default: 30
        title: "Forecast Periods"
        description: "Number of periods to forecast"
      confidence_interval:
        type: number
        minimum: 0.01
        maximum: 0.99
        default: 0.95
        title: "Confidence Interval"
status:
  phase: Active
  registeredAt: "2025-11-04T12:00:00Z"
  lastUpdated: "2025-11-04T12:00:00Z"
  activeSessions: 3
```

### Attributes

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `spec.name` | string | Yes | Lowercase alphanumeric + hyphens, max 63 chars | Unique identifier (cluster-wide) |
| `spec.displayName` | string | Yes | Max 100 chars | Human-readable name for UI |
| `spec.description` | string | No | Max 500 chars | Brief description of workflow purpose |
| `spec.containerImage` | string | Yes | Valid image reference with tag | Container image from whitelisted registry |
| `spec.inputSchema` | object | Yes | Valid JSON Schema Draft 2020-12 | Defines workflow input structure |
| `spec.version` | string | No | Semantic version (vMAJOR.MINOR.PATCH) | Workflow version |
| `spec.tags` | array | No | Array of strings | Categorization tags for UI filtering |
| `status.phase` | string | No | Active, Deprecated, Error | Current lifecycle state |
| `status.registeredAt` | timestamp | No | ISO 8601 | Initial registration time |
| `status.lastUpdated` | timestamp | No | ISO 8601 | Last modification time |
| `status.activeSessions` | integer | No | Non-negative | Count of running sessions |

### Validation Rules

**1. Name Uniqueness (Cluster-Wide)**
- Enforced by: Backend API before CR creation
- Error: 409 Conflict if name already exists

**2. Container Image Registry Whitelist**
- Enforced by: Backend API validation
- Whitelist: Environment variable `ALLOWED_REGISTRIES` (comma-separated)
- Example: `quay.io,docker.io/ambient,ghcr.io/ambient`
- Error: 400 Bad Request if registry not whitelisted

**3. Input Schema Validity**
- Enforced by: Backend API JSON Schema validator
- Must conform to JSON Schema Draft 2020-12
- Error: 400 Bad Request with schema validation errors

**4. Deletion Restrictions**
- Enforced by: Backend API before deletion
- Rule: Cannot delete if `status.activeSessions > 0`
- Error: 409 Conflict with message "Cannot delete workflow with active sessions"

### Lifecycle

```
Register → Active → [Updated] → Deprecated → [Deleted when activeSessions = 0]
                ↓
              Error (invalid image/schema)
```

**States:**
- **Active**: Workflow is available for session creation
- **Deprecated**: Workflow exists but not recommended (admin marked)
- **Error**: Image pull failed or schema validation error

### Indexes / Lookups

**Kubernetes Indexes** (automatic by label):
```yaml
metadata:
  labels:
    vteam.ambient-code/type: workflow
    vteam.ambient-code/category: data-analysis
```

**Common Queries:**
- List all workflows: `kubectl get workflowdefinitions`
- Get by name: `kubectl get workflowdefinitions csv-forecast-analyzer`
- Filter by tag: Label selector `vteam.ambient-code/category=data-analysis`

---

## 2. WorkflowSession (Database Table)

### Overview

Represents an execution instance of a workflow within a project. Stores session metadata, input/output data, and tracks lifecycle state.

### PostgreSQL Schema

```sql
CREATE TABLE workflow_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Identifiers
    project_name TEXT NOT NULL,
    session_name TEXT NOT NULL,
    workflow_name TEXT NOT NULL,

    -- Session data
    input_data JSONB NOT NULL,
    output_data JSONB,

    -- Lifecycle tracking
    status TEXT NOT NULL CHECK (status IN (
        'pending', 'running', 'completed', 'failed', 'waiting_for_input'
    )),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Metadata
    created_by TEXT NOT NULL,
    error_message TEXT,

    -- Constraints
    CONSTRAINT workflow_sessions_project_session_key
        UNIQUE(project_name, session_name),
    CONSTRAINT output_data_size_check
        CHECK (pg_column_size(output_data) < 104857600)  -- 100MB
);

-- Indexes
CREATE INDEX workflow_sessions_project_session_idx
    ON workflow_sessions(project_name, session_name);

CREATE INDEX workflow_sessions_workflow_name_idx
    ON workflow_sessions(workflow_name);

CREATE INDEX workflow_sessions_input_data_gin_idx
    ON workflow_sessions USING GIN (input_data);

CREATE INDEX workflow_sessions_active_idx
    ON workflow_sessions(project_name, status)
    WHERE status IN ('pending', 'running', 'waiting_for_input');

CREATE INDEX workflow_sessions_created_at_idx
    ON workflow_sessions(project_name, created_at DESC);

CREATE INDEX workflow_sessions_completed_at_idx
    ON workflow_sessions(completed_at)
    WHERE completed_at IS NOT NULL;
```

### Attributes

| Field | Type | Nullable | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | UUID | No | gen_random_uuid() | Primary key |
| `project_name` | TEXT | No | - | Project namespace (tenant isolation) |
| `session_name` | TEXT | No | - | Unique name within project |
| `workflow_name` | TEXT | No | - | Reference to WorkflowDefinition |
| `input_data` | JSONB | No | - | User-submitted input matching workflow schema |
| `output_data` | JSONB | Yes | NULL | Workflow execution results |
| `status` | TEXT | No | - | Current lifecycle state |
| `created_at` | TIMESTAMP | No | NOW() | Session creation time |
| `started_at` | TIMESTAMP | Yes | NULL | Execution start time |
| `completed_at` | TIMESTAMP | Yes | NULL | Execution end time |
| `created_by` | TEXT | No | - | Username of creator |
| `error_message` | TEXT | Yes | NULL | Error details if status=failed |

### Validation Rules

**1. Session Name Uniqueness (Per Project)**
- Enforced by: UNIQUE constraint
- Scope: Within project only (different projects can have same session name)

**2. Input Data Schema Validation**
- Enforced by: Backend API before insert
- Validation: JSON Schema from WorkflowDefinition.spec.inputSchema
- Error: 400 Bad Request with validation errors

**3. Output Data Size Limit**
- Enforced by: CHECK constraint
- Limit: 100MB (104,857,600 bytes)
- Rationale: PostgreSQL JSONB performance best practices
- Error: 400 Bad Request "output data exceeds 100MB limit"

**4. Status Transitions**
- Enforced by: Backend API logic
- Valid transitions: See [State Transition Diagram](#workflow-session-states)
- Error: 409 Conflict for invalid state transitions

**5. Workflow Reference Integrity**
- Enforced by: Backend API validation
- Rule: `workflow_name` must reference existing WorkflowDefinition
- Error: 404 Not Found "workflow not found"

### Status Values

| Status | Description | Terminal? |
|--------|-------------|-----------|
| `pending` | Session created, waiting for job to start | No |
| `running` | Job is executing | No |
| `waiting_for_input` | Paused for human approval | No |
| `completed` | Execution finished successfully | Yes |
| `failed` | Execution failed with error | Yes |

### Example Row

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "project_name": "data-team",
  "session_name": "sales-forecast-2025-11",
  "workflow_name": "csv-forecast-analyzer",
  "input_data": {
    "csv_url": "https://storage.example.com/sales_2024.csv",
    "forecast_column": "revenue",
    "forecast_periods": 30,
    "confidence_interval": 0.95
  },
  "output_data": {
    "forecast": [
      {"date": "2025-12-01", "predicted": 125000, "ci_lower": 118000, "ci_upper": 132000},
      {"date": "2025-12-02", "predicted": 127000, "ci_lower": 120000, "ci_upper": 134000}
    ],
    "model_metrics": {
      "mae": 2500,
      "rmse": 3200,
      "mape": 0.02
    }
  },
  "status": "completed",
  "created_at": "2025-11-04T10:00:00Z",
  "started_at": "2025-11-04T10:00:15Z",
  "completed_at": "2025-11-04T10:05:30Z",
  "created_by": "alice@example.com",
  "error_message": null
}
```

### Relationships

- **References**: WorkflowDefinition (by `workflow_name`)
- **Has Many**: SessionMessage (via `workflow_session_id` foreign key)
- **Has Many**: Checkpoint (via thread_id = "project-{project_name}:session-{session_name}")

---

## 3. SessionMessage (Database Table)

### Overview

Represents a single message in a workflow session's conversation history. Provides real-time progress updates and maintains conversation context for resumption.

### PostgreSQL Schema

```sql
CREATE TABLE session_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Foreign key
    workflow_session_id UUID NOT NULL REFERENCES workflow_sessions(id) ON DELETE CASCADE,

    -- Message data
    message_type TEXT NOT NULL CHECK (message_type IN (
        'system', 'agent', 'user', 'error', 'workflow_progress', 'workflow_waiting_for_input'
    )),
    content JSONB NOT NULL,

    -- Ordering
    sequence_number INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT session_messages_session_sequence_key
        UNIQUE(workflow_session_id, sequence_number)
);

-- Indexes
CREATE INDEX session_messages_session_id_idx
    ON session_messages(workflow_session_id, sequence_number);

CREATE INDEX session_messages_timestamp_idx
    ON session_messages(workflow_session_id, timestamp);

CREATE INDEX session_messages_type_idx
    ON session_messages(workflow_session_id, message_type);
```

### Attributes

| Field | Type | Nullable | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | UUID | No | gen_random_uuid() | Primary key |
| `workflow_session_id` | UUID | No | - | Foreign key to workflow_sessions |
| `message_type` | TEXT | No | - | Message category |
| `content` | JSONB | No | - | Message payload (structure varies by type) |
| `sequence_number` | INTEGER | No | - | Order within session |
| `timestamp` | TIMESTAMP | No | NOW() | Message creation time |

### Message Types

| Type | Description | Content Structure |
|------|-------------|-------------------|
| `system` | Platform-generated messages | `{"message": "Session started"}` |
| `agent` | Workflow-generated messages | `{"message": "Processing data...", "metadata": {...}}` |
| `user` | User responses to prompts | `{"response": "approved", "metadata": {...}}` |
| `error` | Error messages | `{"error": "Division by zero", "stack_trace": "..."}` |
| `workflow_progress` | Progress updates | `{"step": "analyze", "progress": 0.75, "message": "..."}` |
| `workflow_waiting_for_input` | Human-in-the-loop prompts | `{"prompt": "Approve?", "options": ["yes", "no"]}` |

### Example Rows

```json
[
  {
    "id": "a1b2c3d4-...",
    "workflow_session_id": "550e8400-...",
    "message_type": "system",
    "content": {"message": "Session started"},
    "sequence_number": 1,
    "timestamp": "2025-11-04T10:00:15Z"
  },
  {
    "id": "b2c3d4e5-...",
    "workflow_session_id": "550e8400-...",
    "message_type": "workflow_progress",
    "content": {
      "step": "load_data",
      "progress": 0.25,
      "message": "Loading CSV file from storage"
    },
    "sequence_number": 2,
    "timestamp": "2025-11-04T10:00:20Z"
  },
  {
    "id": "c3d4e5f6-...",
    "workflow_session_id": "550e8400-...",
    "message_type": "workflow_waiting_for_input",
    "content": {
      "prompt": "Found 3 statistical outliers. Remove them?",
      "options": ["approve", "reject"],
      "outliers": [{"date": "2024-01-15", "value": 250000}]
    },
    "sequence_number": 3,
    "timestamp": "2025-11-04T10:02:00Z"
  },
  {
    "id": "d4e5f6g7-...",
    "workflow_session_id": "550e8400-...",
    "message_type": "user",
    "content": {"response": "approve"},
    "sequence_number": 4,
    "timestamp": "2025-11-04T10:03:30Z"
  },
  {
    "id": "e5f6g7h8-...",
    "workflow_session_id": "550e8400-...",
    "message_type": "workflow_progress",
    "content": {
      "step": "forecast",
      "progress": 0.75,
      "message": "Generating forecasts with approved parameters"
    },
    "sequence_number": 5,
    "timestamp": "2025-11-04T10:04:00Z"
  }
]
```

### Validation Rules

**1. Sequence Number Uniqueness**
- Enforced by: UNIQUE constraint
- Scope: Within workflow_session_id
- Auto-increment: Application logic (backend API)

**2. Message Type Validation**
- Enforced by: CHECK constraint
- Valid values: See Message Types table

**3. Content Structure**
- Enforced by: Application logic (backend API)
- Each message type has expected content schema

**4. Cascade Delete**
- Enforced by: ON DELETE CASCADE foreign key
- Behavior: Delete all messages when session is deleted

### Relationships

- **Belongs To**: WorkflowSession (via `workflow_session_id`)

---

## 4. Checkpoint (Database Tables)

### Overview

Represents saved workflow state at specific execution points. Managed by LangGraph's `langgraph-checkpoint-postgres` library. Enables session resumption and time-travel.

### PostgreSQL Schema

**Note**: Schema is auto-created by `AsyncPostgresSaver.setup()`. Documented here for reference.

**1. checkpoints table** (main checkpoint data):
```sql
CREATE TABLE IF NOT EXISTS checkpoints (
    thread_id TEXT NOT NULL,
    checkpoint_ns TEXT NOT NULL DEFAULT '',
    checkpoint_id TEXT NOT NULL,
    parent_checkpoint_id TEXT,
    type TEXT,
    checkpoint JSONB NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',

    PRIMARY KEY (thread_id, checkpoint_ns, checkpoint_id)
);

CREATE INDEX IF NOT EXISTS checkpoints_thread_id_idx
    ON checkpoints(thread_id);
```

**2. checkpoint_blobs table** (large channel values):
```sql
CREATE TABLE IF NOT EXISTS checkpoint_blobs (
    thread_id TEXT NOT NULL,
    checkpoint_ns TEXT NOT NULL DEFAULT '',
    channel TEXT NOT NULL,
    version TEXT NOT NULL,
    type TEXT NOT NULL,
    blob BYTEA,

    PRIMARY KEY (thread_id, checkpoint_ns, channel, version)
);

CREATE INDEX IF NOT EXISTS checkpoint_blobs_thread_id_idx
    ON checkpoint_blobs(thread_id);
```

**3. checkpoint_writes table** (intermediate writes):
```sql
CREATE TABLE IF NOT EXISTS checkpoint_writes (
    thread_id TEXT NOT NULL,
    checkpoint_ns TEXT NOT NULL DEFAULT '',
    checkpoint_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    idx INTEGER NOT NULL,
    channel TEXT NOT NULL,
    type TEXT,
    blob BYTEA NOT NULL,
    task_path TEXT NOT NULL DEFAULT '',

    PRIMARY KEY (thread_id, checkpoint_ns, checkpoint_id, task_id, idx)
);

CREATE INDEX IF NOT EXISTS checkpoint_writes_thread_id_idx
    ON checkpoint_writes(thread_id);
```

### Thread ID Convention

**Format**: `project-{project_name}:session-{session_name}`

**Examples:**
- Project-scoped: `project-data-team:session-sales-forecast-2025-11`
- Private (future): `project-data-team:user-alice:session-my-forecast`

**Purpose**: Encodes multi-tenant isolation in thread_id for security and cleanup

### Attributes (checkpoints table)

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `thread_id` | TEXT | No | Session identifier (multi-tenant encoded) |
| `checkpoint_ns` | TEXT | No | Namespace for subgraphs (default: empty string) |
| `checkpoint_id` | TEXT | No | Unique checkpoint identifier |
| `parent_checkpoint_id` | TEXT | Yes | Previous checkpoint for time-travel |
| `type` | TEXT | Yes | Checkpoint type (reserved for future use) |
| `checkpoint` | JSONB | No | Serialized state data |
| `metadata` | JSONB | No | Custom metadata (timestamps, user info) |

### Checkpoint Lifecycle

```
Create → [Update] → [Resume from] → [Cleanup after retention period]
```

**Retention Policy** (application-managed):
- Default: 30 days
- Configurable: Environment variable `CHECKPOINT_RETENTION_DAYS`
- Cleanup: Periodic CronJob deletes old checkpoints

### Example Checkpoint

```json
{
  "thread_id": "project-data-team:session-sales-forecast-2025-11",
  "checkpoint_ns": "",
  "checkpoint_id": "1ef8f5b0-7d4e-6f9e-b543-d3e5c8a7b9f1",
  "parent_checkpoint_id": "1ef8f5b0-7d4e-6f9e-b543-c2d4b7a6c8e0",
  "type": null,
  "checkpoint": {
    "v": 1,
    "ts": "2025-11-04T10:02:00.000Z",
    "id": "1ef8f5b0-7d4e-6f9e-b543-d3e5c8a7b9f1",
    "channel_values": {
      "data_state": {
        "csv_data": "...",
        "outliers": [...]
      },
      "agent_state": {
        "current_step": "validate_outliers",
        "pending_approval": true
      }
    },
    "channel_versions": {
      "data_state": "...",
      "agent_state": "..."
    },
    "versions_seen": {}
  },
  "metadata": {
    "created_at": "2025-11-04T10:02:00Z",
    "step": "validate_outliers",
    "user_triggered": false
  }
}
```

### Relationships

- **Linked By**: `thread_id` (corresponds to WorkflowSession via naming convention)
- **Chained By**: `parent_checkpoint_id` (forms checkpoint history)

---

## Entity Relationships

### Diagram

```
┌─────────────────────────────┐
│   WorkflowDefinition (CR)   │  Cluster-scoped
│   ─────────────────────     │
│   - name (PK)               │
│   - displayName             │
│   - containerImage          │
│   - inputSchema             │
└──────────────┬──────────────┘
               │
               │ referenced by
               ↓
┌─────────────────────────────┐
│   WorkflowSession (DB)      │  Project-scoped
│   ──────────────────────    │
│   - id (PK)                 │
│   - project_name + session_name (UNIQUE)
│   - workflow_name (FK)      │
│   - input_data (JSONB)      │
│   - output_data (JSONB)     │
│   - status                  │
└──────┬──────────────┬───────┘
       │              │
       │ has many     │ encoded in thread_id
       ↓              ↓
┌───────────────┐   ┌──────────────────────┐
│ SessionMessage│   │   Checkpoint (DB)    │
│ ────────────  │   │   ─────────────      │
│ - id (PK)     │   │   - thread_id (PK)   │
│ - session_id  │   │   - checkpoint_id(PK)│
│   (FK)        │   │   - checkpoint(JSONB)│
│ - type        │   │   - parent_id        │
│ - content     │   └──────────────────────┘
│ - sequence    │
└───────────────┘
```

### Relationship Details

**WorkflowDefinition → WorkflowSession**
- Type: One-to-Many
- Cardinality: 1 WorkflowDefinition : N WorkflowSessions
- Referential Integrity: Backend API validation (not database FK due to cross-storage)
- Cascade: WorkflowDefinition cannot be deleted if active sessions exist

**WorkflowSession → SessionMessage**
- Type: One-to-Many
- Cardinality: 1 WorkflowSession : N SessionMessages
- Referential Integrity: Foreign key with CASCADE delete
- Cascade: All messages deleted when session is deleted

**WorkflowSession → Checkpoint**
- Type: One-to-Many (conceptual)
- Cardinality: 1 WorkflowSession : N Checkpoints
- Linking: thread_id encoding (not database FK)
- Cascade: Application-managed cleanup (delete checkpoints when session deleted)

---

## State Transition Diagrams

### Workflow Session States

```
         [Create Session]
                ↓
            pending ───────────┐
                ↓              │ [Job fails to start]
            [Job starts]       ↓
                ↓            failed
            running            ↑
                ↓              │
      ┌─────────┴─────────┐   │
      ↓                   ↓   │
[Needs approval]    [Execution │
      ↓              fails]    │
waiting_for_input        │     │
      ↓                  │     │
[User responds]          └─────┘
      ↓
   running
      ↓
[Execution completes]
      ↓
  completed
```

**Valid State Transitions:**

| From | To | Trigger | Notes |
|------|-----|---------|-------|
| pending | running | Job starts executing | `started_at` timestamp set |
| pending | failed | Job fails to start | `error_message` set, `completed_at` set |
| running | waiting_for_input | Workflow hits interrupt | `completed_at` remains NULL |
| running | completed | Workflow finishes successfully | `output_data` set, `completed_at` set |
| running | failed | Workflow encounters error | `error_message` set, `completed_at` set |
| waiting_for_input | running | User provides input | Resume execution |
| waiting_for_input | failed | User cancels or timeout | `error_message` set, `completed_at` set |

**Terminal States:** completed, failed

**Invalid Transitions** (enforced by backend API):
- completed → any (cannot restart completed sessions)
- failed → any (cannot restart failed sessions)
- Any backward transition (no state rollback)

### WorkflowDefinition Phases

```
    [Register]
        ↓
     Active ──────┐
        ↓         │
   [Image pull   [Admin marks deprecated]
    or schema    │
    error]       ↓
        ↓      Deprecated
     Error       ↓
        │    [Delete when activeSessions = 0]
        └──────→ [Deleted]
```

**Phase Meanings:**
- **Active**: Normal operational state, available for session creation
- **Deprecated**: Still exists but admin-flagged as obsolete
- **Error**: Registration or validation failed

---

## Database Migrations

### Migration 001: Initial Schema

```sql
-- File: migrations/001_initial_schema.up.sql

-- WorkflowSession table
CREATE TABLE workflow_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_name TEXT NOT NULL,
    session_name TEXT NOT NULL,
    workflow_name TEXT NOT NULL,
    input_data JSONB NOT NULL,
    output_data JSONB,
    status TEXT NOT NULL CHECK (status IN (
        'pending', 'running', 'completed', 'failed', 'waiting_for_input'
    )),
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

-- SessionMessage table
CREATE TABLE session_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_session_id UUID NOT NULL REFERENCES workflow_sessions(id) ON DELETE CASCADE,
    message_type TEXT NOT NULL CHECK (message_type IN (
        'system', 'agent', 'user', 'error', 'workflow_progress', 'workflow_waiting_for_input'
    )),
    content JSONB NOT NULL,
    sequence_number INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT session_messages_session_sequence_key
        UNIQUE(workflow_session_id, sequence_number)
);

-- Indexes for workflow_sessions
CREATE INDEX workflow_sessions_project_session_idx
    ON workflow_sessions(project_name, session_name);
CREATE INDEX workflow_sessions_workflow_name_idx
    ON workflow_sessions(workflow_name);
CREATE INDEX workflow_sessions_input_data_gin_idx
    ON workflow_sessions USING GIN (input_data);
CREATE INDEX workflow_sessions_active_idx
    ON workflow_sessions(project_name, status)
    WHERE status IN ('pending', 'running', 'waiting_for_input');
CREATE INDEX workflow_sessions_created_at_idx
    ON workflow_sessions(project_name, created_at DESC);
CREATE INDEX workflow_sessions_completed_at_idx
    ON workflow_sessions(completed_at)
    WHERE completed_at IS NOT NULL;

-- Indexes for session_messages
CREATE INDEX session_messages_session_id_idx
    ON session_messages(workflow_session_id, sequence_number);
CREATE INDEX session_messages_timestamp_idx
    ON session_messages(workflow_session_id, timestamp);
CREATE INDEX session_messages_type_idx
    ON session_messages(workflow_session_id, message_type);

-- Checkpoint tables created automatically by langgraph-checkpoint-postgres
-- (No migration needed - AsyncPostgresSaver.setup() handles this)
```

```sql
-- File: migrations/001_initial_schema.down.sql

DROP TABLE IF EXISTS session_messages CASCADE;
DROP TABLE IF EXISTS workflow_sessions CASCADE;
-- Checkpoint tables cleanup handled by application
```

---

## Summary

**Entities Defined:** 4 (WorkflowDefinition, WorkflowSession, SessionMessage, Checkpoint)

**Storage Split:**
- Kubernetes (etcd): 1 entity (WorkflowDefinition - cluster-scoped, admin-managed)
- PostgreSQL: 3 entities (WorkflowSession, SessionMessage, Checkpoint - project-scoped, query-optimized)

**Key Design Decisions:**
- JSONB for flexible input/output schemas (each workflow has different structure)
- 100MB output limit with external storage fallback (PostgreSQL performance)
- Thread ID encoding for multi-tenant checkpoint isolation
- Cascade deletes for data consistency
- GIN indexes on JSONB columns for query performance

**Validation Enforcement:**
- Cluster-wide uniqueness: Backend API (WorkflowDefinition names)
- Project-scoped uniqueness: Database constraints (WorkflowSession names)
- Schema validation: Backend API (input_data against inputSchema)
- State transitions: Backend API logic
- Size limits: Database CHECK constraints

**Next Phase**: API Contract Generation (contracts/)
