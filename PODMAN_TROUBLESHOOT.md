# Podman User Namespace Troubleshooting

## Current Error:
```
potentially insufficient UIDs or GIDs available in user namespace
lchown /usr/bin/write: invalid argument
```

## Step-by-Step Fix:

### 1. Check current subuid/subgid configuration
```bash
cat /etc/subuid
cat /etc/subgid
```

### 2. If empty or missing your user, add it:
```bash
# Add subuid range for your user
sudo usermod --add-subuids 100000-165535 gkrumbac

# Add subgid range for your user  
sudo usermod --add-subgids 100000-165535 gkrumbac

# Verify it was added
cat /etc/subuid | grep gkrumbac
cat /etc/subgid | grep gkrumbac
```

### 3. Restart podman system to pick up changes
```bash
podman system migrate
# Or restart podman service if running as root
sudo systemctl restart podman.socket
```

### 4. Log out and back in (or restart terminal session)

### 5. Try building again
```bash
podman build --platform linux/amd64 -t quay.io/gkrumbach07/langgraph-wrapper:base .
```

## Alternative: Use Docker (if available)
```bash
# Check if docker is installed
which docker

# If available, use docker instead
docker build --platform linux/amd64 -t quay.io/gkrumbach07/langgraph-wrapper:base .
docker push quay.io/gkrumbach07/langgraph-wrapper:base
```

## Alternative: Build as root (quick workaround)
```bash
sudo podman build --platform linux/amd64 -t quay.io/gkrumbach07/langgraph-wrapper:base .
sudo podman push quay.io/gkrumbach07/langgraph-wrapper:base
```

## Alternative: Use a different base image (if Red Hat registry requires auth)
Try using a public Python base image:

```dockerfile
# In Dockerfile, change FROM line to:
FROM python:3.11-slim

# Or use UBI from Docker Hub:
FROM registry.hub.docker.com/library/python:3.11-slim
```

