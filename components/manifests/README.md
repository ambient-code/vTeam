# vTeam Manifests - Kustomize Overlays

This directory contains Kubernetes/OpenShift manifests organized using **Kustomize overlays** to eliminate duplication across environments.

## Directory Structure

```
manifests/
├── base/                          # Common resources shared across all environments
│   ├── backend-deployment.yaml
│   ├── frontend-deployment.yaml
│   ├── operator-deployment.yaml
│   ├── workspace-pvc.yaml
│   ├── namespace.yaml
│   ├── crds/                      # Custom Resource Definitions
│   └── rbac/                      # Role-Based Access Control
│
├── overlays/                      # Environment-specific configurations
│   ├── production/                # OpenShift production environment
│   │   ├── kustomization.yaml
│   │   ├── route.yaml
│   │   ├── backend-route.yaml
│   │   ├── frontend-oauth-*.yaml  # OAuth proxy patches
│   │   ├── github-app-secret.yaml
│   │   └── namespace-patch.yaml
│   │
│   ├── e2e/                       # Kind/K8s testing environment
│   │   ├── kustomization.yaml
│   │   ├── *-ingress.yaml        # K8s Ingress resources
│   │   ├── test-user.yaml        # Test user with cluster-admin
│   │   ├── secrets.yaml
│   │   └── *-patch.yaml          # Environment-specific patches
│   │
│   └── local-dev/                 # CRC local development environment
│       ├── kustomization.yaml
│       ├── build-configs.yaml    # OpenShift BuildConfigs
│       ├── dev-users.yaml        # Local development users
│       ├── frontend-auth.yaml
│       ├── *-route.yaml
│       └── *-patch.yaml          # Local dev patches
│
├── deploy.sh                      # Production deployment script
├── env.example                    # Example environment variables
└── README.md                      # This file
```

## Environment Differences

### Production (OpenShift)
- **Registry**: `quay.io/ambient_code/*`
- **Networking**: OpenShift Routes
- **Auth**: OAuth proxy sidecar in frontend
- **Storage**: Cluster default storage class
- **Namespace**: `ambient-code` with OpenShift monitoring

**Deploy**:
```bash
cd components/manifests
./deploy.sh
```

### E2E Testing (Kind/K8s)
- **Registry**: `quay.io/ambient_code/*`
- **Networking**: K8s Ingress (nginx)
- **Auth**: Test user with cluster-admin
- **Storage**: `standard` storage class
- **Namespace**: `ambient-code`

**Deploy**:
```bash
cd e2e
./scripts/setup-kind.sh
./scripts/deploy.sh
./scripts/run-tests.sh
```

### Local Dev (CRC/OpenShift Local)
- **Registry**: Internal OpenShift registry (`image-registry.openshift-image-registry.svc:5000/vteam-dev/*`)
- **Networking**: OpenShift Routes
- **Auth**: Frontend auth token for local user
- **Storage**: `crc-csi-hostpath-provisioner`
- **Namespace**: `vteam-dev`
- **Build**: Uses BuildConfigs for local image builds

**Deploy**:
```bash
make dev-start
```

## How It Works

### Base Resources
The `base/` directory contains common manifests shared across all environments:
- Deployments (without environment-specific configuration)
- Services
- PVCs (without storageClassName)
- CRDs
- Common RBAC

### Overlays
Each overlay in `overlays/` extends the base with environment-specific:
- **Resources**: Additional manifests (Routes, Ingress, Secrets, etc.)
- **Patches**: Strategic merge or JSON patches to modify base resources
- **Images**: Override image names/tags
- **Namespace**: Set target namespace

### Example: Adding OAuth to Frontend

**Base** (`base/frontend-deployment.yaml`):
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
spec:
  template:
    spec:
      containers:
      - name: frontend
        image: quay.io/ambient_code/vteam_frontend:latest
```

**Production Patch** (`overlays/production/frontend-oauth-deployment-patch.yaml`):
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
spec:
  template:
    spec:
      containers:
      - name: oauth-proxy  # Add OAuth sidecar
        image: quay.io/openshift/origin-oauth-proxy:4.14
        # ... OAuth configuration
```

The patch is applied via the kustomization.yaml:
```yaml
patches:
- path: frontend-oauth-deployment-patch.yaml
  target:
    kind: Deployment
    name: frontend
```

## Building Manifests

### Test a build without applying:
```bash
# Production
kustomize build overlays/production/

# E2E
kustomize build overlays/e2e/

# Local dev
kustomize build overlays/local-dev/
```

### Apply directly with kubectl/oc:
```bash
kubectl apply -k overlays/production/
# or
oc apply -k overlays/production/
```

## Customizing Deployments

### Change Namespace
```bash
cd overlays/production
kustomize edit set namespace my-namespace
kustomize build . | oc apply -f -
# Restore
kustomize edit set namespace ambient-code
```

### Change Images
```bash
cd overlays/production
kustomize edit set image quay.io/ambient_code/vteam_backend:latest=my-registry/backend:v1.0
kustomize build . | oc apply -f -
```

### Environment Variables
Set via `.env` file or environment variables before running `deploy.sh`:
```bash
NAMESPACE=my-namespace IMAGE_TAG=v1.0 ./deploy.sh
```

## Benefits of This Structure

✅ **Single Source of Truth**: Base manifests define common configuration  
✅ **No Duplication**: Environment-specific configs only define differences  
✅ **Easy to Maintain**: Changes to base apply to all environments  
✅ **Clear Differences**: Overlays show exactly what's unique per environment  
✅ **Type-Safe**: Kustomize validates patches against base resources  

## Migration Notes

This structure replaces the previous duplicated manifests:
- ~~`components/manifests/*.yaml`~~ → `base/` + `overlays/production/`
- ~~`e2e/manifests/*.yaml`~~ → `overlays/e2e/`
- ~~`components/scripts/local-dev/manifests/*.yaml`~~ → `overlays/local-dev/`

Old manifest directories have been preserved for reference but are no longer used by deployment scripts.

## Troubleshooting

### Kustomize build fails
```bash
# Validate the kustomization.yaml
kustomize build overlays/production/ --enable-alpha-plugins

# Check for duplicate resources
kustomize build overlays/production/ 2>&1 | grep -i "conflict"
```

### Images not updating
```bash
# Make sure you're in the overlay directory
cd overlays/production
kustomize edit set image quay.io/ambient_code/vteam_backend:latest=...
```

### Namespace issues
```bash
# Check current namespace in kustomization
grep "namespace:" overlays/production/kustomization.yaml

# Verify resources are in correct namespace after build
kustomize build overlays/production/ | grep "namespace:"
```

## Additional Resources

- [Kustomize Documentation](https://kustomize.io/)
- [OpenShift Kustomize Guide](https://docs.openshift.com/container-platform/latest/applications/working_with_quotas.html)
- [Kubernetes Kustomize Tutorial](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/)

