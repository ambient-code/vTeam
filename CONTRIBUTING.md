# Contributing to Ambient Code Platform

Thank you for your interest in contributing to Ambient Code Platform (formerly known as vTeam)! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Ways to Contribute](#ways-to-contribute)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Local Development Setup](#local-development-setup)
- [Troubleshooting](#troubleshooting)
- [Getting Help](#getting-help)
- [License](#license)

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors. We expect:

- Respectful and constructive communication
- Welcoming and inclusive behavior
- Focus on what is best for the community
- Showing empathy towards other community members

## Ways to Contribute

There are many ways to contribute to Ambient Code Platform:

### 🐛 Report Bugs

If you find a bug, please create an issue with:

- Clear, descriptive title
- Steps to reproduce the problem
- Expected vs actual behavior
- Environment details (OS, cluster version, etc.)
- Relevant logs or screenshots

### 💡 Suggest Features

We welcome feature suggestions! Please:

- Check if the feature has already been suggested
- Provide a clear use case and rationale
- Consider implementation approaches
- Be open to discussion and feedback

### 📝 Improve Documentation

Documentation improvements are always appreciated:

- Fix typos or clarify unclear sections
- Add examples or tutorials
- Document undocumented features
- Improve error messages

### 💻 Submit Code Changes

Code contributions should:

- Follow our code standards (see below)
- Include tests where applicable
- Update documentation as needed
- Pass all CI/CD checks

## Getting Started

### Prerequisites

Before contributing, ensure you have:

- Go 1.24+ (for backend/operator development)
- Node.js 20+ and npm (for frontend development)
- Python 3.11+ (for runner development)
- Podman or Docker (for building containers)
- Minikube and kubectl (for local development)
- Git for version control

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/vTeam.git
   cd vTeam
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/ambient-code/vTeam.git
   ```

## Development Workflow

### 1. Create a Feature Branch

Always work on a feature branch, not `main`:

```bash
git checkout main
git pull upstream main
git checkout -b feature/your-feature-name
```

Branch naming conventions:

- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test improvements

### 2. Make Your Changes

- Follow the existing code patterns and style
- Write clear, descriptive commit messages
- Keep commits focused and atomic
- Test your changes locally

### 3. Commit Your Changes

Use conventional commit messages:

```bash
git commit -m "feat: add multi-repo session support"
git commit -m "fix: resolve PVC mounting issue in minikube"
git commit -m "docs: update minikube setup instructions"
git commit -m "test: add integration tests for operator"
```

Commit message prefixes:

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

### 4. Keep Your Branch Updated

Regularly sync with upstream:

```bash
git fetch upstream
git rebase upstream/main
```

### 5. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

## Code Standards

### Go Code (Backend & Operator)

**Formatting:**
```bash
# Auto-format your code
gofmt -w components/backend components/operator
```

**Quality Checks:**
```bash
# Backend
cd components/backend
gofmt -l .                    # Check formatting (should output nothing)
go vet ./...                  # Detect suspicious constructs
golangci-lint run            # Run comprehensive linting

# Operator
cd components/operator
gofmt -l .
go vet ./...
golangci-lint run
```

**Install golangci-lint:**
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Best Practices:**

- Use explicit error handling, never `panic()` in production code
- Always use user-scoped Kubernetes clients for API operations
- Implement proper RBAC checks before resource access
- Never log sensitive data (tokens, API keys)
- Use `unstructured.Nested*` helpers for type-safe CR access
- Set OwnerReferences on child resources for automatic cleanup

See [CLAUDE.md](CLAUDE.md) for comprehensive backend/operator development standards.

### Frontend Code (NextJS)

```bash
cd components/frontend
npm run lint                  # ESLint checks
npm run build                 # Ensure builds without errors/warnings
```

**Best Practices:**

- Zero `any` types (use proper TypeScript types)
- Use Shadcn UI components only (no custom UI from scratch)
- Use React Query for ALL data operations (no manual `fetch()`)
- Use `type` over `interface`
- Colocate single-use components with their pages
- All buttons must show loading states
- All lists must have empty states
- All nested pages must have breadcrumbs

See [components/frontend/DESIGN_GUIDELINES.md](components/frontend/DESIGN_GUIDELINES.md) for complete frontend standards.

### Python Code (Runners)

```bash
cd components/runners/claude-code-runner

# Format code
black .
isort .

# Lint
flake8
```

**Standards:**

- Use `black` formatting (88 char line length)
- Use `isort` for import sorting
- Follow PEP 8 conventions
- Add type hints where appropriate

## Testing Requirements

### Backend Tests

```bash
cd components/backend
make test              # All tests
make test-unit         # Unit tests only
make test-contract     # Contract tests only
make test-integration  # Integration tests (requires k8s cluster)
make test-coverage     # Generate coverage report
```

### Operator Tests

```bash
cd components/operator
go test ./... -v
```

### Frontend Tests

```bash
cd components/frontend
npm test
```

**Testing Guidelines:**

- Add tests for new features
- Ensure tests pass locally before pushing
- Aim for meaningful test coverage
- Write clear test descriptions
- Use table-driven tests in Go

## Pull Request Process

### Before Submitting

1. **Run all quality checks** for the components you modified
2. **Run tests** and ensure they pass
3. **Update documentation** if you changed functionality
4. **Rebase on latest main** to avoid merge conflicts
5. **Test locally** with Minikube if possible

### PR Description

Your PR should include:

- **Clear title** describing the change
- **Description** of what changed and why
- **Related issues** (use "Fixes #123" or "Relates to #123")
- **Testing performed** - how you verified the changes
- **Screenshots** (if UI changes)
- **Breaking changes** (if any)

### Review Process

- All PRs require at least one approval
- GitHub Actions will automatically run:
  - Go linting checks (gofmt, go vet, golangci-lint)
  - Component builds
  - Tests
- Address review feedback promptly
- Keep discussions focused and professional
- Be open to suggestions and alternative approaches

### After Approval

- Squash commits will happen automatically on merge
- Your PR will be merged to `main`
- Delete your feature branch after merge

## Local Development Setup

The recommended way to develop and test Ambient Code Platform locally is using **Minikube**. This provides a lightweight Kubernetes environment on your local machine with no authentication requirements, making development fast and easy.

### Installing Minikube and Prerequisites

#### macOS

```bash
# Install using Homebrew
brew install minikube kubectl
```

#### Linux (Debian/Ubuntu)

```bash
# Install Podman
sudo apt-get update
sudo apt-get install podman

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install Minikube
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
```

#### Linux (Fedora/RHEL)

```bash
# Install Podman
sudo dnf install podman

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install Minikube
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
```

### Quick Start

Once Minikube and prerequisites are installed, you can start the complete development environment with a single command:

#### First-Time Setup

```shell
make local-up
```

This command will:
- Start Minikube with appropriate resources
- Enable required addons (ingress, storage)
- Build container images
- Deploy all components (backend, frontend, operator)
- Set up networking

The setup takes 2-3 minutes on first run.

#### Access the Application

Get the access URL:

```shell
make local-url
```

This will display the frontend and backend URLs, typically:
- Frontend: `http://192.168.64.4:30030`
- Backend: `http://192.168.64.4:30080`

Or manually construct the URL:

```shell
# Get Minikube IP
minikube ip

# Access at http://<minikube-ip>:30030
```

**Authentication:**

Authentication is **completely disabled** for local development:
- ✅ No login required
- ✅ Automatic login as "developer"
- ✅ Full access to all features
- ✅ Backend uses service account for Kubernetes API

#### Stopping and Restarting

Stop the application (keeps Minikube running):

```shell
make local-stop
```

Restart the application:

```shell
make local-up
```

Delete the entire Minikube cluster:

```shell
make local-delete
```

### Additional Development Commands

**Check status:**
```bash
make local-status       # View pod status and deployment info
```

**View logs:**
```bash
make local-logs         # Backend logs
make local-logs-frontend # Frontend logs (if available)
make local-logs-operator # Operator logs (if available)
```

**Cleanup:**
```bash
make local-stop         # Stop deployment, keep Minikube running
make local-delete       # Delete entire Minikube cluster
```

**Access Kubernetes:**
```bash
kubectl get pods -n ambient-code       # View pods
kubectl logs <pod-name> -n ambient-code # View specific pod logs
kubectl describe pod <pod-name> -n ambient-code # Debug pod issues
```

## Troubleshooting

### Minikube Installation and Setup Issues

#### Insufficient Resources

If Minikube or the platform won't start, you may need to allocate more resources:

```shell
# Stop Minikube
minikube stop

# Delete the existing cluster
minikube delete

# Start with more resources
minikube start --memory=8192 --cpus=4 --disk-size=50g

# Then deploy the application
make local-up
```

#### Minikube Won't Start

If Minikube fails to start, try these steps:

```shell
# Check status
minikube status

# View logs
minikube logs

# Try with a specific driver
minikube start --driver=podman
# or
minikube start --driver=docker
```

#### Complete Minikube Reset

If Minikube is completely broken, you can fully reset it:

```shell
# Stop and delete cluster
minikube stop
minikube delete

# Clear cache (optional)
rm -rf ~/.minikube/cache

# Start fresh
minikube start --memory=4096 --cpus=2
make local-up
```

### Application Issues

#### Viewing Logs via CLI

The fastest way to view logs:

```bash
make local-logs         # Backend logs
kubectl logs -n ambient-code -l app=backend --tail=100 -f
kubectl logs -n ambient-code -l app=frontend --tail=100 -f
kubectl logs -n ambient-code -l app=operator --tail=100 -f
```

#### Viewing Logs via Kubernetes Dashboard

For detailed debugging through the Kubernetes dashboard:

```bash
# Open Kubernetes dashboard
minikube dashboard
```

This will open a web interface where you can:
1. Navigate to **Workloads > Pods**
2. Select the `ambient-code` namespace
3. Click on a pod to view details and logs

#### Common Issues

**Pods not starting:**

```bash
kubectl get pods -n ambient-code
kubectl describe pod <pod-name> -n ambient-code
```

**Image pull errors:**

```bash
kubectl get events -n ambient-code --sort-by='.lastTimestamp'
```

**Check if images are loaded:**

```bash
minikube ssh docker images | grep ambient-code
```

**PVC issues:**

```bash
kubectl get pvc -n ambient-code
kubectl describe pvc <pvc-name> -n ambient-code
```

**Service not accessible:**

```bash
# Check services
kubectl get services -n ambient-code

# Check NodePort assignments
kubectl get service backend -n ambient-code -o jsonpath='{.spec.ports[0].nodePort}'
kubectl get service frontend -n ambient-code -o jsonpath='{.spec.ports[0].nodePort}'

# Get Minikube IP
minikube ip
```

**Networking issues:**

```bash
# Verify ingress addon is enabled
minikube addons list | grep ingress

# Enable if disabled
minikube addons enable ingress
```

## Getting Help

If you're stuck or have questions:

1. **Check existing documentation:**
   - [CLAUDE.md](CLAUDE.md) - Comprehensive development standards
   - [README.md](README.md) - Project overview and quick start
   - [docs/](docs/) - Additional documentation

2. **Search existing issues:**
   - Check if your issue has already been reported
   - Look for solutions in closed issues

3. **Create a new issue:**
   - Provide clear description and reproduction steps
   - Include relevant logs and error messages
   - Tag with appropriate labels

## License

By contributing to Ambient Code Platform, you agree that your contributions will be licensed under the same license as the project (MIT License).
