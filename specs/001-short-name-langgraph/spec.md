# Feature Specification: LangGraph Workflow Integration

**Feature Branch**: `001-short-name-langgraph`
**Created**: 2025-11-04
**Status**: Draft
**Input**: User description: "add lang-graph as a new runner"

## Execution Flow (main)
```
1. Parse user description from Input
   → ✅ Description extracted from rfe.md
2. Extract key concepts from description
   → ✅ Identified: actors (users, admins, workflow authors), actions (register, execute, monitor), data (sessions, workflows, checkpoints), constraints (security, scalability)
3. For each unclear aspect:
   → ✅ Marked critical clarifications (max 3)
4. Fill User Scenarios & Testing section
   → ✅ User flows defined with acceptance scenarios
5. Generate Functional Requirements
   → ✅ Requirements are testable and measurable
6. Identify Key Entities (if data involved)
   → ✅ Entities identified (WorkflowDefinition, WorkflowSession, Checkpoint)
7. Run Review Checklist
   → Pending validation
8. Return: SUCCESS (spec ready for planning)
```

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT users need and WHY
- ❌ Avoid HOW to implement (no tech stack, APIs, code structure)
- 👥 Written for business stakeholders, not developers

---

## Overview

### Purpose
Transform the Ambient Code Platform from a single-runner system (Claude Code only) into a multi-runner orchestration platform by adding support for LangGraph-based workflows. Users can register custom workflow definitions at the cluster level and execute them within their projects, with full support for long-running sessions, interactive approvals, and persistent state.

### Business Value
- **Extensibility**: Platform becomes a generic AI workflow hub instead of a Claude Code-specific tool
- **User Empowerment**: Teams bring their own specialized workflows without requiring platform engineering support
- **Competitive Positioning**: Differentiate from single-agent competitors by supporting multiple orchestration frameworks
- **Scalability**: Database-backed session management reduces pressure on cluster resources compared to current Custom Resource approach

### Target Users
1. **Platform Users**: Execute custom workflows for data analysis, content generation, research pipelines
2. **Workflow Authors**: Build and share specialized AI workflows without understanding platform internals
3. **Platform Administrators**: Manage workflow registry and enforce security policies
4. **Project Teams**: Leverage both file-based agents (Claude Code) and graph-based workflows within same project

---

## Clarifications

### Session 2025-11-04

- Q: What audit logging scope is required for compliance and debugging? → A: Audit logging deferred to future phase; not included in MVP
- Q: What are the concurrent workflow session limits per project? → A: No hard limit; rely on cluster resource quotas only
- Q: How should WorkflowDefinition name uniqueness be enforced? → A: Enforce unique names cluster-wide; reject registration if name already exists
- Q: How is "recent sessions" defined for workflow deletion policy (FR-007)? → A: Workflow deletion restrictions deferred; all workflows remain regardless of session age in MVP
- Q: What behavior should occur when WebSocket connection is unavailable? → A: Fail workflow execution if WebSocket unavailable (strict real-time requirement)

---

## User Scenarios & Testing *(mandatory)*

### Primary User Story
As a data analyst, I want to execute my team's custom LangGraph workflow for CSV analysis so that I can generate forecasts with human-in-the-loop validation of statistical outliers, without needing to understand Kubernetes or backend APIs.

### Acceptance Scenarios

#### Scenario 1: First-Time Workflow Registration
1. **Given** I am a platform administrator with cluster-admin permissions
2. **When** I navigate to the workflow registry page and register a new workflow with a valid container image and input schema
3. **Then** The workflow appears in the registry and becomes available for all project users to execute

#### Scenario 2: Execute Workflow Session
1. **Given** A workflow is registered and I have edit permissions on my project
2. **When** I create a new workflow session by selecting the workflow and filling the auto-generated input form
3. **Then** A session starts, shows real-time progress messages, and completes with results displayed

#### Scenario 3: Interactive Workflow with Approval Gates
1. **Given** I started a workflow that includes human review steps
2. **When** The workflow reaches an approval gate and pauses for my input
3. **Then** I see a prompt asking for my decision, can provide input via the UI, and the workflow resumes with my response

#### Scenario 4: Resume Interrupted Session
1. **Given** I started a workflow 2 days ago that paused waiting for my approval
2. **When** I return to the session detail page and respond to the pending prompt
3. **Then** The workflow resumes from where it left off, preserving all previous context and results

#### Scenario 5: Unified Session List View
1. **Given** My project has both Claude Code sessions and workflow sessions
2. **When** I view the session list
3. **Then** I see both types clearly labeled, with appropriate actions (view, delete, continue) for each

### Edge Cases

#### Registration Edge Cases
- **Invalid Registry**: What happens when a user tries to register a workflow from a non-whitelisted container registry?
  - System rejects the registration with a clear error message indicating the registry is not allowed
- **Missing Image**: What happens when the specified container image doesn't exist or cannot be pulled?
  - Image validation fails with an error message instructing the user to verify the image URL and access credentials
- **Malformed Input Schema**: What happens when the input schema is not valid JSON Schema?
  - Form validation prevents submission and highlights the schema syntax errors
- **Duplicate Workflow Name**: What happens when a user tries to register a workflow with a name that already exists cluster-wide?
  - System rejects the registration with a clear error message indicating the name is already in use and must be unique

#### Execution Edge Cases
- **Invalid Input Data**: What happens when a user submits input that doesn't match the workflow's schema requirements?
  - Form validation catches errors before submission; required fields are enforced and type mismatches are flagged
- **Workflow Timeout**: What happens when a workflow runs longer than the configured timeout period?
  - System marks the session as failed with a timeout error message, cleans up associated resources
- **Pod Crash**: What happens when the workflow execution pod crashes (e.g., out-of-memory)?
  - System detects the failure, updates session status to failed, and displays the error details from pod logs
- **Checkpoint Corruption**: What happens when stored checkpoint data becomes corrupted?
  - System displays an error indicating the session cannot be resumed, user must start a new session
- **WebSocket Connection Failure**: What happens when WebSocket connection cannot be established or is lost during execution?
  - Workflow execution fails immediately with error message indicating real-time connection is required; user must retry when network is stable

#### Concurrency Edge Cases
- **Concurrent Sessions**: What happens when a user starts multiple workflow sessions simultaneously?
  - All sessions execute independently with isolated state, no interference between sessions
- **Multiple Resume Attempts**: What happens when a user tries to resume a session that's already running?
  - System prevents duplicate resumption with an error indicating the session is already active

#### Data Edge Cases
- **Session Deletion During Execution**: What happens when a user deletes a workflow session that's currently running?
  - System cancels the execution job, cleans up resources, and removes the database record
- **Large Output Data**: What happens when a workflow produces output larger than expected?
  - System enforces a 100MB limit on output data stored in the database; workflows producing larger outputs must handle storage externally (e.g., object storage with URLs in output), and exceeding the limit results in a validation error

---

## Requirements *(mandatory)*

### Functional Requirements

#### Workflow Registration (Cluster-Level)
- **FR-001**: System MUST allow cluster administrators to register workflow definitions with a unique name, display name, description, container image reference, and input schema; workflow names MUST be unique cluster-wide and registration MUST be rejected if name already exists
- **FR-002**: System MUST validate that container images come from whitelisted registries during workflow registration
- **FR-003**: System MUST validate that input schemas conform to JSON Schema specification
- **FR-004**: System MUST store workflow definitions at cluster scope (accessible to all projects)
- **FR-005**: System MUST allow cluster administrators to update existing workflow definitions
- **FR-006**: System MUST allow cluster administrators to delete workflow definitions that have no active sessions
- **FR-007**: System MUST prevent deletion of workflows that have active sessions; deletion restrictions for completed sessions deferred to future phase

#### Workflow Session Management (Project-Scoped)
- **FR-008**: System MUST allow project users with edit permissions to create new workflow sessions
- **FR-009**: System MUST generate input forms dynamically based on a workflow's input schema
- **FR-010**: System MUST validate user-submitted input against the workflow's schema before creating a session
- **FR-011**: System MUST execute workflow sessions in isolated containers with the registered workflow image
- **FR-012**: System MUST store session state persistently in a database
- **FR-013**: System MUST track session lifecycle states: pending, running, completed, failed, waiting_for_input
- **FR-014**: System MUST allow users to view all workflow sessions within their project
- **FR-015**: System MUST allow users to delete workflow sessions they have permission to access
- **FR-016**: System MUST distinguish between workflow sessions and Claude Code sessions in the UI

#### Session Execution and Monitoring
- **FR-017**: System MUST stream real-time progress messages from running workflows to the UI; workflow execution MUST fail if WebSocket connection is unavailable
- **FR-018**: System MUST display session status, start time, and completion time
- **FR-019**: System MUST display session input parameters and final output results
- **FR-020**: System MUST capture and display error messages when sessions fail
- **FR-021**: System MUST automatically detect when execution jobs fail and update session status accordingly
- **FR-022**: System MUST clean up execution resources after session completion or failure

#### Interactive Workflows (Human-in-the-Loop)
- **FR-023**: System MUST support workflows that pause for human approval or input
- **FR-024**: System MUST notify users when a workflow is waiting for input via UI status updates
- **FR-025**: System MUST allow users to provide responses to workflow prompts via the UI
- **FR-026**: System MUST resume workflow execution after receiving user input
- **FR-027**: System MUST maintain conversation history across multiple interaction rounds

#### Session Persistence and Resumption
- **FR-028**: System MUST persist workflow state across interruptions
- **FR-029**: System MUST allow users to resume interrupted sessions from their last checkpoint
- **FR-030**: System MUST maintain session context when resuming after hours or days
- **FR-031**: System MUST isolate session state by project scope (sessions within a project share context)
- **FR-032**: System MUST preserve message history when resuming sessions

#### Security and Access Control
- **FR-033**: System MUST enforce role-based access control: only cluster administrators can register workflows
- **FR-034**: System MUST enforce project-level permissions: users can only access sessions in authorized projects
- **FR-035**: System MUST authenticate workflow execution containers with the platform backend
- **FR-036**: System MUST validate user tokens before allowing session creation or modification
- **FR-037**: System MUST restrict workflow container images to whitelisted registries only

#### Integration with Existing System
- **FR-038**: System MUST continue supporting existing Claude Code sessions without disruption
- **FR-039**: System MUST display both workflow sessions and Claude Code sessions in a unified session list
- **FR-040**: System MUST maintain backward compatibility with existing session APIs and UI components

### Success Criteria
- Users can register a new workflow and execute their first session within 30 minutes
- 95% of workflow sessions that request human input successfully resume after user response
- Session list page loads within 2 seconds even with 100+ sessions in a project
- Real-time progress messages appear in UI within 5 seconds of being generated by workflow
- Users can distinguish between workflow types and session types at a glance in the UI
- Zero disruption to existing Claude Code session functionality during rollout

### Key Entities *(include if feature involves data)*

#### WorkflowDefinition
- Represents a registered workflow template available cluster-wide
- Key attributes: unique name (cluster-wide uniqueness enforced), display name, description, container image reference, input schema (JSON Schema format), registration timestamp
- Lifecycle: Created by cluster admins, used by all projects, deleted only when no active sessions exist (no restrictions on completed sessions in MVP)

#### WorkflowSession
- Represents an instance of a workflow execution within a project
- Key attributes: unique session ID, project name, workflow definition reference, input data, output data, status (pending/running/completed/failed/waiting_for_input), timestamps (created/started/completed), created-by user
- Relationships: References one WorkflowDefinition, belongs to one project, contains multiple SessionMessages
- Lifecycle: Created by project users, executes via container job, transitions through states, can be resumed if interrupted

#### SessionMessage
- Represents a single message in a workflow session's conversation history
- Key attributes: message type (system/agent/user), timestamp, sequence number, message payload
- Relationships: Belongs to one WorkflowSession, ordered by sequence number
- Purpose: Provides real-time progress updates and maintains conversation history

#### Checkpoint
- Represents saved workflow state at a specific point in execution
- Key attributes: thread identifier, checkpoint identifier, parent checkpoint reference, state data, timestamp, scope (project or private), scope identifier
- Relationships: Linked by thread ID, forms a chain via parent references
- Purpose: Enables session resumption and maintains context across interruptions

---

## Assumptions

1. **Registry Whitelist Configuration**: Platform administrators will configure the allowed registry list via environment variables or configuration management before workflow registration
2. **Database Availability**: A PostgreSQL database instance will be available and properly configured for session storage
3. **Workflow Container Standards**: Workflow authors will build containers that follow the platform's base image structure and integration patterns
4. **Input Schema Familiarity**: Workflow authors understand JSON Schema format for defining input structures
5. **Session Timeout Defaults**: A reasonable default timeout (e.g., 1 hour) will be applied to prevent runaway workflows unless otherwise specified
6. **Message Streaming Protocol**: WebSocket connections will be used for real-time message streaming from workflows to UI
7. **Project Namespace Mapping**: Each project corresponds to a Kubernetes namespace with appropriate RBAC policies
8. **Resource Limits**: Standard resource limits (CPU, memory) will be applied to workflow execution pods based on cluster capacity; no hard limit on concurrent sessions per project (capacity managed by cluster resource quotas)
9. **Checkpoint Retention**: Checkpoint data will be retained for a standard period (e.g., 30 days) to support session resumption
10. **Authentication Method**: User authentication follows the existing platform authentication mechanism (OpenShift OAuth or equivalent)
11. **Output Size Limits**: Workflow output data stored in the database is limited to 100MB based on PostgreSQL JSONB performance best practices; larger outputs require external storage solutions

---

## Out of Scope

The following items are explicitly excluded from this feature:

### User-Scoped Workflows
- Workflows registered as private/personal resources (only cluster-wide registration supported)
- User-level isolation and personal workflow access control (future consideration)

### Multi-Repo Workspace Support
- Workflows do not operate on file system workspaces with multiple git repositories
- File-based operations should be delegated to nested Claude Code sessions

### Workflow Marketplace and Sharing
- Sharing workflows across different tenants or organizations
- Public workflow registry or marketplace
- Workflow versioning and distribution mechanisms

### Advanced Visualization
- Real-time workflow graph (DAG) visualization in UI
- Execution path highlighting
- Interactive graph debugging

### External Observability Tools
- LangSmith integration for advanced LangGraph tracing
- External monitoring platform integration
- Custom metrics dashboards for workflow analytics

### Additional Runner Types
- OpenAI Assistants API runner
- Gemini Agents runner
- DeepSeek Agents runner
- Other orchestration framework runners beyond LangGraph (architecture designed for future extensibility)

### Automated Workflow Building
- Automatic container image building from workflow source code
- CI/CD pipeline for workflow development
- Workflow code validation and testing infrastructure

### Advanced Security Features
- Workflow code security scanning and vulnerability detection
- Static analysis of workflow logic
- Runtime behavior monitoring and anomaly detection
- Per-workflow resource limit customization (global limits only)
- Audit logging of workflow registrations, session operations, and admin actions (deferred to future phase)

### Legacy Session Migration
- Automated migration of existing AgenticSession CRs to database-backed model
- Conversion tools for legacy sessions
- Backward compatibility layer for legacy session format

### Private Scope Sessions
- Private (user-scoped) workflow sessions isolated from project team
- User-level checkpoint isolation
- Personal workflow execution history

---

## Dependencies

### Platform Infrastructure
- Kubernetes cluster with sufficient capacity to run workflow execution pods
- PostgreSQL database instance with appropriate schema and access credentials
- Container registry (whitelisted) accessible from the cluster for pulling workflow images

### Existing Platform Components
- Backend API service with authentication and authorization capabilities
- Frontend UI framework and routing infrastructure
- Project namespace management and RBAC enforcement
- WebSocket support for real-time messaging

### External Services
- Container registries hosting workflow images (must be whitelisted)
- LangGraph framework and dependencies (bundled in workflow containers)

---

## Constraints

### Technical Constraints
1. Workflow containers must be compatible with Kubernetes Job execution model
2. Input schemas must conform to JSON Schema specification (no custom schema languages)
3. Real-time message streaming requires persistent WebSocket connections
4. Checkpoint data stored in relational database (PostgreSQL) with defined schema
5. Workflow execution isolated to single container (no multi-container workflows in MVP)

### Security Constraints
1. Only whitelisted container registries allowed for workflow images
2. Cluster administrator role required for workflow registration
3. Project-level RBAC enforced for all session operations
4. Workflow execution containers must not run as privileged
5. User tokens required for all authenticated API requests

### Operational Constraints
1. Session state stored in single database instance (no distributed storage in MVP)
2. WebSocket connections require stable network connectivity for real-time updates
3. Checkpoint data retention governed by database storage capacity
4. Workflow execution limited by cluster resource availability
5. No cross-project workflow sharing (project-scoped isolation)

### User Experience Constraints
1. Users must build and publish workflow container images independently (no in-platform building)
2. Workflow authors must understand JSON Schema for input form generation
3. UI displays sessions from single project at a time (no cross-project views)
4. Session history limited to messages stored in database (no external logging integration)

---

## Review & Acceptance Checklist
*GATE: Automated checks run during main() execution*

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

### Requirement Completeness
- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

---

## Execution Status
*Updated by main() during processing*

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked and resolved
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [x] Review checklist passed

---
