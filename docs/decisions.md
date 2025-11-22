# Decision Log

Chronological record of significant technical and architectural decisions for the Ambient Code Platform. For formal ADRs, see `docs/adr/`.

**Format:**

- **Date:** When the decision was made
- **Decision:** What was decided
- **Why:** Brief rationale (1-2 sentences)
- **Impact:** What changed as a result
- **Related:** Links to ADRs, PRs, issues

---

## 2024-11-21: User Token Authentication for All API Operations

**Decision:** Backend must use user-provided bearer token for all Kubernetes operations on behalf of users. Service account only for privileged operations (writing CRs after validation, minting tokens).

**Why:** Ensures Kubernetes RBAC is enforced at API boundary, preventing security bypass. Backend should not have elevated permissions for user operations.

**Impact:**

- All handlers now use `GetK8sClientsForRequest(c)` to extract user token
- Return 401 if token is invalid or missing
- K8s audit logs now reflect actual user identity
- Added token redaction in logs to prevent credential leaks

**Related:**

- ADR-0002 (User Token Authentication)
- Security context: `.claude/context/security-standards.md`
- Implementation: `components/backend/handlers/middleware.go`

---

## 2024-11-15: Multi-Repo Support in AgenticSessions

**Decision:** Added support for multiple repositories in a single AgenticSession with `mainRepoIndex` to specify the primary working directory.

**Why:** Users needed to perform cross-repo analysis and make coordinated changes across multiple codebases (e.g., frontend + backend).

**Impact:**

- AgenticSession spec now has `repos` array instead of single `repo`
- Added `mainRepoIndex` field (defaults to 0)
- Per-repo status tracking: `pushed` or `abandoned`
- Clone order matters: mainRepo cloned first to establish working directory

**Related:**

- ADR-0003 (Multi-Repository Support)
- Implementation: `components/backend/types/session.go`
- Runner logic: `components/runners/claude-code-runner/wrapper.py`

**Gotchas:**

- Git operations need absolute paths to handle multiple repos
- Clone order affects workspace initialization
- Need explicit cleanup if clone fails

---

## 2024-11-10: Frontend Migration to React Query

**Decision:** Migrated all frontend data fetching from manual `fetch()` calls to TanStack React Query hooks.

**Why:** React Query provides automatic caching, optimistic updates, and real-time synchronization out of the box. Eliminates boilerplate state management.

**Impact:**

- Created `services/queries/` directory with hooks for each resource
- Removed manual `useState` + `useEffect` data fetching patterns
- Added optimistic updates for create/delete operations
- Reduced API calls by ~60% through intelligent caching

**Related:**

- Frontend context: `.claude/context/frontend-development.md`
- Pattern file: `.claude/patterns/react-query-usage.md`
- Implementation: `components/frontend/src/services/queries/`

---

## 2024-11-05: Adopted Shadcn UI Component Library

**Decision:** Standardized on Shadcn UI for all UI components. Forbidden to create custom components for buttons, inputs, dialogs, etc.

**Why:** Shadcn provides accessible, customizable components built on Radix UI primitives. "Copy-paste" model means we own the code and can customize fully.

**Impact:**

- All existing custom button/input components replaced with Shadcn equivalents
- Added DESIGN_GUIDELINES.md enforcing "Shadcn UI only" rule
- Improved accessibility (WCAG 2.1 AA compliance)
- Consistent design language across the platform

**Related:**

- ADR-0005 (Next.js with Shadcn UI and React Query)
- Frontend guidelines: `components/frontend/DESIGN_GUIDELINES.md`
- Available components: `components/frontend/src/components/ui/`

---

## 2024-10-20: Kubernetes Job-Based Session Execution

**Decision:** Execute AgenticSessions as Kubernetes Jobs instead of long-running Deployments.

**Why:** Jobs provide better lifecycle management for batch workloads. Automatic cleanup on completion, restart policies for failures, and clear success/failure status.

**Impact:**

- Operator creates Job (not Deployment) for each session
- Jobs have OwnerReferences pointing to AgenticSession CR
- Automatic cleanup when session CR is deleted
- Job status mapped to AgenticSession status

**Related:**

- ADR-0001 (Kubernetes-Native Architecture)
- Operator implementation: `components/operator/internal/handlers/sessions.go`

**Gotchas:**

- Jobs cannot be updated once created (must delete and recreate)
- Job pods need proper OwnerReferences for cleanup
- Monitoring requires separate goroutine per job

---

## 2024-10-15: Go for Backend, Python for Runner

**Decision:** Use Go for the backend API server, Python for the Claude Code runner.

**Why:** Go provides excellent Kubernetes client-go integration and performance for the API. Python has first-class Claude Code SDK support and is better for scripting git operations.

**Impact:**

- Backend built with Go + Gin framework
- Runner built with Python + claude-code-sdk
- Two separate container images
- Different build and test tooling for each component

**Related:**

- ADR-0004 (Go Backend with Python Runner)
- Backend: `components/backend/`
- Runner: `components/runners/claude-code-runner/`

---

## 2024-10-01: CRD-Based Architecture

**Decision:** Define AgenticSession, ProjectSettings, and RFEWorkflow as Kubernetes Custom Resources (CRDs).

**Why:** CRDs provide declarative API, automatic RBAC integration, and versioning. Operator pattern allows reconciliation of desired state.

**Impact:**

- Created three CRDs with proper validation schemas
- Operator watches CRs and reconciles state
- Backend translates HTTP API to CR operations
- Users can interact via kubectl or web UI

**Related:**

- ADR-0001 (Kubernetes-Native Architecture)
- CRD definitions: `components/manifests/base/*-crd.yaml`

---

## Template for New Entries

Copy this template when adding new decisions:

```markdown
## YYYY-MM-DD: [Decision Title]

**Decision:** [One sentence: what was decided]

**Why:** [1-2 sentences: rationale]

**Impact:**
- [Change 1]
- [Change 2]
- [Change 3]

**Related:**
- [Link to ADR if exists]
- [Link to implementation]
- [Link to context file]

**Gotchas:** (optional)
- [Gotcha 1]
- [Gotcha 2]
```
