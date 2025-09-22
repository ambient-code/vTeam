#!/bin/bash

set -euo pipefail

# One-shot local dev (Option A):
# - Creates/uses a local Kind cluster (ambient-agentic)
# - Applies CRDs and ensures a dev namespace exists and is labeled
# - Runs backend (Go) on port 8080 and frontend (Next.js) on port 3000 locally
# - Idempotent: safe to re-run; reuses cluster; restarts missing processes
# - Cross-platform: macOS (Darwin) and Fedora/RHEL (Linux)

###############
# Configuration
###############
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
LOG_DIR="${SCRIPT_DIR}/logs"
STATE_DIR="${SCRIPT_DIR}/state"
mkdir -p "${LOG_DIR}" "${STATE_DIR}"

CLUSTER_NAME="${CLUSTER_NAME:-ambient-agentic}"
PROJECT_NS="${PROJECT_NS:-my-project}"

BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-3000}"

BACKEND_DIR="${REPO_ROOT}/components/backend"
FRONTEND_DIR="${REPO_ROOT}/components/frontend"
CRDS_DIR="${REPO_ROOT}/components/manifests/crds"

BACKEND_PID_FILE="${STATE_DIR}/backend.pid"
FRONTEND_PID_FILE="${STATE_DIR}/frontend.pid"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')" # darwin or linux

###############
# Utilities
###############
log() { printf "[%s] %s\n" "$(date '+%H:%M:%S')" "$*"; }
warn() { printf "\033[1;33m%s\033[0m\n" "$*"; }
err() { printf "\033[0;31m%s\033[0m\n" "$*"; }

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    err "Missing required command: $1"
    case "$1" in
      kind)
        warn "Install Kind: https://kind.sigs.k8s.io/docs/user/quick-start/" ;;
      kubectl)
        warn "Install kubectl: https://kubernetes.io/docs/tasks/tools/" ;;
      go)
        warn "Install Go 1.24+: https://go.dev/dl/" ;;
      node|npm)
        warn "Install Node.js 20+: https://nodejs.org/" ;;
    esac
    exit 1
  fi
}

is_pid_alive() {
  local pid_file="$1"
  [[ -f "$pid_file" ]] || return 1
  local pid
  pid="$(cat "$pid_file" 2>/dev/null || echo)"
  [[ -n "$pid" ]] || return 1
  if kill -0 "$pid" >/dev/null 2>&1; then
    return 0
  fi
  return 1
}

port_in_use() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
  else
    # Fallback: try nc if available
    if command -v nc >/dev/null 2>&1; then
      nc -z localhost "$port" >/dev/null 2>&1
    else
      # Best-effort: assume free if we cannot check
      return 1
    fi
  fi
}

wait_http_ok() {
  local url="$1"; local timeout="${2:-60}"; local delay=2
  local start=$(date +%s)
  while true; do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    local now=$(date +%s)
    if (( now - start > timeout )); then
      return 1
    fi
    sleep "$delay"
  done
}

#########################
# Pre-flight requirements
#########################
log "Checking prerequisites..."
need_cmd kind
need_cmd kubectl
need_cmd go
need_cmd node
need_cmd npm

#########################
# Kind cluster and CRDs
#########################
ensure_cluster() {
  if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER_NAME"; then
    log "Creating Kind cluster '$CLUSTER_NAME'..."
    kind create cluster --name "$CLUSTER_NAME"
  else
    log "Reusing existing Kind cluster '$CLUSTER_NAME'"
  fi
}

apply_crds() {
  log "Applying CRDs..."
  kubectl apply -f "${CRDS_DIR}/agenticsessions-crd.yaml"
  kubectl apply -f "${CRDS_DIR}/projectsettings-crd.yaml"
  kubectl apply -f "${CRDS_DIR}/rfeworkflows-crd.yaml"
}

ensure_namespace() {
  if ! kubectl get namespace "$PROJECT_NS" >/dev/null 2>&1; then
    log "Creating namespace ${PROJECT_NS}"
    kubectl create namespace "$PROJECT_NS"
  fi
  # Ensure labels/annotations
  kubectl label namespace "$PROJECT_NS" ambient-code.io/managed=true --overwrite >/dev/null 2>&1 || true
  kubectl annotate namespace "$PROJECT_NS" "ambient-code.io/display-name=${PROJECT_NS}" --overwrite >/dev/null 2>&1 || true
}

#########################
# Backend (Go) service
#########################
start_backend() {
  if is_pid_alive "$BACKEND_PID_FILE"; then
    log "Backend already running (pid $(cat "$BACKEND_PID_FILE"))"
    return 0
  fi

  if port_in_use "$BACKEND_PORT"; then
    err "Port ${BACKEND_PORT} already in use. Stop the process using it or set BACKEND_PORT."
    exit 1
  fi

  log "Starting backend on :${BACKEND_PORT}..."
  (
    set -e
    cd "$BACKEND_DIR"
    export KUBECONFIG="${HOME}/.kube/config"
    export NAMESPACE="$PROJECT_NS"
    # Backend requires valid tokens - create dev service account token for local dev
    # This provides proper auth without requiring external IdP setup
    if ! kubectl get serviceaccount dev-user -n "$PROJECT_NS" >/dev/null 2>&1; then
      kubectl create serviceaccount dev-user -n "$PROJECT_NS"
      kubectl create clusterrolebinding dev-user-admin --clusterrole=cluster-admin --serviceaccount="$PROJECT_NS:dev-user" >/dev/null 2>&1 || true
    fi
    # Ensure Go modules are ready (no-op if already)
    go mod download >/dev/null 2>&1 || true
    # Run backend
    nohup go run . >"${LOG_DIR}/backend.out" 2>"${LOG_DIR}/backend.err" & echo $! >"${BACKEND_PID_FILE}"
  )

  # Wait for health
  if ! wait_http_ok "http://localhost:${BACKEND_PORT}/health" 60; then
    err "Backend failed to become healthy on :${BACKEND_PORT}. Check logs in ${LOG_DIR}."
    exit 1
  fi
  log "Backend is healthy. Logs: ${LOG_DIR}/backend.out"
}

#########################
# Frontend (Next.js)
#########################
start_frontend() {
  if is_pid_alive "$FRONTEND_PID_FILE"; then
    log "Frontend already running (pid $(cat "$FRONTEND_PID_FILE"))"
    return 0
  fi

  if port_in_use "$FRONTEND_PORT"; then
    err "Port ${FRONTEND_PORT} already in use. Stop the process using it or set FRONTEND_PORT."
    exit 1
  fi

  log "Starting frontend on :${FRONTEND_PORT}..."
  (
    set -e
    cd "$FRONTEND_DIR"
    export BACKEND_URL="http://localhost:${BACKEND_PORT}/api"
    # Install deps idempotently
    if [[ ! -d node_modules ]]; then
      npm ci
    fi
    nohup npm run dev >"${LOG_DIR}/frontend.out" 2>"${LOG_DIR}/frontend.err" & echo $! >"${FRONTEND_PID_FILE}"
  )

  # Wait for frontend
  if ! wait_http_ok "http://localhost:${FRONTEND_PORT}" 90; then
    err "Frontend failed to become available on :${FRONTEND_PORT}. Check logs in ${LOG_DIR}."
    exit 1
  fi
  log "Frontend is available. Logs: ${LOG_DIR}/frontend.out"
}

#########################
# Execution
#########################
log "OS detected: ${OS}"
ensure_cluster
apply_crds
ensure_namespace
start_backend
start_frontend

echo ""
log "Local dev ready."
echo "  Backend:  http://localhost:${BACKEND_PORT}/health"
echo "  Frontend: http://localhost:${FRONTEND_PORT}"
echo "  Namespace: ${PROJECT_NS} (labeled for Ambient)"
echo ""
log "To stop: components/scripts/local-dev/stop.sh"


