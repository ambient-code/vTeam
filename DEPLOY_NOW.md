# Deploy and Test LangGraph MVP

## Step 1: Update Backend Deployment

```bash
# Update backend to use your new image
oc set image deployment/backend-api backend-api=quay.io/gkrumbach07/vteam_backend:langgraph-mvp -n ambient-code

# Wait for rollout
oc rollout status deployment/backend-api -n ambient-code

# Check backend is running
oc get pods -n ambient-code -l app=backend-api
```

## Step 2: Verify Postgres is Running

```bash
# Check Postgres pod
oc get pods -n ambient-code | grep postgres

# If not running, deploy Postgres
oc apply -k components/manifests/postgres/ -n ambient-code

# Wait for Postgres to be ready
oc wait --for=condition=ready pod -l app=postgres -n ambient-code --timeout=120s
```

## Step 3: Get Workflow Image Digest

```bash
# Get the digest of your workflow image
DIGEST=$(podman inspect quay.io/gkrumbach07/langgraph-example-workflow:v1.0.0 | jq -r '.[0].RepoDigests[0]')
echo "Workflow digest: $DIGEST"
```

## Step 4: Port-Forward Backend (if needed)

```bash
# Port-forward backend API
oc port-forward -n ambient-code svc/backend-service 8080:8080 &
BACKEND_URL="http://localhost:8080"
```

Or use your route:
```bash
BACKEND_URL="https://ambient-code.apps.gkrumbac.dev.datahub.redhat.com"
```

## Step 5: Create Test Project

```bash
PROJECT="test-langgraph"
oc new-project $PROJECT 2>/dev/null || oc project $PROJECT
oc label namespace $PROJECT ambient-code.io/managed=true --overwrite
```

## Step 6: Register Workflow

```bash
# First create the workflow
curl -X POST "$BACKEND_URL/api/projects/$PROJECT/workflows" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "example-workflow",
    "owner": "test-user"
  }'

# Then register the version (replace DIGEST)
curl -X POST "$BACKEND_URL/api/projects/$PROJECT/workflows/example-workflow/versions" \
  -H "Content-Type: application/json" \
  -d "{
    \"version\": \"v1.0.0\",
    \"imageRef\": \"$DIGEST\",
    \"graphs\": [
      {
        \"name\": \"main\",
        \"entry\": \"app.workflow:build_app\"
      }
    ],
    \"inputsSchema\": {
      \"type\": \"object\",
      \"properties\": {
        \"message\": {
          \"type\": \"string\"
        }
      },
      \"required\": [\"message\"]
    }
  }"
```

## Step 7: Create Workflow Run

```bash
curl -X POST "$BACKEND_URL/api/projects/$PROJECT/agentic-sessions" \
  -H "Content-Type: application/json" \
  -d '{
    "workflowRef": {
      "name": "example-workflow",
      "version": "v1.0.0",
      "graph": "main"
    },
    "inputs": {
      "message": "Hello from LangGraph MVP!",
      "step": 0,
      "result": "",
      "counter": 0
    },
    "displayName": "Test LangGraph Workflow"
  }' | jq -r '.name'
```

Save the session name from the response.

## Step 8: Monitor Execution

```bash
SESSION_NAME="agentic-session-XXXXX"  # From Step 7

# Watch pods
oc get pods -n $PROJECT -w

# Check session status
oc get agenticsessions -n $PROJECT $SESSION_NAME -o yaml

# View runner logs
oc logs -n $PROJECT -l job-name=${SESSION_NAME}-job -c langgraph-runner -f

# Check events
curl "$BACKEND_URL/api/projects/$PROJECT/runs/$SESSION_NAME/events" | jq
```

## Quick Verification Checklist

- [ ] Backend pod is running with new image
- [ ] Postgres pod is running
- [ ] Backend can connect to Postgres (check logs)
- [ ] Workflow registered successfully
- [ ] AgenticSession created
- [ ] Workflow pod started
- [ ] Runner is ready (`/ready` endpoint)
- [ ] Workflow executed successfully

## Troubleshooting

**Backend errors:**
```bash
oc logs -n ambient-code -l app=backend-api --tail=100 | grep -i error
```

**Postgres connection:**
```bash
oc exec -n ambient-code deployment/backend-api -- env | grep POSTGRES
```

**Workflow pod issues:**
```bash
oc describe pod -n $PROJECT <pod-name>
oc logs -n $PROJECT <pod-name> -c langgraph-runner
```

