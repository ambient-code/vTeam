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
