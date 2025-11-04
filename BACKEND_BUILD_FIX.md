# Fix Backend Exec Format Error

## Problem
`exec container process '/app/./main': Exec format error` - This means the binary architecture doesn't match the cluster.

## Solution: Rebuild with explicit platform

### Option 1: Build with --platform flag (Recommended)

```bash
cd components/backend

# Build explicitly for linux/amd64
podman build --platform linux/amd64 -t quay.io/gkrumbach07/vteam_backend:langgraph-mvp .

# Push
podman push quay.io/gkrumbach07/vteam_backend:langgraph-mvp
```

### Option 2: Update Dockerfile (Already done)

The Dockerfile has been updated to explicitly build for `GOARCH=amd64`. Just rebuild:

```bash
cd components/backend
podman build --platform linux/amd64 -t quay.io/gkrumbach07/vteam_backend:langgraph-mvp .
podman push quay.io/gkrumbach07/vteam_backend:langgraph-mvp
```

### Option 3: Check your VM architecture

```bash
# Check what architecture your VM is
uname -m

# If it's aarch64 (ARM64), you MUST use --platform linux/amd64
# If it's x86_64 (AMD64), the build should work, but still use --platform to be safe
```

## After Rebuilding

```bash
# Update deployment
oc set image deployment/backend-api backend-api=quay.io/gkrumbach07/vteam_backend:langgraph-mvp -n ambient-code

# Wait for rollout
oc rollout status deployment/backend-api -n ambient-code

# Check logs
oc logs -n ambient-code -l app=backend-api --tail=50
```

## Verify the Binary Architecture

You can check the binary architecture inside the image:

```bash
# Create a test container
podman run --rm quay.io/gkrumbach07/vteam_backend:langgraph-mvp file /app/main

# Should show: ELF 64-bit LSB executable, x86-64
# If it shows ARM or other architecture, rebuild with --platform linux/amd64
```


