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
  * Fast HTTP response times (<10ms for simple operations)
  * Small container images (~20MB for Go binary)
  * Excellent K8s client-go integration
  * Strong typing prevents many bugs

* **Runner:**
  * Native Claude Code SDK support
  * Rich Python ecosystem for git/file operations
  * Easy to extend with custom agent behaviors
  * Rapid iteration on workflow logic

**Negative:**

* **Maintenance:**
  * Two language ecosystems to maintain
  * Different tooling (go vs. pip/uv)
  * Different testing frameworks

* **Development:**
  * Context switching between languages
  * Cannot share code between backend and runner
  * Different error handling patterns

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
* Framework: Gin (HTTP routing)
* K8s client: client-go + dynamic client
* Testing: table-driven tests with testify

**Runner (Python):**

```python
# Claude Code SDK integration
from claude_code import AgenticSession

session = AgenticSession(prompt=prompt, workspace=workspace)
result = session.run()
```

**Technology Stack:**
* SDK: claude-code-sdk (>=0.0.23)
* API client: anthropic (>=0.68.0)
* Git: GitPython
* Package manager: uv (preferred over pip)

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
