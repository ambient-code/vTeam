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

- OWASP Top 10: <https://owasp.org/www-project-top-ten/>
- Kubernetes Security Best Practices: <https://kubernetes.io/docs/concepts/security/>
- RBAC Documentation: <https://kubernetes.io/docs/reference/access-authn-authz/rbac/>
