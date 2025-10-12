# Jira Integration - Progress Notes

### Recent Progress (2025-10-12)

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
       └─ Sub-task (implementation)
```

**Phase 1 (Simple - Demo):**
- `spec.md` → **Feature**
- `plan.md` → **Feature**
- `tasks.md` → Skip (add later)

**Phase 2 (Future):**
- `plan.md` → **Outcome**
- `spec.md` → **Feature** (linked to Outcome)
- `tasks.md` → Parse and create **Sub-tasks**

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

## Future Enhancements

### Phase 2: Advanced Features
- **Validate Outcome exists** before creating Feature
- **Parse tasks.md** and create Sub-tasks automatically
- **Bi-directional sync**: Update GitHub when Jira changes
- **Status syncing**: Map Jira workflow states to RFE phases
- **Webhook support**: Auto-publish on git push

### Phase 3: Abstraction Layer
If air-gapped/on-prem support needed:
```go
type GitProvider interface {
    ReadFile(owner, repo, branch, path string) ([]byte, error)
}

type JiraProvider interface {
    CreateIssue(project, title, content, issueType string, parent *string) (string, error)
}
```

Support: GitHub/Gitea, Jira Cloud/Server/Linear

---

### Modified:
- ✅ `main.go` - Import types package, use type aliases
- ✅ `git.go` - Added `readGitHubFile()` and `parseGitHubURL()` functions
- ✅ `handlers.go` - Fixed `getProjectRFEWorkflow()` to return `parentOutcome` in API response
- ✅ `components/manifests/crds/rfeworkflows-crd.yaml` - Added `parentOutcome` field to spec
- ✅ `components/frontend/src/types/agentic-session.ts` - Added `parentOutcome` to types
- ✅ `components/frontend/src/app/projects/[name]/rfe/new/page.tsx` - Added UI for parentOutcome input
- ✅ `components/frontend/src/app/projects/[name]/rfe/[id]/page.tsx` - Added UI to display parentOutcome

---

## Integration Flow (Complete)

1. **User creates RFE workflow** with optional `parentOutcome` field (e.g., "RHASTRAT-456")
2. **Backend stores** in Kubernetes CRD (`spec.parentOutcome`)
3. **Backend parses** from CRD in `rfeFromUnstructured()` (handlers.go:2445-2448)
4. **Backend returns** in API response from `getProjectRFEWorkflow()` (handlers.go:2708-2710)
5. **Frontend displays** as badge on RFE detail page
6. **When publishing to Jira** (`publishWorkflowFileToJira` in jira.go):
   - Reads file from GitHub umbrella repo
   - Creates Jira Feature issue
   - Links to parent Outcome if `parentOutcome` provided
   - Stores linkage in `RFEWorkflow.jiraLinks[]`

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

