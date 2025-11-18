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
OPERATOR_DIR="${REPO_ROOT}/components/operator"
CRDS_DIR="${REPO_ROOT}/components/manifests/crds"

###############
# Environment File Loading
###############
load_custom_env() {
  local default_env_file="${REPO_ROOT}/components/manifests/env.example"
  local custom_env_file=""
  
  # Check if there's a .env file in the current directory
  if [[ -f ".env" ]]; then
    custom_env_file=".env"
  elif [[ -f "${REPO_ROOT}/.env" ]]; then
    custom_env_file="${REPO_ROOT}/.env"
  fi
  
  # Prompt user for custom .env file
  echo ""
  log "Environment configuration setup"
  if [[ -n "$custom_env_file" ]]; then
    echo "Found existing .env file: $custom_env_file"
    read -p "Use this .env file? [Y/n]: " -r use_existing
    if [[ "$use_existing" =~ ^[Nn]$ ]]; then
      custom_env_file=""
    fi
  fi
  
  if [[ -z "$custom_env_file" ]]; then
    echo "You can provide a custom .env file to override default configurations."
    echo "Available variables to customize:"
    echo "  - CRC_CPUS (default: $CRC_CPUS)"
    echo "  - CRC_MEMORY (default: $CRC_MEMORY)"  
    echo "  - CRC_DISK (default: $CRC_DISK)"
    echo "  - PROJECT_NAME (default: $PROJECT_NAME)"
    echo "  - DEV_MODE (default: $DEV_MODE)"
    echo ""
    echo "Example .env file location: $default_env_file"
    echo ""
    read -p "Enter path to custom .env file (or press Enter to use defaults): " -r custom_env_file
  fi
  
  # Load the custom .env file if provided and exists
  if [[ -n "$custom_env_file" ]] && [[ -f "$custom_env_file" ]]; then
    log "Loading custom environment from: $custom_env_file"
    set -a  # automatically export all variables
    # shellcheck source=/dev/null
    source "$custom_env_file"
    set +a
    
    # Show what was loaded
    echo "Loaded configuration:"
    echo "  CRC_CPUS: $CRC_CPUS"
    echo "  CRC_MEMORY: $CRC_MEMORY"
    echo "  CRC_DISK: $CRC_DISK"
    echo "  PROJECT_NAME: $PROJECT_NAME"
    echo "  DEV_MODE: $DEV_MODE"
    echo ""
  elif [[ -n "$custom_env_file" ]]; then
    warn "Custom .env file not found: $custom_env_file"
    warn "Continuing with default configuration..."
    echo ""
  else
    log "Using default configuration"
    echo ""
  fi
}

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
    case "$1" in
      crc)
        err "Install CRC:"
        err "  macOS: brew install crc"
        err "  Linux: https://crc.dev/crc/getting_started/getting_started/installing/"
        ;;
      jq)
        err "Install jq:"
        err "  macOS: brew install jq"
        err "  Linux: sudo apt install jq  # or yum install jq"
        ;;
    esac
    exit 1
  fi
}

check_system_resources() {
  log "Checking system resources..."
  
  # Check OS compatibility
  local os_name="$(uname -s)"
  case "$os_name" in
    Darwin|Linux)
      log "OS detected: $os_name ✓"
      ;;
    *)
      err "Unsupported OS: $os_name"
      err "CRC requires macOS or Linux. For Windows, use WSL2."
      exit 1
      ;;
  esac
  
  # Check available memory (basic check)
  if [[ -f /proc/meminfo ]]; then
    local available_mem_kb
    available_mem_kb=$(grep MemAvailable /proc/meminfo | awk '{print $2}')
    local required_mem_kb=$((CRC_MEMORY * 1024))
    if [[ "$available_mem_kb" -lt "$required_mem_kb" ]]; then
      warn "Available memory (${available_mem_kb}KB) may be insufficient for CRC (${required_mem_kb}KB)"
      warn "Consider reducing: CRC_MEMORY=6144 make dev-start"
    fi
  fi
  
  # Check disk space in home directory
  local available_space_gb
  if command -v df >/dev/null 2>&1; then
    available_space_gb=$(df "$HOME" | awk 'NR==2 {print $4}' | sed 's/G//')
    if [[ "$available_space_gb" -lt "$CRC_DISK" ]] 2>/dev/null; then
      warn "Available disk space (~${available_space_gb}GB) may be insufficient for CRC (${CRC_DISK}GB)"
      warn "Consider reducing: CRC_DISK=30 make dev-start"
    fi
  fi
  
  # Check virtualization (basic check for Linux)
  if [[ -f /proc/cpuinfo ]] && ! grep -q -E '(vmx|svm)' /proc/cpuinfo; then
    warn "Virtualization may not be enabled. CRC requires VT-x/AMD-V."
    warn "Enable virtualization in BIOS/UEFI settings."
  fi
  
  # Check if ports might be in use (basic check)
  if command -v lsof >/dev/null 2>&1; then
    for port in 6443 443 80; do
      if lsof -iTCP:$port -sTCP:LISTEN >/dev/null 2>&1; then
        warn "Port $port appears to be in use - may conflict with CRC"
      fi
    done
  fi
}

#########################
# CRC Setup (from original)
#########################
check_crc_setup() {
  # Check if CRC has been set up
  if ! crc version >/dev/null 2>&1; then
    err "CRC not properly installed or not in PATH"
    exit 1
  fi
  
  # Check if pull secret is configured
  local pull_secret_path="$HOME/.crc/pull-secret.json"
  if [[ ! -f "$pull_secret_path" ]]; then
    err "Pull secret not found. You need to:"
    err "1. Get your pull secret from https://console.redhat.com/openshift/create/local"
    err "2. Save it to $pull_secret_path"
    exit 1
  fi
  
  # Configure CRC if not already done
  if ! crc config get enable-cluster-monitoring >/dev/null 2>&1; then
    log "Running initial CRC setup..."
    crc setup
  fi
  
  # Apply resource configuration
  log "Configuring CRC resources (${CRC_CPUS} CPUs, ${CRC_MEMORY}MB RAM, ${CRC_DISK}GB disk)..."
  crc config set cpus "$CRC_CPUS" >/dev/null
  crc config set memory "$CRC_MEMORY" >/dev/null
  crc config set disk-size "$CRC_DISK" >/dev/null
  crc config set pull-secret-file "$pull_secret_path" >/dev/null
  crc config set enable-cluster-monitoring false >/dev/null
}

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
  
  # Apply ambient-code labels for operator to recognize managed namespace
  oc label namespace "$PROJECT_NAME" ambient-code.io/managed=true --overwrite
  log "Namespace labeled as managed for operator"
}

apply_crds() {
  log "Applying CRDs..."
  oc apply -f "${CRDS_DIR}/agenticsessions-crd.yaml"
  oc apply -f "${CRDS_DIR}/projectsettings-crd.yaml"
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

apply_operator_rbac() {
  log "Applying operator RBAC (service account and permissions)..."
  oc apply -f "${MANIFESTS_DIR}/operator-rbac.yaml" -n "$PROJECT_NAME"
}

#########################
# Build and Deploy
#########################
build_and_deploy() {
  log "Creating BuildConfigs..."
  oc apply -f "${MANIFESTS_DIR}/build-configs.yaml" -n "$PROJECT_NAME"
  oc apply -f "${MANIFESTS_DIR}/operator-build-config.yaml" -n "$PROJECT_NAME"
  
  # Start builds
  log "Building backend image..."
  oc start-build vteam-backend --from-dir="$BACKEND_DIR" --wait -n "$PROJECT_NAME"
  
  log "Building frontend image..."  
  oc start-build vteam-frontend --from-dir="$FRONTEND_DIR" --wait -n "$PROJECT_NAME"
  
  log "Building operator image..."
  oc start-build vteam-operator --from-dir="$OPERATOR_DIR" --wait -n "$PROJECT_NAME"
  
  # Deploy services
  log "Creating backend PVC..."
  oc apply -f "${MANIFESTS_DIR}/backend-pvc.yaml" -n "$PROJECT_NAME"

  log "Deploying backend..."
  oc apply -f "${MANIFESTS_DIR}/backend-deployment.yaml" -n "$PROJECT_NAME"
  
  log "Deploying frontend..."
  oc apply -f "${MANIFESTS_DIR}/frontend-deployment.yaml" -n "$PROJECT_NAME"
  
  log "Creating backend service alias for operator..."
  oc apply -f "${MANIFESTS_DIR}/backend-service-alias.yaml" -n "$PROJECT_NAME"

  log "Applying operator configuration (CRC - Vertex disabled)..."
  oc apply -f "${REPO_ROOT}/components/manifests/operator-config-crc.yaml" -n "$PROJECT_NAME"

  log "Deploying operator..."
  oc apply -f "${REPO_ROOT}/components/manifests/operator-deployment.yaml" -n "$PROJECT_NAME"
}

wait_for_ready() {
  log "Waiting for deployments to be ready..."
  oc rollout status deployment/vteam-backend --timeout=300s -n "$PROJECT_NAME"
  oc rollout status deployment/vteam-frontend --timeout=300s -n "$PROJECT_NAME"
  oc rollout status deployment/vteam-operator --timeout=300s -n "$PROJECT_NAME"
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

# Optional tools with warnings  
if ! command -v git >/dev/null 2>&1; then
  warn "Git not found - needed if you haven't cloned the repo yet"
fi

check_system_resources

# Load custom environment configuration
load_custom_env

log "Starting CRC-based local development environment..."

check_crc_setup
ensure_crc_cluster
configure_oc_context
ensure_project
apply_crds
apply_rbac
apply_operator_rbac
build_and_deploy
wait_for_ready
show_results

log "To stop: make dev-stop"
