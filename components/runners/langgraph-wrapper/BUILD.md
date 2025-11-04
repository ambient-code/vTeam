# Build and Push Base LangGraph Wrapper Image

## Build the Base Image

```bash
cd components/runners/langgraph-wrapper

# Build for linux/amd64 (required for OpenShift/K8s)
podman build --platform linux/amd64 -t quay.io/gkrumbach07/langgraph-wrapper:base .
```

## Push to Quay.io

```bash
# Login to Quay.io (if not already logged in)
podman login quay.io
# Enter your Quay.io username: gkrumbach07
# Enter your password/token when prompted

# Push the image
podman push quay.io/gkrumbach07/langgraph-wrapper:base

# Get the digest (for reference)
podman inspect quay.io/gkrumbach07/langgraph-wrapper:base | jq -r '.[0].RepoDigests[0]'
```

## Verify Image

```bash
# Check image exists locally
podman images | grep langgraph-wrapper

# Pull and test (after pushing)
podman pull quay.io/gkrumbach07/langgraph-wrapper:base
```

## Make Image Public (if needed)

1. Go to https://quay.io/repository/gkrumbach07/langgraph-wrapper
2. Click on "Settings" → "Visibility"
3. Make it public so workflows can pull it

## Update Workflow Dockerfiles

All workflow Dockerfiles should now use:
```dockerfile
FROM quay.io/gkrumbach07/langgraph-wrapper:base
```

The Dockerfile.template has been updated with this base image.

