# OpenShift Deployment Guide

The Ambient Code Platform is an OpenShift-native platform that deploys a backend API, frontend, and operator into a managed namespace.

## Prerequisites

- OpenShift cluster with admin access
- Container registry access or use default images from quay.io/ambient_code
- `oc` CLI configured

## Quick Deploy

1. **Deploy** (from project root):
   ```bash
   # Prepare env once
   cp components/manifests/env.example components/manifests/.env
   # Edit .env and set ANTHROPIC_API_KEY
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
export REGISTRY="quay.io/your-username"

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
After deployment, configure runner secrets through Settings → Runner Secrets in the UI. At minimum, provide `ANTHROPIC_API_KEY`.

### OpenShift OAuth (Recommended)
For cluster login and authentication, see [OpenShift OAuth Setup](OPENSHIFT_OAUTH.md). The deploy script also supports a `secrets` subcommand if you only need to (re)configure OAuth secrets:

```bash
cd components/manifests
./deploy.sh secrets
```

## Cleanup

```bash
# Uninstall resources
make clean  # alias to ./components/manifests/deploy.sh clean
```
