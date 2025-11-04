# Fixing Podman User Namespace Error

## Error:
```
potentially insufficient UIDs or GIDs available in user namespace
Check /etc/subuid and /etc/subgid if configured locally and run "podman system migrate"
```

## Solutions (try in order):

### Solution 1: Run podman system migrate
```bash
podman system migrate
```

### Solution 2: Check subuid/subgid mappings
```bash
# Check current mappings
cat /etc/subuid
cat /etc/subgid

# If empty or insufficient, add your user (requires sudo):
sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 $USER

# Log out and back in for changes to take effect
```

### Solution 3: Use --userns=keep-id flag
```bash
podman build --userns=keep-id --platform linux/amd64 -t quay.io/gkrumbach07/langgraph-wrapper:base .
```

### Solution 4: Use rootless with proper groups
```bash
# Check if user is in podman group
groups | grep podman

# If not, add user to podman group (requires sudo):
sudo usermod -aG podman $USER
# Log out and back in
```

### Solution 5: Build as root (not recommended, but works)
```bash
sudo podman build --platform linux/amd64 -t quay.io/gkrumbach07/langgraph-wrapper:base .
sudo podman push quay.io/gkrumbach07/langgraph-wrapper:base
```

### Solution 6: Use Docker instead (if available)
```bash
docker build --platform linux/amd64 -t quay.io/gkrumbach07/langgraph-wrapper:base .
docker push quay.io/gkrumbach07/langgraph-wrapper:base
```

## Quick Fix (Most Common):
```bash
# 1. Run migrate
podman system migrate

# 2. If that doesn't work, configure subuid/subgid
sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 $USER

# 3. Log out and back in, then try again
podman build --platform linux/amd64 -t quay.io/gkrumbach07/langgraph-wrapper:base .
```

