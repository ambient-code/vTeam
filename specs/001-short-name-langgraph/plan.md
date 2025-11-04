# Implementation Plan: LangGraph Workflow Integration

**Branch**: `ambient-langgraph-integration` | **Date**: 2025-11-04 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-short-name-langgraph/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → ✅ Feature spec loaded successfully
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → ✅ Project Type: Web application (backend + frontend)
   → ✅ Structure Decision: Existing component structure
3. Fill the Constitution Check section based on the content of the constitution document.
   → In progress
4. Evaluate Constitution Check section below
   → Pending after constitution check fill
5. Execute Phase 0 → research.md
   → Pending
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, CLAUDE.md
   → Pending
7. Re-evaluate Constitution Check section
   → Pending
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
   → Pending
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary

Add LangGraph as a new runner type to the Ambient Code Platform, enabling users to register and execute custom workflow definitions with interactive human-in-the-loop capabilities. The feature transforms the platform from a single-runner (Claude Code only) system into a multi-runner orchestration platform with database-backed session management for persistence and resumption.

**Technical Approach**: Extend the existing Kubernetes-native architecture by adding cluster-scoped WorkflowDefinition CRs, project-scoped WorkflowSession database records, a new LangGraph runner container, and frontend UI for workflow registration and execution. Leverage PostgreSQL for session state persistence and WebSocket for real-time messaging.

## Technical Context

**Language/Version**:
- Backend: Go 1.24.0
- Frontend: TypeScript with Next.js 15 (App Router)
- Runner: Python 3.11+ (LangGraph SDK)

**Primary Dependencies**:
- Backend: Gin (HTTP), Kubernetes client-go 0.34.0, gorilla/websocket
- Frontend: Next.js 15, React Query, Shadcn UI, Zod
- Runner: LangGraph SDK, Anthropic SDK, PostgreSQL driver (asyncpg/psycopg3)

**Storage**:
- PostgreSQL for WorkflowSession, SessionMessage, and Checkpoint entities
- Kubernetes etcd for WorkflowDefinition CRs (cluster-scoped)
- Persistent Volume Claims for workspace data (existing pattern)

**Testing**:
- Backend: Go testing framework (unit, contract, integration tests)
- Frontend: Jest for components, TypeScript strict mode
- Runner: pytest with async support
- Contract tests: OpenAPI schema validation

**Target Platform**:
- Kubernetes 1.28+ clusters (OpenShift compatible)
- PostgreSQL 14+
- Linux container runtime (OCI-compliant)

**Project Type**: Web application (backend + frontend + operator + runners)

**Performance Goals**:
- Session list page load < 2 seconds with 100+ sessions
- Real-time message delivery < 5 seconds from generation
- Workflow registration validation < 10 seconds
- Database query response < 500ms for session operations

**Constraints**:
- WebSocket connection required for workflow execution (no fallback)
- 100MB limit on output data stored in database
- Workflow containers must follow platform base image structure
- Single database instance (no distributed storage in MVP)
- Whitelisted container registries only

**Scale/Scope**:
- Support 10+ concurrent workflow executions per project
- Handle 1000+ workflow sessions per project
- 20-30 API endpoints (workflow CRUD, session management)
- 5 new frontend pages/components (registry, session list, session detail)
- 3 new database tables (WorkflowSession, SessionMessage, Checkpoint)

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Note**: The constitution template is not yet populated with project-specific principles. Using general software engineering best practices for initial evaluation.

### Preliminary Assessment

**Modularity & Separation of Concerns**:
- ✅ PASS: Feature cleanly separates cluster-scoped registry (WorkflowDefinition CR) from project-scoped execution (database)
- ✅ PASS: Runner is isolated container following existing pattern (Claude Code runner)
- ✅ PASS: Backend API maintains REST principles with clear resource boundaries

**Testing Strategy**:
- ⚠️ NEEDS VERIFICATION: Spec mentions contract tests for API endpoints
- ⚠️ NEEDS VERIFICATION: User stories define acceptance scenarios
- ⚠️ NEEDS VERIFICATION: Test-first approach not explicitly stated (will verify in research phase)

**Simplicity & YAGNI**:
- ✅ PASS: MVP explicitly excludes workflow marketplace, versioning, external observability
- ✅ PASS: Single database instance (deferred distributed storage)
- ✅ PASS: No automated workflow building (users provide containers)

**Integration & Compatibility**:
- ✅ PASS: FR-038, FR-039, FR-040 explicitly maintain backward compatibility with existing Claude Code sessions
- ✅ PASS: No breaking changes to existing APIs or CRDs

**Security**:
- ✅ PASS: Whitelisted registries only (FR-002, FR-037)
- ✅ PASS: RBAC enforced at cluster and project levels (FR-033, FR-034)
- ✅ PASS: User token validation required (FR-036)
- ✅ PASS: Non-privileged containers (constraints section)

### Gate Status: CONDITIONAL PASS
**Rationale**: Core architectural principles are sound. Need to verify TDD approach and contract test coverage during research phase. No violations requiring justification.

## Project Structure

### Documentation (this feature)
```
specs/001-short-name-langgraph/
├── spec.md              # Feature specification (existing)
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command) - TO BE CREATED
├── data-model.md        # Phase 1 output (/plan command) - TO BE CREATED
├── quickstart.md        # Phase 1 output (/plan command) - TO BE CREATED
├── contracts/           # Phase 1 output (/plan command) - TO BE CREATED
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
components/
├── backend/
│   ├── handlers/
│   │   ├── workflows.go         # NEW: WorkflowDefinition CRUD
│   │   ├── workflow_sessions.go # NEW: WorkflowSession CRUD
│   │   └── sessions.go          # MODIFY: Unified session list
│   ├── types/
│   │   ├── workflow.go          # NEW: Workflow types
│   │   └── session.go           # MODIFY: Add workflow session types
│   ├── db/
│   │   ├── postgres.go          # NEW: PostgreSQL connection
│   │   ├── workflow_sessions.go # NEW: Session CRUD
│   │   ├── checkpoints.go       # NEW: Checkpoint storage
│   │   └── migrations/          # NEW: Database migrations
│   └── k8s/
│       └── workflow_job.go      # NEW: Workflow job templates
│
├── frontend/
│   └── src/
│       ├── app/
│       │   ├── workflows/
│       │   │   ├── page.tsx              # NEW: Workflow registry list
│       │   │   ├── new/page.tsx          # NEW: Register workflow
│       │   │   └── [name]/page.tsx       # NEW: Workflow detail
│       │   └── projects/[name]/
│       │       ├── sessions/
│       │       │   └── page.tsx          # MODIFY: Unified session list
│       │       └── workflow-sessions/
│       │           ├── new/page.tsx      # NEW: Create workflow session
│       │           └── [id]/page.tsx     # NEW: Workflow session detail
│       ├── components/
│       │   ├── workflow-registry-card.tsx # NEW
│       │   ├── workflow-input-form.tsx    # NEW: Dynamic form generation
│       │   └── unified-session-list.tsx   # MODIFY: Combined view
│       └── services/
│           ├── api/
│           │   └── workflows.ts          # NEW: Workflow API client
│           └── queries/
│               └── workflows.ts          # NEW: React Query hooks
│
├── operator/
│   └── internal/
│       └── handlers/
│           └── workflow_definitions.go   # NEW: Watch WorkflowDefinitions
│
├── runners/
│   └── langgraph-runner/                # NEW: Entire runner component
│       ├── src/
│       │   ├── __init__.py
│       │   ├── __main__.py
│       │   ├── runner.py                 # Main execution loop
│       │   ├── checkpoint_manager.py     # Checkpoint persistence
│       │   ├── message_publisher.py      # WebSocket messaging
│       │   └── workflow_loader.py        # Dynamic workflow loading
│       ├── tests/
│       │   ├── unit/
│       │   └── integration/
│       ├── Dockerfile
│       ├── pyproject.toml
│       └── README.md
│
└── manifests/
    ├── crds/
    │   └── workflow-definition-crd.yaml  # NEW: WorkflowDefinition CRD
    └── base/
        └── postgres-deployment.yaml      # NEW: PostgreSQL deployment

tests/
├── backend/
│   ├── contract/
│   │   ├── workflow_api_test.go          # NEW: API contract tests
│   │   └── workflow_session_api_test.go  # NEW: Session API tests
│   └── integration/
│       └── workflow_e2e_test.go          # NEW: End-to-end tests
```

**Structure Decision**: This is a web application (backend + frontend) following the existing component-based architecture. The feature adds a new runner type (langgraph-runner) alongside the existing claude-code-runner, introduces database storage for workflow sessions, and extends both backend and frontend with new handlers and UI pages. The modular structure ensures backward compatibility while enabling multi-runner support.

## Phase 0: Outline & Research

### Unknowns from Technical Context (NEEDS CLARIFICATION)
Based on the technical context analysis, the following areas require research:

1. **LangGraph Integration Patterns**
   - How to structure LangGraph workflows for containerized execution
   - Best practices for checkpoint persistence with PostgreSQL
   - Human-in-the-loop patterns and approval gate implementation

2. **Database Schema Design**
   - PostgreSQL schema for WorkflowSession with JSONB fields (input/output)
   - Checkpoint storage format compatible with LangGraph checkpointer API
   - Indexing strategy for session queries and thread-based lookups

3. **Dynamic Form Generation**
   - JSON Schema to React form conversion (best libraries/patterns)
   - Client-side validation matching backend validation
   - Handling complex nested schemas

4. **WebSocket Real-Time Messaging**
   - Existing WebSocket implementation in backend (gorilla/websocket)
   - Message routing from runner pods to frontend clients
   - Connection lifecycle management and reconnection strategy

5. **Kubernetes Job Execution**
   - Existing job creation pattern from claude-code-runner
   - Environment variables and secrets for PostgreSQL access
   - Job monitoring and status updates to database

### Research Tasks

1. **Task: Research LangGraph checkpoint persistence with PostgreSQL**
   - Objective: Understand LangGraph checkpointer API and PostgreSQL backend implementation
   - Key questions: Schema requirements, async support, thread management
   - Deliverable: Checkpoint table schema and Python implementation approach

2. **Task: Evaluate JSON Schema form generation libraries**
   - Objective: Select library/approach for dynamic form generation from JSON Schema
   - Options: react-jsonschema-form, uniforms, custom with Zod + React Hook Form
   - Deliverable: Library choice with rationale and integration approach

3. **Task: Analyze existing runner and job patterns**
   - Objective: Extract reusable patterns from claude-code-runner for langgraph-runner
   - Key areas: Job template structure, environment config, result storage
   - Deliverable: Reusable job template and runner initialization pattern

4. **Task: Design WebSocket message routing for workflow sessions**
   - Objective: Extend existing WebSocket implementation for workflow messages
   - Key questions: Channel/topic structure, authentication, message format
   - Deliverable: Message routing architecture and protocol specification

5. **Task: Investigate PostgreSQL JSONB best practices**
   - Objective: Optimal schema design for semi-structured workflow input/output
   - Key areas: Indexing strategies, query patterns, size limits
   - Deliverable: Database schema with indexing and query patterns

**Output**: research.md with all decisions, rationales, and alternatives

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

### Planned Outputs

1. **data-model.md**: Entity definitions with:
   - WorkflowDefinition (CR spec)
   - WorkflowSession (database table)
   - SessionMessage (database table)
   - Checkpoint (database table)
   - Validation rules and state transitions

2. **contracts/** directory with:
   - `workflow-definition-api.yaml` (OpenAPI spec for registry endpoints)
   - `workflow-session-api.yaml` (OpenAPI spec for session endpoints)
   - `websocket-protocol.md` (WebSocket message format specification)
   - `checkpoint-schema.json` (Checkpoint data structure)

3. **quickstart.md**: Step-by-step guide:
   - Register a simple workflow
   - Create and execute a workflow session
   - View real-time progress
   - Resume an interrupted session

4. **CLAUDE.md updates**: Add LangGraph technology context (via update script)

### Contract Test Approach
- Generate Go contract tests from OpenAPI specs
- Test request/response schemas for all workflow endpoints
- Validate WebSocket message format
- Tests will fail initially (TDD approach)

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
1. Load `.specify/templates/tasks-template.md` as base
2. Generate from Phase 1 artifacts:
   - **Database migrations**: Create tables for WorkflowSession, SessionMessage, Checkpoint
   - **CRD definition**: WorkflowDefinition custom resource
   - **Backend API**: Workflow and session CRUD handlers
   - **Frontend pages**: Workflow registry, session management UI
   - **Runner implementation**: LangGraph execution loop, checkpoint manager
   - **Contract tests**: API endpoint validation
   - **Integration tests**: End-to-end workflow execution

**Ordering Strategy**:
1. Infrastructure first: Database migrations, CRD installation
2. TDD order: Contract tests → API handlers → Frontend
3. Runner last: Depends on API contracts being stable
4. Mark [P] for parallel: Database + CRD, Frontend + Backend (after contracts)

**Estimated Output**: 35-40 numbered, ordered tasks in tasks.md

**Key Task Categories**:
- Database setup (3-4 tasks)
- Kubernetes CRD and operator (3-4 tasks)
- Backend API handlers (8-10 tasks)
- Frontend UI components (10-12 tasks)
- LangGraph runner implementation (8-10 tasks)
- Testing and validation (4-5 tasks)

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)
**Phase 4**: Implementation (execute tasks.md following constitutional principles)
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

No violations identified. Feature follows existing architectural patterns and maintains simplicity by:
- Reusing existing runner pattern (claude-code-runner as template)
- Single database instance (no distributed complexity)
- Standard REST + WebSocket protocols (no custom protocols)
- Explicit MVP scope excluding advanced features

## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [ ] Phase 0: Research complete (/plan command)
- [ ] Phase 1: Design complete (/plan command)
- [ ] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: CONDITIONAL PASS (verify TDD in research)
- [ ] Post-Design Constitution Check: PENDING
- [ ] All NEEDS CLARIFICATION resolved: PENDING (Phase 0)
- [x] Complexity deviations documented: N/A (no deviations)

---
*Based on Constitution template - Project-specific constitution pending*
