# Quick Start Guide

Get Ambient Code Platform running locally in **under 5 minutes**! 

## Prerequisites

Install these tools (one-time setup):

### macOS
```bash
# Install tools
brew install minikube kubectl podman

# Check if you already have a podman machine
podman machine list
```

**If you see a machine already exists:**
```bash
# Check its memory (look for "MEMORY" column)
podman machine list

# If it has less than 6GB, reconfigure it:
podman machine stop
podman machine set --memory 6144
podman machine set --rootful
podman machine start
```

**If no machine exists yet:**
```bash
# Create a new podman machine with sufficient memory
podman machine init --memory 6144 --cpus 4
podman machine set --rootful
podman machine start
```

**Why 6GB?** Kubernetes needs substantial memory for its control plane. Less than 6GB will cause startup failures.

### Linux
```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Install minikube
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# Install podman
sudo apt install podman  # Ubuntu/Debian
# or
sudo dnf install podman  # Fedora/RHEL
```

**Note for Linux users**: Podman runs natively on Linux (no VM/machine needed). Just ensure your system has at least 6GB of free RAM for Kubernetes.

## Configure Vertex AI (Optional, but recommended for ease of use)

### 1. Authenticate with Google Cloud

Note that if you have Claude Code working with Vertex AI, you have probably already done all of this:

**Recommended: Use gcloud (easiest)**
```bash
# Install gcloud CLI if you haven't already
# https://cloud.google.com/sdk/docs/install

# Authenticate with your company Google account
gcloud auth application-default login

# Set your project (get this from your admin)
export ANTHROPIC_VERTEX_PROJECT_ID="your-gcp-project-id"
```

**Alternative: Use a service account key file**
```bash
# If your admin provided a service account key file:
export ANTHROPIC_VERTEX_PROJECT_ID="your-gcp-project-id"
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your-key.json"
```

### 2. Make Configuration Persistent

Add to your `~/.zshrc` or `~/.bashrc`:

```bash
# Vertex AI Configuration (for company work)
export ANTHROPIC_VERTEX_PROJECT_ID="your-gcp-project-id"

# Optional: Specify region (defaults to "global")
export CLOUD_ML_REGION="global"

# Optional: If using service account key instead of gcloud ADC
# export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"
```

Then reload your shell:
```bash
source ~/.zshrc  # or source ~/.bashrc
```

**That's it!** `make local-up` will automatically detect your configuration.

### 3. Verify Configuration

```bash
# Check your environment variables are set
echo $ANTHROPIC_VERTEX_PROJECT_ID

# Verify gcloud authentication
gcloud auth application-default print-access-token

# Or if using service account key:
# ls -l $GOOGLE_APPLICATION_CREDENTIALS
```

**Alternative**: If you skip the Vertex AI setup above, you can set an `ANTHROPIC_API_KEY` in workspace settings instead.

## Start Ambient Code Platform

```bash
# Clone the repository
git clone https://github.com/ambient-code/platform.git
cd platform

# Start everything (automatically detects Vertex AI from environment)
make local-up
```

That's it! The command will:
-  Start minikube (if not running)
-  Build all container images
-  **Auto-detect Vertex AI** from environment variables
-  Deploy backend, frontend, and operator
-  Set up ingress and networking
-  **On macOS**: Automatically start port forwarding in background

**What you'll see:**
-  "Found Vertex AI config in environment" → Using company Vertex AI
-  "Vertex AI not configured" → Using direct Anthropic API (workspace settings)

## Developer Workflow

**Made a code change?** Reload just that component (takes ~30 seconds, keeps everything else running):

```bash
# After changing backend code
make local-reload-backend

# After changing frontend code
make local-reload-frontend

# After changing operator code
make local-reload-operator
```

**These commands automatically:**
-  Rebuild only the changed component
-  Load the new image into minikube
-  Restart only that deployment
-  On macOS: Restart port forwarding for that component

**No need to restart everything!** Your other components keep running.

## Access the Application

### macOS with Podman (Automatic!)

Port forwarding starts automatically. Just wait ~30 seconds for pods to be ready, then access:
- **Frontend**: http://localhost:3000
- **Backend**: http://localhost:8080

**Stop port forwarding** if needed:
```bash
make local-stop-port-forward
```

**Restart port forwarding** if stopped:
```bash
make local-port-forward
```

### Linux or macOS with Docker

**Option 1: Port Forwarding**
```bash
make local-port-forward
```

Then access:
- **Frontend**: http://localhost:3000
- **Backend**: http://localhost:8080

**Option 2: NodePort (Direct access)**

```bash
# Get minikube IP
MINIKUBE_IP=$(minikube ip)

# Frontend: http://$MINIKUBE_IP:30030
# Backend:  http://$MINIKUBE_IP:30080
```

## Verify Everything Works

```bash
# Check status of all components
make local-status

# Run the test suite
./tests/local-dev-test.sh
```

## Quick Commands Reference

```bash
# Component reload (see "Developer Workflow" above for details)
make local-reload-backend    # Rebuild and reload backend only
make local-reload-frontend   # Rebuild and reload frontend only
make local-reload-operator   # Rebuild and reload operator only

# View logs
make local-logs              # All component logs
make local-logs-backend      # Backend logs only
make local-logs-frontend     # Frontend logs only
make local-logs-operator     # Operator logs only

# Port forwarding management (macOS)
make local-stop-port-forward # Stop background port forwarding
make local-port-forward      # Restart port forwarding (foreground)

# Cleanup
make local-down              # Stop app (keeps minikube, stops port forwarding)
make local-clean             # Delete minikube cluster completely
```

## What's Next?

- **Create a project**: Navigate to the frontend and create your first project
- **Run an agentic session**: Submit a task for AI-powered analysis
- **Explore the code**: See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines
- **Read the full docs**: Check out [docs/LOCAL_DEVELOPMENT.md](docs/LOCAL_DEVELOPMENT.md)

## Troubleshooting

### Podman machine has insufficient memory (macOS)?

First, check your current memory allocation:
```bash
podman machine list
# Look at the MEMORY column
```

If it shows less than 6GB (6144MB):
```bash
# Stop and reconfigure podman machine
podman machine stop
podman machine set --memory 6144
podman machine start

# Delete and restart minikube
minikube delete
make local-up
```

**Tip**: You can check if memory is the issue by looking for errors about "insufficient memory" or API server failures in `minikube logs`.

### Minikube can't find a driver?

**On macOS:**
Make sure podman machine is running:
```bash
podman machine list
# Should show "Currently running" in LAST UP column

# If not running:
podman machine start
```

**On Linux:**
Podman should work natively. Verify it's installed:
```bash
podman --version
podman ps  # Should not error
```

### Pods not starting?
```bash
# Check pod status
kubectl get pods -n ambient-code

# View pod logs
kubectl logs -n ambient-code -l app=backend-api
```

### Port already in use?
```bash
# Check what's using the port
lsof -i :30030  # Frontend
lsof -i :30080  # Backend

# Or use different ports by modifying the service YAML files
```

### Minikube issues?
```bash
# Check minikube status
minikube status

# Restart minikube cluster
minikube delete
make local-up

# View detailed minikube logs if startup fails
minikube logs
```

### Vertex AI authentication errors?

Check your authentication and configuration:
```bash
# Verify environment variables are set
echo $ANTHROPIC_VERTEX_PROJECT_ID

# Check gcloud authentication (most common method)
gcloud auth application-default print-access-token
# Should print an access token (not an error)

# Or if using service account key:
echo $GOOGLE_APPLICATION_CREDENTIALS
ls -l $GOOGLE_APPLICATION_CREDENTIALS

# Check if the secret was created in Kubernetes
kubectl get secret ambient-vertex -n ambient-code

# Check the operator logs for authentication errors
kubectl logs -n ambient-code -l app=agentic-operator --tail=50
```

**Common issues:**
- **gcloud not authenticated**: Run `gcloud auth application-default login`
- **Wrong project**: Check `$ANTHROPIC_VERTEX_PROJECT_ID` matches your GCP project
- **Quota/permissions**: Ensure your account has Vertex AI API access
- **Expired credentials**: Re-run `gcloud auth application-default login`

If you need to update configuration:
```bash
# Re-authenticate with gcloud
gcloud auth application-default login

# Or update your environment variables in ~/.zshrc
# Then reload and restart the platform
source ~/.zshrc
make local-down
make local-up  # Will automatically pick up new configuration
```

### Can't access the application?

**On macOS with Podman:**
Port forwarding should have started automatically. Check if it's running:
```bash
# Check port forwarding status
ps aux | grep "kubectl port-forward"

# View port forwarding logs
cat /tmp/ambient-code/port-forward-*.log

# Restart if needed
make local-stop-port-forward
make local-port-forward
```

**On Linux or macOS with Docker:**
Use NodePort with `minikube ip`:
```bash
curl http://$(minikube ip):30080/health
open http://$(minikube ip):30030
```

### Need help?
```bash
# Show all available commands
make help

# Run diagnostic tests
./tests/local-dev-test.sh
```

## Configuration

### Authentication (Local Dev Mode)
By default, authentication is **disabled** for local development:
- No login required
- Automatic user: "developer"
- Full access to all features

 **Security Note**: This is for local development only. Production deployments require proper OAuth.

### Environment Variables
Local development uses these environment variables:
```yaml
ENVIRONMENT: local          # Enables dev mode
DISABLE_AUTH: "true"       # Disables authentication
```

These are set automatically in `components/manifests/minikube/` deployment files.

### AI Access Configuration

**Vertex AI** (Recommended for company work):
-  Set via environment variables (see setup above)
-  Automatically detected by `make local-up`
-  Company-issued service accounts
-  Approved for confidential/proprietary code
- See [README.md](README.md) for advanced configuration

**Direct Anthropic API** (Non-confidential data only):
-  Only for public repos or non-sensitive work
- No environment variables needed
- Provide `ANTHROPIC_API_KEY` in workspace settings when creating a project
- Platform automatically uses this mode if Vertex AI env vars not set

### Optional Integrations

**GitHub App** (for OAuth login and repo browser):
- Follow: [docs/GITHUB_APP_SETUP.md](docs/GITHUB_APP_SETUP.md)
- Create secret: `kubectl create secret generic github-app-secret --from-literal=GITHUB_APP_ID=... -n ambient-code`
- Restart backend: `make local-reload-backend`
- **Note**: Not required for basic Git operations (use tokens in workspace settings)

**Jira Integration** (per-workspace):
- Configure directly in workspace settings UI
- Provide: JIRA_URL, JIRA_EMAIL, JIRA_API_TOKEN
- See: [components/manifests/GIT_AUTH_SETUP.md](components/manifests/GIT_AUTH_SETUP.md)

**Git Tokens** (per-workspace):
- For cloning/pushing to repositories
- Configure in workspace settings UI
- Can use GitHub personal access tokens or SSH keys
- See: [components/manifests/GIT_AUTH_SETUP.md](components/manifests/GIT_AUTH_SETUP.md)

## Next Steps After Quick Start

1. **Explore the UI**: 
   - Port forwarding (all): http://localhost:3000 (with `make local-port-forward` running)
   - NodePort (Linux or macOS+Docker): http://$(minikube ip):30030
2. **Create your first project**: Click "New Project" in the web interface
3. **Submit an agentic session**: Try analyzing a codebase
4. **Check the operator logs**: `make local-logs-operator`
5. **Read the architecture docs**: [CLAUDE.md](CLAUDE.md) for component details

---

**Need more detailed setup?** See [docs/LOCAL_DEVELOPMENT.md](docs/LOCAL_DEVELOPMENT.md)

**Want to contribute?** See [CONTRIBUTING.md](CONTRIBUTING.md)

**Having issues?** Open an issue on [GitHub](https://github.com/ambient-code/platform/issues)

