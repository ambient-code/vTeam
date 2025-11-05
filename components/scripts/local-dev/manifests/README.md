# Local Dev Manifests

This directory contains OpenShift manifests for local development using Code Ready Containers (CRC).

## Directory Structure

### Symlinked Files (HOSTED Canonical Sources)

These files are **symlinks** to the canonical HOSTED manifests in `components/manifests/rbac/`:

**RBAC - Ambient Project Roles:**
- `ambient-project-admin-clusterrole.yaml` → `../../../manifests/rbac/`
- `ambient-project-edit-clusterrole.yaml` → `../../../manifests/rbac/`
- `ambient-project-view-clusterrole.yaml` → `../../../manifests/rbac/`

**RBAC - Backend:**
- `backend-sa.yaml` → `../../../manifests/rbac/`
- `backend-clusterrole.yaml` → `../../../manifests/rbac/`
- `backend-clusterrolebinding.yaml` → `../../../manifests/rbac/`

**RBAC - Operator:**
- `operator-sa.yaml` → `../../../manifests/rbac/`
- `operator-clusterrole.yaml` → `../../../manifests/rbac/`
- `operator-clusterrolebinding.yaml` → `../../../manifests/rbac/`

**Why symlinks?**
- Single source of truth: HOSTED manifests are canonical
- Automatic propagation: Updates to HOSTED automatically apply to local dev
- No duplication: Eliminates drift and maintenance burden
- Clear indication: Symlinks make it obvious which files are HOSTED vs LOCAL-ONLY

### LOCAL-ONLY Files

These files are specific to local development and remain as regular files:

1. **dev-users.yaml** - Test service accounts with admin/edit/view roles for local testing
2. **build-configs.yaml** - OpenShift BuildConfig resources for building images locally
3. **operator-build-config.yaml** - BuildConfig for the operator component
4. **backend-pvc.yaml** - PersistentVolumeClaim for backend state storage
5. **backend-service-alias.yaml** - ExternalName Service alias for operator to reach backend
6. **frontend-auth.yaml** - OAuth proxy configuration for frontend authentication
7. **backend-deployment.yaml** - Deployment using internal registry images
8. **frontend-deployment.yaml** - Deployment using internal registry images  
9. **operator-deployment.yaml** - Deployment using internal registry images

**Key Differences from HOSTED:**
- Use OpenShift internal registry: `image-registry.openshift-image-registry.svc:5000/vteam-dev/*`
- Include BuildConfigs for local image builds
- Include dev test users with tokens
- Include OAuth proxy for frontend
- Use local namespace: `vteam-dev` instead of `ambient-code`

## Syncing Manifests

To create/update symlinks:

```bash
cd components/scripts/local-dev
./sync-manifests.sh
```

This script will:
1. Remove duplicate RBAC files
2. Create symlinks to HOSTED canonical sources
3. Verify LOCAL-ONLY files are present
4. Report status

## Using in crc-start.sh

The `crc-start.sh` script handles namespace differences automatically:

```bash
# Apply symlinked RBAC with namespace patching
oc apply -f "${MANIFESTS_DIR}/backend-sa.yaml" -n "$PROJECT_NAME"
oc apply -f "${MANIFESTS_DIR}/backend-clusterrole.yaml"

# Patch namespace in ClusterRoleBinding (HOSTED uses 'ambient-code', LOCAL uses 'vteam-dev')
cat "${MANIFESTS_DIR}/backend-clusterrolebinding.yaml" | \
  sed "s/namespace: ambient-code/namespace: $PROJECT_NAME/" | \
  oc apply -f -
```

## Updating HOSTED Manifests

When you update HOSTED manifests in `components/manifests/rbac/`:

1. Changes automatically apply to local dev (via symlinks)
2. Re-run `make dev-start` or `crc-start.sh` to apply updates
3. No manual sync needed!

## Troubleshooting

**Symlinks broken?**
```bash
cd components/scripts/local-dev
./sync-manifests.sh
```

**Need to see canonical source?**
```bash
ls -la manifests/
# Symlinks show: filename -> ../../../manifests/rbac/filename
```

**Want to modify RBAC?**
- Edit files in `components/manifests/rbac/` (HOSTED canonical)
- Do NOT edit symlinks in `manifests/` - they point to HOSTED
- Changes propagate automatically via symlinks

## Directory Comparison

| File | HOSTED | LOCAL-DEV |
|------|--------|-----------|
| RBAC ClusterRoles/Bindings | ✅ Canonical | 🔗 Symlink |
| CRDs | ✅ Canonical | Referenced via `$CRDS_DIR` |
| Deployments | quay.io images | Internal registry |
| BuildConfigs | ❌ Not needed | ✅ LOCAL-ONLY |
| Dev Users | ❌ Not needed | ✅ LOCAL-ONLY |
| OAuth Proxy | ❌ Not needed | ✅ LOCAL-ONLY |

## Benefits

✅ **HOSTED = Single Source of Truth** - All RBAC defined once  
✅ **No Duplication** - Symlinks eliminate drift  
✅ **Automatic Updates** - Changes to HOSTED flow to local dev  
✅ **Clear Separation** - Symlinks vs regular files show HOSTED vs LOCAL  
✅ **Git Portable** - Symlinks work across the team  
✅ **Admin Support** - dev-users.yaml provides cluster-admin for testing  
✅ **Local Builds** - BuildConfigs enable rapid iteration  

