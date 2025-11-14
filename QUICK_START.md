# Quick Start Guide

Get vTeam running locally in **under 5 minutes**! 🚀

## Prerequisites

Install these tools (one-time setup):

### macOS
```bash
brew install minikube kubectl podman
```

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

## Start vTeam

```bash
# Clone the repository
git clone https://github.com/ambient-code/vTeam.git
cd vTeam

# Start everything (builds images, starts minikube, deploys all components)
make local-up
```

That's it! The command will:
- ✅ Start minikube (if not running)
- ✅ Build all container images
- ✅ Deploy backend, frontend, and operator
- ✅ Set up ingress and networking

## Access the Application

Get the access URL:
```bash
make local-url
```

Or use NodePort directly:
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

## Quick Commands

```bash
# View logs
make local-logs              # Backend logs
make local-logs-frontend     # Frontend logs
make local-logs-operator     # Operator logs

# Rebuild and reload a component
make local-reload-backend    # After changing backend code
make local-reload-frontend   # After changing frontend code
make local-reload-operator   # After changing operator code

# Stop (keeps minikube running)
make local-down

# Completely remove minikube cluster
make local-clean
```

## What's Next?

- **Create a project**: Navigate to the frontend and create your first project
- **Run an agentic session**: Submit a task for AI-powered analysis
- **Explore the code**: See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines
- **Read the full docs**: Check out [docs/LOCAL_DEVELOPMENT.md](docs/LOCAL_DEVELOPMENT.md)

## Troubleshooting

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
# Restart minikube
minikube delete
minikube start

# Then redeploy
make local-up
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

⚠️ **Security Note**: This is for local development only. Production deployments require proper OAuth.

### Environment Variables
Local development uses these environment variables:
```yaml
ENVIRONMENT: local          # Enables dev mode
DISABLE_AUTH: "true"       # Disables authentication
```

These are set automatically in `components/manifests/minikube/` deployment files.

## Next Steps After Quick Start

1. **Explore the UI**: http://$(minikube ip):30030
2. **Create your first project**: Click "New Project" in the web interface
3. **Submit an agentic session**: Try analyzing a codebase
4. **Check the operator logs**: See how sessions are orchestrated
5. **Read the architecture docs**: [CLAUDE.md](CLAUDE.md) for component details

---

**Need more detailed setup?** See [docs/LOCAL_DEVELOPMENT.md](docs/LOCAL_DEVELOPMENT.md)

**Want to contribute?** See [CONTRIBUTING.md](CONTRIBUTING.md)

**Having issues?** Open an issue on [GitHub](https://github.com/ambient-code/vTeam/issues)

