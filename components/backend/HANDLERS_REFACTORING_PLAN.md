# Backend Handlers Refactoring Plan

## Current State Analysis
- **File**: `handlers.go`
- **Size**: 3,559 lines
- **Functions**: 50+ handler functions
- **Issue**: Single monolithic file that's difficult to maintain, navigate, and test

## Proposed File Structure

### 1. **Core Infrastructure** (`middleware.go`, `auth.go`, `types.go`)

#### `middleware.go` (~200 lines)
- `forwardedIdentityMiddleware()`
- `validateProjectContext()`
- Request logging/metrics middleware

#### `auth.go` (~300 lines)
- `getK8sClientsForRequest()`
- `updateAccessKeyLastUsedAnnotation()`
- `accessCheck()`
- `provisionRunnerTokenForSession()`
- Authentication helper functions

#### `types.go` (~400 lines)
Move all type definitions from `main.go` and `handlers.go`:
- `AgenticSession`, `AgenticSessionSpec`, `AgenticSessionStatus`
- `LLMSettings`, `GitConfig`, `GitUser`, `GitAuthentication`
- `RFEWorkflow`, `CreateRFEWorkflowRequest`
- `UserContext`, `BotAccountRef`, `ResourceOverrides`
- `AmbientProject`, `CreateProjectRequest`
- `PermissionAssignment`
- Request/response structs

### 2. **Domain-Specific Handlers**

#### `sessions.go` (~600 lines)
Agentic Session management:
- `listSessions()`
- `createSession()`
- `getSession()`
- `updateSession()`
- `updateSessionDisplayName()`
- `deleteSession()`
- `cloneSession()`
- `startSession()`
- `stopSession()`
- `updateSessionStatus()`
- `parseSpec()` helper

#### `workspaces.go` (~400 lines)
Workspace and content management:
- `getSessionMessages()`
- `postSessionMessage()`
- `getSessionWorkspace()`
- `getSessionWorkspaceFile()`
- `getRFEWorkflowWorkspace()`
- `getRFEWorkflowWorkspaceFile()`
- `resolveWorkspaceAbsPath()`
- `resolveWorkflowWorkspaceAbsPath()`

#### `content.go` (~300 lines)
Content service integration:
- `writeProjectContentFile()`
- `readProjectContentFile()`
- `listProjectContent()`
- `contentWrite()`
- `contentRead()`
- `contentList()`
- `proxyContentWrites()`

#### `projects.go` (~400 lines)
Project management:
- `listProjects()`
- `createProject()`
- `getProject()`
- `updateProject()`
- `deleteProject()`

#### `permissions.go` (~300 lines)
RBAC and permissions:
- `listProjectPermissions()`
- `addProjectPermission()`
- `removeProjectPermission()`
- `sanitizeName()` helper

#### `keys.go` (~200 lines)
Access key management:
- `listProjectKeys()`
- `createProjectKey()`
- `deleteProjectKey()`

#### `rfe_workflows.go` (~600 lines)
RFE workflow management:
- `listProjectRFEWorkflows()`
- `createProjectRFEWorkflow()`
- `getProjectRFEWorkflow()`
- `getProjectRFEWorkflowSummary()`
- `deleteProjectRFEWorkflow()`
- `listProjectRFEWorkflowSessions()`
- `addProjectRFEWorkflowSession()`
- `removeProjectRFEWorkflowSession()`
- `rfeFromUnstructured()` helper
- `initSpecKitInWorkspace()` helper

#### `secrets.go` (~300 lines)
Runner secrets management:
- `listNamespaceSecrets()`
- `getRunnerSecretsConfig()`
- `updateRunnerSecretsConfig()`
- `validateSourceSecret()`
- `triggerSecretSync()`
- `createSourceSecret()`

#### `git.go` (~200 lines)
Git configuration helpers:
- `loadGitConfigFromConfigMapForProject()`
- `mergeGitConfigs()`
- `stringPtr()` helper

#### `utils.go` (~100 lines)
Utility functions:
- `countArtifacts()`
- `getMetrics()`
- Common helper functions

### 3. **Route Registration** (`routes.go`)

#### `routes.go` (~200 lines)
Centralized route registration (extract from `main.go`):
```go
func RegisterRoutes(r *gin.Engine) {
    // Middleware setup
    r.Use(forwardedIdentityMiddleware())

    // Configure CORS
    setupCORS(r)

    // Content service mode routes
    registerContentRoutes(r)

    // API routes
    api := r.Group("/api")
    registerProjectRoutes(api)
    registerSessionRoutes(api)
    registerRFERoutes(api)
    registerSecretRoutes(api)

    // Health and metrics
    registerHealthRoutes(r)
}

func registerProjectRoutes(api *gin.RouterGroup) { ... }
func registerSessionRoutes(api *gin.RouterGroup) { ... }
// etc.
```

## Implementation Strategy

### Phase 1: Extract Types and Core Infrastructure
1. Create `types.go` - move all type definitions
2. Create `auth.go` - move authentication functions
3. Create `middleware.go` - move middleware functions
4. Update imports in existing files

### Phase 2: Extract Domain Handlers (5-6 files at a time)
1. `sessions.go` - most self-contained
2. `projects.go` - clear domain boundary
3. `secrets.go` - recently modified, good test case
4. `rfe_workflows.go` - complex but well-defined
5. `workspaces.go` and `content.go` - related functionality
6. `permissions.go`, `keys.go`, `git.go`, `utils.go` - smaller files

### Phase 3: Route Registration
1. Create `routes.go` - extract route setup from `main.go`
2. Organize routes by domain
3. Clean up `main.go`

### Phase 4: Package Organization (Future)
Consider organizing into sub-packages:
```
backend/
├── main.go
├── types.go
├── auth/
│   ├── middleware.go
│   ├── clients.go
│   └── rbac.go
├── handlers/
│   ├── sessions.go
│   ├── projects.go
│   ├── rfe_workflows.go
│   ├── secrets.go
│   ├── workspaces.go
│   └── permissions.go
├── content/
│   ├── service.go
│   └── proxy.go
└── routes.go
```

## Benefits

### Maintainability
- **Smaller files**: 200-600 lines per file vs 3,559
- **Clear separation**: Each file has single responsibility
- **Easier navigation**: Find session logic in `sessions.go`

### Testing
- **Unit testing**: Test domain handlers independently
- **Mock interfaces**: Easier to mock dependencies per domain
- **Focused tests**: Test session logic without RFE complexity

### Team Development
- **Reduced conflicts**: Multiple developers can work on different domains
- **Code review**: Smaller, focused PRs per domain
- **Expertise**: Team members can own specific domains

### Code Quality
- **Import optimization**: Only import what each domain needs
- **Dependency clarity**: Clear dependencies between domains
- **Interface definition**: Natural place for domain interfaces

## Implementation Considerations

### Import Management
- Each file will need appropriate Kubernetes client imports
- Shared utilities should be in `utils.go` or separate package
- Consider dependency injection for better testing

### Error Handling
- Consistent error handling patterns across files
- Shared error types in `types.go`
- Domain-specific error handling where appropriate

### Configuration
- Shared configuration constants
- Environment variable handling
- Feature flags per domain

### Backwards Compatibility
- No API changes required
- Same route structure
- Same request/response formats
- Gradual migration possible

## Estimated Effort
- **Phase 1**: 2-3 days (types, auth, middleware)
- **Phase 2**: 1-2 weeks (domain handlers)
- **Phase 3**: 1-2 days (routes)
- **Total**: ~2-3 weeks with testing and validation

## Success Metrics
- ✅ All existing tests pass
- ✅ No API behavior changes
- ✅ Improved build times (parallel compilation)
- ✅ Easier code navigation
- ✅ Reduced cognitive load for new contributors