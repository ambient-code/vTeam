# Secret Management Convergence Plan

## Current Status (Sept 22, 2025)

### Problem Identified
Two conflicting secret management approaches exist:

1. **Operator Approach (New)**: Copies secrets from operator namespace to managed namespaces
2. **UI/Backend Approach (Existing)**: Users directly create/edit secrets via web UI

### Key Conflicts
- **Different default names**: `ambient-code-secrets` (operator) vs `ambient-runner-secrets` (UI)
- **Different ownership**: Operator manages copies vs UI manages direct creation
- **Conflicting sources of truth**: Operator overwrites UI changes

### Decision Made: Option A (Operator-First)
- UI becomes configuration interface for operator (not direct secret editor)
- Operator handles all secret copying/syncing
- Central secrets stored in operator namespace
- Better security and consistency

## Implementation Plan

### Architecture Changes
```
OLD: UI → Backend → Direct K8s Secret Creation
NEW: UI → Backend → ProjectSettings → Operator → Secret Copying
```

### Design Decisions
1. **No Migration Strategy**: Fresh start, clear existing environments
2. **Central Secret Location**: Operator namespace only
   - Created via `deploy.sh` + `.env` OR manually by users
3. **Simple UI**: Single input field for source secret name, no dropdown/discovery

### New UI Design
Replace current secret key/value editor with:
- Source Secret Name input (default: `ambient-code-secrets`)
- Expected keys documentation (ANTHROPIC_API_KEY required, git tokens optional)
- Status display (copied/not found/syncing)
- Save configuration button

### Required Changes

#### Backend (`components/backend/handlers.go`)
- **Remove**: Lines 3395-3444 (`updateRunnerSecrets`, `listRunnerSecrets`)
- **Keep**: `updateRunnerSecretsConfig`, `getRunnerSecretsConfig`
- **Add**: `validateSourceSecret` endpoint

#### Frontend (`components/frontend/src/app/projects/[name]/settings/page.tsx`)
- **Remove**: Lines 24-100+ (secret key/value editing UI)
- **Replace with**: Source secret configuration UI
- **Remove**: `/api/projects/[name]/runner-secrets` PUT endpoint
- **Keep**: `/api/projects/[name]/runner-secrets/config` endpoints

#### Operator (Already Fixed)
- ✅ Constants fixed (`ambient-workspace` → `resources.WorkspacePVCName`)
- ✅ Service URLs fixed (`ambient-content` → `resources.ContentServiceName`)
- ✅ Compilation verified
- **Next**: Deploy fixed operator to resolve current PVC/service issues

## Current Blocker

**Pod Pending Issue**: Agent pod failing to schedule due to:
```
persistentvolumeclaim "resources.WorkspacePVCName" not found
PVC_PROXY_API_URL: http://resources.ContentServiceName.sallyom-rfe-test.svc:8080
```

**Root Cause**: Constants were being used as literal strings instead of values.

**Status**: ✅ **FIXED** - Operator code corrected, needs redeployment

## Next Steps (In Order)

1. **Deploy fixed operator** to resolve immediate pod scheduling issue
2. **Verify basic operator functionality** (PVC creation, service creation)
3. ✅ **Implement backend changes** (remove direct secret management)
4. ✅ **Implement frontend changes** (new configuration UI)
5. **Test end-to-end flow** with operator-first secret management

## Files Modified

### Operator (Fixed)
- ✅ `components/operator/main.go` - Fixed constant usage
- ✅ `components/operator/resources/constants.go` - Added secret key constants
- ✅ `components/operator/resources/secrets.go` - Added validation, documentation
- ✅ `components/operator/RFE-OPERATOR.md` - Added deployment docs

### Backend (Converged)
- ✅ `components/backend/handlers.go` - Removed direct secret management functions:
  - Removed `updateRunnerSecrets` (lines 3395-3444)
  - Removed `listRunnerSecrets` (lines 3351-3393)
  - Added `validateSourceSecret` - validates operator secret existence
  - Added `triggerSecretSync` - triggers operator reconciliation
- ✅ `components/backend/main.go` - Updated route registrations:
  - Changed `/runner-secrets` PUT/GET to `/runner-secrets/validate` GET and `/runner-secrets/trigger-sync` PUT
  - Kept `/runner-secrets/config` GET/PUT for configuration

### Frontend (Converged)
- ✅ `components/frontend/src/app/projects/[name]/settings/page.tsx` - Replaced secret management UI:
  - Removed complex key/value editor (lines 24-100+)
  - Added simple source secret configuration interface
  - Added source secret validation with status display
  - Added trigger sync functionality
  - Added documentation of expected secret keys

## Test Environment
- Namespace: `sallyom-rfe-test`
- Operator: `ambient-code` namespace
- Status: Ready for redeployment with fixed operator