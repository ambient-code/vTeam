# Installation Guide: OpenShift Local (CRC) Development Environment

This guide walks you through installing and setting up the OpenShift Local (CRC) development environment for vTeam.

## Quick Start

```bash
# 1. Install CRC (choose your platform below)
# 2. Get Red Hat pull secret (see below)  
# 3. Start development environment
make dev-start
```

## Platform-Specific Installation

### macOS

**Option 1: Homebrew (Recommended)**
```bash
brew install crc
```

**Option 2: Manual Download**
```bash
# Download latest CRC for macOS
curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest/crc-macos-amd64.tar.xz

# Extract
tar -xf crc-macos-amd64.tar.xz

# Install
sudo cp crc-macos-*/crc /usr/local/bin/
chmod +x /usr/local/bin/crc
```

### Linux (Fedora/RHEL/CentOS)

**Fedora/RHEL/CentOS:**
```bash
# Download latest CRC for Linux
curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest/crc-linux-amd64.tar.xz

# Extract and install
tar -xf crc-linux-amd64.tar.xz
sudo cp crc-linux-*/crc /usr/local/bin/
sudo chmod +x /usr/local/bin/crc
```

**Ubuntu/Debian:**
```bash
# Same as above - CRC is a single binary
curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest/crc-linux-amd64.tar.xz
tar -xf crc-linux-amd64.tar.xz
sudo cp crc-linux-*/crc /usr/local/bin/
sudo chmod +x /usr/local/bin/crc

# Install virtualization dependencies
sudo apt update
sudo apt install -y qemu-kvm libvirt-daemon libvirt-daemon-system
sudo usermod -aG libvirt $USER
# Logout and login for group changes to take effect
```

### Verify Installation
```bash
crc version
# Should show CRC version info
```

## Red Hat Pull Secret Setup

### 1. Get Your Pull Secret
1. Visit: https://console.redhat.com/openshift/create/local
2. **Create a free Red Hat account** if you don't have one
3. **Download your pull secret** (it's a JSON file)

### 2. Save Pull Secret
```bash
# Create CRC config directory
mkdir -p ~/.crc

# Save your downloaded pull secret
cp ~/Downloads/pull-secret.txt ~/.crc/pull-secret.json

# Or if the file has a different name:
cp ~/Downloads/your-pull-secret-file.json ~/.crc/pull-secret.json
```

## Initial Setup

### 1. Run CRC Setup
```bash
# This configures your system for CRC (one-time setup)
crc setup
```

**What this does:**
- Downloads OpenShift VM image (~2.3GB)
- Configures virtualization
- Sets up networking
- **Takes 5-10 minutes**

### 2. Configure CRC
```bash
# Configure pull secret
crc config set pull-secret-file ~/.crc/pull-secret.json

# Optional: Configure resources (adjust based on your system)
crc config set cpus 4
crc config set memory 8192      # 8GB RAM
crc config set disk-size 50     # 50GB disk
```

### 3. Install Additional Tools

**jq (required for scripts):**
```bash
# macOS
brew install jq

# Linux
sudo apt install jq         # Ubuntu/Debian
sudo yum install jq         # RHEL/CentOS
sudo dnf install jq         # Fedora
```

## System Requirements

### Minimum Requirements
- **CPU:** 4 cores
- **RAM:** 11GB free (for CRC VM)
- **Disk:** 50GB free space
- **Network:** Internet access for image downloads

### Recommended Requirements
- **CPU:** 6+ cores
- **RAM:** 12+ GB total system memory
- **Disk:** SSD storage for better performance

### Platform Support
- **macOS:** 10.15+ (Catalina or later)
- **Linux:** RHEL 8+, Fedora 30+, Ubuntu 18.04+
- **Virtualization:** Intel VT-x/AMD-V required

## First Run

```bash
# Start your development environment
make dev-start
```

**First run will:**
1. Start CRC cluster (5-10 minutes)
2. Download/configure OpenShift
3. Create vteam-dev project
4. Build and deploy applications
5. Configure routes and services

**Expected output:**
```
✅ OpenShift Local development environment ready!
  Backend:   https://vteam-backend-vteam-dev.apps-crc.testing/health
  Frontend:  https://vteam-frontend-vteam-dev.apps-crc.testing
  Project:   vteam-dev
  Console:   https://console-openshift-console.apps-crc.testing
```

## Verification

```bash
# Run comprehensive tests
make dev-test

# Should show all tests passing
```

## Common Installation Issues

### Pull Secret Problems
```bash
# Error: "pull secret file not found"
# Solution: Ensure pull secret is saved correctly
ls -la ~/.crc/pull-secret.json
cat ~/.crc/pull-secret.json  # Should be valid JSON
```

### Virtualization Not Enabled
```bash
# Error: "Virtualization not enabled"
# Solution: Enable VT-x/AMD-V in BIOS
# Or check if virtualization is available:
# Linux:
egrep -c '(vmx|svm)' /proc/cpuinfo   # Should be > 0
# macOS: VT-x is usually enabled by default
```

### Insufficient Resources
```bash
# Error: "not enough memory/CPU"
# Solution: Reduce CRC resource allocation
crc config set cpus 2
crc config set memory 6144
```

### Firewall/Network Issues
```bash
# Error: "Cannot reach OpenShift API"
# Solution: 
# 1. Temporarily disable VPN
# 2. Check firewall settings
# 3. Ensure ports 6443, 443, 80 are available
```

### Permission Issues (Linux)
```bash
# Error: "permission denied" during setup
# Solution: Add user to libvirt group
sudo usermod -aG libvirt $USER
# Then logout and login
```

## Resource Configuration

### Low-Resource Systems
```bash
# Minimum viable configuration
crc config set cpus 2
crc config set memory 4096
crc config set disk-size 40
```

### High-Resource Systems
```bash
# Performance configuration
crc config set cpus 6
crc config set memory 12288
crc config set disk-size 80
```

### Check Current Config
```bash
crc config view
```

## Uninstall

### Remove CRC Completely
```bash
# Stop and delete CRC
crc stop
crc delete

# Remove CRC binary
sudo rm /usr/local/bin/crc

# Remove CRC data (optional)
rm -rf ~/.crc

# macOS: If installed via Homebrew
brew uninstall crc
```

## Next Steps

After installation:
1. **Read the [README.md](README.md)** for usage instructions
2. **Read the [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)** if upgrading from Kind
3. **Start developing:** `make dev-start`
4. **Run tests:** `make dev-test`
5. **Access the console:** Visit the console URL from `make dev-start` output

## Getting Help

### Check Installation
```bash
crc version                    # CRC version
crc status                     # Cluster status
crc config view               # Current configuration
```

### Support Resources
- [CRC Official Docs](https://crc.dev/crc/)
- [Red Hat OpenShift Local](https://developers.redhat.com/products/openshift-local/overview)
- [CRC GitHub Issues](https://github.com/code-ready/crc/issues)

### Reset Installation
```bash
# If something goes wrong, reset everything
crc stop
crc delete
rm -rf ~/.crc
# Then start over with crc setup
```
