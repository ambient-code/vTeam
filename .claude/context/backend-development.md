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

```go
reqK8s, reqDyn := GetK8sClientsForRequest(c)
if reqK8s == nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
    c.Abort()
    return
}
```

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

```go
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
```

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
