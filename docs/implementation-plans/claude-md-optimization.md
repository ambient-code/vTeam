# CLAUDE.md Optimization Plan

**Status:** Ready for Execution
**Created:** 2024-11-21
**Prerequisite:** Memory system implementation complete (issue #357)
**Estimated Time:** 20-30 minutes
**Context Required:** None (coldstartable)

## Executive Summary

This plan optimizes `CLAUDE.md` to work as a "routing layer" that points to the new memory system files, rather than containing all context inline. This reduces cognitive load when CLAUDE.md is loaded (which happens every session) while making deep context available on-demand.

**Core Principle:** CLAUDE.md becomes a table of contents with mandatory reading (universal rules) and optional deep dives (memory files).

## Goal

Transform CLAUDE.md from:
- ❌ Monolithic context file with all patterns inline
- ❌ ~2000+ lines of detailed examples
- ❌ Historical decision explanations

To:
- ✅ Routing layer with memory system guide
- ✅ Universal rules that always apply
- ✅ Signposts to deeper context ("For X, load Y")
- ✅ ~1200-1500 lines focused on essentials

## What Stays vs. What Moves

### STAYS in CLAUDE.md (Universal Rules)

**Keep if it:**
- ✅ NEVER has exceptions (e.g., "NEVER push to main")
- ✅ Applies to ALL work (e.g., branch verification)
- ✅ Is a routing decision (e.g., "For backend work, load X")
- ✅ Is a build/deploy command (e.g., `make dev-start`)

**Examples:**
- MANDATORY branch verification before file changes
- NEVER change GitHub repo visibility without permission
- Pre-push linting workflow (ALWAYS run before push)
- Project overview and architecture
- Build commands and development setup
- Critical backend/frontend rules (5-10 per component)

### MOVES to Memory Files (Deep Context)

**Move if it:**
- ❌ Shows HOW to implement (examples → patterns)
- ❌ Explains WHY we decided (rationale → ADRs)
- ❌ Is component-specific deep pattern (→ context files)
- ❌ Has conditions/scenarios (→ context/pattern files)

**Examples:**
- Detailed Go handler patterns → `.claude/context/backend-development.md`
- React Query examples → `.claude/patterns/react-query-usage.md`
- "Why user tokens?" explanation → `docs/adr/0002-user-token-authentication.md`
- K8s client decision tree → `.claude/patterns/k8s-client-usage.md`

## Content Mapping

| Current CLAUDE.md Section | Stays? | Moves To | Replaced With |
|---------------------------|--------|----------|---------------|
| Project Overview | ✅ Stay | - | Keep as-is |
| Development Commands | ✅ Stay | - | Keep as-is |
| Backend Development Standards | ⚠️ Slim | backend-development.md | Critical rules + link |
| Frontend Development Standards | ⚠️ Slim | frontend-development.md | Critical rules + link |
| Backend/Operator Patterns | ❌ Move | patterns/*.md | "See patterns/" |
| Deep code examples | ❌ Move | context/*.md | "Load context file" |
| Security patterns | ⚠️ Slim | security-standards.md | Critical rules + link |
| Testing Strategy | ✅ Stay | - | Keep as-is |
| ADR-like explanations | ❌ Move | docs/adr/*.md | "See ADR-NNNN" |

## Implementation Steps

### Step 1: Add Memory System Guide Section

**Location:** After "Table of Contents", before "Jeremy's Current Context"

**Insert this complete section:**

```markdown
## Memory System Guide

The platform uses a structured "memory system" to provide context on-demand instead of loading everything upfront. This section explains what memory files exist and when to use them.

### Memory System Overview

| Memory Type | Location | Use When | Example Prompt |
|-------------|----------|----------|----------------|
| **Context Files** | `.claude/context/` | Working in specific area of codebase | "Claude, load backend-development context and help me add an endpoint" |
| **ADRs** | `docs/adr/` | Understanding why architectural decisions were made | "Claude, check ADR-0002 and explain user token authentication" |
| **Repomix Views** | `repomix-analysis/` | Deep codebase exploration and tracing flows | "Claude, load backend-focused repomix (04) and trace session creation" |
| **Decision Log** | `docs/decisions.md` | Quick timeline of what changed when | "Claude, check decision log for multi-repo support changes" |
| **Patterns** | `.claude/patterns/` | Applying established code patterns | "Claude, use the error-handling pattern in this handler" |

### Available Context Files

#### Backend Development
**File:** `.claude/context/backend-development.md`

**Contains:**
- Go handler patterns and best practices
- K8s client usage (user token vs. service account)
- Authentication and authorization patterns
- Error handling for handlers and middleware
- Type-safe unstructured resource access

**Load when:**
- Adding new API endpoints
- Modifying backend handlers
- Working with Kubernetes resources
- Implementing authentication/authorization

**Example prompt:**
```
Claude, load the backend-development context file and help me add
a new endpoint for updating project settings with proper RBAC validation.
```

#### Frontend Development
**File:** `.claude/context/frontend-development.md`

**Contains:**
- Next.js App Router patterns
- Shadcn UI component usage
- React Query data fetching patterns
- TypeScript best practices (zero `any` types)
- Component organization and colocation

**Load when:**
- Creating new UI components
- Implementing data fetching
- Adding new pages/routes
- Working with forms or dialogs

**Example prompt:**
```
Claude, load the frontend-development context and help me create
a new page for RFE workflow visualization with proper React Query hooks.
```

#### Security Standards
**File:** `.claude/context/security-standards.md`

**Contains:**
- Token handling and redaction patterns
- RBAC enforcement patterns
- Input validation strategies
- Container security settings
- Security review checklist

**Load when:**
- Implementing authentication/authorization
- Handling sensitive data
- Security reviews
- Adding RBAC checks

**Example prompt:**
```
Claude, reference the security-standards context and review this PR
for token handling issues and RBAC violations.
```

### Available Patterns

#### Error Handling
**File:** `.claude/patterns/error-handling.md`

**Contains:**
- Backend handler error patterns (404, 400, 403, 500)
- Operator reconciliation error handling
- Python runner error patterns
- Anti-patterns to avoid

**Apply when:** Adding error handling to handlers, operators, or runners

#### K8s Client Usage
**File:** `.claude/patterns/k8s-client-usage.md`

**Contains:**
- User-scoped client vs. service account decision tree
- Common patterns for list/create/delete operations
- Validation-then-escalate pattern for writes
- Anti-patterns and security violations

**Apply when:** Working with Kubernetes API, implementing RBAC

#### React Query Usage
**File:** `.claude/patterns/react-query-usage.md`

**Contains:**
- Query hooks for GET operations
- Mutation hooks for create/update/delete
- Optimistic updates and cache invalidation
- Polling and dependent queries

**Apply when:** Implementing frontend data fetching or mutations

### Available Architectural Decision Records (ADRs)

ADRs document WHY architectural decisions were made, not just WHAT was implemented.

**Location:** `docs/adr/`

**Current ADRs:**
- [ADR-0001](../adr/0001-kubernetes-native-architecture.md): Kubernetes-Native Architecture
- [ADR-0002](../adr/0002-user-token-authentication.md): User Token Authentication for API Operations
- [ADR-0003](../adr/0003-multi-repo-support.md): Multi-Repository Support in AgenticSessions
- [ADR-0004](../adr/0004-go-backend-python-runner.md): Go Backend with Python Claude Runner
- [ADR-0005](../adr/0005-nextjs-shadcn-react-query.md): Next.js with Shadcn UI and React Query

**Example usage:**
```
Claude, check ADR-0002 (User Token Authentication) and explain why we
validate user permissions before using the service account to create resources.
```

### Repomix Views Guide

**File:** `.claude/repomix-guide.md`

Contains usage guide for the 7 pre-generated repomix views (architecture-only, backend-focused, frontend-focused, etc.).

**Example usage:**
```
Claude, load the backend-focused repomix view (04) and trace how
AgenticSession creation flows from the API handler to the operator.
```

### Decision Log

**File:** `docs/decisions.md`

Chronological record of major decisions with brief rationale.

**Example usage:**
```
Claude, check the decision log for when multi-repo support was added
and what gotchas were discovered.
```

### How to Use the Memory System

#### Scenario 1: Backend API Work

**Prompt:**
```
Claude, load the backend-development context file and the backend-focused
repomix view (04). Help me add a new endpoint for listing RFE workflows
with proper pagination and RBAC validation.
```

**What Claude loads:**
- Backend development patterns
- K8s client usage patterns
- Existing handler examples from repomix
- RBAC patterns from security context

#### Scenario 2: Frontend Feature

**Prompt:**
```
Claude, load the frontend-development context and the react-query-usage pattern.
Help me add optimistic updates to the session deletion flow.
```

**What Claude loads:**
- Next.js and Shadcn UI patterns
- React Query mutation patterns
- Optimistic update examples

#### Scenario 3: Security Review

**Prompt:**
```
Claude, reference the security-standards context file and review handlers/sessions.go
for token handling issues, RBAC violations, and input validation problems.
```

**What Claude loads:**
- Security patterns and anti-patterns
- Token redaction requirements
- RBAC enforcement checklist

#### Scenario 4: Understanding Architecture

**Prompt:**
```
Claude, check ADR-0001 (Kubernetes-Native Architecture) and explain
why we chose CRDs and Operators instead of traditional microservices.
```

**What Claude loads:**
- Decision context and alternatives considered
- Trade-offs and consequences
- Implementation notes

### Quick Reference: Task → Memory File Mapping

```
Task Type                    → Load This Memory File
──────────────────────────────────────────────────────────
Backend endpoint work        → backend-development.md + k8s-client-usage.md
Frontend UI work             → frontend-development.md + react-query-usage.md
Security review              → security-standards.md
Error handling               → error-handling.md
Why did we choose X?         → docs/adr/NNNN-*.md (relevant ADR)
What changed when?           → docs/decisions.md
Deep codebase exploration    → repomix-analysis/*.xml + repomix-guide.md
Applying a pattern           → .claude/patterns/*.md
```

### Memory System Maintenance

**Weekly:**
- Add new decisions to `docs/decisions.md`

**Monthly:**
- Update context files with new patterns discovered
- Add ADRs for significant architectural changes
- Regenerate repomix views if major codebase changes

**Quarterly:**
- Review ADRs for accuracy (mark deprecated if needed)
- Update pattern catalog
- Audit context files for outdated information

---
```

**Action:** Insert this entire section into CLAUDE.md after the Table of Contents.

### Step 2: Update Backend Development Standards Section

**Current location:** "## Backend and Operator Development Standards"

**Changes:**

1. **Add introductory paragraph with context file link:**

```markdown
## Backend and Operator Development Standards

**For detailed patterns and examples, load:** `.claude/context/backend-development.md`

**This section contains CRITICAL RULES that always apply.** For deep patterns, code examples, and detailed explanations, use the context file above.
```

2. **Keep only critical rules, slim down examples:**

**KEEP:**
- Critical Rules (Never Violate) - all 5 rules
- Package Organization - structure only, no examples
- Pre-Commit Checklist

**SLIM DOWN (add link instead):**

Replace detailed code examples with:

```markdown
### Kubernetes Client Patterns

**CRITICAL RULE:** Always use user-scoped clients for API operations.

**For detailed patterns and examples:** Load `.claude/patterns/k8s-client-usage.md`

**Quick reference:**
- User-scoped clients (`reqK8s`, `reqDyn`): For all user-initiated operations
- Service account clients (`K8sClient`, `DynamicClient`): ONLY for privileged operations after validation

**Pattern:**
```go
// 1. Get user-scoped clients
reqK8s, reqDyn := GetK8sClientsForRequest(c)
if reqK8s == nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
    return
}

// 2. Use for operations
list, err := reqDyn.Resource(gvr).Namespace(project).List(ctx, v1.ListOptions{})
```

**For complete patterns:** See `.claude/patterns/k8s-client-usage.md`
```

**REMOVE (now in context files):**
- All detailed code examples (>20 lines)
- Anti-patterns sections (move to patterns file)
- "How to" sections with step-by-step (move to context)

### Step 3: Update Frontend Development Standards Section

**Current location:** "## Frontend Development Standards"

**Changes:**

1. **Add introductory paragraph:**

```markdown
## Frontend Development Standards

**For detailed patterns and examples, load:** `.claude/context/frontend-development.md`

**This section contains CRITICAL RULES that always apply.** See `components/frontend/DESIGN_GUIDELINES.md` and the context file for complete patterns.
```

2. **Keep Critical Rules (Quick Reference) section as-is** - these 5 rules are non-negotiable

3. **Slim down Pre-Commit Checklist** with link:

```markdown
### Pre-Commit Checklist for Frontend

**Quick checklist:**
- [ ] Zero `any` types
- [ ] All UI uses Shadcn components
- [ ] All data operations use React Query
- [ ] `npm run build` passes with 0 errors, 0 warnings

**For complete checklist:** See `.claude/context/frontend-development.md` or `components/frontend/DESIGN_GUIDELINES.md`
```

4. **Replace Reference Files section:**

```markdown
### Reference Files

**For detailed frontend patterns:**
- `.claude/context/frontend-development.md` - Component patterns, React Query, TypeScript
- `.claude/patterns/react-query-usage.md` - Data fetching patterns
- `components/frontend/DESIGN_GUIDELINES.md` - Comprehensive design guidelines
- `components/frontend/COMPONENT_PATTERNS.md` - Architecture patterns
```

### Step 4: Add Quick Links to Other Sections

**Security-related sections:** Add link to security context

**Example for "Production Considerations → Security" section:**

```markdown
### Security

**For detailed security patterns:** Load `.claude/context/security-standards.md`

**Critical requirements:**
- API keys stored in Kubernetes Secrets
- RBAC: Namespace-scoped isolation
- OAuth integration: OpenShift OAuth for cluster-based authentication
- Network policies: Component isolation

**See also:**
- ADR-0002: User Token Authentication
- Pattern: Token handling and redaction
```

**Testing sections:** Add link to E2E guide

```markdown
### E2E Tests (Cypress + Kind)

**Full guide:** `docs/testing/e2e-guide.md`

**Quick reference:**
- Purpose: Automated end-to-end testing in Kubernetes
- Location: `e2e/`
- Command: `make e2e-test CONTAINER_ENGINE=podman`
```

### Step 5: Update Table of Contents

**Add new section to TOC:**

```markdown
## Table of Contents

- [Memory System Guide](#memory-system-guide)  ← NEW
- [Dynamic Framework Selection](#dynamic-framework-selection)
- [Core Operating Philosophy](#core-operating-philosophy)
- [Strategic Analysis Framework](#strategic-analysis-framework)
[... rest of TOC ...]
```

### Step 6: Validate Changes

After making changes, verify:

1. **Memory system section is complete:**
   ```bash
   grep -A 50 "## Memory System Guide" CLAUDE.md
   ```

2. **Context file links are present:**
   ```bash
   grep "\.claude/context/" CLAUDE.md
   grep "\.claude/patterns/" CLAUDE.md
   grep "docs/adr/" CLAUDE.md
   ```

3. **Critical rules still present:**
   ```bash
   grep "CRITICAL RULE" CLAUDE.md
   grep "NEVER" CLAUDE.md | head -10
   ```

4. **File size reduced (should be ~1200-1500 lines):**
   ```bash
   wc -l CLAUDE.md
   # Before: ~2000+ lines
   # After: ~1200-1500 lines
   ```

## Before/After Comparison

### BEFORE (Current CLAUDE.md)

```markdown
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

[... continues with many more examples ...]
```

### AFTER (Optimized CLAUDE.md)

```markdown
### Kubernetes Client Patterns

**CRITICAL RULE:** Always use user-scoped clients for API operations.

**For detailed patterns and decision trees:** Load `.claude/patterns/k8s-client-usage.md`

**Quick reference:**
- User-scoped clients (`reqK8s`, `reqDyn`): For all user-initiated operations
- Service account clients (`K8sClient`, `DynamicClient`): ONLY for privileged operations after RBAC validation

**Basic pattern:**
```go
// Get user-scoped clients
reqK8s, reqDyn := GetK8sClientsForRequest(c)
if reqK8s == nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
    return
}
// Use for operations
list, err := reqDyn.Resource(gvr).Namespace(project).List(ctx, v1.ListOptions{})
```

**For complete patterns, anti-patterns, and examples:** See `.claude/patterns/k8s-client-usage.md`

**See also:** ADR-0002 (User Token Authentication) for the rationale behind this approach.
```

## Content Removal Guidelines

### Safe to Remove (Already in Memory Files)

After memory system is implemented, these can be removed from CLAUDE.md:

1. **Detailed code examples >20 lines**
   - Already in context files or pattern files
   - Keep only ~5-10 line snippets showing the pattern

2. **Step-by-step "how to" sections**
   - E.g., "Adding a New API Endpoint" with detailed steps
   - Keep the file references, remove the detailed steps

3. **Anti-patterns with explanations**
   - Move to pattern files with full examples
   - Keep only "NEVER do X" in CLAUDE.md

4. **Historical context about decisions**
   - E.g., "We chose X because Y and considered Z"
   - Move to ADRs with full context

5. **Common mistakes sections**
   - Move to pattern files
   - Keep only critical mistakes in CLAUDE.md

### MUST Keep in CLAUDE.md

1. **Universal rules with no exceptions**
   - NEVER change repo visibility
   - MANDATORY branch verification
   - ALWAYS run linters before push

2. **Critical security rules**
   - No panics in production
   - User token required for API operations
   - Token redaction in logs

3. **Build and deployment commands**
   - `make dev-start`
   - `make build-all`
   - Component-specific commands

4. **Project structure overview**
   - High-level architecture
   - Component relationships
   - Key directories

## Validation Checklist

After completing all steps:

- [ ] Memory System Guide section added after Table of Contents
- [ ] All context file links are correct and reference existing files
- [ ] Backend section slimmed down with links to context files
- [ ] Frontend section slimmed down with links to context files
- [ ] Critical rules still present and easy to find
- [ ] File size reduced by ~30-40% (from ~2000 to ~1200-1500 lines)
- [ ] No broken links (all referenced memory files exist)
- [ ] Table of Contents updated
- [ ] Test with Claude Code in new session:
  ```
  Claude, load the backend-development context and help me understand
  the K8s client usage patterns.
  ```

## Success Criteria

**This plan is complete when:**

1. ✅ Memory System Guide section added to CLAUDE.md
2. ✅ Backend/Frontend sections updated with context file links
3. ✅ Detailed examples removed (now in memory files)
4. ✅ CLAUDE.md is ~1200-1500 lines (down from ~2000+)
5. ✅ All critical rules still present and prominent
6. ✅ Claude Code can successfully reference memory files in new session
7. ✅ File validated with checklist above

## Rollback Plan

If optimization causes issues:

1. Revert CLAUDE.md: `git checkout HEAD -- CLAUDE.md`
2. Memory files are additive, so they don't need rollback
3. Re-run this plan with adjustments

## Next Steps After Implementation

1. **Test in practice:** Use memory file references for 1 week
2. **Gather feedback:** Are context files useful? Any missing patterns?
3. **Iterate:** Add new patterns as discovered
4. **Monthly review:** Update context files with new patterns

---

**End of Implementation Plan**

This plan is coldstartable - all changes are specified with exact content. No additional research or decisions needed during implementation.
