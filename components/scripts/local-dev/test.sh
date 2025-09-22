#!/bin/bash

set -euo pipefail

# Basic smoke tests for local dev

BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3000}"
PROJECT_NS="${PROJECT_NS:-my-project}"

fail() { echo "FAIL: $*"; exit 1; }
pass() { echo "PASS: $*"; }

echo "Testing backend health..."
if curl -fsS "${BACKEND_URL}/health" >/dev/null; then
  pass "Backend health OK"
else
  fail "Backend health endpoint not responding"
fi

echo "Testing frontend root..."
if curl -fsS "${FRONTEND_URL}" >/dev/null; then
  pass "Frontend reachable"
else
  fail "Frontend not reachable"
fi

echo "Testing projects API with valid token..."
# Get token from dev service account
TOKEN=$(kubectl create token dev-user -n "${PROJECT_NS}" --duration=1h 2>/dev/null || echo "")
if [[ -n "$TOKEN" ]]; then
  STATUS=$(curl -fsS -o /dev/null -w "%{http_code}\n" \
    "${BACKEND_URL}/api/projects" \
    -H "Authorization: Bearer ${TOKEN}" || echo "000")
  if echo "$STATUS" | grep -Eq '200|204'; then
    pass "Projects API OK (${STATUS})"
  else
    fail "Projects API failed (${STATUS}). Backend may have auth/token issues."
  fi
else
  echo "(Skipping API test - no dev service account token available)"
fi

echo "All tests passed."


