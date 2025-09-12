# Tasks: Multi-Tenant Project-Based Session Management

**Input**: Design documents from `/specs/001-develop-a-feature/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/, quickstart.md

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → Tech stack: Go 1.21+ (backend/operator), NextJS 15.5.2 (frontend), Python 3.11+ (runners)
   → Libraries: Kubernetes controller-runtime, Gin framework, React 19.1.0, Anthropic Claude API
   → Structure: Web application with 4 components (backend, frontend, operator, runners)
2. Load design documents:
   → data-model.md: 2 CRDs (AgenticSession, ProjectSettings), 4 RBAC roles
   → contracts/: openapi.yaml with 14 endpoints across 3 categories
   → quickstart.md: 7 integration scenarios for testing
3. Generate tasks by category:
   → Setup: CRD definitions, operator scaffolding, API routing
   → Tests: 14 contract tests, 7 integration tests (TDD)
   → Core: 2 controllers, 14 API handlers, frontend components
   → Integration: OAuth proxy, webhook handlers, RBAC setup
   → Polish: unit tests, performance validation, documentation
4. Applied task rules:
   → Different CRDs/controllers = [P] parallel
   → Same backend files = sequential
   → Tests before implementation (TDD enforcement)
5. Generated: 32 numbered, ordered tasks
6. Dependencies: Setup → Tests → Controllers → API → Frontend → Integration → Polish
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in component directory structure

## Path Conventions (from plan.md)
- **Backend**: `components/backend/src/`
- **Frontend**: `components/frontend/src/`
- **Operator**: `components/operator/`
- **Runners**: `components/runners/claude-code-runner/src/`

---

## Phase 3.1: Setup & New CRD Definitions
**Note**: Backend, frontend, operator structure already exists. Focus on multi-tenant additions.

- [x] T001 ✅ EXISTING: AgenticSession v1alpha1 CRD already updated in components/manifests/crd.yaml
- [x] T002 Add userContext and project reference fields to existing AgenticSession CRD schema in components/manifests/crd.yaml
- [x] T003 Create NEW ProjectSettings v1alpha1 CRD schema in components/manifests/projectsettings-crd.yaml
- [x] T004 [P] Create custom RBAC roles manifests in components/manifests/rbac/agenticsession-roles.yaml
- [x] T005 ✅ EXISTING: Go modules and dependencies already configured
- [x] T006 ✅ EXISTING: Frontend NextJS project already initialized

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**

### Contract Tests [P] - API Endpoints
- [x] T007 [P] Contract test GET /projects/{project}/agentic-sessions in components/backend/tests/contract/sessions_list_test.go
- [x] T008 [P] Contract test POST /projects/{project}/agentic-sessions in components/backend/tests/contract/sessions_create_test.go
- [x] T009 [P] Contract test GET /projects/{project}/agentic-sessions/{sessionName} in components/backend/tests/contract/sessions_get_test.go
- [x] T010 [P] Contract test PUT /projects/{project}/agentic-sessions/{sessionName} in components/backend/tests/contract/sessions_update_test.go
- [x] T011 [P] Contract test DELETE /projects/{project}/agentic-sessions/{sessionName} in components/backend/tests/contract/sessions_delete_test.go
- [x] T012 [P] Contract test POST /projects/{project}/agentic-sessions/{sessionName}/clone in components/backend/tests/contract/sessions_clone_test.go
- [x] T013 [P] Contract test POST /projects/{project}/agentic-sessions/{sessionName}/start in components/backend/tests/contract/sessions_start_test.go
- [x] T014 [P] Contract test POST /projects/{project}/agentic-sessions/{sessionName}/stop in components/backend/tests/contract/sessions_stop_test.go
- [x] T015 [P] Contract test GET /projects in components/backend/tests/contract/projects_list_test.go
- [x] T016 [P] Contract test POST /projects in components/backend/tests/contract/projects_create_test.go
- [x] T017 [P] Contract test GET /projects/{projectName}/access in components/backend/tests/contract/project_access_test.go
- [x] T018 [P] Contract test GET/POST/DELETE /projects/{projectName}/groups in components/backend/tests/contract/project_groups_test.go
- [x] T019 [P] Contract test GET/POST/DELETE /projects/{projectName}/keys in components/backend/tests/contract/project_keys_test.go

### Integration Tests [P] - End-to-End Scenarios
- [x] T021 [P] Integration test ambient project creation and labeling in components/backend/tests/integration/project_creation_test.go
- [x] T022 [P] Integration test session lifecycle with project context in components/backend/tests/integration/session_lifecycle_test.go
- [x] T023 [P] Integration test ProjectSettings auto-creation on project labeling in components/operator/tests/integration/projectsettings_controller_test.go
- [x] T024 [P] Integration test access keys issuance and RBAC in components/backend/tests/integration/access_keys_test.go
- [x] T025 [P] Integration test permission boundaries and access control in components/backend/tests/integration/permission_test.go
- [x] T026 [P] Integration test monitoring and observability endpoints in components/backend/tests/integration/observability_test.go

## Phase 3.3: Core Implementation (ONLY after tests are failing)

### CRD Controllers
- [x] T028 Extend existing AgenticSession operator in components/operator/main.go to support project context and v1alpha1 fields
- [x] T029 [P] Add NEW ProjectSettings controller logic to components/operator/main.go for ServiceAccount/RBAC creation

### Backend API Extensions (Extend existing main.go)
- [x] T030 Add project context validation to session endpoints in components/backend/main.go
- [x] T031 Add /projects endpoints (GET, POST, PUT, GET by name, DELETE) to components/backend/main.go
- [x] T032 Add /projects/{name}/access endpoint (GET) to components/backend/main.go
- [x] T033 Add /projects/{name}/groups endpoints (GET, POST, DELETE) to components/backend/main.go
- [x] T034 Add /projects/{name}/keys endpoints (GET, POST, DELETE) to components/backend/main.go
- [x] T035 Add session cloning logic (/agentic-sessions/{name}/clone) to components/backend/main.go
- [x] T036 [P] Add project service functions for OpenShift project operations in components/backend/main.go
- [x] T037 [P] Add permission validation functions for RBAC checking in components/backend/main.go

### Frontend Extensions [P] - Extend existing session UI
- [x] T039 [P] Add project selection to existing session creation in components/frontend/src/app/new/
- [x] T040 [P] Create NEW project management pages in components/frontend/src/app/projects/
- [x] T041 [P] Create NEW group management interface in components/frontend/src/app/projects/[name]/groups/
- [x] T042 [P] Create NEW access keys interface in components/frontend/src/app/projects/[name]/keys/
- [x] T043 [P] Add session cloning functionality to existing session pages
- [x] T044 [P] Update existing session list to show project context

## Phase 3.4: Integration & Middleware

- [x] T046 Add OAuth proxy integration to existing backend in components/backend/main.go
- [x] T047 Add X-OpenShift-Project header validation to existing backend in components/backend/main.go
- [x] T048 Add JWT token bypass for webhook endpoints to existing backend in components/backend/main.go
- [x] T049 Add project labeling logic (ambient-code.io/managed=true) to backend in components/backend/main.go
- [x] T050 Update existing runner to support ServiceAccount tokens in components/runners/claude-code-runner/

## Phase 3.5: Polish & Validation

- [x] T051 [P] Unit tests for AgenticSession v1alpha1 validation in components/backend/tests/unit/agenticsession_validation_test.go
- [x] T052 [P] Unit tests for ProjectSettings validation in components/operator/tests/unit/projectsettings_validation_test.go
- [x] T053 [P] Unit tests for RBAC permission checking in components/backend/tests/unit/permission_validation_test.go
- [x] T054 Performance tests for session creation (<200ms) in components/backend/tests/performance/session_perf_test.go
- [x] T055 [P] Update deployment manifests with new RBAC requirements in components/manifests/
- [x] T056 [P] Execute full quickstart validation scenarios from specs/001-develop-a-feature/quickstart.md
- [x] T057 Manual end-to-end testing of all 7 scenarios

## Dependencies

### Critical Path
- **Setup** (T001-T006) before all other phases
- **Tests** (T007-T027) before **Core Implementation** (T028-T053) - TDD enforcement
- **CRD Controllers** (T028-T029) before **API Implementation** (T036-T042)
- **Models & Services** (T030-T035) before **API Handlers** (T036-T042)
- **Backend API** (T036-T042) before **Frontend** (T043-T048)
- **Core** before **Integration** (T049-T053) before **Polish** (T054-T060)

### Blocking Dependencies
- T028 blocks T036-T042 (controllers before API handlers)
- T030-T035 block T036-T042 (models/services before handlers)
- T049-T052 block T053 (auth middleware before runner auth)
- T001-T002 block T028-T029 (CRDs before controllers)

## Parallel Execution Examples

### Phase 3.2 - All Contract Tests Together:
```bash
# Launch T007-T020 (14 contract tests) in parallel:
Task: "Contract test GET /sessions in components/backend/tests/contract/sessions_list_test.go"
Task: "Contract test POST /sessions in components/backend/tests/contract/sessions_create_test.go"
Task: "Contract test GET /sessions/{sessionName} in components/backend/tests/contract/sessions_get_test.go"
# ... continue for all 14 contract tests

# Then launch T021-T027 (7 integration tests) in parallel:
Task: "Integration test ambient project creation in components/backend/tests/integration/project_creation_test.go"
Task: "Integration test session lifecycle in components/backend/tests/integration/session_lifecycle_test.go"
# ... continue for all 7 integration tests
```

### Phase 3.3 - Controllers in Parallel:
```bash
# Launch T028-T029 (CRD controllers) in parallel:
Task: "Extend existing AgenticSession operator in components/operator/main.go to support project context"
Task: "Add NEW ProjectSettings controller logic to components/operator/main.go for ServiceAccount/RBAC creation"

# Launch T037-T038 (service functions) in parallel:
Task: "Add project service functions for OpenShift project operations in components/backend/main.go"
Task: "Add permission validation functions for RBAC checking in components/backend/main.go"

# Launch T039-T045 (frontend components) in parallel:
Task: "Add project selection to existing session creation in components/frontend/src/app/new/"
Task: "Create NEW project management pages in components/frontend/src/app/projects/"
# ... continue for all frontend components
```

## Notes
- **[P] tasks** = different files, no shared dependencies
- **TDD enforcement**: All tests (T007-T027) must fail before starting T028
- **EXISTING infrastructure**: Backend, frontend, operator already exist - tasks extend current functionality
- **Multi-tenant additions**: Focus on ProjectSettings CRD, project context, and RBAC
- **File conflicts**: Backend extensions (T030-T038) are sequential due to shared main.go file
- **Authentication flow**: OAuth/ingress proxy forwards identity; API enforces RBAC
- **Ambient labeling**: `ambient-code.io/managed=true` enables ProjectSettings auto-creation (operator-internal)

## Validation Checklist
*GATE: Must verify before execution*

- [x] All API endpoints have contract tests (sessions, projects, groups, keys, access)
- [x] All required CRDs/controllers implemented (AgenticSession + ProjectSettings internal)
- [x] All quickstart scenarios have integration tests
- [x] All tests come before implementation (T007-T027 before T028+)
- [x] Parallel tasks target different files ([P] validation)
- [x] Each task specifies exact component file path
- [x] Dependencies properly block parallel execution
- [x] TDD cycle enforced (failing tests required first)