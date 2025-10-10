# vTeam: Ambient Agentic Runner

> OpenShift-native AI automation platform for intelligent agentic sessions with multi-agent collaboration

## Overview

**vTeam** is an OpenShift-native AI automation platform that orchestrates Claude Code SDK sessions with multi-agent personas. The platform enables teams to create and manage intelligent agentic sessions and RFE workflows through a modern web interface.

### Key Capabilities

- **Agentic Sessions**: AI-powered automation for code analysis, content creation, and development workflows
- **RFE Workflows**: Multi-phase feature refinement with specialized AI agent personas
- **Multi-Repo Support**: Work with umbrella and supporting repositories simultaneously
- **OpenShift Native**: Built with Custom Resources, Kubernetes Operators, and RBAC for enterprise deployment
- **Real-time Communication**: WebSocket-based live updates and interactive sessions

## Architecture

The platform consists of containerized microservices orchestrated via OpenShift/Kubernetes:

| Component | Technology | Description |
|-----------|------------|-------------|
| **Frontend** | Next.js 15 + Shadcn UI | React-based web interface for managing sessions and workflows |
| **Backend API** | Go + Gin | REST API for managing Custom Resources with project-scoped multi-tenancy |
| **Operator** | Go | Kubernetes operator that reconciles AgenticSession and RFEWorkflow CRs |
| **Claude Code Runner** | Python + Claude Code SDK | Executes AI sessions with workspace management and agent personas |
| **Content Service** | Go (sidecar mode) | Per-session content proxy for file operations and GitHub integration |

### Agentic Session Flow

1. **Create Session**: User creates session via web UI, specifying prompt and optional repos
2. **API Processing**: Backend creates `AgenticSession` Custom Resource in project namespace
3. **Job Scheduling**: Operator reconciles CR and creates Kubernetes Job with runner + content sidecars
4. **AI Execution**: Runner executes Claude Code SDK session with workspace synced via PVC
5. **WebSocket Streaming**: Real-time messages streamed to frontend via WebSocket connection
6. **Result Storage**: Session results and metadata stored in CR status field

## Prerequisites

### Required Infrastructure
- **OpenShift cluster** with admin access (or OpenShift Local/CRC for development)
- **oc CLI** configured to access your cluster
- **Container registry access** - default images available at `quay.io/ambient_code`

### Optional Build Tools (for custom images)
- **Docker or Podman** for building container images
- **Go 1.24+** for building backend and operator from source
- **Node.js 20+** and **pnpm** for building frontend from source

### Required API Keys
- **Anthropic API Key** - Get from [Anthropic Console](https://console.anthropic.com/)
  - Configure during deployment or via UI: Settings → Runner Secrets

## Quick Start

### 1. Deploy to OpenShift

Deploy using pre-built images from `quay.io/ambient_code`:

```bash
# Prepare environment configuration
cd components/manifests
cp env.example .env

# Edit .env and set required values:
# - ANTHROPIC_API_KEY (required for AI sessions)
# - Optionally: GITHUB_APP_ID, GITHUB_PRIVATE_KEY, OAUTH_CLIENT_SECRET

# Deploy to ambient-code namespace (default)
make deploy
```

### 2. Verify Deployment

```bash
# Check deployment status
oc get pods -n ambient-code

# Expected output: frontend, backend, operator pods running
# Watch operator logs
oc logs -f deployment/vteam-operator -n ambient-code
```

### 3. Access the Web Interface

```bash
# Get the frontend route URL
oc get route frontend-route -n ambient-code -o jsonpath='{.spec.host}'

# Open in browser, or use port forwarding:
oc port-forward svc/frontend-service 3000:3000 -n ambient-code
```

### 4. Create Your First Project

1. Access the web interface at the route URL
2. Click "New Project" to create a project namespace
3. Navigate to Settings → Runner Secrets to configure API keys
4. Create an agentic session or RFE workflow

## Usage

### Creating an Agentic Session

1. **Navigate to Sessions**: Select a project, click "Sessions" → "New Session"
2. **Configure Session**:
   - **Prompt**: Task description (e.g., "Analyze this codebase and suggest improvements")
   - **Model**: Choose AI model (Claude Sonnet 4.5 recommended)
   - **Repos** (optional): Add input repositories to clone into workspace
   - **Settings**: Adjust temperature, max tokens, timeout
3. **Monitor Execution**: View real-time message stream via WebSocket
4. **Access Results**: Browse workspace files, review conversation, check cost/usage

### Creating an RFE Workflow

1. **Navigate to RFE**: Select a project, click "RFE Workflows" → "New RFE"
2. **Configure Workflow**:
   - **Title & Description**: Feature overview
   - **Umbrella Repo**: Primary repository for the RFE
   - **Supporting Repos** (optional): Additional context repositories
3. **Execute Phases**: Run ideation, specification, planning, and implementation phases with specialized agent personas
4. **Review Artifacts**: Access generated documents, specifications, and implementation plans

### Example Use Cases

- **Feature Refinement**: Multi-agent RFE workflows with PM, UX, and engineering perspectives
- **Code Analysis**: Repository reviews, security audits, quality assessments
- **Documentation**: Generate technical specs, API docs, user guides
- **Implementation Planning**: Break down features into actionable tasks
- **Interactive Development**: Real-time AI pair programming sessions

## Configuration

### Building Custom Images

Build and push your own container images:

```bash
# Set container registry
export REGISTRY="quay.io/your-username"

# Build all images (frontend, backend, operator, runner)
make build-all

# Push to registry (requires authentication)
make push-all REGISTRY=$REGISTRY

# Deploy with custom images
cd components/manifests
# Edit .env to set CONTAINER_REGISTRY=$REGISTRY
./deploy.sh
```

### Container Engine Options

```bash
# Use Podman instead of Docker
make build-all CONTAINER_ENGINE=podman

# Build for specific platform (default: linux/amd64)
make build-all PLATFORM=linux/arm64

# Build with additional flags
make build-all BUILD_FLAGS="--no-cache --pull"
```

### OpenShift OAuth Integration

Enable cluster SSO authentication using OpenShift OAuth:

- See [docs/OPENSHIFT_OAUTH.md](docs/OPENSHIFT_OAUTH.md) for setup instructions
- Requires configuring OAuthClient and oauth-proxy sidecar
- Provides seamless user authentication with OpenShift credentials

### GitHub App Integration

Enable GitHub repository access and PR automation:

- See [docs/GITHUB_APP_SETUP.md](docs/GITHUB_APP_SETUP.md) for configuration
- Allows cloning private repos, creating branches, pushing changes
- Integrates with session workspace for repository operations

### Runner Secrets Management

Configure API keys per-project via the web interface:

- **Settings → Runner Secrets**: Manage secret references and values
- **Project-scoped**: Each project namespace has isolated secrets
- **Required secrets**: `ANTHROPIC_API_KEY` at minimum
- **Optional secrets**: GitHub tokens, Jira credentials, etc.

## Troubleshooting

### Common Issues

**Pods Not Starting:**
```bash
# Check pod status and events
oc get pods -n ambient-code
oc describe pod <pod-name> -n ambient-code
oc logs <pod-name> -n ambient-code
```

**Session Jobs Failing:**
```bash
# Check job and pod logs
oc get jobs -n <project-namespace>
oc describe job <session-job-name> -n <project-namespace>
oc logs <runner-pod-name> -c runner -n <project-namespace>
```

**WebSocket Connection Issues:**
```bash
# Verify backend health
oc port-forward svc/backend-service 8080:8080 -n ambient-code
curl http://localhost:8080/health

# Check backend logs for WebSocket errors
oc logs deployment/vteam-backend -n ambient-code --tail=50
```

**Authentication Issues:**
```bash
# Verify OAuth configuration (if enabled)
oc get oauthclient vteam-frontend -n ambient-code
oc describe route frontend-route -n ambient-code
```

### Verification Commands

```bash
# Check all platform components
oc get deployments,services,routes -n ambient-code

# View operator reconciliation logs
oc logs -f deployment/vteam-operator -n ambient-code

# Check Custom Resource Definitions
oc get crds | grep vteam.ambient-code

# View session status
oc get agenticsessions -n <project-namespace>
oc describe agenticsession <session-name> -n <project-namespace>
```

## Production Considerations

### Security
- **RBAC**: Per-project namespaces with role-based access control
- **Secret Management**: API keys stored as Kubernetes Secrets, project-scoped
- **Network Policies**: Isolate project namespaces and restrict pod communication
- **Image Scanning**: Scan container images for vulnerabilities
- **OAuth Integration**: Use OpenShift OAuth for enterprise SSO

### Monitoring & Observability
- **Health Endpoints**: Backend and content service expose `/health` endpoints
- **Operator Logs**: Monitor reconciliation loops and CR updates
- **Session Metrics**: Track session duration, cost, and success rates
- **Log Aggregation**: Use OpenShift logging or external log collectors

### Scaling
- **Multi-tenancy**: Projects map to namespaces for resource isolation
- **Resource Limits**: Configure resource requests/limits per project
- **PVC Management**: Sessions use persistent volumes for workspace storage
- **Job Cleanup**: Configure TTL for completed session jobs

## Development

### Local Development with OpenShift Local (CRC)

Run vTeam locally using OpenShift Local (CRC) for a production-like environment:

```bash
# Install CRC (macOS/Linux)
brew install crc  # or download from console.redhat.com

# Setup CRC with pull secret (one-time)
crc setup
# Download pull secret from: https://console.redhat.com/openshift/create/local

# Start local OpenShift cluster and deploy vTeam
make dev-start
```

**What this provides:**
- ✅ Full OpenShift cluster running locally
- ✅ Automatic image builds and deployment
- ✅ Production-like environment with RBAC
- ✅ Frontend, backend, and operator components
- ✅ Live log streaming

**Development with Hot Reloading:**
```bash
# Terminal 1: Start with development images
DEV_MODE=true make dev-start

# Terminal 2: Enable file sync for instant updates
make dev-sync
```

**Access Local Environment:**
- Frontend: `https://vteam-frontend-vteam-dev.apps-crc.testing`
- Backend API: `https://vteam-backend-vteam-dev.apps-crc.testing`
- OpenShift Console: `https://console-openshift-console.apps-crc.testing`

**Development Commands:**
```bash
make dev-logs            # View frontend and backend logs
make dev-logs-operator   # View operator logs
make dev-stop            # Stop local development (keeps cluster)
make dev-clean           # Stop and delete project
```

### Building from Source

```bash
# Build all components
make build-all

# Build individual components
make build-frontend
make build-backend
make build-operator
make build-runner
```

## Project Structure

```
vTeam/
├── components/                      # Main platform components
│   ├── frontend/                    # Next.js 15 web interface (React + Shadcn UI)
│   ├── backend/                     # Go API service (Gin framework)
│   ├── operator/                    # Kubernetes operator (reconciles CRs)
│   ├── runners/
│   │   ├── claude-code-runner/      # Python runner (Claude Code SDK)
│   │   └── runner-shell/            # Runner shell library (WebSocket transport)
│   ├── manifests/                   # Kubernetes manifests (CRDs, RBAC, deployments)
│   └── scripts/                     # Development and deployment scripts
├── docs/                            # Documentation
│   ├── OPENSHIFT_DEPLOY.md          # Deployment guide
│   ├── OPENSHIFT_OAUTH.md           # OAuth configuration
│   ├── GITHUB_APP_SETUP.md          # GitHub App integration
│   ├── user-guide/                  # User documentation
│   ├── developer-guide/             # Developer documentation
│   └── labs/                        # Hands-on tutorials
├── agents/                          # Agent persona definitions (YAML)
├── Makefile                         # Build and deployment automation
└── mkdocs.yml                       # Documentation site configuration
```

## Documentation

- **Deployment Guide**: [docs/OPENSHIFT_DEPLOY.md](docs/OPENSHIFT_DEPLOY.md)
- **OAuth Setup**: [docs/OPENSHIFT_OAUTH.md](docs/OPENSHIFT_OAUTH.md)
- **GitHub Integration**: [docs/GITHUB_APP_SETUP.md](docs/GITHUB_APP_SETUP.md)
- **Runner Architecture**: [docs/CLAUDE_CODE_RUNNER.md](docs/CLAUDE_CODE_RUNNER.md)
- **User Guide**: [docs/user-guide/](docs/user-guide/)
- **Developer Guide**: [docs/developer-guide/](docs/developer-guide/)

## Contributing

Contributions are welcome! To contribute:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Make your changes following existing code patterns
4. Test locally using `make dev-start`
5. Commit with descriptive messages
6. Push and open a Pull Request

## License

This project is licensed under the MIT License - see [LICENSE](LICENSE) for details.
