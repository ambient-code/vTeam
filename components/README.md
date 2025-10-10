# vTeam Components

This directory contains the core components of the vTeam platform. See the main [README.md](../README.md) for complete documentation.

## Component Overview

```
components/
├── frontend/                   # Next.js 15 web interface (React + Shadcn UI)
├── backend/                    # Go API service (Gin framework, K8s CRD management)
├── operator/                   # Kubernetes operator (reconciles AgenticSession CRs)
├── runners/
│   ├── claude-code-runner/     # Python runner (Claude Code SDK wrapper)
│   └── runner-shell/           # Runner shell library (WebSocket transport)
├── manifests/                  # Kubernetes manifests (CRDs, RBAC, deployments)
│   ├── deploy.sh              # Main deployment script
│   └── GIT_AUTH_SETUP.md      # Git authentication configuration
└── scripts/                    # Development and deployment utilities
```

## Architecture

**Agentic Session Flow:**
1. User creates session via web UI with prompt and optional repositories
2. Backend creates `AgenticSession` Custom Resource in project namespace
3. Operator reconciles CR and creates Kubernetes Job with runner + content sidecar
4. Runner executes Claude Code SDK session with workspace synced via PVC
5. Real-time messages streamed to frontend via WebSocket
6. Session results and metadata stored in CR status

## Quick Start

### Production Deployment

```bash
# Prepare configuration
cd manifests
cp env.example .env
# Edit .env: set ANTHROPIC_API_KEY

# Deploy to OpenShift
./deploy.sh
```

**Note:** GitHub secrets for git authentication must be created separately per project. See [manifests/GIT_AUTH_SETUP.md](manifests/GIT_AUTH_SETUP.md).

### Local Development

```bash
# From project root
make dev-start
```

See [scripts/local-dev/README.md](scripts/local-dev/README.md) for detailed local development setup.

### Building Custom Images

```bash
export REGISTRY="quay.io/your-username"
make build-all REGISTRY=$REGISTRY
make push-all REGISTRY=$REGISTRY

# Deploy with custom images
cd manifests
CONTAINER_REGISTRY=$REGISTRY ./deploy.sh
```
