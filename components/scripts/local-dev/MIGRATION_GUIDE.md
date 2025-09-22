# Migration Guide: Kind to OpenShift Local (CRC)

This guide helps you migrate from the old Kind-based local development environment to the new OpenShift Local (CRC) setup.

## Why the Migration?

### Problems with Kind-Based Setup
- ❌ Backend hardcoded for OpenShift, crashes on Kind
- ❌ Uses vanilla K8s namespaces, not OpenShift Projects
- ❌ No OpenShift OAuth/RBAC testing
- ❌ Port-forwarding instead of OpenShift Routes
- ❌ Service account tokens don't match production behavior

### Benefits of CRC-Based Setup
- ✅ Production parity with real OpenShift
- ✅ Native OpenShift Projects and RBAC
- ✅ Real OpenShift OAuth integration
- ✅ OpenShift Routes for external access
- ✅ Proper token-based authentication
- ✅ All backend APIs work without crashes

## Before You Migrate

### Backup Current Work
```bash
# Stop current Kind environment
make dev-stop

# Export any important data from Kind cluster (if needed)
kubectl get all --all-namespaces -o yaml > kind-backup.yaml
```

### System Requirements Check
- **CPU:** 4+ cores (CRC needs more resources than Kind )
- **RAM:** 8+ GB available for CRC
- **Disk:** 50+ GB free space
- **Network:** No VPN conflicts with `192.168.130.0/24`

## Migration Steps

### 1. Clean Up Kind Environment
```bash
# Stop old environment
make dev-stop

# Optional: Remove Kind cluster completely
kind delete cluster --name ambient-agentic
```

### 2. Install Prerequisites

**Install CRC:**
```bash
# macOS
brew install crc

# Linux - download from:
# https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest/
```

**Get Red Hat Pull Secret:**
1. Visit: https://console.redhat.com/openshift/create/local
2. Create free Red Hat account if needed
3. Download pull secret
4. Save to `~/.crc/pull-secret.json`

### 3. Initial CRC Setup
```bash
# Run CRC setup (one-time)
crc setup

# Configure pull secret
crc config set pull-secret-file ~/.crc/pull-secret.json

# Optional: Configure resources
crc config set cpus 4
crc config set memory 8192
```

### 4. Start New Environment
```bash
# Use same Makefile commands!
make dev-start
```

**First run takes 5-10 minutes** (downloads OpenShift images)

### 5. Verify Migration
```bash
make dev-test
```

Should show all tests passing, including API tests that failed with Kind.

## Command Mapping

The Makefile interface remains the same:

| Old Command | New Command | Change |
|-------------|-------------|---------|
| `make dev-start` | `make dev-start` | ✅ Same (now uses CRC) |
| `make dev-stop` | `make dev-stop` | ✅ Same (keeps CRC running) |
| `make dev-test` | `make dev-test` | ✅ Same (more comprehensive tests) |
| N/A | `make dev-stop-cluster` | 🆕 Stop CRC cluster too |
| N/A | `make dev-clean` | 🆕 Delete OpenShift project |

## Access Changes

### Old URLs (Kind + Port Forwarding) - DEPRECATED
```
Backend:  http://localhost:8080/health     # ❌ No longer supported
Frontend: http://localhost:3000            # ❌ No longer supported
```

### New URLs (CRC + OpenShift Routes)
```
Backend:  https://vteam-backend-vteam-dev.apps-crc.testing/health
Frontend: https://vteam-frontend-vteam-dev.apps-crc.testing
Console:  https://console-openshift-console.apps-crc.testing
```

## CLI Changes

### Old (kubectl with Kind)
```bash
kubectl get pods -n my-project
kubectl logs deployment/backend -n my-project
```

### New (oc with OpenShift)
```bash
oc get pods -n vteam-dev
oc logs deployment/vteam-backend -n vteam-dev

# Or switch project context
oc project vteam-dev
oc get pods
```

## Troubleshooting Migration

### CRC Fails to Start
```bash
# Check system resources
crc config get cpus memory

# Reduce if needed
crc config set cpus 2
crc config set memory 6144

# Restart
crc stop && crc start
```

### Pull Secret Issues
```bash
# Re-download from https://console.redhat.com/openshift/create/local
# Save to ~/.crc/pull-secret.json
crc setup
```

### Port Conflicts
CRC uses different access patterns than Kind:
- `6443` - OpenShift API (vs Kind's random port)
- `443/80` - OpenShift Routes with TLS (vs Kind's port-forwarding)
- **Direct HTTPS access** via Routes (no port-forwarding needed)

### Memory Issues
```bash
# Monitor CRC resource usage
crc status

# Reduce allocation
crc stop
crc config set memory 6144
crc start
```

### DNS Issues
Ensure `.apps-crc.testing` resolves to `127.0.0.1`:
```bash
# Check DNS resolution
nslookup api.crc.testing
# Should return 127.0.0.1

# Fix if needed - add to /etc/hosts:
sudo bash -c 'echo "127.0.0.1 api.crc.testing" >> /etc/hosts'
sudo bash -c 'echo "127.0.0.1 oauth-openshift.apps-crc.testing" >> /etc/hosts'
sudo bash -c 'echo "127.0.0.1 console-openshift-console.apps-crc.testing" >> /etc/hosts'
```

### VPN Conflicts
Disable VPN during CRC setup if you get networking errors.

## Rollback Plan

If you need to rollback to Kind temporarily:

### 1. Stop CRC Environment
```bash
make dev-stop-cluster
```

### 2. Use Old Scripts Directly
```bash
# The old scripts have been removed - CRC is now the only supported approach
# If you need to rollback, you can restore from git history:
# git show HEAD~10:components/scripts/local-dev/start.sh > start-backup.sh
```

### 3. Alternative: Historical Kind Approach
```bash
# The Kind-based approach has been deprecated and removed
# If absolutely needed, restore from git history:
git log --oneline --all | grep -i kind
git show <commit-hash>:components/scripts/local-dev/start.sh > legacy-start.sh
```

## FAQ

**Q: Do I need to change my code?**
A: No, your application code remains unchanged.

**Q: Will my container images work?**
A: Yes, CRC uses the same container runtime.

**Q: Can I run both Kind and CRC?**
A: Yes, but not simultaneously due to resource usage.

**Q: Is CRC free?**
A: Yes, CRC and OpenShift Local are free for development use.

**Q: What about CI/CD?**
A: CI/CD should use the production OpenShift deployment method, not local dev.

**Q: How much slower is CRC vs Kind?**
A: Initial startup is slower (5-10 min vs 1-2 min), but runtime performance is similar. **CRC provides production parity** that Kind cannot match.

## Getting Help

### Check Status
```bash
crc status                    # CRC cluster status
make dev-test                 # Full environment test
oc get pods -n vteam-dev      # OpenShift resources
```

### View Logs
```bash
oc logs deployment/vteam-backend -n vteam-dev
oc logs deployment/vteam-frontend -n vteam-dev
```

### Reset Everything
```bash
make dev-clean                # Delete project
crc stop && crc delete        # Delete CRC VM
crc setup && make dev-start   # Fresh start
```

### Documentation
- [CRC Documentation](https://crc.dev/crc/)
- [OpenShift CLI Reference](https://docs.openshift.com/container-platform/latest/cli_reference/openshift_cli/developer-cli-commands.html)
- [vTeam Local Dev README](README.md)
