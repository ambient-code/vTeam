#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "======================================"
echo "Deploying vTeam to kind cluster"
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

# Check if kind cluster exists
if ! kind get clusters 2>/dev/null | grep -q "^vteam-e2e$"; then
  echo "❌ Kind cluster 'vteam-e2e' not found"
  echo "   Run './scripts/setup-kind.sh' first"
  exit 1
fi

echo ""
echo "Waiting for ingress admission webhook to be ready..."
# The admission webhook needs time to start even after the controller is ready
for i in {1..30}; do
  if kubectl get validatingwebhookconfigurations.admissionregistration.k8s.io ingress-nginx-admission &>/dev/null; then
    # Give it a few more seconds to be fully ready
    sleep 3
    break
  fi
  if [ $i -eq 30 ]; then
    echo "⚠️  Warning: Admission webhook may not be ready, but continuing..."
    break
  fi
  sleep 2
done

echo ""
echo "Applying manifests with kustomize..."
# Use e2e overlay from components/manifests
kubectl apply -k ../components/manifests/overlays/e2e/

echo ""
echo "Waiting for deployments to be ready..."
./scripts/wait-for-ready.sh

echo ""
echo "Extracting test user token..."
# Wait for the secret to be populated with a token (max 30 seconds)
TOKEN=""
for i in {1..15}; do
  TOKEN=$(kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' 2>/dev/null | base64 -d 2>/dev/null || echo "")
  if [ -n "$TOKEN" ]; then
    echo "   ✓ Token extracted successfully"
    break
  fi
  if [ $i -eq 15 ]; then
    echo "❌ Failed to extract test token after 30 seconds"
    echo "   The secret may not be ready. Check with:"
    echo "   kubectl get secret test-user-token -n ambient-code"
    exit 1
  fi
  sleep 2
done

# Detect which port to use (check kind cluster config)
HTTP_PORT=80
if kind get clusters 2>/dev/null | grep -q "^vteam-e2e$"; then
  # Check if we're using non-standard ports (Podman)
  if docker ps --filter "name=vteam-e2e-control-plane" --format "{{.Ports}}" 2>/dev/null | grep -q "8080" || \
     podman ps --filter "name=vteam-e2e-control-plane" --format "{{.Ports}}" 2>/dev/null | grep -q "8080"; then
    HTTP_PORT=8080
  fi
fi

BASE_URL="http://vteam.local"
if [ "$HTTP_PORT" != "80" ]; then
  BASE_URL="http://vteam.local:${HTTP_PORT}"
fi

echo "TEST_TOKEN=$TOKEN" > .env.test
echo "CYPRESS_BASE_URL=$BASE_URL" >> .env.test
echo "   ✓ Token saved to .env.test"
echo "   ✓ Base URL: $BASE_URL"

echo ""
echo "✅ Deployment complete!"
echo ""
echo "Access the application:"
echo "   Frontend: $BASE_URL"
echo "   Backend:  $BASE_URL/api/health"
echo ""
echo "Check pod status:"
echo "   kubectl get pods -n ambient-code"
echo ""
echo "Run tests:"
echo "   ./scripts/run-tests.sh"

