# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**vTeam** is a Kubernetes-native AI automation platform that orchestrates intelligent agentic sessions through containerized microservices. The platform enables AI-powered automation for analysis, research, development, and content creation tasks via a modern web interface.

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

### Backend Development (Go)

```bash
cd components/backend

# Build
make build

# Run locally
make run

# Run with hot-reload (requires: go install github.com/cosmtrek/air@latest)
make dev

# Testing
make test              # Unit + contract tests
make test-unit         # Unit tests only
make test-contract     # Contract tests only
make test-integration  # Integration tests (requires k8s cluster)
make test-permissions  # RBAC/permission tests
make test-coverage     # Generate coverage report

# Linting
make fmt               # Format code
make vet               # Run go vet
make lint              # golangci-lint (install with make install-tools)

# Dependencies
make deps              # Download dependencies
make deps-update       # Update dependencies
make deps-verify       # Verify dependencies

# Environment check
make check-env         # Verify Go, kubectl, docker installed
```

### Frontend Development (NextJS)

```bash
cd components/frontend

# Install dependencies
npm install

# Development server
npm run dev

# Build
npm run build

# Production server
npm start

# Linting
npm run lint
```

### Operator Development (Go)

```bash
cd components/operator

# Build
go build -o operator .

# Run locally (requires k8s access and CRDs installed)
go run .

# Testing
go test ./... -v
```

### Claude Code Runner (Python)

```bash
cd components/runners/claude-code-runner

# Create virtual environment
python -m venv venv
source venv/bin/activate

# Install dependencies (prefer uv)
uv pip install -e .

# Run locally (for testing)
python -m claude_code_runner
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
- **Formatting**: black (88 char line length, double quotes)
- **Import sorting**: isort with black profile
- **Linting**: flake8 with line length 88, ignore E203,W503

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

## GitHub Actions CI/CD

### Component Build Pipeline (`.github/workflows/components-build-deploy.yml`)
- **Change detection**: Only builds modified components (frontend, backend, operator, claude-runner)
- **Multi-platform builds**: linux/amd64 and linux/arm64
- **Registry**: Pushes to `quay.io/ambient_code` on main branch
- **PR builds**: Build-only, no push on pull requests

### Other Workflows
- **claude.yml**: Claude Code integration
- **test-local-dev.yml**: Local development environment validation
- **dependabot-auto-merge.yml**: Automated dependency updates
- **project-automation.yml**: GitHub project board automation

## Testing Strategy

### Backend Tests (Go)
- **Unit tests** (`tests/unit/`): Isolated component logic
- **Contract tests** (`tests/contract/`): API contract validation
- **Integration tests** (`tests/integration/`): End-to-end with real k8s cluster
  - Requires `TEST_NAMESPACE` environment variable
  - Set `CLEANUP_RESOURCES=true` for automatic cleanup
  - Permission tests validate RBAC boundaries

### Frontend Tests (NextJS)
- Jest for component testing
- Cypress for e2e testing (when configured)

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

**IMPORTANT**: When working on frontend code in `components/frontend/`, you MUST follow these strict guidelines. See `components/frontend/DESIGN_GUIDELINES.md` for complete details.

### Critical Rules (Never Violate)

1. **Zero `any` Types**
   - FORBIDDEN: `data: any`, `Control<any>` without eslint-disable
   - REQUIRED: Use proper types, `unknown`, or generic constraints
   - Exception: Add `// eslint-disable-next-line @typescript-eslint/no-explicit-any` ONLY when truly necessary

2. **Shadcn UI Components Only**
   - FORBIDDEN: Creating custom UI from scratch with raw divs/buttons
   - REQUIRED: Use `@/components/ui/*` Shadcn components as foundation
   - Example: `import { Button } from '@/components/ui/button'`

3. **React Query for ALL Data Operations**
   - FORBIDDEN: Manual `fetch()` calls in components, manual loading states
   - REQUIRED: Use React Query hooks from `@/services/queries/*`
   - Example: `const { data } = useProjects()`, `mutation.mutate(data)`

4. **Use `type` over `interface`**
   - FORBIDDEN: `interface ButtonProps { ... }`
   - REQUIRED: `type ButtonProps = { ... }`

5. **Service Layer Architecture**
   - API functions in `src/services/api/*.ts`
   - React Query hooks in `src/services/queries/*.ts`
   - No direct fetch() in components (except API routes)

### File Organization

**Component Colocation Rule**: Single-use components MUST be colocated with their page. Reusable components go in `src/components/`.

```
✅ CORRECT:
src/app/projects/[name]/sessions/[sessionName]/
├── page.tsx
├── loading.tsx
├── error.tsx
├── components/          # Only used in this page
│   ├── session-header.tsx
│   └── message-list.tsx
└── hooks/               # Only used in this page

src/components/
├── ui/                  # Shadcn components
├── empty-state.tsx     # Reusable across app
└── breadcrumbs.tsx     # Reusable across app

❌ WRONG:
src/components/
├── session-header.tsx  # Should be colocated
└── message-list.tsx    # Should be colocated
```

### Next.js App Router Requirements

Every route MUST have:
- `page.tsx` - Main page component
- `loading.tsx` - Loading UI with Skeleton components
- `error.tsx` - Error boundary with user-friendly message
- `not-found.tsx` - 404 UI (for dynamic routes)

### UX Standards (Always Required)

1. **Button States**: ALL buttons must show loading state during async operations
   ```tsx
   <Button disabled={mutation.isPending}>
     {mutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
     {mutation.isPending ? 'Creating...' : 'Create'}
   </Button>
   ```

2. **Empty States**: ALL lists must have empty states
   ```tsx
   if (!projects?.length) {
     return <EmptyState title="No projects yet" ... />;
   }
   ```

3. **Breadcrumbs**: All nested pages must display breadcrumbs

4. **Loading States**: Use Skeleton components, NOT spinners

### Component Development Rules

- **Size Limit**: Components over 200 lines MUST be broken down
- **Extract Logic**: Repeated logic MUST be extracted into custom hooks
- **TypeScript Strict**: No `any` types, proper type safety throughout

### React Query Patterns

**Query Keys**: Use structured query keys with a key factory
```tsx
const projectKeys = {
  all: ['projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  detail: (name: string) => [...projectKeys.all, 'detail', name] as const,
};
```

**Mutations**: All mutations MUST invalidate relevant queries
```tsx
export function useCreateProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data) => projectsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}
```

### Pre-Commit Checklist for Frontend

Before committing frontend code:
- [ ] Zero `any` types (or justified with eslint-disable)
- [ ] All UI uses Shadcn components
- [ ] All data operations use React Query
- [ ] Components under 200 lines
- [ ] Single-use components colocated
- [ ] All buttons have loading states
- [ ] All lists have empty states
- [ ] All nested pages have breadcrumbs
- [ ] All routes have loading.tsx, error.tsx
- [ ] `npm run build` passes with 0 errors, 0 warnings
- [ ] All types use `type` instead of `interface`

### Common Mistakes to Avoid

1. Using `any` type → Use proper types or `unknown`
2. Creating custom UI components → Use Shadcn components
3. fetch() in components → Use React Query hooks
4. Manual loading states → Use mutation.isPending
5. Missing empty states → Add EmptyState component
6. Large components → Break down into smaller ones
7. No error boundaries → Add error.tsx to routes
8. Using `interface` → Use `type` instead
9. Components in wrong location → Colocate single-use components

### Reference Files

- `components/frontend/DESIGN_GUIDELINES.md` - Detailed patterns and examples
- `components/frontend/COMPONENT_PATTERNS.md` - Architecture patterns
- `components/frontend/CLEANUP_CHECKLIST.md` - Migration status
- `components/frontend/src/components/ui/` - Available Shadcn components
- `components/frontend/src/services/` - API service layer examples
