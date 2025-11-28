# Memory System Implementation Plan

**Status:** Ready for Execution
**Created:** 2024-11-21
**Estimated Time:** 30-45 minutes
**Context Required:** None (coldstartable)

## Executive Summary

This plan implements a structured "memory system" for the Ambient Code Platform repository to provide Claude Code with better context loading capabilities. Instead of relying solely on the comprehensive CLAUDE.md file (which is always loaded), this system creates:

1. **Scenario-specific context files** - Loadable on-demand for backend, frontend, and security work
2. **Architectural Decision Records (ADRs)** - Document WHY decisions were made
3. **Repomix usage guide** - How to use the 7 existing repomix views effectively
4. **Decision log** - Lightweight chronological record of major decisions
5. **Code pattern catalog** - Reusable patterns with examples

**Why This Matters:** Claude Code can load targeted context when needed rather than processing everything upfront. This improves response accuracy for specialized tasks while keeping the main CLAUDE.md focused on universal rules.

## Implementation Order

Execute in this order for maximum value:

1. ✅ Context files (`.claude/context/`) - Immediate value for daily development
2. ✅ ADR infrastructure (`docs/adr/`) - Captures architectural knowledge
3. ✅ Repomix guide (`.claude/repomix-guide.md`) - Leverages existing assets
4. ✅ Decision log (`docs/decisions.md`) - Lightweight decision tracking
5. ✅ Pattern catalog (`.claude/patterns/`) - Codifies best practices

---

## Component 1: Context Files

### Overview

Create scenario-specific context files that Claude can reference when working in different areas of the codebase.

### Implementation

**Step 1.1:** Create directory structure

```bash
mkdir -p .claude/context
```

**Step 1.2:** Create backend development context

**File:** `.claude/context/backend-development.md`

```markdown
# Backend Development Context

**When to load:** Working on Go backend API, handlers, or Kubernetes integration

## Quick Reference

- **Language:** Go 1.21+
- **Framework:** Gin (HTTP router)
- **K8s Client:** client-go + dynamic client
- **Primary Files:** `components/backend/handlers/*.go`, `components/backend/types/*.go`

## Critical Rules

### Authentication & Authorization

**ALWAYS use user-scoped clients for API operations:**

\```go
reqK8s, reqDyn := GetK8sClientsForRequest(c)
if reqK8s == nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
    c.Abort()
    return
}
\```

**FORBIDDEN:** Using backend service account (`DynamicClient`, `K8sClient`) for user-initiated operations

**Backend service account ONLY for:**
- Writing CRs after validation (handlers/sessions.go:417)
- Minting tokens/secrets for runners (handlers/sessions.go:449)
- Cross-namespace operations backend is authorized for

### Token Security

**NEVER log tokens:**
```go
// ❌ BAD
log.Printf("Token: %s", token)

// ✅ GOOD
log.Printf("Processing request with token (len=%d)", len(token))
```

**Token redaction in logs:** See `server/server.go:22-34` for custom formatter

### Error Handling

**Pattern for handler errors:**

\```go
// Resource not found
if errors.IsNotFound(err) {
    c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
    return
}

// Generic error
if err != nil {
    log.Printf("Failed to create session %s in project %s: %v", name, project, err)
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
    return
}
\```

### Type-Safe Unstructured Access

**FORBIDDEN:** Direct type assertions
```go
// ❌ BAD - will panic if type is wrong
spec := obj.Object["spec"].(map[string]interface{})
```

**REQUIRED:** Use unstructured helpers
```go
// ✅ GOOD
spec, found, err := unstructured.NestedMap(obj.Object, "spec")
if !found || err != nil {
    return fmt.Errorf("spec not found")
}
```

## Common Tasks

### Adding a New API Endpoint

1. **Define route:** `routes.go` with middleware chain
2. **Create handler:** `handlers/[resource].go`
3. **Validate project context:** Use `ValidateProjectContext()` middleware
4. **Get user clients:** `GetK8sClientsForRequest(c)`
5. **Perform operation:** Use `reqDyn` for K8s resources
6. **Return response:** Structured JSON with appropriate status code

### Adding a New Custom Resource Field

1. **Update CRD:** `components/manifests/base/[resource]-crd.yaml`
2. **Update types:** `components/backend/types/[resource].go`
3. **Update handlers:** Extract/validate new field in handlers
4. **Update operator:** Handle new field in reconciliation
5. **Test:** Create sample CR with new field

## Pre-Commit Checklist

- [ ] All user operations use `GetK8sClientsForRequest`
- [ ] No tokens in logs
- [ ] Errors logged with context
- [ ] Type-safe unstructured access
- [ ] `gofmt -w .` applied
- [ ] `go vet ./...` passes
- [ ] `golangci-lint run` passes

## Key Files

- `handlers/sessions.go` - AgenticSession lifecycle (3906 lines)
- `handlers/middleware.go` - Auth, RBAC validation
- `handlers/helpers.go` - Utility functions (StringPtr, BoolPtr)
- `types/session.go` - Type definitions
- `server/server.go` - Server setup, token redaction

## Recent Issues & Learnings

- **2024-11-15:** Fixed token leak in logs - never log raw tokens
- **2024-11-10:** Multi-repo support added - `mainRepoIndex` specifies working directory
- **2024-10-20:** Added RBAC validation middleware - always check permissions
```

**Step 1.3:** Create frontend development context

**File:** `.claude/context/frontend-development.md`

```markdown
# Frontend Development Context

**When to load:** Working on NextJS application, UI components, or React Query integration

## Quick Reference

- **Framework:** Next.js 14 (App Router)
- **UI Library:** Shadcn UI (built on Radix UI primitives)
- **Styling:** Tailwind CSS
- **Data Fetching:** TanStack React Query
- **Primary Directory:** `components/frontend/src/`

## Critical Rules (Zero Tolerance)

### 1. Zero `any` Types

**FORBIDDEN:**
```typescript
// ❌ BAD
function processData(data: any) { ... }
```

**REQUIRED:**
```typescript
// ✅ GOOD - use proper types
function processData(data: AgenticSession) { ... }

// ✅ GOOD - use unknown if type truly unknown
function processData(data: unknown) {
  if (isAgenticSession(data)) { ... }
}
```

### 2. Shadcn UI Components Only

**FORBIDDEN:** Creating custom UI components from scratch for buttons, inputs, dialogs, etc.

**REQUIRED:** Use `@/components/ui/*` components

```typescript
// ❌ BAD
<button className="px-4 py-2 bg-blue-500">Click</button>

// ✅ GOOD
import { Button } from "@/components/ui/button"
<Button>Click</Button>
```

**Available Shadcn components:** button, card, dialog, form, input, select, table, toast, etc.
**Check:** `components/frontend/src/components/ui/` for full list

### 3. React Query for ALL Data Operations

**FORBIDDEN:** Manual `fetch()` calls in components

**REQUIRED:** Use hooks from `@/services/queries/*`

```typescript
// ❌ BAD
const [sessions, setSessions] = useState([])
useEffect(() => {
  fetch('/api/sessions').then(r => r.json()).then(setSessions)
}, [])

// ✅ GOOD
import { useSessions } from "@/services/queries/sessions"
const { data: sessions, isLoading } = useSessions(projectName)
```

### 4. Use `type` Over `interface`

**REQUIRED:** Always prefer `type` for type definitions

```typescript
// ❌ AVOID
interface User { name: string }

// ✅ PREFERRED
type User = { name: string }
```

### 5. Colocate Single-Use Components

**FORBIDDEN:** Creating components in shared directories if only used once

**REQUIRED:** Keep page-specific components with their pages

```
app/
  projects/
    [projectName]/
      sessions/
        _components/        # Components only used in sessions pages
          session-card.tsx
        page.tsx           # Uses session-card
```

## Common Patterns

### Page Structure

```typescript
// app/projects/[projectName]/sessions/page.tsx
import { useSessions } from "@/services/queries/sessions"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"

export default function SessionsPage({
  params,
}: {
  params: { projectName: string }
}) {
  const { data: sessions, isLoading, error } = useSessions(params.projectName)

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>
  if (!sessions?.length) return <div>No sessions found</div>

  return (
    <div>
      {sessions.map(session => (
        <Card key={session.metadata.name}>
          {/* ... */}
        </Card>
      ))}
    </div>
  )
}
```

### React Query Hook Pattern

```typescript
// services/queries/sessions.ts
import { useQuery, useMutation } from "@tanstack/react-query"
import { sessionApi } from "@/services/api/sessions"

export function useSessions(projectName: string) {
  return useQuery({
    queryKey: ["sessions", projectName],
    queryFn: () => sessionApi.list(projectName),
  })
}

export function useCreateSession(projectName: string) {
  return useMutation({
    mutationFn: (data: CreateSessionRequest) =>
      sessionApi.create(projectName, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sessions", projectName] })
    },
  })
}
```

## Pre-Commit Checklist

- [ ] Zero `any` types (or justified with eslint-disable)
- [ ] All UI uses Shadcn components
- [ ] All data operations use React Query
- [ ] Components under 200 lines
- [ ] Single-use components colocated
- [ ] All buttons have loading states
- [ ] All lists have empty states
- [ ] All nested pages have breadcrumbs
- [ ] `npm run build` passes with 0 errors, 0 warnings
- [ ] All types use `type` instead of `interface`

## Key Files

- `components/frontend/DESIGN_GUIDELINES.md` - Comprehensive patterns
- `components/frontend/COMPONENT_PATTERNS.md` - Architecture patterns
- `src/components/ui/` - Shadcn UI components
- `src/services/queries/` - React Query hooks
- `src/services/api/` - API client layer

## Recent Issues & Learnings

- **2024-11-18:** Migrated all data fetching to React Query - no more manual fetch calls
- **2024-11-15:** Enforced Shadcn UI only - removed custom button components
- **2024-11-10:** Added breadcrumb pattern for nested pages
```

**Step 1.4:** Create security standards context

**File:** `.claude/context/security-standards.md`

```markdown
# Security Standards Quick Reference

**When to load:** Working on authentication, authorization, RBAC, or handling sensitive data

## Critical Security Rules

### Token Handling

**1. User Token Authentication Required**

```go
// ALWAYS for user-initiated operations
reqK8s, reqDyn := GetK8sClientsForRequest(c)
if reqK8s == nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
    c.Abort()
    return
}
```

**2. Token Redaction in Logs**

**FORBIDDEN:**
```go
log.Printf("Authorization: Bearer %s", token)
log.Printf("Request headers: %v", headers)
```

**REQUIRED:**
```go
log.Printf("Token length: %d", len(token))
// Redact in URL paths
path = strings.Split(path, "?")[0] + "?token=[REDACTED]"
```

**Token Redaction Pattern:** See `server/server.go:22-34`

```go
// Custom log formatter that redacts tokens
func customRedactingFormatter(param gin.LogFormatterParams) string {
    path := param.Path
    if strings.Contains(path, "token=") {
        path = strings.Split(path, "?")[0] + "?token=[REDACTED]"
    }
    // ... rest of formatting
}
```

### RBAC Enforcement

**1. Always Check Permissions Before Operations**

```go
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

**2. Namespace Isolation**

- Each project maps to a Kubernetes namespace
- User token must have permissions in that namespace
- Never bypass namespace checks

### Container Security

**Always Set SecurityContext for Job Pods**

```go
SecurityContext: &corev1.SecurityContext{
    AllowPrivilegeEscalation: boolPtr(false),
    ReadOnlyRootFilesystem:   boolPtr(false),  // Only if temp files needed
    Capabilities: &corev1.Capabilities{
        Drop: []corev1.Capability{"ALL"},
    },
},
```

### Input Validation

**1. Validate All User Input**

```go
// Validate resource names (K8s DNS label requirements)
if !isValidK8sName(name) {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid name format"})
    return
}

// Validate URLs for repository inputs
if _, err := url.Parse(repoURL); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository URL"})
    return
}
```

**2. Sanitize for Log Injection**

```go
// Prevent log injection with newlines
name = strings.ReplaceAll(name, "\n", "")
name = strings.ReplaceAll(name, "\r", "")
```

## Common Security Patterns

### Pattern 1: Extracting Bearer Token

```go
rawAuth := c.GetHeader("Authorization")
parts := strings.SplitN(rawAuth, " ", 2)
if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
    return
}
token := strings.TrimSpace(parts[1])
// NEVER log token itself
log.Printf("Processing request with token (len=%d)", len(token))
```

### Pattern 2: Validating Project Access

```go
func ValidateProjectContext() gin.HandlerFunc {
    return func(c *gin.Context) {
        projectName := c.Param("projectName")

        // Get user-scoped K8s client
        reqK8s, _ := GetK8sClientsForRequest(c)
        if reqK8s == nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }

        // Check if user can access namespace
        ssar := &authv1.SelfSubjectAccessReview{
            Spec: authv1.SelfSubjectAccessReviewSpec{
                ResourceAttributes: &authv1.ResourceAttributes{
                    Resource:  "namespaces",
                    Verb:      "get",
                    Name:      projectName,
                },
            },
        }
        res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, v1.CreateOptions{})
        if err != nil || !res.Status.Allowed {
            c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to project"})
            c.Abort()
            return
        }

        c.Set("project", projectName)
        c.Next()
    }
}
```

### Pattern 3: Minting Service Account Tokens

```go
// Only backend service account can create tokens for runner pods
tokenRequest := &authv1.TokenRequest{
    Spec: authv1.TokenRequestSpec{
        ExpirationSeconds: int64Ptr(3600),
    },
}

tokenResponse, err := K8sClient.CoreV1().ServiceAccounts(namespace).CreateToken(
    ctx,
    serviceAccountName,
    tokenRequest,
    v1.CreateOptions{},
)
if err != nil {
    return fmt.Errorf("failed to create token: %w", err)
}

// Store token in secret (never log it)
secret := &corev1.Secret{
    ObjectMeta: v1.ObjectMeta{
        Name:      fmt.Sprintf("%s-token", sessionName),
        Namespace: namespace,
    },
    StringData: map[string]string{
        "token": tokenResponse.Status.Token,
    },
}
```

## Security Checklist

Before committing code that handles:

**Authentication:**
- [ ] Using user token (GetK8sClientsForRequest) for user operations
- [ ] Returning 401 if token is invalid/missing
- [ ] Not falling back to service account on auth failure

**Authorization:**
- [ ] RBAC check performed before resource access
- [ ] Using correct namespace for permission check
- [ ] Returning 403 if user lacks permissions

**Secrets & Tokens:**
- [ ] No tokens in logs (use len(token) instead)
- [ ] No tokens in error messages
- [ ] Tokens stored in Kubernetes Secrets
- [ ] Token redaction in request logs

**Input Validation:**
- [ ] All user input validated
- [ ] Resource names validated (K8s DNS label format)
- [ ] URLs parsed and validated
- [ ] Log injection prevented

**Container Security:**
- [ ] SecurityContext set on all Job pods
- [ ] AllowPrivilegeEscalation: false
- [ ] Capabilities dropped (ALL)
- [ ] OwnerReferences set for cleanup

## Recent Security Issues

- **2024-11-15:** Fixed token leak in logs - added custom redacting formatter
- **2024-10-20:** Added RBAC validation middleware - prevent unauthorized access
- **2024-10-10:** Fixed privilege escalation risk - added SecurityContext to Job pods

## Security Review Resources

- OWASP Top 10: https://owasp.org/www-project-top-ten/
- Kubernetes Security Best Practices: https://kubernetes.io/docs/concepts/security/
- RBAC Documentation: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
```

### Success Criteria

- [ ] `.claude/context/` directory created
- [ ] Three context files created (backend, frontend, security)
- [ ] Each file contains actionable, copy-paste ready examples
- [ ] Files reference specific line numbers in codebase where patterns are implemented

---

## Component 2: ADR Infrastructure

### Overview

Architectural Decision Records (ADRs) document WHY decisions were made, not just WHAT was implemented. This is invaluable for understanding when to deviate from patterns vs. follow them strictly.

### Implementation

**Step 2.1:** Create directory structure

```bash
mkdir -p docs/adr
```

**Step 2.2:** Create ADR template

**File:** `docs/adr/template.md`

```markdown
# ADR-NNNN: [Short Title of Decision]

**Status:** [Proposed | Accepted | Deprecated | Superseded by ADR-XXXX]
**Date:** YYYY-MM-DD
**Deciders:** [List of people involved]
**Technical Story:** [Link to issue/PR if applicable]

## Context and Problem Statement

[Describe the context and problem. What forces are at play? What constraints exist? What problem are we trying to solve?]

## Decision Drivers

* [Driver 1 - e.g., Performance requirements]
* [Driver 2 - e.g., Security constraints]
* [Driver 3 - e.g., Team expertise]
* [Driver 4 - e.g., Cost considerations]

## Considered Options

* [Option 1]
* [Option 2]
* [Option 3]

## Decision Outcome

Chosen option: "[Option X]", because [justification. Why this option over others? What were the decisive factors?]

### Consequences

**Positive:**

* [Positive consequence 1 - e.g., Improved performance]
* [Positive consequence 2 - e.g., Better security]

**Negative:**

* [Negative consequence 1 - e.g., Increased complexity]
* [Negative consequence 2 - e.g., Higher learning curve]

**Risks:**

* [Risk 1 - e.g., Third-party dependency risk]
* [Risk 2 - e.g., Scaling limitations]

## Implementation Notes

[How this was actually implemented. Gotchas discovered during implementation. Deviations from original plan.]

**Key Files:**
* [file.go:123] - [What this implements]
* [component.tsx:456] - [What this implements]

**Patterns Established:**
* [Pattern 1]
* [Pattern 2]

## Validation

How do we know this decision was correct?

* [Metric 1 - e.g., Response time improved by 40%]
* [Metric 2 - e.g., Security audit passed]
* [Outcome 1 - e.g., Team velocity increased]

## Links

* [Related ADR-XXXX]
* [Related issue #XXX]
* [Supersedes ADR-YYYY]
* [External reference]
```

**Step 2.3:** Create README for ADR index

**File:** `docs/adr/README.md`

```markdown
# Architectural Decision Records (ADRs)

This directory contains Architectural Decision Records (ADRs) documenting significant architectural decisions made for the Ambient Code Platform.

## What is an ADR?

An ADR captures:
- **Context:** What problem were we solving?
- **Options:** What alternatives did we consider?
- **Decision:** What did we choose and why?
- **Consequences:** What are the trade-offs?

ADRs are immutable once accepted. If a decision changes, we create a new ADR that supersedes the old one.

## When to Create an ADR

Create an ADR for decisions that:
- Affect the overall architecture
- Are difficult or expensive to reverse
- Impact multiple components or teams
- Involve significant trade-offs
- Will be questioned in the future ("Why did we do it this way?")

**Examples:**
- Choosing a programming language or framework
- Selecting a database or messaging system
- Defining authentication/authorization approach
- Establishing API design patterns
- Multi-tenancy architecture decisions

**Not ADR-worthy:**
- Trivial implementation choices
- Decisions easily reversed
- Component-internal decisions with no external impact

## ADR Workflow

1. **Propose:** Copy `template.md` to `NNNN-title.md` with status "Proposed"
2. **Discuss:** Share with team, gather feedback
3. **Decide:** Update status to "Accepted" or "Rejected"
4. **Implement:** Reference ADR in PRs
5. **Learn:** Update "Implementation Notes" with gotchas discovered

## ADR Status Meanings

- **Proposed:** Decision being considered, open for discussion
- **Accepted:** Decision made and being implemented
- **Deprecated:** Decision no longer relevant but kept for historical context
- **Superseded by ADR-XXXX:** Decision replaced by a newer ADR

## Current ADRs

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0001](0001-kubernetes-native-architecture.md) | Kubernetes-Native Architecture | Accepted | 2024-11-21 |
| [0002](0002-user-token-authentication.md) | User Token Authentication for API Operations | Accepted | 2024-11-21 |
| [0003](0003-multi-repo-support.md) | Multi-Repository Support in AgenticSessions | Accepted | 2024-11-21 |
| [0004](0004-go-backend-python-runner.md) | Go Backend with Python Claude Runner | Accepted | 2024-11-21 |
| [0005](0005-nextjs-shadcn-react-query.md) | Next.js with Shadcn UI and React Query | Accepted | 2024-11-21 |

## References

- [ADR GitHub Organization](https://adr.github.io/) - ADR best practices
- [Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions) - Original proposal by Michael Nygard
```

**Step 2.4:** Create 5 critical ADRs

**File:** `docs/adr/0001-kubernetes-native-architecture.md`

```markdown
# ADR-0001: Kubernetes-Native Architecture

**Status:** Accepted
**Date:** 2024-11-21
**Deciders:** Platform Architecture Team
**Technical Story:** Initial platform architecture design

## Context and Problem Statement

We needed to build an AI automation platform that could:
- Execute long-running AI agent sessions
- Isolate execution environments for security
- Scale based on demand
- Integrate with existing OpenShift/Kubernetes infrastructure
- Support multi-tenancy

How should we architect the platform to meet these requirements?

## Decision Drivers

* **Multi-tenancy requirement:** Need strong isolation between projects
* **Enterprise context:** Red Hat runs on OpenShift/Kubernetes
* **Resource management:** AI sessions have varying resource needs
* **Security:** Must prevent cross-project access and resource interference
* **Scalability:** Need to handle variable workload
* **Operational excellence:** Leverage existing K8s operational expertise

## Considered Options

1. **Kubernetes-native with CRDs and Operators**
2. **Traditional microservices on VMs**
3. **Serverless functions (e.g., AWS Lambda, OpenShift Serverless)**
4. **Container orchestration with Docker Swarm**

## Decision Outcome

Chosen option: "Kubernetes-native with CRDs and Operators", because:

1. **Natural multi-tenancy:** K8s namespaces provide isolation
2. **Declarative resources:** CRDs allow users to declare desired state
3. **Built-in scaling:** K8s handles pod scheduling and resource allocation
4. **Enterprise alignment:** Matches Red Hat's OpenShift expertise
5. **Operational maturity:** Established patterns for monitoring, logging, RBAC

### Consequences

**Positive:**

* Strong multi-tenant isolation via namespaces
* Declarative API via Custom Resources (AgenticSession, ProjectSettings, RFEWorkflow)
* Automatic cleanup via OwnerReferences
* RBAC integration for authorization
* Native integration with OpenShift OAuth
* Horizontal scaling of operator and backend components
* Established operational patterns (logs, metrics, events)

**Negative:**

* Higher learning curve for developers unfamiliar with K8s
* Requires K8s cluster for all deployments (including local dev)
* Operator complexity vs. simpler stateless services
* CRD versioning and migration challenges
* Resource overhead of K8s control plane

**Risks:**

* CRD API changes require careful migration planning
* Operator bugs can affect many sessions simultaneously
* K8s version skew between dev/prod environments

## Implementation Notes

**Architecture Components:**

1. **Custom Resources (CRDs):**
   - AgenticSession: Represents AI execution session
   - ProjectSettings: Project-scoped configuration
   - RFEWorkflow: Multi-agent refinement workflows

2. **Operator Pattern:**
   - Watches CRs and reconciles desired state
   - Creates Kubernetes Jobs for session execution
   - Updates CR status with results

3. **Job-Based Execution:**
   - Each AgenticSession spawns a Kubernetes Job
   - Job runs Claude Code runner pod
   - Results stored in CR status, PVCs for workspace

4. **Multi-Tenancy:**
   - Each project = one K8s namespace
   - RBAC enforces access control
   - Backend validates user tokens before CR operations

**Key Files:**
* `components/manifests/base/*-crd.yaml` - CRD definitions
* `components/operator/internal/handlers/sessions.go` - Operator reconciliation
* `components/backend/handlers/sessions.go` - API to CR translation

## Validation

**Success Metrics:**

* ✅ Multi-tenant isolation validated via RBAC tests
* ✅ Sessions scale from 1 to 50+ concurrent executions
* ✅ Zero cross-project access violations in testing
* ✅ Operator handles CRD updates without downtime

**Lessons Learned:**

* OwnerReferences critical for automatic cleanup
* Status subresource prevents race conditions in updates
* Job monitoring requires separate goroutine per session
* Local dev requires kind/CRC for K8s environment

## Links

* [Kubernetes Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
* [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
* Related: ADR-0002 (User Token Authentication)
```

**File:** `docs/adr/0002-user-token-authentication.md`

```markdown
# ADR-0002: User Token Authentication for API Operations

**Status:** Accepted
**Date:** 2024-11-21
**Deciders:** Security Team, Platform Team
**Technical Story:** Security audit revealed RBAC bypass via service account

## Context and Problem Statement

The backend API needs to perform Kubernetes operations (list sessions, create CRs, etc.) on behalf of users. How should we authenticate and authorize these operations?

**Initial implementation:** Backend used its own service account for all operations, checking user identity separately.

**Problem discovered:** This bypassed Kubernetes RBAC, creating a security risk where backend could access resources the user couldn't.

## Decision Drivers

* **Security requirement:** Enforce Kubernetes RBAC at API boundary
* **Multi-tenancy:** Users should only access their authorized namespaces
* **Audit trail:** K8s audit logs should reflect actual user actions
* **Least privilege:** Backend should not have elevated permissions for user operations
* **Trust boundary:** Backend is the entry point, must validate properly

## Considered Options

1. **User token for all operations (user-scoped K8s client)**
2. **Backend service account with custom RBAC layer**
3. **Impersonation (backend impersonates user identity)**
4. **Hybrid: User token for reads, service account for writes**

## Decision Outcome

Chosen option: "User token for all operations", because:

1. **Leverages K8s RBAC:** No need to duplicate authorization logic
2. **Security principle:** User operations use user permissions
3. **Audit trail:** K8s logs show actual user, not service account
4. **Least privilege:** Backend only uses service account when necessary
5. **Simplicity:** One pattern for user operations, exceptions documented

**Exception:** Backend service account ONLY for:
- Writing CRs after user authorization validated (handlers/sessions.go:417)
- Minting service account tokens for runner pods (handlers/sessions.go:449)
- Cross-namespace operations backend is explicitly authorized for

### Consequences

**Positive:**

* Kubernetes RBAC enforced automatically
* No custom authorization layer to maintain
* Audit logs reflect actual user identity
* RBAC violations fail at K8s API, not at backend
* Easy to debug permission issues (use `kubectl auth can-i`)

**Negative:**

* Must extract and validate user token on every request
* Token expiration can cause mid-request failures
* Slightly higher latency (extra K8s API call for RBAC check)
* Backend needs pattern to fall back to service account for specific operations

**Risks:**

* Token handling bugs could expose security vulnerabilities
* Token logging could leak credentials
* Service account fallback could be misused

## Implementation Notes

**Pattern 1: Extract User Token from Request**

```go
func GetK8sClientsForRequest(c *gin.Context) (*kubernetes.Clientset, dynamic.Interface) {
    rawAuth := c.GetHeader("Authorization")
    parts := strings.SplitN(rawAuth, " ", 2)
    if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
        return nil, nil
    }
    token := strings.TrimSpace(parts[1])

    config := &rest.Config{
        Host:        K8sConfig.Host,
        BearerToken: token,
        TLSClientConfig: rest.TLSClientConfig{
            CAData: K8sConfig.CAData,
        },
    }

    k8sClient, _ := kubernetes.NewForConfig(config)
    dynClient, _ := dynamic.NewForConfig(config)
    return k8sClient, dynClient
}
```

**Pattern 2: Use User-Scoped Client in Handlers**

```go
func ListSessions(c *gin.Context) {
    project := c.Param("projectName")

    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    if reqK8s == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
        c.Abort()
        return
    }

    // Use reqDyn for operations - RBAC enforced by K8s
    list, err := reqDyn.Resource(gvr).Namespace(project).List(ctx, v1.ListOptions{})
    // ...
}
```

**Pattern 3: Service Account for Privileged Operations**

```go
func CreateSession(c *gin.Context) {
    // 1. Validate user has permission (using user token)
    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    if reqK8s == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    // 2. Validate request body
    var req CreateSessionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    // 3. Check user can create in this namespace
    ssar := &authv1.SelfSubjectAccessReview{...}
    res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, v1.CreateOptions{})
    if err != nil || !res.Status.Allowed {
        c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
        return
    }

    // 4. NOW use service account to write CR (after validation)
    obj := &unstructured.Unstructured{...}
    created, err := DynamicClient.Resource(gvr).Namespace(project).Create(ctx, obj, v1.CreateOptions{})
    // ...
}
```

**Security Measures:**

* Token redaction in logs (server/server.go:22-34)
* Never log token values, only length: `log.Printf("tokenLen=%d", len(token))`
* Token extraction in dedicated function for consistency
* Return 401 immediately if token invalid

**Key Files:**
* `handlers/middleware.go:GetK8sClientsForRequest()` - Token extraction
* `handlers/sessions.go:227` - User validation then SA create pattern
* `server/server.go:22-34` - Token redaction formatter

## Validation

**Security Testing:**

* ✅ User cannot list sessions in unauthorized namespaces
* ✅ User cannot create sessions without RBAC permissions
* ✅ K8s audit logs show user identity, not service account
* ✅ Token expiration properly handled with 401 response
* ✅ No tokens found in application logs

**Performance Impact:**

* Negligible (<5ms) latency increase for RBAC validation
* No additional K8s API calls (RBAC check happens in K8s)

## Links

* Related: ADR-0001 (Kubernetes-Native Architecture)
* [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
* [Token Review API](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-review-v1/)
```

**File:** `docs/adr/0003-multi-repo-support.md`

```markdown
# ADR-0003: Multi-Repository Support in AgenticSessions

**Status:** Accepted
**Date:** 2024-11-21
**Deciders:** Product Team, Engineering Team
**Technical Story:** User request for cross-repo analysis and modification

## Context and Problem Statement

Users needed to execute AI sessions that operate across multiple Git repositories simultaneously. For example:
- Analyze dependencies between frontend and backend repos
- Make coordinated changes across microservices
- Generate documentation that references multiple codebases

Original design: AgenticSession operated on a single repository.

How should we extend AgenticSessions to support multiple repositories while maintaining simplicity and clear semantics?

## Decision Drivers

* **User need:** Cross-repo analysis and modification workflows
* **Clarity:** Need clear semantics for which repo is "primary"
* **Workspace model:** Claude Code expects a single working directory
* **Git operations:** Push/PR creation needs per-repo configuration
* **Status tracking:** Need to track per-repo outcomes (pushed vs. abandoned)
* **Backward compatibility:** Don't break single-repo workflows

## Considered Options

1. **Multiple repos with mainRepoIndex (chosen)**
2. **Separate sessions per repo with orchestration layer**
3. **Multi-root workspace (multiple working directories)**
4. **Merge all repos into monorepo temporarily**

## Decision Outcome

Chosen option: "Multiple repos with mainRepoIndex", because:

1. **Claude Code compatibility:** Single working directory aligns with claude-code CLI
2. **Clear semantics:** mainRepoIndex explicitly specifies "primary" repo
3. **Flexibility:** Can reference other repos via relative paths
4. **Status tracking:** Per-repo pushed/abandoned status in CR
5. **Backward compatible:** Single-repo sessions just have one entry in repos array

### Consequences

**Positive:**

* Enables cross-repo workflows (analysis, coordinated changes)
* Per-repo push status provides clear outcome tracking
* mainRepoIndex makes "primary repository" explicit
* Backward compatible with single-repo sessions
* Supports different git configs per repo (fork vs. direct push)

**Negative:**

* Increased complexity in session CR structure
* Clone order matters (mainRepo must be cloned first to establish working directory)
* File paths between repos can be confusing for users
* Workspace cleanup more complex with multiple repos

**Risks:**

* Users might not understand which repo is "main"
* Large number of repos could cause workspace size issues
* Git credentials management across repos more complex

## Implementation Notes

**AgenticSession Spec Structure:**

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: multi-repo-session
spec:
  prompt: "Analyze API compatibility between frontend and backend"

  # repos is an array of repository configurations
  repos:
    - input:
        url: "https://github.com/org/frontend"
        branch: "main"
      output:
        type: "fork"
        targetBranch: "feature-update"
        createPullRequest: true

    - input:
        url: "https://github.com/org/backend"
        branch: "main"
      output:
        type: "direct"
        pushBranch: "feature-update"

  # mainRepoIndex specifies which repo is the working directory (0-indexed)
  mainRepoIndex: 0  # frontend is the main repo

  interactive: false
  timeout: 3600
```

**Status Structure:**

```yaml
status:
  phase: "Completed"
  startTime: "2024-11-21T10:00:00Z"
  completionTime: "2024-11-21T10:30:00Z"

  # Per-repo status tracking
  repoStatuses:
    - repoURL: "https://github.com/org/frontend"
      status: "pushed"
      message: "PR #123 created"

    - repoURL: "https://github.com/org/backend"
      status: "abandoned"
      message: "No changes made"
```

**Clone Implementation Pattern:**

```python
# components/runners/claude-code-runner/wrapper.py

def clone_repositories(repos, main_repo_index, workspace):
    """Clone repos in correct order: mainRepo first, others after."""

    # Clone main repo first to establish working directory
    main_repo = repos[main_repo_index]
    main_path = clone_repo(main_repo["input"]["url"], workspace)
    os.chdir(main_path)  # Set as working directory

    # Clone other repos relative to workspace
    for i, repo in enumerate(repos):
        if i == main_repo_index:
            continue
        clone_repo(repo["input"]["url"], workspace)

    return main_path
```

**Key Files:**
* `components/backend/types/session.go:RepoConfig` - Repo configuration types
* `components/backend/handlers/sessions.go:227` - Multi-repo validation
* `components/runners/claude-code-runner/wrapper.py:clone_repositories` - Clone logic
* `components/operator/internal/handlers/sessions.go:150` - Status tracking

**Patterns Established:**

* mainRepoIndex defaults to 0 if not specified
* repos array must have at least one entry
* Per-repo output configuration (fork vs. direct push)
* Per-repo status tracking (pushed, abandoned, error)

## Validation

**Testing Scenarios:**

* ✅ Single-repo session (backward compatibility)
* ✅ Two-repo session with mainRepoIndex=0
* ✅ Two-repo session with mainRepoIndex=1
* ✅ Cross-repo file analysis
* ✅ Per-repo push status correctly reported
* ✅ Clone failure in secondary repo doesn't block main repo

**User Feedback:**

* Positive: Enables new workflow patterns (monorepo analysis)
* Confusion: Initially unclear which repo is "main"
* Resolution: Added documentation and examples

## Links

* Related: ADR-0001 (Kubernetes-Native Architecture)
* Implementation PR: #XXX
* User documentation: `docs/user-guide/multi-repo-sessions.md`
```

**File:** `docs/adr/0004-go-backend-python-runner.md`

```markdown
# ADR-0004: Go Backend with Python Claude Runner

**Status:** Accepted
**Date:** 2024-11-21
**Deciders:** Architecture Team
**Technical Story:** Technology stack selection for platform components

## Context and Problem Statement

We need to choose programming languages for two distinct components:

1. **Backend API:** HTTP server managing Kubernetes resources, authentication, project management
2. **Claude Code Runner:** Executes claude-code CLI in Job pods

What languages should we use for each component, and should they be the same or different?

## Decision Drivers

* **Backend needs:** HTTP routing, K8s client-go, RBAC, high concurrency
* **Runner needs:** Claude Code SDK, file manipulation, git operations
* **Performance:** Backend handles many concurrent requests
* **Developer experience:** Team expertise, library ecosystems
* **Operational:** Container size, startup time, resource usage
* **Maintainability:** Type safety, tooling, debugging

## Considered Options

1. **Go backend + Python runner (chosen)**
2. **All Python (FastAPI backend + Python runner)**
3. **All Go (Go backend + Go wrapper for claude-code)**
4. **Polyglot (Node.js backend + Python runner)**

## Decision Outcome

Chosen option: "Go backend + Python runner", because:

**Go for Backend:**
1. **K8s ecosystem:** client-go is canonical K8s library
2. **Performance:** Low latency HTTP handling, efficient concurrency
3. **Type safety:** Compile-time checks for K8s resources
4. **Deployment:** Single static binary, fast startup
5. **Team expertise:** Red Hat strong Go background

**Python for Runner:**
1. **Claude Code SDK:** Official SDK is Python-first (`claude-code-sdk`)
2. **Anthropic ecosystem:** Python has best library support
3. **Scripting flexibility:** Git operations, file manipulation easier in Python
4. **Dynamic execution:** Easier to handle varying prompts and workflows

### Consequences

**Positive:**

* **Backend:**
  - Fast HTTP response times (<10ms for simple operations)
  - Small container images (~20MB for Go binary)
  - Excellent K8s client-go integration
  - Strong typing prevents many bugs

* **Runner:**
  - Native Claude Code SDK support
  - Rich Python ecosystem for git/file operations
  - Easy to extend with custom agent behaviors
  - Rapid iteration on workflow logic

**Negative:**

* **Maintenance:**
  - Two language ecosystems to maintain
  - Different tooling (go vs. pip/uv)
  - Different testing frameworks

* **Development:**
  - Context switching between languages
  - Cannot share code between backend and runner
  - Different error handling patterns

**Risks:**

* Python runner startup slower than Go (~1-2s vs. <100ms)
* Python container images larger (~500MB vs. ~20MB)
* Dependency vulnerabilities in Python ecosystem

## Implementation Notes

**Backend (Go):**

```go
// Fast HTTP routing with Gin
r := gin.Default()
r.GET("/api/projects/:project/sessions", handlers.ListSessions)

// Type-safe K8s client
clientset, _ := kubernetes.NewForConfig(config)
sessions, err := clientset.CoreV1().Pods(namespace).List(ctx, v1.ListOptions{})
```

**Technology Stack:**
- Framework: Gin (HTTP routing)
- K8s client: client-go + dynamic client
- Testing: table-driven tests with testify

**Runner (Python):**

```python
# Claude Code SDK integration
from claude_code import AgenticSession

session = AgenticSession(prompt=prompt, workspace=workspace)
result = session.run()
```

**Technology Stack:**
- SDK: claude-code-sdk (>=0.0.23)
- API client: anthropic (>=0.68.0)
- Git: GitPython
- Package manager: uv (preferred over pip)

**Key Files:**
* `components/backend/` - Go backend
* `components/runners/claude-code-runner/` - Python runner
* `components/backend/go.mod` - Go dependencies
* `components/runners/claude-code-runner/requirements.txt` - Python dependencies

**Build Optimization:**

* Go: Multi-stage Docker build, static binary
* Python: uv for fast dependency resolution, layer caching

## Validation

**Performance Metrics:**

* Backend response time: <10ms for simple operations
* Backend concurrency: Handles 100+ concurrent requests
* Runner startup: ~2s (acceptable for long-running sessions)
* Container build time: <2min for both components

**Developer Feedback:**

* Positive: Go backend very stable, easy to debug
* Positive: Python runner easy to extend
* Concern: Context switching between languages
* Mitigation: Clear component boundaries reduce switching

## Links

* Related: ADR-0001 (Kubernetes-Native Architecture)
* [client-go documentation](https://github.com/kubernetes/client-go)
* [Claude Code SDK](https://github.com/anthropics/claude-code-sdk)
```

**File:** `docs/adr/0005-nextjs-shadcn-react-query.md`

```markdown
# ADR-0005: Next.js with Shadcn UI and React Query

**Status:** Accepted
**Date:** 2024-11-21
**Deciders:** Frontend Team
**Technical Story:** Frontend technology stack selection

## Context and Problem Statement

We need to build a modern web UI for the Ambient Code Platform with:
- Server-side rendering for fast initial loads
- Rich interactive components (session monitoring, project management)
- Real-time updates for session status
- Type-safe API integration
- Responsive design with accessible components

What frontend framework and UI library should we use?

## Decision Drivers

* **Modern patterns:** Server components, streaming, type safety
* **Developer experience:** Good tooling, active community
* **UI quality:** Professional design system, accessibility
* **Performance:** Fast initial load, efficient updates
* **Data fetching:** Caching, optimistic updates, real-time sync
* **Team expertise:** React knowledge on team

## Considered Options

1. **Next.js 14 + Shadcn UI + React Query (chosen)**
2. **Create React App + Material-UI + Redux**
3. **Remix + Chakra UI + React Query**
4. **Svelte/SvelteKit + Custom components**

## Decision Outcome

Chosen option: "Next.js 14 + Shadcn UI + React Query", because:

**Next.js 14 (App Router):**
1. **Server components:** Reduced client bundle size
2. **Streaming:** Progressive page rendering
3. **File-based routing:** Intuitive project structure
4. **TypeScript:** First-class type safety
5. **Industry momentum:** Large ecosystem, active development

**Shadcn UI:**
1. **Copy-paste components:** Own your component code
2. **Built on Radix UI:** Accessibility built-in
3. **Tailwind CSS:** Utility-first styling
4. **Customizable:** Full control over styling
5. **No runtime dependency:** Just copy components you need

**React Query:**
1. **Declarative data fetching:** Clean component code
2. **Automatic caching:** Reduces API calls
3. **Optimistic updates:** Better UX
4. **Real-time sync:** Easy integration with WebSockets
5. **DevTools:** Excellent debugging experience

### Consequences

**Positive:**

* **Performance:**
  - Server components reduce client JS by ~40%
  - React Query caching reduces redundant API calls
  - Streaming improves perceived performance

* **Developer Experience:**
  - TypeScript end-to-end (API to UI)
  - Shadcn components copy-pasted and owned
  - React Query hooks simplify data management
  - Next.js DevTools for debugging

* **User Experience:**
  - Fast initial page loads (SSR)
  - Smooth client-side navigation
  - Accessible components (WCAG 2.1 AA)
  - Responsive design (mobile-first)

**Negative:**

* **Learning curve:**
  - Next.js App Router is new (released 2023)
  - Server vs. client component mental model
  - React Query concepts (queries, mutations, invalidation)

* **Complexity:**
  - More moving parts than simple SPA
  - Server component restrictions (no hooks, browser APIs)
  - Hydration errors if server/client mismatch

**Risks:**

* Next.js App Router still evolving (breaking changes possible)
* Shadcn UI components need manual updates (not npm package)
* React Query cache invalidation can be tricky

## Implementation Notes

**Project Structure:**

```
components/frontend/src/
├── app/                    # Next.js App Router pages
│   ├── projects/
│   │   └── [projectName]/
│   │       ├── sessions/
│   │       │   ├── page.tsx           # Sessions list
│   │       │   └── [sessionName]/
│   │       │       └── page.tsx       # Session detail
│   │       └── layout.tsx
│   └── layout.tsx
├── components/
│   ├── ui/                 # Shadcn UI components (owned)
│   │   ├── button.tsx
│   │   ├── card.tsx
│   │   └── dialog.tsx
│   └── [feature]/          # Feature-specific components
├── services/
│   ├── api/                # API client layer
│   │   └── sessions.ts
│   └── queries/            # React Query hooks
│       └── sessions.ts
└── lib/
    └── utils.ts
```

**Key Patterns:**

**1. Server Component for Initial Data**

```typescript
// app/projects/[projectName]/sessions/page.tsx
export default async function SessionsPage({
  params,
}: {
  params: { projectName: string }
}) {
  // Fetch on server for initial render
  const sessions = await sessionApi.list(params.projectName)

  return <SessionsList initialData={sessions} projectName={params.projectName} />
}
```

**2. Client Component with React Query**

```typescript
// components/sessions/sessions-list.tsx
'use client'

import { useSessions } from "@/services/queries/sessions"

export function SessionsList({
  initialData,
  projectName
}: {
  initialData: Session[]
  projectName: string
}) {
  const { data: sessions, isLoading } = useSessions(projectName, {
    initialData,  // Use server data initially
    refetchInterval: 5000,  // Poll every 5s
  })

  return (
    <div>
      {sessions.map(session => (
        <SessionCard key={session.metadata.name} session={session} />
      ))}
    </div>
  )
}
```

**3. Mutations with Optimistic Updates**

```typescript
// services/queries/sessions.ts
export function useCreateSession(projectName: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateSessionRequest) =>
      sessionApi.create(projectName, data),

    onMutate: async (newSession) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: ["sessions", projectName] })

      // Snapshot previous value
      const previous = queryClient.getQueryData(["sessions", projectName])

      // Optimistically update
      queryClient.setQueryData(["sessions", projectName], (old: Session[]) => [
        ...old,
        { ...newSession, status: { phase: "Pending" } },
      ])

      return { previous }
    },

    onError: (err, variables, context) => {
      // Rollback on error
      queryClient.setQueryData(["sessions", projectName], context?.previous)
    },

    onSuccess: () => {
      // Refetch after success
      queryClient.invalidateQueries({ queryKey: ["sessions", projectName] })
    },
  })
}
```

**4. Shadcn Component Usage**

```typescript
import { Button } from "@/components/ui/button"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Dialog, DialogTrigger, DialogContent } from "@/components/ui/dialog"

export function SessionCard({ session }: { session: Session }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{session.metadata.name}</CardTitle>
      </CardHeader>
      <CardContent>
        <Dialog>
          <DialogTrigger asChild>
            <Button variant="outline">View Details</Button>
          </DialogTrigger>
          <DialogContent>
            {/* Session details */}
          </DialogContent>
        </Dialog>
      </CardContent>
    </Card>
  )
}
```

**Technology Versions:**

- Next.js: 14.x (App Router)
- React: 18.x
- Shadcn UI: Latest (no version, copy-paste)
- TanStack React Query: 5.x
- Tailwind CSS: 3.x
- TypeScript: 5.x

**Key Files:**
* `components/frontend/DESIGN_GUIDELINES.md` - Comprehensive patterns
* `components/frontend/src/components/ui/` - Shadcn components
* `components/frontend/src/services/queries/` - React Query hooks
* `components/frontend/src/app/` - Next.js pages

## Validation

**Performance Metrics:**

* Initial page load: <2s (Lighthouse score >90)
* Client bundle size: <200KB (with code splitting)
* Time to Interactive: <3s
* API call reduction: 60% fewer calls (React Query caching)

**Developer Feedback:**

* Positive: React Query simplifies data management significantly
* Positive: Shadcn components easy to customize
* Challenge: Server component restrictions initially confusing
* Resolution: Clear guidelines in DESIGN_GUIDELINES.md

**User Feedback:**

* Fast perceived performance (streaming)
* Smooth interactions (optimistic updates)
* Accessible (keyboard navigation, screen readers)

## Links

* Related: ADR-0004 (Go Backend with Python Runner)
* [Next.js 14 Documentation](https://nextjs.org/docs)
* [Shadcn UI](https://ui.shadcn.com/)
* [TanStack React Query](https://tanstack.com/query/latest)
* Frontend Guidelines: `components/frontend/DESIGN_GUIDELINES.md`
```

### Success Criteria

- [ ] `docs/adr/` directory created
- [ ] ADR template created with complete structure
- [ ] ADR README with index and workflow instructions
- [ ] 5 ADRs created documenting critical architectural decisions
- [ ] Each ADR includes context, options, decision, and consequences

---

## Component 3: Repomix Usage Guide

### Overview

You already have 7 repomix views of the codebase! Create a guide for when to use each one.

### Implementation

**File:** `.claude/repomix-guide.md`

```markdown
# Repomix Context Switching Guide

**Purpose:** Quick reference for loading the right repomix view based on the task.

## Available Views

The `repomix-analysis/` directory contains 7 pre-generated codebase views optimized for different scenarios:

| File | Size | Use When |
|------|------|----------|
| `01-full-context.xml` | 2.1MB | Deep dive into specific component implementation |
| `02-production-optimized.xml` | 4.2MB | General development work, most common use case |
| `03-architecture-only.xml` | 737KB | Understanding system design, new team member onboarding |
| `04-backend-focused.xml` | 403KB | Backend API work (Go handlers, K8s integration) |
| `05-frontend-focused.xml` | 767KB | UI development (NextJS, React Query, Shadcn) |
| `06-ultra-compressed.xml` | 10MB | Quick overview, exploring unfamiliar areas |
| `07-metadata-rich.xml` | 849KB | File structure analysis, refactoring planning |

## Usage Patterns

### Scenario 1: Backend Development

**Task:** Adding a new API endpoint for project settings

**Command:**
```
"Claude, reference the backend-focused repomix view (04-backend-focused.xml) and help me add a new endpoint for updating project settings."
```

**Why this view:**
- Contains all backend handlers and types
- Includes K8s client patterns
- Focused context without frontend noise

### Scenario 2: Frontend Development

**Task:** Creating a new UI component for RFE workflows

**Command:**
```
"Claude, load the frontend-focused repomix view (05-frontend-focused.xml) and help me create a new component for displaying RFE workflow steps."
```

**Why this view:**
- All React components and pages
- Shadcn UI patterns
- React Query hooks

### Scenario 3: Architecture Understanding

**Task:** Explaining the system to a new team member

**Command:**
```
"Claude, using the architecture-only repomix view (03-architecture-only.xml), explain how the operator watches for AgenticSession creation and spawns jobs."
```

**Why this view:**
- High-level component structure
- CRD definitions
- Component relationships
- No implementation details

### Scenario 4: Cross-Component Analysis

**Task:** Tracing a request from frontend through backend to operator

**Command:**
```
"Claude, use the production-optimized repomix view (02-production-optimized.xml) and trace the flow of creating an AgenticSession from UI click to Job creation."
```

**Why this view:**
- Balanced coverage of all components
- Includes key implementation files
- Not overwhelmed with test files

### Scenario 5: Quick Exploration

**Task:** Finding where a specific feature is implemented

**Command:**
```
"Claude, use the ultra-compressed repomix view (06-ultra-compressed.xml) to help me find where multi-repo support is implemented."
```

**Why this view:**
- Fast to process
- Good for keyword searches
- Covers entire codebase breadth

### Scenario 6: Refactoring Planning

**Task:** Planning to break up large handlers/sessions.go file

**Command:**
```
"Claude, analyze the metadata-rich repomix view (07-metadata-rich.xml) and suggest how to split handlers/sessions.go into smaller modules."
```

**Why this view:**
- File size and structure metadata
- Module boundaries
- Import relationships

### Scenario 7: Deep Implementation Dive

**Task:** Debugging a complex operator reconciliation issue

**Command:**
```
"Claude, load the full-context repomix view (01-full-context.xml) and help me understand why the operator is creating duplicate jobs for the same session."
```

**Why this view:**
- Complete implementation details
- All edge case handling
- Full operator logic

## Best Practices

### Start Broad, Then Narrow

1. **First pass:** Use `03-architecture-only.xml` to understand where the feature lives
2. **Second pass:** Use component-specific view (`04-backend` or `05-frontend`)
3. **Deep dive:** Use `01-full-context.xml` for specific implementation details

### Combine with Context Files

For even better results, combine repomix views with context files:

```
"Claude, load the backend-focused repomix view (04) and the backend-development context file, then help me add user token authentication to the new endpoint."
```

### Regenerate Periodically

Repomix views are snapshots in time. Regenerate monthly (or after major changes):

```bash
# Full regeneration
cd repomix-analysis
./regenerate-all.sh  # If you create this script

# Or manually
repomix --output 02-production-optimized.xml --config repomix-production.json
```

**Tip:** Add to monthly maintenance calendar.

## Quick Reference Table

| Task Type | Repomix View | Context File |
|-----------|--------------|--------------|
| Backend API work | 04-backend-focused | backend-development.md |
| Frontend UI work | 05-frontend-focused | frontend-development.md |
| Security review | 02-production-optimized | security-standards.md |
| Architecture overview | 03-architecture-only | - |
| Quick exploration | 06-ultra-compressed | - |
| Refactoring | 07-metadata-rich | - |
| Deep debugging | 01-full-context | (component-specific) |

## Maintenance

**When to regenerate:**
- After major architectural changes
- Monthly (scheduled)
- Before major refactoring efforts
- When views feel "stale" (>2 months old)

**How to regenerate:**
See `.repomixignore` for exclusion patterns. Adjust as needed to balance completeness with token efficiency.
```

### Success Criteria

- [ ] `.claude/repomix-guide.md` created
- [ ] All 7 repomix views documented with use cases
- [ ] Practical examples for each scenario
- [ ] Quick reference table for task-to-view mapping

---

## Component 4: Decision Log

### Overview

Lightweight chronological log of major decisions. Easier to maintain than full ADRs.

### Implementation

**File:** `docs/decisions.md`

```markdown
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
```

### Success Criteria

- [ ] `docs/decisions.md` created
- [ ] Includes template for new entries
- [ ] 7-10 initial entries covering major decisions
- [ ] Each entry links to relevant ADRs, code, and context files

---

## Component 5: Pattern Catalog

### Overview

Document recurring code patterns with concrete examples from the codebase.

### Implementation

**Step 5.1:** Create directory structure

```bash
mkdir -p .claude/patterns
```

**Step 5.2:** Create error handling pattern

**File:** `.claude/patterns/error-handling.md`

```markdown
# Error Handling Patterns

Consistent error handling patterns across backend and operator components.

## Backend Handler Errors

### Pattern 1: Resource Not Found

```go
// handlers/sessions.go:350
func GetSession(c *gin.Context) {
    projectName := c.Param("projectName")
    sessionName := c.Param("sessionName")

    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    if reqK8s == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
        return
    }

    obj, err := reqDyn.Resource(gvr).Namespace(projectName).Get(ctx, sessionName, v1.GetOptions{})
    if errors.IsNotFound(err) {
        c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
        return
    }
    if err != nil {
        log.Printf("Failed to get session %s/%s: %v", projectName, sessionName, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve session"})
        return
    }

    c.JSON(http.StatusOK, obj)
}
```

**Key points:**
- Check `errors.IsNotFound(err)` for 404 scenarios
- Log errors with context (project, session name)
- Return generic error messages to user (don't expose internals)
- Use appropriate HTTP status codes

### Pattern 2: Validation Errors

```go
// handlers/sessions.go:227
func CreateSession(c *gin.Context) {
    var req CreateSessionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        return
    }

    // Validate resource name format
    if !isValidK8sName(req.Name) {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid name: must be a valid Kubernetes DNS label",
        })
        return
    }

    // Validate required fields
    if req.Prompt == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Prompt is required"})
        return
    }

    // ... create session
}
```

**Key points:**
- Validate early, return 400 Bad Request
- Provide specific error messages for validation failures
- Check K8s naming requirements (DNS labels)

### Pattern 3: Authorization Errors

```go
// handlers/sessions.go:250
ssar := &authv1.SelfSubjectAccessReview{
    Spec: authv1.SelfSubjectAccessReviewSpec{
        ResourceAttributes: &authv1.ResourceAttributes{
            Group:     "vteam.ambient-code",
            Resource:  "agenticsessions",
            Verb:      "create",
            Namespace: projectName,
        },
    },
}

res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, v1.CreateOptions{})
if err != nil {
    log.Printf("Authorization check failed: %v", err)
    c.JSON(http.StatusForbidden, gin.H{"error": "Authorization check failed"})
    return
}

if !res.Status.Allowed {
    log.Printf("User not authorized to create sessions in %s", projectName)
    c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to create sessions in this project"})
    return
}
```

**Key points:**
- Always check RBAC before operations
- Return 403 Forbidden for authorization failures
- Log authorization failures for security auditing

## Operator Reconciliation Errors

### Pattern 1: Resource Deleted During Processing

```go
// operator/internal/handlers/sessions.go:85
func handleAgenticSessionEvent(obj *unstructured.Unstructured) error {
    name := obj.GetName()
    namespace := obj.GetNamespace()

    // Verify resource still exists (race condition check)
    currentObj, err := config.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
    if errors.IsNotFound(err) {
        log.Printf("AgenticSession %s/%s no longer exists, skipping reconciliation", namespace, name)
        return nil  // NOT an error - resource was deleted
    }
    if err != nil {
        return fmt.Errorf("failed to get current object: %w", err)
    }

    // ... continue reconciliation with currentObj
}
```

**Key points:**
- `IsNotFound` during reconciliation is NOT an error (resource deleted)
- Return `nil` to avoid retries for deleted resources
- Log the skip for debugging purposes

### Pattern 2: Job Creation Failures

```go
// operator/internal/handlers/sessions.go:125
job := buildJobSpec(sessionName, namespace, spec)

createdJob, err := config.K8sClient.BatchV1().Jobs(namespace).Create(ctx, job, v1.CreateOptions{})
if err != nil {
    log.Printf("Failed to create job for session %s/%s: %v", namespace, sessionName, err)

    // Update session status to reflect error
    updateAgenticSessionStatus(namespace, sessionName, map[string]interface{}{
        "phase":   "Error",
        "message": fmt.Sprintf("Failed to create job: %v", err),
    })

    return fmt.Errorf("failed to create job: %w", err)
}

log.Printf("Created job %s for session %s/%s", createdJob.Name, namespace, sessionName)
```

**Key points:**
- Log failures with full context
- Update CR status to reflect error state
- Return error to trigger retry (if appropriate)
- Include wrapped error for debugging (`%w`)

### Pattern 3: Status Update Failures (Non-Fatal)

```go
// operator/internal/handlers/sessions.go:200
if err := updateAgenticSessionStatus(namespace, sessionName, map[string]interface{}{
    "phase":     "Running",
    "startTime": time.Now().Format(time.RFC3339),
}); err != nil {
    log.Printf("Warning: failed to update status for %s/%s: %v", namespace, sessionName, err)
    // Continue - job was created successfully, status update is secondary
}
```

**Key points:**
- Status updates are often non-fatal (job still created)
- Log as warning, not error
- Don't return error if primary operation succeeded

## Python Runner Errors

### Pattern: Graceful Error Handling with Status Updates

```python
# components/runners/claude-code-runner/wrapper.py
try:
    result = run_claude_session(prompt, workspace, interactive)

    # Update CR with success
    update_session_status(namespace, name, {
        "phase": "Completed",
        "results": result,
        "completionTime": datetime.utcnow().isoformat() + "Z",
    })

except GitError as e:
    logger.error(f"Git operation failed: {e}")
    update_session_status(namespace, name, {
        "phase": "Error",
        "message": f"Git operation failed: {str(e)}",
    })
    sys.exit(1)

except ClaudeAPIError as e:
    logger.error(f"Claude API error: {e}")
    update_session_status(namespace, name, {
        "phase": "Error",
        "message": f"AI service error: {str(e)}",
    })
    sys.exit(1)

except Exception as e:
    logger.error(f"Unexpected error: {e}", exc_info=True)
    update_session_status(namespace, name, {
        "phase": "Error",
        "message": f"Unexpected error: {str(e)}",
    })
    sys.exit(1)
```

**Key points:**
- Catch specific exceptions first, generic last
- Always update CR status before exiting
- Use `exc_info=True` for unexpected errors (full traceback)
- Exit with non-zero code on errors (K8s Job will show failure)

## Anti-Patterns (DO NOT USE)

### ❌ Panic in Production Code

```go
// NEVER DO THIS in handlers or operator
if err != nil {
    panic(fmt.Sprintf("Failed to create session: %v", err))
}
```

**Why wrong:** Crashes the entire process, affects all requests/sessions.
**Use instead:** Return errors, update status, log failures.

### ❌ Silent Failures

```go
// NEVER DO THIS
if err := doSomething(); err != nil {
    // Ignore error, continue
}
```

**Why wrong:** Hides bugs, makes debugging impossible.
**Use instead:** At minimum, log the error. Better: return or update status.

### ❌ Exposing Internal Errors to Users

```go
// DON'T DO THIS
if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{
        "error": fmt.Sprintf("Database query failed: %v", err),  // Exposes internals
    })
}
```

**Why wrong:** Leaks implementation details, security risk.
**Use instead:** Generic user message, detailed log message.

```go
// DO THIS
if err != nil {
    log.Printf("Database query failed: %v", err)  // Detailed log
    c.JSON(http.StatusInternalServerError, gin.H{
        "error": "Failed to retrieve session",  // Generic user message
    })
}
```

## Quick Reference

| Scenario | HTTP Status | Log Level | Return Error? |
|----------|-------------|-----------|---------------|
| Resource not found | 404 | Info | No |
| Invalid input | 400 | Info | No |
| Auth failure | 401/403 | Warning | No |
| K8s API error | 500 | Error | No (user), Yes (operator) |
| Unexpected error | 500 | Error | Yes |
| Status update failure (after success) | - | Warning | No |
| Resource deleted during processing | - | Info | No (return nil) |
```

**Step 5.3:** Create K8s client usage pattern

**File:** `.claude/patterns/k8s-client-usage.md`

```markdown
# Kubernetes Client Usage Patterns

When to use user-scoped clients vs. backend service account clients.

## The Two Client Types

### 1. User-Scoped Clients (reqK8s, reqDyn)

**Created from user's bearer token** extracted from HTTP request.

```go
reqK8s, reqDyn := GetK8sClientsForRequest(c)
if reqK8s == nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
    c.Abort()
    return
}
```

**Use for:**
- ✅ Listing resources in user's namespaces
- ✅ Getting specific resources
- ✅ RBAC permission checks
- ✅ Any operation "on behalf of user"

**Permissions:** Limited to what the user is authorized for via K8s RBAC.

### 2. Backend Service Account Clients (K8sClient, DynamicClient)

**Created from backend service account credentials** (usually cluster-scoped).

```go
// Package-level variables in handlers/
var K8sClient *kubernetes.Clientset
var DynamicClient dynamic.Interface
```

**Use for:**
- ✅ Writing CRs **after** user authorization validated
- ✅ Minting service account tokens for runner pods
- ✅ Cross-namespace operations backend is authorized for
- ✅ Cleanup operations (deleting resources backend owns)

**Permissions:** Elevated (often cluster-admin or namespace-admin).

## Decision Tree

```
┌─────────────────────────────────────────┐
│   Is this a user-initiated operation?   │
└───────────────┬─────────────────────────┘
                │
        ┌───────┴───────┐
        │               │
       YES             NO
        │               │
        ▼               ▼
┌──────────────┐  ┌───────────────┐
│ Use User     │  │ Use Service   │
│ Token Client │  │ Account Client│
│              │  │               │
│ reqK8s       │  │ K8sClient     │
│ reqDyn       │  │ DynamicClient │
└──────────────┘  └───────────────┘
```

## Common Patterns

### Pattern 1: List Resources (User Operation)

```go
// handlers/sessions.go:180
func ListSessions(c *gin.Context) {
    projectName := c.Param("projectName")

    // ALWAYS use user token for list operations
    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    if reqK8s == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
        return
    }

    gvr := types.GetAgenticSessionResource()
    list, err := reqDyn.Resource(gvr).Namespace(projectName).List(ctx, v1.ListOptions{})
    if err != nil {
        log.Printf("Failed to list sessions: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"items": list.Items})
}
```

**Why user token:** User should only see sessions they have permission to view.

### Pattern 2: Create Resource (Validate Then Escalate)

```go
// handlers/sessions.go:227
func CreateSession(c *gin.Context) {
    projectName := c.Param("projectName")

    // Step 1: Get user-scoped clients for validation
    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    if reqK8s == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    // Step 2: Validate request body
    var req CreateSessionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    // Step 3: Check user has permission to create in this namespace
    ssar := &authv1.SelfSubjectAccessReview{
        Spec: authv1.SelfSubjectAccessReviewSpec{
            ResourceAttributes: &authv1.ResourceAttributes{
                Group:     "vteam.ambient-code",
                Resource:  "agenticsessions",
                Verb:      "create",
                Namespace: projectName,
            },
        },
    }
    res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, v1.CreateOptions{})
    if err != nil || !res.Status.Allowed {
        c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to create sessions"})
        return
    }

    // Step 4: NOW use service account to write CR
    //         (backend SA has permission to write CRs in project namespaces)
    obj := buildSessionObject(req, projectName)
    created, err := DynamicClient.Resource(gvr).Namespace(projectName).Create(ctx, obj, v1.CreateOptions{})
    if err != nil {
        log.Printf("Failed to create session: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": "Session created", "name": created.GetName()})
}
```

**Why this pattern:**
1. Validate user identity and permissions (user token)
2. Validate request is well-formed
3. Check RBAC authorization
4. **Then** use service account to perform the write

**This prevents:** User bypassing RBAC by using backend's elevated permissions.

### Pattern 3: Minting Tokens for Runner Pods

```go
// handlers/sessions.go:449 (in createRunnerJob function)
func createRunnerJob(sessionName, namespace string, spec map[string]interface{}) error {
    // Create service account for this session
    sa := &corev1.ServiceAccount{
        ObjectMeta: v1.ObjectMeta{
            Name:      fmt.Sprintf("%s-sa", sessionName),
            Namespace: namespace,
        },
    }

    // MUST use backend service account to create SA
    _, err := K8sClient.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, v1.CreateOptions{})
    if err != nil {
        return fmt.Errorf("failed to create service account: %w", err)
    }

    // Mint token for the service account
    tokenRequest := &authv1.TokenRequest{
        Spec: authv1.TokenRequestSpec{
            ExpirationSeconds: int64Ptr(3600),
        },
    }

    // MUST use backend service account to mint tokens
    tokenResponse, err := K8sClient.CoreV1().ServiceAccounts(namespace).CreateToken(
        ctx,
        sa.Name,
        tokenRequest,
        v1.CreateOptions{},
    )
    if err != nil {
        return fmt.Errorf("failed to create token: %w", err)
    }

    // Store token in secret
    secret := &corev1.Secret{
        ObjectMeta: v1.ObjectMeta{
            Name:      fmt.Sprintf("%s-token", sessionName),
            Namespace: namespace,
        },
        StringData: map[string]string{
            "token": tokenResponse.Status.Token,  // NEVER log this
        },
    }

    _, err = K8sClient.CoreV1().Secrets(namespace).Create(ctx, secret, v1.CreateOptions{})
    return err
}
```

**Why service account:** Only backend SA has permission to mint tokens. Users should not be able to mint arbitrary tokens.

### Pattern 4: Cross-Namespace Operations

```go
// handlers/projects.go (hypothetical)
func ListAllProjects(c *gin.Context) {
    // User wants to list all projects they can access across all namespaces

    reqK8s, _ := GetK8sClientsForRequest(c)
    if reqK8s == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    // List namespaces user can access (use user token)
    nsList, err := reqK8s.CoreV1().Namespaces().List(ctx, v1.ListOptions{
        LabelSelector: "vteam.ambient-code/project=true",
    })
    if err != nil {
        log.Printf("Failed to list namespaces: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
        return
    }

    // Return list of accessible projects
    projects := make([]string, 0, len(nsList.Items))
    for _, ns := range nsList.Items {
        projects = append(projects, ns.Name)
    }

    c.JSON(http.StatusOK, gin.H{"projects": projects})
}
```

**Why user token:** User should only see namespaces they have access to. Using service account would show ALL namespaces.

## Anti-Patterns (DO NOT USE)

### ❌ Using Service Account for List Operations

```go
// NEVER DO THIS
func ListSessions(c *gin.Context) {
    projectName := c.Param("projectName")

    // ❌ BAD: Using service account bypasses RBAC
    list, err := DynamicClient.Resource(gvr).Namespace(projectName).List(ctx, v1.ListOptions{})

    c.JSON(http.StatusOK, gin.H{"items": list.Items})
}
```

**Why wrong:** User could access resources they don't have permission to see.

### ❌ Falling Back to Service Account on Auth Failure

```go
// NEVER DO THIS
func GetSession(c *gin.Context) {
    reqK8s, reqDyn := GetK8sClientsForRequest(c)

    // ❌ BAD: Falling back to service account if user token invalid
    if reqK8s == nil {
        log.Println("User token invalid, using service account")
        reqDyn = DynamicClient  // SECURITY VIOLATION
    }

    obj, _ := reqDyn.Resource(gvr).Namespace(project).Get(ctx, name, v1.GetOptions{})
    c.JSON(http.StatusOK, obj)
}
```

**Why wrong:** Bypasses authentication entirely. User with invalid token shouldn't get access via backend SA.

### ❌ Not Checking RBAC Before Service Account Operations

```go
// NEVER DO THIS
func CreateSession(c *gin.Context) {
    var req CreateSessionRequest
    c.ShouldBindJSON(&req)

    // ❌ BAD: Using service account without checking user permissions
    obj := buildSessionObject(req, projectName)
    created, _ := DynamicClient.Resource(gvr).Namespace(projectName).Create(ctx, obj, v1.CreateOptions{})

    c.JSON(http.StatusCreated, created)
}
```

**Why wrong:** User can create resources they don't have permission to create.

## Quick Reference

| Operation | Use User Token | Use Service Account |
|-----------|----------------|---------------------|
| List resources in namespace | ✅ | ❌ |
| Get specific resource | ✅ | ❌ |
| RBAC permission check | ✅ | ❌ |
| Create CR (after RBAC validation) | ❌ | ✅ |
| Update CR status | ❌ | ✅ |
| Delete resource user created | ✅ | ⚠️  (can use either) |
| Mint service account token | ❌ | ✅ |
| Create Job for session | ❌ | ✅ |
| Cleanup orphaned resources | ❌ | ✅ |

**Legend:**
- ✅ Correct choice
- ❌ Wrong choice (security violation)
- ⚠️  Context-dependent

## Validation Checklist

Before merging code that uses K8s clients:

- [ ] User operations use `GetK8sClientsForRequest(c)`
- [ ] Return 401 if user client creation fails
- [ ] RBAC check performed before using service account to write
- [ ] Service account used ONLY for privileged operations
- [ ] No fallback to service account on auth failures
- [ ] Tokens never logged (use `len(token)` instead)
```

**Step 5.4:** Create React Query usage pattern

**File:** `.claude/patterns/react-query-usage.md`

```markdown
# React Query Usage Patterns

Standard patterns for data fetching, mutations, and cache management in the frontend.

## Core Principles

1. **ALL data fetching uses React Query** - No manual `fetch()` in components
2. **Queries for reads** - `useQuery` for GET operations
3. **Mutations for writes** - `useMutation` for POST/PUT/DELETE
4. **Cache invalidation** - Invalidate queries after mutations
5. **Optimistic updates** - Update UI before server confirms

## File Structure

```
src/services/
├── api/                    # API client layer (pure functions)
│   ├── sessions.ts         # sessionApi.list(), .create(), .delete()
│   ├── projects.ts
│   └── common.ts           # Shared fetch logic, error handling
└── queries/                # React Query hooks
    ├── sessions.ts         # useSessions(), useCreateSession()
    ├── projects.ts
    └── common.ts           # Query client config
```

**Separation of concerns:**
- `api/`: Pure API functions (no React, no hooks)
- `queries/`: React Query hooks that use API functions

## Pattern 1: Query Hook (List Resources)

```typescript
// services/queries/sessions.ts
import { useQuery } from "@tanstack/react-query"
import { sessionApi } from "@/services/api/sessions"

export function useSessions(projectName: string) {
  return useQuery({
    queryKey: ["sessions", projectName],
    queryFn: () => sessionApi.list(projectName),
    staleTime: 5000,          // Consider data fresh for 5s
    refetchInterval: 10000,   // Poll every 10s for updates
  })
}
```

**Usage in component:**

```typescript
// app/projects/[projectName]/sessions/page.tsx
'use client'

import { useSessions } from "@/services/queries/sessions"

export function SessionsList({ projectName }: { projectName: string }) {
  const { data: sessions, isLoading, error } = useSessions(projectName)

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>
  if (!sessions?.length) return <div>No sessions found</div>

  return (
    <div>
      {sessions.map(session => (
        <SessionCard key={session.metadata.name} session={session} />
      ))}
    </div>
  )
}
```

**Key points:**
- `queryKey` includes all parameters that affect the query
- `staleTime` prevents unnecessary refetches
- `refetchInterval` for polling (optional)
- Destructure `data`, `isLoading`, `error` for UI states

## Pattern 2: Query Hook (Single Resource)

```typescript
// services/queries/sessions.ts
export function useSession(projectName: string, sessionName: string) {
  return useQuery({
    queryKey: ["sessions", projectName, sessionName],
    queryFn: () => sessionApi.get(projectName, sessionName),
    enabled: !!sessionName,  // Only run if sessionName provided
    staleTime: 3000,
  })
}
```

**Usage:**

```typescript
// app/projects/[projectName]/sessions/[sessionName]/page.tsx
'use client'

export function SessionDetailPage({ params }: {
  params: { projectName: string; sessionName: string }
}) {
  const { data: session, isLoading } = useSession(
    params.projectName,
    params.sessionName
  )

  if (isLoading) return <div>Loading session...</div>
  if (!session) return <div>Session not found</div>

  return <SessionDetail session={session} />
}
```

**Key points:**
- `enabled: !!sessionName` prevents query if parameter missing
- More specific queryKey for targeted cache invalidation

## Pattern 3: Create Mutation with Optimistic Update

```typescript
// services/queries/sessions.ts
import { useMutation, useQueryClient } from "@tanstack/react-query"

export function useCreateSession(projectName: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateSessionRequest) =>
      sessionApi.create(projectName, data),

    // Optimistic update: show immediately before server confirms
    onMutate: async (newSession) => {
      // Cancel any outgoing refetches (prevent overwriting optimistic update)
      await queryClient.cancelQueries({
        queryKey: ["sessions", projectName]
      })

      // Snapshot current value
      const previousSessions = queryClient.getQueryData([
        "sessions",
        projectName
      ])

      // Optimistically update cache
      queryClient.setQueryData(
        ["sessions", projectName],
        (old: AgenticSession[] | undefined) => [
          ...(old || []),
          {
            metadata: { name: newSession.name },
            spec: newSession,
            status: { phase: "Pending" },  // Optimistic status
          },
        ]
      )

      // Return context with snapshot
      return { previousSessions }
    },

    // Rollback on error
    onError: (err, variables, context) => {
      queryClient.setQueryData(
        ["sessions", projectName],
        context?.previousSessions
      )

      // Show error toast/notification
      console.error("Failed to create session:", err)
    },

    // Refetch after success (get real data from server)
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["sessions", projectName]
      })
    },
  })
}
```

**Usage:**

```typescript
// components/sessions/create-session-dialog.tsx
'use client'

import { useCreateSession } from "@/services/queries/sessions"
import { Button } from "@/components/ui/button"

export function CreateSessionDialog({ projectName }: { projectName: string }) {
  const createSession = useCreateSession(projectName)

  const handleSubmit = (data: CreateSessionRequest) => {
    createSession.mutate(data)
  }

  return (
    <form onSubmit={handleSubmit}>
      {/* form fields */}
      <Button
        type="submit"
        disabled={createSession.isPending}
      >
        {createSession.isPending ? "Creating..." : "Create Session"}
      </Button>
    </form>
  )
}
```

**Key points:**
- `onMutate`: Optimistic update (runs before server call)
- `onError`: Rollback on failure
- `onSuccess`: Invalidate queries to refetch real data
- Use `isPending` for loading states

## Pattern 4: Delete Mutation

```typescript
// services/queries/sessions.ts
export function useDeleteSession(projectName: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (sessionName: string) =>
      sessionApi.delete(projectName, sessionName),

    // Optimistic delete
    onMutate: async (sessionName) => {
      await queryClient.cancelQueries({
        queryKey: ["sessions", projectName]
      })

      const previousSessions = queryClient.getQueryData([
        "sessions",
        projectName
      ])

      // Remove from cache
      queryClient.setQueryData(
        ["sessions", projectName],
        (old: AgenticSession[] | undefined) =>
          old?.filter(s => s.metadata.name !== sessionName) || []
      )

      return { previousSessions }
    },

    onError: (err, sessionName, context) => {
      queryClient.setQueryData(
        ["sessions", projectName],
        context?.previousSessions
      )
    },

    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["sessions", projectName]
      })
    },
  })
}
```

**Usage:**

```typescript
const deleteSession = useDeleteSession(projectName)

<Button
  variant="destructive"
  onClick={() => deleteSession.mutate(sessionName)}
  disabled={deleteSession.isPending}
>
  {deleteSession.isPending ? "Deleting..." : "Delete"}
</Button>
```

## Pattern 5: Dependent Queries

```typescript
// services/queries/sessions.ts
export function useSessionResults(
  projectName: string,
  sessionName: string
) {
  // First, get the session
  const sessionQuery = useSession(projectName, sessionName)

  // Then, get results (only if session is completed)
  const resultsQuery = useQuery({
    queryKey: ["sessions", projectName, sessionName, "results"],
    queryFn: () => sessionApi.getResults(projectName, sessionName),
    enabled: sessionQuery.data?.status.phase === "Completed",
  })

  return {
    session: sessionQuery.data,
    results: resultsQuery.data,
    isLoading: sessionQuery.isLoading || resultsQuery.isLoading,
  }
}
```

**Key points:**
- `enabled` depends on first query's data
- Second query doesn't run until first succeeds

## Pattern 6: Polling Until Condition Met

```typescript
// services/queries/sessions.ts
export function useSessionWithPolling(
  projectName: string,
  sessionName: string
) {
  return useQuery({
    queryKey: ["sessions", projectName, sessionName],
    queryFn: () => sessionApi.get(projectName, sessionName),
    refetchInterval: (query) => {
      const session = query.state.data

      // Stop polling if completed or error
      if (session?.status.phase === "Completed" ||
          session?.status.phase === "Error") {
        return false  // Stop polling
      }

      return 3000  // Poll every 3s while running
    },
  })
}
```

**Key points:**
- Dynamic `refetchInterval` based on query data
- Return `false` to stop polling
- Return number (ms) to continue polling

## API Client Layer Pattern

```typescript
// services/api/sessions.ts
import { API_BASE_URL } from "@/config"
import type { AgenticSession, CreateSessionRequest } from "@/types/session"

async function fetchWithAuth(url: string, options: RequestInit = {}) {
  const token = getAuthToken()  // From auth context or storage

  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${token}`,
      ...options.headers,
    },
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.message || "Request failed")
  }

  return response.json()
}

export const sessionApi = {
  list: async (projectName: string): Promise<AgenticSession[]> => {
    const data = await fetchWithAuth(
      `${API_BASE_URL}/projects/${projectName}/agentic-sessions`
    )
    return data.items || []
  },

  get: async (
    projectName: string,
    sessionName: string
  ): Promise<AgenticSession> => {
    return fetchWithAuth(
      `${API_BASE_URL}/projects/${projectName}/agentic-sessions/${sessionName}`
    )
  },

  create: async (
    projectName: string,
    data: CreateSessionRequest
  ): Promise<AgenticSession> => {
    return fetchWithAuth(
      `${API_BASE_URL}/projects/${projectName}/agentic-sessions`,
      {
        method: "POST",
        body: JSON.stringify(data),
      }
    )
  },

  delete: async (projectName: string, sessionName: string): Promise<void> => {
    return fetchWithAuth(
      `${API_BASE_URL}/projects/${projectName}/agentic-sessions/${sessionName}`,
      {
        method: "DELETE",
      }
    )
  },
}
```

**Key points:**
- Shared `fetchWithAuth` for token injection
- Pure functions (no React, no hooks)
- Type-safe inputs and outputs
- Centralized error handling

## Cache Invalidation Strategies

### Strategy 1: Invalidate Parent Query After Mutation

```typescript
onSuccess: () => {
  // Invalidate list after creating/deleting item
  queryClient.invalidateQueries({
    queryKey: ["sessions", projectName]
  })
}
```

### Strategy 2: Invalidate Multiple Related Queries

```typescript
onSuccess: () => {
  // Invalidate both list and individual session
  queryClient.invalidateQueries({
    queryKey: ["sessions", projectName]
  })
  queryClient.invalidateQueries({
    queryKey: ["sessions", projectName, sessionName]
  })
}
```

### Strategy 3: Exact vs. Fuzzy Matching

```typescript
// Exact match: Only ["sessions", "project-1"]
queryClient.invalidateQueries({
  queryKey: ["sessions", "project-1"],
  exact: true,
})

// Fuzzy match: All queries starting with ["sessions", "project-1"]
// Includes ["sessions", "project-1", "session-1"], etc.
queryClient.invalidateQueries({
  queryKey: ["sessions", "project-1"]
})
```

## Anti-Patterns (DO NOT USE)

### ❌ Manual fetch() in Components

```typescript
// NEVER DO THIS
const [sessions, setSessions] = useState([])

useEffect(() => {
  fetch('/api/sessions')
    .then(r => r.json())
    .then(setSessions)
}, [])
```

**Why wrong:** No caching, no automatic refetching, manual state management.
**Use instead:** React Query hooks.

### ❌ Not Using Query Keys Properly

```typescript
// BAD: Same query key for different data
useQuery({
  queryKey: ["sessions"],  // Missing projectName!
  queryFn: () => sessionApi.list(projectName),
})
```

**Why wrong:** Cache collisions, wrong data shown.
**Use instead:** Include all parameters in query key.

### ❌ Mutating State Directly in onSuccess

```typescript
// BAD: Manually updating state instead of cache
onSuccess: (newSession) => {
  setSessions([...sessions, newSession])  // Wrong!
}
```

**Why wrong:** Bypasses React Query cache, causes sync issues.
**Use instead:** Invalidate queries or update cache via `setQueryData`.

## Quick Reference

| Pattern | Hook | When to Use |
|---------|------|-------------|
| List resources | `useQuery` | GET /resources |
| Get single resource | `useQuery` | GET /resources/:id |
| Create resource | `useMutation` | POST /resources |
| Update resource | `useMutation` | PUT /resources/:id |
| Delete resource | `useMutation` | DELETE /resources/:id |
| Polling | `useQuery` + `refetchInterval` | Real-time updates |
| Optimistic update | `onMutate` | Instant UI feedback |
| Dependent query | `enabled` | Query depends on another |

## Validation Checklist

Before merging frontend code:

- [ ] All data fetching uses React Query (no manual fetch)
- [ ] Query keys include all relevant parameters
- [ ] Mutations invalidate related queries
- [ ] Loading and error states handled
- [ ] Optimistic updates for create/delete (where appropriate)
- [ ] API client layer is pure functions (no hooks)
```

### Success Criteria

- [ ] `.claude/patterns/` directory created
- [ ] Three pattern files created (error-handling, k8s-client-usage, react-query-usage)
- [ ] Each pattern includes concrete examples from the codebase
- [ ] Anti-patterns documented with explanations
- [ ] Quick reference tables for easy lookup

---

## Validation & Next Steps

### Overall Success Criteria

Once all components are implemented:

- [ ] `.claude/context/` with 3 context files (backend, frontend, security)
- [ ] `docs/adr/` with template, README, and 5 ADRs
- [ ] `.claude/repomix-guide.md` with usage guide
- [ ] `docs/decisions.md` with decision log and template
- [ ] `.claude/patterns/` with 3 pattern files

### How to Use This Plan

**Option 1: Execute Yourself**

1. Create directories: `mkdir -p .claude/context docs/adr .claude/patterns`
2. Copy file content from this plan into each file
3. Review and customize for your specific needs
4. Commit: `git add .claude/ docs/ && git commit -m "feat: implement memory system for better Claude context"`

**Option 2: Have Claude Execute**

```
Claude, execute the memory system implementation plan in docs/implementation-plans/memory-system-implementation.md
```

**Option 3: Incremental Implementation**

Implement one component at a time:
1. Start with context files (immediate value)
2. Add ADRs (captures knowledge)
3. Add repomix guide (leverages existing assets)
4. Add decision log (lightweight tracking)
5. Add pattern catalog (codifies best practices)

### Maintenance Schedule

**Weekly:**
- [ ] Add new decisions to `docs/decisions.md` as they're made

**Monthly:**
- [ ] Review and update context files with new patterns
- [ ] Add new ADRs for significant architectural changes
- [ ] Regenerate repomix views if codebase has changed significantly

**Quarterly:**
- [ ] Review ADRs for accuracy (mark deprecated if needed)
- [ ] Update pattern catalog with new patterns discovered
- [ ] Audit context files for outdated information

### Measuring Success

You'll know this system is working when:

1. **Claude gives more accurate responses** - Especially for security and architecture questions
2. **Onboarding is faster** - New team members (or Claude sessions) understand context quickly
3. **Decisions are traceable** - "Why did we do it this way?" has documented answers
4. **Patterns are reused** - Less reinventing the wheel, more consistent code

---

## Appendix: Example Claude Prompts

Once this system is in place, you can use prompts like:

**Backend Work:**
```
Claude, load the backend-development context file and the backend-focused repomix view (04).
Help me add a new endpoint for listing RFE workflows in a project.
```

**Security Review:**
```
Claude, reference the security-standards context file and review this PR for token handling issues.
```

**Architecture Question:**
```
Claude, check ADR-0002 (User Token Authentication) and explain why we use user tokens instead of service accounts for API operations.
```

**Pattern Application:**
```
Claude, use the error-handling pattern file and help me add proper error handling to this handler function.
```

**Cross-Component Analysis:**
```
Claude, load the production-optimized repomix view and trace how an AgenticSession creation flows from frontend to backend to operator.
```

---

**End of Implementation Plan**

This plan is now ready to be executed by you or by Claude Code. All file content is copy-paste ready, and success criteria are clearly defined for each component.
