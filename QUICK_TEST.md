# Quick Start Testing Guide

## Setup (One-time)

### 1. Port-forward Backend (for local testing)
```bash
kubectl port-forward -n ambient-code svc/backend-service 8080:8080
```

### 2. Get Auth Token (if using OAuth proxy)
```bash
# If using OpenShift oauth-proxy, get token from browser dev tools
# Or use oc to get token:
oc whoami -t
```

### 3. Set Environment Variables
```bash
export PROJECT="your-project-name"
export BACKEND_URL="http://localhost:8080"
export TOKEN="your-token-here"  # Optional if using port-forward without auth
```

## Quick Test

### Step 1: Register Workflow
```bash
curl -X POST "${BACKEND_URL}/api/projects/${PROJECT}/workflows" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hello-world",
    "imageDigest": "quay.io/ambient_code/hello-world@sha256:YOUR_DIGEST",
    "graphs": [{"name": "main", "entry": "app:build_app"}]
  }'
```

### Step 2: Create Run
```bash
curl -X POST "${BACKEND_URL}/api/projects/${PROJECT}/agentic-sessions" \
  -H "Content-Type: application/json" \
  -d '{
    "workflowRef": {"name": "hello-world", "graph": "main"},
    "inputs": {"message": "Hello!"}
  }' | jq -r '.name'
```

### Step 3: Check Status
```bash
SESSION_NAME="agentic-session-..."  # From Step 2
watch -n 2 "curl -s ${BACKEND_URL}/api/projects/${PROJECT}/agentic-sessions/${SESSION_NAME} | jq '.status'"
```

## Example Test Workflow

See `test-workflow-example/` directory for a complete example.

