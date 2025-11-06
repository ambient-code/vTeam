# OAuth Scope Restriction - Deployment Guide

## Overview

This change restricts the OAuth scope from `user:full` to `user:info` to prevent users from accessing the OpenShift console and kubectl/oc CLI while still allowing authentication to the vTeam application.

## What Changed

**File: `components/manifests/frontend-deployment.yaml`**
- **Before:** `--scope=user:full` (granted full cluster access)
- **After:** `--scope=user:info` (authentication only)

## Impact

### Users CAN:
✅ Log in to vTeam application via Google OAuth
✅ Access all vTeam features
✅ View their username and basic profile information

### Users CANNOT:
❌ Access OpenShift web console
❌ Use kubectl or oc CLI commands
❌ Access any cluster resources outside vTeam

## Deployment Instructions

### Option 1: Quick Deployment (Recommended)

```bash
# Apply the updated frontend deployment
oc apply -f components/manifests/frontend-deployment.yaml

# Restart the frontend to pick up changes
oc rollout restart deployment/frontend -n ambient-code
```

### Option 2: Full Redeploy

```bash
cd components/manifests
./deploy.sh
```

### Verification

1. **Check OAuth proxy configuration:**
   ```bash
   oc get deployment frontend -n ambient-code -o jsonpath='{.spec.template.spec.containers[?(@.name=="oauth-proxy")].args}' | grep scope
   ```
   Should show: `--scope=user:info`

2. **Test user authentication:**
   - User should be able to log in to vTeam
   - User should see their username displayed

3. **Verify console access is blocked:**
   - User attempts to access OpenShift console
   - User should be denied access (not authorized)

4. **Verify kubectl access is blocked:**
   ```bash
   oc login --token=<user-oauth-token> --server=<cluster-api>
   oc get pods
   ```
   Should return: `Error from server (Forbidden): pods is forbidden`

## Understanding OAuth Scopes

| Scope | Description | Console Access | kubectl Access |
|-------|-------------|----------------|----------------|
| `user:info` | Basic user info only | ❌ No | ❌ No |
| `user:check-access` | Check RBAC permissions | ❌ No | ❌ No |
| `user:list-projects` | List projects only | ⚠️ Limited | ⚠️ Limited |
| `user:full` | Full cluster access | ✅ Yes | ✅ Yes |

## Rollback Instructions

If you need to restore full cluster access:

```bash
# Edit the deployment
oc edit deployment frontend -n ambient-code

# Find the oauth-proxy container args section and change:
# --scope=user:info
# to:
# --scope=user:full

# Save and exit - deployment will auto-rollout
```

Or via patch:

```bash
oc patch deployment frontend -n ambient-code --type=json -p='[
  {
    "op": "replace",
    "path": "/spec/template/spec/containers/1/args/5",
    "value": "--scope=user:full"
  }
]'
```

## Technical Details

The `user:info` scope grants access to the following OpenShift API endpoints:
- `/apis/user.openshift.io/v1/users/~` - Get current user info
- Basic authentication validation

It does NOT grant:
- `/api/v1/namespaces` - List/access namespaces
- `/api/v1/pods` - Access to pods
- Any cluster-admin or elevated permissions
- Token exchange for console/CLI access

## Troubleshooting

**Issue:** Users cannot log in after deployment
**Solution:** Check that OAuthClient is still configured correctly:
```bash
oc get oauthclient ambient-frontend -o yaml
```

**Issue:** Backend API calls fail with 403
**Solution:** The `--openshift-delegate-urls` setting handles backend API authorization separately from user cluster access. Verify the backend service account has proper RBAC permissions.

**Issue:** Need to grant specific users cluster access
**Solution:** Add users to OpenShift RBAC groups separately from vTeam authentication:
```bash
oc adm policy add-cluster-role-to-user cluster-admin <username>
```

## References

- OpenShift OAuth Proxy Documentation: https://github.com/openshift/oauth-proxy
- OpenShift OAuth Scopes: https://docs.openshift.com/container-platform/4.14/authentication/tokens-scoping.html
- vTeam OAuth Setup: `docs/OPENSHIFT_OAUTH.md`
