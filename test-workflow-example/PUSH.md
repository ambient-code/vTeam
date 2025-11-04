# Push Workflow Image to Quay.io

## Image Built Successfully ✅

**Base Image**: `quay.io/gkrumbach07/langgraph-wrapper:base`  
**Workflow Image**: `quay.io/gkrumbach07/langgraph-example-workflow:v1.0.0`  
**Platform**: linux/amd64 (for OpenShift/K8s)

## Push to Quay.io

```bash
# Make sure you're logged in
podman login quay.io

# Push the workflow image
podman push quay.io/gkrumbach07/langgraph-example-workflow:v1.0.0

# Get the digest (required for registration)
podman inspect quay.io/gkrumbach07/langgraph-example-workflow:v1.0.0 | jq -r '.[0].RepoDigests[0]'
```

## After Pushing

The digest will look like:
```
quay.io/gkrumbach07/langgraph-example-workflow@sha256:1a3fffb16fcec808995907361828439aeb63ae28b6e0474c46e0f04ae7cb14c6
```

Use this digest to register the workflow in vTeam.

## Make Repository Public (if needed)

1. Go to: https://quay.io/repository/gkrumbach07/langgraph-example-workflow
2. Create the repository if it doesn't exist
3. Settings → Visibility → Make Public

