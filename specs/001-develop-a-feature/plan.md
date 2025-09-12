# Implementation Plan: Multi-Tenant Project-Based Session Management

**Branch**: `001-develop-a-feature` | **Date**: 2025-09-14 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-develop-a-feature/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   ✓ If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
4. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
5. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, or `GEMINI.md` for Gemini CLI).
6. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
7. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
8. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
Transform the existing single-tenant Ambient Agentic Runner platform into a multi-tenant system where users can organize sessions into projects (Kubernetes namespaces) with fine-grained permission controls, project access keys (ServiceAccounts), and group access via RoleBindings. This maintains the existing OpenShift operator and CRD architecture while adding project isolation and user/group permissions without custom webhook endpoints.

## Technical Context
**Language/Version**: Go 1.24+ (backend/operator), NextJS 15.x (frontend), Python 3.11+ (runners)
**Primary Dependencies**: Kubernetes client-go/controller-runtime, Gin framework, React 19, Anthropic Claude API
**Storage**: Kubernetes Custom Resources (AgenticSession v1alpha1; ProjectSettings internal), OpenShift identity provider for users/groups
**Testing**: Go testing, Jest/React Testing Library, Python pytest
**Target Platform**: OpenShift/Kubernetes cluster
**Project Type**: web - extends existing Kubernetes-native microservices architecture
**Performance Goals**: Start small - 10-20 projects with 50+ sessions total
**Constraints**: Leverage Kubernetes RBAC natively, clean slate CRD design with v1alpha1 versioning
**Scale/Scope**: Basic multi-tenant platform with project isolation, simple permissions, and access keys for automation

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Simplicity**:
- Projects: 4 (frontend, backend, operator, runners) - existing architecture
- Using framework directly? Yes - Kubernetes controller-runtime, Gin, NextJS without wrappers
- Single data model? Yes - extending existing AgenticSession CRD with project references
- Avoiding patterns? Yes - direct Kubernetes API usage, no unnecessary abstractions

**Architecture**:
- EVERY feature as library? Simplified - core functionality integrated into existing services
- Libraries listed:
  - project-settings-manager (ProjectSettings CRD management)
- CLI per library: Basic CLI support where needed
- Library docs: Simple documentation

**Testing (NON-NEGOTIABLE)**:
- RED-GREEN-Refactor cycle enforced? Yes - tests written first for new functionality
- Order: Basic tests before implementation
- Real dependencies used? Yes - actual Kubernetes clusters for key integration tests
- Focus areas: CRD functionality, basic permissions, webhook integration

**Observability**:
- Structured logging included? Yes - basic JSON logging
- Error context sufficient? Yes - basic error context for operations

**Versioning**:
- CRD Version: v1alpha1 (new experimental API, can evolve to beta/stable)
- Platform Version: v2.0.0 (major change due to multi-tenancy)
- BUILD increments on every change? Yes
- Breaking changes handled? Yes - clean slate approach, v1alpha1 allows API evolution

## Project Structure

### Documentation (this feature)
```
specs/001-develop-a-feature/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
# Option 2: Web application (existing structure)
components/
├── backend/             # Go API service
│   ├── src/
│   │   ├── models/      # CRD definitions, project models
│   │   ├── services/    # Project management, permission validation
│   │   └── api/         # REST endpoints for frontend/webhooks
│   └── tests/
├── frontend/            # NextJS web interface
│   ├── src/
│   │   ├── components/  # Project selection, permission UI
│   │   ├── pages/       # Multi-tenant session management
│   │   └── services/    # API integration
│   └── tests/
├── operator/            # Kubernetes operator
│   ├── controllers/     # AgenticSession controller with project support
│   └── tests/
└── runners/             # AI execution services
    └── claude-code-runner/
        ├── src/         # Bot authentication, project-aware execution
        └── tests/
```

**Structure Decision**: Option 2 (Web application) - extends existing Kubernetes-native microservices

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - Project namespace management patterns in Kubernetes operators
   - OpenShift identity provider integration best practices
   - Webhook authentication and authorization patterns
   - CRD design patterns for multi-tenant systems

2. **Generate and dispatch research agents**:
   ```
   Task: "Research Kubernetes namespace-per-tenant patterns for multi-tenant operators"
   Task: "Find best practices for OpenShift user/group integration in custom operators"
   Task: "Research webhook security patterns for external system integration (Jira)"
   Task: "Find CRD design patterns for multi-tenant systems from scratch"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all technical decisions resolved

**Key Research Decisions Made**:
- **Multi-Tenancy**: Native OpenShift projects with standard RBAC (no custom Project CRD)
- **Authentication**: OAuth/ingress proxy forwards user identity; backend performs RBAC via Kubernetes. No dedicated webhook bypass endpoints.
- **CRD Design**: AgenticSession v1alpha1 (user-facing). ProjectSettings v1alpha1 is operator-internal (auto-created) and not exposed via REST.
- **Credentials**: Per-project secrets with RBAC-based access control
- **Access Keys**: Standard ServiceAccounts issued tokens for automation (per-project)
- **Observability**: TBD solution with backend data push (no user auth needed)

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Native OpenShift projects (no custom CRD needed)
   - AgenticSession v1alpha1 with user context and project reference
   - ProjectSettings v1alpha1 for operator-internal configuration (auto-created, not REST-managed)
   - Standard ServiceAccounts for access keys (automation)
   - Custom RBAC roles for fine-grained permissions

2. **Generate API contracts** from functional requirements:
   - Session CRUD endpoints with project context: `/api/projects/{project}/agentic-sessions`
   - Project operations: `/api/projects`
   - Group access via RoleBindings: `/api/projects/{project}/groups`
   - Project access keys via ServiceAccounts: `/api/projects/{project}/keys`
   - Project access check: `/api/projects/{project}/access`
   - Output comprehensive OpenAPI schema to `/contracts/`

3. **Generate contract tests** from contracts:
   - Session lifecycle test scenarios in OpenShift projects
   - RBAC permission enforcement test cases
   - ServiceAccount authentication test scenarios
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Multi-user project collaboration flows
   - Bot-triggered session creation scenarios
   - Permission boundary validation tests

5. **Update agent file incrementally** (O(1) operation):
   - Run `/scripts/bash/update-agent-context.sh claude` for Claude Code
   - Add new multi-tenancy concepts to existing context
   - Preserve existing Kubernetes/operator knowledge
   - Update with project management patterns

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, CLAUDE.md

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Each API contract → contract test task [P]
- Each CRD change → CRD update + controller test task [P]
- Each UI component → component test + implementation task
- Integration tasks for cross-service communication

**Ordering Strategy**:
- TDD order: Tests before implementation
- Dependency order: CRDs → Operator → Backend API → Frontend
- Mark [P] for parallel execution (independent components)
- Kubernetes deployment order for integration testing

**Estimated Output**: 16-20 numbered, ordered tasks in tasks.md covering:
- AgenticSession v1alpha1 usage and controller
- Operator: ProjectSettings auto-create and RBAC wiring
- Backend API enhancements: projects, agentic-sessions, groups, keys, access check
- Frontend OpenShift project UI: projects list/create/update, groups/keys management
- No public webhook endpoint; ProjectSettings not exposed via REST

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)
**Phase 4**: Implementation (execute tasks.md following constitutional principles)
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 4th project (runners) | Existing architecture requirement | Consolidation would break container isolation |

## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*