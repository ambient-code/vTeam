# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The **Ambient Code Platform** is a Kubernetes-native AI automation platform that orchestrates intelligent agentic sessions through containerized microservices. The platform enables AI-powered automation for analysis, research, development, and content creation tasks via a modern web interface.

> **Note:** This project was formerly known as "vTeam". Technical artifacts (image names, namespaces, API groups) still use "vteam" for backward compatibility.

### Amber Background Agent

The platform includes **Amber**, a background agent that automates common development tasks via GitHub Issues. Team members can trigger automated fixes, refactoring, and test additions without requiring direct access to Claude Code.

**Quick Links**:

- [Amber Quickstart](docs/amber-quickstart.md) - Get started in 5 minutes
- [Full Documentation](docs/amber-automation.md) - Complete automation guide
- [Amber Config](.claude/amber-config.yml) - Automation policies

**Common Workflows**:

- 🤖 **Auto-Fix** (label: `amber:auto-fix`): Formatting, linting, trivial fixes
- 🔧 **Refactoring** (label: `amber:refactor`): Break large files, extract patterns
- 🧪 **Test Coverage** (label: `amber:test-coverage`): Add missing tests

### Core Architecture

The system follows a Kubernetes-native pattern with Custom Resources, Operators, and Job execution:

1. **Frontend** (NextJS + Shadcn): Web UI for session management and monitoring
2. **Backend API** (Go + Gin): REST API managing Kubernetes Custom Resources with multi-tenant project isolation
3. **Agentic Operator** (Go): Kubernetes controller watching CRs and creating Jobs
4. **Claude Code Runner** (Python): Job pods executing Claude Code CLI with multi-agent collaboration

### Agentic Session Flow

```
User Creates Session → Backend Creates CR → Operator Spawns Job →
Pod Runs Claude CLI → Results Stored in CR → UI Displays Progress
```

## Memory System - Loadable Context

This repository uses a structured **memory system** to provide targeted, loadable context instead of relying solely on this comprehensive CLAUDE.md file.

### Quick Reference

**Load these files when working in specific areas:**

| Task Type | Context File | Repomix View | Pattern File |
|-----------|--------------|--------------|--------------|
| **Backend API work** | `.claude/context/backend-development.md` | `repomix-analysis/04-backend-focused.xml` | `.claude/patterns/k8s-client-usage.md` |
| **Frontend UI work** | `.claude/context/frontend-development.md` | `repomix-analysis/05-frontend-focused.xml` | `.claude/patterns/react-query-usage.md` |
| **Security review** | `.claude/context/security-standards.md` | `repomix-analysis/02-production-optimized.xml` | `.claude/patterns/error-handling.md` |
| **Architecture questions** | - | `repomix-analysis/03-architecture-only.xml` | See ADRs below |

### Available Memory Files

**1. Context Files** (`.claude/context/`)

- `backend-development.md` - Go backend, K8s integration, handler patterns
- `frontend-development.md` - NextJS, Shadcn UI, React Query patterns
- `security-standards.md` - Auth, RBAC, token handling, security patterns

**2. Architectural Decision Records** (`docs/adr/`)

- Documents WHY decisions were made, not just WHAT
- `0001-kubernetes-native-architecture.md`
- `0002-user-token-authentication.md`
- `0003-multi-repo-support.md`
- `0004-go-backend-python-runner.md`
- `0005-nextjs-shadcn-react-query.md`

**3. Code Pattern Catalog** (`.claude/patterns/`)

- `error-handling.md` - Consistent error patterns (backend, operator, runner)
- `k8s-client-usage.md` - When to use user token vs. service account
- `react-query-usage.md` - Data fetching patterns (queries, mutations, caching)

**4. Repomix Usage Guide** (`.claude/repomix-guide.md`)

- How to use the 7 existing repomix views effectively
- When to use each view based on the task

**5. Decision Log** (`docs/decisions.md`)

- Lightweight chronological record of major decisions
- Links to ADRs, code, and context files

### Example Usage

```
"Claude, load the backend-development context file and the backend-focused repomix view (04),
then help me add a new endpoint for listing RFE workflows in a project."
```

```
"Claude, reference the security-standards context file and review this PR for token handling issues."
```

```
"Claude, check ADR-0002 (User Token Authentication) and explain why we use user tokens
instead of service accounts for API operations."
```

## Development Commands

### Quick Start - Local Development

**Single command setup with OpenShift Local (CRC):**

```bash
# Prerequisites: brew install crc
# Get free Red Hat pull secret from console.redhat.com/openshift/create/local
make dev-start

# Access at https://vteam-frontend-vteam-dev.apps-crc.testing
```

**Hot-reloading development:**

```bash
# Terminal 1
DEV_MODE=true make dev-start

# Terminal 2 (separate terminal)
make dev-sync
```

### Building Components

```bash
# Build all container images (default: docker, linux/amd64)
make build-all

# Build with podman
make build-all CONTAINER_ENGINE=podman

# Build for ARM64
make build-all PLATFORM=linux/arm64

# Build individual components
make build-frontend
make build-backend
make build-operator
make build-runner

# Push to registry
make push-all REGISTRY=quay.io/your-username
```

### Deployment

```bash
# Deploy with default images from quay.io/ambient_code
make deploy

# Deploy to custom namespace
make deploy NAMESPACE=my-namespace

# Deploy with custom images
cd components/manifests
cp env.example .env
# Edit .env with ANTHROPIC_API_KEY and CONTAINER_REGISTRY
./deploy.sh

# Clean up deployment
make clean
```

### Component Development

See component-specific documentation for detailed development commands:

- **Backend** (`components/backend/README.md`): Go API development, testing, linting
- **Frontend** (`components/frontend/README.md`): NextJS development, see also `DESIGN_GUIDELINES.md`
- **Operator** (`components/operator/README.md`): Operator development, watch patterns
- **Claude Code Runner** (`components/runners/claude-code-runner/README.md`): Python runner development

**Common commands**:

```bash
make build-all         # Build all components
make deploy            # Deploy to cluster
make test              # Run tests
make lint              # Lint code
```

### Documentation

```bash
# Install documentation dependencies
pip install -r requirements-docs.txt

# Serve locally at http://127.0.0.1:8000
mkdocs serve

# Build static site
mkdocs build

# Deploy to GitHub Pages
mkdocs gh-deploy

# Markdown linting
markdownlint docs/**/*.md
```

### Local Development Helpers

```bash
# View logs
make dev-logs              # Both backend and frontend
make dev-logs-backend      # Backend only
make dev-logs-frontend     # Frontend only
make dev-logs-operator     # Operator only

# Operator management
make dev-restart-operator  # Restart operator deployment
make dev-operator-status   # Show operator status and events

# Cleanup
make dev-stop              # Stop processes, keep CRC running
make dev-stop-cluster      # Stop processes and shutdown CRC
make dev-clean             # Stop and delete OpenShift project

# Testing
make dev-test              # Run smoke tests
make dev-test-operator     # Test operator only
```

## Key Architecture Patterns

### Custom Resource Definitions (CRDs)

The platform defines three primary CRDs:

1. **AgenticSession** (`agenticsessions.vteam.ambient-code`): Represents an AI execution session
   - Spec: prompt, repos (multi-repo support), interactive mode, timeout, model selection
   - Status: phase, startTime, completionTime, results, error messages, per-repo push status

2. **ProjectSettings** (`projectsettings.vteam.ambient-code`): Project-scoped configuration
   - Manages API keys, default models, timeout settings
   - Namespace-isolated for multi-tenancy

3. **RFEWorkflow** (`rfeworkflows.vteam.ambient-code`): RFE (Request For Enhancement) workflows
   - 7-step agent council process for engineering refinement
   - Agent roles: PM, Architect, Staff Engineer, PO, Team Lead, Team Member, Delivery Owner

### Multi-Repo Support

AgenticSessions support operating on multiple repositories simultaneously:

- Each repo has required `input` (URL, branch) and optional `output` (fork/target) configuration
- `mainRepoIndex` specifies which repo is the Claude working directory (default: 0)
- Per-repo status tracking: `pushed` or `abandoned`

### Interactive vs Batch Mode

- **Batch Mode** (default): Single prompt execution with timeout
- **Interactive Mode** (`interactive: true`): Long-running chat sessions using inbox/outbox files

### Backend API Structure

The Go backend (`components/backend/`) implements:

- **Project-scoped endpoints**: `/api/projects/:project/*` for namespaced resources
- **Multi-tenant isolation**: Each project maps to a Kubernetes namespace
- **WebSocket support**: Real-time session updates via `websocket_messaging.go`
- **Git operations**: Repository cloning, forking, PR creation via `git.go`
- **RBAC integration**: OpenShift OAuth for authentication

Main handler logic in `handlers.go` (3906 lines) manages:

- Project CRUD operations
- AgenticSession lifecycle
- ProjectSettings management
- RFE workflow orchestration

### Operator Reconciliation Loop

The Kubernetes operator (`components/operator/`) watches for:

- AgenticSession creation/updates → spawns Jobs with runner pods
- Job completion → updates CR status with results
- Timeout handling and cleanup

### Runner Execution

The Claude Code runner (`components/runners/claude-code-runner/`) provides:

- Claude Code SDK integration (`claude-code-sdk>=0.0.23`)
- Workspace synchronization via PVC proxy
- Multi-agent collaboration capabilities
- Anthropic API streaming (`anthropic>=0.68.0`)

## Configuration Standards

### Python

- **Virtual environments**: Always use `python -m venv venv` or `uv venv`
- **Package manager**: Prefer `uv` over `pip`
- **Formatting**: black (double quotes)
- **Import sorting**: isort with black profile
- **Linting**: flake8 (ignore E203, W503)

### Go

- **Formatting**: `go fmt ./...` (enforced)
- **Linting**: golangci-lint (install via `make install-tools`)
- **Testing**: Table-driven tests with subtests
- **Error handling**: Explicit error returns, no panic in production code

### Container Images

- **Default registry**: `quay.io/ambient_code`
- **Image tags**: Component-specific (vteam_frontend, vteam_backend, vteam_operator, vteam_claude_runner)
- **Platform**: Default `linux/amd64`, ARM64 supported via `PLATFORM=linux/arm64`
- **Build tool**: Docker or Podman (`CONTAINER_ENGINE=podman`)

### Git Workflow

- **Default branch**: `main`
- **Feature branches**: Required for development
- **Commit style**: Conventional commits (squashed on merge)
- **Branch verification**: Always check current branch before file modifications

### Kubernetes/OpenShift

- **Default namespace**: `ambient-code` (production), `vteam-dev` (local dev)
- **CRD group**: `vteam.ambient-code`
- **API version**: `v1alpha1` (current)
- **RBAC**: Namespace-scoped service accounts with minimal permissions

## Backend and Operator Development Standards

**IMPORTANT**: When working on backend (`components/backend/`) or operator (`components/operator/`) code, you MUST follow these strict guidelines based on established patterns in the codebase.

### Critical Rules (Never Violate)

1. **User Token Authentication Required**
   - FORBIDDEN: Using backend service account for user-initiated API operations
   - REQUIRED: Always use `GetK8sClientsForRequest(c)` to get user-scoped K8s clients
   - REQUIRED: Return `401 Unauthorized` if user token is missing or invalid
   - Exception: Backend service account ONLY for CR writes and token minting (handlers/sessions.go:227, handlers/sessions.go:449)

2. **Never Panic in Production Code**
   - FORBIDDEN: `panic()` in handlers, reconcilers, or any production path
   - REQUIRED: Return explicit errors with context: `return fmt.Errorf("failed to X: %w", err)`
   - REQUIRED: Log errors before returning: `log.Printf("Operation failed: %v", err)`

3. **Token Security and Redaction**
   - FORBIDDEN: Logging tokens, API keys, or sensitive headers
   - REQUIRED: Redact tokens in logs using custom formatters (server/server.go:22-34)
   - REQUIRED: Use `log.Printf("tokenLen=%d", len(token))` instead of logging token content
   - Example: `path = strings.Split(path, "?")[0] + "?token=[REDACTED]"`

4. **Type-Safe Unstructured Access**
   - FORBIDDEN: Direct type assertions without checking: `obj.Object["spec"].(map[string]interface{})`
   - REQUIRED: Use `unstructured.Nested*` helpers with three-value returns
   - Example: `spec, found, err := unstructured.NestedMap(obj.Object, "spec")`
   - REQUIRED: Check `found` before using values; handle type mismatches gracefully

5. **OwnerReferences for Resource Lifecycle**
   - REQUIRED: Set OwnerReferences on all child resources (Jobs, Secrets, PVCs, Services)
   - REQUIRED: Use `Controller: boolPtr(true)` for primary owner
   - FORBIDDEN: `BlockOwnerDeletion` (causes permission issues in multi-tenant environments)
   - Pattern: (operator/internal/handlers/sessions.go:125-134, handlers/sessions.go:470-476)

### Package Organization

**Backend Structure** (`components/backend/`):

```
backend/
├── handlers/          # HTTP handlers grouped by resource
│   ├── sessions.go    # AgenticSession CRUD + lifecycle
│   ├── projects.go    # Project management
│   ├── rfe.go         # RFE workflows
│   ├── helpers.go     # Shared utilities (StringPtr, etc.)
│   └── middleware.go  # Auth, validation, RBAC
├── types/             # Type definitions (no business logic)
│   ├── session.go
│   ├── project.go
│   └── common.go
├── server/            # Server setup, CORS, middleware
├── k8s/               # K8s resource templates
├── git/, github/      # External integrations
├── websocket/         # Real-time messaging
├── routes.go          # HTTP route registration
└── main.go            # Wiring, dependency injection
```

**Operator Structure** (`components/operator/`):

```
operator/
├── internal/
│   ├── config/        # K8s client init, config loading
│   ├── types/         # GVR definitions, resource helpers
│   ├── handlers/      # Watch handlers (sessions, namespaces, projectsettings)
│   └── services/      # Reusable services (PVC provisioning, etc.)
└── main.go            # Watch coordination
```

**Rules**:

- Handlers contain HTTP/watch logic ONLY
- Types are pure data structures
- Business logic in separate service packages
- No cyclic dependencies between packages

### Kubernetes Client Patterns

**User-Scoped Clients** (for API operations):

```go
// ALWAYS use for user-initiated operations (list, get, create, update, delete)
reqK8s, reqDyn := GetK8sClientsForRequest(c)
if reqK8s == nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
    c.Abort()
    return
}
// Use reqDyn for CR operations in user's authorized namespaces
list, err := reqDyn.Resource(gvr).Namespace(project).List(ctx, v1.ListOptions{})
```

**Backend Service Account Clients** (limited use cases):

```go
// ONLY use for:
// 1. Writing CRs after validation (handlers/sessions.go:417)
// 2. Minting tokens/secrets for runners (handlers/sessions.go:449)
// 3. Cross-namespace operations backend is authorized for
// Available as: DynamicClient, K8sClient (package-level in handlers/)
created, err := DynamicClient.Resource(gvr).Namespace(project).Create(ctx, obj, v1.CreateOptions{})
```

**Never**:

- ❌ Fall back to service account when user token is invalid
- ❌ Use service account for list/get operations on behalf of users
- ❌ Skip RBAC checks by using elevated permissions

### Error Handling Patterns

**Handler Errors**:

```go
// Pattern 1: Resource not found
if errors.IsNotFound(err) {
    c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
    return
}

// Pattern 2: Log + return error
if err != nil {
    log.Printf("Failed to create session %s in project %s: %v", name, project, err)
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
    return
}

// Pattern 3: Non-fatal errors (continue operation)
if err := updateStatus(...); err != nil {
    log.Printf("Warning: status update failed: %v", err)
    // Continue - session was created successfully
}
```

**Operator Errors**:

```go
// Pattern 1: Resource deleted during processing (non-fatal)
if errors.IsNotFound(err) {
    log.Printf("AgenticSession %s no longer exists, skipping", name)
    return nil  // Don't treat as error
}

// Pattern 2: Retriable errors in watch loop
if err != nil {
    log.Printf("Failed to create job: %v", err)
    updateAgenticSessionStatus(ns, name, map[string]interface{}{
        "phase": "Error",
        "message": fmt.Sprintf("Failed to create job: %v", err),
    })
    return fmt.Errorf("failed to create job: %v", err)
}
```

**Never**:

- ❌ Silent failures (always log errors)
- ❌ Generic error messages ("operation failed")
- ❌ Retrying indefinitely without backoff

### Resource Management

**OwnerReferences Pattern**:

```go
// Always set owner when creating child resources
ownerRef := v1.OwnerReference{
    APIVersion: obj.GetAPIVersion(),  // e.g., "vteam.ambient-code/v1alpha1"
    Kind:       obj.GetKind(),        // e.g., "AgenticSession"
    Name:       obj.GetName(),
    UID:        obj.GetUID(),
    Controller: boolPtr(true),        // Only one controller per resource
    // BlockOwnerDeletion: intentionally omitted (permission issues)
}

// Apply to child resources
job := &batchv1.Job{
    ObjectMeta: v1.ObjectMeta{
        Name: jobName,
        Namespace: namespace,
        OwnerReferences: []v1.OwnerReference{ownerRef},
    },
    // ...
}
```

**Cleanup Patterns**:

```go
// Rely on OwnerReferences for automatic cleanup, but delete explicitly when needed
policy := v1.DeletePropagationBackground
err := K8sClient.BatchV1().Jobs(ns).Delete(ctx, jobName, v1.DeleteOptions{
    PropagationPolicy: &policy,
})
if err != nil && !errors.IsNotFound(err) {
    log.Printf("Failed to delete job: %v", err)
    return err
}
```

### Security Patterns

**Token Handling**:

```go
// Extract token from Authorization header
rawAuth := c.GetHeader("Authorization")
parts := strings.SplitN(rawAuth, " ", 2)
if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
    return
}
token := strings.TrimSpace(parts[1])

// NEVER log the token itself
log.Printf("Processing request with token (len=%d)", len(token))
```

**RBAC Enforcement**:

```go
// Always check permissions before operations
ssar := &authv1.SelfSubjectAccessReview{
    Spec: authv1.SelfSubjectAccessReviewSpec{
        ResourceAttributes: &authv1.ResourceAttributes{
            Group:     "vteam.ambient-code",
            Resource:  "agenticsessions",
            Verb:      "list",
            Namespace: project,
        },
    },
}
res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, v1.CreateOptions{})
if err != nil || !res.Status.Allowed {
    c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
    return
}
```

**Container Security**:

```go
// Always set SecurityContext for Job pods
SecurityContext: &corev1.SecurityContext{
    AllowPrivilegeEscalation: boolPtr(false),
    ReadOnlyRootFilesystem:   boolPtr(false),  // Only if temp files needed
    Capabilities: &corev1.Capabilities{
        Drop: []corev1.Capability{"ALL"},  // Drop all by default
    },
},
```

### API Design Patterns

**Project-Scoped Endpoints**:

```go
// Standard pattern: /api/projects/:projectName/resource
r.GET("/api/projects/:projectName/agentic-sessions", ValidateProjectContext(), ListSessions)
r.POST("/api/projects/:projectName/agentic-sessions", ValidateProjectContext(), CreateSession)
r.GET("/api/projects/:projectName/agentic-sessions/:sessionName", ValidateProjectContext(), GetSession)

// ValidateProjectContext middleware:
// 1. Extracts project from route param
// 2. Validates user has access via RBAC check
// 3. Sets project in context: c.Set("project", projectName)
```

**Middleware Chain**:

```go
// Order matters: Recovery → Logging → CORS → Identity → Validation → Handler
r.Use(gin.Recovery())
r.Use(gin.LoggerWithFormatter(customRedactingFormatter))
r.Use(cors.New(corsConfig))
r.Use(forwardedIdentityMiddleware())  // Extracts X-Forwarded-User, etc.
r.Use(ValidateProjectContext())       // RBAC check
```

**Response Patterns**:

```go
// Success with data
c.JSON(http.StatusOK, gin.H{"items": sessions})

// Success with created resource
c.JSON(http.StatusCreated, gin.H{"message": "Session created", "name": name, "uid": uid})

// Success with no content
c.Status(http.StatusNoContent)

// Errors with structured messages
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
```

### Operator Patterns

**Watch Loop with Reconnection**:

```go
func WatchAgenticSessions() {
    gvr := types.GetAgenticSessionResource()

    for {  // Infinite loop with reconnection
        watcher, err := config.DynamicClient.Resource(gvr).Watch(ctx, v1.ListOptions{})
        if err != nil {
            log.Printf("Failed to create watcher: %v", err)
            time.Sleep(5 * time.Second)  // Backoff before retry
            continue
        }

        log.Println("Watching for events...")

        for event := range watcher.ResultChan() {
            switch event.Type {
            case watch.Added, watch.Modified:
                obj := event.Object.(*unstructured.Unstructured)
                handleEvent(obj)
            case watch.Deleted:
                // Handle cleanup
            }
        }

        log.Println("Watch channel closed, restarting...")
        watcher.Stop()
        time.Sleep(2 * time.Second)
    }
}
```

**Reconciliation Pattern**:

```go
func handleEvent(obj *unstructured.Unstructured) error {
    name := obj.GetName()
    namespace := obj.GetNamespace()

    // 1. Verify resource still exists (avoid race conditions)
    currentObj, err := getDynamicClient().Get(ctx, name, namespace)
    if errors.IsNotFound(err) {
        log.Printf("Resource %s no longer exists, skipping", name)
        return nil  // Not an error
    }

    // 2. Get current phase/status
    status, found, _ := unstructured.NestedMap(currentObj.Object, "status")
    phase := getPhaseOrDefault(status, "Pending")

    // 3. Only reconcile if in expected state
    if phase != "Pending" {
        return nil  // Already processed
    }

    // 4. Create resources idempotently (check existence first)
    if _, err := getResource(name); err == nil {
        log.Printf("Resource %s already exists", name)
        return nil
    }

    // 5. Create and update status
    createResource(...)
    updateStatus(namespace, name, map[string]interface{}{"phase": "Creating"})

    return nil
}
```

**Status Updates** (use UpdateStatus subresource):

```go
func updateAgenticSessionStatus(namespace, name string, updates map[string]interface{}) error {
    gvr := types.GetAgenticSessionResource()

    obj, err := config.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
    if errors.IsNotFound(err) {
        log.Printf("Resource deleted, skipping status update")
        return nil  // Not an error
    }

    if obj.Object["status"] == nil {
        obj.Object["status"] = make(map[string]interface{})
    }

    status := obj.Object["status"].(map[string]interface{})
    for k, v := range updates {
        status[k] = v
    }

    // Use UpdateStatus subresource (requires /status permission)
    _, err = config.DynamicClient.Resource(gvr).Namespace(namespace).UpdateStatus(ctx, obj, v1.UpdateOptions{})
    if errors.IsNotFound(err) {
        return nil  // Resource deleted during update
    }
    return err
}
```

**Goroutine Monitoring**:

```go
// Start background monitoring (operator/internal/handlers/sessions.go:477)
go monitorJob(jobName, sessionName, namespace)

// Monitoring loop checks both K8s Job status AND custom container status
func monitorJob(jobName, sessionName, namespace string) {
    for {
        time.Sleep(5 * time.Second)

        // 1. Check if parent resource still exists (exit if deleted)
        if _, err := getSession(namespace, sessionName); errors.IsNotFound(err) {
            log.Printf("Session deleted, stopping monitoring")
            return
        }

        // 2. Check Job status
        job, err := K8sClient.BatchV1().Jobs(namespace).Get(ctx, jobName, v1.GetOptions{})
        if errors.IsNotFound(err) {
            return
        }

        // 3. Update status based on Job conditions
        if job.Status.Succeeded > 0 {
            updateStatus(namespace, sessionName, map[string]interface{}{
                "phase": "Completed",
                "completionTime": time.Now().Format(time.RFC3339),
            })
            cleanup(namespace, jobName)
            return
        }
    }
}
```

### Pre-Commit Checklist for Backend/Operator

Before committing backend or operator code, verify:

- [ ] **Authentication**: All user-facing endpoints use `GetK8sClientsForRequest(c)`
- [ ] **Authorization**: RBAC checks performed before resource access
- [ ] **Error Handling**: All errors logged with context, appropriate HTTP status codes
- [ ] **Token Security**: No tokens or sensitive data in logs
- [ ] **Type Safety**: Used `unstructured.Nested*` helpers, checked `found` before using values
- [ ] **Resource Cleanup**: OwnerReferences set on all child resources
- [ ] **Status Updates**: Used `UpdateStatus` subresource, handled IsNotFound gracefully
- [ ] **Tests**: Added/updated tests for new functionality
- [ ] **Logging**: Structured logs with relevant context (namespace, resource name, etc.)
- [ ] **Code Quality**: Ran all linting checks locally (see below)

**Run these commands before committing:**

```bash
# Backend
cd components/backend
gofmt -l .                    # Check formatting (should output nothing)
go vet ./...                  # Detect suspicious constructs
golangci-lint run            # Run comprehensive linting

# Operator
cd components/operator
gofmt -l .
go vet ./...
golangci-lint run
```

**Auto-format code:**

```bash
gofmt -w components/backend components/operator
```

**Note**: GitHub Actions will automatically run these checks on your PR. Fix any issues locally before pushing.

### Common Mistakes to Avoid

**Backend**:

- ❌ Using service account client for user operations (always use user token)
- ❌ Not checking if user-scoped client creation succeeded
- ❌ Logging full token values (use `len(token)` instead)
- ❌ Not validating project access in middleware
- ❌ Type assertions without checking: `val := obj["key"].(string)` (use `val, ok := ...`)
- ❌ Not setting OwnerReferences (causes resource leaks)
- ❌ Treating IsNotFound as fatal error during cleanup
- ❌ Exposing internal error details to API responses (use generic messages)

**Operator**:

- ❌ Not reconnecting watch on channel close
- ❌ Processing events without verifying resource still exists
- ❌ Updating status on main object instead of /status subresource
- ❌ Not checking current phase before reconciliation (causes duplicate resources)
- ❌ Creating resources without idempotency checks
- ❌ Goroutine leaks (not exiting monitor when resource deleted)
- ❌ Using `panic()` in watch/reconciliation loops
- ❌ Not setting SecurityContext on Job pods

### Reference Files

Study these files to understand established patterns:

**Backend**:

- `components/backend/handlers/sessions.go` - Complete session lifecycle, user/SA client usage
- `components/backend/handlers/middleware.go` - Auth patterns, token extraction, RBAC
- `components/backend/handlers/helpers.go` - Utility functions (StringPtr, BoolPtr)
- `components/backend/types/common.go` - Type definitions
- `components/backend/server/server.go` - Server setup, middleware chain, token redaction
- `components/backend/routes.go` - HTTP route definitions and registration

**Operator**:

- `components/operator/internal/handlers/sessions.go` - Watch loop, reconciliation, status updates
- `components/operator/internal/config/config.go` - K8s client initialization
- `components/operator/internal/types/resources.go` - GVR definitions
- `components/operator/internal/services/infrastructure.go` - Reusable services

## GitHub Actions CI/CD

### Component Build Pipeline (`.github/workflows/components-build-deploy.yml`)

- **Change detection**: Only builds modified components (frontend, backend, operator, claude-runner)
- **Multi-platform builds**: linux/amd64 and linux/arm64
- **Registry**: Pushes to `quay.io/ambient_code` on main branch
- **PR builds**: Build-only, no push on pull requests

### Automation Workflows

- **amber-issue-handler.yml**: Amber background agent - automated fixes via GitHub issue labels (`amber:auto-fix`, `amber:refactor`, `amber:test-coverage`) or `/amber execute` command
- **amber-dependency-sync.yml**: Daily sync of dependency versions to Amber agent knowledge base
- **claude.yml**: Claude Code integration - responds to `@claude` mentions in issues/PRs
- **claude-code-review.yml**: Automated code reviews on pull requests

### Code Quality Workflows

- **go-lint.yml**: Go code formatting, vetting, and linting (gofmt, go vet, golangci-lint)
- **frontend-lint.yml**: Frontend code quality (ESLint, TypeScript checking, build validation)

### Deployment & Testing Workflows

- **prod-release-deploy.yaml**: Production releases with semver versioning and changelog generation
- **e2e.yml**: End-to-end Cypress testing in kind cluster (see Testing Strategy section)
- **test-local-dev.yml**: Local development environment validation

### Utility Workflows

- **docs.yml**: Deploy MkDocs documentation to GitHub Pages
- **dependabot-auto-merge.yml**: Auto-approve and merge Dependabot dependency updates

## Testing Strategy

### E2E Tests (Cypress + Kind)

**Purpose**: Automated end-to-end testing of the complete vTeam stack in a Kubernetes environment.

**Location**: `e2e/`

**Quick Start**:

```bash
make e2e-test CONTAINER_ENGINE=podman  # Or docker
```

**What Gets Tested**:

- ✅ Full vTeam deployment in kind (Kubernetes in Docker)
- ✅ Frontend UI rendering and navigation
- ✅ Backend API connectivity
- ✅ Project creation workflow (main user journey)
- ✅ Authentication with ServiceAccount tokens
- ✅ Ingress routing
- ✅ All pods deploy and become ready

**What Doesn't Get Tested**:

- ❌ OAuth proxy flow (uses direct token auth for simplicity)
- ❌ Session pod execution (requires Anthropic API key)
- ❌ Multi-user scenarios

**Test Suite** (`e2e/cypress/e2e/vteam.cy.ts`):

1. UI loads with token authentication
2. Navigate to new project page
3. Create a new project
4. List created projects
5. Backend API cluster-info endpoint

**CI Integration**: Tests run automatically on all PRs via GitHub Actions (`.github/workflows/e2e.yml`)

**Key Implementation Details**:

- **Architecture**: Frontend without oauth-proxy, direct token injection via environment variables
- **Authentication**: Test user ServiceAccount with cluster-admin permissions
- **Token Handling**: Frontend deployment includes `OC_TOKEN`, `OC_USER`, `OC_EMAIL` env vars
- **Podman Support**: Auto-detects runtime, uses ports 8080/8443 for rootless Podman
- **Ingress**: Standard nginx-ingress with path-based routing

**Adding New Tests**:

```typescript
it('should test new feature', () => {
  cy.visit('/some-page')
  cy.contains('Expected Content').should('be.visible')
  cy.get('#button').click()
  // Auth header automatically injected via beforeEach interceptor
})
```

**Debugging Tests**:

```bash
cd e2e
source .env.test
CYPRESS_TEST_TOKEN="$TEST_TOKEN" CYPRESS_BASE_URL="http://vteam.local:8080" npm run test:headed
```

**Documentation**: See `e2e/README.md` and `docs/testing/e2e-guide.md` for comprehensive testing guide.

### Backend Tests (Go)

- **Unit tests** (`tests/unit/`): Isolated component logic
- **Contract tests** (`tests/contract/`): API contract validation
- **Integration tests** (`tests/integration/`): End-to-end with real k8s cluster
  - Requires `TEST_NAMESPACE` environment variable
  - Set `CLEANUP_RESOURCES=true` for automatic cleanup
  - Permission tests validate RBAC boundaries

### Frontend Tests (NextJS)

- Jest for component testing (when configured)
- Cypress for e2e testing (see E2E Tests section above)

### Operator Tests (Go)

- Controller reconciliation logic tests
- CRD validation tests

## Documentation Structure

The MkDocs site (`mkdocs.yml`) provides:

- **User Guide**: Getting started, RFE creation, agent framework, configuration
- **Developer Guide**: Setup, architecture, plugin development, API reference, testing
- **Labs**: Hands-on exercises (basic → advanced → production)
  - Basic: First RFE, agent interaction, workflow basics
  - Advanced: Custom agents, workflow modification, integration testing
  - Production: Jira integration, OpenShift deployment, scaling
- **Reference**: Agent personas, API endpoints, configuration schema, glossary

### Director Training Labs

Special lab track for leadership training located in `docs/labs/director-training/`:

- Structured exercises for understanding the vTeam system from a strategic perspective
- Validation reports for tracking completion and understanding

## Production Considerations

### Security

- **API keys**: Store in Kubernetes Secrets, managed via ProjectSettings CR
- **RBAC**: Namespace-scoped isolation prevents cross-project access
- **OAuth integration**: OpenShift OAuth for cluster-based authentication (see `docs/OPENSHIFT_OAUTH.md`)
- **Network policies**: Component isolation and secure communication

### Monitoring

- **Health endpoints**: `/health` on backend API
- **Logs**: Structured logging with OpenShift integration
- **Metrics**: Prometheus-compatible (when configured)
- **Events**: Kubernetes events for operator actions

### Scaling

- **Horizontal Pod Autoscaling**: Configure based on CPU/memory
- **Job concurrency**: Operator manages concurrent session execution
- **Resource limits**: Set appropriate requests/limits per component
- **Multi-tenancy**: Project-based isolation with shared infrastructure

---

## Frontend Development Standards

**See `components/frontend/DESIGN_GUIDELINES.md` for complete frontend development patterns.**

### Critical Rules (Quick Reference)

1. **Zero `any` Types** - Use proper types, `unknown`, or generic constraints
2. **Shadcn UI Components Only** - Use `@/components/ui/*` components, no custom UI from scratch
3. **React Query for ALL Data Operations** - Use hooks from `@/services/queries/*`, no manual `fetch()`
4. **Use `type` over `interface`** - Always prefer `type` for type definitions
5. **Colocate Single-Use Components** - Keep page-specific components with their pages

### Pre-Commit Checklist for Frontend

Before committing frontend code:

- [ ] Zero `any` types (or justified with eslint-disable)
- [ ] All UI uses Shadcn components
- [ ] All data operations use React Query
- [ ] Components under 200 lines
- [ ] Single-use components colocated with their pages
- [ ] All buttons have loading states
- [ ] All lists have empty states
- [ ] All nested pages have breadcrumbs
- [ ] All routes have loading.tsx, error.tsx
- [ ] `npm run build` passes with 0 errors, 0 warnings
- [ ] All types use `type` instead of `interface`

### Reference Files

- `components/frontend/DESIGN_GUIDELINES.md` - Detailed patterns and examples
- `components/frontend/COMPONENT_PATTERNS.md` - Architecture patterns
- `components/frontend/src/components/ui/` - Available Shadcn components
- `components/frontend/src/services/` - API service layer examples
