# Build and Push Commands

## Prerequisites

```bash
# Login to Quay.io (do this once)
podman login quay.io
# Enter username: gkrumbach07
# Enter password/token when prompted
```

---

## 1. LangGraph Wrapper Base Image

**Location:** `components/runners/langgraph-wrapper/`

```bash
cd components/runners/langgraph-wrapper

# Build for linux/amd64 (required for OpenShift/K8s)
podman build --platform linux/amd64 -t quay.io/gkrumbach07/langgraph-wrapper:base .

# Push to Quay.io
podman push quay.io/gkrumbach07/langgraph-wrapper:base

# Get digest (save this for reference)
podman inspect quay.io/gkrumbach07/langgraph-wrapper:base | jq -r '.[0].RepoDigests[0]'
```

**Note:** Make sure this image is public in Quay.io settings so workflows can pull it.

---

## 2. Backend Image

**Location:** `components/backend/`

**On your Linux VM:**

```bash
cd components/backend

# Build the backend image
podman build -t quay.io/gkrumbach07/vteam_backend:langgraph-mvp .

# Push to Quay.io
podman push quay.io/gkrumbach07/vteam_backend:langgraph-mvp

# Get digest (save this for reference)
podman inspect quay.io/gkrumbach07/vteam_backend:langgraph-mvp | jq -r '.[0].RepoDigests[0]'
```

**After pushing, update the deployment:**

```bash
# Update deployment to use new image
oc set image deployment/backend-api backend-api=quay.io/gkrumbach07/vteam_backend:langgraph-mvp -n ambient-code

# Or use digest for pinning:
oc set image deployment/backend-api backend-api=quay.io/gkrumbach07/vteam_backend@sha256:YOUR_DIGEST -n ambient-code
```

---

## 3. Example Workflow Image (Optional - for testing)

**Location:** `test-workflow-example/`

```bash
cd test-workflow-example

# Build for linux/amd64
podman build --platform linux/amd64 -t quay.io/gkrumbach07/langgraph-example-workflow:v1.0.0 .

# Push to Quay.io
podman push quay.io/gkrumbach07/langgraph-example-workflow:v1.0.0

# Get digest (required for registration)
podman inspect quay.io/gkrumbach07/langgraph-example-workflow:v1.0.0 | jq -r '.[0].RepoDigests[0]'
```

**After pushing, register in the backend:**

```bash
# Replace YOUR_DIGEST with the digest from above
curl -X POST "http://localhost:8080/api/projects/YOUR_PROJECT/workflows/example-workflow/versions" \
  -H "Content-Type: application/json" \
  -d '{
    "version": "v1.0.0",
    "imageRef": "quay.io/gkrumbach07/langgraph-example-workflow@sha256:YOUR_DIGEST",
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

---

## Quick Reference

### Image Tags:
- Base wrapper: `quay.io/gkrumbach07/langgraph-wrapper:base`
- Backend: `quay.io/gkrumbach07/vteam_backend:langgraph-mvp`
- Example workflow: `quay.io/gkrumbach07/langgraph-example-workflow:v1.0.0`

### Registry Paths:
- Your personal org: `quay.io/gkrumbach07/*`
- Main org: `quay.io/ambient_code/*`

### Build Order:
1. **First:** Build and push `langgraph-wrapper:base` (workflows depend on it)
2. **Second:** Build and push `vteam_backend:langgraph-mvp` (system component)
3. **Third:** Build and push example workflow (for testing)

