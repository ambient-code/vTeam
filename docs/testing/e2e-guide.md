# E2E Testing Guide

This guide provides comprehensive documentation for writing and maintaining end-to-end tests for the Ambient Code Platform.

## Quick Start

```bash
# Run E2E tests
make e2e-test CONTAINER_ENGINE=podman  # Or docker
```

See `e2e/README.md` for comprehensive guide, troubleshooting, and architecture details.

## E2E Testing Patterns

### 1. Test Environment Isolation

Each test run gets a fresh environment:

```bash
# Setup: Create new kind cluster
# Test: Run Cypress suite
# Teardown: Delete cluster and artifacts
```

### 2. Authentication Strategy

Frontend deployment gets test token via env vars:

```yaml
env:
- name: OC_TOKEN
  valueFrom:
    secretKeyRef:
      name: test-user-token
      key: token
```

- Leverages existing frontend fallback auth logic (`buildForwardHeadersAsync`)
- No code changes needed in frontend
- ServiceAccount with cluster-admin for e2e tests only

### 3. Port Configuration

Auto-detects container runtime:

```bash
Docker:  ports 80/443  → http://vteam.local
Podman:  ports 8080/8443 → http://vteam.local:8080
```

### 4. Manifest Management

```
e2e/manifests/
├── Production manifests (copied as-is):
│   ├── crds/ (all CRDs)
│   ├── rbac/ (all RBAC)
│   ├── backend-deployment.yaml
│   └── operator-deployment.yaml
├── Adapted for kind:
│   ├── frontend-deployment.yaml (no oauth-proxy)
│   ├── workspace-pvc.yaml (storageClassName: standard)
│   └── namespace.yaml (no OpenShift annotations)
└── Kind-specific:
    ├── *-ingress.yaml (replaces Routes)
    ├── test-user.yaml (ServiceAccount)
    └── secrets.yaml (minimal config)
```

### 5. Test Organization

Use descriptive test names:

```typescript
it('should create a new project', () => {
  // Arrange: Navigate to form
  cy.visit('/projects/new')

  // Act: Fill and submit
  cy.get('#name').type('test-project')
  cy.contains('button', 'Create Project').click()

  // Assert: Verify success
  cy.url().should('include', '/projects/test-project')
})
```

### 6. Adding Tests for New Features

- Add test to `e2e/cypress/e2e/vteam.cy.ts`
- Ensure auth header is automatically added (no manual setup needed)
- Use `cy.visit()`, `cy.contains()`, `cy.get()` for UI interactions
- Use `cy.request()` for direct API testing
- Run locally first: `cd e2e && npm run test:headed`

## When to Add E2E Tests

✅ **DO write E2E tests for:**
- New critical user workflows (project creation, session management)
- Multi-component integrations (frontend → backend → operator)
- Breaking changes to core flows

❌ **DON'T write E2E tests for:**
- Unit-testable logic (use unit tests instead)
- Internal implementation details

## E2E Test Writing Rules

### 1. Use Descriptive Test Names

Test names should describe user actions and expected results, not technical details.

### 2. Use Data Attributes for Selectors

Use stable `data-testid` selectors, not CSS classes or element positions.

### 3. Wait for Conditions

Wait for actual conditions, not fixed timeouts like `cy.wait(3000)`.

### 4. Test User Workflows

Test from user perspective, not implementation details or API internals.

### 5. Auth Headers Automatic

Don't manually add auth headers - they're auto-injected in vTeam.

### 6. Use Unique Test Data

Use timestamps or UUIDs, not hardcoded names.

### 7. Follow Arrange-Act-Assert Pattern

Structure tests clearly with setup, action, and verification phases.

## Example of a Well-Written E2E Test

```typescript
it('should create a new project when user fills form and clicks submit', () => {
  // Arrange: Navigate to form
  cy.visit('/projects/new')

  // Act: Fill unique data and submit
  const projectName = `test-${Date.now()}`
  cy.get('[data-testid="project-name-input"]').type(projectName)
  cy.get('[data-testid="create-project-btn"]').click()

  // Assert: Verify success
  cy.contains('Loading...').should('not.exist')
  cy.url().should('include', `/projects/${projectName}`)
})
```

## Common E2E Mistakes to Avoid

- ❌ Testing implementation details instead of user workflows
- ❌ Using fragile CSS selectors instead of data-testid
- ❌ Fixed waits (`cy.wait(3000)`) instead of conditional waits
- ❌ Manually adding auth headers (automatic in vTeam e2e)
- ❌ Not cleaning up test data
- ❌ Hardcoded test data causing conflicts
- ❌ Tests that depend on execution order
- ❌ Missing assertions (test passes but doesn't verify anything)

## Pre-Commit Checklist for E2E Tests

Before committing e2e test changes:

- [ ] Tests pass locally: `make e2e-test`
- [ ] Test names describe user actions and outcomes
- [ ] Used `data-testid` selectors (not CSS classes)
- [ ] No fixed waits (`cy.wait(3000)`), only conditional waits
- [ ] No manual auth headers (automatic via interceptor)
- [ ] Used unique test data (timestamps, UUIDs)
- [ ] Tests are independent (no execution order dependency)
- [ ] All assertions present and meaningful
- [ ] Video shows expected behavior
- [ ] Added data-testid to components if needed
- [ ] Updated `e2e/README.md` if adding new test categories
- [ ] Ran with UI to verify: `npm run test:headed`

## Run Before Committing

```bash
# Test locally
make e2e-test CONTAINER_ENGINE=podman

# Verify video
open e2e/cypress/videos/vteam.cy.ts.mp4

# Check for console errors
# Review screenshots if any tests failed
```

## Troubleshooting E2E Failures

### View Pod Logs

```bash
kubectl logs -n ambient-code -l app=frontend
kubectl logs -n ambient-code -l app=backend-api
```

### Check Ingress

```bash
kubectl get ingress -n ambient-code
kubectl describe ingress frontend-ingress -n ambient-code
```

### Test Manually

```bash
curl http://vteam.local:8080/api/cluster-info
```

### Run with UI for Debugging

```bash
cd e2e
source .env.test
CYPRESS_TEST_TOKEN="$TEST_TOKEN" CYPRESS_BASE_URL="$CYPRESS_BASE_URL" npm run test:headed
```

## CI/CD Integration

Tests run automatically on all PRs via GitHub Actions (`.github/workflows/e2e.yml`).

### Constitution Alignment (Principle IV: Test-Driven Development)

- ✅ **E2E Tests for Critical Journeys**: Project creation workflow is core user journey
- ✅ **CI/CD Enforcement**: GitHub Actions runs e2e tests on all PRs
- ✅ **Tests Must Pass**: PR merge blocked if tests fail
- 📋 **Future**: Add session creation and execution tests (requires API key setup)
