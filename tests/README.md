# Ambient Code Platform - Test Suite

This directory contains tests for the Ambient Code Platform, with a focus on validating the local developer experience.

## Test Categories

### Local Developer Experience Tests

**File:** `local-dev-test.sh`

Comprehensive integration test suite that validates the complete local development environment.

**What it tests:**
- ✅ Prerequisites (make, kubectl, minikube, podman/docker)
- ✅ Makefile commands and syntax
- ✅ Minikube cluster status
- ✅ Kubernetes configuration
- ✅ Namespace and CRDs
- ✅ Pod health and readiness
- ✅ Service configuration
- ✅ Ingress setup
- ✅ Backend health endpoints
- ✅ Frontend accessibility
- ✅ RBAC configuration
- ✅ Build and reload commands
- ✅ Logging functionality
- ✅ Storage configuration
- ✅ Environment variables
- ✅ Resource limits
- ✅ Ingress controller

**49 tests total**

## Running Tests

### All Tests - One Command (~35 seconds)

Run everything with a single command:

```bash
make test-all
```

This runs:
1. Quick smoke test (5 tests)
2. Comprehensive test suite (49 tests)

**Total: 54 tests**

### Quick Smoke Test (5 seconds)

Run a fast validation of the essential components:

```bash
make local-test-quick
```

Tests:
- Minikube running
- Namespace exists
- Pods running
- Backend healthy
- Frontend accessible

### Full Test Suite (~30 seconds)

Run all 49 tests:

```bash
make local-test-dev
```

Or run directly:

```bash
./tests/local-dev-test.sh
```

### Test Options

The comprehensive test suite supports several options:

```bash
# Skip initial setup
./tests/local-dev-test.sh --skip-setup

# Clean up after tests
./tests/local-dev-test.sh --cleanup

# Verbose output
./tests/local-dev-test.sh --verbose

# Show help
./tests/local-dev-test.sh --help
```

## Typical Usage

### Run All Tests

One command to run everything:

```bash
make test-all
```

### Before Starting Work

Validate your environment is ready:

```bash
make local-test-quick
```

### After Making Changes

Verify everything still works:

```bash
make test-all
```

Or just the comprehensive suite:

```bash
make local-test-dev
```

### In CI/CD Pipeline

```bash
# Start environment
make local-up

# Wait for pods to be ready
sleep 30

# Run tests
make local-test-dev

# Cleanup
make local-down
```

## Test Output

### Success

```
═══════════════════════════════════════════
  Test Summary
═══════════════════════════════════════════

Results:
  Passed: 49
  Failed: 0
  Total:  49

✓ All tests passed!

ℹ Your local development environment is ready!
ℹ Access the application:
ℹ   • Frontend: http://192.168.64.4:30030
ℹ   • Backend:  http://192.168.64.4:30080
```

Exit code: 0

### Failure

```
═══════════════════════════════════════════
  Test Summary
═══════════════════════════════════════════

Results:
  Passed: 45
  Failed: 4
  Total:  49

✗ Some tests failed

✗ Your local development environment has issues
ℹ Run 'make local-troubleshoot' for more details
```

Exit code: 1

## Understanding Test Results

### Color Coding

- 🔵 **Blue (ℹ)** - Information
- 🟢 **Green (✓)** - Test passed
- 🔴 **Red (✗)** - Test failed
- 🟡 **Yellow (⚠)** - Warning (non-critical)

### Common Failures

#### "Minikube not running"
```bash
make local-up
```

#### "Namespace missing"
```bash
kubectl create namespace ambient-code
```

#### "Pods not running"
```bash
make local-status
make local-troubleshoot
```

#### "Backend not responding"
```bash
make local-logs-backend
make local-reload-backend
```

## Writing New Tests

### Test Structure

```bash
test_my_feature() {
    log_section "Test X: My Feature"
    
    # Test logic here
    if condition; then
        log_success "Feature works"
        ((PASSED_TESTS++))
    else
        log_error "Feature broken"
        ((FAILED_TESTS++))
    fi
}
```

### Available Assertions

```bash
assert_command_exists "command"              # Check if command exists
assert_equals "expected" "actual" "desc"     # Check equality
assert_contains "haystack" "needle" "desc"   # Check substring
assert_http_ok "url" "desc" [retries]        # Check HTTP endpoint
assert_pod_running "label" "desc"            # Check pod status
```

### Adding Tests

1. Create a new test function in `local-dev-test.sh`
2. Add it to the `main()` function
3. Update the test count in this README
4. Document what it tests

## Integration with Makefile

The Makefile provides convenient shortcuts:

```makefile
# Quick smoke test (5 seconds)
make local-test-quick

# Full test suite (30 seconds)
make local-test-dev

# Backward compatibility
make local-test    # → local-test-quick
```

## Future Test Categories

Planned additions:

### Unit Tests
- Backend Go code tests
- Frontend React component tests
- Utility function tests

### Contract Tests
- API contract validation
- CRD schema validation
- Service interface tests

### Integration Tests
- Multi-component workflows
- End-to-end scenarios
- Session creation and execution

### Performance Tests
- Load testing
- Resource usage
- Startup time

## Contributing

When adding features to the local development environment:

1. **Update tests** - Add tests for new commands or features
2. **Run tests** - Ensure `make local-test-dev` passes
3. **Document** - Update this README if adding new test categories

## Troubleshooting

### Tests fail on fresh environment

Wait for pods to be ready before running tests:

```bash
make local-up
sleep 30
make local-test-dev
```

### Tests pass but application doesn't work

Run troubleshooting:

```bash
make local-troubleshoot
```

### Tests are slow

Use quick smoke test for rapid validation:

```bash
make local-test-quick
```

### Need to debug a test

Run with verbose output:

```bash
./tests/local-dev-test.sh --verbose
```

## Test Maintenance

### Regular Testing Schedule

- **Before every commit** - `make local-test-quick`
- **Before every PR** - `make local-test-dev`
- **Weekly** - Full cleanup and restart
  ```bash
  make local-clean
  make local-up
  make local-test-dev
  ```

### Keeping Tests Up to Date

When you:
- Add new Makefile commands → Add tests
- Change component names → Update test expectations
- Modify deployments → Update pod/service tests
- Update RBAC → Update permission tests

## Support

If tests are failing and you need help:

1. Check the output for specific failures
2. Run `make local-troubleshoot`
3. Check pod logs: `make local-logs`
4. Review the test source: `tests/local-dev-test.sh`
5. Ask the team in Slack

## Links

- [Makefile](../Makefile) - Developer commands
- [Local Development Guide](../docs/LOCAL_DEVELOPMENT.md) - Setup instructions
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guidelines

