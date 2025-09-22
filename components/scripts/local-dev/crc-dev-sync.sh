#!/bin/bash

set -euo pipefail

# CRC Development Sync Script
# Continuously syncs local source code to CRC pods for hot-reloading

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BACKEND_DIR="${REPO_ROOT}/components/backend"
FRONTEND_DIR="${REPO_ROOT}/components/frontend"

PROJECT_NAME="${PROJECT_NAME:-vteam-dev}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[$(date '+%H:%M:%S')]${NC} $*"; }
warn() { echo -e "${YELLOW}[$(date '+%H:%M:%S')]${NC} $*"; }
err() { echo -e "${RED}[$(date '+%H:%M:%S')]${NC} $*"; }

usage() {
  echo "Usage: $0 [backend|frontend|both]"
  echo ""
  echo "Continuously sync source code to CRC pods for hot-reloading"
  echo ""
  echo "Options:"
  echo "  backend   - Sync only backend code"
  echo "  frontend  - Sync only frontend code"
  echo "  both      - Sync both (default)"
  exit 1
}

sync_backend() {
  log "Starting backend sync..."
  
  # Get backend pod name
  local pod_name
  pod_name=$(oc get pod -l app=vteam-backend -o jsonpath='{.items[0].metadata.name}' -n "$PROJECT_NAME" 2>/dev/null)
  
  if [[ -z "$pod_name" ]]; then
    err "Backend pod not found. Is the backend deployment running?"
    return 1
  fi
  
  log "Syncing to backend pod: $pod_name"
  
  # Initial full sync
  oc rsync "$BACKEND_DIR/" "$pod_name:/app/" \
    --exclude=tmp \
    --exclude=.git \
    --exclude=.air.toml \
    --exclude=go.sum \
    -n "$PROJECT_NAME"
  
  # Watch for changes and sync
  log "Watching backend directory for changes..."
  fswatch -o "$BACKEND_DIR" | while read -r _; do
    log "Detected backend changes, syncing..."
    oc rsync "$BACKEND_DIR/" "$pod_name:/app/" \
      --exclude=tmp \
      --exclude=.git \
      --exclude=.air.toml \
      --exclude=go.sum \
      -n "$PROJECT_NAME" || warn "Sync failed, will retry on next change"
  done
}

sync_frontend() {
  log "Starting frontend sync..."
  
  # Get frontend pod name
  local pod_name
  pod_name=$(oc get pod -l app=vteam-frontend -o jsonpath='{.items[0].metadata.name}' -n "$PROJECT_NAME" 2>/dev/null)
  
  if [[ -z "$pod_name" ]]; then
    err "Frontend pod not found. Is the frontend deployment running?"
    return 1
  fi
  
  log "Syncing to frontend pod: $pod_name"
  
  # Initial full sync (excluding node_modules and build artifacts)
  oc rsync "$FRONTEND_DIR/" "$pod_name:/app/" \
    --exclude=node_modules \
    --exclude=.next \
    --exclude=.git \
    --exclude=out \
    --exclude=build \
    -n "$PROJECT_NAME"
  
  # Watch for changes and sync
  log "Watching frontend directory for changes..."
  fswatch -o "$FRONTEND_DIR" \
    --exclude node_modules \
    --exclude .next \
    --exclude .git | while read -r _; do
    log "Detected frontend changes, syncing..."
    oc rsync "$FRONTEND_DIR/" "$pod_name:/app/" \
      --exclude=node_modules \
      --exclude=.next \
      --exclude=.git \
      --exclude=out \
      --exclude=build \
      -n "$PROJECT_NAME" || warn "Sync failed, will retry on next change"
  done
}

check_dependencies() {
  if ! command -v fswatch >/dev/null 2>&1; then
    err "fswatch is required but not installed"
    echo "Install with:"
    echo "  macOS: brew install fswatch"
    echo "  Linux: apt-get install fswatch or yum install fswatch"
    exit 1
  fi
  
  if ! command -v oc >/dev/null 2>&1; then
    err "oc (OpenShift CLI) is required but not installed"
    exit 1
  fi
  
  # Check if logged in
  if ! oc whoami >/dev/null 2>&1; then
    err "Not logged into OpenShift. Run 'oc login' first"
    exit 1
  fi
  
  # Check project exists
  if ! oc get project "$PROJECT_NAME" >/dev/null 2>&1; then
    err "Project '$PROJECT_NAME' not found"
    exit 1
  fi
}

main() {
  local target="${1:-both}"
  
  check_dependencies
  
  log "OpenShift project: $PROJECT_NAME"
  
  case "$target" in
    backend)
      sync_backend
      ;;
    frontend)
      sync_frontend
      ;;
    both)
      # Run both in parallel
      sync_backend &
      BACKEND_PID=$!
      sync_frontend &
      FRONTEND_PID=$!
      
      # Wait for both or handle interrupts
      trap 'kill $BACKEND_PID $FRONTEND_PID 2>/dev/null' EXIT
      wait $BACKEND_PID $FRONTEND_PID
      ;;
    *)
      usage
      ;;
  esac
}

main "$@"
