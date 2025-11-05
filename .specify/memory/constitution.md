<!--
Sync Impact Report - Constitution Update
Version Change: Initial → 0.0.1
Type: DRAFT

Modified Principles: N/A (initial creation)
Added Sections:
  - 7 Core Principles (Kubernetes-Native, Security, Type Safety, TDD, Modularity, Observability, Resource Lifecycle)
  - Development Standards
  - Deployment & Operations
  - Governance

Removed Sections: N/A

Templates Status:
  ✅ plan-template.md - Reviewed, references constitution check (line 48)
  ✅ spec-template.md - Reviewed, no constitution references needed (user-focused)
  ✅ tasks-template.md - Reviewed, TDD principles align with Principle III

Follow-up TODOs: None
-->

# ACP Constitution (DRAFT)

## Core Principles

### I. Kubernetes-Native Architecture

All features MUST be built using Kubernetes primitives and patterns:

- Custom Resource Definitions (CRDs) for domain objects (AgenticSession, ProjectSettings, RFEWorkflow)
- Operators for reconciliation loops and lifecycle management
- Jobs for execution workloads with proper resource limits
- ConfigMaps and Secrets for configuration management
- Services and Routes for network exposure
- RBAC for authorization boundaries

**Rationale**: Kubernetes-native design ensures portability, scalability, and enterprise-grade operational tooling. Violations create operational complexity and reduce platform value.

### II. Security & Multi-Tenancy First

Security and isolation MUST be embedded in every component:

- **Authentication**: All user-facing endpoints MUST use user tokens via `GetK8sClientsForRequest()`
- **Authorization**: RBAC checks MUST be performed before resource access
- **Token Security**: NEVER log tokens, API keys, or sensitive headers; use redaction in logs
- **Multi-Tenancy**: Project-scoped namespaces with strict isolation
- **Principle of Least Privilege**: Service accounts with minimal permissions
- **Container Security**: SecurityContext with `AllowPrivilegeEscalation: false`, drop all capabilities
- **No Fallback**: Backend service account ONLY for CR writes and token minting, never as fallback

**Rationale**: Security breaches and privilege escalation destroy trust. Multi-tenant isolation is non-negotiable for enterprise deployment.

### III. Type Safety & Error Handling (NON-NEGOTIABLE)

Production code MUST follow strict type safety and error handling rules:

- **No Panic**: FORBIDDEN in handlers, reconcilers, or any production path
- **Explicit Errors**: Return `fmt.Errorf("context: %w", err)` with wrapped errors
- **Type-Safe Unstructured**: Use `unstructured.Nested*` helpers, check `found` before using values
- **Frontend Type Safety**: Zero `any` types without eslint-disable justification
- **Structured Errors**: Log errors before returning with relevant context (namespace, resource name)
- **Graceful Degradation**: `IsNotFound` during cleanup is not an error

**Rationale**: Runtime panics crash operator loops and kill services. Type assertions without checks cause nil pointer dereferences. Explicit error handling ensures debuggability and operational stability.

### IV. Test-Driven Development

TDD is MANDATORY for all new functionality:

- **Contract Tests**: Every API endpoint/library interface MUST have contract tests
- **Integration Tests**: Multi-component interactions MUST have integration tests
- **Unit Tests**: Business logic MUST have unit tests
- **Red-Green-Refactor**: Tests written → Tests fail → Implementation → Tests pass → Refactor
- **Test Categories**:
  - Contract: API contracts, interface compliance
  - Integration: Cross-service communication, end-to-end flows
  - Unit: Isolated component logic
  - Permission: RBAC boundary validation

**Rationale**: Tests written after implementation miss edge cases and don't drive design. TDD ensures testability, catches regressions, and documents expected behavior.

### V. Component Modularity

Code MUST be organized into clear, single-responsibility modules:

- **Handlers**: HTTP/watch logic ONLY, no business logic
- **Types**: Pure data structures, no methods or business logic
- **Services**: Reusable business logic, no direct HTTP handling
- **No Cyclic Dependencies**: Package imports must form a DAG
- **Frontend Colocation**: Single-use components colocated with pages, reusable components in `/components`
- **File Size Limit**: Components over 200 lines MUST be broken down

**Rationale**: Modular architecture enables parallel development, simplifies testing, and reduces cognitive load. Cyclic dependencies create maintenance nightmares.

### VI. Observability & Monitoring

All components MUST support operational visibility:

- **Structured Logging**: Use structured logs with context (namespace, resource, operation)
- **Health Endpoints**: `/health` endpoints for all services
- **Status Updates**: Use `UpdateStatus` subresource for CR status changes
- **Event Emission**: Kubernetes events for operator actions
- **Metrics**: Prometheus-compatible metrics (when configured)
- **Error Context**: Errors must include actionable context for debugging

**Rationale**: Production systems fail. Without observability, debugging is impossible and MTTR explodes.

### VII. Resource Lifecycle Management

Kubernetes resources MUST have proper lifecycle management:

- **OwnerReferences**: ALWAYS set on child resources (Jobs, Secrets, PVCs, Services)
- **Controller References**: Use `Controller: true` for primary owner
- **No BlockOwnerDeletion**: Causes permission issues in multi-tenant environments
- **Idempotency**: Resource creation MUST check existence first
- **Cleanup**: Rely on OwnerReferences for cascading deletes
- **Goroutine Safety**: Exit monitoring goroutines when parent resource deleted

**Rationale**: Resource leaks waste cluster capacity and cause outages. Proper lifecycle management ensures automatic cleanup and prevents orphaned resources.

## Development Standards

### Go Code (Backend & Operator)

**Formatting**:
- Run `gofmt -w .` before committing
- Use `golangci-lint run` for comprehensive linting
- Run `go vet ./...` to detect suspicious constructs

**Error Handling**:
- Return wrapped errors: `fmt.Errorf("operation failed: %w", err)`
- Log errors with context before returning
- Use `IsNotFound` checks for graceful cleanup
- Never ignore errors (use `// nolint:errcheck` with justification if truly needed)

**Kubernetes Client Patterns**:
- User operations: `GetK8sClientsForRequest(c)`
- Service account: ONLY for CR writes and token minting
- Status updates: Use `UpdateStatus` subresource
- Watch loops: Reconnect on channel close with backoff

### Frontend Code (NextJS)

**UI Components**:
- Use Shadcn UI components from `@/components/ui/*`
- Use `type` instead of `interface` for type definitions
- All buttons MUST show loading states during async operations
- All lists MUST have empty states

**Data Operations**:
- Use React Query hooks from `@/services/queries/*`
- All mutations MUST invalidate relevant queries
- No direct `fetch()` calls in components

**File Organization**:
- Colocate single-use components with pages
- All routes MUST have `page.tsx`, `loading.tsx`, `error.tsx`
- Components over 200 lines MUST be broken down

### Python Code (Runner)

**Environment**:
- ALWAYS use virtual environments (`python -m venv venv` or `uv venv`)
- Prefer `uv` over `pip` for package management

**Formatting**:
- Use `black` with 88 character line length
- Use `isort` with black profile
- Run linters before committing

## Deployment & Operations

### Pre-Deployment Validation

**Go Components**:
```bash
gofmt -l .
go vet ./...
golangci-lint run
make test
```

**Frontend**:
```bash
npm run lint
npm run build  # Must pass with 0 errors, 0 warnings
```

**Container Security**:
- Set SecurityContext on all Job pods
- Drop all capabilities by default
- Use non-root users where possible

### Production Requirements

**Security**:
- Store API keys in Kubernetes Secrets
- Implement RBAC for namespace-scoped isolation
- Enable network policies for component isolation
- Scan container images for vulnerabilities

**Monitoring**:
- Configure Prometheus metrics collection
- Set up centralized logging (ELK, Loki)
- Implement alerting for pod failures and resource exhaustion
- Deploy comprehensive health endpoints

**Scaling**:
- Configure Horizontal Pod Autoscaling based on CPU/memory
- Set appropriate resource requests and limits
- Plan for job concurrency and queue management
- Design for multi-tenancy with shared infrastructure

## Governance

### Amendment Process

1. **Proposal**: Document proposed change with rationale
2. **Review**: Evaluate impact on existing code and templates
3. **Approval**: Requires project maintainer approval
4. **Migration**: Update all dependent templates and documentation
5. **Versioning**: Increment version according to semantic versioning

### Version Policy

- **MAJOR**: Backward incompatible governance/principle removals or redefinitions
- **MINOR**: New principle/section added or materially expanded guidance
- **PATCH**: Clarifications, wording, typo fixes, non-semantic refinements

### Compliance

- All pull requests MUST verify constitution compliance
- Pre-commit checklists MUST be followed for backend, frontend, and operator code
- Complexity violations MUST be justified in implementation plans
- Constitution supersedes all other practices and guidelines

### Development Guidance

Runtime development guidance is maintained in:
- `/CLAUDE.md` for Claude Code development
- Component-specific README files
- MkDocs documentation in `/docs`

**Version**: 0.0.1 (DRAFT) | **Status**: Draft | **Created**: 2025-11-05
