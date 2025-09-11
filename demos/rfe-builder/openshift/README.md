# OpenShift Deployment

This directory contains OpenShift/Kubernetes manifests and deployment scripts for the RHOAI AI Feature Sizing Platform.

## 🔨 Building the Image

```bash
# Build with podman/docker (auto-detected)
./build.sh

# Build with custom image name
IMAGE_FULL_NAME=quay.io/my/image:latest ./build.sh

# Build with custom registry and tag
./build.sh quay.io/myregistry/myapp v1.0.0
```

## 🚀 Deploy Script Usage

```bash
# Deploy with defaults
./deploy.sh

# Deploy to custom namespace
NAMESPACE=my-namespace ./deploy.sh

# Deploy custom image to custom namespace
NAMESPACE=my-namespace IMAGE_FULL_NAME=quay.io/my/image:latest ./deploy.sh

# Deploy with custom registry/tag
./deploy.sh quay.io/myregistry/myapp v1.0.0
```

## 📦 Kustomize Usage

```bash
# Deploy with kustomize (default namespace)
kustomize build . | oc apply -f -

# Deploy to custom namespace
kustomize edit set namespace my-namespace
kustomize build . | oc apply -f -

# Deploy custom image
kustomize edit set image quay.io/gkrumbach07/llama-index-demo/rhoai-ai-feature-sizing=quay.io/my/image:latest
kustomize build . | oc apply -f -
```

**Note**: `kustomize edit set namespace` automatically overrides the namespace in all resources at build time, including the namespace.yaml file. You don't need to manually edit individual manifest files.

## 📄 Individual Manifests

```bash
# 1. Create namespace
# For default namespace:
oc apply -f namespace.yaml

# For custom namespace:
oc create namespace my-namespace

# 2. Switch to namespace
oc project my-namespace

# 3. Apply manifests directly
oc apply -f pvc.yaml
oc apply -f configmap.yaml
oc apply -f secret.yaml
oc apply -f deployment.yaml
oc apply -f service.yaml
oc apply -f route.yaml

# 4. Update deployment image if needed
oc set image deployment/rhoai-ai-feature-sizing rhoai-app=quay.io/my/image:latest
```

## 🔧 Environment Variables

- **NAMESPACE**: Target namespace (default: `rhoai-ai-feature-sizing`)
- **IMAGE_FULL_NAME**: Complete image name with registry and tag

## 📋 Manifest Files

- `namespace.yaml` - Namespace definition (contains `${NAMESPACE}` placeholder)
- `pvc.yaml` - Persistent volume claims for uploads and output
- `configmap.yaml` - Application configuration
- `secret.yaml` - API keys and secrets (update after deployment)
- `deployment.yaml` - Main application deployment
- `service.yaml` - ClusterIP services for API and UI
- `route.yaml` - OpenShift routes for external access
- `kustomization.yaml` - Kustomize configuration

## 🔑 Post-Deployment

Update API keys in the secret:
```bash
oc patch secret rhoai-secrets -p '{"stringData":{"OPENAI_API_KEY":"your-actual-key"}}'
```