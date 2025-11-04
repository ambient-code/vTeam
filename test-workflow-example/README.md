# Example LangGraph Workflow

A simple example workflow demonstrating LangGraph integration with the vTeam workflow system.

## Workflow Structure

This workflow has 3 sequential nodes:
1. **step_one**: Processes the input message
2. **step_two**: Further processes the message
3. **step_three**: Produces final output

## Building the Image

**Important**: Build for Linux AMD64 architecture (required for OpenShift/K8s clusters):

```bash
# Build for linux/amd64 platform (required for OpenShift/K8s)
podman build --platform linux/amd64 -t quay.io/ambient_code/langgraph-example-workflow:v1.0.0 .

# Push to registry
podman push quay.io/ambient_code/langgraph-example-workflow:v1.0.0

# Get digest (required for registration)
podman inspect quay.io/ambient_code/langgraph-example-workflow:v1.0.0 | jq -r '.[0].RepoDigests[0]'
```

**Note**: If you're on an ARM Mac (M1/M2/M3), you must use `--platform linux/amd64` to build for OpenShift/K8s clusters which typically run on AMD64.

## Registering in vTeam

```bash
curl -X POST "http://localhost:8080/api/projects/YOUR_PROJECT/workflows" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "example-workflow",
    "imageDigest": "quay.io/ambient_code/langgraph-example-workflow@sha256:YOUR_DIGEST",
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
          "description": "Input message to process"
        }
      },
      "required": ["message"]
    }
  }'
```

## Running the Workflow

```bash
curl -X POST "http://localhost:8080/api/projects/YOUR_PROJECT/agentic-sessions" \
  -H "Content-Type: application/json" \
  -d '{
    "workflowRef": {
      "name": "example-workflow",
      "graph": "main"
    },
    "inputs": {
      "message": "Hello from LangGraph!"
    },
    "displayName": "Example Workflow Run"
  }'
```

## Local Testing

```bash
# Install dependencies
pip install langgraph

# Run locally
python app/workflow.py
```

## Input/Output

**Input:**
```json
{
  "message": "Hello World"
}
```

**Output State:**
```json
{
  "message": "Step 1 processed: Hello World",
  "step": 3,
  "result": "[Step 1] Hello World\n[Step 2] Processed message\n[Step 3] Final result: Step 1 processed: Hello World\n",
  "counter": 3
}
```
