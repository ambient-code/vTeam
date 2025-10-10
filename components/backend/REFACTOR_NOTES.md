
## Code Refactoring

### ✅ Phase 1: Types Package (DONE)

**Created `types/` package:**
```
### Created:
- ✅ `types/common.go` - Common type definitions
- ✅ `types/session.go` - Session-related types
- ✅ `types/rfe.go` - RFE workflow types (with `ParentOutcome` field)
- ✅ `types/project.go` - Project types
- ✅ `jira.go` - Jira integration with GitHub file reading

**Updated `main.go`:**
- Import: `"ambient-code-backend/types"`
- Type aliases for backward compatibility:
  ```go
  type AgenticSession = types.AgenticSession
  type RFEWorkflow = types.RFEWorkflow
  // ... etc
  ```

**Verified:** ✅ Compiles successfully with zero logic changes

## Next Steps: Code Organization Refactor

### Current State (2025-01-10)
- ✅ `handlers.go`: 3614 lines (in progress - content handlers extracted)
- ✅ `main.go`: 162 lines (simplified with server package)
- ✅ All Jira integration complete
- ✅ Types package created (`types/common.go`, `types/session.go`, `types/rfe.go`, `types/project.go`)
- ✅ Server package created (`server/server.go`, `server/k8s.go`)
- ✅ Health handler extracted (`handlers/health.go`)
- ✅ Content handlers extracted (`handlers/content.go`)
- ✅ Build clean

### Refactor Goals

**Goal 1: Simplify main.go (Standard Go Practice)**
- Current: 497 lines with routing, initialization, helpers
- Target: ~20-30 lines (just initialize and run)
- Move logic to:
  - `server/server.go` - Server setup, routing, middleware
  - `server/k8s.go` - Kubernetes client initialization
  - Keep `main.go` minimal (industry standard)

**Goal 2: Break Down handlers.go**
- Current: 3835 lines (too large)
- Strategy: **One handler at a time, one commit each**
- Target structure:
  ```
  handlers/
  ├── middleware.go    - Auth, validation, K8s client helpers
  ├── sessions.go      - AgenticSession CRUD + lifecycle
  ├── projects.go      - Project CRUD
  ├── permissions.go   - RBAC + access keys
  ├── rfe.go           - RFEWorkflow CRUD + seeding
  ├── secrets.go       - Runner secrets management
  └── helpers.go       - Shared utility functions
  ```

### Incremental Refactor Strategy

**Phase 1: Simplify main.go** ✅ COMPLETE
1. ✅ Create `server/` package
2. ✅ Move server setup to `server/server.go`
3. ✅ Move K8s initialization to `server/k8s.go`
4. ✅ Reduce `main.go` to ~160 lines
5. ✅ Commit: "refactor: simplify main.go following Go best practices"

**Phase 2: Extract handlers.go (One Handler Per Commit)**

**Commit 1:** ✅ Health check handler
- ✅ Extract health endpoint
- ✅ Commit: "refactor: extract health handler"

**Commit 2:** ✅ Content service handlers
- ✅ Extract content service endpoints (ContentGitPush, ContentGitAbandon, ContentGitDiff, ContentWrite, ContentRead, ContentList)
- ✅ Create `handlers/content.go` (261 lines, 6 handlers)
- ✅ Removed from `handlers.go` (lines 1657-1876 deleted)
- ✅ Fixed unused imports and variables
- ✅ Build verified clean
- ✅ Commit: "refactor: extract content handlers to handlers/content.go"

**Commit 3:** GitHub auth handlers
- Extract GitHub auth endpoints
- Create `handlers/github_auth.go`
- Commit: "refactor: extract GitHub auth handlers"

**Commit 4:** Project handlers
- Extract project CRUD endpoints
- Create `handlers/projects.go`
- Commit: "refactor: extract project handlers"

**Commit 5:** Permissions handlers
- Extract RBAC + access key endpoints
- Create `handlers/permissions.go`
- Commit: "refactor: extract permissions handlers"

**Commit 6:** Secrets handlers
- Extract runner secrets endpoints
- Create `handlers/secrets.go`
- Commit: "refactor: extract secrets handlers"

**Commit 7:** Session handlers
- Extract agentic session endpoints
- Create `handlers/sessions.go`
- Commit: "refactor: extract session handlers"

**Commit 8:** RFE workflow handlers
- Extract RFE workflow endpoints
- Create `handlers/rfe.go`
- Commit: "refactor: extract RFE workflow handlers"

**Commit 9:** Repo browsing handlers
- Extract repo tree/blob endpoints
- Create `handlers/repo.go`
- Commit: "refactor: extract repo browsing handlers"

**Commit 10:** WebSocket handlers
- Extract WebSocket endpoints
- Create `handlers/websocket.go` (or move to websocket_messaging.go)
- Commit: "refactor: extract WebSocket handlers"

**Commit 11:** Middleware and helpers
- Extract middleware functions
- Extract shared helpers
- Create `handlers/middleware.go` and `handlers/helpers.go`
- Commit: "refactor: extract middleware and helpers"

**Commit 12:** Remove old handlers.go
- Delete empty handlers.go
- Update imports in all files
- Commit: "refactor: remove old handlers.go after migration complete"

### Benefits
- ✅ Small, reviewable commits (easier to debug if issues arise)
- ✅ Git history shows exactly what moved where
- ✅ Can test after each commit
- ✅ Easy to revert individual changes if needed
- ✅ Clear separation of concerns
- ✅ Better code navigation and maintenance
- ✅ Follows Go community best practices

### Verification After Each Commit
```bash
go build -v  # Must compile
go test ./...  # Run tests if available
git add -A
git commit -m "refactor: <specific change>"
```

### Notes
- Branch: `jira-backend-refactor`
- All upstream/main logic preserved
- Build verified clean after each change
