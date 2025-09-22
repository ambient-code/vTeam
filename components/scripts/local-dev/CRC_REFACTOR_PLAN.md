# Planning Prompt: Refactor Local Dev to Use OpenShift Local (CRC)

## Context

The vTeam project currently has local development scripts that attempt to run on Kind/vanilla Kubernetes, but the backend is **hardcoded for OpenShift** and fails on Kind due to missing OpenShift-specific resources.

### Current Architecture Issues
- Backend expects `project.openshift.io/v1` resources (OpenShift Projects)
- Hardcoded OpenShift RBAC patterns
- No fallback to vanilla Kubernetes namespaces
- Crashes with nil pointer dereference on Kind: `handlers.go:1972`

### What We Have
- Working scripts in `components/scripts/local-dev/` that:
  - ✅ Create Kind cluster and apply CRDs
  - ✅ Start backend (health endpoint works)
  - ✅ Start frontend (UI loads)
  - ❌ Backend APIs crash when called (OpenShift dependency)

### Repository Structure
```
vTeam/
├── components/
│   ├── backend/           # Go API - OpenShift specific
│   ├── frontend/          # Next.js UI
│   ├── manifests/         # K8s manifests with deploy.sh for OpenShift
│   └── scripts/local-dev/ # Current Kind-based scripts (partially working)
└── Makefile              # Has dev-start/dev-stop/dev-test targets
```

### Current Scripts
- `start.sh`: Creates Kind cluster, applies CRDs, starts Go backend + Next.js frontend
- `stop.sh`: Kills local processes
- `test.sh`: Smoke tests (health endpoints work, APIs fail)
- Makefile targets: `make dev-start`, `make dev-stop`, `make dev-test`

## Task: Refactor for OpenShift Local (CRC)

### Goal
Create a reliable local development environment that matches production OpenShift behavior using Red Hat CodeReady Containers (CRC).

### Requirements

#### 1. CRC Integration
- Replace Kind with CRC for local OpenShift cluster
- Detect if CRC is installed, provide installation guidance if missing
- Handle CRC lifecycle (start/stop/status)
- Manage CRC resource allocation (CPU/memory requirements)

#### 2. OpenShift-Native Development
- Use actual OpenShift Projects instead of vanilla K8s namespaces
- Leverage OpenShift OAuth for local authentication (matches production)
- Use OpenShift Routes instead of port-forwarding
- Test with real OpenShift RBAC patterns

#### 3. Cross-Platform Support
- macOS (primary target)
- Fedora/RHEL (secondary)
- Handle different CRC installation methods per platform

#### 4. Developer Experience
- Maintain same Makefile interface (`make dev-start`, etc.)
- Idempotent operations (safe to re-run)
- Clear error messages and installation guidance
- Proper cleanup on stop

#### 5. Authentication Parity
- Use OpenShift OAuth (same as production)
- Create development users with proper roles
- Test actual auth flows (no bypass/disable modes)
- Validate role-based access controls

### Technical Specifications

#### CRC Requirements
- Minimum CRC version: 2.x
- Resource allocation: 4 CPU, 8GB RAM (configurable)
- Pull secret handling for Red Hat registry access
- Network configuration for host access

#### Authentication Flow
- OpenShift OAuth client configuration
- Development user creation with roles:
  - `admin`: Full project access
  - `edit`: Project edit permissions  
  - `view`: Read-only access
- Token-based API testing (no header spoofing)

#### Service Deployment
- Use existing `components/manifests/deploy.sh` as reference
- Apply CRDs and manifests to CRC cluster
- Configure Routes for external access
- Validate all services start correctly

### Expected Deliverables

#### 1. Updated Scripts
- `components/scripts/local-dev/crc-start.sh`
- `components/scripts/local-dev/crc-stop.sh`  
- `components/scripts/local-dev/crc-test.sh`
- Update Makefile targets to use CRC scripts

#### 2. Documentation
- Installation guide for CRC per platform
- Authentication setup instructions
- Troubleshooting guide for common CRC issues
- Migration guide from Kind-based scripts

#### 3. Configuration
- CRC resource configuration
- OpenShift user/role setup
- Route configuration for local access
- Environment variable documentation

### Success Criteria
- [ ] `make dev-start` creates working OpenShift Local environment
- [ ] All backend APIs work without crashes
- [ ] Frontend can successfully call backend APIs
- [ ] Authentication works with real OpenShift OAuth
- [ ] `make dev-test` passes all checks including API calls
- [ ] Idempotent - safe to run multiple times
- [ ] Proper cleanup with `make dev-stop`

### Edge Cases to Handle
- CRC not installed (provide installation instructions)
- Insufficient system resources (memory/CPU warnings)
- CRC cluster already exists (reuse vs recreate)
- Pull secret missing (guide user to Red Hat developer account)
- Port conflicts (CRC default ports vs other services)
- Network issues (firewall, VPN interference)

### Testing Strategy
- Health checks for all services
- API endpoint validation with real tokens
- Authentication flow testing (login/logout)
- Role-based access control validation
- Cross-platform testing (macOS primary, Linux secondary)

### Implementation Notes
- Preserve existing Makefile interface for consistency
- Use CRC's built-in OpenShift features instead of recreating them
- Follow OpenShift best practices for local development
- Ensure scripts work in both interactive and CI environments

### Current Working Directory
All scripts should be implemented in `components/scripts/local-dev/` and called via Makefile targets from the repository root.

---

**Please implement this refactor to provide a fully functional OpenShift Local development environment that matches production behavior.**
