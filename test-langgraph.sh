#!/bin/bash
# Quick E2E Test Script for LangGraph Workflows
# Usage: ./test-langgraph.sh <project-name> <workflow-image-digest>

set -e

PROJECT="${1:-test-project}"
IMAGE_DIGEST="${2}"
BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"

if [ -z "$IMAGE_DIGEST" ]; then
  echo "Usage: $0 <project-name> <workflow-image-digest>"
  echo "Example: $0 myproject quay.io/ambient_code/test-workflow@sha256:abc123..."
  exit 1
fi

echo "🧪 Testing LangGraph Workflow MVP"
echo "=================================="
echo "Project: $PROJECT"
echo "Image: $IMAGE_DIGEST"
echo "Backend: $BACKEND_URL"
echo ""

# Step 1: Register workflow
echo "1️⃣  Registering workflow..."
WORKFLOW_RESPONSE=$(curl -s -X POST "${BACKEND_URL}/api/projects/${PROJECT}/workflows" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"test-workflow\",
    \"imageDigest\": \"${IMAGE_DIGEST}\",
    \"graphs\": [{\"name\": \"main\", \"entry\": \"app.workflow:build_app\"}]
  }")

if echo "$WORKFLOW_RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
  echo "❌ Failed to register workflow:"
  echo "$WORKFLOW_RESPONSE" | jq
  exit 1
fi

echo "✅ Workflow registered"
echo "$WORKFLOW_RESPONSE" | jq

# Step 2: Create run
echo ""
echo "2️⃣  Creating workflow run..."
SESSION_RESPONSE=$(curl -s -X POST "${BACKEND_URL}/api/projects/${PROJECT}/agentic-sessions" \
  -H "Content-Type: application/json" \
  -d '{
    "workflowRef": {"name": "test-workflow", "graph": "main"},
    "inputs": {"message": "Hello from test script!"},
    "displayName": "Test Run"
  }')

SESSION_NAME=$(echo "$SESSION_RESPONSE" | jq -r '.name // empty')

if [ -z "$SESSION_NAME" ] || [ "$SESSION_NAME" = "null" ]; then
  echo "❌ Failed to create session:"
  echo "$SESSION_RESPONSE" | jq
  exit 1
fi

echo "✅ Session created: $SESSION_NAME"
echo "$SESSION_RESPONSE" | jq

# Step 3: Monitor status
echo ""
echo "3️⃣  Monitoring workflow execution..."
echo "   (Press Ctrl+C to stop monitoring)"

for i in {1..60}; do
  STATUS_RESPONSE=$(curl -s "${BACKEND_URL}/api/projects/${PROJECT}/agentic-sessions/${SESSION_NAME}")
  PHASE=$(echo "$STATUS_RESPONSE" | jq -r '.status.phase // "Unknown"')
  CURRENT_NODE=$(echo "$STATUS_RESPONSE" | jq -r '.status.currentNode // "N/A"')
  MESSAGE=$(echo "$STATUS_RESPONSE" | jq -r '.status.message // ""')
  
  printf "\r   [%02d] Phase: %-12s | Node: %-20s" "$i" "$PHASE" "$CURRENT_NODE"
  
  if [ "$PHASE" = "Completed" ]; then
    echo ""
    echo "✅ Workflow completed successfully!"
    break
  elif [ "$PHASE" = "Failed" ] || [ "$PHASE" = "Error" ]; then
    echo ""
    echo "❌ Workflow failed: $MESSAGE"
    exit 1
  fi
  
  sleep 2
done

if [ "$PHASE" != "Completed" ] && [ "$PHASE" != "Failed" ] && [ "$PHASE" != "Error" ]; then
  echo ""
  echo "⏱️  Workflow still running after 2 minutes"
fi

# Step 4: Check events
echo ""
echo "4️⃣  Checking workflow events..."
EVENTS=$(curl -s "${BACKEND_URL}/api/projects/${PROJECT}/runs/${SESSION_NAME}/events")
EVENT_COUNT=$(echo "$EVENTS" | jq '.events | length')
echo "   Found $EVENT_COUNT events"
echo "$EVENTS" | jq '.events[-3:]'  # Show last 3 events

# Step 5: Final status
echo ""
echo "5️⃣  Final session status:"
curl -s "${BACKEND_URL}/api/projects/${PROJECT}/agentic-sessions/${SESSION_NAME}" | jq '.status'

echo ""
echo "🎉 Test complete!"
echo "   Session: $SESSION_NAME"
echo "   View logs: kubectl logs -n $PROJECT -l job-name=${SESSION_NAME}-job -c langgraph-runner"

