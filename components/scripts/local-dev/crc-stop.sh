#!/bin/bash

set -euo pipefail

# CRC-based local dev cleanup:
# - Removes vTeam deployments from OpenShift project
# - Optionally stops CRC cluster (keeps it running by default for faster restarts)
# - Cleans up local state files

###############
# Configuration
###############
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="${SCRIPT_DIR}/state"

# Project Configuration
PROJECT_NAME="${PROJECT_NAME:-vteam-dev}"

# Command line options
STOP_CLUSTER="${STOP_CLUSTER:-false}"
DELETE_PROJECT="${DELETE_PROJECT:-false}"

###############
# Utilities
###############
log() { printf "[%s] %s\n" "$(date '+%H:%M:%S')" "$*"; }
warn() { printf "\033[1;33m%s\033[0m\n" "$*"; }
err() { printf "\033[0;31m%s\033[0m\n" "$*"; }
success() { printf "\033[0;32m%s\033[0m\n" "$*"; }

usage() {
  echo "Usage: $0 [OPTIONS]"
  echo ""
  echo "Options:"
  echo "  --stop-cluster    Stop the CRC cluster (default: keep running)"
  echo "  --delete-project  Delete the entire OpenShift project (default: keep project)"
  echo "  -h, --help        Show this help message"
  echo ""
  echo "Examples:"
  echo "  $0                    # Remove deployments but keep CRC running"
  echo "  $0 --stop-cluster     # Remove deployments and stop CRC cluster"
  echo "  $0 --delete-project   # Remove entire project but keep CRC running"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case $1 in
      --stop-cluster)
        STOP_CLUSTER=true
        shift
        ;;
      --delete-project)
        DELETE_PROJECT=true
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        err "Unknown option: $1"
        usage
        exit 1
        ;;
    esac
  done
}

check_oc_available() {
  if ! command -v oc >/dev/null 2>&1; then
    warn "OpenShift CLI (oc) not available. CRC might not be running or configured."
    return 1
  fi
  
  if ! oc whoami >/dev/null 2>&1; then
    warn "Not logged into OpenShift. CRC might not be running or you're not authenticated."
    return 1
  fi
  
  return 0
}

#########################
# Cleanup functions
#########################
cleanup_deployments() {
  if ! check_oc_available; then
    log "Skipping deployment cleanup - OpenShift not accessible"
    return 0
  fi
  
  if ! oc get project "$PROJECT_NAME" >/dev/null 2>&1; then
    log "Project '$PROJECT_NAME' not found, skipping deployment cleanup"
    return 0
  fi
  
  log "Cleaning up vTeam deployments from project '$PROJECT_NAME'..."
  
  # Switch to the project
  oc project "$PROJECT_NAME" >/dev/null 2>&1 || true
  
  # Delete vTeam resources
  log "Removing routes..."
  oc delete route vteam-backend vteam-frontend --ignore-not-found=true
  
  log "Removing services..."
  oc delete service vteam-backend vteam-frontend --ignore-not-found=true
  
  log "Removing deployments..."
  oc delete deployment vteam-backend vteam-frontend --ignore-not-found=true
  
  log "Removing imagestreams..."
  oc delete imagestream vteam-backend vteam-frontend --ignore-not-found=true
  
  # Clean up service accounts (but keep them for faster restart)
  log "Service accounts preserved for faster restart"
  
  success "Deployments cleaned up from project '$PROJECT_NAME'"
}

delete_project() {
  if ! check_oc_available; then
    log "Skipping project deletion - OpenShift not accessible"
    return 0
  fi
  
  if ! oc get project "$PROJECT_NAME" >/dev/null 2>&1; then
    log "Project '$PROJECT_NAME' not found, nothing to delete"
    return 0
  fi
  
  log "Deleting OpenShift project '$PROJECT_NAME'..."
  oc delete project "$PROJECT_NAME"
  
  # Wait for project to be fully deleted
  local timeout=60
  local delay=2
  local start=$(date +%s)
  
  while oc get project "$PROJECT_NAME" >/dev/null 2>&1; do
    local now=$(date +%s)
    if (( now - start > timeout )); then
      warn "Timeout waiting for project deletion"
      break
    fi
    log "Waiting for project deletion..."
    sleep "$delay"
  done
  
  success "Project '$PROJECT_NAME' deleted"
}

stop_crc_cluster() {
  if ! command -v crc >/dev/null 2>&1; then
    warn "CRC not available, skipping cluster stop"
    return 0
  fi
  
  local crc_status
  crc_status=$(crc status -o json 2>/dev/null | jq -r '.crcStatus // "Stopped"' 2>/dev/null || echo "Unknown")
  
  case "$crc_status" in
    "Running")
      log "Stopping CRC cluster..."
      crc stop
      success "CRC cluster stopped"
      ;;
    "Stopped")
      log "CRC cluster is already stopped"
      ;;
    *)
      log "CRC cluster status: $crc_status"
      ;;
  esac
}

cleanup_state() {
  log "Cleaning up local state files..."
  rm -f "${STATE_DIR}/urls.env"
  success "Local state cleaned up"
}

#########################
# Execution
#########################
parse_args "$@"

echo "Stopping vTeam local development environment..."

if [[ "$DELETE_PROJECT" == "true" ]]; then
  delete_project
else
  cleanup_deployments
fi

if [[ "$STOP_CLUSTER" == "true" ]]; then
  stop_crc_cluster
else
  log "CRC cluster kept running for faster restarts (use --stop-cluster to stop it)"
fi

cleanup_state

echo ""
success "Local development environment stopped"

if [[ "$STOP_CLUSTER" == "false" ]]; then
  echo ""
  log "CRC cluster is still running. To fully stop:"
  echo "  $0 --stop-cluster"
  echo ""
  log "To restart development:"
  echo "  make dev-start"
fi
