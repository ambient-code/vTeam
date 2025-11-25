<!--
Sync Impact Report - Constitution Update
Version: 1.0.0
Last Updated: 2025-11-13

Changelog (v1.0.0):
  - RATIFIED: Constitution officially ratified and adopted
  - Version bump: v0.2.0 (DRAFT) → v1.0.0 (RATIFIED)
  - Ratification Date: 2025-11-13
  - All 10 core principles now in force
  - Development standards and governance policies active

Changelog (v0.2.0):
  - Added Development Standards: Naming & Legacy Migration subsection
    * Safe vs. breaking change guidance for vTeam → ACP transition
    * Incremental migration approach (documentation first, then UI, then code)
    * DO NOT update list: API groups, CRDs, container names, K8s resources
    * Safe to update: docs, comments, logs, UI text, new variable names
    * Rationale: Gradual migration improves clarity while preserving backward compatibility

Changelog (v0.1.0):
  - Added Principle X: Commit Discipline & Code Review
    * Line count thresholds by change type (bugfix ≤150, feature ≤300/500, refactor ≤400)
    * Mandatory exceptions for generated code, migrations, dependencies
    * Conventional commit format requirements
    * PR size limits (600 lines) with justification requirements
    * Measurement guidelines (what counts vs excluded)

Changelog (v0.0.1):
  - Added Principle VIII: Context Engineering & Prompt Optimization
  - Added Principle IX: Data Access & Knowledge Augmentation
  - Enhanced Principle IV: E2E testing, coverage standards, CI/CD automation
  - Enhanced Principle VI: /metrics endpoint REQUIRED, simplified key metrics guidance
  - Simplified Principle IX: Consolidated RAG/MCP/RLHF into concise bullets
  - Removed redundant test categories section in Principle IV
  - Consolidated Development Standards: Reference principles instead of duplicating
  - Consolidated Production Requirements: Reference principles, add only unique items
  - Reduced total length by ~30 lines while maintaining clarity

Templates Status:
  ✅ plan-template.md - References constitution check dynamically
  ✅ tasks-template.md - Added Phase 3.9 for commit planning/validation (T036-T040)
  ✅ spec-template.md - No updates needed

Follow-up TODOs:
  - Implement /metrics endpoints in all components
  - Create prompt template library
  - Design RAG pipeline architecture
  - Add commit size validation tooling (pre-commit hook or CI check)
  - Update PR template to include commit discipline checklist
  - Continue vTeam → ACP migration incrementally (docs → UI → code)
-->

# ACP Constitution

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
- **Permission Tests**: RBAC boundary validation
- **E2E Tests**: Critical user journeys MUST have end-to-end tests
- **Red-Green-Refactor**: Tests written → Tests fail → Implementation → Tests pass → Refactor

**Coverage Standards**:

- Maintain high test coverage across all categories
- Critical paths MUST have comprehensive test coverage
- CI/CD pipeline MUST enforce test passing before merge
- Coverage reports generated automatically in CI

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
- **Health Endpoints**: `/health` endpoints for all services (liveness, readiness)
- **Metrics Endpoints**: `/metrics` endpoints REQUIRED for all services (Prometheus format)
- **Status Updates**: Use `UpdateStatus` subresource for CR status changes
- **Event Emission**: Kubernetes events for operator actions
- **Error Context**: Errors must include actionable context for debugging
- **Key Metrics**: Expose latency percentiles (p50/p95/p99), error rates, throughput, and component-specific operational metrics aligned with project goals

**Metrics Standards**:

- Prometheus format on dedicated management port
- Standard labels: service, namespace, version
- Focus on metrics critical to project success (e.g., session execution time for vTeam)

**Rationale**: Production systems fail. Without observability, debugging is impossible and MTTR explodes. Metrics enable proactive monitoring and capacity planning.

### VII. Resource Lifecycle Management

Kubernetes resources MUST have proper lifecycle management:

- **OwnerReferences**: ALWAYS set on child resources (Jobs, Secrets, PVCs, Services)
- **Controller References**: Use `Controller: true` for primary owner
- **No BlockOwnerDeletion**: Causes permission issues in multi-tenant environments
- **Idempotency**: Resource creation MUST check existence first
- **Cleanup**: Rely on OwnerReferences for cascading deletes
- **Goroutine Safety**: Exit monitoring goroutines when parent resource deleted

**Rationale**: Resource leaks waste cluster capacity and cause outages. Proper lifecycle management ensures automatic cleanup and prevents orphaned resources.

### VIII. Context Engineering & Prompt Optimization

vTeam is a context engineering hub - AI output quality depends on input quality:

- **Context Budgets**: Respect token limits (200K for Claude Sonnet 4.5)
- **Context Prioritization**: System context > conversation history > examples
- **Prompt Templates**: Use standardized templates for common operations (RFE analysis, code review)
- **Context Compression**: Summarize long-running sessions to preserve history within budget
- **Agent Personas**: Maintain consistency through well-defined agent roles
- **Pre-Deployment Optimization**: ALL prompts MUST be optimized for clarity and token efficiency before deployment
- **Incremental Loading**: Build context incrementally, avoid reloading static content

**Rationale**: Poor context management causes hallucinations, inconsistent outputs, and wasted API costs. Context engineering is a first-class engineering discipline for AI platforms.

### IX. Data Access & Knowledge Augmentation

Enable agents to access external knowledge and learn from interactions:

- **RAG**: Embed and index repository contents, chunk semantically (512-1024 tokens), use consistent models, apply reranking
- **MCP**: Support MCP servers for structured data access, enforce namespace isolation, handle failures gracefully
- **RLHF**: Capture user ratings (thumbs up/down), store with session metadata, refine prompts from patterns, support A/B testing

**Rationale**: Static prompts have limited effectiveness. Platforms must continuously improve through knowledge retrieval and learning from user feedback.

### X. Commit Discipline & Code Review

Each commit MUST be atomic, reviewable, and independently testable:

**Line Count Thresholds** (excludes generated code, test fixtures, vendor/deps):

- **Bug Fix**: ≤150 lines
  - Single issue resolution
  - Includes test demonstrating the bug
  - Includes fix verification

- **Feature (Small)**: ≤300 lines
  - Single user-facing capability
  - Includes unit + contract tests
  - Updates relevant documentation

- **Feature (Medium)**: ≤500 lines
  - Multi-component feature
  - Requires design justification in commit message
  - MUST be reviewable in 30 minutes

- **Refactoring**: ≤400 lines
  - Behavior-preserving changes only
  - MUST NOT mix with feature/bug changes
  - Existing tests MUST pass unchanged

- **Documentation**: ≤200 lines
  - Pure documentation changes
  - Can be larger for initial docs

- **Test Addition**: ≤250 lines
  - Adding missing test coverage
  - MUST NOT include implementation changes

**Mandatory Exceptions** (requires justification in PR description):

- **Code Generation**: Generated CRD YAML, OpenAPI schemas, protobuf
- **Data Migration**: Database migrations, fixture updates
- **Dependency Updates**: go.mod, package.json, requirements.txt
- **Configuration**: Kubernetes manifests for new components (≤800 lines)

**Commit Requirements**:

- **Atomic**: Single logical change that can be independently reverted
- **Self-Contained**: Each commit MUST pass all tests and linters
- **Conventional Format**: `type(scope): description`
  - Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`, `ci`
  - Scope: component name (backend, frontend, operator, runner)
- **Message Content**: Explain WHY, not WHAT (code shows what)
- **No WIP Commits**: Squash before PR submission

**Review Standards**:

- PR over 600 lines MUST be broken into multiple PRs
- Each commit reviewed independently (enable per-commit review in GitHub)
- Large PRs require design doc or RFC first
- Incremental delivery preferred over "big bang" merges

**Measurement** (what counts toward limits):

- ✅ Source code (`*.go`, `*.ts`, `*.tsx`, `*.py`)
- ✅ Configuration specific to feature (new YAML, JSON)
- ✅ Test code
- ❌ Generated code (CRDs, OpenAPI, mocks)
- ❌ Lock files (`go.sum`, `package-lock.json`)
- ❌ Vendored dependencies
- ❌ Binary files

**Rationale**: Large commits hide bugs, slow reviews, complicate bisecting, and create merge conflicts. Specific thresholds provide objective guidance while exceptions handle legitimate cases. Small, focused commits enable faster feedback, easier debugging (git bisect), and safer reverts.

## Development Standards

### Go Code (Backend & Operator)

**Formatting**:

- Run `gofmt -w .` before committing
- Use `golangci-lint run` for comprehensive linting
- Run `go vet ./...` to detect suspicious constructs

**Error Handling**: See Principle III: Type Safety & Error Handling

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

### Naming & Legacy Migration

**vTeam → ACP Transition**:

Replace usage of "vTeam" with "ACP" (Ambient Code Platform) where it is safe and unobtrusive to do so:

**Safe to Update** (non-breaking changes):

- User-facing documentation and README files
- Code comments and inline documentation
- Log messages and error messages
- UI text and labels
- Variable names in new code

**DO NOT Update** (breaking changes - maintain for backward compatibility):

- Kubernetes API group: `vteam.ambient-code`
- Custom Resource Definitions (CRD kinds)
- Container image names: `vteam_frontend`, `vteam_backend`, etc.
- Kubernetes resource names: deployments, services, routes
- Environment variables referenced in deployment configs
- File paths in scripts that reference namespaces/resources
- Git repository name and URLs

**Incremental Approach**:

- Update documentation first (README, AGENTS.md, docs/)
- Update UI text in new features
- Use ACP naming in new code modules
- Do NOT perform mass renames - update organically during feature work
- Document remaining vTeam references in "Legacy vTeam References" section

**Rationale**: The project rebranded from vTeam to Ambient Code Platform, but technical artifacts retain "vteam" for backward compatibility. Gradual, safe migration improves clarity while avoiding breaking changes for existing deployments.

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

**Security**: Apply Principle II security requirements. Additionally: Scan container images for vulnerabilities before deployment.

**Monitoring**: Implement Principle VI observability requirements in production environment. Additionally: Set up centralized logging and alerting infrastructure.

**Scaling**:

- Configure Horizontal Pod Autoscaling based on CPU/memory
- Set appropriate resource requests and limits
- Plan for job concurrency and queue management
- Design for multi-tenancy with shared infrastructure
- Do not use etcd as a database for unbounded objects like CRs. Use an external database like Postgres.

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

- `/AGENTS.md` for Claude Code development
- Component-specific README files
- MkDocs documentation in `/docs`

**Version**: [CONSTITUTION_VERSION] | **Ratified**: [RATIFICATION_DATE] | **Last Amended**: [LAST_AMENDED_DATE]
<!-- Example: Version: 2.1.1 | Ratified: 2025-06-13 | Last Amended: 2025-07-16 -->
