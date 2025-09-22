# vTeam Local Development

> **🎉 STATUS: FULLY WORKING** - Project creation, authentication, hot-reloading all functional!

## Quick Start

### 1. Install Prerequisites
```bash
# macOS
brew install crc

# Get Red Hat pull secret (free account):
# 1. Visit: https://console.redhat.com/openshift/create/local
# 2. Download to ~/.crc/pull-secret.json
# 3. Run: crc setup
```

### 2. Start Development Environment
```bash
make dev-start
```
*First run: ~5-10 minutes. Subsequent runs: ~2-3 minutes.*

### 3. Access Your Environment
- **Frontend**: https://vteam-frontend-vteam-dev.apps-crc.testing
- **Backend**: https://vteam-backend-vteam-dev.apps-crc.testing/health  
- **Console**: https://console-openshift-console.apps-crc.testing

### 4. Verify Everything Works
```bash
make dev-test  # Should show 11/12 tests passing
```

## Hot-Reloading Development

```bash
# Terminal 1: Start with development mode
DEV_MODE=true make dev-start

# Terminal 2: Enable file sync  
make dev-sync
```

## Essential Commands

```bash
# Day-to-day workflow
make dev-start          # Start environment
make dev-test           # Run tests
make dev-stop           # Stop (keep CRC running)

# Troubleshooting
make dev-clean          # Delete project, fresh start
crc status              # Check CRC status
oc get pods -n vteam-dev # Check pod status
```

## System Requirements

- **CPU**: 4 cores, **RAM**: 11GB, **Disk**: 50GB
- **OS**: macOS 10.15+ or Linux with KVM
- **Reduce if needed**: `CRC_CPUS=2 CRC_MEMORY=6144 make dev-start`

## Common Issues & Fixes

**CRC won't start:**
```bash
crc stop && crc start
```

**DNS issues:**
```bash
sudo bash -c 'echo "127.0.0.1 api.crc.testing" >> /etc/hosts'
```

**Memory issues:**
```bash
CRC_MEMORY=6144 make dev-start
```

**Complete reset:**
```bash
crc stop && crc delete && crc setup && make dev-start
```

---

**📖 Detailed Guides:**
- [Installation Guide](INSTALLATION.md) - Complete setup instructions
- [Hot-Reload Guide](DEV_MODE.md) - Development mode details  
- [Migration Guide](MIGRATION_GUIDE.md) - Moving from Kind to CRC