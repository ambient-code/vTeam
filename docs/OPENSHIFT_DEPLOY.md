# OpenShift Deployment Guide

vTeam is an OpenShift-native platform that deploys a backend API, frontend, and operator into a managed namespace.

## Prerequisites

- **OpenShift cluster** with admin access
- **oc CLI** configured and authenticated
- **kustomize** installed
- Container registry access (or use default images from `quay.io/ambient_code`)

## Quick Deploy

### 1. Prepare Configuration

```bash
# From project root
cd components/manifests
cp env.example .env

# Edit .env and set required values:
# - ANTHROPIC_API_KEY (required for AI sessions)
# - Optionally: GitHub App credentials, OAuth settings
vim .env
```

### 2. Deploy

```bash
# Deploy to default namespace (ambient-code)
./deploy.sh

# Or deploy to custom namespace
NAMESPACE=my-vteam ./deploy.sh
```

This deploys using pre-built images from `quay.io/ambient_code`.

### 3. Verify Deployment

```bash
# Check pods
oc get pods -n ambient-code

# Check services and routes
oc get svc,route -n ambient-code
```

### 4. Access the UI

```bash
# Get the route URL
oc get route frontend-route -n ambient-code -o jsonpath='{.spec.host}'

# Or use port forwarding as fallback
oc port-forward svc/frontend-service 3000:3000 -n ambient-code
```

## Configuration

### Git Authentication

**Important:** GitHub secrets for git operations must be created separately per project. The deploy script does NOT create these automatically.

See [../components/manifests/GIT_AUTH_SETUP.md](../components/manifests/GIT_AUTH_SETUP.md) for detailed instructions.

**Quick example:**
```bash
oc create secret generic my-runner-secret \
  --from-literal=ANTHROPIC_API_KEY="your-anthropic-key" \
  --from-literal=GIT_TOKEN="ghp_your_github_token" \
  --from-literal=GIT_USER_NAME="Your Name" \
  --from-literal=GIT_USER_EMAIL="your.email@example.com" \
  -n your-project-namespace
```

### Building Custom Images

To build and use your own images:

```bash
# Set your container registry
export REGISTRY="quay.io/your-username"
docker login $REGISTRY

# Build and push all images
make build-all REGISTRY=$REGISTRY
make push-all REGISTRY=$REGISTRY

# Deploy with custom images
cd components/manifests
CONTAINER_REGISTRY=$REGISTRY ./deploy.sh
```

### Deploying to Custom Namespace

```bash
# Deploy to specific namespace
NAMESPACE=my-vteam ./deploy.sh

# Or from project root
make deploy NAMESPACE=my-vteam
```

### OpenShift OAuth (Recommended)

For cluster SSO authentication, see [OPENSHIFT_OAUTH.md](OPENSHIFT_OAUTH.md).

The deploy script also supports a `secrets` subcommand to (re)configure OAuth without full redeployment:

```bash
cd components/manifests
./deploy.sh secrets
```

## Deployment Script Options

The `deploy.sh` script supports several commands:

```bash
# Standard deployment
./deploy.sh

# Deploy to custom namespace
NAMESPACE=my-namespace ./deploy.sh

# Configure OAuth secrets only (no redeployment)
./deploy.sh secrets

# Uninstall/cleanup
./deploy.sh uninstall
# or
./deploy.sh clean
```

## Post-Deployment Setup

### 1. Configure Runner Secrets

Access the UI and navigate to **Settings → Runner Secrets** to configure API keys per project.

Required: `ANTHROPIC_API_KEY`

### 2. Create Projects

Create project namespaces via the UI (**New Project** button). Each project gets isolated RBAC and resources.

### 3. Set Up Git Authentication

Create Kubernetes secrets in each project namespace with git credentials. See [GIT_AUTH_SETUP.md](../components/manifests/GIT_AUTH_SETUP.md).

## Cleanup

```bash
# Uninstall from default namespace
cd components/manifests
./deploy.sh uninstall

# Or from project root
make clean

# Uninstall from custom namespace
NAMESPACE=my-vteam ./deploy.sh uninstall
```

This removes all resources but keeps the namespace. To fully remove:

```bash
oc delete namespace ambient-code
```
