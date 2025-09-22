# vTeam OpenShift Deployment Guide

## What is vTeam?

**vTeam** is a Kubernetes-native AI automation platform that combines Claude Code CLI with browser automation and spec-driven development capabilities.

### Core Components
- **Web UI**: React frontend for creating and monitoring AI sessions
- **Backend API**: Go service managing sessions and Kubernetes resources
- **Operator**: Kubernetes controller creating AI runner jobs
- **Claude Runner**: Python service executing AI tasks with browser automation and SpekKit

### Key Features
- **Website Analysis**: AI-powered browser automation using Playwright MCP
- **Spec-Driven Development**: Generate specifications, plans, and tasks with `/specify`, `/plan`, `/tasks` commands
- **Git Integration**: Clone repositories, configure Git users, support SSH/token authentication
- **Configurable Defaults**: Set organization-wide Git repositories and settings via ConfigMap

## Prerequisites

- OpenShift cluster with admin access
- Container registry access (quay.io/ambient_code)
- `kubectl` or `oc` CLI configured

## Quick Deploy

1. **Navigate to manifests directory**:
   ```bash
   cd components/manifests
   ```

2. **Run deployment script**:
   ```bash
   ./deploy.sh
   ```

3. **Access the UI**: Check for route creation or port-forward:
   ```bash
   oc get route frontend-route
   # or
   oc get route frontend-route -o jsonpath='https://{.spec.host}'
   ```

## What Gets Deployed

### Core Services
- **Frontend** (`vteam_frontend:latest`) - React UI on port 3000
- **Backend** (`vteam_backend:latest`) - Go API on port 8080
- **Operator** (`vteam_operator:latest`) - Kubernetes controller
- **Git ConfigMap** - Default Git configuration (optional)

### Per-Session Resources
- **Claude Runner Jobs** (`vteam_claude_runner:latest`) - AI execution pods
- **AgenticSession CRDs** - Custom resources tracking AI sessions
- **Persistent Storage** - Session state and artifacts

## Configuration

### Git Integration (Optional)
Edit `git-configmap.yaml` to set default repositories:
```yaml
git-repositories: |
  https://github.com/your-org/project.git
  https://github.com/your-org/docs.git
```

### Environment Variables
Key operator settings:
- `IMAGE_PULL_POLICY=Always` (default) - Always pull latest images
- `AMBIENT_CODE_RUNNER_IMAGE` - Claude runner image to use

## Usage

1. **Open the web UI** at the frontend route/port
2. **Create AI Session**:
   - Enter prompt and target website URL
   - Configure Git settings (optional)
   - Submit to start AI analysis

3. **Use SpekKit Commands**:
   - `/specify Build user authentication system` - Generate specifications
   - `/plan Use Node.js and PostgreSQL` - Create implementation plans
   - `/tasks Focus on backend API first` - Break down into tasks

4. **Monitor Progress**: View real-time updates and final results in the UI

## Troubleshooting

- **Check pod logs**: `kubectl logs -l app=backend-api`
- **Verify operator**: `kubectl logs -l app=agentic-operator`
- **Session status**: `kubectl get agenticsessions`
- **Storage issues**: `kubectl get pvc vteam-state-storage`

## Cleanup

```bash
kubectl delete -f .
kubectl delete crd agenticsessions.vteam.ambient-code
```
