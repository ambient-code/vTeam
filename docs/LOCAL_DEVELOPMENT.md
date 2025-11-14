# Local Development Guide

This guide explains how to set up and use the minikube-based local development environment for the Ambient Code Platform.

> **⚠️ SECURITY WARNING - LOCAL DEVELOPMENT ONLY**
>
> This setup is **ONLY for local development** and is **COMPLETELY INSECURE** for production use:
> - ❌ Authentication is disabled
> - ❌ Mock tokens are accepted without validation
> - ❌ Backend uses cluster-admin service account (full cluster access)
> - ❌ All RBAC restrictions are bypassed
> - ❌ No multi-tenant isolation
>
> **NEVER use this configuration in production, staging, or any shared environment.**
>
> For production deployments, see the main [README.md](../README.md) and ensure proper OpenShift OAuth, RBAC, and namespace isolation are configured.

## Complete Feature List

✅ **Authentication Disabled** - No login required  
✅ **Automatic Mock User** - Login automatically as "developer"  
✅ **Full Project Management** - Create, view, and manage projects  
✅ **Service Account Permissions** - Backend uses Kubernetes service account in dev mode  
✅ **Ingress Routing** - Access via hostname or NodePort  
✅ **All Components Running** - Frontend, backend, and operator fully functional

## Prerequisites

- Podman
- Minikube
- kubectl

### Installation

```bash
# macOS
brew install podman minikube kubectl

# Linux - Podman
sudo apt-get install podman  # Debian/Ubuntu
# OR
sudo dnf install podman      # Fedora/RHEL

# Linux - Minikube
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
```

## Quick Start

```bash
# Start local environment
make local-up

```

## Access URLs

Access the application using NodePort:

```bash
# Get minikube IP
minikube ip

# Access URLs (replace IP with output from above)
# Frontend: http://192.168.64.4:30030
# Backend: http://192.168.64.4:30080/health
```

Or use the Makefile command:
```bash
make local-url
```

## Authentication

> **⚠️ INSECURE - LOCAL ONLY**
>
> Authentication is **completely disabled** for local development. This setup has NO security and should **NEVER** be used outside of isolated local environments.

Authentication is **completely disabled** for local development:

- ✅ No OpenShift OAuth required
- ✅ Automatic login as "developer"
- ✅ Full access to all features
- ✅ Backend uses service account for Kubernetes API

### How It Works

1. **Frontend**: Sets `DISABLE_AUTH=true` environment variable
2. **Auth Handler**: Automatically provides mock credentials:
   - User: developer
   - Email: developer@localhost
   - Token: mock-token-for-local-dev

3. **Backend**: Detects mock token and uses service account credentials

> **Security Note**: The mock token `mock-token-for-local-dev` is hardcoded and provides full cluster access. This is acceptable ONLY in isolated local minikube clusters. Production environments use real OAuth tokens with proper RBAC enforcement.

## Features Tested

### ✅ Projects
- View project list
- Create new projects
- Access project details

### ✅ Backend API
- Health endpoint working
- Projects API returning data
- Service account permissions working

### ✅ Ingress
- Frontend routing works
- Backend API routing works  
- Load balancer configured

## Common Commands

```bash
# View status
make local-status

# View logs
make local-logs              # Backend
make local-logs-frontend     # Frontend
make local-logs-operator     # Operator

# Restart components
make local-restart           # All
make local-restart-backend   # Backend only

# Stop/delete
make local-stop              # Stop deployment
make local-delete            # Delete minikube cluster
```

## Development Workflow

1. Make code changes
2. Rebuild images:
   ```bash
   # Build with Podman (default)
   podman build -t vteam-backend:latest components/backend
   
   # Load into minikube
   minikube image load vteam-backend:latest
   ```
3. Restart deployment:
   ```bash
   make local-restart-backend
   ```

**Note:** Images are built locally with Podman and then loaded into minikube using `minikube image load`. This approach works with any container runtime configuration in minikube.

## Troubleshooting

### Projects Not Showing
- Backend requires cluster-admin permissions
- Added via: `kubectl create clusterrolebinding backend-admin --clusterrole=cluster-admin --serviceaccount=ambient-code:backend-api`

### Frontend Auth Errors
- Frontend needs `DISABLE_AUTH=true` environment variable
- Backend middleware checks for mock token

### Ingress Not Working
- Wait for ingress controller to be ready
- Check: `kubectl get pods -n ingress-nginx`

## Technical Details

### Authentication Flow

> **⚠️ INSECURE FLOW - DO NOT USE IN PRODUCTION**

1. Frontend sends request with `X-Forwarded-Access-Token: mock-token-for-local-dev`
2. Backend middleware checks: `if token == "mock-token-for-local-dev"`
3. Backend uses `server.K8sClient` and `server.DynamicClient` (service account)
4. No RBAC restrictions - full cluster access

**Why this is insecure:**
- Mock token is a known, hardcoded value that anyone can use
- Backend bypasses all RBAC checks when this token is detected
- Service account has cluster-admin permissions (unrestricted access)
- No user identity verification or authorization

### Environment Variables
- `DISABLE_AUTH=true` (Frontend & Backend) - **NEVER set in production**
- `MOCK_USER=developer` (Frontend) - **Local development only**
- `ENVIRONMENT=local` or `development` - Required for dev mode to activate

### RBAC

> **⚠️ DANGEROUS - FULL CLUSTER ACCESS**

- Backend service account has **cluster-admin** role
- All namespaces accessible (no isolation)
- Full Kubernetes API access (read/write/delete everything)
- **This would be a critical security vulnerability in production**

**Production RBAC:**
In production, the backend service account has minimal permissions, and user tokens determine access via namespace-scoped RBAC policies.

## Production Differences

> **Critical Security Differences**
>
> The local development setup intentionally disables all security measures for convenience. Production environments have multiple layers of security that are completely absent in local dev.

| Feature | Minikube (Dev) ⚠️ INSECURE | OpenShift (Prod) ✅ SECURE |
|---------|---------------------------|---------------------------|
| **Authentication** | Disabled, mock user accepted | OpenShift OAuth with real identity |
| **User Tokens** | Hardcoded mock token | Cryptographically signed OAuth tokens |
| **Kubernetes Access** | Service account (cluster-admin) | User token with namespace-scoped RBAC |
| **Namespace Visibility** | All namespaces (unrestricted) | Only authorized namespaces |
| **Authorization** | None - full access for all | RBAC enforced on every request |
| **Token Validation** | Mock token bypasses validation | Token signature verified, expiration checked |
| **Service Account** | Cluster-admin permissions | Minimal permissions (no user impersonation) |
| **Multi-tenancy** | No isolation | Full namespace isolation |
| **Audit Trail** | Mock user only | Real user identity in audit logs |

**Why local dev is insecure:**
1. **No identity verification**: Anyone can use the mock token
2. **No authorization**: RBAC is completely bypassed
3. **Unrestricted access**: Cluster-admin can do anything
4. **No audit trail**: All actions appear as "developer"
5. **No token expiration**: Mock token never expires
6. **No namespace isolation**: Can access all projects/namespaces

## Changes Made for Local Development

> **⚠️ SECURITY WARNING**
>
> These code changes disable authentication and should **ONLY** activate in verified local development environments. Production deployments must never enable these code paths.

### Backend (`components/backend/handlers/middleware.go`)

```go
// In dev mode, use service account credentials for mock tokens
// WARNING: This bypasses all RBAC and provides cluster-admin access
// Only activates when:
// 1. ENVIRONMENT=local or development
// 2. DISABLE_AUTH=true
// 3. Namespace does not contain 'prod'
if token == "mock-token-for-local-dev" || os.Getenv("DISABLE_AUTH") == "true" {
    log.Printf("Dev mode detected - using service account credentials for %s", c.FullPath())
    return server.K8sClient, server.DynamicClient
}
```

**Safety Mechanisms:**
- Requires `ENVIRONMENT=local` or `development` (line 297-299 in middleware.go)
- Requires `DISABLE_AUTH=true` explicitly set (line 303-305)
- Rejects if namespace contains "prod" (line 314-317)
- Logs activation for audit trail (line 319)

### Frontend (`components/frontend/src/lib/auth.ts`)

```typescript
// If auth is disabled, provide mock credentials
// WARNING: This provides a hardcoded token that grants full cluster access
// Only use in isolated local development environments
if (process.env.DISABLE_AUTH === 'true') {
  const mockUser = process.env.MOCK_USER || 'developer';
  headers['X-Forwarded-User'] = mockUser;
  headers['X-Forwarded-Preferred-Username'] = mockUser;
  headers['X-Forwarded-Email'] = `${mockUser}@localhost`;
  headers['X-Forwarded-Access-Token'] = 'mock-token-for-local-dev';
  return headers;
}
```

**Security Note:** These changes create a "dev mode" backdoor. While protected by environment checks, this code should be reviewed carefully during security audits.

## Success Criteria

✅ All components running  
✅ Projects create and list successfully  
✅ No authentication required  
✅ Full application functionality available  
✅ Development workflow simple and fast

## Security Checklist

Before using this setup, verify:

- [ ] Running on **isolated local machine only** (not a shared server)
- [ ] Minikube cluster is **not accessible from network**
- [ ] `ENVIRONMENT=local` or `development` is set
- [ ] You understand this setup has **NO security**
- [ ] You will **NEVER deploy this to production**
- [ ] You will **NOT set `DISABLE_AUTH=true`** in production
- [ ] You will **NOT use mock tokens** in production

## Transitioning to Production

When deploying to production:

1. **Remove Development Settings:**
   - Remove `DISABLE_AUTH=true` environment variable
   - Remove `ENVIRONMENT=local` or `development` settings
   - Remove `MOCK_USER` environment variable

2. **Enable Production Security:**
   - Configure OpenShift OAuth (see main README)
   - Set up namespace-scoped RBAC policies
   - Use minimal service account permissions (not cluster-admin)
   - Enable network policies for component isolation
   - Configure proper TLS certificates

3. **Verify Security:**
   - Test with real user tokens
   - Verify RBAC restrictions work
   - Ensure mock token is rejected
   - Check audit logs show real user identities
   - Validate namespace isolation

**Never assume local dev configuration is production-ready.**

