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
