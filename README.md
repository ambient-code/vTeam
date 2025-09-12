# vTeam: Ambient Agentic Runner

> AI-powered automation system to reduce engineering refinement time and improve ticket quality

## Overview

**vTeam** is a comprehensive AI automation platform centered around the **Ambient Agentic Runner** - a production-ready Kubernetes system that revolutionizes how teams handle complex analysis and automation tasks. The platform enables users to create and manage intelligent agentic sessions through a modern web interface, leveraging AI with integrated browser automation capabilities.

### Key Capabilities

The Refinement Agent Team system transforms Request for Enhancement (RFE) submissions into well-refined, implementation-ready tickets through intelligent AI agent collaboration, enabling engineering teams to start work immediately with comprehensive context and clear acceptance criteria.

## Problem Statement

Engineering teams currently spend excessive time in refinement meetings due to:
- Poorly prepared tickets lacking necessary context
- Missing detailed feature breakdowns  
- Unclear acceptance criteria
- Disconnected information across RFEs, code repositories, and architectural documents

## Solution

The Refinement Agent Team system addresses these challenges through intelligent AI automation:
- **7-Agent Council Process** - Specialized AI agents (PM, Architect, Staff Engineer, PO, Team Lead, Team Member, Delivery Owner) handle different refinement aspects
- **Conversational RFE Creation** - Natural language interface powered by Anthropic Claude for intuitive ticket creation
- **Comprehensive Context Assembly** - Automatically enriches tickets with business justification, technical requirements, and success criteria
- **Workflow Orchestration** - Guided progression through standardized refinement steps
- **Integration Ready** - Built for seamless integration with existing Jira workflows

## Success Metrics

- 🎯 **90% ticket readiness** for immediate engineering execution
- ⏱️ **50% reduction** in refinement meeting duration
- 🚀 **25% improvement** in engineering velocity
- 📊 **Measurable time savings** in refinement hours per ticket

## Architecture

The platform consists of multiple containerized services orchestrated via Kubernetes:

### Agent Council Workflow
The system implements a 7-step refinement process with specialized AI agents:

1. **Parker (PM)** - RFE Prioritization
2. **Archie (Architect)** - Technical Review
3. **Stella (Staff Engineer)** - Completeness Check
4. **Archie (Architect)** - Acceptance Criteria Validation
5. **Stella (Staff Engineer)** - Accept/Reject Decision
6. **Parker (PM)** - Assessment Communication
7. **Derek (Delivery Owner)** - Feature Ticket Creation

### Integration Points
- **Jira integration (optional)** - Bot accounts/webhook patterns available; not enabled by default
- **Anthropic Claude** - Conversational AI and agent assistance
- **Google Vertex AI** - Alternative AI provider support
- **Git Repositories** - Future integration for code context

## Key Features

- **Conversational RFE Creation**: Natural language interface with real-time structured data extraction
- **Multi-Agent Workflow**: Specialized AI agents model realistic software team dynamics
- **Visual Workflow Tracking**: Progress visualization with step-by-step status updates
- **Cost Management**: Built-in API usage tracking and response caching
- **Optional Jira Automation**: Bot/webhook-driven session creation (no default Epic sync)
- **Agent Dashboard**: Role-specific views for different team members

### Components

| Component | Technology | Description |
|-----------|------------|-------------|
| **Frontend** | NextJS + Shadcn | User interface for managing agentic sessions |
| **Backend API** | Go + Gin | REST API for managing Kubernetes Custom Resources (multi-tenant: projects, agentic-sessions, groups, keys) |
| **Agentic Operator** | Go | Kubernetes operator that watches CRs and creates Jobs |
| **Ambient Runner** | Python + AI CLI | Pod that executes AI with Playwright MCP server |
| **Playwright MCP** | MCP Server | Provides browser automation capabilities to AI |

## Prerequisites

Before deploying the Ambient Agentic Runner, ensure you have:

### Required Tools
- **Kubernetes cluster** (local with minikube/kind or cloud-based like EKS/GKE/AKS)
- **kubectl** v1.28+ configured to access your cluster  
- **Docker or Podman** for building container images
- **Container registry access** (Docker Hub, Quay.io, ECR, etc.)
- **Go 1.24+** for building backend services (if building from source)
- **Node.js 18+** and **npm/pnpm** for the frontend (if building from source)

### Required Accounts & API Keys
- **Anthropic API Key** - Get one from [Anthropic Console](https://console.anthropic.com/)
  - Provide keys via the app UI after deployment (Settings → Runner Secrets)

### OpenShift OAuth (Recommended)
- For cluster login and auth in front of the UI, follow [OpenShift OAuth Setup](docs/OPENSHIFT_OAUTH.md)


## Quick Start

### 1. Verify Prerequisites

```bash
# Check required tools
kubectl version --client
docker --version  # or podman --version
git --version

# Verify cluster access
kubectl cluster-info
kubectl get nodes
```

### 2. Clone and Setup

```bash
# Clone the repository
git clone https://github.com/your-org/vTeam.git
cd vTeam
```

### 3. Configure Container Registry

```bash
# Set your container registry (replace with your registry)
export REGISTRY="your-registry.com"  # e.g., "quay.io/your-username"

# Login to your container registry
docker login $REGISTRY
```

### 4. Build and Push Images

```bash
# Build all container images
make build-all REGISTRY=$REGISTRY

# Push images to registry
make push-all REGISTRY=$REGISTRY
```

### 5. Configure Environment (optional overrides)

```bash
# Navigate to deployment folder
cd components/manifests

# Create optional environment file for overrides (namespace, images)
cp env.example .env

# Edit .env to set NAMESPACE, CONTAINER_REGISTRY, IMAGE_TAG if desired
nano .env  # or your preferred editor
```

Notes:
- Secrets (e.g., Anthropic API keys) are configured in the UI after deployment: Settings → Runner Secrets
- For OAuth-protected access on OpenShift, complete [OpenShift OAuth Setup](../../docs/OPENSHIFT_OAUTH.md)

### 6. (Optional) Override Images

The deploy script supports image overrides without editing YAML. You can set these via environment or `.env`:

```bash
# Example (one-off):
CONTAINER_REGISTRY="quay.io/your-username" IMAGE_TAG="latest" ./deploy.sh

# Or in .env (preferred):
# CONTAINER_REGISTRY=quay.io/your-username
# IMAGE_TAG=latest
# DEFAULT_BACKEND_IMAGE=
# DEFAULT_FRONTEND_IMAGE=
# DEFAULT_OPERATOR_IMAGE=
# DEFAULT_RUNNER_IMAGE=
```

### 7. Deploy to OpenShift/Kubernetes

```bash
# From components/manifests
./deploy.sh
```

### 8. Verify Deployment

```bash
# Check all pods are running
oc get pods -n ambient-code

# Check services are available
oc get services -n ambient-code
```

### 9. Access the Application

```bash
# On OpenShift (recommended): use the Route
oc -n ambient-code get route frontend -o jsonpath='{.spec.host}' | sed 's#^#https://#'

# Fallback (testing): Port forward locally
kubectl -n ambient-code port-forward svc/frontend-service 3000:3000
# Open: http://localhost:3000
```

For configuring cluster login for the UI, see [OpenShift OAuth Setup](docs/OPENSHIFT_OAUTH.md).

## Usage

Once deployed, you can create and manage agentic sessions through the web interface:

### Creating an Agentic Session (project-scoped)

1. **Access the Web Interface**
   - Navigate to `http://localhost:3000` (if using port forwarding)
   - Or your configured domain/ingress endpoint

2. **Create New Session**
   - Click "New Agentic Session"
   - Fill out the form with:
     - **Prompt**: Task description for the AI (e.g., "Analyze this website's user experience")
     - **Website URL**: Target website to analyze
   - **Model**: Choose AI model (e.g., Claude Sonnet/Haiku)
     - **Settings**: Adjust temperature, token limits as needed

3. **Monitor Progress**
   - View real-time status updates
   - Monitor job execution logs
   - Track completion status

4. **Review Results**
   - Download analysis results
   - View structured output
   - Export findings

### Example Use Cases

- **Website Analysis**: Analyze user experience, accessibility, performance
- **Competitive Research**: Compare features across competitor websites  
- **Content Auditing**: Review content quality and SEO optimization
- **Automation Testing**: Verify functionality across different scenarios

## Troubleshooting

### Common Issues

#### Pods Not Starting
```bash
# Check pod status and events
kubectl describe pod <pod-name> -n ambient-code

# Check logs
kubectl logs <pod-name> -n ambient-code

# Check resource constraints
kubectl top pods -n ambient-code
```

#### API Connection Issues
```bash
# Check service endpoints
oc get endpoints -n ambient-code

# Test API connectivity
oc exec -it <pod-name> -n ambient-code -- curl http://backend-service:8080/health
```

#### Image Pull Errors
```bash
# Verify registry access
docker pull $REGISTRY/backend:latest

# Check image references in manifests
grep "image:" manifests/*.yaml

# Update registry references if needed
sed -i "s|old-registry|new-registry|g" manifests/*.yaml
```

#### Job Failures
```bash
# List jobs
oc get jobs -n ambient-code

# Check job details
oc describe job <job-name> -n ambient-code

# Check failed pod logs
oc logs <failed-pod-name> -n ambient-code
```

### Verification Commands

```bash
# Check all resources
kubectl get all -l app=ambient-code -n ambient-code

# View recent events
kubectl get events --sort-by='.lastTimestamp' -n ambient-code

# Test frontend access
curl -f http://localhost:3000 || echo "Frontend not accessible"

# Test backend API
kubectl port-forward svc/backend-service 8080:8080 -n ambient-code &
curl http://localhost:8080/health
```

## Production Considerations

### Security
- **API Key Management**: Store Anthropic API keys securely in Kubernetes secrets
- **RBAC**: Configure appropriate role-based access controls
- **Network Policies**: Implement network isolation between components
- **Image Scanning**: Scan container images for vulnerabilities before deployment

### Monitoring
- **Prometheus Metrics**: Configure metrics collection for all components
- **Log Aggregation**: Set up centralized logging (ELK, Loki, etc.)
- **Alerting**: Configure alerts for pod failures, resource exhaustion
- **Health Checks**: Implement comprehensive health endpoints

### Scaling
- **Horizontal Pod Autoscaling**: Configure HPA based on CPU/memory usage
- **Resource Limits**: Set appropriate resource requests and limits
- **Node Affinity**: Configure pod placement for optimal resource usage

## Development

### Local Development
```bash
# Frontend development
cd components/frontend
npm install
npm run dev

# Backend development
cd components/backend
export KUBECONFIG=~/.kube/config
go run main.go

# Testing with Kind
kind create cluster --name ambient-agentic
# Load images built locally (names must match Makefile tags)
kind load docker-image vteam_backend:latest --name ambient-agentic
kind load docker-image vteam_frontend:latest --name ambient-agentic
kind load docker-image vteam_operator:latest --name ambient-agentic
kind load docker-image vteam_claude_runner:latest --name ambient-agentic
```

### Building from Source
```bash
# Build all images locally
make build-all

# Build specific components
make build-frontend
make build-backend
make build-operator
make build-runner
```

## File Structure

```
vTeam/
├── components/                     # 🚀 Ambient Agentic Runner (Main Platform)
│   ├── frontend/                   # NextJS web interface
│   ├── backend/                    # Go API service
│   ├── operator/                   # Kubernetes operator
│   ├── runners/                   # AI runner services
│   │   └── claude-code-runner/    # Python Claude Code CLI service
│   ├── manifests/                  # Kubernetes deployment files
│   └── README.md                   # Detailed setup documentation
├── docs/                           # Documentation (MkDocs)
│   └── ambient-runner/             # Platform-specific documentation
├── diagrams/                       # Architecture diagrams
└── Makefile                        # Build and deployment automation
```

## Other vTeam Components

This repository also contains additional tools and demonstrations that complement the main Ambient Agentic Runner platform:

### 📊 RFE Builder Demo
A Streamlit-based demonstration of multi-agent RFE (Request for Enhancement) analysis system.

- **Location**: `demos/rfe-builder/`
- **Technology**: Python, Streamlit, LlamaIndex, LlamaDeploy
- **Purpose**: Showcase AI-powered feature analysis with 7 specialized agent roles

**Quick Start:**
```bash
cd demos/rfe-builder
uv sync && uv run generate
uv run -m llama_deploy.apiserver
# See demos/rfe-builder/README.md for complete setup
```

[📖 View RFE Builder Documentation](demos/rfe-builder/README.md)

### 🔧 vTeam Shared Configurations
Automated team configuration management for development standards.

- **Location**: `tools/vteam_shared_configs/`
- **Technology**: Python CLI, Git hooks
- **Purpose**: Enforce consistent development workflows and Claude Code configurations

**Quick Start:**
```bash
cd tools/vteam_shared_configs
uv pip install -e .
vteam-config install
```

**Available Commands:**
- `vteam-config status` - Show current configuration
- `vteam-config update` - Update to latest version  
- `vteam-config uninstall` - Remove configuration

### 🔌 MCP Client Integration
Library for integrating with Model Context Protocol servers.

- **Location**: `tools/mcp_client_integration/`
- **Technology**: Python, asyncio
- **Purpose**: Simplify MCP server communication in AI applications

[📖 View MCP Client Documentation](tools/mcp_client_integration/README.md)

---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests if applicable
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.



