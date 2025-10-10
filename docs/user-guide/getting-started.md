# Getting Started

Get vTeam up and running on OpenShift in just a few minutes.

## Prerequisites

Before starting, ensure you have:

- **OpenShift cluster** with admin access (or OpenShift Local/CRC for local development)
- **oc CLI** configured to access your cluster
- **Anthropic API key** for Claude AI ([Get one here](https://console.anthropic.com/))
- **Internet connection** for pulling container images and API calls

## Installation

### Step 1: Prepare Environment Configuration

```bash
# Clone the repository (if not already done)
git clone https://github.com/red-hat-data-services/vTeam.git
cd vTeam

# Prepare environment file for deployment
cd components/manifests
cp env.example .env
```

### Step 2: Configure API Keys

Edit the `.env` file and set required values:

```bash
# Required: Anthropic API key for Claude AI
ANTHROPIC_API_KEY=sk-ant-api03-your-key-here

# Optional: GitHub App integration (for private repo access)
GITHUB_APP_ID=your-app-id
GITHUB_PRIVATE_KEY_BASE64=your-base64-encoded-private-key

# Optional: OAuth client secret (for OpenShift SSO)
OAUTH_CLIENT_SECRET=your-oauth-secret
```

!!! warning "Keep Your Keys Secret"
    Never commit `.env` to version control. It's already in `.gitignore`.

### Step 3: Deploy to OpenShift

Deploy vTeam using pre-built images:

```bash
# Deploy to ambient-code namespace (default)
cd ../..  # Back to repo root
make deploy
```

This will:
- Create the `ambient-code` namespace
- Deploy frontend, backend, and operator
- Create Custom Resource Definitions (CRDs)
- Set up RBAC and service accounts
- Create routes for web access

### Step 4: Verify Deployment

Check that all components are running:

```bash
# Check pod status
oc get pods -n ambient-code

# Expected output:
# NAME                              READY   STATUS    RESTARTS   AGE
# vteam-backend-xxx                 1/1     Running   0          2m
# vteam-frontend-xxx                1/1     Running   0          2m
# vteam-operator-xxx                1/1     Running   0          2m

# Watch operator logs
oc logs -f deployment/vteam-operator -n ambient-code
```

### Step 5: Access the Web Interface

Get the frontend URL and open in your browser:

```bash
# Get the route URL
oc get route frontend-route -n ambient-code -o jsonpath='{.spec.host}'

# Or use port forwarding as alternative
oc port-forward svc/frontend-service 3000:3000 -n ambient-code
# Then open http://localhost:3000
```

## First Steps

### Create Your First Project

1. **Open the web interface** at the route URL
2. **Click "New Project"** to create a project
   - Name: `my-first-project`
   - Display Name: "My First Project"
   - Description: Optional description
3. **Project namespace created**: vTeam creates a dedicated OpenShift namespace for your project

### Configure Runner Secrets

1. **Select your project** from the project list
2. **Navigate to Settings** → **Runner Secrets**
3. **Add secrets**:
   - `ANTHROPIC_API_KEY`: Your Claude API key
   - Other secrets as needed (GitHub tokens, Jira credentials, etc.)

### Create Your First Agentic Session

1. **Navigate to Sessions** → **New Session**
2. **Configure the session**:
   ```
   Prompt: "Analyze the vTeam architecture and suggest improvements"
   Model: Claude Sonnet 4.5
   Temperature: 0.7
   Max Tokens: 4096
   Timeout: 300 seconds
   ```
3. **Add repositories (optional)**:
   - Click "Add Repository"
   - Enter repository URL and branch
4. **Click "Create Session"**
5. **Monitor execution**: Watch real-time message stream as Claude analyzes the code

### Review Results

After the session completes:

1. **Overview Tab**: View session metadata, cost, and status
2. **Messages Tab**: Review the full conversation with Claude
3. **Workspace Tab**: Browse generated files and modified code
4. **Results Summary**: Check execution time, token usage, and cost

## Verification Checklist

Ensure your installation is working correctly:

- [ ] Frontend, backend, and operator pods are running
- [ ] Route is accessible and UI loads
- [ ] Project creation succeeds
- [ ] Runner secrets can be configured
- [ ] Agentic session can be created
- [ ] Session executes and completes successfully
- [ ] WebSocket messages stream in real-time
- [ ] Workspace files are accessible

## Common Issues

### Pods Not Starting

**Symptom**: Pods stuck in `Pending` or `CrashLoopBackOff`

**Solution**:
```bash
# Check pod events
oc describe pod <pod-name> -n ambient-code

# Check logs
oc logs <pod-name> -n ambient-code

# Common causes:
# - Image pull errors: Check image registry access
# - Resource limits: Check node capacity
# - ConfigMap/Secret missing: Verify deployment script ran completely
```

### Session Jobs Failing

**Symptom**: Agentic sessions fail immediately or timeout

**Solution**:
```bash
# Check job status
oc get jobs -n <project-namespace>

# Check runner pod logs
oc logs <runner-pod-name> -c runner -n <project-namespace>

# Common causes:
# - Missing ANTHROPIC_API_KEY in runner secrets
# - Network connectivity issues to Anthropic API
# - Invalid repository URLs
# - PVC mount issues
```

### WebSocket Connection Issues

**Symptom**: Messages don't stream in real-time, "Connecting..." message persists

**Solution**:
```bash
# Verify backend health
oc port-forward svc/backend-service 8080:8080 -n ambient-code
curl http://localhost:8080/health

# Check backend logs for WebSocket errors
oc logs deployment/vteam-backend -n ambient-code --tail=50

# Common causes:
# - Route not configured for WebSocket passthrough
# - Backend pod not ready
# - Browser blocking WebSocket connections
```

### API Authentication Errors

**Symptom**: "Invalid API key" or "Authentication failed" errors

**Solution**:
1. Verify API key is correct in Settings → Runner Secrets
2. Check key has sufficient credits at [Anthropic Console](https://console.anthropic.com/)
3. Ensure key format is correct: `sk-ant-api03-...`
4. Verify secret is properly mounted in runner pods

## What's Next?

Now that vTeam is running, explore these topics:

1. **RFE Workflows** → Learn how to create multi-phase feature refinement workflows
2. **Agent Personas** → Understand the specialized AI agents available
3. **GitHub Integration** → Set up GitHub App for private repository access
4. **OAuth Setup** → Enable OpenShift SSO for seamless authentication
5. **Multi-Repo Sessions** → Work with multiple repositories simultaneously

## Getting Help

If you encounter issues not covered here:

- **Check the troubleshooting guide** → [Troubleshooting](../reference/troubleshooting.md)
- **Review deployment docs** → [OpenShift Deployment](../OPENSHIFT_DEPLOY.md)
- **Search existing issues** → [GitHub Issues](https://github.com/red-hat-data-services/vTeam/issues)
- **Create a new issue** with error details and environment info

Welcome to vTeam! 🚀