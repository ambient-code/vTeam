
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

### Current State (2025-10-10)
- ✅ `handlers.go`: 441 lines (in progress - RFE, repo browsing, WebSocket, and middleware remaining)
- ✅ `main.go`: 375 lines (will eventually be ~20-30 lines following Go best practices)
- ✅ All Jira integration complete
- ✅ Types package created (`types/common.go`, `types/session.go`, `types/rfe.go`, `types/project.go`)
- ✅ Server package created (`server/server.go`, `server/k8s.go`)
- ✅ Health handler extracted (`handlers/health.go`)
- ✅ Content handlers extracted (`handlers/content.go`)
- ✅ GitHub auth handlers extracted (`handlers/github_auth.go`)
- ✅ Project handlers extracted (`handlers/projects.go`)
- ✅ Permissions handlers extracted (`handlers/permissions.go`)
- ✅ Secrets handlers extracted (`handlers/secrets.go`)
- ✅ Session handlers extracted (`handlers/sessions.go`)
- ✅ RFE workflow handlers extracted (`handlers/rfe.go`)
- ✅ Build clean

### Refactor Goals

**Goal 1: Simplify main.go (Standard Go Practice)**
- Current: 162 lines with routing, initialization, helpers
- Target: ~20-30 lines (just initialize and run)
- Move logic to appropriate packages:
  - `server/` - Server setup, routing, middleware
  - `handlers/` - All HTTP handlers and business logic
  - `types/` - Type definitions
- **IMPORTANT**: Most code in main.go will eventually move to new packages
- Keep `main.go` minimal (industry standard)
- **NO LOGIC CHANGES** - everything must continue working with existing frontend!

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

**Commit 3:** ✅ GitHub auth handlers
- ✅ Extract GitHub auth endpoints from `github_app.go`
- ✅ Create `handlers/github_auth.go` (441 lines, 4 handlers)
- ✅ Added wrapper function for backward compatibility
- ✅ Removed unused imports
- ✅ Build verified clean
- ✅ Commit: "refactor: extract GitHub auth handlers"

**Commit 4:** ✅ Project handlers
- ✅ Extract project CRUD endpoints (listProjects, createProject, getProject, deleteProject, updateProject)
- ✅ Create `handlers/projects.go` (384 lines, 5 handlers)
- ✅ Removed from `handlers.go` (lines 1910-2263 deleted)
- ✅ Build verified clean
- ✅ Commit: "refactor: extract project handlers"

**Commit 5:** ✅ Permissions handlers
- ✅ Extract RBAC + access key endpoints (listProjectPermissions, addProjectPermission, removeProjectPermission, listProjectKeys, createProjectKey, deleteProjectKey)
- ✅ Create `handlers/permissions.go` (430 lines, 6 handlers)
- ✅ Removed from `handlers.go` (lines 1910-2307 deleted)
- ✅ Build verified clean
- ✅ Commit: "refactor: extract permissions handlers"

**Commit 6:** ✅ Secrets handlers
- ✅ Extract runner secrets endpoints (listNamespaceSecrets, getRunnerSecretsConfig, updateRunnerSecretsConfig, listRunnerSecrets, updateRunnerSecrets)
- ✅ Create `handlers/secrets.go` (260 lines, 5 handlers)
- ✅ Create `handlers/helpers.go` (shared GetProjectSettingsResource function)
- ✅ Removed from `handlers.go` (lines 2629-2859 deleted)
- ✅ Build verified clean
- ✅ Commit: "refactor: extract secrets handlers"

**Commit 7:** ✅ Session handlers
- ✅ Extract agentic session endpoints (listSessions, createSession, getSession, updateSession, deleteSession, cloneSession, startSession, stopSession, updateSessionStatus, listSessionWorkspace, getSessionWorkspaceFile, putSessionWorkspaceFile, pushSessionRepo, abandonSessionRepo, diffSessionRepo, mintSessionGitHubToken)
- ✅ Create `handlers/sessions.go` (1,657 lines, 16 handlers)
- ✅ Helper functions: provisionRunnerTokenForSession, updateSessionDisplayName, setRepoStatus, parseSpec, stringPtr, contentListItem
- ✅ Updated `main.go` to initialize session handler dependencies and update routes
- ✅ Removed from `handlers.go` (lines 334-471, 475-1909 deleted - 1,575 lines total)
- ✅ Added shared helpers to `handlers.go` (stringPtr, contentListItem) for RFE handlers
- ✅ Build verified clean
- ✅ Commit: "refactor: extract session handlers"

**Commit 8:** ✅ RFE workflow handlers
- ✅ Extract RFE workflow endpoints (listProjectRFEWorkflows, createProjectRFEWorkflow, seedProjectRFEWorkflow, checkProjectRFEWorkflowSeeding, getProjectRFEWorkflow, getProjectRFEWorkflowSummary, deleteProjectRFEWorkflow, listProjectRFEWorkflowSessions, addProjectRFEWorkflowSession, removeProjectRFEWorkflowSession, getWorkflowJira)
- ✅ Create `handlers/rfe.go` (652 lines, 11 handlers)
- ✅ Kept rfeFromUnstructured and extractTitleFromContent in main package (handlers.go) for jira.go dependency
- ✅ Updated main.go to initialize RFE handler dependencies
- ✅ Removed from `handlers.go` (reduced from 1,058 to 441 lines)
- ✅ Build verified clean
- ✅ Commit: "refactor: extract RFE workflow handlers"

**Commit 9:** ✅ Repo browsing handlers
- ✅ Extract repo browsing endpoints (accessCheck, listUserForks, createUserFork, getRepoTree, getRepoBlob)
- ✅ Create `handlers/repo.go` (408 lines, 5 handlers)
- ✅ Extracted helper function parseOwnerRepo (githubAPIBaseURL and doGitHubRequest already in github_auth.go)
- ✅ Updated main.go to initialize repo handler dependencies and update routes
- ✅ Removed from `handlers.go` (accessCheck, lines 279-333 deleted - 55 lines)
- ✅ Removed from `github_app.go` (listUserForks, createUserFork, getRepoTree, getRepoBlob - 295 lines)
- ✅ Build verified clean
- ✅ Commit: "refactor: extract repo browsing handlers"

**Commit 10:** ✅ Middleware and helpers
- ✅ Extract middleware functions from handlers.go
- ✅ Create `handlers/middleware.go` (283 lines)
- ✅ Extracted middleware: ValidateProjectContext, GetK8sClientsForRequest, updateAccessKeyLastUsedAnnotation, ExtractServiceAccountFromAuth
- ✅ Extracted helpers: BoolPtr, StringPtr, ContentListItem type
- ✅ Kept rfeFromUnstructured and extractTitleFromContent in handlers.go (used by jira.go)
- ✅ Reduced handlers.go from 384 to 120 lines (only Jira-related functions remain)
- ✅ Updated main.go to initialize middleware dependencies and use handlers.ValidateProjectContext()
- ✅ Updated websocket_messaging.go to use handlers.ExtractServiceAccountFromAuth
- ✅ Updated projects.go to remove duplicate variable declarations
- ✅ Updated rfe.go and sessions.go to remove duplicate helper declarations
- ✅ Build verified clean
- ✅ Commit: "refactor: extract middleware to handlers/middleware.go"

**Phase 3: Organize remaining files into packages**

**Commit 11:** ✅ Git operations package
- ✅ Create `git/operations.go` (715 lines)
- ✅ Moved functions: PushRepo, AbandonRepo, DiffRepo, GetGitHubToken, CheckRepoSeeding, PerformRepoSeeding, ParseGitHubURL, ReadGitHubFile, InjectGitHubToken, DeriveRepoFolderFromURL
- ✅ Created wrapper functions in main.go for performRepoSeeding and checkRepoSeeding (RFEWorkflow adapter pattern)
- ✅ Updated handlers/content.go to use git.DiffSummary type
- ✅ Updated jira.go to use git.ParseGitHubURL, git.ReadGitHubFile, git.GetGitHubToken
- ✅ Updated main.go to initialize git package dependencies
- ✅ Removed git.go (all functions moved to git package)
- ✅ Build verified clean
- ✅ Commit: "refactor: create git package for git operations"

**Commit 12:** ✅ Jira integration package
- ✅ Created `jira/integration.go` (355 lines)
- ✅ Moved publishWorkflowFileToJira handler to jira package
- ✅ Moved RFEFromUnstructured and ExtractTitleFromContent to jira package (exported)
- ✅ Removed old jira.go file (258 lines)
- ✅ Removed old handlers.go file (120 lines - all code previously moved to packages)
- ✅ Updated main.go to initialize jira handler dependencies
- ✅ Updated route to use jiraHandler.PublishWorkflowFileToJira
- ✅ Jira package uses handlers.StringPtr from middleware.go
- ✅ Removed duplicate stringPtr from handlers/sessions.go (uses StringPtr from middleware)
- ✅ Build verified clean
- ✅ Commit: "refactor: create jira package for Jira integration"
- ✅ Commit: "refactor: remove duplicate stringPtr from handlers/sessions.go"

**Commit 13:** ✅ WebSocket package organization
- ✅ Consolidate websocket_messaging.go into `websocket/` package
- ✅ Create `websocket/hub.go` (219 lines) - WebSocket hub and connection management
- ✅ Create `websocket/handlers.go` (233 lines) - WebSocket HTTP handlers
- ✅ Updated main.go to initialize websocket package and update routes
- ✅ Removed websocket_messaging.go (405 lines)
- ✅ Build verified clean
- ✅ Commit: "refactor: create websocket package"

**Commit 14:** ✅ GitHub integration package
- ✅ Created `github/token.go` (258 lines) - Token management and caching (TokenManager)
- ✅ Created `github/app.go` (50 lines) - GitHub App integration (InitializeTokenManager, GetInstallation, MintSessionToken)
- ✅ Added GetInstallationID and GetHost methods to GitHubAppInstallation in handlers/github_auth.go
- ✅ Updated main.go to use github.Manager and github.GetInstallation
- ✅ Removed github_app.go (40 lines) and github_token.go (266 lines)
- ✅ Build verified clean
- ✅ Commit: "refactor: create github package"

**Commit 15:** Final cleanup
- Remove empty handlers.go if all code extracted
- Update all imports across packages
- Verify all tests pass
- Commit: "refactor: final cleanup after package reorganization"

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
