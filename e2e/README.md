# vTeam E2E Tests

End-to-end testing suite for the vTeam platform using Cypress and kind (Kubernetes in Docker).

## Overview

This test suite deploys the complete vTeam application stack to a local kind cluster and runs automated tests to verify core functionality including project creation and navigation.

**Architecture:**
- **Kind cluster**: Lightweight local Kubernetes cluster
- **Direct authentication**: ServiceAccount token (no OAuth proxy)
- **Cypress**: Modern e2e testing framework
- **CI-ready**: Automated testing in GitHub Actions

## Prerequisites

### Required Software

- **Docker OR Podman**: Container runtime for kind
  - Docker: https://docs.docker.com/get-docker/
  - Podman (alternative): `brew install podman` (macOS)
- **kind**: Kubernetes in Docker
  - Install: `brew install kind` (macOS) or see https://kind.sigs.k8s.io/docs/user/quick-start/
- **kubectl**: Kubernetes CLI
  - Install: `brew install kubectl` (macOS) or see https://kubernetes.io/docs/tasks/tools/
- **Node.js 20+**: For Cypress
  - Install: `brew install node` (macOS) or https://nodejs.org/

### Verify Installation

**With Docker:**
```bash
docker --version
docker ps  # Verify Docker is running
kind --version
kubectl version --client
node --version
npm --version
```

**With Podman:**
```bash
podman --version
podman machine init     # First time only
podman machine start    # Start Podman VM
podman ps              # Verify Podman is running
kind --version
kubectl version --client
node --version
npm --version
```

## Quick Start

Run the complete test suite with one command:

**With Docker (auto-detected):**
```bash
cd e2e
./scripts/setup-kind.sh    # Create kind cluster
./scripts/deploy.sh         # Deploy vTeam
./scripts/run-tests.sh      # Run Cypress tests
./scripts/cleanup.sh        # Clean up (when done)
```

**With Podman (explicitly specify):**
```bash
cd e2e
CONTAINER_ENGINE=podman ./scripts/setup-kind.sh
./scripts/deploy.sh
./scripts/run-tests.sh
./scripts/cleanup.sh

# Note: Podman uses ports 8080/8443 (not 80/443)
# Access at: http://vteam.local:8080
```

**Or use the Makefile from the repo root:**
```bash
# Auto-detect (prefers Docker, falls back to Podman)
make e2e-test

# Force Podman
make e2e-test CONTAINER_ENGINE=podman
```

## Detailed Workflow

### 1. Create Kind Cluster

```bash
cd e2e
./scripts/setup-kind.sh
```

This will:
- Create a kind cluster named `vteam-e2e`
- Install nginx-ingress controller
- Add `vteam.local` to `/etc/hosts` (requires sudo)

**Verify:**
```bash
kind get clusters
kubectl cluster-info
kubectl get nodes
```

### 2. Deploy vTeam

```bash
./scripts/deploy.sh
```

This will:
- Apply all Kubernetes manifests (CRDs, RBAC, deployments, ingress)
- Wait for all pods to be ready
- Extract test user token to `.env.test`

**Verify:**
```bash
kubectl get pods -n ambient-code
kubectl get ingress -n ambient-code

# With Docker:
curl http://vteam.local/api/health

# With Podman (port 8080):
curl http://vteam.local:8080/api/health
```

### 3. Run Tests

```bash
./scripts/run-tests.sh
```

This will:
- Install npm dependencies (if needed)
- Load test token from `.env.test`
- Run Cypress tests in headless mode

**Run in headed mode (with UI):**
```bash
source .env.test
CYPRESS_TEST_TOKEN="$TEST_TOKEN" npm run test:headed
```

### 4. Cleanup

```bash
./scripts/cleanup.sh
```

This will:
- Delete the kind cluster
- Remove `vteam.local` from `/etc/hosts`
- Clean up test artifacts

## Test Suite

The Cypress test suite (`cypress/e2e/vteam.cy.ts`) includes:

1. **Authentication test**: Verify token-based auth works
2. **Navigation test**: Access new project page
3. **Project creation**: Create a new project via UI
4. **Project listing**: Verify created projects appear
5. **API health check**: Test backend connectivity

## Project Structure

```
e2e/
├── manifests/              # Kubernetes manifests
│   ├── crds/              # Custom Resource Definitions (copied from prod)
│   ├── rbac/              # RBAC roles and bindings (copied from prod)
│   ├── namespace.yaml     # Namespace definition
│   ├── backend-deployment.yaml
│   ├── backend-ingress.yaml
│   ├── operator-deployment.yaml
│   ├── frontend-deployment.yaml  # Without oauth-proxy sidecar
│   ├── frontend-ingress.yaml
│   ├── workspace-pvc.yaml
│   ├── test-user.yaml     # ServiceAccount for testing
│   ├── secrets.yaml       # Minimal secrets
│   └── kustomization.yaml # Kustomize configuration
├── scripts/               # Orchestration scripts
│   ├── setup-kind.sh      # Create kind cluster
│   ├── deploy.sh          # Deploy vTeam
│   ├── wait-for-ready.sh  # Wait for pods
│   ├── run-tests.sh       # Run Cypress tests
│   └── cleanup.sh         # Teardown
├── cypress/               # Cypress test framework
│   ├── e2e/
│   │   └── vteam.cy.ts    # Main test suite
│   ├── support/
│   │   ├── commands.ts    # Custom commands
│   │   └── e2e.ts        # Support file
│   └── fixtures/          # Test data
├── cypress.config.ts      # Cypress configuration
├── package.json           # npm dependencies
├── tsconfig.json          # TypeScript config
└── README.md             # This file
```

## Configuration

### Environment Variables

The test token is stored in `.env.test` (auto-generated by `deploy.sh`):

```bash
TEST_TOKEN=eyJhbGciOiJSUzI1NiIsImtpZCI6Ii...
```

Cypress loads this via `CYPRESS_TEST_TOKEN` environment variable.

### Custom Cypress Settings

Edit `cypress.config.ts` to customize:
- Base URL
- Timeouts
- Screenshot/video settings
- Viewport size

### Kubernetes Resources

Manifests are in `manifests/` directory. Key differences from production:

**Frontend:**
- No oauth-proxy sidecar (direct port 3000 exposure)
- Simplified service (only HTTP port)

**Ingress:**
- Uses nginx-ingress instead of OpenShift Routes
- Host: `vteam.local`

**Storage:**
- Explicit `storageClassName: standard` for kind

**Authentication:**
- Test user ServiceAccount with cluster-admin role
- Token authentication instead of OAuth flow

## Troubleshooting

### Kind cluster won't start

**With Docker:**
```bash
# Check Docker is running
docker ps

# If Docker isn't running, start it (GUI or colima/orbstack)

# Delete and recreate
kind delete cluster --name vteam-e2e
./scripts/setup-kind.sh
```

**With Podman:**
```bash
# Check Podman machine is running
podman machine list

# Start Podman machine if needed
podman machine start

# Verify Podman works
podman ps

# Delete and recreate with Podman
kind delete cluster --name vteam-e2e
CONTAINER_ENGINE=podman ./scripts/setup-kind.sh
```

**Common issues:**
- **"Cannot connect to Docker daemon"**: Docker/Podman not running
  - Docker: Start Docker Desktop or your container runtime
  - Podman: Run `podman machine start`
- **"rootlessport cannot expose privileged port 80"**: Podman can't bind to ports < 1024
  - This is expected! The setup script automatically uses port 8080 instead
  - Access at: `http://vteam.local:8080`
- **"docker.sock permission denied"**: User not in docker group (Linux)
  - Add user to docker group: `sudo usermod -aG docker $USER`
  - Log out and back in

### Pods not starting

```bash
# Check pod status
kubectl get pods -n ambient-code

# Check pod logs
kubectl logs -n ambient-code -l app=frontend
kubectl logs -n ambient-code -l app=backend-api
kubectl logs -n ambient-code -l app=agentic-operator

# Describe pod for events
kubectl describe pod -n ambient-code <pod-name>
```

### Ingress not working

```bash
# Check ingress controller
kubectl get pods -n ingress-nginx

# Check ingress resources
kubectl get ingress -n ambient-code
kubectl describe ingress frontend-ingress -n ambient-code

# Test directly (bypass ingress)
kubectl port-forward -n ambient-code svc/frontend-service 3000:3000
# Then visit http://localhost:3000

# Verify /etc/hosts entry
grep vteam.local /etc/hosts
# Should see: 127.0.0.1 vteam.local
```

### Test failures

```bash
# Run with UI for debugging
source .env.test
CYPRESS_TEST_TOKEN="$TEST_TOKEN" npm run test:headed

# Check screenshots
ls cypress/screenshots/

# Verify backend is accessible
curl http://vteam.local/api/health

# Check frontend
curl http://vteam.local
```

### Token extraction fails

```bash
# Check secret exists
kubectl get secret test-user-token -n ambient-code

# Manually extract token
kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' | base64 -d
```

### Permission denied on scripts

```bash
chmod +x scripts/*.sh
```

## CI/CD Integration

The GitHub Actions workflow (`.github/workflows/e2e.yml`) runs automatically on:
- Pull requests to main/master
- Pushes to main/master
- Manual workflow dispatch

**Workflow steps:**
1. Checkout code
2. Set up Node.js
3. Install Cypress dependencies
4. Create kind cluster
5. Deploy vTeam
6. Run tests
7. Upload artifacts (screenshots/videos) on failure
8. Cleanup cluster

**View test results:**
- GitHub Actions tab → E2E Tests workflow
- Artifacts (screenshots/videos) available on failure

## Known Limitations

### What This Tests

✅ Core application functionality (project creation, navigation)  
✅ Backend API endpoints  
✅ Frontend UI rendering  
✅ Kubernetes deployment success  
✅ Service-to-service communication  

### What This Doesn't Test

❌ OAuth authentication flow (uses direct token auth)  
❌ OpenShift-specific features (Routes, OAuth server)  
❌ Production-like authentication (oauth-proxy sidecar removed)  
❌ Session creation and runner execution (requires additional setup)  

These limitations are acceptable trade-offs for fast, reliable CI testing.

## Development

### Adding New Tests

Edit `cypress/e2e/vteam.cy.ts`:

```typescript
it('should do something new', () => {
  cy.visit('/some-page')
  cy.contains('Expected Text').should('be.visible')
  // ... more assertions
})
```

### Running Individual Tests

```bash
# Run specific test file
CYPRESS_TEST_TOKEN="$TEST_TOKEN" npx cypress run --spec "cypress/e2e/vteam.cy.ts"

# Run tests matching pattern
CYPRESS_TEST_TOKEN="$TEST_TOKEN" npx cypress run --spec "cypress/e2e/**/*project*.cy.ts"
```

### Debugging

```bash
# Open Cypress UI
source .env.test
CYPRESS_TEST_TOKEN="$TEST_TOKEN" npm run test:headed

# Enable debug logs
DEBUG=cypress:* npm test
```

## Performance

**Typical run times:**
- Cluster setup: ~2 minutes
- Deployment: ~3-5 minutes
- Test execution: ~30 seconds
- Total: ~6-7 minutes

**Resource usage:**
- Docker containers: ~4-6 running
- Memory: ~4-6 GB
- CPU: Moderate during startup, low during tests

## Additional Resources

- [Cypress Documentation](https://docs.cypress.io/)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Kubernetes Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)
- [vTeam Main Documentation](../README.md)

## Support

For issues or questions:
1. Check [Troubleshooting](#troubleshooting) section
2. Review GitHub Actions logs for CI failures
3. Check pod logs: `kubectl logs -n ambient-code <pod-name>`
4. Open an issue in the repository

## License

Same as parent project (MIT License)

