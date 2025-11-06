#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "======================================"
echo "Running vTeam E2E Tests"
echo "======================================"

# Check if .env.test exists
if [ ! -f .env.test ]; then
  echo "❌ Error: .env.test not found"
  echo "   Run './scripts/deploy.sh' first to set up the environment"
  exit 1
fi

# Load test token and base URL
source .env.test

if [ -z "${TEST_TOKEN:-}" ]; then
  echo "❌ Error: TEST_TOKEN not set in .env.test"
  exit 1
fi

# Use CYPRESS_BASE_URL from .env.test, or default
CYPRESS_BASE_URL="${CYPRESS_BASE_URL:-http://vteam.local}"

echo ""
echo "Test token loaded ✓"
echo "Base URL: $CYPRESS_BASE_URL"
echo ""

# Check if npm packages are installed
if [ ! -d node_modules ]; then
  echo "Installing npm dependencies..."
  npm install
  echo ""
fi

# Run Cypress tests
echo "Starting Cypress tests..."
echo ""

CYPRESS_TEST_TOKEN="$TEST_TOKEN" CYPRESS_BASE_URL="$CYPRESS_BASE_URL" npm test

exit_code=$?

echo ""
if [ $exit_code -eq 0 ]; then
  echo "✅ All tests passed!"
else
  echo "❌ Some tests failed (exit code: $exit_code)"
  echo ""
  echo "Debugging tips:"
  echo "  - Check pod logs: kubectl logs -n ambient-code -l app=frontend"
  echo "  - Check services: kubectl get svc -n ambient-code"
  echo "  - Check ingress: kubectl get ingress -n ambient-code"
  echo "  - Test manually: curl http://vteam.local"
fi

exit $exit_code

