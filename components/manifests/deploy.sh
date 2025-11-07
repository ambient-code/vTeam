#!/bin/bash

# OpenShift Deployment Script for vTeam Ambient Agentic Runner
# Usage: ./deploy.sh
# Or with environment variables: NAMESPACE=my-namespace ./deploy.sh
# Note: This script deploys pre-built images. Build and push images first.

set -e

# Always run from the script's directory (manifests root)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Load .env file if it exists (optional for local CRC setups)
if [ -f ".env" ]; then
    set -a  # automatically export all variables
    source .env
    set +a
fi

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

# Helper: Run the OAuth setup (Route host, OAuthClient, Secret)
oauth_setup() {
    echo -e "${YELLOW}Configuring OpenShift OAuth for the frontend...${NC}"

    # Determine Route name (try known names then fallback by label)
    ROUTE_NAME_CANDIDATE="${ROUTE_NAME:-}"
    if [[ -z "$ROUTE_NAME_CANDIDATE" ]]; then
        if oc get route frontend-route -n ${NAMESPACE} >/dev/null 2>&1; then
            ROUTE_NAME_CANDIDATE="frontend-route"
        elif oc get route frontend -n ${NAMESPACE} >/dev/null 2>&1; then
            ROUTE_NAME_CANDIDATE="frontend"
        else
            ROUTE_NAME_CANDIDATE=$(oc get route -n ${NAMESPACE} -l app=frontend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
        fi
    fi

    if [[ -z "$ROUTE_NAME_CANDIDATE" ]]; then
        echo -e "${RED}❌ Could not find a Route for the frontend in namespace ${NAMESPACE}.${NC}"
        echo -e "${YELLOW}Make sure manifests are applied and a Route exists (e.g., name 'frontend-route').${NC}"
        return 1
    fi
    ROUTE_NAME="$ROUTE_NAME_CANDIDATE"
    echo -e "${BLUE}Using Route: ${ROUTE_NAME}${NC}"

    # Ensure Route host is set to <namespace>.<cluster apps domain>
    echo -e "${BLUE}Setting Route host if needed...${NC}"
    ROUTE_DOMAIN=$(oc get ingresses.config cluster -o jsonpath='{.spec.domain}')
    if [[ -z "$ROUTE_DOMAIN" ]]; then
        echo -e "${YELLOW}Could not detect cluster apps domain; skipping Route host patch.${NC}"
    else
        DESIRED_HOST="${NAMESPACE}.${ROUTE_DOMAIN}"
        CURRENT_HOST=$(oc -n ${NAMESPACE} get route ${ROUTE_NAME} -o jsonpath='{.spec.host}' 2>/dev/null || true)
        if [[ -z "$CURRENT_HOST" || "$CURRENT_HOST" != "$DESIRED_HOST" ]]; then
            echo -e "${BLUE}Patching Route host to ${DESIRED_HOST}...${NC}"
            oc -n ${NAMESPACE} patch route ${ROUTE_NAME} --type=merge -p "{\"spec\":{\"host\":\"${DESIRED_HOST}\"}}"
        else
            echo -e "${GREEN}Route host already set to ${CURRENT_HOST}${NC}"
        fi
    fi

    ROUTE_HOST=$(oc -n ${NAMESPACE} get route ${ROUTE_NAME} -o jsonpath='{.spec.host}' 2>/dev/null || true)
    if [[ -z "$ROUTE_HOST" ]]; then
        echo -e "${YELLOW}Route host is empty; OAuthClient redirect URI may be incomplete.${NC}"
    else
        echo -e "${GREEN}Route host: https://${ROUTE_HOST}${NC}"
    fi

    # Create/Update cluster-scoped OAuthClient (requires cluster-admin)
    echo -e "${BLUE}Creating/Updating OAuthClient 'ambient-frontend'...${NC}"
    cat > /tmp/ambient-frontend-oauthclient.yaml <<EOF
apiVersion: oauth.openshift.io/v1
kind: OAuthClient
metadata:
  name: ambient-frontend
secret: ${CLIENT_SECRET_VALUE}
redirectURIs:
- https://${ROUTE_HOST}/oauth/callback
grantMethod: auto
EOF
    set +e
    oc apply -f /tmp/ambient-frontend-oauthclient.yaml
    OAUTH_APPLY_RC=$?
    set -e
    rm -f /tmp/ambient-frontend-oauthclient.yaml
    if [[ ${OAUTH_APPLY_RC} -ne 0 ]]; then
        echo -e "${YELLOW}⚠️ Could not create/update cluster-scoped OAuthClient. You likely need cluster-admin.${NC}"
        echo -e "${YELLOW}Ask an admin to run:${NC}"
        echo "oc apply -f - <<'EOF'"
        echo "apiVersion: oauth.openshift.io/v1"
        echo "kind: OAuthClient"
        echo "metadata:"
        echo "  name: ambient-frontend"
        echo "secret: ${CLIENT_SECRET_VALUE}"
        echo "redirectURIs:"
        echo "- https://${ROUTE_HOST}/oauth/callback"
        echo "grantMethod: auto"
        echo "EOF"
    else
        echo -e "${GREEN}✅ OAuthClient configured${NC}"
    fi

    # Create/Update the frontend OAuth secret in the namespace
    echo -e "${BLUE}Creating/Updating Secret 'frontend-oauth-config'...${NC}"
    oc -n ${NAMESPACE} create secret generic frontend-oauth-config \
      --from-literal=client-secret="${CLIENT_SECRET_VALUE}" \
      --from-literal=cookie_secret="${COOKIE_SECRET_VALUE}" \
      --dry-run=client -o yaml | oc apply -f -
    echo -e "${GREEN}✅ Secret configured${NC}"

    # Restart frontend to pick up new secret
    echo -e "${BLUE}Restarting frontend deployment...${NC}"
    oc -n ${NAMESPACE} rollout restart deployment/frontend
}

# Configuration
NAMESPACE="${NAMESPACE:-ambient-code}"
# Allow overriding images via CONTAINER_REGISTRY/IMAGE_TAG or explicit DEFAULT_*_IMAGE
CONTAINER_REGISTRY="${CONTAINER_REGISTRY:-quay.io/ambient_code}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
DEFAULT_BACKEND_IMAGE="${DEFAULT_BACKEND_IMAGE:-${CONTAINER_REGISTRY}/vteam_backend:${IMAGE_TAG}}"
DEFAULT_FRONTEND_IMAGE="${DEFAULT_FRONTEND_IMAGE:-${CONTAINER_REGISTRY}/vteam_frontend:${IMAGE_TAG}}"
DEFAULT_OPERATOR_IMAGE="${DEFAULT_OPERATOR_IMAGE:-${CONTAINER_REGISTRY}/vteam_operator:${IMAGE_TAG}}"
DEFAULT_RUNNER_IMAGE="${DEFAULT_RUNNER_IMAGE:-${CONTAINER_REGISTRY}/vteam_claude_runner:${IMAGE_TAG}}"
# Content service image (defaults to same as backend, but can be overridden)
CONTENT_SERVICE_IMAGE="${CONTENT_SERVICE_IMAGE:-${DEFAULT_BACKEND_IMAGE}}"

# Handle uninstall/clean command early
if [ "${1:-}" = "uninstall" ] || [ "${1:-}" = "clean" ]; then
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

    # Delete using kustomize (from production overlay)
    cd overlays/production
    if [ "$NAMESPACE" != "ambient-code" ]; then
        kustomize edit set namespace "$NAMESPACE"
    fi

    kustomize build . | oc delete -f - --ignore-not-found=true

    # Restore kustomization if we modified it
    if [ "$NAMESPACE" != "ambient-code" ]; then
        kustomize edit set namespace ambient-code
    fi
    cd ../..

    echo -e "${GREEN}✅ vTeam uninstalled from namespace ${NAMESPACE}${NC}"
    echo -e "${YELLOW}Note: Namespace ${NAMESPACE} still exists. Delete manually if needed:${NC}"
    echo -e "   ${BLUE}oc delete namespace ${NAMESPACE}${NC}"
    exit 0
fi

# Handle secrets-only command (OAuth setup only)
if [ "${1:-}" = "secrets" ]; then
    echo -e "${YELLOW}Running OAuth secrets setup only...${NC}"

    # Check prerequisites for secrets subcommand
    if ! command_exists oc; then
        echo -e "${RED}❌ OpenShift CLI (oc) not found. Please install it first.${NC}"
        exit 1
    fi
    if ! oc whoami >/dev/null 2>&1; then
        echo -e "${RED}❌ Not logged in to OpenShift. Please run 'oc login' first.${NC}"
        exit 1
    fi

    # Generate secrets values like in full deploy
    OAUTH_ENV_FILE="oauth-secret.env"
    CLIENT_SECRET_VALUE="${OCP_OAUTH_CLIENT_SECRET:-}"
    COOKIE_SECRET_VALUE="${OCP_OAUTH_COOKIE_SECRET:-}"
    if [[ -z "$CLIENT_SECRET_VALUE" ]]; then
        CLIENT_SECRET_VALUE=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)
    fi
    COOKIE_LEN=${#COOKIE_SECRET_VALUE}
    if [[ -z "$COOKIE_SECRET_VALUE" || ( $COOKIE_LEN -ne 16 && $COOKIE_LEN -ne 24 && $COOKIE_LEN -ne 32 ) ]]; then
        COOKIE_SECRET_VALUE=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)
    fi
    cat > "$OAUTH_ENV_FILE" << EOF
client-secret=${CLIENT_SECRET_VALUE}
cookie_secret=${COOKIE_SECRET_VALUE}
EOF

    # Ensure namespace exists and switch
    if ! oc get namespace ${NAMESPACE} >/dev/null 2>&1; then
        echo -e "${RED}❌ Namespace ${NAMESPACE} does not exist. Deploy manifests first.${NC}"
        rm -f "$OAUTH_ENV_FILE"
        exit 1
    fi
    oc project ${NAMESPACE}

    # Perform OAuth setup
    if ! oauth_setup; then
        echo -e "${YELLOW}OAuth setup completed with warnings/errors. See messages above.${NC}"
    fi

    # Cleanup
    rm -f "$OAUTH_ENV_FILE"
    echo -e "${GREEN}✅ Secrets subcommand completed${NC}"
    exit 0
fi

echo -e "${BLUE}🚀 vTeam Ambient Agentic Runner - OpenShift Deployment${NC}"
echo -e "${BLUE}====================================================${NC}"
echo -e "Namespace: ${GREEN}${NAMESPACE}${NC}"
echo -e "Backend Image: ${GREEN}${DEFAULT_BACKEND_IMAGE}${NC}"
echo -e "Frontend Image: ${GREEN}${DEFAULT_FRONTEND_IMAGE}${NC}"
echo -e "Operator Image: ${GREEN}${DEFAULT_OPERATOR_IMAGE}${NC}"
echo -e "Runner Image: ${GREEN}${DEFAULT_RUNNER_IMAGE}${NC}"
echo -e "Content Service Image: ${GREEN}${CONTENT_SERVICE_IMAGE}${NC}"
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

# Prepare oauth secret env file for kustomize secretGenerator
echo -e "${YELLOW}Preparing oauth secret env for kustomize...${NC}"
OAUTH_ENV_FILE="oauth-secret.env"
CLIENT_SECRET_VALUE="${OCP_OAUTH_CLIENT_SECRET:-}"
COOKIE_SECRET_VALUE="${OCP_OAUTH_COOKIE_SECRET:-}"
if [[ -z "$CLIENT_SECRET_VALUE" ]]; then
    CLIENT_SECRET_VALUE=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)
fi
# cookie_secret must be exactly 16, 24, or 32 bytes. Use 32 ASCII bytes by default.
if [[ -z "$COOKIE_SECRET_VALUE" ]]; then
    COOKIE_SECRET_VALUE=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)
fi
# If provided via .env, ensure it meets required length
COOKIE_LEN=${#COOKIE_SECRET_VALUE}
if [[ $COOKIE_LEN -ne 16 && $COOKIE_LEN -ne 24 && $COOKIE_LEN -ne 32 ]]; then
    echo -e "${YELLOW}Provided OCP_OAUTH_COOKIE_SECRET length ($COOKIE_LEN) is invalid; regenerating 32-byte value...${NC}"
    COOKIE_SECRET_VALUE=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)
fi
cat > "$OAUTH_ENV_FILE" << EOF
client-secret=${CLIENT_SECRET_VALUE}
cookie_secret=${COOKIE_SECRET_VALUE}
EOF
echo -e "${GREEN}✅ Generated ${OAUTH_ENV_FILE}${NC}"
echo ""


# Deploy using kustomize
echo -e "${YELLOW}Deploying to OpenShift using Kustomize...${NC}"

# Use production overlay
cd overlays/production

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

# Return to manifests root
cd ../..

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

###############################################
# OAuth setup: Route host, OAuthClient, Secret
###############################################
if ! oauth_setup; then
    echo -e "${YELLOW}OAuth setup completed with warnings/errors. You may need a cluster-admin to apply the OAuthClient.${NC}"
fi

# Apply git configuration if we created a patch
if [[ -f "/tmp/git-config-patch.yaml" ]]; then
    echo -e "${BLUE}Applying Git configuration...${NC}"
    oc apply -f /tmp/git-config-patch.yaml
    rm -f /tmp/git-config-patch.yaml
fi

# Update operator deployment with custom runner image
echo -e "${BLUE}Updating operator with custom runner image...${NC}"
oc patch deployment agentic-operator -n ${NAMESPACE} -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"agentic-operator\",\"env\":[{\"name\":\"AMBIENT_CODE_RUNNER_IMAGE\",\"value\":\"${DEFAULT_RUNNER_IMAGE}\"}]}]}}}}" --type=strategic

# Update backend deployment with content service image
echo -e "${BLUE}Updating backend with content service image...${NC}"
oc patch deployment backend-api -n ${NAMESPACE} -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"backend-api\",\"env\":[{\"name\":\"CONTENT_SERVICE_IMAGE\",\"value\":\"${CONTENT_SERVICE_IMAGE}\"},{\"name\":\"IMAGE_PULL_POLICY\",\"value\":\"Always\"}]}]}}}}" --type=strategic

echo ""
echo -e "${GREEN}✅ Deployment completed!${NC}"
echo ""

# Wait for deployments to be ready
echo -e "${YELLOW}Waiting for deployments to be ready...${NC}"
oc rollout status deployment/backend-api --namespace=${NAMESPACE} --timeout=300s
oc rollout status deployment/agentic-operator --namespace=${NAMESPACE} --timeout=300s
oc rollout status deployment/frontend --namespace=${NAMESPACE} --timeout=300s

# Get service and route information
echo -e "${BLUE}Getting service and route information...${NC}"
echo ""
echo -e "${GREEN}🎉 Deployment successful!${NC}"
echo -e "${GREEN}========================${NC}"
echo -e "Namespace: ${BLUE}${NAMESPACE}${NC}"
echo ""

# Show pod status
echo -e "${BLUE}Pod Status:${NC}"
oc get pods -n ${NAMESPACE}
echo ""

# Show services and route
echo -e "${BLUE}Services:${NC}"
oc get services -n ${NAMESPACE}
echo ""
echo -e "${BLUE}Routes:${NC}"
oc get route -n ${NAMESPACE} || true
if [[ -z "${ROUTE_NAME:-}" ]]; then
    if oc get route frontend-route -n ${NAMESPACE} >/dev/null 2>&1; then
        ROUTE_NAME="frontend-route"
    elif oc get route frontend -n ${NAMESPACE} >/dev/null 2>&1; then
        ROUTE_NAME="frontend"
    fi
fi
ROUTE_HOST=$(oc get route ${ROUTE_NAME:-frontend-route} -n ${NAMESPACE} -o jsonpath='{.spec.host}' 2>/dev/null || true)
echo ""

# Cleanup generated files
echo -e "${BLUE}Cleaning up generated files...${NC}"
rm -f "$OAUTH_ENV_FILE"

echo -e "${YELLOW}Next steps:${NC}"
if [[ -n "${ROUTE_HOST}" ]]; then
    echo -e "1. Access the frontend via Route:"
    echo -e "   ${BLUE}https://${ROUTE_HOST}${NC}"
else
    echo -e "1. Access the frontend (fallback via port-forward):"
    echo -e "   ${BLUE}oc port-forward svc/frontend-service 3000:3000 -n ${NAMESPACE}${NC}"
    echo -e "   Then open: http://localhost:3000"
fi
echo -e "2. Configure secrets in the UI (Runner/API keys, project settings)."
echo -e "   Open the app and follow Settings → Runner Secrets."
echo -e "3. Monitor the deployment:"
echo -e "   ${BLUE}oc get pods -n ${NAMESPACE} -w${NC}"
echo -e "4. View logs:"
echo -e "   ${BLUE}oc logs -f deployment/backend-api -n ${NAMESPACE}${NC}"
echo -e "   ${BLUE}oc logs -f deployment/agentic-operator -n ${NAMESPACE}${NC}"
echo -e "4. Monitor RFE workflows:"
echo -e "   ${BLUE}oc get agenticsessions -n ${NAMESPACE}${NC}"
echo ""

# Restore kustomization if we modified it
echo -e "${BLUE}Restoring kustomization defaults...${NC}"
cd overlays/production
if [ "$NAMESPACE" != "ambient-code" ]; then
    kustomize edit set namespace ambient-code
fi
# Restore default images
kustomize edit set image quay.io/ambient_code/vteam_backend:latest=quay.io/ambient_code/vteam_backend:latest
kustomize edit set image quay.io/ambient_code/vteam_frontend:latest=quay.io/ambient_code/vteam_frontend:latest
kustomize edit set image quay.io/ambient_code/vteam_operator:latest=quay.io/ambient_code/vteam_operator:latest
kustomize edit set image quay.io/ambient_code/vteam_claude_runner:latest=quay.io/ambient_code/vteam_claude_runner:latest
cd ../..

echo -e "${GREEN}🎯 Ready to create RFE workflows with multi-agent collaboration!${NC}"
