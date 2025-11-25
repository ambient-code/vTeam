#!/bin/bash
# Setup script for Vertex AI on local minikube
set -e

echo "🔐 Vertex AI Setup for Local Development"
echo ""

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Please install kubectl first."
    exit 1
fi

# Check if minikube is running
if ! kubectl cluster-info &> /dev/null; then
    echo "❌ Cannot connect to Kubernetes cluster. Is minikube running?"
    echo "   Try: make local-up"
    exit 1
fi

# Prompt for GCP project ID
echo "📝 Enter your GCP Project ID:"
read -r GCP_PROJECT_ID

if [ -z "$GCP_PROJECT_ID" ]; then
    echo "❌ GCP Project ID is required"
    exit 1
fi

# Prompt for service account key file
echo ""
echo "📝 Enter path to your service account key file (ambient-code-key.json):"
read -r KEY_FILE

if [ ! -f "$KEY_FILE" ]; then
    echo "❌ File not found: $KEY_FILE"
    exit 1
fi

# Update operator-config.yaml with project ID
echo ""
echo "📝 Updating operator-config.yaml with project ID: $GCP_PROJECT_ID"
sed -i.bak "s/ANTHROPIC_VERTEX_PROJECT_ID: .*/ANTHROPIC_VERTEX_PROJECT_ID: \"$GCP_PROJECT_ID\"/" \
    components/manifests/minikube/operator-config.yaml

# Delete existing secret if it exists
kubectl delete secret ambient-vertex -n ambient-code 2>/dev/null || true

# Create the secret
echo "📝 Creating ambient-vertex secret..."
kubectl create secret generic ambient-vertex \
    --from-file=ambient-code-key.json="$KEY_FILE" \
    -n ambient-code

# Apply the updated config
echo "📝 Applying operator configuration..."
kubectl apply -f components/manifests/minikube/operator-config.yaml

# Restart operator if it's running
if kubectl get deployment agentic-operator -n ambient-code &> /dev/null; then
    echo "🔄 Restarting operator to pick up new configuration..."
    kubectl rollout restart deployment/agentic-operator -n ambient-code
    kubectl rollout status deployment/agentic-operator -n ambient-code --timeout=60s
fi

echo ""
echo "✅ Vertex AI configuration complete!"
echo ""
echo "Next steps:"
echo "  1. Run: make local-up"
echo "  2. Create a workspace in the UI"
echo "  3. Start using AI features with your company's Vertex AI"
echo ""
echo "Note: Configuration backup saved to operator-config.yaml.bak"

