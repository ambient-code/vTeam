#!/bin/bash
set -euo pipefail

# Parse command line arguments
PLATFORM="auto"
while [[ $# -gt 0 ]]; do
  case $1 in
    --openshift)
      PLATFORM="openshift"
      shift
      ;;
    --kubernetes|--k8s)
      PLATFORM="kubernetes"
      shift
      ;;
    --help|-h)
      echo "Usage: $0 [--openshift|--kubernetes]"
      echo ""
      echo "Options:"
      echo "  --openshift    Force OpenShift mode (use oc, create Route)"
      echo "  --kubernetes   Force Kubernetes mode (use kubectl, create Ingress)"
      echo "  (default)      Auto-detect based on available CLI and cluster type"
      echo ""
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

echo "======================================"
echo "Deploying Langfuse to Kubernetes"
echo "======================================"
echo ""

# Detect platform if auto mode
if [ "$PLATFORM" = "auto" ]; then
  echo "Auto-detecting platform..."

  # Check if oc is available and we're on OpenShift
  if command -v oc &> /dev/null; then
    if oc api-resources --api-group=route.openshift.io &>/dev/null 2>&1; then
      PLATFORM="openshift"
      echo "   ✓ Detected OpenShift cluster"
    else
      PLATFORM="kubernetes"
      echo "   ✓ Detected Kubernetes cluster (oc CLI available)"
    fi
  elif command -v kubectl &> /dev/null; then
    PLATFORM="kubernetes"
    echo "   ✓ Detected Kubernetes cluster"
  else
    echo "❌ Neither kubectl nor oc found. Please install Kubernetes CLI."
    exit 1
  fi
  echo ""
fi

# Set CLI tool based on platform
if [ "$PLATFORM" = "openshift" ]; then
  CLI="oc"
  PLATFORM_NAME="OpenShift"
else
  CLI="kubectl"
  PLATFORM_NAME="Kubernetes"
fi

echo "Platform: $PLATFORM_NAME"
echo "CLI: $CLI"
echo ""

# Check prerequisites
if ! command -v helm &> /dev/null; then
  echo "❌ Helm not found. Please install Helm 3.x first."
  echo "   Visit: https://helm.sh/docs/intro/install/"
  exit 1
fi

if ! command -v $CLI &> /dev/null; then
  echo "❌ $CLI not found. Please install $PLATFORM_NAME CLI first."
  if [ "$PLATFORM" = "openshift" ]; then
    echo "   Visit: https://docs.openshift.com/container-platform/latest/cli_reference/openshift_cli/getting-started-cli.html"
  else
    echo "   Visit: https://kubernetes.io/docs/tasks/tools/"
  fi
  exit 1
fi

# Check cluster connection
if ! $CLI cluster-info &>/dev/null; then
  echo "❌ Not connected to $PLATFORM_NAME cluster"
  if [ "$PLATFORM" = "openshift" ]; then
    echo "   Please run: $CLI login <cluster-url>"
  else
    echo "   Please configure kubectl: kubectl config use-context <context>"
  fi
  exit 1
fi

CLUSTER_USER=$($CLI config view --minify -o jsonpath='{.contexts[0].context.user}' 2>/dev/null || echo "unknown")
CLUSTER_URL=$($CLI config view --minify -o jsonpath='{.clusters[0].cluster.server}')
echo "Connected to $PLATFORM_NAME:"
echo "   User: $CLUSTER_USER"
echo "   Cluster: $CLUSTER_URL"
echo ""

# Prompt for credentials or use defaults for testing
read -p "Use simple test passwords? (y/n, default: y): " USE_TEST_CREDS
USE_TEST_CREDS=${USE_TEST_CREDS:-y}

if [[ "$USE_TEST_CREDS" =~ ^[Yy]$ ]]; then
  echo "Setting simple passwords for test environment..."
  NEXTAUTH_SECRET="test-nextauth-secret-12345678"
  SALT="test-salt-12345678"
  POSTGRES_PASSWORD="postgres123"
  CLICKHOUSE_PASSWORD="clickhouse123"
  REDIS_PASSWORD="redis123"
  echo "   ✓ Test credentials configured"
else
  echo "Generating secure random credentials..."
  NEXTAUTH_SECRET=$(openssl rand -hex 32)
  SALT=$(openssl rand -hex 32)
  POSTGRES_PASSWORD=$(openssl rand -base64 32)
  CLICKHOUSE_PASSWORD=$(openssl rand -base64 32)
  REDIS_PASSWORD=$(openssl rand -base64 32)
  echo "   ✓ Secure credentials generated"
fi

# Add Langfuse Helm repository
echo ""
echo "Adding Langfuse Helm repository..."
helm repo add langfuse https://langfuse.github.io/langfuse-k8s &>/dev/null || true
helm repo update &>/dev/null
echo "   ✓ Helm repository updated"

# Create namespace
echo ""
echo "Creating namespace 'langfuse'..."
if $CLI get namespace langfuse &>/dev/null; then
  echo "   ℹ️  Namespace 'langfuse' already exists"
else
  $CLI create namespace langfuse
  echo "   ✓ Namespace created"
fi

# Install or upgrade Langfuse
echo ""
echo "Installing Langfuse with Helm..."
echo "   (This may take 5-10 minutes...)"
helm upgrade --install langfuse langfuse/langfuse \
  --namespace langfuse \
  --version ">= 3.63.0" \
  --set langfuse.nextauth.secret.value="$NEXTAUTH_SECRET" \
  --set langfuse.salt.value="$SALT" \
  --set postgresql.auth.password="$POSTGRES_PASSWORD" \
  --set clickhouse.auth.password="$CLICKHOUSE_PASSWORD" \
  --set redis.auth.password="$REDIS_PASSWORD" \
  --set langfuse.ingress.enabled=false \
  --set resources.limits.cpu=1000m \
  --set resources.limits.memory=2Gi \
  --set resources.requests.cpu=500m \
  --set resources.requests.memory=1Gi \
  --set clickhouse.replicaCount=1 \
  --set clickhouse.podAntiAffinityPreset=none \
  --set clickhouse.resources.requests.memory=1Gi \
  --set clickhouse.resources.limits.memory=2Gi \
  --set clickhouse.resources.requests.cpu=500m \
  --set clickhouse.resources.limits.cpu=1 \
  --set postgresql.primary.podAntiAffinityPreset=none \
  --set redis.master.podAntiAffinityPreset=none \
  --set zookeeper.replicas=1 \
  --set zookeeper.podAntiAffinityPreset=none \
  --set zookeeper.resources.requests.memory=256Mi \
  --set zookeeper.resources.limits.memory=512Mi \
  --set zookeeper.resources.requests.cpu=250m \
  --set zookeeper.resources.limits.cpu=500m \
  --set minio.enabled=true \
  --wait \
  --timeout=10m

echo "   ✓ Langfuse installed"

# Wait for all pods to be ready
echo ""
echo "⏳ Waiting for Langfuse pods to be ready..."

# Wait for deployments
for deployment in langfuse-web langfuse-worker; do
  if $CLI get deployment $deployment -n langfuse &>/dev/null; then
    $CLI wait --namespace langfuse \
      --for=condition=available \
      --timeout=300s \
      deployment/$deployment &>/dev/null || true
  fi
done

# Wait for StatefulSets
for statefulset in langfuse-postgresql langfuse-clickhouse-shard0 langfuse-redis-primary; do
  if $CLI get statefulset $statefulset -n langfuse &>/dev/null; then
    $CLI wait --namespace langfuse \
      --for=jsonpath='{.status.readyReplicas}'=1 \
      --timeout=300s \
      statefulset/$statefulset &>/dev/null || true
  fi
done

# Wait for zookeeper (may have multiple replicas)
if $CLI get statefulset langfuse-zookeeper -n langfuse &>/dev/null; then
  ZOOKEEPER_REPLICAS=$($CLI get statefulset langfuse-zookeeper -n langfuse -o jsonpath='{.spec.replicas}')
  $CLI wait --namespace langfuse \
    --for=jsonpath="{.status.readyReplicas}"=$ZOOKEEPER_REPLICAS \
    --timeout=300s \
    statefulset/langfuse-zookeeper &>/dev/null || true
fi

echo "   ✓ All pods ready"

# Fix S3 credentials for langfuse-web and langfuse-worker
echo ""
echo "Applying S3 credential fix..."

# Create JSON patch for S3 credentials
cat > /tmp/langfuse-s3-patch.json <<'EOF'
[
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/env/-",
    "value": {
      "name": "LANGFUSE_S3_EVENT_UPLOAD_ACCESS_KEY_ID",
      "valueFrom": {
        "secretKeyRef": {
          "name": "langfuse-s3",
          "key": "root-user"
        }
      }
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/env/-",
    "value": {
      "name": "LANGFUSE_S3_EVENT_UPLOAD_SECRET_ACCESS_KEY",
      "valueFrom": {
        "secretKeyRef": {
          "name": "langfuse-s3",
          "key": "root-password"
        }
      }
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/env/-",
    "value": {
      "name": "LANGFUSE_S3_BATCH_EXPORT_ACCESS_KEY_ID",
      "valueFrom": {
        "secretKeyRef": {
          "name": "langfuse-s3",
          "key": "root-user"
        }
      }
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/env/-",
    "value": {
      "name": "LANGFUSE_S3_BATCH_EXPORT_SECRET_ACCESS_KEY",
      "valueFrom": {
        "secretKeyRef": {
          "name": "langfuse-s3",
          "key": "root-password"
        }
      }
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/env/-",
    "value": {
      "name": "LANGFUSE_S3_MEDIA_UPLOAD_ACCESS_KEY_ID",
      "valueFrom": {
        "secretKeyRef": {
          "name": "langfuse-s3",
          "key": "root-user"
        }
      }
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/env/-",
    "value": {
      "name": "LANGFUSE_S3_MEDIA_UPLOAD_SECRET_ACCESS_KEY",
      "valueFrom": {
        "secretKeyRef": {
          "name": "langfuse-s3",
          "key": "root-password"
        }
      }
    }
  }
]
EOF

# Apply patch to langfuse-web deployment
echo "   Patching langfuse-web deployment..."
$CLI patch deployment langfuse-web -n langfuse \
  --type='json' \
  -p="$(cat /tmp/langfuse-s3-patch.json)" &>/dev/null

# Apply patch to langfuse-worker deployment
echo "   Patching langfuse-worker deployment..."
$CLI patch deployment langfuse-worker -n langfuse \
  --type='json' \
  -p="$(cat /tmp/langfuse-s3-patch.json)" &>/dev/null

# Wait for rollouts to complete
echo "   Waiting for deployments to rollout..."
$CLI rollout status deployment/langfuse-web -n langfuse --timeout=120s &>/dev/null
$CLI rollout status deployment/langfuse-worker -n langfuse --timeout=120s &>/dev/null

# Cleanup temp file
rm -f /tmp/langfuse-s3-patch.json

echo "   ✓ S3 credentials configured"

# Create Ingress or Route based on platform
echo ""
if [ "$PLATFORM" = "openshift" ]; then
  echo "Creating OpenShift Route..."
  if $CLI get route langfuse -n langfuse &>/dev/null; then
    echo "   ℹ️  Route already exists"
  else
    $CLI create route edge langfuse \
      --service=langfuse-web \
      --port=3000 \
      --namespace=langfuse &>/dev/null
    echo "   ✓ Route created"
  fi

  # Get the Route URL
  ROUTE_HOST=$($CLI get route langfuse -n langfuse -o jsonpath='{.spec.host}')
  LANGFUSE_URL="https://${ROUTE_HOST}"
else
  echo "Creating Kubernetes Ingress..."

  # Check if Ingress controller is available
  if ! $CLI get ingressclass &>/dev/null 2>&1; then
    echo "   ⚠️  Warning: No Ingress controller detected"
    echo "   You may need to install an Ingress controller (nginx, traefik, etc.)"
    echo "   See: https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/"
  fi

  # Create Ingress if it doesn't exist
  if $CLI get ingress langfuse -n langfuse &>/dev/null; then
    echo "   ℹ️  Ingress already exists"
  else
    cat <<EOF | $CLI apply -f - &>/dev/null
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: langfuse
  namespace: langfuse
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  rules:
  - host: langfuse.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: langfuse-web
            port:
              number: 3000
  tls:
  - hosts:
    - langfuse.local
    secretName: langfuse-tls
EOF
    echo "   ✓ Ingress created"
  fi

  # Try to get Ingress host
  INGRESS_HOST=$($CLI get ingress langfuse -n langfuse -o jsonpath='{.spec.rules[0].host}' 2>/dev/null || echo "langfuse.local")
  LANGFUSE_URL="https://${INGRESS_HOST}"

  echo ""
  echo "   ⚠️  Note: For local access, add to /etc/hosts:"
  echo "   127.0.0.1 $INGRESS_HOST"
  echo "   Or configure DNS to point $INGRESS_HOST to your Ingress controller"
fi

# Save credentials
echo ""
echo "Saving credentials to langfuse-credentials.env..."
cat > langfuse-credentials.env <<EOF
# Langfuse Deployment Credentials
# IMPORTANT: Keep this file secure and never commit to version control!

# Platform
PLATFORM=$PLATFORM_NAME

# Authentication Secrets
NEXTAUTH_SECRET=$NEXTAUTH_SECRET
SALT=$SALT

# Database Passwords
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
CLICKHOUSE_PASSWORD=$CLICKHOUSE_PASSWORD
REDIS_PASSWORD=$REDIS_PASSWORD

# Access URLs
LANGFUSE_URL=$LANGFUSE_URL
LANGFUSE_INTERNAL_URL=http://langfuse-web.langfuse.svc.cluster.local:3000
EOF
chmod 600 langfuse-credentials.env
echo "   ✓ Credentials saved to langfuse-credentials.env (file permissions set to 600)"

# Print status
echo ""
echo "======================================"
echo "✅ Langfuse deployment complete!"
echo "======================================"
echo ""
echo "Platform: $PLATFORM_NAME"
echo ""
echo "Access Langfuse UI:"
echo "   External URL: $LANGFUSE_URL"
echo "   Internal URL (from within cluster): http://langfuse-web.langfuse.svc.cluster.local:3000"
echo ""
if [[ "$USE_TEST_CREDS" =~ ^[Yy]$ ]]; then
  echo "Database Credentials (test environment):"
  echo "   PostgreSQL: postgres123"
  echo "   ClickHouse: clickhouse123"
  echo "   Redis: redis123"
else
  echo "Database Credentials: See langfuse-credentials.env"
fi
echo ""
echo "Credentials saved to: langfuse-credentials.env"
echo ""
echo "Check deployment status:"
echo "   $CLI get pods -n langfuse"
echo "   $CLI get svc -n langfuse"
if [ "$PLATFORM" = "openshift" ]; then
  echo "   $CLI get route -n langfuse"
else
  echo "   $CLI get ingress -n langfuse"
fi
echo ""
echo "View logs:"
echo "   $CLI logs -n langfuse -l app.kubernetes.io/name=langfuse --tail=50 -f"
echo ""
echo "Next steps:"
echo "   1. Open $LANGFUSE_URL in your browser"
if [ "$PLATFORM" = "kubernetes" ]; then
  echo "      (Make sure DNS or /etc/hosts is configured)"
fi
echo "   2. Sign up and create an account"
echo "   3. Create a project for your application"
echo "   4. Generate API keys: Settings → API Keys"
echo "   5. Configure your application to use:"
echo "      LANGFUSE_PUBLIC_KEY=<your-public-key>"
echo "      LANGFUSE_SECRET_KEY=<your-secret-key>"
echo "      LANGFUSE_HOST=$LANGFUSE_URL"
echo ""
echo "Cleanup (WARNING: This deletes all Langfuse data):"
echo "   $CLI delete namespace langfuse"
echo ""
