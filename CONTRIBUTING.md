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
- Docker or Podman (for building containers)
- OpenShift Local (CRC) or access to an OpenShift/Kubernetes cluster
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

### Install Git Hooks (Recommended)

To prevent accidental commits to protected branches (`main`, `master`, `production`), install our git hooks:

```bash
make setup-hooks
```

Or run the installation script directly:

```bash
./scripts/install-git-hooks.sh
```

**What the hooks do:**

- **pre-commit** - Blocks commits to `main`/`master`/`production` branches
- **pre-push** - Blocks pushes to `main`/`master`/`production` branches

**Hooks are automatically installed** when you run `make dev-start`.

If you need to override the hooks (e.g., for hotfixes):

```bash
git commit --no-verify -m "hotfix: critical fix"
git push --no-verify origin main
```

See [scripts/git-hooks/README.md](scripts/git-hooks/README.md) for more details.

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
git commit -m "fix: resolve PVC mounting issue in CRC"
git commit -m "docs: update CRC setup instructions"
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
5. **Test locally** with CRC if possible

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

The recommended way to develop and test Ambient Code Platform locally is using OpenShift Local (CRC - CodeReady Containers). This provides a complete OpenShift environment running on your local machine with real authentication, RBAC, and production-like behavior.

### Installing and Setting Up CRC

#### RHEL/Fedora

See [crc instructions for RHEL/Fedora](https://medium.com/@Tal-Hason/openshift-local-aka-crc-install-and-customize-on-fedora-any-linux-6eb775035e06)

#### macOS

1. **Download CRC 2.54.0** (recommended version):
   - Download from: [CRC 2.54.0](https://mirror.openshift.com/pub/openshift-v4/clients/crc/2.54.0/)
   - **Why 2.54.0?** Later versions have known certificate expiration issues that can cause failures like `Failed to update pull secret on the disk: Temporary error: pull secret not updated to disk (x204)`
   - Choose the appropriate file for your system (e.g., `crc-macos-amd64.pkg` or `crc-macos-arm64.pkg`)

2. **Download your pull secret**:
   - Visit: https://console.redhat.com/openshift/create/local
   - Click the "Download pull secret" button
   - This downloads a file called `pull-secret`

3. **Install CRC**:
   - Run the downloaded `.pkg` installer
   - Follow the installation prompts

4. **Set up pull secret**:

   ```bash
   mkdir -p ~/.crc
   mv ~/Downloads/pull-secret ~/.crc/pull-secret.json
   ```

### Quick Start with CRC

Once CRC is installed and configured, you can start the complete development environment:

#### First-Time Setup

First, set up and start CRC:

```shell
crc setup
crc start
```

After the last command, make note of the admin usernames and passwords since you may need them to log in to the OpenShift console.

Next run the command to start the Ambient Code Platform:

```shell
make dev-start
```

To access Ambient Code Platform:

- open https://vteam-frontend-vteam-dev.apps-crc.testing in a browser

#### Stopping and Restarting

You can stop `crc` with:

```shell
crc stop
```

and then restart `crc` and Ambient Code Platform with:

```shell
crc start
make dev-start
```

If this doesn't work, you may want to do a full cleanup to get an entirely fresh start:

```shell
crc stop
crc cleanup
rm -rf ~/.crc/cache
rm -rf ~/.crc/machines
crc setup
crc start
make dev-start
```

Be sure to keep the new admin credentials after running `crc start` too.

### Development with Hot Reloading

If you have made local changes and want to test them with hot-reloading, use development mode:

#### Enable Development Mode

Instead of `make dev-start`, first run:

```shell
DEV_MODE=true make dev-start
```

#### Start File Sync

Then, in a **separate terminal**, run:

```shell
make dev-sync
```

This enables hot-reloading for both backend and frontend, automatically syncing your local changes to the running pods. You can now edit code locally and see changes reflected immediately.

**Sync individual components:**
```shell
make dev-sync-backend   # Sync only backend
make dev-sync-frontend  # Sync only frontend
```

### Additional Development Commands

**View logs:**
```bash
make dev-logs           # Both backend and frontend
make dev-logs-backend   # Backend only
make dev-logs-frontend  # Frontend only
make dev-logs-operator  # Operator only
```

**Operator management:**
```bash
make dev-restart-operator  # Restart operator
make dev-operator-status   # Show operator status
```

**Cleanup:**
```bash
make dev-stop              # Stop processes, keep CRC running
make dev-stop-cluster      # Stop processes and shutdown CRC
make dev-clean             # Stop and delete OpenShift project
```

## Troubleshooting

### CRC Installation and Setup Issues

#### Insufficient Resources

If `crc` or the platform won't start, you may need to allocate more resources:

```shell
crc stop
crc config set cpus 8
crc config set memory 16384
crc config set disk-size 200
crc start
```

#### CRC Version Issues

If you encounter issues with CRC (especially certificate expiration problems), try version 2.54.0 which is known to work well:

- Download: [CRC 2.54.0](https://mirror.openshift.com/pub/openshift-v4/clients/crc/2.54.0/)

#### Complete CRC Reset

If CRC is completely broken, you can fully reset it:

```shell
crc stop
crc delete
crc cleanup

# Remove CRC user directory
sudo rm -rf ~/.crc

# Remove CRC installation
sudo rm -rf /usr/local/crc
sudo rm /usr/local/bin/crc

# Verify they're gone
ls -la ~/.crc 2>&1
ls -la /usr/local/crc 2>&1
which crc 2>&1
```

After resetting, restart from the [Installing and Setting Up CRC](#installing-and-setting-up-crc) section.

#### Pull Secret Issues

If CRC can't find your pull secret, verify the pull secret file exists at `~/.crc/pull-secret.json` and then run:

```shell
crc config set pull-secret-file ~/.crc/pull-secret.json
```

Then restart CRC.

### Application Issues

#### Viewing Logs via CLI

The fastest way to view logs:

```bash
make dev-logs              # Both backend and frontend
make dev-logs-backend      # Backend only
make dev-logs-frontend     # Frontend only
make dev-logs-operator     # Operator only
```

#### Viewing Logs via OpenShift Console

For detailed debugging through the OpenShift web console:

1. Open https://console-openshift-console.apps-crc.testing in a browser
2. Log in with the administrator credentials (shown when you ran `crc start`)
3. Navigate to **Home > Projects** → select `vteam-dev`
4. Go to **Workloads > Pods**
5. Find pods in `Running` state (backend, frontend, operator)
6. Click on a pod → **Logs** tab

**Tip:** Start with the backend pod for most issues, as it handles core platform logic.

#### Common Issues

**Pods not starting:**

```bash
oc get pods -n vteam-dev
oc describe pod <pod-name> -n vteam-dev
```

**Image pull errors:**

```bash
oc get events -n vteam-dev --sort-by='.lastTimestamp'
```

**PVC issues:**

```bash
oc get pvc -n vteam-dev
oc describe pvc backend-state-pvc -n vteam-dev
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
