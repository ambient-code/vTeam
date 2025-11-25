# Getting Started

Get the Ambient Code Platform up and running quickly! This guide walks you through everything needed to create your first AI-powered agentic session.

## Prerequisites

Before starting, ensure you have:

- **Kubernetes or OpenShift cluster** (or OpenShift Local for development)
- **Git** for cloning the repository
- **kubectl** or **oc** CLI tools
- **Anthropic Claude API key** ([Get one here](https://console.anthropic.com/))
- **Internet connection** for container image pulls and API calls

For local development:

- **OpenShift Local (CRC)** - [Installation guide](https://developers.redhat.com/products/openshift-local/overview)
- **Make** for running build commands
- **Docker or Podman** (optional, for building custom images)

## Quick Start - Local Development

The fastest way to get started is using OpenShift Local (CRC):

### Step 1: Install OpenShift Local

```bash
# Install CRC (one-time setup)
brew install crc

# Get your free Red Hat pull secret from:
# https://console.redhat.com/openshift/create/local

# Setup CRC (follow prompts to add pull secret)
crc setup
```

### Step 2: Clone and Deploy

```bash
# Clone the repository
git clone https://github.com/ambient-code/platform.git
cd platform

# Single command to start everything
make dev-start
```

This command will:

- Start OpenShift Local if not running
- Create the vteam-dev project/namespace
- Deploy all components (frontend, backend, operator, runner)
- Configure routes and services
- Display the frontend URL when ready

### Step 3: Configure API Key

After deployment, you need to configure your Anthropic API key:

```bash
# Create a project settings with your API key
# Access the UI (URL shown after dev-start)
# Navigate to Project Settings
# Add your ANTHROPIC_API_KEY
```

Alternatively, create it via CLI:

```bash
# Create the Secret with your API key
oc apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: runner-secrets
  namespace: vteam-dev
type: Opaque
stringData:
  ANTHROPIC_API_KEY: "sk-ant-api03-your-key-here"
EOF

# Create the ProjectSettings referencing the Secret
oc apply -f - <<EOF
apiVersion: vteam.ambient-code/v1alpha1
kind: ProjectSettings
metadata:
  name: projectsettings
  namespace: vteam-dev
spec:
  groupAccess:
    - groupName: "developers"
      role: "edit"
  runnerSecretsName: "runner-secrets"
EOF
```

### Step 4: Access the UI

```bash
# Get the frontend URL
echo "https://$(oc get route vteam-frontend -n vteam-dev -o jsonpath='{.spec.host}')"

# Open in browser and start creating agentic sessions!
```

## First Agentic Session

Now let's create your first agentic session to verify everything works:

### Using the Web Interface

1. **Access the ACP UI** in your browser
2. **Create a new project** (if not already created)
3. **Start a new AgenticSession**:
   - Provide a prompt describing your task
   - Optionally specify GitHub repositories to work with
   - Click "Create Session"
4. **Monitor progress** in real-time as the Claude Code agent executes your task
5. **Review results** when the session completes

## Verification Checklist

Ensure your installation is working correctly:

- [ ] All pods are running: `oc get pods -n vteam-dev`
- [ ] Frontend is accessible via browser
- [ ] Backend API health check passes: `/health` endpoint
- [ ] AgenticSession CR can be created
- [ ] Operator spawns Job pods for sessions
- [ ] No API authentication errors in operator logs

## Common Issues

### API Key Errors

**Symptom**: Agentic sessions fail with authentication errors
**Solution**:

1. Verify your Anthropic API key is correct in ProjectSettings
2. Check you have available credits in your Anthropic account
3. Ensure the API key is properly formatted: `sk-ant-api03-...`

### Pod Not Starting

**Symptom**: Pods stuck in `ImagePullBackOff` or `CrashLoopBackOff`
**Solution**:

```bash
# Check pod status and events
oc describe pod <pod-name> -n vteam-dev

# Check pod logs
oc logs <pod-name> -n vteam-dev

# Verify images are accessible
oc get pods -n vteam-dev -o jsonpath='{.items[*].spec.containers[*].image}'
```

### Deployment Failures

**Symptom**: `make dev-start` fails or times out
**Solution**:

1. Check CRC status: `crc status`
2. Ensure CRC has enough resources (recommend 8GB RAM minimum)
3. Check deployment logs: `make dev-logs`
4. Verify all CRDs are installed: `oc get crd | grep vteam`

### Session Job Failures

**Symptom**: AgenticSession jobs fail or timeout
**Solution**:

1. Check job logs: `oc logs job/<session-name> -n vteam-dev`
2. Verify workspace PVC is accessible
3. Check operator logs for errors: `make dev-logs-operator`
4. Ensure sufficient cluster resources for job pods

## What's Next?

Now that the Ambient Code Platform is running, you're ready to:

1. **Try hands-on exercises** → [Lab 1: Your First Agentic Session](../labs/basic/lab-1-first-rfe.md)
2. **Explore the reference documentation** → [Reference Guide](../reference/index.md)
3. **Review deployment options** → [OpenShift Deployment](../OPENSHIFT_DEPLOY.md)

## Getting Help

If you encounter issues not covered here:

- **Check AGENTS.md** in the repository root for detailed development documentation
- **Search existing issues** → [GitHub Issues](https://github.com/ambient-code/platform/issues)
- **Create a new issue** with your error details and environment info

Welcome to Kubernetes-native AI automation! 🚀
