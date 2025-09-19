#!/bin/bash

# OpenShift Deployment Script for vTeam Ambient Agentic Runner
# Usage: ./deploy.sh [--atlassian-mcp-url=http://atlassian-mcp:8080/v1/sse]
# Or with environment variables: NAMESPACE=my-namespace ./deploy.sh
# Note: This script deploys pre-built images. Build and push images first.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Configuration
NAMESPACE="${NAMESPACE:-ambient-code}"
DEFAULT_BACKEND_IMAGE="${DEFAULT_BACKEND_IMAGE:-quay.io/ambient_code/vteam_backend:latest}"
DEFAULT_FRONTEND_IMAGE="${DEFAULT_FRONTEND_IMAGE:-quay.io/ambient_code/vteam_frontend:latest}"
DEFAULT_OPERATOR_IMAGE="${DEFAULT_OPERATOR_IMAGE:-quay.io/ambient_code/vteam_operator:latest}"
DEFAULT_RUNNER_IMAGE="${DEFAULT_RUNNER_IMAGE:-quay.io/ambient_code/vteam_claude_runner:latest}"

# Parse command line arguments
ATLASSIAN_MCP_URL=""
for arg in "$@"; do
    case $arg in
        --atlassian-mcp-url=*)
            ATLASSIAN_MCP_URL="${arg#*=}"
            shift
            ;;
        uninstall)
            # Keep uninstall as a positional argument
            ;;
        *)
            # Unknown option
            echo -e "${RED}❌ Unknown option: $arg${NC}"
            echo "Usage: $0 [--atlassian-mcp-url=http://atlassian-mcp:8080/v1/sse] [uninstall]"
            exit 1
            ;;
    esac
done

# Function to validate Atlassian MCP URL
validate_atlassian_mcp_url() {
    local url="$1"
    
    if [[ -z "$url" ]]; then
        return 0  # Empty URL is valid (optional parameter)
    fi
    
    # Basic URL validation - must start with http or https
    if [[ ! "$url" =~ ^https?:// ]]; then
        echo -e "${RED}❌ Invalid Atlassian MCP URL format: $url${NC}"
        echo -e "${YELLOW}URL must start with http:// or https://${NC}"
        return 1
    fi
    
    # Check if URL contains a valid domain or host
    if [[ ! "$url" =~ ^https?://[a-zA-Z0-9.-]+(:[0-9]+)?(/.*)?$ ]]; then
        echo -e "${RED}❌ Invalid Atlassian MCP URL format: $url${NC}"
        echo -e "${YELLOW}URL must be a valid MCP server endpoint${NC}"
        return 1
    fi
    
    echo -e "${GREEN}✅ Atlassian MCP URL validation passed: $url${NC}"
    return 0
}

# Validate Atlassian MCP URL if provided
if [[ -n "$ATLASSIAN_MCP_URL" ]]; then
    if ! validate_atlassian_mcp_url "$ATLASSIAN_MCP_URL"; then
        exit 1
    fi
fi

# Function to create MCP servers ConfigMap
create_mcp_servers_configmap() {
    local namespace="$1"
    local atlassian_mcp_url="$2"
    
    echo -e "${BLUE}Creating MCP servers ConfigMap...${NC}"
    
    # Base configuration with playwright
    local mcp_config='{
  "mcpServers": {
    "playwright": {
      "command": "npx",
      "args": [
        "@playwright/mcp",
        "--headless",
        "--browser",
        "chromium",
        "--no-sandbox"
      ]
    }'
    
    # Add atlassian-mcp if MCP URL is provided
    if [[ -n "$atlassian_mcp_url" ]]; then
        echo -e "${YELLOW}Adding Atlassian MCP server for URL: ${atlassian_mcp_url}${NC}"
        mcp_config="$mcp_config"',
    "atlassian-mcp": {
      "type": "sse",
      "url": "'"$atlassian_mcp_url"'"
    }'
    fi
    
    # Close the JSON structure
    mcp_config="$mcp_config"'
  }
}'
    
    # Create or update the ConfigMap
    if oc get configmap mcp-servers-config -n "$namespace" >/dev/null 2>&1; then
        echo -e "${YELLOW}Updating existing MCP servers ConfigMap...${NC}"
        oc delete configmap mcp-servers-config -n "$namespace"
    fi
    
    oc create configmap mcp-servers-config \
        --from-literal=".mcp.json=$mcp_config" \
        -n "$namespace"
    
    echo -e "${GREEN}✅ MCP servers ConfigMap created successfully${NC}"
    
    if [[ -n "$atlassian_mcp_url" ]]; then
        echo -e "${YELLOW}Note: Ensure the atlassian-mcp service is deployed and accessible at ${atlassian_mcp_url}${NC}"
    fi
}

# Handle uninstall command early
if [ "${1:-}" = "uninstall" ]; then
    echo -e "${YELLOW}Uninstalling vTeam from namespace ${NAMESPACE}...${NC}"

    # Check prerequisites for uninstall
    if ! command_exists oc; then
        echo -e "${RED}❌ OpenShift CLI (oc) not found. Please install it first.${NC}"
        exit 1
    fi

    if ! command_exists kustomize; then
        echo -e "${RED}❌ Kustomize not found. Please install it first.${NC}"
        exit 1
    fi

    # Check if logged in to OpenShift
    if ! oc whoami >/dev/null 2>&1; then
        echo -e "${RED}❌ Not logged in to OpenShift. Please run 'oc login' first.${NC}"
        exit 1
    fi

    # Delete using kustomize
    if [ "$NAMESPACE" != "ambient-code" ]; then
        kustomize edit set namespace "$NAMESPACE"
    fi

    kustomize build . | oc delete -f - --ignore-not-found=true

    # Restore kustomization if we modified it
    if [ "$NAMESPACE" != "ambient-code" ]; then
        kustomize edit set namespace ambient-code
    fi

    echo -e "${GREEN}✅ vTeam uninstalled from namespace ${NAMESPACE}${NC}"
    echo -e "${YELLOW}Note: Namespace ${NAMESPACE} still exists. Delete manually if needed:${NC}"
    echo -e "   ${BLUE}oc delete namespace ${NAMESPACE}${NC}"
    exit 0
fi

echo -e "${BLUE}🚀 vTeam Ambient Agentic Runner - OpenShift Deployment${NC}"
echo -e "${BLUE}====================================================${NC}"
echo -e "Namespace: ${GREEN}${NAMESPACE}${NC}"
echo -e "Backend Image: ${GREEN}${DEFAULT_BACKEND_IMAGE}${NC}"
echo -e "Frontend Image: ${GREEN}${DEFAULT_FRONTEND_IMAGE}${NC}"
echo -e "Operator Image: ${GREEN}${DEFAULT_OPERATOR_IMAGE}${NC}"
echo -e "Runner Image: ${GREEN}${DEFAULT_RUNNER_IMAGE}${NC}"
if [[ -n "$ATLASSIAN_MCP_URL" ]]; then
    echo -e "Atlassian MCP URL: ${GREEN}${ATLASSIAN_MCP_URL}${NC} (Atlassian MCP enabled)"
fi
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"
if ! command_exists oc; then
    echo -e "${RED}❌ OpenShift CLI (oc) not found. Please install it first.${NC}"
    exit 1
fi

if ! command_exists kustomize; then
    echo -e "${RED}❌ Kustomize not found. Please install it first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites check passed${NC}"
echo ""

# Check if logged in to OpenShift
echo -e "${YELLOW}Checking OpenShift authentication...${NC}"
if ! oc whoami >/dev/null 2>&1; then
    echo -e "${RED}❌ Not logged in to OpenShift. Please run 'oc login' first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Authenticated as: $(oc whoami)${NC}"
echo ""

# Check environment file
echo -e "${YELLOW}Checking environment configuration...${NC}"
ENV_FILE=".env"
if [[ ! -f "$ENV_FILE" ]]; then
    echo -e "${RED}❌ .env file not found${NC}"
    echo -e "${YELLOW}Please create .env file from env.example:${NC}"
    echo "  cp env.example .env"
    echo "  # Edit .env and add your actual API key and Git configuration"
    exit 1
fi

# Source environment variables
source "$ENV_FILE"

if [[ -z "$ANTHROPIC_API_KEY" ]]; then
    echo -e "${RED}❌ ANTHROPIC_API_KEY not set in .env file${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Environment configuration loaded${NC}"
echo ""

# Update git-configmap with environment variables if they exist
echo -e "${YELLOW}Updating Git configuration...${NC}"
if [[ -n "$GIT_USER_NAME" ]] || [[ -n "$GIT_USER_EMAIL" ]]; then
    echo -e "${BLUE}Found Git configuration in .env, updating git-configmap...${NC}"

    # Create temporary configmap patch
    cat > /tmp/git-config-patch.yaml << EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: git-config
  namespace: $NAMESPACE
data:
  git-user-name: "${GIT_USER_NAME:-}"
  git-user-email: "${GIT_USER_EMAIL:-}"
  git-ssh-key-secret: "${GIT_SSH_KEY_SECRET:-}"
  git-token-secret: "${GIT_TOKEN_SECRET:-}"
  git-repositories: |
    ${GIT_REPOSITORIES:-}
  git-clone-on-startup: "${GIT_CLONE_ON_STARTUP:-false}"
  git-workspace-path: "/workspace/git-repos"
EOF
else
    echo -e "${YELLOW}No Git configuration found in .env, using defaults${NC}"
fi
echo ""

# Deploy using kustomize
echo -e "${YELLOW}Deploying to OpenShift using Kustomize...${NC}"

# Set namespace if different from default
if [ "$NAMESPACE" != "ambient-code" ]; then
    echo -e "${BLUE}Setting custom namespace: ${NAMESPACE}${NC}"
    kustomize edit set namespace "$NAMESPACE"
fi

# Set custom images if different from defaults
echo -e "${BLUE}Setting custom images...${NC}"
kustomize edit set image quay.io/ambient_code/vteam_backend:latest=${DEFAULT_BACKEND_IMAGE}
kustomize edit set image quay.io/ambient_code/vteam_frontend:latest=${DEFAULT_FRONTEND_IMAGE}
kustomize edit set image quay.io/ambient_code/vteam_operator:latest=${DEFAULT_OPERATOR_IMAGE}
kustomize edit set image quay.io/ambient_code/vteam_claude_runner:latest=${DEFAULT_RUNNER_IMAGE}

# Build and apply manifests
echo -e "${BLUE}Building and applying manifests...${NC}"
kustomize build . | oc apply -f -

# Create or update MCP servers ConfigMap with optional Atlassian MCP support
create_mcp_servers_configmap "$NAMESPACE" "$ATLASSIAN_MCP_URL"

# Check if namespace exists and is active
echo -e "${YELLOW}Checking namespace status...${NC}"
if ! oc get namespace ${NAMESPACE} >/dev/null 2>&1; then
    echo -e "${RED}❌ Namespace ${NAMESPACE} does not exist${NC}"
    exit 1
fi

# Check if namespace is active
NAMESPACE_PHASE=$(oc get namespace ${NAMESPACE} -o jsonpath='{.status.phase}')
if [ "$NAMESPACE_PHASE" != "Active" ]; then
    echo -e "${RED}❌ Namespace ${NAMESPACE} is not active (phase: ${NAMESPACE_PHASE})${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Namespace ${NAMESPACE} is active${NC}"

# Switch to the target namespace
echo -e "${BLUE}Switching to namespace ${NAMESPACE}...${NC}"
oc project ${NAMESPACE}

# Create API key secret (kustomize creates empty secret, we populate it)
echo -e "${BLUE}Creating API key secret...${NC}"
oc patch secret ambient-code-secrets -n ${NAMESPACE} -p "{\"stringData\":{\"anthropic-api-key\":\"$ANTHROPIC_API_KEY\"}}" || {
    echo -e "${YELLOW}Secret patch failed, ensuring secret exists and retrying...${NC}"
    sleep 1
    oc patch secret ambient-code-secrets -n ${NAMESPACE} -p "{\"stringData\":{\"anthropic-api-key\":\"$ANTHROPIC_API_KEY\"}}"
}

# Apply git configuration if we created a patch
if [[ -f "/tmp/git-config-patch.yaml" ]]; then
    echo -e "${BLUE}Applying Git configuration...${NC}"
    oc apply -f /tmp/git-config-patch.yaml
    rm -f /tmp/git-config-patch.yaml
fi

# Update operator deployment with custom runner image
echo -e "${BLUE}Updating operator with custom runner image...${NC}"
oc patch deployment agentic-operator -n ${NAMESPACE} -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"agentic-operator\",\"env\":[{\"name\":\"AMBIENT_CODE_RUNNER_IMAGE\",\"value\":\"${DEFAULT_RUNNER_IMAGE}\"}]}]}}}}" --type=strategic

echo ""
echo -e "${GREEN}✅ Deployment completed!${NC}"
echo ""

# Wait for deployments to be ready
echo -e "${YELLOW}Waiting for deployments to be ready...${NC}"
oc rollout status deployment/backend-api --namespace=${NAMESPACE} --timeout=300s
oc rollout status deployment/agentic-operator --namespace=${NAMESPACE} --timeout=300s
oc rollout status deployment/frontend --namespace=${NAMESPACE} --timeout=300s

# Get service information
echo -e "${BLUE}Getting service information...${NC}"
echo ""
echo -e "${GREEN}🎉 Deployment successful!${NC}"
echo -e "${GREEN}========================${NC}"
echo -e "Namespace: ${BLUE}${NAMESPACE}${NC}"
echo ""

# Show pod status
echo -e "${BLUE}Pod Status:${NC}"
oc get pods -n ${NAMESPACE}
echo ""

# Show services
echo -e "${BLUE}Services:${NC}"
oc get services -n ${NAMESPACE}
echo ""

echo -e "${YELLOW}Next steps:${NC}"
echo -e "1. Access the RFE workflow frontend:"
echo -e "   ${BLUE}oc port-forward svc/frontend-service 3000:3000 -n ${NAMESPACE}${NC}"
echo -e "   Then open: http://localhost:3000"
echo -e "   Create new RFE workflows at: http://localhost:3000/rfe/new"
echo -e "2. Monitor the deployment:"
echo -e "   ${BLUE}oc get pods -n ${NAMESPACE} -w${NC}"
echo -e "3. View logs:"
echo -e "   ${BLUE}oc logs -f deployment/backend-api -n ${NAMESPACE}${NC}"
echo -e "   ${BLUE}oc logs -f deployment/agentic-operator -n ${NAMESPACE}${NC}"
echo -e "4. Monitor RFE workflows:"
echo -e "   ${BLUE}oc get agenticsessions -n ${NAMESPACE}${NC}"
if [[ -n "$ATLASSIAN_MCP_URL" ]]; then
echo -e "5. Atlassian MCP server:"
echo -e "   ${YELLOW}Note: Atlassian MCP is configured but requires a separate deployment${NC}"
echo -e "   ${YELLOW}Ensure atlassian-mcp service is available at ${ATLASSIAN_MCP_URL}${NC}"
fi
echo ""

# Restore kustomization if we modified it
echo -e "${BLUE}Restoring kustomization defaults...${NC}"
if [ "$NAMESPACE" != "ambient-code" ]; then
    kustomize edit set namespace ambient-code
fi
# Restore default images
kustomize edit set image quay.io/ambient_code/vteam_backend:latest=quay.io/ambient_code/vteam_backend:latest
kustomize edit set image quay.io/ambient_code/vteam_frontend:latest=quay.io/ambient_code/vteam_frontend:latest
kustomize edit set image quay.io/ambient_code/vteam_operator:latest=quay.io/ambient_code/vteam_operator:latest
kustomize edit set image quay.io/ambient_code/vteam_claude_runner:latest=quay.io/ambient_code/vteam_claude_runner:latest

echo -e "${GREEN}🎯 Ready to create RFE workflows with multi-agent collaboration!${NC}"
