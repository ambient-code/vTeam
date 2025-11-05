# Tasks: LangGraph Workflow Integration

**Feature Branch**: `001-short-name-langgraph` (or `ambient-langgraph-integration`)
**Input**: Design documents from `/specs/001-short-name-langgraph/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/, spec.md

## Execution Flow

This tasks.md is generated from the feature specification and design artifacts. Tasks are organized by user story to enable independent implementation and testing of each feature increment.

## User Story Mapping

Based on spec.md, the tasks are organized around these primary user scenarios:

1. **User Story 1 (US1)**: Workflow Registration - Administrators can register workflow definitions cluster-wide
2. **User Story 2 (US2)**: Workflow Session Execution - Users can create and execute workflow sessions
3. **User Story 3 (US3)**: Interactive Workflows - Workflows can pause for human input and resume
4. **User Story 4 (US4)**: Session Resumption - Users can resume interrupted sessions from checkpoints
5. **User Story 5 (US5)**: Unified Session List - Users see both Claude Code and workflow sessions together

## Phase 1: Setup & Infrastructure

**Goal**: Establish foundational infrastructure for workflow support (database, CRD, migrations)

- [ ] T001 Create database migration 001_initial_schema.up.sql for workflow_sessions and session_messages tables in components/backend/db/migrations/
- [ ] T002 Create database connection module components/backend/db/postgres.go with PostgreSQL connection pool and migration runner
- [ ] T003 [P] Create WorkflowDefinition CRD YAML in components/manifests/crds/workflow-definition-crd.yaml per data-model.md spec
- [ ] T004 [P] Create PostgreSQL deployment manifest in components/manifests/base/postgres-deployment.yaml with persistent volume
- [ ] T005 [P] Add database configuration to backend config in components/backend/config/database.go
- [ ] T006 Run database migrations on startup in components/backend/main.go initialization

## Phase 2: Foundational Components (Blocking Prerequisites)

**Goal**: Build shared types, utilities, and base services that all user stories depend on

- [ ] T007 [P] Create workflow types in components/backend/types/workflow.go for WorkflowDefinition and WorkflowSession structs
- [ ] T008 [P] Create database access layer components/backend/db/workflow_sessions.go with CRUD operations for workflow_sessions table
- [ ] T009 [P] Create database access layer components/backend/db/session_messages.go with message insert and query operations
- [ ] T010 [P] Create K8s workflow job template in components/backend/k8s/workflow_job.go following existing claude-code-runner pattern
- [ ] T011 Create operator workflow definition watch handler in components/operator/internal/handlers/workflow_definitions.go skeleton (no job creation yet)
- [ ] T012 [P] Create LangGraph runner directory structure components/runners/langgraph-runner/ with pyproject.toml and Dockerfile
- [ ] T013 [P] Create runner base modules: components/runners/langgraph-runner/src/langgraph_runner/__init__.py, __main__.py, config.py

## Phase 3: User Story 1 - Workflow Registration

**Goal**: Enable cluster administrators to register workflow definitions

**Independent Test Criteria**:
- Can create a WorkflowDefinition CR via API
- Registry validation enforces whitelisted registries
- Input schema validation rejects invalid JSON Schema
- Cluster-wide name uniqueness is enforced

### Contract Tests (if requested)

- [ ] T014 [P] [US1] Contract test POST /api/workflows in tests/backend/contract/workflow_api_test.go validating request/response schemas
- [ ] T015 [P] [US1] Contract test GET /api/workflows in tests/backend/contract/workflow_api_test.go with filtering parameters
- [ ] T016 [P] [US1] Contract test GET /api/workflows/{name} in tests/backend/contract/workflow_api_test.go for single workflow retrieval
- [ ] T017 [P] [US1] Contract test DELETE /api/workflows/{name} in tests/backend/contract/workflow_api_test.go with active session check

### Implementation

- [ ] T018 [US1] Create workflow registry handler components/backend/handlers/workflows.go with POST /api/workflows endpoint
- [ ] T019 [US1] Add GET /api/workflows list endpoint to components/backend/handlers/workflows.go with filtering support
- [ ] T020 [US1] Add GET /api/workflows/{name} detail endpoint to components/backend/handlers/workflows.go
- [ ] T021 [US1] Add DELETE /api/workflows/{name} endpoint to components/backend/handlers/workflows.go with active session check
- [ ] T022 [US1] Add registry whitelist validation to components/backend/handlers/workflows.go using ALLOWED_REGISTRIES env var
- [ ] T023 [US1] Add JSON Schema validation to components/backend/handlers/workflows.go using JSON Schema Draft 2020-12 validator
- [ ] T024 [US1] Register workflow routes in components/backend/routes.go
- [ ] T025 [P] [US1] Create workflow registry list page components/frontend/src/app/workflows/page.tsx with filtering and search
- [ ] T026 [P] [US1] Create workflow registration form components/frontend/src/app/workflows/new/page.tsx with schema input
- [ ] T027 [P] [US1] Create workflow detail page components/frontend/src/app/workflows/[name]/page.tsx showing spec and active sessions
- [ ] T028 [P] [US1] Create workflow API client components/frontend/src/services/api/workflows.ts with all CRUD operations
- [ ] T029 [P] [US1] Create workflow React Query hooks components/frontend/src/services/queries/workflows.ts with cache invalidation
- [ ] T030 [P] [US1] Create workflow registry card component components/frontend/src/components/workflow-registry-card.tsx using Shadcn Card

## Phase 4: User Story 2 - Workflow Session Execution

**Goal**: Enable users to create and execute workflow sessions with real-time progress

**Independent Test Criteria**:
- Can create a workflow session via API with valid input
- Input validation rejects data not matching workflow schema
- Session executes in isolated container
- Real-time progress messages stream via WebSocket
- Session status transitions correctly (pending → running → completed)

### Contract Tests (if requested)

- [ ] T031 [P] [US2] Contract test POST /api/projects/{project}/workflow-sessions in tests/backend/contract/workflow_session_api_test.go
- [ ] T032 [P] [US2] Contract test GET /api/projects/{project}/workflow-sessions in tests/backend/contract/workflow_session_api_test.go
- [ ] T033 [P] [US2] Contract test GET /api/projects/{project}/workflow-sessions/{id} in tests/backend/contract/workflow_session_api_test.go
- [ ] T034 [P] [US2] Contract test DELETE /api/projects/{project}/workflow-sessions/{id} in tests/backend/contract/workflow_session_api_test.go

### Backend Implementation

- [ ] T035 [US2] Create workflow session handler components/backend/handlers/workflow_sessions.go with POST endpoint
- [ ] T036 [US2] Add input data validation against workflow inputSchema to components/backend/handlers/workflow_sessions.go
- [ ] T037 [US2] Add GET /api/projects/{project}/workflow-sessions list endpoint to components/backend/handlers/workflow_sessions.go
- [ ] T038 [US2] Add GET /api/projects/{project}/workflow-sessions/{id} detail endpoint to components/backend/handlers/workflow_sessions.go
- [ ] T039 [US2] Add DELETE /api/projects/{project}/workflow-sessions/{id} endpoint to components/backend/handlers/workflow_sessions.go with cleanup
- [ ] T040 [US2] Create job creation logic in components/backend/handlers/workflow_sessions.go to spawn LangGraph runner pod
- [ ] T041 [US2] Add session status update endpoint to components/backend/handlers/workflow_sessions.go for runner callbacks
- [ ] T042 [US2] Register workflow session routes in components/backend/routes.go under project-scoped paths
- [ ] T043 [US2] Update operator handler components/operator/internal/handlers/workflow_definitions.go to watch for workflow sessions and create Jobs

### LangGraph Runner Implementation

- [ ] T044 [P] [US2] Create checkpoint manager components/runners/langgraph-runner/src/langgraph_runner/checkpoint_manager.py using AsyncPostgresSaver
- [ ] T045 [P] [US2] Create message publisher components/runners/langgraph-runner/src/langgraph_runner/message_publisher.py extending ResilientWebSocketTransport
- [ ] T046 [P] [US2] Create workflow loader components/runners/langgraph-runner/src/langgraph_runner/workflow_loader.py for dynamic workflow import
- [ ] T047 [US2] Create main runner execution loop components/runners/langgraph-runner/src/langgraph_runner/runner.py with checkpoint initialization
- [ ] T048 [US2] Add workflow execution logic to components/runners/langgraph-runner/src/langgraph_runner/runner.py using LangGraph astream
- [ ] T049 [US2] Add progress message publishing to components/runners/langgraph-runner/src/langgraph_runner/runner.py via WebSocket
- [ ] T050 [US2] Add error handling and status updates to components/runners/langgraph-runner/src/langgraph_runner/runner.py
- [ ] T051 [US2] Update runner Dockerfile components/runners/langgraph-runner/Dockerfile with LangGraph dependencies

### Frontend Implementation

- [ ] T052 [P] [US2] Create dynamic form generation utility components/frontend/src/lib/json-schema-to-form.ts using zod-from-json-schema
- [ ] T053 [P] [US2] Create workflow input form component components/frontend/src/components/workflow-input-form.tsx with dynamic field rendering
- [ ] T054 [P] [US2] Create workflow session creation page components/frontend/src/app/projects/[name]/workflow-sessions/new/page.tsx
- [ ] T055 [P] [US2] Create workflow session detail page components/frontend/src/app/projects/[name]/workflow-sessions/[id]/page.tsx with WebSocket connection
- [ ] T056 [P] [US2] Create message list component for workflow sessions in components/frontend/src/app/projects/[name]/workflow-sessions/[id]/components/message-list.tsx
- [ ] T057 [P] [US2] Update workflow API client components/frontend/src/services/api/workflows.ts with session operations
- [ ] T058 [P] [US2] Create workflow session React Query hooks components/frontend/src/services/queries/workflow-sessions.ts

## Phase 5: User Story 3 - Interactive Workflows

**Goal**: Enable workflows to pause for human input and resume after user response

**Independent Test Criteria**:
- Workflow can send waiting_for_input message
- Session status transitions to waiting_for_input
- User can submit response via UI
- Workflow resumes with user input and continues execution
- Conversation history is preserved across interactions

### Implementation

- [ ] T059 [US3] Add human-in-the-loop interrupt handling to components/runners/langgraph-runner/src/langgraph_runner/runner.py
- [ ] T060 [US3] Add user input listener to components/runners/langgraph-runner/src/langgraph_runner/message_publisher.py via WebSocket
- [ ] T061 [US3] Add workflow resume logic to components/runners/langgraph-runner/src/langgraph_runner/runner.py after receiving user input
- [ ] T062 [US3] Add waiting_for_input status update to components/backend/handlers/workflow_sessions.go
- [ ] T063 [P] [US3] Create user input prompt component components/frontend/src/app/projects/[name]/workflow-sessions/[id]/components/input-prompt.tsx
- [ ] T064 [US3] Add user input submission handler to components/frontend/src/app/projects/[name]/workflow-sessions/[id]/page.tsx via WebSocket
- [ ] T065 [P] [US3] Update message list component to display waiting_for_input prompts in components/frontend/src/app/projects/[name]/workflow-sessions/[id]/components/message-list.tsx

## Phase 6: User Story 4 - Session Resumption

**Goal**: Enable users to resume interrupted sessions from their last checkpoint

**Independent Test Criteria**:
- Can resume a session that was interrupted days ago
- Session state is preserved from last checkpoint
- Message history is maintained
- Workflow continues from exact interruption point
- Thread ID correctly encodes project scope

### Implementation

- [ ] T066 [US4] Add checkpoint retrieval logic to components/runners/langgraph-runner/src/langgraph_runner/checkpoint_manager.py for session resumption
- [ ] T067 [US4] Add resume session endpoint POST /api/projects/{project}/workflow-sessions/{id}/resume to components/backend/handlers/workflow_sessions.go
- [ ] T068 [US4] Update runner initialization in components/runners/langgraph-runner/src/langgraph_runner/runner.py to load from checkpoint if resuming
- [ ] T069 [P] [US4] Add resume button to workflow session detail page components/frontend/src/app/projects/[name]/workflow-sessions/[id]/page.tsx
- [ ] T070 [P] [US4] Add checkpoint retention cleanup job implementation in components/backend/db/checkpoint_cleanup.go with 30-day retention

## Phase 7: User Story 5 - Unified Session List

**Goal**: Display both Claude Code sessions and workflow sessions in a single unified view

**Independent Test Criteria**:
- Session list shows both session types clearly labeled
- Filtering works for both types
- Correct actions available per session type (view, delete, resume)
- Performance < 2s with 100+ sessions

### Implementation

- [ ] T071 [US5] Update unified session list endpoint GET /api/projects/{project}/sessions in components/backend/handlers/sessions.go to include workflow sessions
- [ ] T072 [US5] Add session type field to response in components/backend/handlers/sessions.go
- [ ] T073 [P] [US5] Update unified session list page components/frontend/src/app/projects/[name]/sessions/page.tsx to handle both types
- [ ] T074 [P] [US5] Create unified session card component components/frontend/src/components/unified-session-card.tsx with type-specific rendering
- [ ] T075 [P] [US5] Update session queries to fetch both types in components/frontend/src/services/queries/sessions.ts

## Phase 8: Polish & Cross-Cutting Concerns

**Goal**: Complete testing, documentation, performance optimization, and production readiness

- [ ] T076 [P] Create end-to-end test in tests/backend/integration/workflow_e2e_test.go testing full workflow lifecycle
- [ ] T077 [P] Add performance test for session list page load time in tests/backend/integration/workflow_performance_test.go
- [ ] T078 [P] Update CLAUDE.md with LangGraph runner patterns in root CLAUDE.md Backend and Operator Development Standards section
- [ ] T079 [P] Create workflow runner README components/runners/langgraph-runner/README.md with setup and development instructions
- [ ] T080 [P] Add workflow session cleanup on deletion to components/backend/handlers/workflow_sessions.go (delete messages, checkpoints)
- [ ] T081 [P] Add output data size validation (100MB limit) to components/backend/handlers/workflow_sessions.go
- [ ] T082 [P] Add registry whitelist configuration documentation in components/manifests/README.md
- [ ] T083 Run manual quickstart test per specs/001-short-name-langgraph/quickstart.md and fix any issues
- [ ] T084 Verify all contract tests pass and match OpenAPI specs in contracts/ directory
- [ ] T085 Review and refactor duplicate code across handlers (extract common validation, error handling)

## Dependencies

**Critical Path** (must complete sequentially):
1. Phase 1 (Setup) → Phase 2 (Foundational) → User Story Phases (3-7) → Phase 8 (Polish)
2. Within Phase 2: T007 (types) must complete before T008-T009 (database access)
3. Within Phase 2: T010 (job template) must complete before T011 (operator handler)

**User Story Dependencies**:
- US1 (Workflow Registration) must complete before US2 (Session Execution)
- US2 (Session Execution) must complete before US3 (Interactive) and US4 (Resumption)
- US2-US4 must complete before US5 (Unified List)

**Parallel Opportunities**:
- Within each user story phase, tasks marked [P] can run in parallel
- Frontend and backend tasks for same user story can run in parallel after contracts exist
- Database and CRD tasks in Phase 1 are independent

## Parallel Execution Examples

### Phase 1 - Setup (4 parallel tasks)
```bash
# Can run simultaneously:
T003: Create WorkflowDefinition CRD YAML
T004: Create PostgreSQL deployment manifest
T005: Add database configuration to backend
T012: Create LangGraph runner directory structure
```

### User Story 1 - Contract Tests (4 parallel tasks)
```bash
# All contract tests are independent:
T014: Contract test POST /api/workflows
T015: Contract test GET /api/workflows
T016: Contract test GET /api/workflows/{name}
T017: Contract test DELETE /api/workflows/{name}
```

### User Story 2 - Runner + Frontend (parallel tracks)
```bash
# Runner implementation (independent files):
T044: Create checkpoint manager
T045: Create message publisher
T046: Create workflow loader

# Frontend implementation (independent files):
T052: Create form generation utility
T053: Create workflow input form component
T056: Create message list component
```

## Implementation Strategy

**MVP Scope** (Recommended first iteration):
- Phase 1: Setup & Infrastructure
- Phase 2: Foundational Components
- Phase 3: User Story 1 - Workflow Registration (complete)
- Phase 4: User Story 2 - Workflow Session Execution (basic, no interrupts)

This delivers a working end-to-end flow: register workflow → execute session → view results.

**Incremental Delivery**:
- Iteration 2: Add US3 (Interactive Workflows)
- Iteration 3: Add US4 (Session Resumption)
- Iteration 4: Add US5 (Unified Session List)
- Iteration 5: Phase 8 (Polish)

## Task Statistics

- **Total Tasks**: 85
- **Setup & Infrastructure**: 6 tasks
- **Foundational**: 7 tasks
- **User Story 1 (Registration)**: 17 tasks (4 tests + 13 implementation)
- **User Story 2 (Execution)**: 28 tasks (4 tests + 24 implementation)
- **User Story 3 (Interactive)**: 7 tasks
- **User Story 4 (Resumption)**: 5 tasks
- **User Story 5 (Unified List)**: 5 tasks
- **Polish**: 10 tasks

**Parallel Tasks**: 48 marked with [P] (56% parallelizable)

**Test Tasks**: 8 contract tests (if TDD approach requested)

## Validation Checklist

- [x] All user stories from spec.md have corresponding task phases
- [x] All entities from data-model.md have implementation tasks (WorkflowDefinition, WorkflowSession, SessionMessage, Checkpoint)
- [x] All API endpoints from contracts/ have handler tasks
- [x] Contract tests precede implementation (TDD-ready)
- [x] Each task specifies exact file path
- [x] Tasks follow strict format: `- [ ] [ID] [P?] [Story?] Description with file path`
- [x] Parallel tasks truly independent (different files, no dependencies)
- [x] Dependencies clearly documented
- [x] MVP scope identified for incremental delivery
- [x] All tasks map to user stories (US1-US5) where applicable

## Notes

- **[P] marker**: Tasks can run in parallel (different files, no blocking dependencies)
- **[US#] marker**: Maps task to user story for traceability
- **Tests optional**: Contract test tasks (T014-T017, T031-T034) can be skipped if not following TDD approach
- **Commit strategy**: Commit after each task or logical group of [P] tasks
- **Backend/Operator standards**: Follow patterns in CLAUDE.md Backend and Operator Development Standards section
- **Frontend standards**: Follow patterns in DESIGN_GUIDELINES.md (Shadcn UI, React Query, zero `any` types)

---

**Generated**: 2025-11-05
**From**: spec.md (user stories), plan.md (architecture), data-model.md (entities), contracts/ (API specs), research.md (technical decisions)
