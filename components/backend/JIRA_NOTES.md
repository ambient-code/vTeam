# Jira Integration - Progress Notes

### Recent Progress (2025-10-12 Evening)

✅ **Phase-Specific Jira Integration with Attachments**
- Implemented phase-specific Jira publishing logic (specify, plan, tasks)
- Added file attachment support to Jira issues
- **Specify Phase**: Creates Feature from spec.md, attaches rfe.md
- **Plan Phase**: Creates Epic from plan.md (Option B architecture)
- **Tasks Phase**: Attaches tasks.md to existing Feature (from specify)
- Added "Push to Jira" button to sessions overview page
- Frontend auto-detects phase and determines correct file paths
- Backend validates prerequisites (e.g., spec.md must exist before tasks)

### Earlier Progress (2025-10-12)

✅ **Frontend Jira Integration Fully Restored**
- Fixed workflow.jiraLinks type assertion in page.tsx (lines 342-344)
- All Jira integration frontend code verified and working
- Error handling complete across backend and frontend
- Types properly defined in types/agentic-session.ts

### Previous Progress (2025-01-09)

✅ **Jira Integration Complete**
- All Jira integration restored
- Types refactored into `types/` package

## Goal
Restore Jira integration from commit `9d76b17b6ca62d1f3` to current codebase, with improvements.

---

## Background: Why Jira Integration Was Broken

**Old Implementation (commit 9d76b17b6ca62d1f3):**
- Read spec files from workspace PVC via `ambient-content` service
- Published content to Jira via v2 REST API
- Stored linkage in `RFEWorkflow.JiraLinks[]`

**Why It Broke:**
- Workspace PVC/content service was removed
- Design shifted to **GitHub as source of truth** for specs
- `publishWorkflowFileToJira` was stubbed out with "workspace API removed" error

---

## Jira Integration Design

### Mapping: Spec-Kit Artifacts → Jira Issue Types

Based on team's hierarchy (from diagram):
```
Outcome (strategic, top-level)
  └─ Feature (work unit)
       ├─ Epic (implementation plan)
       └─ Sub-task (implementation)
```

**Current Implementation (Phase 1.5):**
- `spec.md` → **Feature** (with `rfe.md` attached)
- `plan.md` → **Epic** (separate issue with plan content and artifact links)
- `tasks.md` → **Attachment** to Feature

**Phase 2 (Future):**
- Parse `tasks.md` and create **Sub-tasks** under Feature
- Auto-link Epic to Feature
- Support multiple Features under single Outcome

### Parent Outcome Linking

**User-provided field in RFE creation:**
```json
{
  "title": "...",
  "description": "...",
  "parentOutcome": "RHASTRAT-456"  // Optional Jira Outcome key
}
```

**Logic:**
- If `parentOutcome` provided → include `"parent": {"key": "RHASTRAT-456"}` in Jira API
- If not → create standalone Feature
- No validation errors, just works either way

### Jira API Support

**Jira Cloud vs Server/Data Center:**
- **Endpoint**: Same (`/rest/api/2/issue`)
- **Payload**: Identical
- **Auth Difference**:
  - Cloud: `Authorization: Basic base64(email:api_token)`
  - Server: `Authorization: Bearer PAT_token`

**Auto-detection:**
```go
if strings.Contains(jiraURL, "atlassian.net") {
    // Jira Cloud
    return "Basic " + base64(jiraToken)
}
// Jira Server
return "Bearer " + jiraToken
```

**Runner Secrets Configuration:**
```
JIRA_URL=https://issues.redhat.com (or https://yourorg.atlassian.net)
JIRA_PROJECT=RHASTRAT
JIRA_API_TOKEN=<token>  // Format depends on Cloud vs Server
```

---

---

## Testing the Integration

### Test Scenario 1: Specify Phase
1. Create RFE workflow with umbrella repo
2. Run specify session → creates spec.md and rfe.md
3. Click "Publish to Jira" on RFE page (or "Push to Jira" in sessions list)
4. Verify in Jira:
   - Feature created with spec.md title and content
   - rfe.md attached as file
   - If parentOutcome set, Feature linked to Outcome

### Test Scenario 2: Plan Phase
1. Run plan session on same RFE → creates plan.md
2. Click "Publish to Jira" for plan phase
3. Verify in Jira:
   - Epic created (not Feature)
   - Epic contains plan.md content
   - Epic key stored in jiraLinks

### Test Scenario 3: Tasks Phase
1. Run tasks session on same RFE → creates tasks.md
2. Click "Push to Jira" for tasks phase
3. Verify in Jira:
   - tasks.md attached to the Feature from specify
   - No new issue created
   - Error if spec.md wasn't published first

### Test Scenario 4: Sessions Overview
1. Navigate to Sessions page
2. Find completed specify/plan/tasks session
3. Click actions dropdown → "Push to Jira"
4. Verify sync succeeds with success alert

---

## Future Enhancements

### Phase 2: Advanced Features
- **Auto-link Epic to Feature**: Link plan Epic to spec Feature
- **Validate Outcome exists** before creating Feature
- **Parse tasks.md** and create Sub-tasks automatically
- **Bi-directional sync**: Update GitHub when Jira changes
- **Status syncing**: Map Jira workflow states to RFE phases
- **Webhook support**: Auto-publish on git push
- **Bulk operations**: Publish all phases at once
- **Attachment visualization**: Show attached files in UI

### Phase 3: Abstraction Layer
If air-gapped/on-prem support needed:
```go
type GitProvider interface {
    ReadFile(owner, repo, branch, path string) ([]byte, error)
}

type JiraProvider interface {
    CreateIssue(project, title, content, issueType string, parent *string) (string, error)
    AttachFile(issueKey, filename string, content []byte) error
}
```

Support: GitHub/Gitea, Jira Cloud/Server/Linear

---

### Modified Files:

**Backend:**
- ✅ `jira/integration.go`:
  - Added `AttachFileToJiraIssue()` function for file attachments
  - Implemented custom `MultipartWriter` for proper multipart form-data
  - Enhanced `PublishWorkflowFileToJira()` with phase-specific logic
  - Added specify phase: auto-attaches rfe.md to Feature
  - Added plan phase: creates Epic issue type
  - Added tasks phase: attaches tasks.md to Feature from specify
- ✅ `git/operations.go` - Existing `ReadGitHubFile()` and `ParseGitHubURL()` functions used
- ✅ `handlers/rfe.go` - Returns `parentOutcome` in API response
- ✅ `types/rfe.go` - RFEWorkflow types with JiraLinks support

**Frontend:**
- ✅ `src/app/projects/[name]/rfe/[id]/page.tsx`:
  - Updated Jira publish to pass `phase` parameter
  - Existing per-phase buttons now include phase context
- ✅ `src/app/projects/[name]/sessions/page.tsx`:
  - Added `handleJiraSync()` function
  - Added "Push to Jira" button in actions dropdown
  - Auto-detects phase from session labels
  - Determines correct file path (spec.md/plan.md/tasks.md)
  - Shows for completed sessions with RFE workflow linkage
- ✅ `src/types/agentic-session.ts` - WorkflowJiraLink and parentOutcome types

**Kubernetes:**
- ✅ `components/manifests/crds/rfeworkflows-crd.yaml` - parentOutcome and jiraLinks fields

---

## Integration Flow (Complete)

### RFE Creation Flow:
1. **User creates RFE workflow** with optional `parentOutcome` field (e.g., "RHASTRAT-456")
2. **Backend stores** in Kubernetes CRD (`spec.parentOutcome`)
3. **Backend parses** from CRD in `RfeFromUnstructured()` (jira/integration.go)
4. **Backend returns** in API response from `GetProjectRFEWorkflow()` (handlers/rfe.go)
5. **Frontend displays** as badge on RFE detail page

### Jira Publishing Flow:

**Specify Phase (spec.md):**
1. User clicks "Publish to Jira" on RFE page or sessions page
2. Frontend sends: `{ path: "spec.md", phase: "specify" }`
3. Backend reads `spec.md` from GitHub umbrella repo
4. Creates Jira **Feature** with spec.md content
5. Links to parent Outcome if `parentOutcome` provided
6. Reads `rfe.md` from GitHub (if exists)
7. **Attaches rfe.md** to the Feature
8. Stores linkage in `RFEWorkflow.jiraLinks[]`

**Plan Phase (plan.md):**
1. User clicks "Publish to Jira" for plan session
2. Frontend sends: `{ path: "plan.md", phase: "plan" }`
3. Backend reads `plan.md` from GitHub umbrella repo
4. Creates Jira **Epic** (not Feature) with plan.md content
5. Epic description contains plan content and links to artifacts
6. Stores linkage in `RFEWorkflow.jiraLinks[]`

**Tasks Phase (tasks.md):**
1. User clicks "Push to Jira" for tasks session
2. Frontend sends: `{ path: "tasks.md", phase: "tasks" }`
3. Backend searches for existing Feature (from spec.md jiraLink)
4. **Validates Feature exists** (spec.md must be published first)
5. Reads `tasks.md` from GitHub umbrella repo
6. **Attaches tasks.md** to the existing Feature
7. Updates linkage in `RFEWorkflow.jiraLinks[]`

---

## Notes on Design Choices

**Why GitHub as source of truth:**
- Specs are deliverables, not scratch space
- Version history is critical for understanding decisions
- Jira should link to committed, stable versions
- No PVC management overhead

**Why user-provided parentOutcome:**
- Outcomes are strategic, created by leadership
- Pre-exist before RFE work starts
- Realistic workflow: users know the Outcome key
- Simple UX: optional text field

**Why support both Jira Cloud and Server:**
- Auto-detection is trivial (check URL)
- Same API payload
- Only auth header differs
- Minimal code complexity

---

