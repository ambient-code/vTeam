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
- **Jira API** - Epic creation and synchronization (implemented)
- **Anthropic Claude** - Conversational AI and agent assistance
- **Google Vertex AI** - Alternative AI provider support
- **Git Repositories** - Future integration for code context

## Key Features

- **Conversational RFE Creation**: Natural language interface with real-time structured data extraction
- **Multi-Agent Workflow**: Specialized AI agents model realistic software team dynamics
- **Visual Workflow Tracking**: Progress visualization with step-by-step status updates
- **Cost Management**: Built-in API usage tracking and response caching
- **Jira Integration**: Automated Epic creation from refined RFEs
- **Agent Dashboard**: Role-specific views for different team members

### Components

| Component | Technology | Description |
|-----------|------------|-------------|
| **Frontend** | NextJS + Shadcn | User interface for managing agentic sessions |
| **Backend API** | Go + Gin | REST API for managing Kubernetes Custom Resources |
| **Agentic Operator** | Go | Kubernetes operator that watches CRs and creates Jobs |
| **Ambient Runner** | Python + AI CLI | Pod that executes AI with Playwright MCP server |
| **Playwright MCP** | MCP Server | Provides browser automation capabilities to AI |

## Prerequisites

### For Using Pre-built Images (Recommended)
- **Kubernetes cluster** (local with minikube/kind or cloud-based) or **OpenShift**
- **kubectl** v1.28+ configured to access your cluster
- **Anthropic API Key** - Get one from [Anthropic Console](https://console.anthropic.com/)

### For Building Images from Source (Advanced)
- **Docker or Podman** for building container images
- **Container registry access** (Docker Hub, Quay.io, ECR, etc.)
- **Go 1.24+** for building backend services
- **Node.js 18+** and **npm/pnpm** for the frontend


## Quick Start

### **Common Setup (All Deployment Options)**

1. **Clone and Setup**
   ```bash
   git clone https://github.com/your-org/vTeam.git
   cd vTeam
   ```

2. **Configure Environment**
   ```bash
   cd components/manifests
   cp env.example .env
   # Edit .env and add: ANTHROPIC_API_KEY=your-actual-key-here
   ```

Now choose your deployment path:

### 🚀 **Option A: OpenShift Deployment (Recommended)**

For complete OpenShift deployment with pre-built images and HTTPS routes:

```bash
# Deploy (includes route with automatic HTTPS)
./deploy.sh

# Get the HTTPS route URL
oc get route frontend-route -n ambient-code -o jsonpath='{.spec.host}'
# Open browser to the displayed URL
```

### ⚡ **Option B: Kubernetes Deployment**

For basic Kubernetes deployment with pre-built images:

```bash
# Comment out route.yaml in kustomization.yaml (not needed for regular K8s)
sed -i '' 's/^- route.yaml$/# - route.yaml/' kustomization.yaml

# Deploy with kubectl and kustomize
kubectl apply -k .

# Access via port forward
kubectl port-forward svc/frontend-service 3000:3000 -n ambient-code
# Open browser to: http://localhost:3000
```

### 🔧 **Option C: Build from Source**

For custom images or development:

1. **Build All Images**
   ```bash
   # Using Docker
   make build-all

   # Using Podman
   make build-all CONTAINER_ENGINE=podman

   # For specific platform
   make build-all PLATFORM=linux/amd64
   ```

2. **Push to Registry (if using custom registry)**
   ```bash
   make push-all REGISTRY=your-registry.com
   ```

3. **Update Image References**
   ```bash
   # Edit components/manifests/kustomization.yaml
   # Uncomment and modify the images: section
   ```

4. **Deploy**
   ```bash
   cd components/manifests
   ./deploy.sh
   ```

## Usage

Once deployed, you can create and manage agentic sessions through the web interface:

### Creating an Agentic Session

1. **Access the Web Interface**
   - Navigate to `http://localhost:3000` (if using port forwarding)
   - Or your configured domain/ingress endpoint

2. **Create New Session**
   - Click "New Agentic Session"
   - Fill out the form with:
     - **Prompt**: Task description for the AI (e.g., "Analyze this website's user experience")
     - **Website URL**: Target website to analyze
     - **Model**: Choose AI model (Ambient AI v1, etc.)
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
kubectl get endpoints -n ambient-code

# Test API connectivity
kubectl exec -it <pod-name> -n ambient-code -- curl http://backend-service:8080/health
```

#### Image Pull Errors

**Common Symptoms:**
- Pods stuck in `ImagePullBackOff` or `ErrImagePull` status
- Error: `401 UNAUTHORIZED` when pulling from private registries

**For Public Images:**
```bash
# Verify registry access
docker pull $REGISTRY/backend:latest

# Check image references in manifests
grep "image:" manifests/*.yaml

# Update registry references if needed
sed -i '' "s|old-registry|new-registry|g" manifests/*.yaml
```

**For Private Images (e.g., quay.io, Docker Hub private repos):**

1. **Create Registry Secret:**
```bash
# Method 1: Using explicit credentials
kubectl create secret docker-registry registry-secret \
  --docker-server=quay.io \
  --docker-username=YOUR_USERNAME \
  --docker-password=YOUR_PASSWORD \
  --docker-email=YOUR_EMAIL \
  --namespace=ambient-code

# Method 2: Using robot account (recommended for quay.io)
kubectl create secret docker-registry registry-secret \
  --docker-server=quay.io \
  --docker-username=YOUR_ROBOT_ACCOUNT \
  --docker-password=YOUR_ROBOT_TOKEN \
  --docker-email=YOUR_EMAIL \
  --namespace=ambient-code
```

2. **Configure Service Accounts (No Manifest Changes Required):**
```bash
# Add image pull secrets to existing service accounts
kubectl patch serviceaccount backend-api -n ambient-code \
  -p '{"imagePullSecrets": [{"name": "registry-secret"}]}'

kubectl patch serviceaccount agentic-operator -n ambient-code \
  -p '{"imagePullSecrets": [{"name": "registry-secret"}]}'

# For deployments using default service account (like frontend)
kubectl patch serviceaccount default -n ambient-code \
  -p '{"imagePullSecrets": [{"name": "registry-secret"}]}'
```

3. **Restart Deployments:**
```bash
kubectl rollout restart deployment backend-api -n ambient-code
kubectl rollout restart deployment agentic-operator -n ambient-code
kubectl rollout restart deployment frontend -n ambient-code
```

4. **Verify Configuration:**
```bash
# Check if secret exists and is properly formatted
kubectl get secret registry-secret -n ambient-code -o yaml
kubectl get secret registry-secret -n ambient-code -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d

# Verify service accounts have image pull secrets
kubectl get serviceaccount backend-api -n ambient-code -o yaml
kubectl get serviceaccount agentic-operator -n ambient-code -o yaml
kubectl get serviceaccount default -n ambient-code -o yaml

# Check pod status
kubectl get pods -n ambient-code
kubectl describe pod <failing-pod-name> -n ambient-code
```

**Troubleshooting Private Image Issues:**
```bash
# If using Docker Desktop credential store, create explicit config
docker --config /tmp/docker-config login quay.io
kubectl create secret generic registry-secret \
  --from-file=.dockerconfigjson=/tmp/docker-config/config.json \
  --type=kubernetes.io/dockerconfigjson \
  --namespace=ambient-code
rm -rf /tmp/docker-config

# Test local access to verify credentials
docker pull quay.io/your-username/your-image:latest
```

#### Job Failures
```bash
# List jobs
kubectl get jobs -n ambient-code

# Check job details
kubectl describe job <job-name> -n ambient-code

# Check failed pod logs
kubectl logs <failed-pod-name> -n ambient-code
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
kind load docker-image backend:latest --name ambient-agentic
kind load docker-image frontend:latest --name ambient-agentic
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



