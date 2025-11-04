# LangGraph Workflow Integration - Architecture Document

## Table of Contents

1. [Overview](#overview)
2. [System Architecture](#system-architecture)
3. [Component Design](#component-design)
4. [Data Model](#data-model)
5. [API Specifications](#api-specifications)
6. [Deployment Architecture](#deployment-architecture)
7. [Security Architecture](#security-architecture)
8. [Sequence Diagrams](#sequence-diagrams)
9. [Integration Patterns](#integration-patterns)
10. [Migration Strategy](#migration-strategy)
11. [Observability & Monitoring](#observability--monitoring)
12. [Performance Considerations](#performance-considerations)

---

## Overview

### Purpose

This document provides the technical architecture for integrating LangGraph workflows into the Ambient Code Platform. The design introduces a database-backed orchestration model that coexists with the legacy Custom Resource-based system, enabling users to execute custom LangGraph workflows alongside Claude Code sessions.

### Design Principles

1. **Separation of Concerns**: Clear boundaries between orchestration (platform) and execution (workflows)
2. **Extensibility**: Generic interfaces supporting multiple runner types (LangGraph, future: OpenAI, Gemini)
3. **Database-First**: PostgreSQL as source of truth for workflow sessions (addresses K8s etcd scalability)
4. **Non-Breaking**: Legacy AgenticSession CRs remain functional during transition period
5. **Security by Default**: Registry whitelisting, pod security contexts, token-based authentication
6. **Observable**: Comprehensive logging, metrics, and tracing for debugging and monitoring

### Key Architectural Decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| **Database-backed sessions** | Avoid K8s etcd pressure, enable complex queries, support future analytics | Requires PostgreSQL dependency, adds operational complexity |
| **Backend creates Jobs directly** | Simplifies architecture, reduces Operator complexity | Backend needs K8s API permissions, bypasses Operator pattern |
| **Cluster-scoped WorkflowDefinitions** | Enable reusability across projects, reduce duplication | Requires cluster-admin for registration |
| **BYOI (Bring Your Own Image)** | Avoid building infrastructure complexity, shift responsibility to users | Users must build/push images, no platform-managed builds |
| **Single-container Jobs** | Workflows don't need file access, simpler pod design | Cannot reuse ambient-content service pattern |
| **Shared PostgreSQL checkpointer** | Platform-managed persistence, consistent UX | Checkpoint isolation complexity, DB sizing concerns |

---

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Ambient Code Platform                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────┐         ┌──────────────┐       ┌──────────────┐  │
│  │   Frontend   │────────▶│   Backend    │──────▶│  PostgreSQL  │  │
│  │  (Next.js)   │         │   (Go/Gin)   │       │  (Sessions + │  │
│  │              │◀────────│              │◀──────│  Checkpoints)│  │
│  └──────────────┘   HTTP  └──────┬───────┘       └──────────────┘  │
│         │              WebSocket  │                                  │
│         │                         │ K8s API                          │
│         │                         ▼                                  │
│         │              ┌────────────────────┐                        │
│         │              │   Kubernetes API   │                        │
│         │              │    (Jobs, Pods,    │                        │
│         │              │  WorkflowDef CRs)  │                        │
│         │              └─────────┬──────────┘                        │
│         │                        │                                   │
│         │              ┌─────────▼───────────────────┐               │
│         │              │      Job (per session)      │               │
│         │              │  ┌────────────────────────┐ │               │
│         │              │  │  Workflow Runner Pod   │ │               │
│         │              │  │ ┌────────────────────┐ │ │               │
│         │              │  │ │ User's LangGraph   │ │ │               │
│         └──────────────┼──┼─│    Workflow +      │ │ │               │
│           WebSocket    │  │ │  Base Runner Layer │ │ │               │
│                        │  │ └────────────────────┘ │ │               │
│                        │  └────────────────────────┘ │               │
│                        └─────────────────────────────┘               │
│                                                                       │
│  ┌──────────────────────────────────────────────────────┐           │
│  │             Legacy System (Unchanged)                 │           │
│  │  AgenticSession CR → Operator → Job → Claude Runner  │           │
│  └──────────────────────────────────────────────────────┘           │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

### Component Interaction Flow

```
User → Frontend → Backend → PostgreSQL
                  ↓
                  K8s API (create Job)
                  ↓
                  Job/Pod (workflow runner)
                  ↓
                  Backend API (status updates) → PostgreSQL
                  ↓
                  Frontend (WebSocket updates)
```

### Dual-Mode Architecture

The platform operates in **hybrid mode** with two parallel systems:

| Aspect | Legacy (CR-backed) | New (DB-backed) |
|--------|-------------------|-----------------|
| **Session Type** | AgenticSession CR | `workflow_sessions` table |
| **Orchestration** | Operator watches CRs | Backend creates Jobs directly |
| **Storage** | K8s etcd | PostgreSQL |
| **Runner Type** | Claude Code only | LangGraph workflows |
| **UI Routes** | `/projects/:project/sessions` | `/projects/:project/workflow-sessions` |
| **API Endpoints** | `/api/projects/:project/agentic-sessions` | `/api/v2/projects/:project/workflow-sessions` |

**Convergence Plan**: Eventually migrate legacy sessions to DB-backed model (post-MVP).

---

## Component Design

### 1. WorkflowDefinition Custom Resource

**Location**: Cluster-scoped (no namespace)

**CRD Schema**:

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
            required: [displayName, image, inputSchema]
            properties:
              displayName:
                type: string
                description: Human-readable name shown in UI
              description:
                type: string
                description: Detailed description of workflow purpose
              image:
                type: string
                description: Container image URL (must be from whitelisted registry)
                pattern: '^[a-z0-9\-\.]+/[a-z0-9\-\./_]+:[a-z0-9\-\.]+$'
              imagePullSecret:
                type: string
                description: Optional Kubernetes secret name for private registry auth
              inputSchema:
                type: object
                description: JSON Schema defining workflow inputs
                x-kubernetes-preserve-unknown-fields: true
          status:
            type: object
            properties:
              validated:
                type: boolean
                description: Whether image was successfully validated
              lastValidated:
                type: string
                format: date-time
              validationMessage:
                type: string
                description: Error message if validation failed
              activeSessionCount:
                type: integer
                description: Number of currently running sessions using this workflow
  scope: Cluster
  names:
    plural: workflowdefinitions
    singular: workflowdefinition
    kind: WorkflowDefinition
    shortNames: [wfd, workflow]
```

**RBAC Requirements**:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: workflow-definition-admin
rules:
- apiGroups: ["vteam.ambient-code"]
  resources: ["workflowdefinitions"]
  verbs: ["create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: workflow-definition-viewer
rules:
- apiGroups: ["vteam.ambient-code"]
  resources: ["workflowdefinitions"]
  verbs: ["get", "list", "watch"]
```

**Backend Service Account** (for creating/reading CRs):

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: backend-service-account
  namespace: ambient-code
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: backend-workflow-definitions
subjects:
- kind: ServiceAccount
  name: backend-service-account
  namespace: ambient-code
roleRef:
  kind: ClusterRole
  name: workflow-definition-viewer
  apiGroup: rbac.authorization.k8s.io
```

---

### 2. PostgreSQL Database Schema

**Connection Configuration**:

```go
// Backend config
type DatabaseConfig struct {
    Host     string // From env: DB_HOST
    Port     int    // From env: DB_PORT
    Database string // From env: DB_NAME
    Username string // From env: DB_USER
    Password string // From env: DB_PASSWORD (from Secret)
    SSLMode  string // From env: DB_SSL_MODE (prefer, require, disable)

    // Connection pool settings
    MaxOpenConns    int // Default: 25
    MaxIdleConns    int // Default: 5
    ConnMaxLifetime time.Duration // Default: 5 minutes
}
```

**Schema Migrations** (using golang-migrate):

```sql
-- Migration: 001_create_workflow_sessions.up.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE workflow_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_name VARCHAR(255) NOT NULL,
    workflow_definition_name VARCHAR(255) NOT NULL,
    workflow_image VARCHAR(512) NOT NULL,
    workflow_version VARCHAR(128),

    -- Session lifecycle
    status VARCHAR(50) NOT NULL CHECK (status IN (
        'pending', 'running', 'completed', 'failed',
        'waiting_for_input', 'cancelled', 'timeout'
    )),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- User context
    created_by_user VARCHAR(255) NOT NULL,

    -- Workflow data
    input_data JSONB NOT NULL,
    output_data JSONB,
    error_message TEXT,

    -- LangGraph persistence
    thread_id VARCHAR(255) UNIQUE,
    checkpoint_id VARCHAR(255),

    -- Kubernetes Job reference
    job_name VARCHAR(255),

    -- Metadata
    display_name VARCHAR(255),
    labels JSONB DEFAULT '{}',
    annotations JSONB DEFAULT '{}',

    -- Statistics
    execution_duration_seconds INT,
    message_count INT DEFAULT 0,

    -- Soft delete
    deleted_at TIMESTAMPTZ
);

-- Indexes for query performance
CREATE INDEX idx_workflow_sessions_project ON workflow_sessions(project_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_workflow_sessions_workflow ON workflow_sessions(workflow_definition_name);
CREATE INDEX idx_workflow_sessions_status ON workflow_sessions(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_workflow_sessions_user ON workflow_sessions(created_by_user);
CREATE INDEX idx_workflow_sessions_thread ON workflow_sessions(thread_id);
CREATE INDEX idx_workflow_sessions_created ON workflow_sessions(created_at DESC);
CREATE INDEX idx_workflow_sessions_labels ON workflow_sessions USING GIN (labels);

-- Migration: 002_create_langgraph_checkpoints.up.sql
CREATE TABLE langgraph_checkpoints (
    thread_id VARCHAR(255) NOT NULL,
    checkpoint_id VARCHAR(255) NOT NULL,
    parent_checkpoint_id VARCHAR(255),

    -- Checkpoint payload
    checkpoint_data JSONB NOT NULL,
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Scope isolation
    scope VARCHAR(50) NOT NULL CHECK (scope IN ('project', 'private', 'global')),
    scope_identifier VARCHAR(255) NOT NULL,

    PRIMARY KEY (thread_id, checkpoint_id)
);

CREATE INDEX idx_checkpoints_thread ON langgraph_checkpoints(thread_id);
CREATE INDEX idx_checkpoints_scope ON langgraph_checkpoints(scope, scope_identifier);
CREATE INDEX idx_checkpoints_created ON langgraph_checkpoints(created_at);

-- Migration: 003_create_workflow_session_messages.up.sql
CREATE TABLE workflow_session_messages (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES workflow_sessions(id) ON DELETE CASCADE,

    sequence_number INT NOT NULL,
    message_type VARCHAR(50) NOT NULL CHECK (message_type IN (
        'system.message', 'agent.message', 'user.message',
        'message.partial', 'agent.running', 'agent.waiting',
        'tool.use', 'tool.result', 'thinking'
    )),

    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payload JSONB NOT NULL,

    -- For partial messages
    partial_index INT,
    is_complete BOOLEAN DEFAULT true
);

CREATE INDEX idx_messages_session ON workflow_session_messages(session_id, sequence_number);
CREATE INDEX idx_messages_timestamp ON workflow_session_messages(timestamp);

-- Migration: 004_create_workflow_execution_logs.up.sql
CREATE TABLE workflow_execution_logs (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES workflow_sessions(id) ON DELETE CASCADE,

    log_level VARCHAR(20) NOT NULL CHECK (log_level IN ('DEBUG', 'INFO', 'WARN', 'ERROR')),
    message TEXT NOT NULL,
    context JSONB DEFAULT '{}',

    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_logs_session ON workflow_execution_logs(session_id, timestamp DESC);
CREATE INDEX idx_logs_level ON workflow_execution_logs(log_level, timestamp DESC);
```

**Database Access Patterns**:

```go
// Backend database interface
type WorkflowSessionRepository interface {
    // CRUD operations
    Create(ctx context.Context, session *WorkflowSession) error
    GetByID(ctx context.Context, id uuid.UUID) (*WorkflowSession, error)
    List(ctx context.Context, project string, filters ListFilters) ([]*WorkflowSession, error)
    Update(ctx context.Context, session *WorkflowSession) error
    Delete(ctx context.Context, id uuid.UUID) error

    // Status updates (called by runner via Backend API)
    UpdateStatus(ctx context.Context, id uuid.UUID, status string, data map[string]interface{}) error

    // Messages
    AddMessage(ctx context.Context, sessionID uuid.UUID, message *SessionMessage) error
    GetMessages(ctx context.Context, sessionID uuid.UUID, since int) ([]*SessionMessage, error)

    // Checkpoint management
    SaveCheckpoint(ctx context.Context, checkpoint *Checkpoint) error
    GetLatestCheckpoint(ctx context.Context, threadID string) (*Checkpoint, error)
    ListCheckpoints(ctx context.Context, threadID string) ([]*Checkpoint, error)
}
```

---

### 3. LangGraph Base Runner Image

**Directory Structure**:

```
components/runners/ambient-langgraph-runner/
├── Dockerfile
├── pyproject.toml
├── README.md
├── src/
│   └── ambient_langgraph_runner/
│       ├── __init__.py
│       ├── adapter.py              # Main entry point
│       ├── platform_client.py      # Backend API client
│       ├── websocket_transport.py  # WebSocket messaging
│       ├── checkpointer.py         # PostgreSQL checkpointer wrapper
│       ├── loader.py               # User workflow discovery/loading
│       └── context.py              # Runtime context (env vars, config)
└── examples/
    └── simple_workflow/
        ├── workflow.py
        └── Dockerfile
```

**Base Image Dockerfile**:

```dockerfile
# Multi-stage build for minimal image size
FROM python:3.11-slim AS builder

WORKDIR /build

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY pyproject.toml .
RUN pip install --no-cache-dir --user \
    langgraph>=0.2.0 \
    langchain>=0.3.0 \
    psycopg2-binary>=2.9.0 \
    websockets>=12.0 \
    httpx>=0.27.0 \
    pydantic>=2.0.0

# Runtime stage
FROM python:3.11-slim

WORKDIR /runner

# Copy Python packages from builder
COPY --from=builder /root/.local /root/.local
ENV PATH=/root/.local/bin:$PATH

# Copy runner code
COPY src/ambient_langgraph_runner /runner/ambient_langgraph_runner

# Create mount points for user workflows
RUN mkdir -p /workflow

# Non-root user for security
RUN useradd -m -u 1000 runner && chown -R runner:runner /runner /workflow
USER runner

# Health check endpoint (optional)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python -c "import sys; sys.exit(0)"

ENTRYPOINT ["python", "-m", "ambient_langgraph_runner.adapter"]
```

**Adapter Implementation** (adapter.py):

```python
import asyncio
import json
import os
import sys
from typing import Any, Dict, Optional
from uuid import UUID

from langgraph.checkpoint.postgres import PostgresSaver
from langgraph.graph import StateGraph

from .platform_client import PlatformClient
from .websocket_transport import WebSocketTransport
from .loader import WorkflowLoader
from .context import RuntimeContext


class LangGraphAdapter:
    """
    Base adapter that wraps user's LangGraph workflow and integrates
    with the Ambient Code Platform.
    """

    def __init__(self):
        self.context = RuntimeContext.from_environment()
        self.client = PlatformClient(
            base_url=self.context.backend_url,
            token=self.context.bot_token,
            session_id=self.context.session_id
        )
        self.websocket: Optional[WebSocketTransport] = None
        self.graph: Optional[StateGraph] = None
        self.checkpointer: Optional[PostgresSaver] = None

    async def initialize(self):
        """Initialize platform connections and load user workflow."""
        try:
            # Connect to WebSocket for real-time updates
            self.websocket = WebSocketTransport(self.context.websocket_url)
            await self.websocket.connect()
            await self._send_message("system.message", {
                "text": "Initializing workflow runner..."
            })

            # Initialize PostgreSQL checkpointer
            self.checkpointer = PostgresSaver(
                connection_string=self.context.checkpoint_db_connection
            )
            await self.checkpointer.setup()

            # Load user's workflow
            loader = WorkflowLoader(workflow_path="/workflow")
            self.graph = await loader.load_graph()

            await self._send_message("system.message", {
                "text": f"Loaded workflow: {self.graph.name}"
            })

        except Exception as e:
            await self._send_message("system.message", {
                "text": f"Initialization failed: {str(e)}",
                "level": "error"
            })
            raise

    async def run(self) -> Dict[str, Any]:
        """Execute the workflow with user inputs."""
        try:
            # Update session status to running
            await self.client.update_status({
                "status": "running",
                "started_at": self.context.current_timestamp()
            })

            # Compile graph with checkpointer
            compiled_graph = self.graph.compile(
                checkpointer=self.checkpointer,
                interrupt_before=self.graph.interrupt_nodes if hasattr(self.graph, 'interrupt_nodes') else None
            )

            # Prepare initial state from input_data
            initial_state = self.context.input_data.copy()

            # Configure thread for persistence
            config = {
                "configurable": {
                    "thread_id": self.context.thread_id,
                    "checkpoint_ns": f"{self.context.checkpoint_scope}:{self.context.checkpoint_scope_id}"
                }
            }

            # Execute graph with streaming
            result = None
            async for event in compiled_graph.astream(initial_state, config=config):
                await self._handle_graph_event(event)
                result = event

            # Check if workflow is interrupted (waiting for user input)
            state = await compiled_graph.aget_state(config)
            if state.next:  # Has pending nodes (interrupted)
                await self.client.update_status({
                    "status": "waiting_for_input",
                    "checkpoint_id": state.checkpoint_id,
                    "pending_nodes": state.next
                })
                await self._send_message("agent.waiting", {
                    "text": "Workflow paused. Waiting for user input...",
                    "pending_nodes": state.next
                })
                return {"status": "interrupted", "checkpoint_id": state.checkpoint_id}

            # Workflow completed successfully
            await self.client.update_status({
                "status": "completed",
                "completed_at": self.context.current_timestamp(),
                "output_data": result,
                "execution_duration_seconds": self.context.elapsed_seconds()
            }, blocking=True)

            await self._send_message("system.message", {
                "text": "Workflow completed successfully.",
                "level": "success"
            })

            return {"status": "completed", "result": result}

        except Exception as e:
            # Workflow failed
            error_message = str(e)
            await self.client.update_status({
                "status": "failed",
                "completed_at": self.context.current_timestamp(),
                "error_message": error_message
            }, blocking=True)

            await self._send_message("system.message", {
                "text": f"Workflow failed: {error_message}",
                "level": "error"
            })

            return {"status": "failed", "error": error_message}

    async def _handle_graph_event(self, event: Dict[str, Any]):
        """Process and forward graph execution events to UI."""
        # Extract event type and node info
        node_name = event.get("node", "unknown")
        node_output = event.get("output", {})

        # Send message to UI
        await self._send_message("agent.message", {
            "node": node_name,
            "output": node_output,
            "timestamp": self.context.current_timestamp()
        })

    async def _send_message(self, message_type: str, payload: Dict[str, Any]):
        """Send message via WebSocket."""
        if self.websocket:
            await self.websocket.send({
                "type": message_type,
                "timestamp": self.context.current_timestamp(),
                "payload": payload
            })

    async def cleanup(self):
        """Cleanup resources."""
        if self.websocket:
            await self.websocket.close()
        if self.checkpointer:
            await self.checkpointer.close()


async def main():
    """Main entry point for the adapter."""
    adapter = LangGraphAdapter()

    try:
        await adapter.initialize()
        result = await adapter.run()

        # Exit with appropriate code
        sys.exit(0 if result.get("status") in ("completed", "interrupted") else 1)

    except Exception as e:
        print(f"Fatal error: {e}", file=sys.stderr)
        sys.exit(1)

    finally:
        await adapter.cleanup()


if __name__ == "__main__":
    asyncio.run(main())
```

**User Workflow Template** (examples/simple_workflow/workflow.py):

```python
from typing import TypedDict
from langgraph.graph import StateGraph, END


class WorkflowState(TypedDict):
    """Define your workflow state."""
    input_text: str
    processed_text: str
    result: str


def process_node(state: WorkflowState) -> WorkflowState:
    """Example processing node."""
    state["processed_text"] = state["input_text"].upper()
    return state


def finalize_node(state: WorkflowState) -> WorkflowState:
    """Example finalization node."""
    state["result"] = f"Processed: {state['processed_text']}"
    return state


# Build the graph (this is what the adapter will load)
workflow = StateGraph(WorkflowState)

workflow.add_node("process", process_node)
workflow.add_node("finalize", finalize_node)

workflow.set_entry_point("process")
workflow.add_edge("process", "finalize")
workflow.add_edge("finalize", END)

# Export the graph (adapter will look for this)
graph = workflow
```

**User Dockerfile** (examples/simple_workflow/Dockerfile):

```dockerfile
FROM quay.io/ambient_code/ambient_langgraph_runner:v1.0.0

# Copy user's workflow code
COPY workflow.py /workflow/

# Install any additional dependencies (optional)
# RUN pip install pandas matplotlib

# Base image entrypoint will automatically load /workflow/workflow.py
```

---

### 4. Backend API Design

**Package Structure**:

```
components/backend/
├── handlers/
│   ├── workflows.go              # WorkflowDefinition CRUD
│   ├── workflow_sessions.go      # Workflow session management
│   ├── workflow_execution.go     # Job creation and monitoring
│   └── workflow_websocket.go     # WebSocket for real-time updates
├── repositories/
│   ├── workflow_session_repo.go  # PostgreSQL access
│   ├── checkpoint_repo.go        # Checkpoint storage
│   └── message_repo.go           # Message history
├── services/
│   ├── workflow_registry.go      # Workflow validation and registry
│   ├── job_manager.go            # Kubernetes Job creation
│   └── checkpoint_service.go     # LangGraph checkpoint management
├── models/
│   ├── workflow_session.go       # Domain models
│   └── checkpoint.go
└── database/
    ├── connection.go              # PostgreSQL connection pool
    └── migrations/                # SQL migration files
```

**API Endpoint Definitions**:

```go
// handlers/workflows.go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

// POST /api/workflows
func RegisterWorkflow(c *gin.Context) {
    var req RegisterWorkflowRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Validate user is cluster-admin
    if !isClusterAdmin(c) {
        c.JSON(http.StatusForbidden, gin.H{"error": "Cluster-admin role required"})
        return
    }

    // Validate image registry is whitelisted
    if !isRegistryAllowed(req.Image) {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "Registry not whitelisted",
            "allowed_registries": getAllowedRegistries(),
        })
        return
    }

    // Validate JSON Schema
    if err := validateJSONSchema(req.InputSchema); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid input schema: %v", err)})
        return
    }

    // Create WorkflowDefinition CR
    wfd := buildWorkflowDefinitionCR(req)
    created, err := DynamicClient.Resource(workflowGVR).Create(ctx, wfd, v1.CreateOptions{})
    if err != nil {
        log.Printf("Failed to create WorkflowDefinition: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register workflow"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "message": "Workflow registered successfully",
        "name": created.GetName(),
    })
}

// GET /api/workflows
func ListWorkflows(c *gin.Context) {
    list, err := DynamicClient.Resource(workflowGVR).List(ctx, v1.ListOptions{})
    if err != nil {
        log.Printf("Failed to list workflows: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list workflows"})
        return
    }

    workflows := []WorkflowSummary{}
    for _, item := range list.Items {
        workflows = append(workflows, mapToWorkflowSummary(&item))
    }

    c.JSON(http.StatusOK, gin.H{"items": workflows})
}

// POST /api/workflows/:name/validate
func ValidateWorkflow(c *gin.Context) {
    name := c.Param("name")

    wfd, err := DynamicClient.Resource(workflowGVR).Get(ctx, name, v1.GetOptions{})
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
        return
    }

    image, _, _ := unstructured.NestedString(wfd.Object, "spec", "image")

    // Attempt to pull image (validate it exists and is accessible)
    if err := validateImagePullable(image); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "valid": false,
            "error": fmt.Sprintf("Image validation failed: %v", err),
        })
        return
    }

    // Update status
    updateWorkflowStatus(name, map[string]interface{}{
        "validated": true,
        "lastValidated": time.Now().Format(time.RFC3339),
    })

    c.JSON(http.StatusOK, gin.H{"valid": true})
}

// handlers/workflow_sessions.go

// POST /api/v2/projects/:project/workflow-sessions
func CreateWorkflowSession(c *gin.Context) {
    project := c.Param("project")

    var req CreateWorkflowSessionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Get user-scoped K8s client
    reqK8s, _ := GetK8sClientsForRequest(c)
    if reqK8s == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
        return
    }

    // Validate user has edit/admin permission on project
    if !hasProjectPermission(reqK8s, project, "edit") {
        c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
        return
    }

    // Validate workflow definition exists
    wfd, err := DynamicClient.Resource(workflowGVR).Get(ctx, req.WorkflowName, v1.GetOptions{})
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
        return
    }

    // Extract workflow image and schema
    image, _, _ := unstructured.NestedString(wfd.Object, "spec", "image")
    schema, _, _ := unstructured.NestedMap(wfd.Object, "spec", "inputSchema")

    // Validate input data against schema
    if err := validateInputAgainstSchema(req.InputData, schema); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid input: %v", err)})
        return
    }

    // Create database record
    session := &models.WorkflowSession{
        ProjectName:            project,
        WorkflowDefinitionName: req.WorkflowName,
        WorkflowImage:          image,
        Status:                 "pending",
        CreatedByUser:          getUserEmail(c),
        InputData:              req.InputData,
        DisplayName:            req.DisplayName,
        ThreadID:               generateThreadID(),
    }

    if err := workflowRepo.Create(ctx, session); err != nil {
        log.Printf("Failed to create session in database: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
        return
    }

    // Create Kubernetes Job
    jobName := fmt.Sprintf("workflow-%s", session.ID.String()[:8])
    job := buildWorkflowJob(session, jobName, project)

    _, err = K8sClient.BatchV1().Jobs(project).Create(ctx, job, v1.CreateOptions{})
    if err != nil {
        log.Printf("Failed to create Job: %v", err)
        workflowRepo.UpdateStatus(ctx, session.ID, "failed", map[string]interface{}{
            "error_message": fmt.Sprintf("Failed to create Job: %v", err),
        })
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start workflow"})
        return
    }

    // Update database with job name and status
    workflowRepo.Update(ctx, session.ID, map[string]interface{}{
        "job_name": jobName,
        "status":   "running",
    })

    // Start background monitoring
    go monitorWorkflowJob(session.ID, jobName, project)

    c.JSON(http.StatusCreated, gin.H{
        "id":          session.ID,
        "job_name":    jobName,
        "status":      "running",
        "ws_url":      fmt.Sprintf("/api/v2/projects/%s/workflow-sessions/%s/ws", project, session.ID),
    })
}

// PUT /api/v2/workflow-sessions/:id/status (called by runner)
func UpdateWorkflowSessionStatus(c *gin.Context) {
    sessionID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
        return
    }

    // Validate BOT_TOKEN (runner authentication)
    token := extractBearerToken(c)
    if !isValidBotToken(token) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid bot token"})
        return
    }

    var updates map[string]interface{}
    if err := c.ShouldBindJSON(&updates); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Update database
    if err := workflowRepo.UpdateStatus(ctx, sessionID, updates["status"].(string), updates); err != nil {
        log.Printf("Failed to update session status: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
        return
    }

    // Broadcast update to WebSocket clients
    broadcastToWebSocket(sessionID, map[string]interface{}{
        "type": "status.update",
        "payload": updates,
    })

    c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}

// WebSocket handler
func WorkflowSessionWebSocket(c *gin.Context) {
    sessionID, _ := uuid.Parse(c.Param("id"))

    // Upgrade connection
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Printf("WebSocket upgrade failed: %v", err)
        return
    }
    defer conn.Close()

    // Register client
    clientID := registerWebSocketClient(sessionID, conn)
    defer unregisterWebSocketClient(sessionID, clientID)

    // Send initial session state
    session, _ := workflowRepo.GetByID(ctx, sessionID)
    conn.WriteJSON(map[string]interface{}{
        "type": "session.state",
        "payload": session,
    })

    // Listen for messages (e.g., user responses to interrupts)
    for {
        var msg map[string]interface{}
        if err := conn.ReadJSON(&msg); err != nil {
            break
        }

        handleWebSocketMessage(sessionID, msg)
    }
}
```

**Job Creation Logic**:

```go
// services/job_manager.go

func buildWorkflowJob(session *models.WorkflowSession, jobName, namespace string) *batchv1.Job {
    backoffLimit := int32(0) // No retries
    ttlSecondsAfterFinished := int32(86400) // Keep for 24 hours

    return &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      jobName,
            Namespace: namespace,
            Labels: map[string]string{
                "app":                       "workflow-session",
                "session-id":                session.ID.String(),
                "workflow":                  session.WorkflowDefinitionName,
                "ambient-code.io/component": "workflow-runner",
            },
            Annotations: map[string]string{
                "ambient-code.io/created-by": session.CreatedByUser,
            },
        },
        Spec: batchv1.JobSpec{
            BackoffLimit:            &backoffLimit,
            TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app":        "workflow-session",
                        "session-id": session.ID.String(),
                    },
                },
                Spec: corev1.PodSpec{
                    RestartPolicy: corev1.RestartPolicyNever,
                    Containers: []corev1.Container{
                        {
                            Name:  "workflow-runner",
                            Image: session.WorkflowImage,
                            Env:   buildEnvironmentVariables(session),
                            Resources: corev1.ResourceRequirements{
                                Limits: corev1.ResourceList{
                                    corev1.ResourceCPU:    resource.MustParse("2"),
                                    corev1.ResourceMemory: resource.MustParse("4Gi"),
                                },
                                Requests: corev1.ResourceList{
                                    corev1.ResourceCPU:    resource.MustParse("500m"),
                                    corev1.ResourceMemory: resource.MustParse("1Gi"),
                                },
                            },
                            SecurityContext: &corev1.SecurityContext{
                                AllowPrivilegeEscalation: boolPtr(false),
                                ReadOnlyRootFilesystem:   boolPtr(false),
                                RunAsNonRoot:             boolPtr(true),
                                RunAsUser:                int64Ptr(1000),
                                Capabilities: &corev1.Capabilities{
                                    Drop: []corev1.Capability{"ALL"},
                                },
                            },
                        },
                    },
                    SecurityContext: &corev1.PodSecurityContext{
                        FSGroup: int64Ptr(1000),
                    },
                },
            },
        },
    }
}

func buildEnvironmentVariables(session *models.WorkflowSession) []corev1.EnvVar {
    inputDataJSON, _ := json.Marshal(session.InputData)

    return []corev1.EnvVar{
        // Session identification
        {Name: "SESSION_ID", Value: session.ID.String()},
        {Name: "PROJECT_NAME", Value: session.ProjectName},
        {Name: "WORKFLOW_NAME", Value: session.WorkflowDefinitionName},

        // Workflow inputs
        {Name: "INPUT_DATA", Value: string(inputDataJSON)},

        // Platform connectivity
        {Name: "BACKEND_API_URL", Value: fmt.Sprintf("http://backend-service.%s.svc.cluster.local:8080", getBackendNamespace())},
        {Name: "WEBSOCKET_URL", Value: fmt.Sprintf("ws://backend-service.%s.svc.cluster.local:8080/api/v2/projects/%s/workflow-sessions/%s/ws",
            getBackendNamespace(), session.ProjectName, session.ID)},
        {
            Name: "BOT_TOKEN",
            ValueFrom: &corev1.EnvVarSource{
                SecretKeyRef: &corev1.SecretKeySelector{
                    LocalObjectReference: corev1.LocalObjectReference{Name: "workflow-runner-token"},
                    Key:                  "token",
                },
            },
        },

        // LangGraph persistence
        {Name: "THREAD_ID", Value: session.ThreadID},
        {Name: "CHECKPOINT_DB_CONNECTION", Value: getCheckpointDBConnection()},
        {Name: "CHECKPOINT_SCOPE", Value: "project"},
        {Name: "CHECKPOINT_SCOPE_ID", Value: session.ProjectName},

        // Execution settings
        {Name: "TIMEOUT", Value: "3600"},
    }
}
```

---

## Security Architecture

### Registry Whitelist

**Configuration** (environment variable or ConfigMap):

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-registry-config
  namespace: ambient-code
data:
  allowed_registries: |
    quay.io/ambient_code
    quay.io/approved-org
    gcr.io/company-prod
    docker.io/company
```

**Validation Logic**:

```go
func isRegistryAllowed(image string) bool {
    allowedRegistries := getAllowedRegistries() // From ConfigMap or env var

    for _, registry := range allowedRegistries {
        if strings.HasPrefix(image, registry) {
            return true
        }
    }

    return false
}
```

### Authentication & Authorization

**Runner Authentication** (BOT_TOKEN):

```go
// Generated per-session service account token
func mintRunnerToken(sessionID uuid.UUID, namespace string) (string, error) {
    // Create service account (if not exists)
    sa := &corev1.ServiceAccount{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "workflow-runner",
            Namespace: namespace,
        },
    }
    _, err := K8sClient.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, v1.CreateOptions{})
    if err != nil && !errors.IsAlreadyExists(err) {
        return "", err
    }

    // Create token secret
    secret := &corev1.Secret{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("workflow-runner-token-%s", sessionID.String()[:8]),
            Namespace: namespace,
            Annotations: map[string]string{
                "kubernetes.io/service-account.name": "workflow-runner",
            },
        },
        Type: corev1.SecretTypeServiceAccountToken,
    }

    created, err := K8sClient.CoreV1().Secrets(namespace).Create(ctx, secret, v1.CreateOptions{})
    if err != nil {
        return "", err
    }

    // Wait for token to be populated
    time.Sleep(2 * time.Second)

    tokenSecret, _ := K8sClient.CoreV1().Secrets(namespace).Get(ctx, created.Name, v1.GetOptions{})
    token := string(tokenSecret.Data["token"])

    return token, nil
}
```

### Pod Security

**SecurityContext** (applied to all workflow runner pods):

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: false  # Workflows may need temp files
  capabilities:
    drop:
      - ALL
```

**NetworkPolicy** (restrict egress - future enhancement):

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: workflow-runner-netpol
  namespace: <project-namespace>
spec:
  podSelector:
    matchLabels:
      app: workflow-session
  policyTypes:
    - Egress
  egress:
    # Allow DNS
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
      ports:
        - protocol: UDP
          port: 53
    # Allow backend API
    - to:
        - namespaceSelector:
            matchLabels:
              name: ambient-code
      ports:
        - protocol: TCP
          port: 8080
    # Allow PostgreSQL (checkpoints)
    - to:
        - namespaceSelector:
            matchLabels:
              name: ambient-code
      ports:
        - protocol: TCP
          port: 5432
```

---

## Sequence Diagrams

### Workflow Registration Flow

```
┌─────┐      ┌─────────┐      ┌─────────┐      ┌──────────┐
│User │      │Frontend │      │ Backend │      │K8s API   │
└──┬──┘      └────┬────┘      └────┬────┘      └────┬─────┘
   │              │                 │                │
   │ Navigate to  │                 │                │
   │ /workflows   │                 │                │
   ├─────────────>│                 │                │
   │              │ GET /api/       │                │
   │              │ workflows       │                │
   │              ├────────────────>│                │
   │              │                 │ List           │
   │              │                 │ WorkflowDef CRs│
   │              │                 ├───────────────>│
   │              │                 │<───────────────┤
   │              │<────────────────┤                │
   │              │ [Workflow List] │                │
   │<─────────────┤                 │                │
   │              │                 │                │
   │ Click        │                 │                │
   │ "Register"   │                 │                │
   ├─────────────>│                 │                │
   │              │ [Modal opens]   │                │
   │              │                 │                │
   │ Fill form &  │                 │                │
   │ submit       │                 │                │
   ├─────────────>│                 │                │
   │              │ POST /api/      │                │
   │              │ workflows       │                │
   │              │ {name, image,   │                │
   │              │  inputSchema}   │                │
   │              ├────────────────>│                │
   │              │                 │ Validate       │
   │              │                 │ cluster-admin  │
   │              │                 │ role           │
   │              │                 │                │
   │              │                 │ Validate       │
   │              │                 │ registry       │
   │              │                 │ whitelist      │
   │              │                 │                │
   │              │                 │ Create         │
   │              │                 │ WorkflowDef CR │
   │              │                 ├───────────────>│
   │              │                 │<───────────────┤
   │              │<────────────────┤                │
   │              │ {success}       │                │
   │<─────────────┤                 │                │
   │ [Workflow    │                 │                │
   │  registered] │                 │                │
   │              │                 │                │
```

### Workflow Session Creation & Execution Flow

```
┌─────┐  ┌─────────┐  ┌─────────┐  ┌──────┐  ┌──────────┐  ┌─────────┐
│User │  │Frontend │  │ Backend │  │  DB  │  │K8s API   │  │ Runner  │
└──┬──┘  └────┬────┘  └────┬────┘  └───┬──┘  └────┬─────┘  └────┬────┘
   │          │             │           │          │             │
   │ Select   │             │           │          │             │
   │ workflow │             │           │          │             │
   ├─────────>│             │           │          │             │
   │          │ [Dynamic    │           │          │             │
   │          │  form       │           │          │             │
   │          │  renders]   │           │          │             │
   │          │             │           │          │             │
   │ Fill     │             │           │          │             │
   │ inputs & │             │           │          │             │
   │ submit   │             │           │          │             │
   ├─────────>│             │           │          │             │
   │          │ POST        │           │          │             │
   │          │ /workflow-  │           │          │             │
   │          │ sessions    │           │          │             │
   │          ├────────────>│           │          │             │
   │          │             │ Validate  │          │             │
   │          │             │ RBAC      │          │             │
   │          │             │           │          │             │
   │          │             │ INSERT    │          │             │
   │          │             │ session   │          │             │
   │          │             ├──────────>│          │             │
   │          │             │<──────────┤          │             │
   │          │             │ [session  │          │             │
   │          │             │  ID]      │          │             │
   │          │             │           │          │             │
   │          │             │ Create    │          │             │
   │          │             │ Job       │          │             │
   │          │             ├─────────────────────>│             │
   │          │             │<─────────────────────┤             │
   │          │             │           │          │             │
   │          │             │ UPDATE    │          │             │
   │          │             │ status=   │          │             │
   │          │             │ running   │          │             │
   │          │             ├──────────>│          │             │
   │          │             │           │          │             │
   │          │<────────────┤           │          │             │
   │          │ {sessionID, │           │          │             │
   │          │  ws_url}    │           │          │             │
   │<─────────┤             │           │          │             │
   │          │             │           │          │             │
   │ Redirect │             │           │          │   Pod       │
   │ to       │             │           │          │   starts    │
   │ session  │             │           │          ├────────────>│
   │ page     │             │           │          │             │
   ├─────────>│             │           │          │             │
   │          │ Establish   │           │          │             │
   │          │ WebSocket   │           │          │             │
   │          ├────────────>│           │          │             │
   │          │<────────────┤           │          │             │
   │          │             │           │          │ Initialize  │
   │          │             │           │          │ adapter     │
   │          │             │           │          │             │
   │          │             │           │          │ Connect WS  │
   │          │             │<──────────────────────────────────┤
   │          │<────────────┤           │          │             │
   │ [Status: │             │           │          │             │
   │  Running]│             │           │          │ Execute     │
   │          │             │           │          │ graph       │
   │          │             │           │          │             │
   │          │             │           │          │ Stream      │
   │          │             │           │          │ progress    │
   │          │             │<──────────────────────────────────┤
   │          │<────────────┤           │          │             │
   │<─────────┤             │           │          │             │
   │ [Progress│             │           │          │             │
   │  msgs]   │             │           │          │             │
   │          │             │           │          │             │
   │          │             │           │          │ Interrupt   │
   │          │             │           │          │ reached     │
   │          │             │           │          │             │
   │          │             │           │          │ PUT /status │
   │          │             │<──────────────────────────────────┤
   │          │             │ UPDATE    │          │             │
   │          │             │ status=   │          │             │
   │          │             │ waiting   │          │             │
   │          │             ├──────────>│          │             │
   │          │             │           │          │             │
   │          │<────────────┤           │          │             │
   │<─────────┤             │           │          │             │
   │ [Status: │             │           │          │             │
   │  Waiting]│             │           │          │             │
   │          │             │           │          │             │
   │ User     │             │           │          │             │
   │ responds │             │           │          │             │
   ├─────────>│             │           │          │             │
   │          │ POST /      │           │          │             │
   │          │ respond     │           │          │             │
   │          ├────────────>│           │          │             │
   │          │             │ UPDATE    │          │             │
   │          │             │ response  │          │             │
   │          │             ├──────────>│          │             │
   │          │             │           │          │             │
   │          │             │ Notify    │          │             │
   │          │             │ runner    │          │             │
   │          │             ├─────────────────────────────────────>│
   │          │             │           │          │ Resume      │
   │          │             │           │          │ execution   │
   │          │             │           │          │             │
   │          │             │           │          │ Complete    │
   │          │             │           │          │             │
   │          │             │           │          │ PUT /status │
   │          │             │<──────────────────────────────────┤
   │          │             │ UPDATE    │          │             │
   │          │             │ status=   │          │             │
   │          │             │ completed │          │             │
   │          │             ├──────────>│          │             │
   │          │             │           │          │             │
   │          │<────────────┤           │          │             │
   │<─────────┤             │           │          │             │
   │ [Status: │             │           │          │             │
   │ Completed│             │           │          │             │
   │ + output]│             │           │          │             │
   │          │             │           │          │             │
```

---

## Integration Patterns

### LangGraph Persistence Integration

**Checkpointer Configuration**:

```python
# In base runner adapter
from langgraph.checkpoint.postgres import PostgresSaver

# Initialize checkpointer
checkpointer = PostgresSaver(
    connection_string=os.environ["CHECKPOINT_DB_CONNECTION"],
    # Custom table name with scope prefix
    table_name=f"checkpoints_{os.environ['CHECKPOINT_SCOPE']}_{os.environ['CHECKPOINT_SCOPE_ID']}"
)

await checkpointer.setup()

# Compile graph with checkpointer
compiled = graph.compile(checkpointer=checkpointer)

# Execute with thread config
config = {
    "configurable": {
        "thread_id": os.environ["THREAD_ID"],
        "checkpoint_ns": f"{os.environ['CHECKPOINT_SCOPE']}:{os.environ['CHECKPOINT_SCOPE_ID']}"
    }
}

async for event in compiled.astream(inputs, config=config):
    # Process events
    pass

# Get state for continuation
state = await compiled.aget_state(config)
if state.next:  # Has pending nodes (interrupted)
    # Save checkpoint ID for resumption
    checkpoint_id = state.checkpoint_id
```

### Human-in-the-Loop (Interrupt) Pattern

**Workflow Definition** (user's code):

```python
from langgraph.graph import StateGraph, END

workflow = StateGraph(MyState)

# Add interrupt before approval node
workflow.add_node("generate_report", generate_report_node)
workflow.add_node("review_report", review_report_node, interrupt="before")
workflow.add_node("publish_report", publish_report_node)

workflow.add_edge("generate_report", "review_report")
workflow.add_edge("review_report", "publish_report")
workflow.add_edge("publish_report", END)
```

**Runner Handling**:

```python
# In adapter.py
async for event in compiled.astream(inputs, config=config):
    await self._handle_graph_event(event)

# Check if interrupted
state = await compiled.aget_state(config)
if state.next:  # Interrupted
    await self.client.update_status({
        "status": "waiting_for_input",
        "checkpoint_id": state.checkpoint_id,
        "pending_nodes": state.next,
        "prompt": "Please review the generated report and approve or reject."
    })

    # Wait for user response (via WebSocket or database polling)
    response = await self._wait_for_user_input()

    # Resume with response
    async for event in compiled.astream(
        {"user_response": response},
        config=config
    ):
        await self._handle_graph_event(event)
```

**UI Handling**:

```typescript
// When status becomes waiting_for_input
if (session.status === 'waiting_for_input') {
  setShowResponseForm(true);
}

// User submits response
const handleRespond = async (response: string) => {
  await api.respondToWorkflowInterrupt(session.id, { response });
  // Runner resumes automatically
};
```

---

## Migration Strategy

### Phase 1: MVP (Coexistence)

**Goals**:
- Launch LangGraph workflow support without disrupting existing users
- Validate database-backed architecture
- Gather user feedback

**Implementation**:
- Legacy system: `AgenticSession` CR → Operator → Claude Code Job (unchanged)
- New system: `workflow_sessions` table → Backend → LangGraph Job (parallel)
- UI: Separate routes and forms for each system
- Users explicitly choose: "Claude Code Session" vs. "Workflow Session"

**Duration**: 3-6 months (MVP + iteration based on feedback)

---

### Phase 2: Claude Code Migration (Future)

**Goals**:
- Migrate existing Claude Code sessions to database-backed model
- Unified orchestration approach
- Maintain backward compatibility

**Implementation**:
1. Create new `claude_code_sessions` table (similar schema to `workflow_sessions`)
2. Update Backend to write both CR and DB record (dual-write)
3. Operator continues watching CRs (no change initially)
4. Frontend reads from database instead of CRs
5. Gradual migration of existing sessions (export CR → import to DB)
6. Monitor for issues, rollback if needed

**Duration**: 2-3 months

---

### Phase 3: Unified Orchestration (Future)

**Goals**:
- Single session management model
- Deprecate AgenticSession CRs
- Operator watches database instead of K8s CRs

**Implementation**:
1. Merge `workflow_sessions` and `claude_code_sessions` into unified `sessions` table
2. Add `runner_type` discriminator field (`claude-code`, `langgraph-workflow`, etc.)
3. Backend API unifies endpoints: `/api/v2/projects/:project/sessions`
4. Operator polls database for pending sessions (instead of watching CRs)
5. Deprecate AgenticSession CRD (mark as legacy, announce sunset date)
6. Archive old CRs after migration complete

**Duration**: 3-4 months

---

### Phase 4: Full Database-Backed Platform (Future)

**Goals**:
- Complete transition to database-centric architecture
- Remove all CR dependencies
- Enhanced analytics and auditing capabilities

**Implementation**:
- Delete AgenticSession CRD entirely
- Operator fully database-driven
- Advanced features enabled (session analytics, cost tracking, bulk operations)
- Multi-runner support (OpenAI, Gemini, etc.) using same architecture

**Duration**: Ongoing evolution

---

## Observability & Monitoring

### Logging Strategy

**Backend Logging**:

```go
// Structured logging with context
log.Printf("[WorkflowSession] id=%s project=%s workflow=%s status=%s action=created user=%s",
    session.ID, session.ProjectName, session.WorkflowDefinitionName, session.Status, session.CreatedByUser)

log.Printf("[WorkflowJob] id=%s job=%s namespace=%s action=job_created",
    session.ID, jobName, project)

log.Printf("[WorkflowStatus] id=%s old_status=%s new_status=%s duration=%ds",
    session.ID, oldStatus, newStatus, duration)
```

**Runner Logging** (captured by K8s):

```python
import logging

logging.basicConfig(
    level=logging.INFO,
    format='[%(asctime)s] %(levelname)s [%(name)s] %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("langgraph_adapter")

logger.info(f"Session {session_id} starting execution")
logger.info(f"Loaded workflow graph: {graph.name} with {len(graph.nodes)} nodes")
logger.error(f"Execution failed: {error_message}")
```

---

### Metrics (Prometheus-compatible)

**Backend Metrics**:

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    workflowSessionsCreated = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "workflow_sessions_total",
            Help: "Total number of workflow sessions created",
        },
        []string{"project", "workflow", "status"},
    )

    workflowExecutionDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "workflow_execution_duration_seconds",
            Help:    "Workflow execution duration in seconds",
            Buckets: []float64{10, 30, 60, 300, 600, 1800, 3600},
        },
        []string{"project", "workflow"},
    )

    activeWorkflowSessions = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "workflow_sessions_active",
            Help: "Number of currently running workflow sessions",
        },
        []string{"project", "workflow"},
    )
)

// Record metrics
workflowSessionsCreated.WithLabelValues(project, workflowName, "created").Inc()
workflowExecutionDuration.WithLabelValues(project, workflowName).Observe(duration)
activeWorkflowSessions.WithLabelValues(project, workflowName).Set(count)
```

---

### Health Checks

**Backend Health Endpoint**:

```go
// GET /health
func HealthCheck(c *gin.Context) {
    health := map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now().Format(time.RFC3339),
        "components": map[string]interface{}{
            "database": checkDatabaseHealth(),
            "kubernetes": checkKubernetesHealth(),
        },
    }

    if !allHealthy(health["components"]) {
        c.JSON(http.StatusServiceUnavailable, health)
        return
    }

    c.JSON(http.StatusOK, health)
}

func checkDatabaseHealth() map[string]interface{} {
    err := db.Ping()
    if err != nil {
        return map[string]interface{}{"status": "unhealthy", "error": err.Error()}
    }
    return map[string]interface{}{"status": "healthy"}
}
```

---

## Performance Considerations

### Database Optimization

**Connection Pooling**:

```go
// Backend database config
db, err := sql.Open("postgres", connectionString)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

**Query Optimization**:

```sql
-- Use indexes for common queries
CREATE INDEX idx_workflow_sessions_project_status
ON workflow_sessions(project_name, status)
WHERE deleted_at IS NULL;

-- Partial index for active sessions
CREATE INDEX idx_workflow_sessions_active
ON workflow_sessions(status, created_at DESC)
WHERE status IN ('pending', 'running', 'waiting_for_input');
```

**Checkpoint Cleanup**:

```sql
-- Periodic cleanup of old checkpoints (retention policy)
DELETE FROM langgraph_checkpoints
WHERE created_at < NOW() - INTERVAL '90 days'
AND thread_id NOT IN (
    SELECT DISTINCT thread_id
    FROM workflow_sessions
    WHERE status IN ('running', 'waiting_for_input')
);
```

---

### WebSocket Scalability

**Challenge**: Multiple Backend replicas require coordinated WebSocket message broadcasting.

**Solution Options**:

1. **Redis Pub/Sub** (recommended):
   ```go
   // Backend subscribes to Redis channel per session
   pubsub := redisClient.Subscribe(ctx, fmt.Sprintf("session:%s", sessionID))

   // When runner updates status, Backend publishes to Redis
   redisClient.Publish(ctx, fmt.Sprintf("session:%s", sessionID), message)

   // All Backend replicas receive and forward to connected WebSocket clients
   ```

2. **Sticky Sessions** (simpler but less resilient):
   ```yaml
   # Ingress annotation
   nginx.ingress.kubernetes.io/affinity: "cookie"
   nginx.ingress.kubernetes.io/session-cookie-name: "backend-affinity"
   ```

---

### Job Cleanup

**TTL for Completed Jobs**:

```go
// In Job spec
TTLSecondsAfterFinished: int32Ptr(86400)  // 24 hours
```

**Manual Cleanup** (CronJob):

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: workflow-job-cleanup
  namespace: ambient-code
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: workflow-cleanup
          containers:
          - name: cleanup
            image: bitnami/kubectl:latest
            command:
            - /bin/sh
            - -c
            - |
              kubectl delete jobs -l app=workflow-session \
                --field-selector status.successful=1 \
                --all-namespaces \
                --ignore-not-found
```

---

## Deployment Architecture

### Kubernetes Resources

**Namespace Structure**:

```
ambient-code (platform namespace)
  ├── backend (Deployment)
  ├── postgres (StatefulSet)
  ├── redis (StatefulSet)
  └── workflow-operator (Deployment) [future]

project-a (tenant namespace)
  └── workflow-session-jobs (Jobs)

project-b (tenant namespace)
  └── workflow-session-jobs (Jobs)
```

**Resource Manifests**:

```yaml
# PostgreSQL for workflow sessions and checkpoints
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: ambient-code
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:16
        env:
        - name: POSTGRES_DB
          value: ambient_workflows
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: username
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: password
        volumeMounts:
        - name: postgres-data
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2"
  volumeClaimTemplates:
  - metadata:
      name: postgres-data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 50Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: ambient-code
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
```

---

## Summary

This architecture document provides the technical foundation for integrating LangGraph workflows into the Ambient Code Platform. The design emphasizes:

1. **Scalability**: Database-backed sessions avoid K8s etcd limitations
2. **Extensibility**: Generic interfaces support future runner types
3. **Security**: Registry whitelisting, pod security contexts, token-based auth
4. **Observability**: Comprehensive logging, metrics, and tracing
5. **User Experience**: Seamless integration via base runner image abstracting platform complexity

The phased migration strategy ensures safe rollout and allows iterative improvement based on user feedback. The dual-mode architecture (legacy CRs + new DB) provides a clear path toward a unified, database-centric orchestration platform supporting multiple AI agent frameworks.
