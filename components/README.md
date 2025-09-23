# vTeam Components

This directory contains the core components of the vTeam Ambient Agentic Runner platform.

See the main [README.md](../README.md) for complete documentation, deployment instructions, and usage examples.

## Component Directory Structure

```
components/
├── frontend/                   # NextJS web interface with Shadcn UI
├── backend/                    # Go API service for Kubernetes CRD management
├── operator/                   # Kubernetes operator (Go)
├── runners/                    # AI runner services
│   └── claude-code-runner/     # Python Claude Code CLI with MCP integration
└── manifests/                  # Kubernetes deployment manifests and deploy script
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
