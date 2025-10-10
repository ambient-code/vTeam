# Getting Started with vTeam

This guide walks you through setting up vTeam for local development and deploying to OpenShift.

## Prerequisites

### Required Tools
- **oc CLI** - OpenShift command-line tool
- **kustomize** - Kubernetes manifest management
- **Docker or Podman** - Container engine (for building custom images)

### For Local Development
- **OpenShift Local (CRC)** - Local OpenShift cluster
  ```bash
  brew install crc
  ```
- **Red Hat Pull Secret** - Free from [console.redhat.com](https://console.redhat.com/openshift/create/local)

### Required API Keys
- **Anthropic API Key** - Get from [Anthropic Console](https://console.anthropic.com/)

## Local Development Setup

### 1. Install OpenShift Local

```bash
# Install CRC
brew install crc

# Initialize CRC (one-time setup)
crc setup

# Configure CRC with your pull secret
crc config set pull-secret-file /path/to/pull-secret.txt

# Start local OpenShift cluster
crc start
```

### 2. Clone and Configure

```bash
# Clone the repository
git clone https://github.com/your-org/vTeam.git
cd vTeam

# Prepare environment configuration
cd components/manifests
cp env.example .env

# Edit .env and set ANTHROPIC_API_KEY
vim .env
```

### 3. Deploy to Local Cluster

```bash
# From project root, deploy to local CRC cluster
make dev-start
```

This command will:
- Build container images locally
- Deploy to OpenShift Local
- Set up frontend, backend, operator, and CRDs
- Configure routes and services

### 4. Access the Application

```bash
# Get the frontend route
oc get route frontend-route -n ambient-code

# Or use port forwarding
oc port-forward svc/frontend-service 3000:3000 -n ambient-code
```

Access the UI at `http://localhost:3000` (or the route URL).

## Production Deployment to OpenShift

### 1. Login to OpenShift Cluster

```bash
# Login to your OpenShift cluster
oc login --server=https://api.your-cluster.com:6443 --token=<your-token>
```

### 2. Prepare Configuration

```bash
cd components/manifests
cp env.example .env

# Edit .env and configure:
# - ANTHROPIC_API_KEY (required)
# - GitHub App credentials (optional, for OAuth)
# - OAuth settings (optional, for SSO)
vim .env
```

### 3. Deploy with Pre-built Images

Deploy using pre-built images from `quay.io/ambient_code`:

```bash
# Deploy to default namespace (ambient-code)
./deploy.sh

# Or deploy to custom namespace
NAMESPACE=my-vteam ./deploy.sh
```

### 4. Verify Deployment

```bash
# Check pods
oc get pods -n ambient-code

# Expected pods:
# - frontend-*
# - backend-api-*
# - agentic-operator-*

# Check services and routes
oc get svc,route -n ambient-code
```

### 5. Configure Git Authentication

GitHub secrets for git operations must be created per project. See [components/manifests/GIT_AUTH_SETUP.md](../components/manifests/GIT_AUTH_SETUP.md) for details.

**Quick setup:**

```bash
# Create a project-level secret with git credentials
oc create secret generic my-runner-secret \
  --from-literal=ANTHROPIC_API_KEY="your-anthropic-key" \
  --from-literal=GIT_USER_NAME="Your Name" \
  --from-literal=GIT_USER_EMAIL="your.email@example.com" \
  --from-literal=GIT_TOKEN="ghp_your_github_token" \
  -n your-project-namespace

# Reference the secret in ProjectSettings via the UI:
# Settings → Runner Secrets → Select "my-runner-secret"
```

## Building Custom Images

If you want to build and push your own container images:

### 1. Set Container Registry

```bash
export REGISTRY="quay.io/your-username"

# Login to your registry
docker login $REGISTRY
```

### 2. Build All Components

```bash
# From project root
make build-all REGISTRY=$REGISTRY
```

This builds:
- `vteam_frontend:latest` - Next.js web UI
- `vteam_backend:latest` - Go API service
- `vteam_operator:latest` - Kubernetes operator
- `vteam_claude_runner:latest` - Python Claude Code runner

### 3. Push to Registry

```bash
make push-all REGISTRY=$REGISTRY
```

### 4. Deploy with Custom Images

```bash
cd components/manifests
CONTAINER_REGISTRY=$REGISTRY ./deploy.sh
```

## Post-Deployment Configuration

### 1. Create a Project

1. Access the web UI
2. Click **"New Project"**
3. Enter project name and display name
4. Click **"Create"**

This creates a new OpenShift namespace with RBAC configured.

### 2. Configure Runner Secrets

1. Navigate to your project
2. Go to **Settings → Runner Secrets**
3. Create or select a Kubernetes secret containing:
   - `ANTHROPIC_API_KEY` (required)
   - `GIT_TOKEN` (optional, for git operations)
   - Other environment variables as needed

### 3. Create an Agentic Session

1. Navigate to **Sessions → New Session**
2. Configure the session:
   - **Prompt**: Task description
   - **Model**: Select Claude model (e.g., claude-sonnet-4.5)
   - **Repos** (optional): Add git repositories to clone
3. Click **"Create Session"**
4. Monitor real-time execution via WebSocket stream

### 4. (Optional) Enable OpenShift OAuth

For cluster SSO authentication:

1. See [OPENSHIFT_OAUTH.md](OPENSHIFT_OAUTH.md) for setup instructions
2. Requires cluster-admin permissions to create OAuthClient
3. Provides seamless login with OpenShift credentials

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status and events
oc get pods -n ambient-code
oc describe pod <pod-name> -n ambient-code
oc logs <pod-name> -n ambient-code
```

### Session Jobs Failing

```bash
# Check job status
oc get jobs -n <project-namespace>

# View job logs
oc logs job/<session-job-name> -c ambient-code-runner -n <project-namespace>
```

### WebSocket Connection Issues

```bash
# Verify backend health
oc port-forward svc/backend-service 8080:8080 -n ambient-code
curl http://localhost:8080/health

# Check backend logs
oc logs deployment/backend-api -n ambient-code
```

### Authentication Issues

```bash
# Verify OAuth configuration (if enabled)
oc get oauthclient vteam-frontend

# Check route configuration
oc describe route frontend-route -n ambient-code
```

## Next Steps

- Review [Architecture Documentation](../README.md#architecture)
- Set up [GitHub App Integration](GITHUB_APP_SETUP.md) for OAuth and PR automation
- Configure [OpenShift OAuth](OPENSHIFT_OAUTH.md) for SSO
- Explore [RFE Workflows](../README.md#creating-an-rfe-workflow) for multi-agent feature development
- Check [User Guide](user-guide/index.md) for detailed usage instructions

## Cleanup

To remove vTeam from your cluster:

```bash
# From components/manifests directory
./deploy.sh uninstall

# Or from project root
make clean
```

This removes all resources from the namespace but keeps the namespace itself. To fully remove:

```bash
oc delete namespace ambient-code
```
