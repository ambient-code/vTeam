# Ambient Code Platform Components

This directory contains the core components of the Ambient Code Platform.

See the main [README.md](../README.md) for complete documentation, deployment instructions, and usage examples.

## Component Directory Structure

```
components/
├── frontend/                   # NextJS web interface with Shadcn UI
├── backend/                    # Go API service for Kubernetes CRD management
├── operator/                   # Kubernetes operator (Go)
├── runners/                    # AI runner services
│   └── claude-code-runner/     # Python service running Claude Code CLI with MCP
├── manifests/                  # Kubernetes deployment manifests
└── README.md                   # This documentation
```

## 🎯 Agentic Session Flow

1. **Create Session**: User creates a new agentic session via the web UI
2. **API Processing**: Backend creates an `AgenticSession` Custom Resource in Kubernetes
3. **Job Scheduling**: Operator detects the CR and creates a Kubernetes Job
4. **Execution**: Job runs a pod with AI CLI and Playwright MCP server
5. **Task Execution**: AI executes the specified task using MCP capabilities
6. **Result Storage**: Results are stored back in the Custom Resource
7. **UI Update**: Frontend displays the completed agentic session with results

## ⚡ Quick Start

### Local Development (Recommended)
```bash
# Single command to start everything
make dev-start
```

**Prerequisites:**
- Minikube: `brew install minikube`
- Red Hat pull secret: Get free from [console.redhat.com](https://console.redhat.com/openshift/create/local)

**What you get:**
- ✅ Complete OpenShift development environment
- ✅ Frontend: `https://vteam-frontend-vteam-dev.apps-crc.testing`
- ✅ Backend API working with authentication
- ✅ OpenShift console access
- ✅ Ready for project creation and agentic sessions

### Production Deployment
```bash
# Build and push images to your registry
export REGISTRY="your-registry.com"
make build-all push-all REGISTRY=$REGISTRY

# Deploy to OpenShift/Kubernetes
cd components/manifests
CONTAINER_REGISTRY=$REGISTRY ./deploy.sh
```

### Hot Reloading Development
```bash
# Terminal 1: Start with development mode
DEV_MODE=true make dev-start

# Terminal 2: Enable file sync for hot-reloading
make dev-sync
```

## Quick Deploy

From the project root:

```bash
# Deploy with default images
make deploy

# Or deploy to custom namespace
make deploy NAMESPACE=my-namespace
```

For detailed deployment instructions, see [../docs/OPENSHIFT_DEPLOY.md](../docs/OPENSHIFT_DEPLOY.md).
