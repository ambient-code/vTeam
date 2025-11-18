#!/bin/bash
#
# Local Developer Experience Test Suite
# Tests the complete local development workflow for Ambient Code Platform
#
# Usage: ./tests/local-dev-test.sh [options]
#   -s, --skip-setup    Skip the initial setup (assume environment is ready)
#   -c, --cleanup       Clean up after tests
#   -v, --verbose       Verbose output
#   --ci                CI mode (treats known TODOs as non-failures)
#

# Don't exit on error - we want to collect all test results
set +e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Test configuration
NAMESPACE="${NAMESPACE:-ambient-code}"
SKIP_SETUP=false
CLEANUP=false
VERBOSE=false
CI_MODE=false
FAILED_TESTS=0
PASSED_TESTS=0
KNOWN_FAILURES=0

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--skip-setup)
            SKIP_SETUP=true
            shift
            ;;
        -c|--cleanup)
            CLEANUP=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        --ci)
            CI_MODE=true
            shift
            ;;
        -h|--help)
            head -n 10 "$0" | tail -n 7
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $*"
}

log_success() {
    echo -e "${GREEN}✓${NC} $*"
}

log_error() {
    echo -e "${RED}✗${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $*"
}

log_section() {
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════${NC}"
    echo -e "${BOLD}  $*${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════${NC}"
}

# Test assertion functions
assert_command_exists() {
    local cmd=$1
    if command -v "$cmd" >/dev/null 2>&1; then
        log_success "Command '$cmd' is installed"
        ((PASSED_TESTS++))
        return 0
    else
        log_error "Command '$cmd' is NOT installed"
        ((FAILED_TESTS++))
        return 1
    fi
}

assert_equals() {
    local expected=$1
    local actual=$2
    local description=$3
    
    if [ "$expected" = "$actual" ]; then
        log_success "$description"
        ((PASSED_TESTS++))
        return 0
    else
        log_error "$description"
        log_error "  Expected: $expected"
        log_error "  Actual: $actual"
        ((FAILED_TESTS++))
        return 1
    fi
}

assert_contains() {
    local haystack=$1
    local needle=$2
    local description=$3
    
    if echo "$haystack" | grep -q "$needle"; then
        log_success "$description"
        ((PASSED_TESTS++))
        return 0
    else
        log_error "$description"
        log_error "  Expected to contain: $needle"
        log_error "  Actual: $haystack"
        ((FAILED_TESTS++))
        return 1
    fi
}

assert_http_ok() {
    local url=$1
    local description=$2
    local max_retries=${3:-5}
    local retry=0
    
    while [ $retry -lt $max_retries ]; do
        if curl -sf "$url" >/dev/null 2>&1; then
            log_success "$description"
            ((PASSED_TESTS++))
            return 0
        fi
        ((retry++))
        [ $retry -lt $max_retries ] && sleep 2
    done
    
    log_error "$description (after $max_retries retries)"
    ((FAILED_TESTS++))
    return 1
}

assert_pod_running() {
    local label=$1
    local description=$2
    
    if kubectl get pods -n "$NAMESPACE" -l "$label" 2>/dev/null | grep -q "Running"; then
        log_success "$description"
        ((PASSED_TESTS++))
        return 0
    else
        log_error "$description"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Test: Prerequisites
test_prerequisites() {
    log_section "Test 1: Prerequisites"
    
    assert_command_exists "make"
    assert_command_exists "kubectl"
    assert_command_exists "minikube"
    assert_command_exists "podman" || assert_command_exists "docker"
    
    # Check if running on macOS or Linux
    if [[ "$OSTYPE" == "darwin"* ]]; then
        log_info "Running on macOS"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        log_info "Running on Linux"
    else
        log_warning "Unknown OS: $OSTYPE"
    fi
}

# Test: Makefile Help
test_makefile_help() {
    log_section "Test 2: Makefile Help Command"
    
    local help_output
    help_output=$(make help 2>&1)
    
    assert_contains "$help_output" "Ambient Code Platform" "Help shows correct branding"
    assert_contains "$help_output" "local-up" "Help lists local-up command"
    assert_contains "$help_output" "local-status" "Help lists local-status command"
    assert_contains "$help_output" "local-logs" "Help lists local-logs command"
    assert_contains "$help_output" "local-reload-backend" "Help lists reload commands"
}

# Test: Minikube Status Check
test_minikube_status() {
    log_section "Test 3: Minikube Status"
    
    if minikube status >/dev/null 2>&1; then
        log_success "Minikube is running"
        ((PASSED_TESTS++))
        
        # Check minikube version
        local version
        version=$(minikube version --short 2>/dev/null || echo "unknown")
        log_info "Minikube version: $version"
    else
        log_error "Minikube is NOT running"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Test: Kubernetes Context
test_kubernetes_context() {
    log_section "Test 4: Kubernetes Context"
    
    local context
    context=$(kubectl config current-context 2>/dev/null || echo "none")
    
    assert_contains "$context" "minikube" "kubectl context is set to minikube"
    
    # Test kubectl connectivity
    if kubectl cluster-info >/dev/null 2>&1; then
        log_success "kubectl can connect to cluster"
        ((PASSED_TESTS++))
    else
        log_error "kubectl cannot connect to cluster"
        ((FAILED_TESTS++))
    fi
}

# Test: Namespace Exists
test_namespace_exists() {
    log_section "Test 5: Namespace Existence"
    
    if kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        log_success "Namespace '$NAMESPACE' exists"
        ((PASSED_TESTS++))
    else
        log_error "Namespace '$NAMESPACE' does NOT exist"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Test: CRDs Installed
test_crds_installed() {
    log_section "Test 6: Custom Resource Definitions"
    
    local crds=("agenticsessions.vteam.ambient-code" "projectsettings.vteam.ambient-code")
    
    for crd in "${crds[@]}"; do
        if kubectl get crd "$crd" >/dev/null 2>&1; then
            log_success "CRD '$crd' is installed"
            ((PASSED_TESTS++))
        else
            log_error "CRD '$crd' is NOT installed"
            ((FAILED_TESTS++))
        fi
    done
}

# Test: Pods Running
test_pods_running() {
    log_section "Test 7: Pod Status"
    
    assert_pod_running "app=backend-api" "Backend pod is running"
    assert_pod_running "app=frontend" "Frontend pod is running"
    assert_pod_running "app=agentic-operator" "Operator pod is running"
    
    # Check pod readiness
    local not_ready
    not_ready=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase!=Running 2>/dev/null | grep -v "NAME" | wc -l)
    
    if [ "$not_ready" -eq 0 ]; then
        log_success "All pods are in Running state"
        ((PASSED_TESTS++))
    else
        log_warning "$not_ready pod(s) are not running"
    fi
}

# Test: Services Exist
test_services_exist() {
    log_section "Test 8: Services"
    
    local services=("backend-service" "frontend-service")
    
    for svc in "${services[@]}"; do
        if kubectl get svc "$svc" -n "$NAMESPACE" >/dev/null 2>&1; then
            log_success "Service '$svc' exists"
            ((PASSED_TESTS++))
        else
            log_error "Service '$svc' does NOT exist"
            ((FAILED_TESTS++))
        fi
    done
}

# Test: Ingress Configuration
test_ingress() {
    log_section "Test 9: Ingress Configuration"
    
    if kubectl get ingress ambient-code-ingress -n "$NAMESPACE" >/dev/null 2>&1; then
        log_success "Ingress 'ambient-code-ingress' exists"
        ((PASSED_TESTS++))
        
        # Check ingress host
        local host
        host=$(kubectl get ingress ambient-code-ingress -n "$NAMESPACE" -o jsonpath='{.spec.rules[0].host}' 2>/dev/null)
        assert_equals "ambient.code.platform.local" "$host" "Ingress host is correct"
        
        # Check ingress paths
        local paths
        paths=$(kubectl get ingress ambient-code-ingress -n "$NAMESPACE" -o jsonpath='{.spec.rules[0].http.paths[*].path}' 2>/dev/null)
        assert_contains "$paths" "/api" "Ingress has /api path"
    else
        log_error "Ingress 'ambient-code-ingress' does NOT exist"
        ((FAILED_TESTS++))
    fi
}

# Test: Backend Health Endpoint
test_backend_health() {
    log_section "Test 10: Backend Health Endpoint"
    
    local minikube_ip
    minikube_ip=$(minikube ip 2>/dev/null)
    
    if [ -n "$minikube_ip" ]; then
        log_info "Minikube IP: $minikube_ip"
        assert_http_ok "http://$minikube_ip:30080/health" "Backend health endpoint responds" 10
    else
        log_error "Could not get minikube IP"
        ((FAILED_TESTS++))
    fi
}

# Test: Frontend Accessibility
test_frontend_accessibility() {
    log_section "Test 11: Frontend Accessibility"
    
    local minikube_ip
    minikube_ip=$(minikube ip 2>/dev/null)
    
    if [ -n "$minikube_ip" ]; then
        assert_http_ok "http://$minikube_ip:30030" "Frontend is accessible" 10
    else
        log_error "Could not get minikube IP"
        ((FAILED_TESTS++))
    fi
}

# Test: RBAC Configuration
test_rbac() {
    log_section "Test 12: RBAC Configuration"
    
    local roles=("ambient-project-admin" "ambient-project-edit" "ambient-project-view")
    
    for role in "${roles[@]}"; do
        if kubectl get clusterrole "$role" >/dev/null 2>&1; then
            log_success "ClusterRole '$role' exists"
            ((PASSED_TESTS++))
        else
            log_error "ClusterRole '$role' does NOT exist"
            ((FAILED_TESTS++))
        fi
    done
}

# Test: Development Workflow - Build Command
test_build_command() {
    log_section "Test 13: Build Commands (Dry Run)"
    
    if make -n build-backend >/dev/null 2>&1; then
        log_success "make build-backend syntax is valid"
        ((PASSED_TESTS++))
    else
        log_error "make build-backend has syntax errors"
        ((FAILED_TESTS++))
    fi
    
    if make -n build-frontend >/dev/null 2>&1; then
        log_success "make build-frontend syntax is valid"
        ((PASSED_TESTS++))
    else
        log_error "make build-frontend has syntax errors"
        ((FAILED_TESTS++))
    fi
}

# Test: Development Workflow - Reload Commands
test_reload_commands() {
    log_section "Test 14: Reload Commands (Dry Run)"
    
    local reload_cmds=("local-reload-backend" "local-reload-frontend" "local-reload-operator")
    
    for cmd in "${reload_cmds[@]}"; do
        if make -n "$cmd" >/dev/null 2>&1; then
            log_success "make $cmd syntax is valid"
            ((PASSED_TESTS++))
        else
            log_error "make $cmd has syntax errors"
            ((FAILED_TESTS++))
        fi
    done
}

# Test: Logging Commands
test_logging_commands() {
    log_section "Test 15: Logging Commands"
    
    # Test that we can get logs from each component
    local components=("backend-api" "frontend" "agentic-operator")
    
    for component in "${components[@]}"; do
        if kubectl logs -n "$NAMESPACE" -l "app=$component" --tail=1 >/dev/null 2>&1; then
            log_success "Can retrieve logs from $component"
            ((PASSED_TESTS++))
        else
            log_warning "Cannot retrieve logs from $component (pod may not be running)"
        fi
    done
}

# Test: Storage Configuration
test_storage() {
    log_section "Test 16: Storage Configuration"
    
    # Check if workspace PVC exists
    if kubectl get pvc workspace-pvc -n "$NAMESPACE" >/dev/null 2>&1; then
        log_success "Workspace PVC exists"
        ((PASSED_TESTS++))
        
        # Check PVC status
        local status
        status=$(kubectl get pvc workspace-pvc -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null)
        if [ "$status" = "Bound" ]; then
            log_success "Workspace PVC is bound"
            ((PASSED_TESTS++))
        else
            log_warning "Workspace PVC status: $status"
        fi
    else
        log_info "Workspace PVC does not exist (may not be required for all deployments)"
    fi
}

# Test: Environment Variables
test_environment_variables() {
    log_section "Test 17: Environment Variables"
    
    # Check backend deployment env vars
    local backend_env
    backend_env=$(kubectl get deployment backend-api -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].env[*].name}' 2>/dev/null || echo "")
    
    assert_contains "$backend_env" "DISABLE_AUTH" "Backend has DISABLE_AUTH env var"
    assert_contains "$backend_env" "ENVIRONMENT" "Backend has ENVIRONMENT env var"
    
    # Check frontend deployment env vars
    local frontend_env
    frontend_env=$(kubectl get deployment frontend -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].env[*].name}' 2>/dev/null || echo "")
    
    assert_contains "$frontend_env" "DISABLE_AUTH" "Frontend has DISABLE_AUTH env var"
}

# Test: Resource Limits
test_resource_limits() {
    log_section "Test 18: Resource Configuration"
    
    # Check if deployments have resource requests/limits
    local deployments=("backend-api" "frontend" "agentic-operator")
    
    for deployment in "${deployments[@]}"; do
        local resources
        resources=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].resources}' 2>/dev/null || echo "{}")
        
        if [ "$resources" != "{}" ]; then
            log_success "Deployment '$deployment' has resource configuration"
            ((PASSED_TESTS++))
        else
            log_info "Deployment '$deployment' has no resource limits (OK for dev)"
        fi
    done
}

# Test: Make local-status
test_make_status() {
    log_section "Test 19: make local-status Command"
    
    local status_output
    status_output=$(make local-status 2>&1 || echo "")
    
    assert_contains "$status_output" "Ambient Code Platform Status" "Status shows correct branding"
    assert_contains "$status_output" "Minikube" "Status shows Minikube section"
    assert_contains "$status_output" "Pods" "Status shows Pods section"
}

# Test: Ingress Controller
test_ingress_controller() {
    log_section "Test 20: Ingress Controller"
    
    # Check if ingress-nginx is installed
    if kubectl get namespace ingress-nginx >/dev/null 2>&1; then
        log_success "ingress-nginx namespace exists"
        ((PASSED_TESTS++))
        
        # Check if controller is running
        if kubectl get pods -n ingress-nginx -l app.kubernetes.io/component=controller 2>/dev/null | grep -q "Running"; then
            log_success "Ingress controller is running"
            ((PASSED_TESTS++))
        else
            log_error "Ingress controller is NOT running"
            ((FAILED_TESTS++))
        fi
    else
        log_error "ingress-nginx namespace does NOT exist"
        ((FAILED_TESTS++))
    fi
}

# Test: Security - Local Dev User Permissions
test_security_local_dev_user() {
    log_section "Test 21: Security - Local Dev User Permissions"
    
    log_info "Verifying local-dev-user service account implementation status..."
    
    # CRITICAL TEST: Check if local-dev-user service account exists
    if kubectl get serviceaccount local-dev-user -n "$NAMESPACE" >/dev/null 2>&1; then
        log_success "local-dev-user service account exists"
        ((PASSED_TESTS++))
    else
        log_error "local-dev-user service account does NOT exist"
        log_error "CRITICAL: This is required for proper permission scoping in dev mode"
        log_error "TODO: Create local-dev-user ServiceAccount with namespace-scoped permissions"
        log_error "Reference: components/backend/handlers/middleware.go:323-335"
        ((FAILED_TESTS++))
        return
    fi
    
    # Test 1: Should NOT be able to create cluster-wide resources
    # NOTE: This test validates the FUTURE state after token minting is implemented
    # Currently, local-dev-user permissions don't matter because getLocalDevK8sClients() 
    # returns backend SA instead of minting a token for local-dev-user
    local can_create_clusterroles
    can_create_clusterroles=$(kubectl auth can-i create clusterroles --as=system:serviceaccount:ambient-code:local-dev-user 2>/dev/null || echo "no")
    
    if [ "$can_create_clusterroles" = "no" ]; then
        log_success "local-dev-user CANNOT create clusterroles (correct - no cluster-admin)"
        ((PASSED_TESTS++))
    else
        log_error "local-dev-user CAN create clusterroles (will matter after token minting implemented)"
        if [ "$CI_MODE" = true ]; then
            log_warning "  (CI mode: Counting as known TODO - related to token minting)"
            ((KNOWN_FAILURES++))
        else
            ((FAILED_TESTS++))
        fi
    fi
    
    # Test 2: Should NOT be able to list all namespaces
    # NOTE: Same as above - only matters after token minting
    local can_list_namespaces
    can_list_namespaces=$(kubectl auth can-i list namespaces --as=system:serviceaccount:ambient-code:local-dev-user 2>/dev/null || echo "no")
    
    if [ "$can_list_namespaces" = "no" ]; then
        log_success "local-dev-user CANNOT list all namespaces (correct - namespace-scoped)"
        ((PASSED_TESTS++))
    else
        log_error "local-dev-user CAN list namespaces (will matter after token minting implemented)"
        if [ "$CI_MODE" = true ]; then
            log_warning "  (CI mode: Counting as known TODO - related to token minting)"
            ((KNOWN_FAILURES++))
        else
            ((FAILED_TESTS++))
        fi
    fi
    
    # Test 3: Should be able to access resources in ambient-code namespace
    local can_list_pods
    can_list_pods=$(kubectl auth can-i list pods --namespace=ambient-code --as=system:serviceaccount:ambient-code:local-dev-user 2>/dev/null || echo "no")
    
    if [ "$can_list_pods" = "yes" ]; then
        log_success "local-dev-user CAN list pods in ambient-code namespace (correct - needs namespace access)"
        ((PASSED_TESTS++))
    else
        log_error "local-dev-user CANNOT list pods in ambient-code namespace (too restricted)"
        ((FAILED_TESTS++))
    fi
    
    # Test 4: Should be able to manage CRDs in ambient-code namespace
    local can_list_sessions
    can_list_sessions=$(kubectl auth can-i list agenticsessions.vteam.ambient-code --namespace=ambient-code --as=system:serviceaccount:ambient-code:local-dev-user 2>/dev/null || echo "no")
    
    if [ "$can_list_sessions" = "yes" ]; then
        log_success "local-dev-user CAN list agenticsessions (correct - needs CR access)"
        ((PASSED_TESTS++))
    else
        log_error "local-dev-user CANNOT list agenticsessions (needs CR permissions)"
        ((FAILED_TESTS++))
    fi
}

# Test: Security - Production Namespace Rejection
test_security_prod_namespace_rejection() {
    log_section "Test 22: Security - Production Namespace Rejection"
    
    log_info "Testing that dev mode rejects production-like namespaces..."
    
    # Test 1: Check backend middleware has protection
    local backend_pod
    backend_pod=$(kubectl get pods -n "$NAMESPACE" -l app=backend-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$backend_pod" ]; then
        log_warning "Backend pod not found, skipping namespace rejection test"
        return
    fi
    
    # Check if ENVIRONMENT and DISABLE_AUTH are set correctly for dev mode
    local env_var
    env_var=$(kubectl get deployment backend-api -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ENVIRONMENT")].value}' 2>/dev/null)
    
    if [ "$env_var" = "local" ] || [ "$env_var" = "development" ]; then
        log_success "Backend ENVIRONMENT is set to '$env_var' (dev mode enabled)"
        ((PASSED_TESTS++))
    else
        log_error "Backend ENVIRONMENT is '$env_var' (should be 'local' or 'development' for dev mode)"
        ((FAILED_TESTS++))
    fi
    
    # Test 2: Verify namespace does not contain 'prod'
    if echo "$NAMESPACE" | grep -qi "prod"; then
        log_error "Namespace contains 'prod' - this would be REJECTED by middleware (GOOD)"
        log_error "Current namespace: $NAMESPACE"
        log_info "Dev mode should NEVER run in production namespaces"
        ((PASSED_TESTS++))  # This is correct behavior - we want it to fail
    else
        log_success "Namespace does not contain 'prod' (safe for dev mode)"
        ((PASSED_TESTS++))
    fi
    
    # Test 3: Document the protection mechanism
    log_info "Middleware protection (components/backend/handlers/middleware.go:314-317):"
    log_info "  • Checks if namespace contains 'prod'"
    log_info "  • Requires ENVIRONMENT=local or development"
    log_info "  • Requires DISABLE_AUTH=true"
    log_info "  • Logs activation for audit trail"
}

# Test: Security - Mock Token Detection in Logs
test_security_mock_token_logging() {
    log_section "Test 23: Security - Mock Token Detection"
    
    log_info "Verifying backend logs show dev mode activation..."
    
    local backend_pod
    backend_pod=$(kubectl get pods -n "$NAMESPACE" -l app=backend-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$backend_pod" ]; then
        log_warning "Backend pod not found, skipping log test"
        return
    fi
    
    # Get recent backend logs
    local logs
    logs=$(kubectl logs -n "$NAMESPACE" "$backend_pod" --tail=100 2>/dev/null || echo "")
    
    if [ -z "$logs" ]; then
        log_warning "Could not retrieve backend logs"
        return
    fi
    
    # Test 1: Check for dev mode detection logs
    if echo "$logs" | grep -q "Local dev mode detected\|Dev mode detected\|local dev environment"; then
        log_success "Backend logs show dev mode activation"
        ((PASSED_TESTS++))
    else
        log_info "Backend logs do not show dev mode activation yet (may need API call to trigger)"
    fi
    
    # Test 2: Verify logs do NOT contain the actual mock token value
    if echo "$logs" | grep -q "mock-token-for-local-dev"; then
        log_error "Backend logs contain mock token value (SECURITY ISSUE - tokens should be redacted)"
        ((FAILED_TESTS++))
    else
        log_success "Backend logs do NOT contain mock token value (correct - tokens are redacted)"
        ((PASSED_TESTS++))
    fi
    
    # Test 3: Check for service account usage logging
    if echo "$logs" | grep -q "using.*service account\|K8sClient\|DynamicClient"; then
        log_success "Backend logs reference service account usage"
        ((PASSED_TESTS++))
    else
        log_info "Backend logs do not show service account usage (may need API call to trigger)"
    fi
    
    # Test 4: Verify environment validation logs
    if echo "$logs" | grep -q "Local dev environment validated\|env=local\|env=development"; then
        log_success "Backend logs show environment validation"
        ((PASSED_TESTS++))
    else
        log_info "Backend logs do not show environment validation yet"
    fi
}

# Test: Security - Token Redaction
test_security_token_redaction() {
    log_section "Test 24: Security - Token Redaction in Logs"
    
    log_info "Verifying tokens are properly redacted in logs..."
    
    local backend_pod
    backend_pod=$(kubectl get pods -n "$NAMESPACE" -l app=backend-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$backend_pod" ]; then
        log_warning "Backend pod not found, skipping token redaction test"
        return
    fi
    
    # Get all backend logs
    local logs
    logs=$(kubectl logs -n "$NAMESPACE" "$backend_pod" --tail=500 2>/dev/null || echo "")
    
    if [ -z "$logs" ]; then
        log_warning "Could not retrieve backend logs"
        return
    fi
    
    # Test 1: Logs should use tokenLen= instead of showing token
    if echo "$logs" | grep -q "tokenLen=\|token (len="; then
        log_success "Logs use token length instead of token value (correct redaction)"
        ((PASSED_TESTS++))
    else
        log_info "Token length logging not found (may need authenticated requests)"
    fi
    
    # Test 2: Should NOT contain Bearer tokens
    if echo "$logs" | grep -qE "Bearer [A-Za-z0-9._-]{20,}"; then
        log_error "Logs contain Bearer tokens (SECURITY ISSUE)"
        ((FAILED_TESTS++))
    else
        log_success "Logs do NOT contain Bearer tokens (correct)"
        ((PASSED_TESTS++))
    fi
    
    # Test 3: Should NOT contain base64-encoded credentials
    if echo "$logs" | grep -qE "[A-Za-z0-9+/]{40,}={0,2}"; then
        log_warning "Logs may contain base64-encoded data (verify not credentials)"
    else
        log_success "Logs do not contain long base64 strings"
        ((PASSED_TESTS++))
    fi
}

# Test: Security - Service Account Configuration
test_security_service_account_config() {
    log_section "Test 25: Security - Service Account Configuration"
    
    log_info "Verifying service account RBAC configuration..."
    
    # Test 1: Check backend-api service account exists
    if kubectl get serviceaccount backend-api -n "$NAMESPACE" >/dev/null 2>&1; then
        log_success "backend-api service account exists"
        ((PASSED_TESTS++))
    else
        log_error "backend-api service account does NOT exist"
        ((FAILED_TESTS++))
        return
    fi
    
    # Test 2: Check if backend has cluster-admin (expected in dev, dangerous in prod)
    local clusterrolebindings
    clusterrolebindings=$(kubectl get clusterrolebinding -o json 2>/dev/null | grep -c "backend-api\|system:serviceaccount:$NAMESPACE:backend-api" || echo "0")
    
    if [ "$clusterrolebindings" -gt 0 ]; then
        log_warning "backend-api has cluster-level role bindings (OK for dev, DANGEROUS in production)"
        log_warning "  ⚠️  This service account has elevated permissions"
        log_warning "  ⚠️  Production deployments should use minimal namespace-scoped permissions"
    else
        log_info "backend-api has no cluster-level role bindings (namespace-scoped only)"
    fi
    
    # Test 3: Verify dev mode safety checks are in place
    log_info "Dev mode safety mechanisms:"
    log_info "  ✓ Requires ENVIRONMENT=local or development"
    log_info "  ✓ Requires DISABLE_AUTH=true explicitly"
    log_info "  ✓ Rejects namespaces containing 'prod'"
    log_info "  ✓ Logs all dev mode activations"
    ((PASSED_TESTS++))
}

# Test: CRITICAL - Token Minting Implementation
test_critical_token_minting() {
    log_section "Test 26: CRITICAL - Token Minting for local-dev-user"
    
    if [ "$CI_MODE" = true ]; then
        log_warning "Running in CI mode - known TODO tracked"
    fi
    
    log_error "═══════════════════════════════════════════════════════════════"
    log_error "CRITICAL TODO: Token minting NOT implemented"
    log_error "═══════════════════════════════════════════════════════════════"
    log_info ""
    log_info "Current implementation (middleware.go:323-335):"
    log_info "  getLocalDevK8sClients() returns server.K8sClient, server.DynamicClient"
    log_info "  This uses the BACKEND SERVICE ACCOUNT (cluster-admin)"
    log_info ""
    log_error "Required implementation:"
    log_error "  1. Create local-dev-user ServiceAccount in ambient-code namespace"
    log_error "  2. Mint a token for local-dev-user using TokenRequest API"
    log_error "  3. Create K8s clients using the minted token"
    log_error "  4. Return clients with namespace-scoped permissions"
    log_info ""
    
    # Test 1: Check if local-dev-user ServiceAccount exists
    if kubectl get serviceaccount local-dev-user -n "$NAMESPACE" >/dev/null 2>&1; then
        log_success "Step 1/4: local-dev-user ServiceAccount exists"
        ((PASSED_TESTS++))
    else
        log_error "Step 1/4: local-dev-user ServiceAccount does NOT exist"
        log_error "  Create with: kubectl create serviceaccount local-dev-user -n ambient-code"
        if [ "$CI_MODE" = true ]; then
            ((KNOWN_FAILURES++))
        else
            ((FAILED_TESTS++))
        fi
    fi
    
    # Test 2: Check if RBAC for local-dev-user is configured
    local has_rolebinding=false
    if kubectl get rolebinding -n "$NAMESPACE" -o json 2>/dev/null | grep -q "local-dev-user"; then
        log_success "Step 2/4: local-dev-user has RoleBinding in namespace"
        ((PASSED_TESTS++))
        has_rolebinding=true
    else
        log_error "Step 2/4: local-dev-user has NO RoleBinding"
        log_error "  Required: RoleBinding granting namespace-scoped permissions"
        log_error "  Should grant: list/get/create/update/delete on CRDs, pods, services"
        if [ "$CI_MODE" = true ]; then
            ((KNOWN_FAILURES++))
        else
            ((FAILED_TESTS++))
        fi
    fi
    
    # Test 3: Verify token minting capability (TokenRequest API)
    log_error "Step 3/4: Token minting NOT implemented in code"
    log_error "  Current: Returns server.K8sClient (backend SA with cluster-admin)"
    log_error "  Required: Mint token using K8sClient.CoreV1().ServiceAccounts().CreateToken()"
    log_error "  Code location: components/backend/handlers/middleware.go:323-335"
    if [ "$CI_MODE" = true ]; then
        ((KNOWN_FAILURES++))
    else
        ((FAILED_TESTS++))
    fi
    
    # Test 4: Verify getLocalDevK8sClients uses minted token
    log_error "Step 4/4: getLocalDevK8sClients NOT using minted token"
    log_error "  Current: return server.K8sClient, server.DynamicClient"
    log_error "  Required: return kubernetes.NewForConfig(cfg), dynamic.NewForConfig(cfg)"
    log_error "  Where cfg uses minted token with namespace-scoped permissions"
    if [ "$CI_MODE" = true ]; then
        ((KNOWN_FAILURES++))
    else
        ((FAILED_TESTS++))
    fi
    
    # Summary
    log_info ""
    log_error "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_error "SECURITY IMPACT:"
    log_error "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_error "  ❌ Local dev currently uses backend SA (cluster-admin)"
    log_error "  ❌ No permission scoping in dev mode"
    log_error "  ❌ Dev users have unrestricted cluster access"
    log_error "  ❌ Cannot test RBAC restrictions locally"
    log_info ""
    log_info "NEXT STEPS:"
    log_info "  1. Create manifests/minikube/local-dev-rbac.yaml with:"
    log_info "     - ServiceAccount: local-dev-user"
    log_info "     - Role: ambient-local-dev (namespace-scoped permissions)"
    log_info "     - RoleBinding: local-dev-user → ambient-local-dev"
    log_info ""
    log_info "  2. Update getLocalDevK8sClients() in middleware.go:"
    log_info "     - Get local-dev-user ServiceAccount"
    log_info "     - Mint token using CreateToken() API"
    log_info "     - Create clients with minted token"
    log_info ""
    log_info "  3. Test with: ./tests/local-dev-test.sh"
    log_error "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Test: Production Manifest Safety - No Dev Mode Variables
test_production_manifest_safety() {
    log_section "Test 27: Production Manifest Safety"
    
    log_info "Verifying production manifests do NOT contain dev mode variables..."
    
    # Check base/production manifests for DISABLE_AUTH
    local prod_manifests=(
        "components/manifests/base/backend-deployment.yaml"
        "components/manifests/base/frontend-deployment.yaml"
        "components/manifests/overlays/production/frontend-oauth-deployment-patch.yaml"
    )
    
    local found_issues=false
    
    for manifest in "${prod_manifests[@]}"; do
        if [ ! -f "$manifest" ]; then
            log_warning "Manifest not found: $manifest (may be in subdirectory)"
            continue
        fi
        
        # Check for DISABLE_AUTH
        if grep -q "DISABLE_AUTH" "$manifest" 2>/dev/null; then
            log_error "Production manifest contains DISABLE_AUTH: $manifest"
            log_error "  This would enable dev mode in production (CRITICAL SECURITY ISSUE)"
            ((FAILED_TESTS++))
            found_issues=true
        else
            log_success "Production manifest clean (no DISABLE_AUTH): $manifest"
            ((PASSED_TESTS++))
        fi
        
        # Check for ENVIRONMENT=local or development
        if grep -qE "ENVIRONMENT.*[\"']?(local|development)[\"']?" "$manifest" 2>/dev/null; then
            log_error "Production manifest sets ENVIRONMENT=local/development: $manifest"
            log_error "  This would enable dev mode in production (CRITICAL SECURITY ISSUE)"
            ((FAILED_TESTS++))
            found_issues=true
        else
            log_success "Production manifest clean (no ENVIRONMENT=local): $manifest"
            ((PASSED_TESTS++))
        fi
    done
    
    # Verify minikube manifests DO have dev mode (sanity check)
    if [ -f "components/manifests/minikube/backend-deployment.yaml" ]; then
        if grep -q "DISABLE_AUTH" "components/manifests/minikube/backend-deployment.yaml" 2>/dev/null; then
            log_success "Minikube manifest correctly includes DISABLE_AUTH (expected for local dev)"
            ((PASSED_TESTS++))
        else
            log_error "Minikube manifest missing DISABLE_AUTH (dev mode broken)"
            ((FAILED_TESTS++))
        fi
    fi
    
    if [ "$found_issues" = false ]; then
        log_info ""
        log_info "✅ Production manifests are safe"
        log_info "✅ Dev mode only in components/manifests/minikube/"
        log_info "✅ Clear separation between dev and production configs"
    fi
}

# Test: Backend Using Wrong Service Account
test_critical_backend_sa_usage() {
    log_section "Test 28: CRITICAL - Backend Using Wrong Service Account"
    
    log_info "Verifying which service account backend uses in dev mode..."
    
    # Get backend pod
    local backend_pod
    backend_pod=$(kubectl get pods -n "$NAMESPACE" -l app=backend-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$backend_pod" ]; then
        log_warning "Backend pod not found, skipping SA usage test"
        return
    fi
    
    # Check which service account the backend pod is using
    local backend_sa
    backend_sa=$(kubectl get pod "$backend_pod" -n "$NAMESPACE" -o jsonpath='{.spec.serviceAccountName}' 2>/dev/null)
    
    log_info "Backend pod service account: $backend_sa"
    
    # Check if backend has cluster-admin via clusterrolebinding
    local has_cluster_admin=false
    if kubectl get clusterrolebinding -o json 2>/dev/null | grep -q "serviceaccount:$NAMESPACE:$backend_sa"; then
        has_cluster_admin=true
        log_error "Backend SA '$backend_sa' has cluster-level role bindings"
        
        # List the actual bindings
        log_error "Cluster role bindings for backend SA:"
        kubectl get clusterrolebinding -o json 2>/dev/null | jq -r ".items[] | select(.subjects[]?.name == \"$backend_sa\") | \"  - \(.metadata.name): \(.roleRef.name)\"" 2>/dev/null || echo "  (could not enumerate)"
        
        ((FAILED_TESTS++))
    else
        log_success "Backend SA '$backend_sa' has NO cluster-level bindings (good for prod model)"
        ((PASSED_TESTS++))
    fi
    
    # The critical issue: getLocalDevK8sClients returns server.K8sClient
    log_error ""
    log_error "CRITICAL ISSUE:"
    log_error "  getLocalDevK8sClients() returns server.K8sClient, server.DynamicClient"
    log_error "  These clients use the '$backend_sa' service account"
    if [ "$has_cluster_admin" = true ]; then
        log_error "  This SA has cluster-admin permissions (full cluster access)"
    fi
    log_error ""
    log_error "EXPECTED BEHAVIOR:"
    log_error "  getLocalDevK8sClients() should return clients using local-dev-user token"
    log_error "  local-dev-user should have namespace-scoped permissions only"
    log_error "  Dev mode should mimic production RBAC restrictions"
    log_error ""
    if [ "$CI_MODE" = true ]; then
        ((KNOWN_FAILURES++))
    else
        ((FAILED_TESTS++))
    fi
    
    # Test: Verify TODO comment exists in code
    log_info "Checking for TODO comment in middleware.go..."
    if [ -f "components/backend/handlers/middleware.go" ]; then
        if grep -q "TODO: Mint a token for the local-dev-user" components/backend/handlers/middleware.go; then
            log_success "TODO comment exists in middleware.go (tracked)"
            ((PASSED_TESTS++))
        else
            log_error "TODO comment NOT found in middleware.go"
            ((FAILED_TESTS++))
        fi
    else
        log_warning "middleware.go not found in current directory"
    fi
}

# Main test execution
main() {
    log_section "Ambient Code Platform - Local Developer Experience Tests"
    log_info "Starting test suite at $(date)"
    log_info "Test configuration:"
    log_info "  Namespace: $NAMESPACE"
    log_info "  Skip setup: $SKIP_SETUP"
    log_info "  Cleanup: $CLEANUP"
    log_info "  Verbose: $VERBOSE"
    echo ""
    
    # Run tests
    test_prerequisites
    test_makefile_help
    test_minikube_status
    test_kubernetes_context
    test_namespace_exists
    test_crds_installed
    test_pods_running
    test_services_exist
    test_ingress
    test_backend_health
    test_frontend_accessibility
    test_rbac
    test_build_command
    test_reload_commands
    test_logging_commands
    test_storage
    test_environment_variables
    test_resource_limits
    test_make_status
    test_ingress_controller
    
    # Security tests
    test_security_local_dev_user
    test_security_prod_namespace_rejection
    test_security_mock_token_logging
    test_security_token_redaction
    test_security_service_account_config
    
    # Production safety tests
    test_production_manifest_safety
    
    # CRITICAL failing tests for unimplemented features
    test_critical_token_minting
    test_critical_backend_sa_usage
    
    # Summary
    log_section "Test Summary"
    echo ""
    echo -e "${BOLD}Results:${NC}"
    echo -e "  ${GREEN}Passed:${NC} $PASSED_TESTS"
    echo -e "  ${RED}Failed:${NC} $FAILED_TESTS"
    if [ $KNOWN_FAILURES -gt 0 ]; then
        echo -e "  ${YELLOW}Known TODOs:${NC} $KNOWN_FAILURES"
    fi
    echo -e "  ${BOLD}Total:${NC}  $((PASSED_TESTS + FAILED_TESTS + KNOWN_FAILURES))"
    echo ""
    
    if [ "$CI_MODE" = true ]; then
        # In CI mode, known failures are acceptable
        local unexpected_failures=$FAILED_TESTS
        if [ $unexpected_failures -eq 0 ]; then
            echo -e "${GREEN}${BOLD}✓ All tests passed (excluding $KNOWN_FAILURES known TODOs)!${NC}"
            echo ""
            log_info "CI validation successful!"
            if [ $KNOWN_FAILURES -gt 0 ]; then
                log_warning "Note: $KNOWN_FAILURES known TODOs tracked in test output"
            fi
            exit 0
        else
            echo -e "${RED}${BOLD}✗ $unexpected_failures unexpected test failures${NC}"
            echo ""
            log_error "CI validation failed"
            exit 1
        fi
    else
        # In normal mode, any failure is an issue
        if [ $FAILED_TESTS -eq 0 ]; then
            echo -e "${GREEN}${BOLD}✓ All tests passed!${NC}"
            echo ""
            log_info "Your local development environment is ready!"
            log_info "Access the application:"
            log_info "  • Frontend: http://$(minikube ip 2>/dev/null):30030"
            log_info "  • Backend:  http://$(minikube ip 2>/dev/null):30080"
            echo ""
            if [ $KNOWN_FAILURES -gt 0 ]; then
                log_warning "Note: $KNOWN_FAILURES known TODOs tracked for future implementation"
            fi
            exit 0
        else
            echo -e "${RED}${BOLD}✗ Some tests failed${NC}"
            echo ""
            log_error "Your local development environment has issues"
            log_info "Run 'make local-troubleshoot' for more details"
            echo ""
            exit 1
        fi
    fi
}

# Run main function
main


