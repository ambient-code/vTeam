# Local Dev with OpenShift Local (CRC)

## Status: ✅ FULLY FUNCTIONAL WITH PROJECT CREATION

The vTeam local development environment uses **OpenShift Local (CRC 2.54.0)** with **OpenShift 4.19.8** to provide a production-like development experience that matches the backend's OpenShift requirements.

**✅ CONFIRMED WORKING:**
- Project creation via UI
- Frontend-backend authentication  
- OpenShift Routes with TLS
- Real token-based API access
- Hot-reloading development mode

## What Works
- ✅ Full OpenShift cluster with CRC
- ✅ Backend API endpoints (including `/api/projects`)
- ✅ Frontend UI with backend integration
- ✅ OpenShift Projects (not vanilla K8s namespaces)
- ✅ OpenShift OAuth and RBAC
- ✅ OpenShift Routes (no port-forwarding needed)
- ✅ Real token-based authentication
- ✅ End-to-end workflow testing

## Prerequisites

### Install CodeReady Containers (CRC)

**macOS:**
```bash
# Using Homebrew (recommended)
brew install crc

# Or download from: https://crc.dev/crc/getting_started/getting_started/installing/
```

**Linux (Fedora/RHEL/Ubuntu):**
```bash
# Download the latest release
curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest/crc-linux-amd64.tar.xz

# Extract and install
tar -xf crc-linux-amd64.tar.xz
sudo cp crc-linux-*/crc /usr/local/bin/
```

### Get Red Hat Pull Secret

1. Visit: https://console.redhat.com/openshift/create/local
2. Log in with your Red Hat account (free registration available)
3. Download your pull secret
4. Save it to `~/.crc/pull-secret.json`

### System Requirements

- **CPU:** 4 cores minimum (configurable with `CRC_CPUS`)
- **RAM:** 11GB minimum (configurable with `CRC_MEMORY`)
- **Disk:** 50GB free space (configurable with `CRC_DISK`)
- **OS:** macOS 10.15+, Linux with KVM support

## Usage

### Start Development Environment
```bash
make dev-start
```

**What it does:**
- Starts CRC OpenShift cluster (if not running)
- Creates `vteam-dev` OpenShift project
- Applies CRDs (Custom Resource Definitions)
- Creates service accounts with proper RBAC
- Builds and deploys backend/frontend containers
- Configures OpenShift Routes for external access
- **Takes 5-10 minutes on first run**

### Test Environment
```bash
make dev-test
```

**Validates:**
- CRC cluster status
- OpenShift authentication
- Project and resource existence
- Service deployments and health
- Route accessibility
- Backend API with real OpenShift tokens
- RBAC permissions

### Stop Development Environment
```bash
make dev-stop                    # Keep CRC running (faster restart)
make dev-stop-cluster           # Stop CRC cluster too
make dev-clean                  # Delete entire OpenShift project
```

## Access URLs

After running `make dev-start`, you'll see output like:
```
✅ OpenShift Local development environment ready!
  Backend:   https://vteam-backend-vteam-dev.apps-crc.testing/health
  Frontend:  https://vteam-frontend-vteam-dev.apps-crc.testing
  Project:   vteam-dev
  Console:   https://console-openshift-console.apps-crc.testing
```

## Configuration

### Environment Variables
- `CRC_CPUS=4` - CPU cores for CRC
- `CRC_MEMORY=11264` - Memory in MB for CRC
- `CRC_DISK=50` - Disk size in GB for CRC
- `PROJECT_NAME=vteam-dev` - OpenShift project name

### Example:
```bash
CRC_CPUS=6 CRC_MEMORY=12288 make dev-start
```

## Development Workflow

1. **Start environment:** `make dev-start`
2. **Develop:** Edit code in `components/backend/` or `components/frontend/`
3. **Rebuild/Deploy:** Re-run `make dev-start` (idempotent)
4. **Test:** `make dev-test`
5. **Access:** Use the provided URLs
6. **Stop:** `make dev-stop` when done

## OpenShift CLI Usage

After starting, you can use `oc` commands:
```bash
oc project vteam-dev                    # Switch to project
oc get pods                             # View running pods
oc logs deployment/vteam-backend        # View backend logs
oc describe route vteam-frontend        # Check frontend route
```

## Troubleshooting

### CRC Won't Start
```bash
crc status                              # Check status
crc stop && crc start                   # Restart
crc delete && make dev-start            # Full reset
```

### Pull Secret Issues
```bash
# Re-download from https://console.redhat.com/openshift/create/local
cp ~/Downloads/pull-secret.txt ~/.crc/pull-secret.json
crc setup
```

### Memory Issues
```bash
# Reduce resources
CRC_MEMORY=6144 CRC_CPUS=2 make dev-start
```

### Port Conflicts
CRC uses these ports by default:
- `6443` - OpenShift API
- `443` - OpenShift Routes (HTTPS)
- `80` - OpenShift Routes (HTTP)

### Firewall/VPN Issues
- Disable VPN temporarily during CRC setup
- Ensure `.apps-crc.testing` domains resolve to `127.0.0.1`

### DNS Resolution Issues
If you get "NXDOMAIN" errors for `api.crc.testing`:

**Option 1: Manual /etc/hosts entries**
```bash
sudo bash -c 'echo "127.0.0.1 api.crc.testing" >> /etc/hosts'
sudo bash -c 'echo "127.0.0.1 oauth-openshift.apps-crc.testing" >> /etc/hosts' 
sudo bash -c 'echo "127.0.0.1 console-openshift-console.apps-crc.testing" >> /etc/hosts'
```

**Option 2: DNS server configuration (corporate environments)**
Configure your DNS server to resolve `*.crc.testing` to `127.0.0.1`

### Reset Everything
```bash
make dev-clean                          # Delete project
crc stop && crc delete                  # Delete CRC VM
crc setup && make dev-start             # Fresh start
```

## Migration from Kind-based Setup

The old Kind-based scripts (`start.sh`, `stop.sh`, `test.sh`) have been **removed**. The new CRC-based setup provides:

- ✅ Production parity with OpenShift
- ✅ Real OpenShift Projects (not namespaces)
- ✅ Native OpenShift OAuth/RBAC
- ✅ OpenShift Routes instead of port-forwarding
- ✅ Proper token-based authentication
- ✅ No backend crashes due to missing OpenShift resources

## Architecture

```
┌─────────────────────────────────────────────┐
│ CRC (OpenShift Local)                       │
│ ┌─────────────────────────────────────────┐ │
│ │ vteam-dev Project                       │ │
│ │ ┌─────────────┐  ┌─────────────────┐   │ │
│ │ │ Backend     │  │ Frontend        │   │ │
│ │ │ Deployment  │  │ Deployment      │   │ │
│ │ └─────────────┘  └─────────────────┘   │ │
│ │ ┌─────────────────────────────────────┐ │ │
│ │ │ Routes (External Access)            │ │ │
│ │ └─────────────────────────────────────┘ │ │
│ └─────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

This provides a complete OpenShift development environment that matches production behavior.