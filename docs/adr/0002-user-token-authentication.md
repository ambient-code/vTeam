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
* Writing CRs after user authorization validated (handlers/sessions.go:417)
* Minting service account tokens for runner pods (handlers/sessions.go:449)
* Cross-namespace operations backend is explicitly authorized for

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
