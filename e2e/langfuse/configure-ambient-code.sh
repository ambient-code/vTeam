#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")"

echo "======================================"
echo "Configuring Langfuse for Ambient Code"
echo "======================================"
echo ""

# Check if .env.langfuse-keys exists
if [ ! -f .env.langfuse-keys ]; then
  echo "❌ .env.langfuse-keys not found!"
  echo ""
  echo "Create this file with your Langfuse API keys:"
  echo "  cd e2e/langfuse"
  echo "  cat > .env.langfuse-keys <<EOF"
  echo "  LANGFUSE_PUBLIC_KEY=pk-lf-your-public-key-here"
  echo "  LANGFUSE_SECRET_KEY=sk-lf-your-secret-key-here"
  echo "  EOF"
  echo ""
  exit 1
fi

# Check if envsubst is available
if ! command -v envsubst &> /dev/null; then
  echo "❌ envsubst not found. Please install gettext:"
  echo "   brew install gettext"
  echo "   brew link --force gettext"
  exit 1
fi

# Check if oc is available and logged in
if ! command -v oc &> /dev/null; then
  echo "❌ oc CLI not found. Please install OpenShift CLI."
  exit 1
fi

if ! oc whoami &>/dev/null; then
  echo "❌ Not logged into OpenShift cluster"
  echo "   Please run: oc login <cluster-url>"
  exit 1
fi

CLUSTER_USER=$(oc whoami)
echo "Logged in as: $CLUSTER_USER"
echo ""

# Load API keys from .env.langfuse-keys
echo "Loading API keys from .env.langfuse-keys..."
source .env.langfuse-keys

# Validate that keys are set
if [ -z "${LANGFUSE_PUBLIC_KEY:-}" ] || [ -z "${LANGFUSE_SECRET_KEY:-}" ]; then
  echo "❌ API keys not set in .env.langfuse-keys"
  echo "   Ensure both LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY are defined"
  exit 1
fi

echo "   ✓ API keys loaded"
echo ""

# Create or update Secret
echo "Creating/updating Secret (langfuse-keys) in ambient-code namespace..."
export LANGFUSE_PUBLIC_KEY
export LANGFUSE_SECRET_KEY
envsubst < secret-template.yaml | oc apply -f -
echo "   ✓ Secret created/updated"
echo ""

# Create or update ConfigMap
echo "Creating/updating ConfigMap (langfuse-config) in ambient-code namespace..."
oc apply -f configmap.yaml
echo "   ✓ ConfigMap created/updated"
echo ""

echo "======================================"
echo "✅ Langfuse configuration complete!"
echo "======================================"
echo ""
echo "Resources created in namespace: ambient-code"
echo "  • Secret: langfuse-keys"
echo "  • ConfigMap: langfuse-config"
echo ""
echo "Next steps:"
echo "  1. Update operator to inject these into runner Job pods"
echo "  2. Update Claude Code runner to use Langfuse SDK"
echo "  3. Rebuild and redeploy ambient-code components"
echo ""
