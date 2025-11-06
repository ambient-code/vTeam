# vTeam E2E Tests

End-to-end testing suite for the vTeam platform using Cypress and kind (Kubernetes in Docker).

> **Status**: ✅ Production Ready | **Tests**: 5/5 Passing | **CI**: Automated on PRs

## Overview

This test suite deploys the complete vTeam application stack to a local kind cluster and runs automated tests to verify core functionality including project creation and navigation.

**What This Provides:**
- 🚀 **Automated E2E Testing**: Full stack deployment verification
- 🔄 **CI Integration**: Runs on every PR automatically  
- 🧪 **Local Testing**: Developers can run tests before pushing
- 📊 **Visual Debugging**: Video recordings and screenshots
- 🐳 **Flexible Runtime**: Supports both Docker and Podman

## Quick Start

Run the complete test suite with one command:

**From repository root (recommended):**
```bash
# Auto-detect container engine
make e2e-test

# Force Podman
make e2e-test CONTAINER_ENGINE=podman
```

**From e2e directory:**
```bash
cd e2e
./scripts/setup-kind.sh    # Create kind cluster
./scripts/deploy.sh         # Deploy vTeam
./scripts/run-tests.sh      # Run Cypress tests
./scripts/cleanup.sh        # Clean up (when done)
```

## Prerequisites

### Required Software

- **Docker OR Podman**: Container runtime for kind
  - Docker: https://docs.docker.com/get-docker/
  - Podman (alternative): `brew install podman` (macOS)
- **kind**: Kubernetes in Docker
  - Install: `brew install kind` (macOS) or see https://kind.sigs.k8s.io/
- **kubectl**: Kubernetes CLI
  - Install: `brew install kubectl` (macOS) or see https://kubernetes.io/
- **Node.js 20+**: For Cypress
  - Install: `brew install node` (macOS) or https://nodejs.org/

### Verify Installation

**With Docker:**
```bash
docker --version && docker ps
kind --version
kubectl version --client
node --version
```

**With Podman:**
```bash
podman --version
podman machine start    # Start Podman VM
podman ps              # Verify Podman is running
kind --version
kubectl version --client
```

## Architecture

**Test Environment:**
- **Kind cluster**: Lightweight local Kubernetes cluster
- **Direct authentication**: ServiceAccount token (no OAuth proxy for CI simplicity)
- **Cypress**: Modern e2e testing framework with TypeScript
- **Nginx Ingress**: Standard Kubernetes ingress controller
- **Kustomize overlays**: Uses `components/manifests/overlays/e2e/`

**Key Differences from Production:**
- Frontend: No oauth-proxy sidecar (direct token via env vars)
- Ingress: Uses Kubernetes Ingress instead of OpenShift Routes
- Storage: Explicit `storageClassName: standard` for kind
- Auth: ServiceAccount token instead of OAuth flow

## Project Structure

```
e2e/
├── scripts/               # Orchestration scripts
│   ├── setup-kind.sh      # Create kind cluster + ingress
│   ├── deploy.sh          # Deploy vTeam (uses overlay)
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

# Manifests are in components/manifests/overlays/e2e/
../components/manifests/overlays/e2e/
├── kustomization.yaml     # E2E overlay config
├── frontend-ingress.yaml
├── backend-ingress.yaml
├── test-user.yaml         # ServiceAccount for testing
├── secrets.yaml           # Minimal secrets
└── *-patch.yaml          # Environment-specific patches
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

**With Podman:** The script detects Podman and automatically uses ports 8080/8443 (not 80/443).

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
- Apply manifests using `../components/manifests/overlays/e2e/`
- Wait for all pods to be ready
- Extract test user token to `.env.test`

**Verify:**
```bash
kubectl get pods -n ambient-code

# With Docker:
curl http://vteam.local/api/health

# With Podman:
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

### Writing Tests

Example test structure:

```typescript
describe('vTeam Feature', () => {
  beforeEach(() => {
    // Setup runs before each test
    cy.visit('/')
  })

  it('should do something', () => {
    cy.get('[data-testid="element"]').click()
    cy.contains('Expected Text').should('be.visible')
  })
})
```

### Running Individual Tests

```bash
source .env.test

# Run specific test file
CYPRESS_TEST_TOKEN="$TEST_TOKEN" npx cypress run --spec "cypress/e2e/vteam.cy.ts"

# Run with UI
CYPRESS_TEST_TOKEN="$TEST_TOKEN" npm run test:headed
```

### Debugging Tests

```bash
# Open Cypress UI
source .env.test
CYPRESS_TEST_TOKEN="$TEST_TOKEN" npm run test:headed

# Enable debug logs
DEBUG=cypress:* npm test

# Check screenshots/videos
ls cypress/screenshots/
ls cypress/videos/
```

## Configuration

### Environment Variables

The test token is stored in `.env.test` (auto-generated by `deploy.sh`):

```bash
TEST_TOKEN=eyJhbGciOiJSUzI1NiIsImtpZCI6Ii...
CYPRESS_BASE_URL=http://vteam.local  # or :8080 for Podman
```

Cypress loads this via `CYPRESS_TEST_TOKEN` environment variable.

### Cypress Settings

Edit `cypress.config.ts` to customize:
- Base URL
- Timeouts
- Screenshot/video settings
- Viewport size

### Kubernetes Manifests

E2E manifests are managed via Kustomize overlay at:
```
../components/manifests/overlays/e2e/
```

Key configurations:
- **Frontend**: No oauth-proxy sidecar, test env vars injected
- **Ingress**: nginx-ingress with `vteam.local` host
- **Storage**: `storageClassName: standard` for kind
- **Auth**: Test user ServiceAccount with cluster-admin role

See `../components/manifests/README.md` for overlay structure details.

## Troubleshooting

### Kind cluster won't start

**With Docker:**
```bash
# Check Docker is running
docker ps

# Delete and recreate
kind delete cluster --name vteam-e2e
./scripts/setup-kind.sh
```

**With Podman:**
```bash
# Check Podman machine
podman machine list
podman machine start

# Verify Podman works
podman ps

# Recreate with Podman
kind delete cluster --name vteam-e2e
CONTAINER_ENGINE=podman ./scripts/setup-kind.sh
```

**Common issues:**
- **"Cannot connect to Docker daemon"**: Docker/Podman not running
  - Docker: Start Docker Desktop
  - Podman: Run `podman machine start`
- **"rootlessport cannot expose privileged port 80"**: Expected with Podman!
  - The setup script automatically uses port 8080 instead
  - Access at: `http://vteam.local:8080`

### Pods not starting

```bash
# Check pod status
kubectl get pods -n ambient-code

# Check pod logs
kubectl logs -n ambient-code -l app=frontend
kubectl logs -n ambient-code -l app=backend-api

# Describe pod for events
kubectl describe pod -n ambient-code <pod-name>
```

### Ingress not working

```bash
# Check ingress controller
kubectl get pods -n ingress-nginx

# Check ingress resources
kubectl get ingress -n ambient-code

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
curl http://vteam.local/api/health  # Add :8080 for Podman

# Manually test with token
source .env.test
curl -H "Authorization: Bearer $TEST_TOKEN" http://vteam.local/api/projects
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
5. Deploy vTeam using e2e overlay
6. Run tests
7. Upload artifacts (screenshots/videos) on failure
8. Cleanup cluster (always runs, even on failure)

**CI Environment:**
- **No password prompt**: GitHub Actions runners have passwordless sudo
- **Uses Docker**: Standard setup (no Podman needed)
- **Standard ports**: Port 80 (no rootless restrictions)
- **Timeout**: 15 minutes (typical runtime: 6-7 minutes)
- **Cleanup guaranteed**: Runs even if tests fail

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

## Quick Reference

### Manual Verification

After running `./scripts/deploy.sh`, test manually:

```bash
# Check all pods running
kubectl get pods -n ambient-code

# Test frontend (add :8080 for Podman)
curl http://vteam.local

# Test backend API
curl http://vteam.local/api/health

# Get test token
cat .env.test

# Test with authentication
source .env.test
curl -H "Authorization: Bearer $TEST_TOKEN" http://vteam.local/api/projects
```

### Keep Cluster Running

For iterative test development:

```bash
# Setup once
./scripts/setup-kind.sh
./scripts/deploy.sh

# Run tests multiple times
./scripts/run-tests.sh

# Iterate on tests...
npm run test:headed

# When done
./scripts/cleanup.sh
```

### Port Reference

| Container Engine | HTTP Port | HTTPS Port | URL |
|-----------------|-----------|------------|-----|
| Docker | 80 | 443 | http://vteam.local |
| Podman | 8080 | 8443 | http://vteam.local:8080 |

## Maintenance Checklist

### Before Merging PR

- [ ] All tests passing locally
- [ ] Tests passing in CI
- [ ] No new console errors in Cypress
- [ ] Screenshots/videos reviewed if tests failed
- [ ] Test covers new functionality (if applicable)

### Monthly

- [ ] Update Cypress and dependencies: `npm update`
- [ ] Verify tests still pass with latest versions
- [ ] Review and update test timeouts if needed
- [ ] Check for deprecated Cypress commands

### After Major Changes

- [ ] Backend API changes: Update test assertions
- [ ] Frontend UI changes: Update selectors
- [ ] Auth flow changes: Update token handling
- [ ] Deployment changes: Verify manifests in overlay

## External Resources

- [Cypress Documentation](https://docs.cypress.io/)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Kubernetes Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)
- [vTeam Manifests](../components/manifests/README.md) - Kustomize overlay structure
- [vTeam Main Documentation](../README.md)

## Support

For issues or questions:
1. Check [Troubleshooting](#troubleshooting) section above
2. Check GitHub Actions logs for CI failures
3. Check pod logs: `kubectl logs -n ambient-code <pod-name>`
4. Review manifest overlay: `../components/manifests/overlays/e2e/`
5. Open an issue in the repository

## License

Same as parent project (MIT License)
