#!/bin/bash

# OpenShift Deployment Script for RHOAI AI Feature Sizing Platform
# Usage: ./deploy.sh
# Or with environment variables: NAMESPACE=my-namespace IMAGE_FULL_NAME=quay.io/my/image:latest ./deploy.sh
# Note: This script deploys a pre-built image. Use build.sh first to build and push the image.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="${NAMESPACE:-rhoai-ai-feature-sizing}"
DEFAULT_IMAGE="quay.io/gkrumbach07/llama-index-demo/rhoai-ai-feature-sizing:latest"

# Use IMAGE_FULL_NAME environment variable or default
IMAGE_FULL_NAME="${IMAGE_FULL_NAME:-$DEFAULT_IMAGE}"

echo -e "${BLUE}🚀 RHOAI AI Feature Sizing - OpenShift Deployment${NC}"
echo -e "${BLUE}=================================================${NC}"
echo -e "Image: ${GREEN}${IMAGE_FULL_NAME}${NC}"
echo -e "Namespace: ${GREEN}${NAMESPACE}${NC}"
echo ""

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to apply manifest (no namespace substitution needed)
apply_manifest() {
    local manifest_file="$1"
    echo -e "${BLUE}📋 Applying ${manifest_file}...${NC}"
    oc apply -f "openshift/${manifest_file}"
}

# Function to apply deployment with image substitution
apply_deployment() {
    echo -e "${BLUE}📋 Applying deployment.yaml with image ${IMAGE_FULL_NAME}...${NC}"
    sed "s|image:.*|image: ${IMAGE_FULL_NAME}|g" "openshift/deployment.yaml" | oc apply -f -
}

# Check prerequisites
echo -e "${YELLOW}🔍 Checking prerequisites...${NC}"
if ! command_exists oc; then
    echo -e "${RED}❌ OpenShift CLI (oc) not found. Please install it first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites check passed${NC}"
echo ""

# Check if logged in to OpenShift
echo -e "${YELLOW}🔐 Checking OpenShift authentication...${NC}"
if ! oc whoami >/dev/null 2>&1; then
    echo -e "${RED}❌ Not logged in to OpenShift. Please run 'oc login' first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Authenticated as: $(oc whoami)${NC}"
echo ""

# Verify image exists in registry (optional check)
echo -e "${YELLOW}🔍 Using pre-built image...${NC}"
echo -e "Image: ${BLUE}${IMAGE_FULL_NAME}${NC}"
echo -e "${YELLOW}💡 If image doesn't exist, run: ${BLUE}./openshift/build.sh${NC}"
echo ""

# Deploy to OpenShift
echo -e "${YELLOW}🚀 Deploying to OpenShift...${NC}"

# Create namespace first
echo -e "${BLUE}📁 Creating namespace ${NAMESPACE}...${NC}"
if [ "$NAMESPACE" != "rhoai-ai-feature-sizing" ]; then
    # Create custom namespace
    oc create namespace "$NAMESPACE" --dry-run=client -o yaml | oc apply -f -
else
    # Use default namespace manifest
    oc apply -f "openshift/namespace.yaml"
fi

# Wait for namespace to be ready
echo -e "${YELLOW}⏳ Waiting for namespace to be ready...${NC}"
oc wait --for=condition=Active namespace/${NAMESPACE} --timeout=300s || {
    echo -e "${RED}❌ Namespace creation timed out. Checking status...${NC}"
    oc describe namespace ${NAMESPACE}
    echo -e "${YELLOW}💡 Try running: oc delete namespace ${NAMESPACE} && sleep 10${NC}"
    exit 1
}

# Switch to the target namespace
echo -e "${BLUE}🔄 Switching to namespace ${NAMESPACE}...${NC}"
oc project ${NAMESPACE}

# Create persistent volume claims
apply_manifest "pvc.yaml"

# Create config map and secrets
apply_manifest "configmap.yaml"

# Check if secrets file exists and has content
if [ -s openshift/secret.yaml ] && grep -q "OPENAI_API_KEY:" openshift/secret.yaml; then
    apply_manifest "secret.yaml"
else
    echo -e "${YELLOW}⚠️  Secret file is empty or missing API keys. Creating empty secret...${NC}"
    echo -e "${YELLOW}   Please update the secret with your API keys:${NC}"
    echo -e "${YELLOW}   oc patch secret rhoai-secrets -n ${NAMESPACE} -p '{\"stringData\":{\"OPENAI_API_KEY\":\"your-key-here\"}}'${NC}"
    oc create secret generic rhoai-secrets --namespace=${NAMESPACE} --from-literal=OPENAI_API_KEY="" || true
fi

# Deploy application
apply_deployment

# Create services
apply_manifest "service.yaml"

# Create routes
apply_manifest "route.yaml"

echo ""
echo -e "${GREEN}✅ Deployment completed!${NC}"
echo ""

# Wait for deployment to be ready
echo -e "${YELLOW}⏳ Waiting for deployment to be ready...${NC}"
oc rollout status deployment/${APP_NAME} --namespace=${NAMESPACE} --timeout=300s

# Get route URLs
echo -e "${BLUE}🌐 Getting route URLs...${NC}"
API_ROUTE=$(oc get route rhoai-api -n ${NAMESPACE} -o jsonpath='{.spec.host}')
UI_ROUTE=$(oc get route rhoai-ui -n ${NAMESPACE} -o jsonpath='{.spec.host}')

echo ""
echo -e "${GREEN}🎉 Deployment successful!${NC}"
echo -e "${GREEN}========================${NC}"
echo -e "API URL: ${BLUE}https://${API_ROUTE}${NC}"
echo -e "UI URL:  ${BLUE}https://${UI_ROUTE}${NC}"
echo -e "Docs:    ${BLUE}https://${API_ROUTE}/docs${NC}"
echo ""
echo -e "${YELLOW}📝 Next steps:${NC}"
echo -e "1. Update API keys in the secret if not done already:"
echo -e "   ${BLUE}oc patch secret rhoai-secrets -n ${NAMESPACE} -p '{\"stringData\":{\"OPENAI_API_KEY\":\"your-actual-key\"}}'${NC}"
echo -e "2. Monitor the deployment:"
echo -e "   ${BLUE}oc get pods -n ${NAMESPACE}${NC}"
echo -e "3. View logs:"
echo -e "   ${BLUE}oc logs -f deployment/${APP_NAME} -n ${NAMESPACE}${NC}"
echo ""

# Note: No cleanup needed as we use in-memory substitution
