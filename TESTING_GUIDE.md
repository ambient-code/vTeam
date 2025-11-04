# E2E MVP Testing Guide

## Prerequisites

### 1. Deploy Infrastructure

```bash
# Deploy Postgres
kubectl apply -k components/manifests/postgres/

# Wait for Postgres to be ready
kubectl wait --for=condition=ready pod -l app=postgres -n ambient-code --timeout=300s

# Verify Postgres secret exists
kubectl get secret postgres-secret -n ambient-code
```

### 2. Build and Push Base Wrapper Image

```bash
cd components/runners/langgraph-wrapper
podman build --platform linux/amd64 -t quay.io/ambient_code/langgraph-wrapper:base .
podman push quay.io/ambient_code/langgraph-wrapper:base
```

**Note**: Use `--platform linux/amd64` to ensure compatibility with OpenShift/K8s clusters.

### 3. Rebuild Backend

```bash
cd components/backend
go mod tidy
go build
# Deploy updated backend (or restart if already deployed)
```

### 4. Copy Postgres Secret to Project Namespace

For each project namespace where you'll test workflows:

```bash
PROJECT_NAME="your-project-name"
kubectl get secret postgres-secret -n ambient-code -o yaml | \
  sed "s/namespace: ambient-code/namespace: $PROJECT_NAME/" | \
  kubectl apply -f -
```

## Test Workflow

### Step 1: Create Example LangGraph Workflow Image

Create a simple test workflow:

```bash
mkdir test-workflow
cd test-workflow
```

**Or use the example workflow from GitHub:**
```bash
git clone https://github.com/Gkrumbach07/langgraph-example-workflow.git
cd langgraph-example-workflow
```

Create `app/workflow.py`:
```python
from langgraph.graph import StateGraph, END
from typing import TypedDict

class State(TypedDict):
    message: str
    step: int

def build_app():
    graph = StateGraph(State)
    
    def node1(state: State) -> State:
        return {"message": f"Step 1: {state.get('message', '')}", "step": 1}
    
    def node2(state: State) -> State:
        return {"message": f"Step 2: {state['message']}", "step": 2}
    
    graph.add_node("node1", node1)
    graph.add_node("node2", node2)
    graph.set_entry_point("node1")
    graph.add_edge("node1", "node2")
    graph.add_edge("node2", END)
    
    return graph.compile()
```

Create `Dockerfile`:
```dockerfile
FROM quay.io/ambient_code/langgraph-wrapper:base

WORKDIR /app/workflow
COPY app/ ./app/

# Ensure app module is importable
ENV PYTHONPATH=/app/workflow:$PYTHONPATH
```

Build and push:
```bash
# Build for linux/amd64 (required for OpenShift/K8s)
podman build --platform linux/amd64 -t quay.io/ambient_code/test-workflow:v1.0.0 .
podman push quay.io/ambient_code/test-workflow:v1.0.0

# Get digest
podman inspect quay.io/ambient_code/test-workflow:v1.0.0 | jq -r '.[0].RepoDigests[0]'
# Output: quay.io/ambient_code/test-workflow@sha256:...
```

**Note**: Always use `--platform linux/amd64` when building on ARM Macs to ensure compatibility with OpenShift/K8s clusters.

### Step 2: Register Workflow via API

```bash
PROJECT="your-project-name"
BACKEND_URL="http://backend-service.ambient-code.svc.cluster.local:8080"
# Or use port-forward: kubectl port-forward -n ambient-code svc/backend-service 8080:8080
# Then BACKEND_URL="http://localhost:8080"

# Register workflow
curl -X POST "${BACKEND_URL}/api/projects/${PROJECT}/workflows" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "test-workflow",
    "imageDigest": "quay.io/ambient_code/test-workflow@sha256:YOUR_DIGEST_HERE",
    "graphs": [
      {
        "name": "main",
        "entry": "app.workflow:build_app"
      }
    ],
    "inputsSchema": {
      "type": "object",
      "properties": {
        "message": {
          "type": "string",
          "description": "Input message"
        }
      },
      "required": ["message"]
    }
  }'

# Verify workflow registered
curl "${BACKEND_URL}/api/projects/${PROJECT}/workflows/test-workflow" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Step 3: Create Workflow Run

```bash
# Create AgenticSession with workflowRef
curl -X POST "${BACKEND_URL}/api/projects/${PROJECT}/agentic-sessions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "workflowRef": {
      "name": "test-workflow",
      "graph": "main"
    },
    "inputs": {
      "message": "Hello from test!"
    },
    "displayName": "Test LangGraph Workflow Run"
  }'

# Note the session name from response
# Response: {"message":"Agentic session created successfully","name":"agentic-session-1234567890","uid":"..."}
```

### Step 4: Monitor Run Status

```bash
SESSION_NAME="agentic-session-1234567890"  # From previous response

# Check session status
curl "${BACKEND_URL}/api/projects/${PROJECT}/agentic-sessions/${SESSION_NAME}" \
  -H "Authorization: Bearer YOUR_TOKEN" | jq

# Check events
curl "${BACKEND_URL}/api/projects/${PROJECT}/runs/${SESSION_NAME}/events" \
  -H "Authorization: Bearer YOUR_TOKEN" | jq

# Watch pod logs
kubectl logs -n ${PROJECT} -l job-name=${SESSION_NAME}-job -c langgraph-runner -f

# Check pod status
kubectl get pods -n ${PROJECT} -l job-name=${SESSION_NAME}-job

# Check operator logs
kubectl logs -n ambient-code -l app=operator -f
```

### Step 5: Test Approval Flow (if workflow has interrupt)

If your workflow has an interrupt node:

```bash
# Approve interrupted workflow
curl -X POST "${BACKEND_URL}/api/projects/${PROJECT}/runs/${SESSION_NAME}/approve" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "node": "approval_node",
    "decision": {
      "approved": true,
      "notes": "Looks good"
    }
  }'
```

## Verification Checklist

- [ ] Postgres pod is running: `kubectl get pods -n ambient-code -l app=postgres`
- [ ] Backend can connect to Postgres: Check backend logs
- [ ] Workflow tables created: `kubectl exec -n ambient-code -it postgres-0 -- psql -U langgraph -d langgraph -c "\dt"`
- [ ] Workflow registered successfully: Check API response
- [ ] Operator created job: `kubectl get jobs -n ${PROJECT}`
- [ ] Runner pod started: `kubectl get pods -n ${PROJECT} -l app=langgraph-runner`
- [ ] Runner service exists: `kubectl get svc -n ${PROJECT} -l app=langgraph-runner`
- [ ] Runner called /start: Check operator logs for "Successfully started LangGraph workflow"
- [ ] Events are being emitted: Check events API
- [ ] Status updates: Check session status.currentNode
- [ ] Workflow completes: Check session status.phase = "Completed"

## Troubleshooting

### Pod not starting
```bash
# Check pod events
kubectl describe pod -n ${PROJECT} <pod-name>

# Check image pull errors
kubectl get events -n ${PROJECT} --sort-by='.lastTimestamp'
```

### Runner not ready
```bash
# Check runner logs
kubectl logs -n ${PROJECT} <pod-name> -c langgraph-runner

# Check readiness probe
kubectl exec -n ${PROJECT} <pod-name> -c langgraph-runner -- curl http://localhost:8000/ready
```

### Events not reaching backend
```bash
# Check runner logs for event emission errors
kubectl logs -n ${PROJECT} <pod-name> -c langgraph-runner | grep "emit_event"

# Verify backend URL is correct
kubectl exec -n ${PROJECT} <pod-name> -c langgraph-runner -- env | grep BACKEND_API_URL
```

### Postgres connection issues
```bash
# Test connection from pod
kubectl exec -n ${PROJECT} <pod-name> -c langgraph-runner -- \
  python -c "from langgraph.checkpoint.postgres import PostgresSaver; \
  import os; \
  saver = PostgresSaver.from_conn_string(os.getenv('POSTGRES_DB', 'postgresql://...')); \
  saver.setup()"
```

## Quick Test Script

Use the provided `test-langgraph.sh` script:

```bash
# Make executable (if not already)
chmod +x test-langgraph.sh

# Run test
./test-langgraph.sh <project-name> <workflow-image-digest>

# Example:
./test-langgraph.sh myproject quay.io/ambient_code/test-workflow@sha256:abc123...
```

The script will:
1. Register the workflow
2. Create a workflow run
3. Monitor execution status
4. Show events
5. Display final status

