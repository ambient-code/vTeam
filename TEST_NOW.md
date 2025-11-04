# Quick Test Guide for LangGraph MVP

## Prerequisites Check ✅

- ✅ Postgres running
- ✅ Backend running  
- ✅ Frontend running
- ✅ Operator running
- ✅ Workflow image built: `quay.io/gkrumbach07/langgraph-example-workflow:v1.0.0`

## Step 1: Push Workflow Image to Quay.io

```bash
# Login to Quay.io (if not already)
podman login quay.io

# Push the workflow image
podman push quay.io/gkrumbach07/langgraph-example-workflow:v1.0.0

# Get the digest (required for registration)
DIGEST=$(podman inspect quay.io/gkrumbach07/langgraph-example-workflow:v1.0.0 | jq -r '.[0].RepoDigests[0]')
echo "Image digest: $DIGEST"
```

**Make sure the repository is public:**
- Go to: https://quay.io/repository/gkrumbach07/langgraph-example-workflow
- Settings → Visibility → Make Public

## Step 2: Port-Forward Backend

```bash
# Port-forward backend API (run in background)
oc port-forward -n ambient-code svc/backend-service 8080:8080 &

# Or use the route (already exposed)
BACKEND_URL="https://ambient-code.apps.gkrumbac.dev.datahub.redhat.com"
```

## Step 3: Create/Verify Project Namespace

```bash
# Create a test project namespace
PROJECT="test-langgraph"
oc new-project $PROJECT || oc project $PROJECT

# Label it as managed
oc label namespace $PROJECT ambient-code.io/managed=true --overwrite
```

## Step 4: Register Workflow

```bash
# Replace DIGEST with the actual digest from Step 1
DIGEST="quay.io/gkrumbach07/langgraph-example-workflow@sha256:YOUR_DIGEST"

curl -X POST "http://localhost:8080/api/projects/$PROJECT/workflows" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"example-workflow\",
    \"owner\": \"test-user\",
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
          \"type\": \"string\",
          \"description\": \"Input message to process\"
        }
      },
      \"required\": [\"message\"]
    }
  }"
```

**Expected response:**
```json
{"message":"Workflow version registered successfully","id":"..."}
```

## Step 5: Create Workflow Run (AgenticSession)

```bash
curl -X POST "http://localhost:8080/api/projects/$PROJECT/agentic-sessions" \
  -H "Content-Type: application/json" \
  -d "{
    \"workflowRef\": {
      \"name\": \"example-workflow\",
      \"version\": \"v1.0.0\",
      \"graph\": \"main\"
    },
    \"inputs\": {
      \"message\": \"Hello from LangGraph MVP!\",
      \"step\": 0,
      \"result\": \"\",
      \"counter\": 0
    },
    \"displayName\": \"Test LangGraph Workflow\"
  }"
```

**Expected response:**
```json
{"name":"agentic-session-...","status":{"phase":"Pending"}}
```

Save the `name` value - you'll need it for monitoring.

## Step 6: Monitor Workflow Execution

```bash
SESSION_NAME="agentic-session-XXXXX"  # From Step 5

# Watch pod creation
oc get pods -n $PROJECT -w

# Check session status
oc get agenticsessions -n $PROJECT $SESSION_NAME -o yaml

# View logs (once pod is running)
oc logs -n $PROJECT -l job-name=${SESSION_NAME}-job -c langgraph-runner -f

# Check workflow events via API
curl "http://localhost:8080/api/projects/$PROJECT/runs/$SESSION_NAME/events" | jq
```

## Step 7: Verify Completion

```bash
# Check final status
curl "http://localhost:8080/api/projects/$PROJECT/agentic-sessions/$SESSION_NAME" | jq '.status'

# Check Postgres checkpoint data
oc exec postgres-0 -n ambient-code -- psql -U langgraph -d langgraph -c "SELECT thread_id, checkpoint_ns, checkpoint_id FROM checkpoints ORDER BY created_at DESC LIMIT 5;"
```

## Automated Test Script

For convenience, use the provided test script:

```bash
cd /Users/gkrumbac/.cursor/worktrees/vTeam/syBFS

# Make script executable
chmod +x test-langgraph.sh

# Run automated test
./test-langgraph.sh $PROJECT "$DIGEST"
```

## Troubleshooting

**Backend not connecting to Postgres?**
```bash
# Check backend logs
oc logs -n ambient-code -l app=backend-api --tail=50 | grep -i postgres

# Verify Postgres service is accessible
oc exec backend-api-XXX -n ambient-code -- nc -zv postgres-service.ambient-code.svc.cluster.local 5432
```

**Workflow pod not starting?**
```bash
# Check operator logs
oc logs -n ambient-code -l app=agentic-operator --tail=50

# Check pod events
oc describe pod -n $PROJECT <pod-name>
```

**Runner not ready?**
```bash
# Check runner logs
oc logs -n $PROJECT <pod-name> -c langgraph-runner

# Check readiness endpoint
oc exec -n $PROJECT <pod-name> -c langgraph-runner -- curl http://localhost:8000/ready
```

## Next Steps

Once this basic test works:
1. Test with interrupts (HITL)
2. Test workflow resumption
3. Test multiple graphs per image
4. Test workflow versioning

