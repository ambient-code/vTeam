# OpenShift Deployment Guide

vTeam is a Kubernetes-native AI automation platform that combines Claude Code CLI with browser automation capabilities.

## Prerequisites

- OpenShift cluster with admin access
- Container registry access or use default images from quay.io/ambient_code
- `oc` CLI configured

## Quick Deploy

1. **Deploy** (from project root):
   ```bash
   make deploy
   ```
   This deploys to the `ambient-code` namespace using default images from quay.io/ambient_code.

2. **Verify deployment**:
   ```bash
   oc get pods -n ambient-code
   oc get services -n ambient-code
   ```

3. **Access the UI**:
   ```bash
   # Get the route URL
   oc get route frontend-route -n ambient-code

   # Or use port forwarding as fallback
   kubectl port-forward svc/frontend-service 3000:3000 -n ambient-code
   ```

## Configuration

### Customizing Namespace
To deploy to a different namespace:
```bash
make deploy NAMESPACE=my-namespace
```

### Building Custom Images
To build and use your own images:
```bash
# Set your container registry
export REGISTRY="your-registry.com"  # e.g., "quay.io/your-username"

# Login to your container registry
docker login $REGISTRY

# Build and push all images
make build-all REGISTRY=$REGISTRY
make push-all REGISTRY=$REGISTRY

# Deploy with custom images
make deploy CONTAINER_REGISTRY=$REGISTRY
```

### Advanced Configuration
Create and edit environment file for detailed customization:
```bash
cd components/manifests
cp env.example .env
# Edit .env to set CONTAINER_REGISTRY, IMAGE_TAG, Git settings, etc.
```

### Setting up API Keys
After deployment, create runner secrets through the project settings in the web UI. The system requires an Anthropic API key to function.

### OpenShift OAuth (Recommended)
For cluster login and authentication, see [OpenShift OAuth Setup](docs/OPENSHIFT_OAUTH.md).

## Cleanup

```bash
make clean
```
