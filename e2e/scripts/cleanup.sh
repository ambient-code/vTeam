#!/bin/bash
set -euo pipefail

echo "======================================"
echo "Cleaning up vTeam E2E environment"
echo "======================================"

# Detect container runtime (same logic as setup-kind.sh)
CONTAINER_ENGINE="${CONTAINER_ENGINE:-}"

if [ -z "$CONTAINER_ENGINE" ]; then
  if command -v docker &> /dev/null && docker ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="docker"
  elif command -v podman &> /dev/null && podman ps &> /dev/null 2>&1; then
    CONTAINER_ENGINE="podman"
  fi
fi

# Set KIND_EXPERIMENTAL_PROVIDER if using Podman
if [ "$CONTAINER_ENGINE" = "podman" ]; then
  export KIND_EXPERIMENTAL_PROVIDER=podman
fi

echo ""
echo "Deleting kind cluster..."
if kind get clusters 2>/dev/null | grep -q "^vteam-e2e$"; then
  kind delete cluster --name vteam-e2e
  echo "   ✓ Cluster deleted"
else
  echo "   ℹ️  Cluster 'vteam-e2e' not found (already deleted?)"
fi

echo ""
echo "Removing /etc/hosts entry..."
if grep -q "vteam.local" /etc/hosts 2>/dev/null; then
  # Create backup
  sudo cp /etc/hosts /etc/hosts.bak.$(date +%Y%m%d_%H%M%S)
  # Remove the entry
  sudo sed -i.bak '/vteam.local/d' /etc/hosts
  echo "   ✓ Removed vteam.local from /etc/hosts"
  echo "   ℹ️  Backup created"
else
  echo "   ℹ️  vteam.local not found in /etc/hosts"
fi

echo ""
echo "Cleaning up test artifacts..."
cd "$(dirname "$0")/.."
if [ -f .env.test ]; then
  rm .env.test
  echo "   ✓ Removed .env.test"
fi

# Only clean screenshots/videos if CLEANUP_ARTIFACTS=true (for CI)
# Keep them locally for debugging
if [ "${CLEANUP_ARTIFACTS:-false}" = "true" ]; then
  if [ -d cypress/screenshots ]; then
    rm -rf cypress/screenshots
    echo "   ✓ Removed Cypress screenshots"
  fi

  if [ -d cypress/videos ]; then
    rm -rf cypress/videos
    echo "   ✓ Removed Cypress videos"
  fi
else
  if [ -d cypress/screenshots ] || [ -d cypress/videos ]; then
    echo "   ℹ️  Keeping screenshots/videos for review"
    echo "   To remove: rm -rf cypress/screenshots cypress/videos"
  fi
fi

echo ""
echo "✅ Cleanup complete!"

