# How to Test the LangGraph MVP

## Quick Start (3 Steps)

### 1. Setup (One-time)

```bash
# Port-forward backend
kubectl port-forward -n ambient-code svc/backend-service 8080:8080 &

# Copy Postgres secret to your project namespace
PROJECT="your-project"
kubectl get secret postgres-secret -n ambient-code -o yaml | \
  sed "s/namespace: ambient-code/namespace: $PROJECT/" | \
  kubectl apply -f -
```

### 2. Build Test Workflow

```bash
cd test-workflow-example
# Build for linux/amd64 (required for OpenShift/K8s)
podman build --platform linux/amd64 -t quay.io/ambient_code/test-workflow:v1.0.0 .
podman push quay.io/ambient_code/test-workflow:v1.0.0

# Get digest
DIGEST=$(podman inspect quay.io/ambient_code/test-workflow:v1.0.0 | jq -r '.[0].RepoDigests[0]')
echo $DIGEST
```

### 3. Run Test

```bash
# Use the test script
./test-langgraph.sh your-project "$DIGEST"

# Or manually:
# Register workflow → Create run → Monitor status
```

## Manual Testing

### Register Workflow
```bash
curl -X POST "http://localhost:8080/api/projects/YOUR_PROJECT/workflows" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-workflow",
    "imageDigest": "quay.io/ambient_code/test-workflow@sha256:...",
    "graphs": [{"name": "main", "entry": "app.workflow:build_app"}]
  }'
```

### Create Run
```bash
curl -X POST "http://localhost:8080/api/projects/YOUR_PROJECT/agentic-sessions" \
  -H "Content-Type: application/json" \
  -d '{
    "workflowRef": {"name": "test-workflow", "graph": "main"},
    "inputs": {"message": "Hello!"}
  }'
```

### Monitor Status
```bash
SESSION_NAME="agentic-session-..."  # From create response

# Check status
curl "http://localhost:8080/api/projects/YOUR_PROJECT/agentic-sessions/$SESSION_NAME" | jq

# Check events
curl "http://localhost:8080/api/projects/YOUR_PROJECT/runs/$SESSION_NAME/events" | jq

# Watch logs
kubectl logs -n YOUR_PROJECT -l job-name=${SESSION_NAME}-job -c langgraph-runner -f
```

## Troubleshooting

**Pod not starting?**
```bash
kubectl describe pod -n YOUR_PROJECT <pod-name>
kubectl get events -n YOUR_PROJECT --sort-by='.lastTimestamp'
```

**Runner not ready?**
```bash
kubectl logs -n YOUR_PROJECT <pod-name> -c langgraph-runner
kubectl exec -n YOUR_PROJECT <pod-name> -c langgraph-runner -- curl http://localhost:8000/ready
```

**Events not appearing?**
```bash
kubectl logs -n YOUR_PROJECT <pod-name> -c langgraph-runner | grep emit_event
kubectl exec -n YOUR_PROJECT <pod-name> -c langgraph-runner -- env | grep BACKEND_API_URL
```

For detailed testing guide, see `TESTING_GUIDE.md`

