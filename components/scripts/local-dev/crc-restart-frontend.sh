#!/bin/bash
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


build_and_deploy() {
    oc start-build vteam-frontend --from-dir="$FRONTEND_DIR" --wait -n "$PROJECT_NAME"
    oc apply -f "${MANIFESTS_DIR}/frontend-deployment.yaml" -n "$PROJECT_NAME"
}

build_and_deploy
