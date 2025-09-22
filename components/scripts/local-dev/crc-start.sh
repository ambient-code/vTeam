#!/bin/bash

set -euo pipefail

# CRC-based local dev following manifests/ pattern:
# - Clean, modular approach using separate manifest files
# - Mirrors production manifests structure
# - Simplified and maintainable

###############
# Configuration
###############
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
MANIFESTS_DIR="${SCRIPT_DIR}/manifests"
STATE_DIR="${SCRIPT_DIR}/state"
mkdir -p "${STATE_DIR}"

# CRC Configuration
CRC_CPUS="${CRC_CPUS:-4}"
CRC_MEMORY="${CRC_MEMORY:-11264}"
CRC_DISK="${CRC_DISK:-50}"

# Project Configuration
PROJECT_NAME="${PROJECT_NAME:-vteam-dev}"
DEV_MODE="${DEV_MODE:-false}"

# Component directories
BACKEND_DIR="${REPO_ROOT}/components/backend"
FRONTEND_DIR="${REPO_ROOT}/components/frontend"
CRDS_DIR="${REPO_ROOT}/components/manifests/crds"

###############
# Utilities
###############
log() { printf "[%s] %s\n" "$(date '+%H:%M:%S')" "$*"; }
warn() { printf "\033[1;33m%s\033[0m\n" "$*"; }
err() { printf "\033[0;31m%s\033[0m\n" "$*"; }
success() { printf "\033[0;32m%s\033[0m\n" "$*"; }

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    err "Missing required command: $1"
    exit 1
  fi
}

#########################
# CRC Setup (from original)
#########################
ensure_crc_cluster() {
  local crc_status
  crc_status=$(crc status -o json 2>/dev/null | jq -r '.crcStatus // "Stopped"' 2>/dev/null || echo "Stopped")
  
  case "$crc_status" in
    "Running")
      log "CRC cluster is already running"
      ;;
    *)
      log "Starting CRC cluster..."
      if ! crc start; then
        err "Failed to start CRC cluster"
        exit 1
      fi
      ;;
  esac
}

configure_oc_context() {
  log "Configuring OpenShift CLI context..."
  eval "$(crc oc-env)"
  
  local admin_pass
  admin_pass=$(crc console --credentials 2>/dev/null | grep kubeadmin | sed -n 's/.*-p \([^ ]*\).*/\1/p')
  
  if [[ -z "$admin_pass" ]]; then
    err "Failed to get admin credentials"
    exit 1
  fi
  
  oc login -u kubeadmin -p "$admin_pass" "https://api.crc.testing:6443" --insecure-skip-tls-verify=true
}

#########################
# OpenShift Project Setup
#########################
ensure_project() {
  log "Ensuring OpenShift project '$PROJECT_NAME'..."
  
  if ! oc get project "$PROJECT_NAME" >/dev/null 2>&1; then
    oc new-project "$PROJECT_NAME" --display-name="vTeam Development"
  else
    oc project "$PROJECT_NAME"
  fi
  
  # Apply ambient-code labels like production
  oc label project "$PROJECT_NAME" ambient-code.io/managed=true --overwrite >/dev/null 2>&1 || true
}

apply_crds() {
  log "Applying CRDs..."
  oc apply -f "${CRDS_DIR}/agenticsessions-crd.yaml"
  oc apply -f "${CRDS_DIR}/projectsettings-crd.yaml"  
  oc apply -f "${CRDS_DIR}/rfeworkflows-crd.yaml"
}

apply_rbac() {
  log "Applying RBAC (backend service account and permissions)..."
  oc apply -f "${MANIFESTS_DIR}/backend-rbac.yaml" -n "$PROJECT_NAME"
  oc apply -f "${MANIFESTS_DIR}/dev-users.yaml" -n "$PROJECT_NAME"
  
  log "Creating frontend authentication..."
  oc apply -f "${MANIFESTS_DIR}/frontend-auth.yaml" -n "$PROJECT_NAME"
  
  # Wait for token secret to be populated
  log "Waiting for frontend auth token to be created..."
  oc wait --for=condition=complete secret/frontend-auth-token --timeout=60s -n "$PROJECT_NAME" || true
}

#########################
# Build and Deploy
#########################
build_and_deploy() {
  log "Creating BuildConfigs..."
  oc apply -f "${MANIFESTS_DIR}/build-configs.yaml" -n "$PROJECT_NAME"
  
  # Start builds
  log "Building backend image..."
  oc start-build vteam-backend --from-dir="$BACKEND_DIR" --wait -n "$PROJECT_NAME"
  
  log "Building frontend image..."  
  oc start-build vteam-frontend --from-dir="$FRONTEND_DIR" --wait -n "$PROJECT_NAME"
  
  # Deploy services
  log "Deploying backend..."
  oc apply -f "${MANIFESTS_DIR}/backend-deployment.yaml" -n "$PROJECT_NAME"
  
  log "Deploying frontend..."
  oc apply -f "${MANIFESTS_DIR}/frontend-deployment.yaml" -n "$PROJECT_NAME"
}

wait_for_ready() {
  log "Waiting for deployments to be ready..."
  oc rollout status deployment/vteam-backend --timeout=300s -n "$PROJECT_NAME"
  oc rollout status deployment/vteam-frontend --timeout=300s -n "$PROJECT_NAME"
}

show_results() {
  BACKEND_URL="https://$(oc get route vteam-backend -o jsonpath='{.spec.host}' -n "$PROJECT_NAME")"
  FRONTEND_URL="https://$(oc get route vteam-frontend -o jsonpath='{.spec.host}' -n "$PROJECT_NAME")"
  
  echo ""
  success "OpenShift Local development environment ready!"
  echo "  Backend:   $BACKEND_URL/health"
  echo "  Frontend:  $FRONTEND_URL"
  echo "  Project:   $PROJECT_NAME"
  echo "  Console:   $(crc console --url 2>/dev/null)"
  echo ""
  
  # Store URLs for testing
  cat > "${STATE_DIR}/urls.env" << EOF
BACKEND_URL=$BACKEND_URL
FRONTEND_URL=$FRONTEND_URL
PROJECT_NAME=$PROJECT_NAME
EOF
}

#########################
# Execution
#########################
log "Checking prerequisites..."
need_cmd crc
need_cmd jq
need_cmd oc

log "Starting CRC-based local development environment..."

ensure_crc_cluster
configure_oc_context
ensure_project
apply_crds
apply_rbac
build_and_deploy
wait_for_ready
show_results

log "To stop: make dev-stop"
