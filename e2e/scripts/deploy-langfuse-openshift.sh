#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "======================================"
echo "Deploying Langfuse to OpenShift (ROSA)"
echo "======================================"
echo ""

# Check prerequisites
if ! command -v helm &> /dev/null; then
  echo "❌ Helm not found. Please install Helm 3.x first."
  echo "   Visit: https://helm.sh/docs/intro/install/"
  exit 1
fi

if ! command -v oc &> /dev/null; then
  echo "❌ oc CLI not found. Please install OpenShift CLI first."
  echo "   Visit: https://docs.openshift.com/container-platform/latest/cli_reference/openshift_cli/getting-started-cli.html"
  exit 1
fi

# Check cluster connection
if ! oc whoami &>/dev/null; then
  echo "❌ Not logged into OpenShift cluster"
  echo "   Please run: oc login <cluster-url>"
  exit 1
fi

CLUSTER_USER=$(oc whoami)
CLUSTER_URL=$(oc whoami --show-server)
echo "Connected to OpenShift:"
echo "   User: $CLUSTER_USER"
echo "   Cluster: $CLUSTER_URL"
echo ""

# Use simple passwords for test environment
echo "Setting simple passwords for test environment..."
NEXTAUTH_SECRET="test-nextauth-secret-12345678"
SALT="test-salt-12345678"
POSTGRES_PASSWORD="postgres123"
CLICKHOUSE_PASSWORD="clickhouse123"
REDIS_PASSWORD="redis123"
echo "   ✓ Credentials configured"

# Add Langfuse Helm repository
echo ""
echo "Adding Langfuse Helm repository..."
helm repo add langfuse https://langfuse.github.io/langfuse-k8s &>/dev/null || true
helm repo update &>/dev/null
echo "   ✓ Helm repository updated"

# Create namespace
echo ""
echo "Creating namespace 'langfuse'..."
if oc get namespace langfuse &>/dev/null; then
  echo "   ℹ️  Namespace 'langfuse' already exists"
else
  oc create namespace langfuse
  echo "   ✓ Namespace created"
fi

# Install or upgrade Langfuse
echo ""
echo "Installing Langfuse with Helm..."
echo "   (This may take 5-10 minutes...)"
helm upgrade --install langfuse langfuse/langfuse \
  --namespace langfuse \
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
  --set clickhouse.resources.requests.memory=512Mi \
  --set clickhouse.resources.limits.memory=1Gi \
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
  if oc get deployment $deployment -n langfuse &>/dev/null; then
    oc wait --namespace langfuse \
      --for=condition=available \
      --timeout=300s \
      deployment/$deployment &>/dev/null || true
  fi
done

# Wait for StatefulSets (they may have different names in newer chart versions)
for statefulset in langfuse-postgresql langfuse-clickhouse langfuse-redis-master langfuse-zookeeper; do
  if oc get statefulset $statefulset -n langfuse &>/dev/null; then
    oc wait --namespace langfuse \
      --for=jsonpath='{.status.readyReplicas}'=1 \
      --timeout=300s \
      statefulset/$statefulset &>/dev/null || true
  fi
done

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
oc patch deployment langfuse-web -n langfuse \
  --type='json' \
  -p="$(cat /tmp/langfuse-s3-patch.json)" &>/dev/null

# Apply patch to langfuse-worker deployment
echo "   Patching langfuse-worker deployment..."
oc patch deployment langfuse-worker -n langfuse \
  --type='json' \
  -p="$(cat /tmp/langfuse-s3-patch.json)" &>/dev/null

# Wait for rollouts to complete
echo "   Waiting for deployments to rollout..."
oc rollout status deployment/langfuse-web -n langfuse --timeout=120s &>/dev/null
oc rollout status deployment/langfuse-worker -n langfuse --timeout=120s &>/dev/null

# Cleanup temp file
rm -f /tmp/langfuse-s3-patch.json

echo "   ✓ S3 credentials configured"

# Create OpenShift Route
echo ""
echo "Creating OpenShift Route..."
if oc get route langfuse -n langfuse &>/dev/null; then
  echo "   ℹ️  Route already exists"
else
  oc create route edge langfuse \
    --service=langfuse-web \
    --port=3000 \
    --namespace=langfuse &>/dev/null
  echo "   ✓ Route created"
fi

# Get the Route URL
ROUTE_URL=$(oc get route langfuse -n langfuse -o jsonpath='{.spec.host}')
LANGFUSE_URL="https://${ROUTE_URL}"

# Save credentials
echo ""
echo "Saving credentials to .env.langfuse..."
cat > .env.langfuse <<EOF
# Langfuse Credentials (Test Environment - Simple Passwords)
NEXTAUTH_SECRET=$NEXTAUTH_SECRET
SALT=$SALT
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
CLICKHOUSE_PASSWORD=$CLICKHOUSE_PASSWORD
REDIS_PASSWORD=$REDIS_PASSWORD
LANGFUSE_URL=$LANGFUSE_URL

# Internal Service URLs
LANGFUSE_INTERNAL_URL=http://langfuse-web.langfuse.svc.cluster.local:3000
EOF
echo "   ✓ Credentials saved to e2e/.env.langfuse"

# Print status
echo ""
echo "======================================"
echo "✅ Langfuse deployment complete!"
echo "======================================"
echo ""
echo "Access Langfuse UI:"
echo "   External URL: $LANGFUSE_URL"
echo "   Internal URL: http://langfuse-web.langfuse.svc.cluster.local:3000"
echo ""
echo "Test Credentials (simple for test environment):"
echo "   PostgreSQL: postgres123"
echo "   ClickHouse: clickhouse123"
echo "   Redis: redis123"
echo ""
echo "Credentials saved to:"
echo "   e2e/.env.langfuse"
echo ""
echo "Check deployment status:"
echo "   oc get pods -n langfuse"
echo "   oc get svc -n langfuse"
echo "   oc get route -n langfuse"
echo ""
echo "View logs:"
echo "   oc logs -n langfuse -l app.kubernetes.io/name=langfuse --tail=50"
echo ""
echo "Next steps:"
echo "   1. Open $LANGFUSE_URL in your browser"
echo "   2. Sign up / create an account" echo "   3. Create a project (e.g., 'ambient-code-platform')"
echo "   4. Generate API keys: Settings → API Keys"
echo ""
echo "Cleanup:"
echo "   oc delete namespace langfuse"
echo ""
