# LangGraph Workflow Integration: Generic Orchestration Platform

**Feature Overview:**

Transform the Ambient Code Platform from a Claude Code-centric orchestration system into a generic AI workflow platform by introducing LangGraph workflow support. This feature enables users to register and execute custom LangGraph-based workflows alongside the existing Claude Code runner, with workflow sessions backed by PostgreSQL instead of Kubernetes Custom Resources. Users can bring their own pre-built LangGraph workflow images, register them at the cluster level, and execute them within project-scoped contexts with full support for LangGraph's native persistence, interrupts (human-in-the-loop), and memory features.

**Goals:**

**Who benefits:**
- **Platform Users**: Execute custom LangGraph workflows for specialized use cases (data analysis, content generation, research pipelines) alongside Claude Code sessions
- **Workflow Authors**: Bring their own LangGraph applications without needing to understand the Ambient Code Platform's backend APIs or orchestration
- **Platform Administrators**: Manage a unified orchestration platform supporting multiple AI agent frameworks
- **Future Integrations**: Establish extensible architecture for additional orchestration platforms (OpenAI Assistants, Gemini Agents, etc.)

**User outcomes:**
- Register pre-built LangGraph workflow container images at cluster level with input schema definitions
- Create and execute workflow sessions with dynamically generated input forms based on JSON Schema
- Leverage LangGraph's native features (persistence, interrupts, memory) for long-running and interactive workflows
- Monitor workflow execution status and outputs through the existing UI with dedicated workflow session pages
- Resume interrupted workflows and maintain conversation history across sessions

**Difference from today's state:**
- **Today**: Platform exclusively supports Claude Code sessions backed by Kubernetes CRs, tightly coupled to file-based git workflows
- **Future**: Multi-runner platform supporting both file-based agents (Claude Code) and graph-based orchestration (LangGraph), with database-backed session management enabling better scalability and reduced K8s resource pressure

**Out of Scope:**

- **User-scoped workflows**: Workflow definitions registered as private/personal resources (future: user-level isolation and personal resource access)
- **Multi-repo workspace support for workflows**: Workflows are graph-based, not file-system-based (file access happens via nested Claude Code sessions)
- **Workflow marketplace**: Sharing workflows across tenants or organizations
- **Workflow DAG visualization**: Real-time graph execution visualization in UI
- **LangSmith integration**: Advanced LangGraph observability and tracing
- **Custom runner types beyond LangGraph**: OpenAI Assistants, Gemini Agents, DeepSeek Agents (future extensibility designed in, implementation deferred)
- **Automatic image building**: Users must build and push images themselves (build pipeline deferred)
- **Advanced workflow validation**: Schema validation, graph structure analysis, security scanning (basic registry whitelist only)
- **Resource limits per workflow**: CPU/memory constraints enforced globally, not per-workflow
- **Migration of legacy sessions**: Existing `AgenticSession` CR-backed sessions remain unchanged

**Requirements:**

### R1: WorkflowDefinition Custom Resource (MVP)

Create cluster-scoped `WorkflowDefinition` CRD for registering LangGraph workflows:

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: WorkflowDefinition
metadata:
  name: data-analysis-workflow
spec:
  displayName: "Data Analysis Pipeline"
  description: "Analyzes CSV data and generates reports"
  image: "quay.io/approved-org/data-analysis:v1.2.3"
  imagePullSecret: "registry-credentials"  # Optional
  inputSchema:  # JSON Schema for dynamic form generation
    type: object
    properties:
      data_file:
        type: string
        description: "Path to CSV file"
      output_format:
        type: string
        enum: ["pdf", "html", "markdown"]
        default: "pdf"
    required: ["data_file"]
status:
  validated: true
  lastValidated: "2025-01-15T10:30:00Z"
  validationMessage: ""
```

**Acceptance Criteria:**
- CRD installed in cluster with proper RBAC (cluster-admin can create/update/delete)
- Workflow definitions accessible across all project namespaces
- Backend API validates image registry against whitelist during registration
- Backend API validates input schema is valid JSON Schema

---

### R2: PostgreSQL Database Schema for Workflow Sessions (MVP)

Implement database-backed workflow session management:

```sql
-- Main workflow sessions table
CREATE TABLE workflow_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_name VARCHAR(255) NOT NULL,
  workflow_definition_name VARCHAR(255) NOT NULL,
  workflow_image VARCHAR(512) NOT NULL,
  workflow_version VARCHAR(128),

  -- Session lifecycle
  status VARCHAR(50) NOT NULL, -- pending, running, completed, failed, waiting_for_input
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  started_at TIMESTAMP,
  completed_at TIMESTAMP,

  -- User context
  created_by_user VARCHAR(255) NOT NULL,

  -- Workflow data
  input_data JSONB NOT NULL,
  output_data JSONB,
  error_message TEXT,

  -- LangGraph persistence
  thread_id VARCHAR(255) UNIQUE, -- LangGraph thread ID
  checkpoint_id VARCHAR(255),     -- Latest checkpoint for resumption

  -- Kubernetes Job reference
  job_name VARCHAR(255),

  -- Metadata
  display_name VARCHAR(255),
  labels JSONB,
  annotations JSONB,

  INDEX idx_project_name (project_name),
  INDEX idx_workflow_definition (workflow_definition_name),
  INDEX idx_status (status),
  INDEX idx_created_by_user (created_by_user),
  INDEX idx_thread_id (thread_id)
);

-- LangGraph checkpoint storage (shared across workflow sessions)
CREATE TABLE langgraph_checkpoints (
  thread_id VARCHAR(255) NOT NULL,
  checkpoint_id VARCHAR(255) NOT NULL,
  parent_checkpoint_id VARCHAR(255),
  checkpoint_data JSONB NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  scope VARCHAR(50) NOT NULL, -- 'project' or 'private'
  scope_identifier VARCHAR(255) NOT NULL, -- project name or user email

  PRIMARY KEY (thread_id, checkpoint_id),
  INDEX idx_scope (scope, scope_identifier)
);

-- Session messages (for UI display and history)
CREATE TABLE workflow_session_messages (
  id BIGSERIAL PRIMARY KEY,
  session_id UUID NOT NULL REFERENCES workflow_sessions(id) ON DELETE CASCADE,
  sequence_number INT NOT NULL,
  message_type VARCHAR(50) NOT NULL, -- system.message, agent.message, user.message
  timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
  payload JSONB NOT NULL,

  INDEX idx_session_id (session_id),
  INDEX idx_sequence (session_id, sequence_number)
);
```

**Acceptance Criteria:**
- Database migrations managed via tool (e.g., golang-migrate, Atlas)
- Backend API connects to PostgreSQL using connection string from environment variable
- All workflow session CRUD operations use database (no CRs involved)
- LangGraph checkpoints stored with proper scope isolation (project or private)

---

### R3: LangGraph Base Runner Image (MVP)

Create base container image providing platform integration for user workflows:

**Base Image Structure:**
```
ambient-langgraph-runner/
├── Dockerfile
├── pyproject.toml
├── src/
│   ├── langgraph_adapter.py      # Main adapter implementing runner interface
│   ├── platform_client.py        # Backend API client for status updates
│   ├── websocket_transport.py    # WebSocket messaging to UI
│   └── checkpointer.py            # PostgreSQL checkpointer wrapper
└── entrypoint.sh
```

**Key Responsibilities:**
- Load user's LangGraph application following [LangGraph application structure](https://docs.langchain.com/oss/python/langgraph/application-structure)
- Initialize PostgreSQL-backed checkpointer using platform-provided connection string
- Execute graph with user inputs from `INPUT_DATA` environment variable
- Stream progress updates to UI via WebSocket (agent messages, thinking, tool use)
- Handle LangGraph interrupts for human-in-the-loop interactions
- Update session status in database via Backend API (`/api/v2/workflow-sessions/:id/status`)
- Capture final output and errors, update database before exit

**Environment Variables (passed by Backend when creating Job):**
```bash
# Session identification
SESSION_ID="uuid-1234"
PROJECT_NAME="my-project"
WORKFLOW_NAME="data-analysis-workflow"

# Workflow inputs (JSON-encoded based on inputSchema)
INPUT_DATA='{"data_file": "data.csv", "output_format": "pdf"}'

# Platform connectivity
BACKEND_API_URL="http://backend-service.ambient-code.svc.cluster.local:8080"
WEBSOCKET_URL="ws://backend-service.ambient-code.svc.cluster.local:8080/api/v2/workflow-sessions/uuid-1234/ws"
BOT_TOKEN="<service-account-token>"

# LangGraph persistence
THREAD_ID="langgraph-thread-abc123"
CHECKPOINT_DB_CONNECTION="postgresql://checkpointer:password@postgres.ambient-code.svc.cluster.local:5432/langgraph_checkpoints"
CHECKPOINT_SCOPE="project"  # or "private"
CHECKPOINT_SCOPE_ID="my-project"  # or user email

# Execution settings
TIMEOUT="3600"
```

**Acceptance Criteria:**
- Base image published to `quay.io/ambient_code/ambient_langgraph_runner:latest`
- Users can extend base image with their LangGraph application code
- Example Dockerfile for users:
  ```dockerfile
  FROM quay.io/ambient_code/ambient_langgraph_runner:v1.0.0

  # Copy user's workflow code
  COPY my_workflow/ /workflow/

  # Install additional dependencies if needed
  RUN pip install pandas matplotlib

  # Platform will execute: python -m langgraph_adapter
  ```
- Adapter automatically discovers and loads user's graph from standard location
- All platform interactions abstracted (users don't write Backend API or WebSocket code)

---

### R4: Backend API for Workflow Management (MVP)

Implement new API endpoints for workflow sessions:

**Workflow Definition Management:**
```
POST   /api/workflows                          # Register new workflow (cluster-admin only)
GET    /api/workflows                          # List all registered workflows
GET    /api/workflows/:name                    # Get workflow details
PUT    /api/workflows/:name                    # Update workflow (cluster-admin only)
DELETE /api/workflows/:name                    # Delete workflow (cluster-admin only)
POST   /api/workflows/:name/validate           # Validate image exists and is pullable
```

**Workflow Session Management (Project-scoped):**
```
POST   /api/v2/projects/:project/workflow-sessions       # Create new workflow session
GET    /api/v2/projects/:project/workflow-sessions       # List sessions in project
GET    /api/v2/projects/:project/workflow-sessions/:id   # Get session details
DELETE /api/v2/projects/:project/workflow-sessions/:id   # Delete session
POST   /api/v2/projects/:project/workflow-sessions/:id/continue  # Resume interrupted session
POST   /api/v2/projects/:project/workflow-sessions/:id/respond   # Respond to interrupt
GET    /api/v2/projects/:project/workflow-sessions/:id/messages  # Get session messages
WS     /api/v2/projects/:project/workflow-sessions/:id/ws        # WebSocket for real-time updates
```

**Status Update Endpoint (called by runner):**
```
PUT    /api/v2/workflow-sessions/:id/status   # Update session status (requires BOT_TOKEN auth)
```

**Job Creation Logic (Backend creates Jobs directly, bypassing Operator):**

When `POST /api/v2/projects/:project/workflow-sessions` is called:
1. Validate user has `edit` or `admin` permission on project namespace
2. Validate workflow definition exists and image registry is whitelisted
3. Insert record into `workflow_sessions` table with status `pending`
4. Generate service account token for runner (BOT_TOKEN)
5. Create LangGraph thread ID if new session (or reuse for continuation)
6. Create Kubernetes Job in project namespace with:
   - **Single container** (workflow runner using user's registered image)
   - Environment variables with session metadata and platform endpoints
   - OwnerReference to project namespace (for cleanup)
   - Labels: `app=workflow-session`, `session-id=<uuid>`, `workflow=<name>`
7. Start background goroutine to monitor Job status
8. Update database record with `job_name` and status `running`
9. Return session ID to client

**Job Monitoring (Backend polls Job status):**
- Every 10 seconds, check Job status for all `running` workflow sessions
- If Job completes successfully but session status not `completed` → mark as `failed` with error "Runner did not update final status"
- If Job fails (pod crash, OOMKilled) → update session status to `failed` with error from pod logs
- If session status becomes `completed` or `failed` → delete Job and cleanup resources

**Acceptance Criteria:**
- All workflow endpoints protected by user token authentication
- RBAC enforced: users can only access sessions in projects they have permission for
- Cluster-admin role required to register/update/delete workflow definitions
- Registry whitelist enforced during workflow registration (configurable via env var or ConfigMap)
- Backend creates Jobs directly (no Operator involvement for workflow sessions)
- WebSocket connection established before runner starts execution
- Status updates from runner properly authenticated via BOT_TOKEN

---

### R5: Frontend UI for Workflow Management (MVP)

**Component 1: Workflow Registry Page (Cluster-level)**

New route: `/workflows`

Features:
- List all registered WorkflowDefinitions with cards showing:
  - Display name and description
  - Container image reference
  - Number of active sessions using this workflow
  - Registration date
- "Register New Workflow" button (visible only to cluster-admins)
- Registration modal/form with fields:
  - Name (kebab-case identifier)
  - Display name
  - Description
  - Container image URL
  - Image pull secret (optional dropdown of secrets)
  - Input schema (JSON Schema textarea with validation)
- View workflow details modal showing full spec and input schema
- Delete workflow action (with confirmation, only if no active sessions)

**Component 2: New Workflow Session Page**

New route: `/projects/:project/workflow-sessions/new`

Two-step form:
1. **Select Workflow**: Dropdown of registered workflows with descriptions
2. **Configure Inputs**: Dynamic form generated from workflow's `inputSchema`
   - Use `react-hook-form` + Shadcn UI components
   - Render form fields based on JSON Schema types:
     - `string` → Input
     - `number` → Input type="number"
     - `boolean` → Checkbox
     - `enum` → Select dropdown
     - `object` → Nested fieldset
     - `array` → Dynamic list with add/remove buttons
   - Show field descriptions as helper text
   - Validate required fields and types before submission
   - Optional: Display name for session
   - Optional: Scope selector (project-scoped vs private) [future: private scope]

**Component 3: Workflow Session List (Project-scoped)**

Route: `/projects/:project/sessions`

Unified session list showing both:
- **Legacy Claude Code sessions** (fetched from existing `/api/projects/:project/agentic-sessions`)
- **Workflow sessions** (fetched from `/api/v2/projects/:project/workflow-sessions`)

Combined list with columns:
- Type badge ("Claude Code" | "Workflow: {name}")
- Display name / Session ID
- Status (with appropriate badges)
- Created by
- Created at
- Actions (View, Delete, Continue if applicable)

**Component 4: Workflow Session Detail Page**

New route: `/projects/:project/workflow-sessions/:id`

Layout:
- **Header**: Session metadata (workflow name, status, created by, timestamps)
- **Breadcrumbs**: Projects → {project} → Sessions → {session}
- **Message Stream**: Real-time messages from WebSocket
  - System messages (status updates)
  - Agent messages (workflow progress, thinking, tool use)
  - User messages (responses to interrupts)
- **Input Panel**: Show original input data (formatted JSON)
- **Output Panel**: Show final output data when completed
- **Interactive Controls**:
  - "Respond" button when status is `waiting_for_input` (human-in-the-loop)
  - "Continue" button to resume interrupted session
  - "Delete" button to cancel/cleanup session
- **Error Display**: Show error message if status is `failed`

**Component 5: Update Main Session Creation Flow**

Route: `/projects/:project/sessions/new`

Add runner type selector at top of form:
- **Radio buttons or Tabs**:
  - `[Claude Code Session]` → Existing form (repos, prompt, interactive mode)
  - `[Workflow Session]` → Redirect to `/projects/:project/workflow-sessions/new`

**Acceptance Criteria:**
- Workflow registry page accessible from main navigation (e.g., "Workflows" link in sidebar)
- Only cluster-admins see "Register New Workflow" button
- Dynamic form generation supports all common JSON Schema types
- Form validation matches inputSchema requirements (required fields, type constraints)
- Session list shows unified view of legacy and workflow sessions with visual distinction
- WebSocket connection established for real-time message streaming on detail page
- Loading states (Skeleton components) shown while fetching data
- Empty states shown when no workflows or sessions exist
- Error boundaries catch and display errors gracefully
- All buttons show loading state during async operations

---

### R6: LangGraph Feature Integration (MVP)

**Persistence:**
- Backend provides PostgreSQL connection string to runner via environment variable
- Runner uses `langgraph.checkpoint.postgres.PostgresSaver`
- Checkpoints stored in `langgraph_checkpoints` table with proper scope isolation
- Thread IDs generated by Backend and passed to runner (reused for continuations)

**Interrupts (Human-in-the-Loop):**
- When LangGraph graph hits interrupt node, runner updates session status to `waiting_for_input`
- Runner sends WebSocket message to UI with interrupt details (prompt, expected input type)
- UI displays "Respond" button and input form
- User submits response via `POST /api/v2/workflow-sessions/:id/respond`
- Backend stores response in database and wakes runner (via signal or polling mechanism)
- Runner resumes graph execution with user's response

**Memory:**
- LangGraph's built-in memory features work automatically via checkpointer
- Conversation history stored in checkpoint data
- Session continuation loads previous checkpoints to maintain context

**Acceptance Criteria:**
- Workflow sessions can be interrupted and resumed across multiple executions
- Checkpoint data properly scoped (project-level sessions share checkpoints within project)
- Runner correctly loads previous state when continuing session
- Human-in-the-loop workflows display prompt and accept user input via UI
- Memory persists across session continuations (conversation history maintained)

---

### R7: Registry Whitelist Security (MVP)

Implement registry access control:

**Configuration:**
- Environment variable or ConfigMap: `ALLOWED_WORKFLOW_REGISTRIES`
- Format: Comma-separated list of registry prefixes
- Example: `quay.io/approved-org,gcr.io/company-prod,docker.io/company`

**Validation:**
- During workflow registration (`POST /api/workflows`), check image URL starts with allowed prefix
- Return `403 Forbidden` if image registry not whitelisted
- During validation (`POST /api/workflows/:name/validate`), attempt image pull to verify accessibility

**Acceptance Criteria:**
- Registry whitelist configurable via environment variable
- Workflow registration fails with clear error message if registry not allowed
- Image validation checks pullability (returns error if image doesn't exist or credentials invalid)

---

### R8: Example Workflow - SpecKit Graph (MVP)

Provide reference implementation demonstrating LangGraph integration:

**Repository Structure:**
```
ambient-langgraph-examples/
└── speckit-workflow/
    ├── README.md
    ├── Dockerfile
    ├── pyproject.toml
    ├── workflow/
    │   ├── __init__.py
    │   ├── graph.py           # Main LangGraph graph definition
    │   ├── nodes/
    │   │   ├── specify.py     # Create spec.md
    │   │   ├── plan.py        # Create plan.md
    │   │   ├── tasks.py       # Create tasks.md
    │   │   └── analyze.py     # Cross-artifact analysis
    │   └── state.py           # Graph state definition
    └── workflow-definition.yaml  # WorkflowDefinition CR for registration
```

**Graph Structure:**
```
User Input (feature description)
  ↓
[specify] Create spec.md
  ↓
[Interrupt: Review spec.md]
  ↓
[plan] Create plan.md based on spec
  ↓
[Interrupt: Review plan.md]
  ↓
[tasks] Generate tasks.md
  ↓
[analyze] Cross-artifact consistency check
  ↓
Output: {spec, plan, tasks, analysis}
```

**Input Schema:**
```yaml
inputSchema:
  type: object
  properties:
    feature_description:
      type: string
      description: "Natural language description of feature to specify"
    project_context:
      type: string
      description: "Optional: Additional project context or constraints"
    auto_approve_steps:
      type: boolean
      default: false
      description: "Skip human review interrupts (for testing)"
  required: ["feature_description"]
```

**Acceptance Criteria:**
- Example workflow demonstrates:
  - Multi-step graph with sequential nodes
  - Human-in-the-loop interrupts (spec/plan review)
  - Persistent state across interrupts
  - Structured output (documents as artifacts)
- README includes:
  - How to build and push container image
  - How to register workflow in platform
  - How to execute workflow via UI
  - Example inputs and expected outputs
- Dockerfile extends `ambient_langgraph_runner` base image
- Pre-built image published to `quay.io/ambient_code/example_speckit_workflow:latest`
- WorkflowDefinition YAML provided for easy registration

---

**Done - Acceptance Criteria:**

**End-to-End User Flow:**
1. ✅ Cluster admin registers a LangGraph workflow via UI (`/workflows`) with container image and input schema
2. ✅ System validates image registry is whitelisted and image is pullable
3. ✅ WorkflowDefinition CR created and visible in cluster
4. ✅ Project user navigates to "New Workflow Session" and selects registered workflow from dropdown
5. ✅ Dynamic form renders based on workflow's input schema with proper validation
6. ✅ User fills form and submits, creating database record and Kubernetes Job
7. ✅ Job starts, runner connects to Backend WebSocket, begins executing LangGraph graph
8. ✅ Progress messages stream to UI in real-time (agent messages, thinking, status updates)
9. ✅ When graph hits interrupt node, UI displays "Respond" button with prompt
10. ✅ User provides input, workflow resumes execution with response
11. ✅ Workflow completes, final output displayed in UI, session status updated to `completed`
12. ✅ User can view session details, messages history, and output data
13. ✅ User can continue/resume interrupted sessions, maintaining conversation history
14. ✅ Legacy Claude Code sessions continue to work unchanged (coexist with workflow sessions)

**System Integration:**
- ✅ Database migrations applied successfully, tables created
- ✅ Backend API endpoints functional with proper authentication and RBAC
- ✅ Backend creates and monitors Jobs directly (Operator not involved)
- ✅ Runner base image published and documented
- ✅ Example SpecKit workflow registered and executable
- ✅ UI components built with TypeScript strict mode (zero `any` types)
- ✅ UI uses Shadcn components and React Query for data management
- ✅ WebSocket connections stable, messages delivered in real-time
- ✅ Error handling comprehensive (failed jobs, crashed pods, runner errors)
- ✅ Session cleanup works (deleted sessions remove Jobs and database records)

**Technical Quality:**
- ✅ All Go code passes `gofmt`, `go vet`, and `golangci-lint`
- ✅ All TypeScript code builds with zero errors/warnings (`npm run build`)
- ✅ Database schema includes proper indexes for query performance
- ✅ API responses include structured error messages
- ✅ Security contexts applied to Job pods (non-root, dropped capabilities)
- ✅ Secrets (BOT_TOKEN, DB credentials) properly managed via Kubernetes Secrets
- ✅ Documentation updated (architecture diagrams, API reference, user guides)

---

**Use Cases - i.e. User Experience & Workflow:**

### Use Case 1: Data Analyst - Custom Analysis Pipeline

**Actors:** Sarah (Data Analyst), Ambient Code Platform

**Preconditions:**
- Sarah's team built a LangGraph workflow (`data-pipeline:v2.1`) for CSV analysis
- Workflow registered in platform by cluster admin
- Sarah has `edit` permission on `analytics-project` namespace

**Main Success Scenario:**
1. Sarah navigates to `/projects/analytics-project/workflow-sessions/new`
2. Selects "Data Analysis Pipeline" from workflow dropdown
3. Dynamic form appears with fields:
   - `dataset_url` (string, required) - Sarah enters: `https://s3.amazonaws.com/company/sales-q4.csv`
   - `analysis_type` (enum: trend, forecast, anomaly) - Sarah selects: `forecast`
   - `time_period` (number) - Sarah enters: `90` (days)
4. Sarah clicks "Start Workflow"
5. System creates workflow session in database, spawns Job
6. UI redirects to `/projects/analytics-project/workflow-sessions/{uuid}`
7. Sarah sees real-time progress:
   - "Downloading dataset..." (system message)
   - "Validating data schema..." (agent message)
   - "Running statistical analysis..." (agent message)
8. Workflow hits interrupt: "Dataset has 3 outliers. Remove them? (yes/no)"
9. Sarah sees "Respond" button, clicks it, enters: `yes`
10. Workflow continues: "Generating forecast model..."
11. After 5 minutes, workflow completes
12. Sarah sees output panel with:
    - Forecast chart URL
    - Statistical summary (JSON)
    - Confidence intervals
13. Sarah downloads results and shares with team

**Alternative Flows:**
- **AF1: Invalid Input**: Sarah enters invalid dataset URL → Form validation shows error before submission
- **AF2: Workflow Fails**: Dataset too large, Job OOMKilled → UI shows status `failed` with error "Pod exceeded memory limit"
- **AF3: Timeout**: Workflow exceeds configured timeout → Backend marks session as `failed` with "Execution timeout exceeded"

---

### Use Case 2: Engineering Manager - Review-and-Refine Workflow

**Actors:** Alex (Engineering Manager), SpecKit Workflow, Ambient Code Platform

**Preconditions:**
- SpecKit LangGraph workflow registered (`example_speckit_workflow:latest`)
- Alex has `admin` permission on `platform-team` project

**Main Success Scenario:**
1. Alex creates new workflow session for feature: "Add user authentication to API"
2. Enters input: `{"feature_description": "Add JWT-based authentication to REST API", "auto_approve_steps": false}`
3. Workflow starts, creates `spec.md` in first node
4. Workflow interrupts: "Review generated spec.md. Approve or provide feedback?"
5. Alex reviews spec in UI, notices missing requirement
6. Alex responds: "Add requirement for OAuth 2.0 provider support"
7. Workflow updates spec.md, regenerates
8. Workflow shows updated spec, Alex responds: "Approved"
9. Workflow proceeds to planning phase, generates `plan.md`
10. Workflow interrupts again for plan review
11. Alex approves plan
12. Workflow generates `tasks.md` with 15 actionable tasks
13. Workflow runs consistency analysis across all artifacts
14. Workflow completes, Alex sees all generated documents as output
15. Alex exports documents to project repository

**Alternative Flows:**
- **AF1: Multi-iteration Refinement**: Alex requests changes multiple times → Workflow maintains context, updates artifacts incrementally
- **AF2: Session Interruption**: Alex closes browser during interrupt → Returns later, continues from checkpoint
- **AF3: Collaborative Review**: Alex shares session URL with teammate → Both can view progress (future: both can respond)

---

### Use Case 3: Platform Admin - Register New Workflow

**Actors:** Morgan (Platform Admin), Ambient Code Platform

**Preconditions:**
- Morgan has `cluster-admin` role
- Workflow container built and pushed to `quay.io/company/risk-assessment:v1.0.0`
- Registry `quay.io/company` whitelisted in platform config

**Main Success Scenario:**
1. Morgan navigates to `/workflows`
2. Clicks "Register New Workflow"
3. Registration form appears, Morgan fills:
   - **Name**: `risk-assessment-workflow`
   - **Display Name**: "Security Risk Assessment"
   - **Description**: "Analyzes codebase for security vulnerabilities and generates risk report"
   - **Container Image**: `quay.io/company/risk-assessment:v1.0.0`
   - **Image Pull Secret**: (dropdown) → selects `quay-credentials`
   - **Input Schema** (JSON):
     ```json
     {
       "type": "object",
       "properties": {
         "repo_url": {"type": "string", "format": "uri"},
         "scan_depth": {"type": "string", "enum": ["quick", "standard", "deep"], "default": "standard"}
       },
       "required": ["repo_url"]
     }
     ```
4. Morgan clicks "Validate Image"
5. System attempts to pull image → Success message: "Image validated successfully"
6. Morgan clicks "Register Workflow"
7. Backend creates WorkflowDefinition CR
8. Morgan sees new workflow in list with status "Active"
9. Morgan shares workflow name with team: "New 'Security Risk Assessment' workflow available!"
10. Team members can now select workflow when creating sessions

**Alternative Flows:**
- **AF1: Invalid Registry**: Morgan enters image from non-whitelisted registry → Error: "Registry not allowed. Contact platform admin to whitelist."
- **AF2: Image Pull Failure**: Image doesn't exist or credentials invalid → Error: "Failed to pull image. Verify image URL and pull secret."
- **AF3: Invalid Schema**: Morgan enters malformed JSON Schema → Form validation error before submission

---

### Use Case 4: Developer - Resume Interrupted Long-Running Workflow

**Actors:** Jordan (Developer), Ambient Code Platform

**Preconditions:**
- Jordan started a workflow session 2 days ago
- Workflow interrupted at step 5 of 10 (waiting for approval)
- Session still in `waiting_for_input` status

**Main Success Scenario:**
1. Jordan navigates to `/projects/dev-team/sessions`
2. Sees workflow session in list with status badge "Waiting for Input"
3. Clicks session to view details
4. Sees conversation history:
   - Step 1-4 completed messages
   - Current interrupt prompt: "Approve database schema changes?"
5. Jordan reviews proposed schema in output panel
6. Jordan clicks "Respond", enters: "Approved with modification: add index on user_id column"
7. Workflow resumes from checkpoint, applies modification
8. Jordan sees new message: "Schema updated with index. Proceeding to step 6..."
9. Workflow continues through remaining steps
10. Workflow completes 10 minutes later
11. Jordan sees final output with all results

**Alternative Flows:**
- **AF1: Session Timeout**: Session exceeded max idle time → Status changed to `expired`, Jordan can view history but cannot continue
- **AF2: Checkpoint Corruption**: Database checkpoint data corrupted → Error displayed, Jordan must restart workflow from beginning

---

**Documentation Considerations:**

### User Documentation

**Getting Started Guide** (`docs/user-guide/workflows.md`):
- Overview of workflow concepts (vs. Claude Code sessions)
- When to use workflows vs. Claude Code
- How to browse available workflows
- How to create and execute workflow sessions
- How to respond to interrupts (human-in-the-loop)
- How to resume interrupted sessions
- Troubleshooting common issues (timeouts, failed jobs, invalid inputs)

**Workflow Author Guide** (`docs/developer-guide/building-workflows.md`):
- LangGraph primer and application structure
- How to extend the `ambient_langgraph_runner` base image
- Required environment variables and platform integration
- How to define input/output schemas (JSON Schema)
- Testing workflows locally before registration
- Best practices for checkpoint/memory usage
- Example: Building a simple workflow from scratch
- Example: Complex workflow with interrupts

**Administrator Guide** (`docs/admin-guide/workflow-registry.md`):
- How to register/update/delete workflows
- Registry whitelist configuration
- Image validation process
- RBAC requirements (cluster-admin role)
- Security considerations (untrusted workflow code)
- Monitoring workflow execution (logs, metrics)
- Troubleshooting failed registrations

**API Reference** (`docs/api-reference/workflows.md`):
- Full OpenAPI/Swagger spec for workflow endpoints
- Authentication and authorization requirements
- Request/response examples
- Error codes and messages
- WebSocket protocol for real-time updates

### Developer Documentation

**Architecture Documentation** (`docs/architecture/workflows.md`):
- System architecture diagram (Frontend → Backend → PostgreSQL + K8s Jobs)
- Sequence diagram: Workflow session lifecycle
- Database schema documentation with ER diagrams
- Comparison: Legacy CR-backed sessions vs. DB-backed workflows
- Job creation and monitoring flow
- LangGraph integration architecture

**Migration Guide** (`docs/developer-guide/workflow-migration.md`):
- Roadmap for migrating legacy sessions to database-backed model (future)
- Comparison of CR-based vs. DB-based approaches
- Plan for deprecating AgenticSession CRs (timeline TBD)

**Testing Guide** (`docs/developer-guide/testing-workflows.md`):
- Unit testing LangGraph workflows
- Integration testing with platform (local dev environment)
- Contract testing for platform APIs
- End-to-end testing workflow sessions

---

**Questions to answer:**

### Technical Architecture Questions

1. **Database Connection Pooling**: Should Backend use a connection pool for PostgreSQL? What pool size is appropriate for expected load? (PgBouncer vs. application-level pooling)

2. **Job Cleanup Strategy**: When should completed/failed Jobs be deleted? Immediate vs. retention period (e.g., keep for 24 hours for debugging)?

3. **WebSocket Scalability**: Current WebSocket design assumes single Backend pod. How do we handle multiple Backend replicas? (Redis pub/sub, Kubernetes Lease-based coordination, sticky sessions)

4. **Checkpoint Database Sizing**: What are expected checkpoint data sizes? Should we implement compression, TTL-based cleanup, or archival strategy for old checkpoints?

5. **Runner Crash Handling**: If runner pod crashes without updating status, how long should Backend wait before marking session as failed? (Dead letter queue, exponential backoff)

6. **Image Pull Secrets**: Should image pull secrets be namespace-scoped or cluster-scoped? How do we manage credentials for multiple registries?

7. **Interrupt Response Mechanism**: How does runner know when user responds to interrupt? Polling vs. push notification? (WebSocket bidirectional, pub/sub, database trigger)

8. **Concurrent Session Limits**: Should we enforce max concurrent workflow sessions per project or per user? (Resource quota management)

9. **Audit Logging**: Should we log all workflow registrations, session creations, and admin actions? Where (database table, external service, K8s events)?

10. **GraphQL vs. REST**: Future consideration - should workflow APIs use GraphQL for better querying flexibility (nested messages, filtering)?

### Security & Compliance Questions

11. **User Code Execution Isolation**: What runtime security measures should we implement? (gVisor, Kata Containers, seccomp profiles, AppArmor/SELinux)

12. **Sensitive Data in Inputs**: If workflow inputs contain PII or secrets, should we encrypt `input_data` JSONB column at rest? Key management strategy?

13. **Registry Authentication**: Should we support private registries with authentication? How to securely store registry credentials (Kubernetes Secrets, external vault)?

14. **Network Policies**: Should workflow runner pods have restricted network access? (Egress-only, allowlist of external services, block pod-to-pod communication)

15. **RBAC Granularity**: Should we add more fine-grained permissions? (e.g., `workflow-viewer`, `workflow-executor`, `workflow-admin` roles)

### Product & UX Questions

16. **Workflow Versioning Strategy**: How should users manage workflow versions? (Semantic versioning in image tags, separate WorkflowDefinition per version, version history in UI)

17. **Session Sharing**: Should users be able to share workflow sessions with other users? Read-only vs. collaborative editing? (URL-based sharing, ACLs)

18. **Workflow Templates**: Should we provide a library of starter templates? How to distribute (Git repo, in-platform catalog, external marketplace)?

19. **Cost Tracking**: Should we track workflow execution costs (compute time, LLM API calls if applicable)? Display in UI, set budget alerts?

20. **Notification System**: Should users receive notifications when long-running workflows complete or require input? (Email, Slack, in-app notifications)

21. **Workflow Composition**: Future - should workflows be able to invoke other workflows as sub-graphs? How to handle nesting, circular dependencies?

22. **Input Presets/Favorites**: Should users be able to save common input configurations as presets for faster session creation?

23. **Bulk Operations**: Should admins be able to bulk-register workflows from a Git repo or manifest file?

24. **Workflow Execution History**: Should we show aggregate statistics per workflow? (Total executions, success rate, average duration, most common errors)

25. **Dark Mode for Graph Visualization**: Future - when we add DAG visualization, should it support dark mode? Accessibility considerations (WCAG compliance)?

---

**Background & Strategic Fit:**

### Market Context

The AI agent orchestration space is rapidly evolving beyond single-provider solutions. Organizations are adopting **multi-agent architectures** where specialized agents collaborate to solve complex problems. LangGraph has emerged as a leading open-source framework for building stateful, multi-step agent workflows with features like:

- **Persistence**: Long-running workflows that survive restarts
- **Human-in-the-loop**: Interrupts for approval gates and interactive refinement
- **Memory**: Conversational context across multiple sessions
- **Composability**: Graphs as reusable building blocks

Meanwhile, the Ambient Code Platform currently tightly couples orchestration to Claude Code, limiting extensibility. As competitors (Replit, Cursor, GitHub Copilot Workspace) expand beyond single-agent models, we risk falling behind.

### Strategic Drivers

1. **Extensibility**: Position platform as **orchestration hub** rather than Claude Code wrapper
2. **User Empowerment**: Enable users to bring custom AI workflows without platform team bottleneck
3. **Competitive Differentiation**: Unique value prop = "Run any AI workflow, not just coding agents"
4. **Scalability**: Database-backed sessions address K8s resource pressure (CR bloat, etcd limits)
5. **Future-Proofing**: Architecture supports upcoming runners (OpenAI Assistants API, Gemini, DeepSeek)

### Alignment with Product Vision

This feature advances three strategic pillars:

**Pillar 1: Platform Generalization**
- Move from "vTeam = Claude Code platform" to "vTeam = AI agent platform"
- Enable workflows beyond software development (data analysis, content generation, research)

**Pillar 2: Developer Experience**
- Reduce cognitive load: Users bring workflows, platform handles infrastructure
- Clear separation: Workflow logic vs. platform orchestration

**Pillar 3: Enterprise Readiness**
- Database-backed model enables better auditing, analytics, and compliance
- Cluster-wide workflows support reusability across teams (governance at scale)

### Technical Debt & Migration Path

This feature intentionally creates a **dual-mode system** (legacy CRs + new DB) to:
- De-risk rollout: Test new architecture without disrupting existing users
- Gather feedback: Validate DB model before migrating legacy sessions
- Incremental transition: Phase out CRs once new model proven stable

**Planned Migration Phases** (post-MVP):
1. **Phase 1** (MVP): New workflow sessions use DB, legacy sessions use CRs (coexistence)
2. **Phase 2** (Q2 2025): Migrate existing Claude Code sessions to DB, deprecate AgenticSession CR
3. **Phase 3** (Q3 2025): Operator watches DB instead of CRs, unified orchestration model
4. **Phase 4** (Q4 2025): Remove CR dependencies entirely, pure DB-backed platform

---

**Customer Considerations**

### Target Customer Segments

**Segment 1: Data Science Teams**
- **Needs**: Custom analysis pipelines, Jupyter-style workflows, result visualization
- **Workflows**: Data cleaning → Statistical analysis → Report generation
- **Key Requirement**: Interrupt-driven review (human validation of outliers, model parameters)
- **Example Customer**: Financial services firm running risk analysis workflows

**Segment 2: Content Operations Teams**
- **Needs**: Content generation, SEO optimization, multi-step editorial workflows
- **Workflows**: Topic research → Draft creation → Fact-checking → Publishing
- **Key Requirement**: Human-in-the-loop for editorial approval
- **Example Customer**: Media company automating blog post production

**Segment 3: DevOps/Platform Engineering Teams**
- **Needs**: Infrastructure automation, runbook execution, incident response
- **Workflows**: Alert triage → Root cause analysis → Remediation → Postmortem
- **Key Requirement**: Session persistence across long-running incidents
- **Example Customer**: SaaS company automating incident response

**Segment 4: Research Teams**
- **Needs**: Literature review, experiment design, data synthesis
- **Workflows**: Query formulation → Source discovery → Summarization → Citation management
- **Key Requirement**: Memory/context across multi-day research sessions
- **Example Customer**: Pharmaceutical company accelerating drug discovery research

### Customer Success Metrics

**Adoption Metrics:**
- Number of registered workflows per organization (target: 5+ within 90 days)
- Workflow session execution volume (target: 50+ sessions/week per active team)
- Workflow reuse rate (target: Each workflow executed 10+ times)

**Engagement Metrics:**
- Interactive session completion rate (target: 80%+ of interrupted sessions resumed)
- Average session duration (indicator of workflow complexity support)
- User retention (target: 70%+ of users create 2nd workflow session within 30 days)

**Satisfaction Metrics:**
- Workflow author NPS (target: 50+)
- Time-to-first-workflow (target: <2 hours from registration to successful execution)
- Support ticket volume (target: <5% of sessions result in support requests)

### Customer Migration Support

**For Existing Claude Code Users:**
- No disruption: Legacy sessions continue working
- Educational content: "When to use workflows vs. Claude Code" decision tree
- Migration incentive: "Try workflows" campaign with example use cases

**For New Customers:**
- Onboarding flow: Show both options (Claude Code + workflows) with guided tour
- Template library: Pre-built workflows for common use cases (reduce time-to-value)
- Office hours: Weekly demo sessions showing workflow development

### Customer Feedback Integration

**Alpha Testing (Pre-MVP):**
- Recruit 5 early adopters from different segments
- Weekly feedback sessions during development
- Focus areas: UX friction points, missing features, performance issues

**Beta Testing (Post-MVP):**
- Expand to 20 customers across all segments
- Track metrics dashboard (adoption, engagement, satisfaction)
- Prioritize top 3 feature requests for post-MVP roadmap

**General Availability:**
- In-app feedback widget on workflow pages
- Quarterly customer advisory board meetings
- Public roadmap with voting (community-driven prioritization)

### Customer Risk Mitigation

**Risk 1: Workflow Development Complexity**
- **Mitigation**: Comprehensive docs, video tutorials, reference implementations
- **Fallback**: Professional services offering (workflow development as a service)

**Risk 2: Performance/Scalability Issues**
- **Mitigation**: Load testing before GA, auto-scaling infrastructure, resource quotas
- **Fallback**: Priority support SLA for enterprise customers

**Risk 3: Security Concerns (Untrusted Code)**
- **Mitigation**: Registry whitelist, security scanning (future), runtime isolation (gVisor)
- **Fallback**: Private deployment option (air-gapped clusters)

**Risk 4: Migration Confusion (Dual-Mode System)**
- **Mitigation**: Clear UI indicators (legacy badge), migration guides, deprecation timeline
- **Fallback**: Extended legacy support (CR-backed sessions supported through 2025)

---

## Summary

This feature transforms the Ambient Code Platform from a Claude Code-specific tool into a **generic AI workflow orchestration platform** by introducing LangGraph workflow support. The database-backed session model addresses scalability concerns while the cluster-wide workflow registry enables reusability across teams. By providing a base runner image and abstracting platform integration, we empower users to bring custom workflows without needing to understand backend APIs or Kubernetes. The dual-mode architecture (legacy CRs + new DB) enables safe rollout and incremental migration. Strategic focus on human-in-the-loop workflows, persistence, and memory positions the platform for enterprise adoption across diverse use cases beyond software development.
