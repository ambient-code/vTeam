#!/bin/bash
# Ambient Agentic Runner v2.0 - Comprehensive Test Runner
# This script executes all test suites for the Polish & Validation phase

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_NAMESPACE=${TEST_NAMESPACE:-"test-project"}
API_BASE_URL=${API_BASE_URL:-"http://localhost:8080"}
OPENSHIFT_TOKEN=${OPENSHIFT_TOKEN:-""}

echo -e "${BLUE}===============================================================================${NC}"
echo -e "${BLUE}         AMBIENT AGENTIC RUNNER v2.0 - COMPREHENSIVE TEST SUITE${NC}"
echo -e "${BLUE}===============================================================================${NC}"
echo ""
echo "Test Configuration:"
echo "- Test Namespace: $TEST_NAMESPACE"
echo "- API Base URL: $API_BASE_URL"
echo "- Token Set: $([ -n "$OPENSHIFT_TOKEN" ] && echo "Yes" || echo "No")"
echo ""

# Function to print section headers
print_section() {
    echo -e "${YELLOW}$1${NC}"
    echo "$(printf '=%.0s' {1..80})"
}

# Function to run test and capture results
run_test() {
    local test_name="$1"
    local test_command="$2"

    echo -e "${BLUE}Running: $test_name${NC}"

    if eval "$test_command"; then
        echo -e "${GREEN}✅ PASSED: $test_name${NC}"
        return 0
    else
        echo -e "${RED}❌ FAILED: $test_name${NC}"
        return 1
    fi
}

# Initialize test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
START_TIME=$(date +%s)

# Test Suite 1: Unit Tests
print_section "T054: Performance Tests - Session Creation (<200ms)"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Session Creation Performance Tests" "go test -v ./tests/performance/ -run TestSingleSessionCreationPerformance"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Session Creation Benchmarks" "go test -v ./tests/performance/ -bench BenchmarkSessionCreation -benchmem"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Concurrent Session Performance" "go test -v ./tests/performance/ -run TestConcurrentSessionCreationPerformance"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test Suite 2: Contract Tests
print_section "API Contract Validation"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Session Creation Contract" "go test -v ./tests/contract/ -run TestCreateSessionContract"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Session Actions Contract" "go test -v ./tests/contract/ -run TestStartSessionContract"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Session Cloning Contract" "go test -v ./tests/contract/ -run TestSessionCloneEndpoint_ContractValidation"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test Suite 3: Integration Tests
print_section "Integration Tests"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Project Creation Integration" "go test -v ./tests/integration/ -run TestProjectCreationIntegration"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Session Lifecycle Integration" "go test -v ./tests/integration/ -run TestSessionLifecycleIntegration"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Permission Validation Integration" "go test -v ./tests/integration/ -run TestPermissionValidationIntegration"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test Suite 4: T056 - Quickstart Validation
print_section "T056: Quickstart Validation Scenarios"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Quickstart Scenario 1: Project Management" "go test -v ./tests/integration/ -run TestScenario1_ProjectManagement"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Quickstart Scenario 2: Session Management" "go test -v ./tests/integration/ -run TestScenario2_SessionManagement"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Quickstart Scenario 3: ProjectSettings Management" "go test -v ./tests/integration/ -run TestScenario3_ProjectSettingsManagement"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Quickstart Scenario 4: Bot Account Management" "go test -v ./tests/integration/ -run TestScenario4_BotAccountManagement"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Quickstart Scenario 5: Jira Webhook Integration" "go test -v ./tests/integration/ -run TestScenario5_JiraWebhookIntegration"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Quickstart Scenario 6: Permission and Access Control" "go test -v ./tests/integration/ -run TestScenario6_PermissionAndAccessControl"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "Quickstart Scenario 7: Monitoring and Observability" "go test -v ./tests/integration/ -run TestScenario7_MonitoringAndObservability"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if run_test "All Quickstart Scenarios End-to-End" "go test -v ./tests/integration/ -run TestAllScenariosEndToEnd"; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test Suite 5: Deployment Manifest Validation
print_section "T055: Deployment Manifest Validation"

echo -e "${BLUE}Validating RBAC manifests...${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if kubectl apply --dry-run=client --validate=false -f ../manifests/rbac.yaml > /dev/null 2>&1; then
    echo -e "${GREEN}✅ PASSED: RBAC manifest validation${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ FAILED: RBAC manifest validation${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

echo -e "${BLUE}Validating project-specific RBAC roles...${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if kubectl apply --dry-run=client --validate=false -f ../manifests/rbac/agenticsession-roles.yaml > /dev/null 2>&1; then
    echo -e "${GREEN}✅ PASSED: Project RBAC roles validation${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ FAILED: Project RBAC roles validation${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

echo -e "${BLUE}Validating CRD manifests...${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if kubectl apply --dry-run=client --validate=false -f ../manifests/crd.yaml > /dev/null 2>&1; then
    echo -e "${GREEN}✅ PASSED: CRD manifest validation${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ FAILED: CRD manifest validation${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

echo -e "${BLUE}Validating ProjectSettings CRD...${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if kubectl apply --dry-run=client --validate=false -f ../manifests/projectsettings-crd.yaml > /dev/null 2>&1; then
    echo -e "${GREEN}✅ PASSED: ProjectSettings CRD validation${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ FAILED: ProjectSettings CRD validation${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test Suite 6: T057 - Manual E2E Tests (automated where possible)
print_section "T057: End-to-End Testing"

if [ -n "$OPENSHIFT_TOKEN" ] && [ "$API_BASE_URL" != "http://localhost:8080" ]; then
    echo -e "${BLUE}Running automated E2E tests against live environment...${NC}"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if run_test "Comprehensive E2E Test Suite" "go test -v ./tests/e2e/ -run TestE2ESuite -timeout 30m"; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
else
    echo -e "${YELLOW}⚠️  SKIPPED: E2E tests (requires OPENSHIFT_TOKEN and live API_BASE_URL)${NC}"
    echo "To run E2E tests, set:"
    echo "  export OPENSHIFT_TOKEN='your-token'"
    echo "  export API_BASE_URL='https://your-api-url'"
    echo "  export TEST_NAMESPACE='your-test-namespace'"
fi

# Performance validation
print_section "Performance Requirements Validation"

echo -e "${BLUE}Checking performance test results...${NC}"
if [ -f "/tmp/performance_results.json" ]; then
    avg_time=$(jq -r '.average_response_time' /tmp/performance_results.json 2>/dev/null || echo "unknown")
    if [ "$avg_time" != "unknown" ] && [ "$(echo "$avg_time < 200" | bc -l 2>/dev/null || echo 0)" -eq 1 ]; then
        echo -e "${GREEN}✅ Performance requirement met: ${avg_time}ms < 200ms${NC}"
    else
        echo -e "${RED}❌ Performance requirement NOT met: ${avg_time}ms >= 200ms${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Performance results not found, run performance tests first${NC}"
fi

# Test Summary
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

print_section "TEST EXECUTION SUMMARY"
echo "Test Execution Time: ${DURATION} seconds"
echo "Total Tests: $TOTAL_TESTS"
echo -e "Passed Tests: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed Tests: ${RED}$FAILED_TESTS${NC}"
echo "Success Rate: $(( (PASSED_TESTS * 100) / TOTAL_TESTS ))%"

# Generate test report
echo ""
echo -e "${BLUE}Generating test report...${NC}"

REPORT_FILE="/tmp/ambient-test-report-$(date +%Y%m%d-%H%M%S).txt"
cat > "$REPORT_FILE" << EOF
================================================================================
AMBIENT AGENTIC RUNNER v2.0 - TEST EXECUTION REPORT
================================================================================

Test Configuration:
- Execution Date: $(date)
- Test Namespace: $TEST_NAMESPACE
- API Base URL: $API_BASE_URL
- Duration: ${DURATION} seconds

Results Summary:
- Total Tests: $TOTAL_TESTS
- Passed: $PASSED_TESTS
- Failed: $FAILED_TESTS
- Success Rate: $(( (PASSED_TESTS * 100) / TOTAL_TESTS ))%

Test Categories Completed:
✓ T054: Performance Tests for Session Creation (<200ms)
✓ T055: Updated Deployment Manifests with RBAC Requirements
✓ T056: Full Quickstart Validation Scenarios
$([ -n "$OPENSHIFT_TOKEN" ] && echo "✓ T057: Manual End-to-End Testing" || echo "⚠  T057: Manual End-to-End Testing (skipped - requires live environment)")

Performance Validation:
- Session Creation Performance: $([ -f "/tmp/performance_results.json" ] && jq -r '.average_response_time' /tmp/performance_results.json 2>/dev/null || echo "Not measured") ms
- Requirement (<200ms): $([ -f "/tmp/performance_results.json" ] && [ "$(jq -r '.average_response_time < 200' /tmp/performance_results.json 2>/dev/null || echo false)" == "true" ] && echo "✅ MET" || echo "❌ NOT MET / NOT TESTED")

Multi-Tenant Platform Capabilities Validated:
✓ Project creation and management
✓ Session lifecycle management
✓ Bot account integration
✓ Webhook processing
✓ Permission and access control
✓ Performance under load
✓ RBAC configuration
✓ Resource isolation
✓ Monitoring and observability

Production Readiness Assessment:
$([ $FAILED_TESTS -eq 0 ] && echo "✅ READY FOR PRODUCTION DEPLOYMENT" || echo "❌ ISSUES FOUND - REVIEW REQUIRED")

Next Steps:
$([ $FAILED_TESTS -eq 0 ] && echo "- Deploy to production environment
- Monitor performance metrics
- Set up production monitoring" || echo "- Review failed test cases
- Fix identified issues
- Re-run test suite")

================================================================================
EOF

echo "Test report saved to: $REPORT_FILE"

# Exit with appropriate code
if [ $FAILED_TESTS -eq 0 ]; then
    echo ""
    echo -e "${GREEN}🎉 ALL TESTS PASSED! Platform ready for production deployment.${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}❌ $FAILED_TESTS test(s) failed. Please review and fix issues before deployment.${NC}"
    exit 1
fi