#!/bin/bash
set -euo pipefail

echo "======================================"
echo "Setting up kind cluster for vTeam E2E"
echo "======================================"

# Detect container runtime (prefer explicit CONTAINER_ENGINE, then Docker, then Podman)
CONTAINER_ENGINE="${CONTAINER_ENGINE:-}"

if [ -z "$CONTAINER_ENGINE" ]; then
  # Check if KIND_EXPERIMENTAL_PROVIDER is already set (kind will use this)
  if [ -n "${KIND_EXPERIMENTAL_PROVIDER:-}" ]; then
    CONTAINER_ENGINE="$KIND_EXPERIMENTAL_PROVIDER"
    echo "   ℹ️  Detected KIND_EXPERIMENTAL_PROVIDER=$KIND_EXPERIMENTAL_PROVIDER"
  # Check for real Docker (not podman-docker alias)
  elif command -v docker &> /dev/null && docker ps &> /dev/null 2>&1; then
    # Verify it's actual Docker, not Podman masquerading as Docker
    if docker version 2>/dev/null | grep -q "Server.*Podman"; then
      echo "   ℹ️  Detected podman-docker compatibility package"
      CONTAINER_ENGINE="podman"
    else
      CONTAINER_ENGINE="docker"
    fi
  elif command -v podman &> /dev/null; then
    CONTAINER_ENGINE="podman"
  else
    echo "❌ Error: Neither Docker nor Podman found or running"
    echo "   Please install and start Docker or Podman"
    echo "   Docker: https://docs.docker.com/get-docker/"
    echo "   Podman: brew install podman && podman machine init && podman machine start"
    exit 1
  fi
fi

echo "Using container runtime: $CONTAINER_ENGINE"

# Configure kind to use Podman if selected
if [ "$CONTAINER_ENGINE" = "podman" ]; then
  export KIND_EXPERIMENTAL_PROVIDER=podman
  echo "   ℹ️  Set KIND_EXPERIMENTAL_PROVIDER=podman"
  
  # Verify Podman is running
  if ! podman ps &> /dev/null; then
    echo "❌ Podman is installed but not running"
    echo "   Start it with: podman machine start"
    exit 1
  fi
fi

# Check if kind cluster already exists
if kind get clusters 2>/dev/null | grep -q "^vteam-e2e$"; then
  echo "⚠️  Kind cluster 'vteam-e2e' already exists"
  echo "   Run './scripts/cleanup.sh' first to remove it"
  exit 1
fi

echo ""
echo "Creating kind cluster with ingress support..."

# Unset any existing HTTP_PORT/HTTPS_PORT from environment to avoid conflicts
unset HTTP_PORT HTTPS_PORT

# Use higher ports for Podman rootless compatibility (ports >= 1024)
# These port numbers are used as hostPort in the kind cluster config
if [ "$CONTAINER_ENGINE" = "podman" ]; then
  HTTP_PORT=8080
  HTTPS_PORT=8443
  echo "   ℹ️  Using ports 8080/8443 (Podman rootless compatibility)"
else
  HTTP_PORT=80
  HTTPS_PORT=443
  echo "   ℹ️  Using ports 80/443 (Docker with root access)"
fi

echo "   Creating cluster with port mappings: ${HTTP_PORT}->80, ${HTTPS_PORT}->443"

# Check if ports are already in use
echo "   Checking if ports are available..."
if [ "$CONTAINER_ENGINE" = "podman" ]; then
  # Check with podman
  if podman ps --format '{{.Ports}}' 2>/dev/null | grep -q "${HTTP_PORT}"; then
    echo "❌ Error: Port ${HTTP_PORT} is already in use by another container"
    echo "   Check with: podman ps"
    echo "   You may need to run: ./scripts/cleanup.sh"
    exit 1
  fi
elif command -v lsof &> /dev/null; then
  # Check with lsof if available (works for both Docker and other services)
  if lsof -i :"${HTTP_PORT}" &> /dev/null; then
    echo "❌ Error: Port ${HTTP_PORT} is already in use"
    echo "   Check with: lsof -i :${HTTP_PORT}"
    echo "   You may need to run: ./scripts/cleanup.sh"
    exit 1
  fi
elif command -v netstat &> /dev/null; then
  # Fallback to netstat
  if netstat -tln 2>/dev/null | grep -q ":${HTTP_PORT} "; then
    echo "❌ Error: Port ${HTTP_PORT} is already in use"
    echo "   Check with: netstat -tln | grep ${HTTP_PORT}"
    echo "   You may need to run: ./scripts/cleanup.sh"
    exit 1
  fi
fi

# Create kind config file with dynamic ports
# Note: containerPort is what nginx-ingress listens on inside the kind node (always 80/443)
#       hostPort is what we expose on the host machine (80/443 for Docker, 8080/8443 for Podman)

# Generate the config first to verify variable expansion
KIND_CONFIG=$(cat <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: ${HTTP_PORT}
    protocol: TCP
  - containerPort: 443
    hostPort: ${HTTPS_PORT}
    protocol: TCP
EOF
)

# Debug: Show the actual configuration if DEBUG is set
if [ "${DEBUG:-}" = "1" ]; then
  echo "   DEBUG: Generated kind config:"
  echo "$KIND_CONFIG" | sed 's/^/     /'
fi

# Create the cluster
echo "$KIND_CONFIG" | kind create cluster --name vteam-e2e --config=-

echo ""
echo "Installing nginx-ingress controller..."
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

echo ""
echo "Waiting for ingress controller to be ready..."

# Wait for deployment to exist first
echo "   Waiting for deployment to be created..."
for i in {1..30}; do
  if kubectl get deployment ingress-nginx-controller -n ingress-nginx &>/dev/null; then
    break
  fi
  if [ $i -eq 30 ]; then
    echo "❌ Timeout waiting for ingress controller deployment"
    exit 1
  fi
  sleep 2
done

# Wait for pods to be created
echo "   Waiting for pods to be created..."
for i in {1..30}; do
  if kubectl get pods -n ingress-nginx -l app.kubernetes.io/component=controller &>/dev/null; then
    POD_COUNT=$(kubectl get pods -n ingress-nginx -l app.kubernetes.io/component=controller --no-headers 2>/dev/null | wc -l)
    if [ "$POD_COUNT" -gt 0 ]; then
      break
    fi
  fi
  if [ $i -eq 30 ]; then
    echo "❌ Timeout waiting for ingress controller pods"
    exit 1
  fi
  sleep 2
done

# Now wait for pods to be ready
echo "   Waiting for pods to be ready..."
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s

echo ""
echo "Adding vteam.local to /etc/hosts..."
if grep -q "vteam.local" /etc/hosts 2>/dev/null; then
  echo "   vteam.local already in /etc/hosts"
else
  # In CI, sudo typically doesn't require password (NOPASSWD configured)
  # Locally, user will be prompted for password
  if echo "127.0.0.1 vteam.local" | sudo tee -a /etc/hosts > /dev/null 2>&1; then
    echo "   ✓ Added vteam.local to /etc/hosts"
  else
    echo "   ⚠️  Warning: Could not modify /etc/hosts (permission denied)"
    echo "   Tests may fail if DNS resolution doesn't work"
    echo "   Manual fix: Add '127.0.0.1 vteam.local' to /etc/hosts"
  fi
fi

echo ""
echo "✅ Kind cluster ready!"
echo "   Cluster: vteam-e2e"
echo "   Ingress: nginx"
if [ "$CONTAINER_ENGINE" = "podman" ]; then
  echo "   Access: http://vteam.local:8080"
else
  echo "   Access: http://vteam.local"
fi

