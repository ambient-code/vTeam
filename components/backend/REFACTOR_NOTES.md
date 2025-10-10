# Backend Refactor & Jira Integration - Progress Notes

## Status: 🔄 REBASED ON PR #152 - READY FOR HANDLERS REFACTOR

### Recent Progress (2025-01-09)

✅ **Successfully rebased on PR #152** (merged into upstream/main)
- Resolved conflicts in `git.go` (merged both Jira helpers and GitHub file reading)
- Resolved conflicts in `main.go` (kept types package refactoring with aliases)
- Fixed missing fields in types after merge:
  - Added `EnvironmentVariables` to `AgenticSessionSpec`
  - Added `Status` to `SessionRepoMapping`
- ✅ Build verified clean with no errors

✅ **Jira Integration Complete**
- All Jira integration features working
- Types refactored into `types/` package
- Frontend has GitHub App note in Project Settings

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

**Design Decision: GitHub is Better**
- ✅ Specs are versioned artifacts, not ephemeral workspace files
- ✅ Collaboration via git (PRs, branches)
- ✅ Auditability (full history)
- ✅ Jira links to **committed** versions (stable)
- ✅ No PVC management overhead
- ✅ Portable (not locked in cluster)

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

## Code Refactoring Completed

### ✅ Phase 1: Types Package (DONE)

**Created `types/` package:**
```
types/
├── common.go     - GitRepository, UserContext, LLMSettings, etc.
├── session.go    - AgenticSession, CreateSessionRequest, etc.
├── rfe.go        - RFEWorkflow, WorkflowJiraLink, etc.
├── project.go    - AmbientProject, CreateProjectRequest
```

**Updated `main.go`:**
- Import: `"ambient-code-backend/types"`
- Type aliases for backward compatibility:
  ```go
  type AgenticSession = types.AgenticSession
  type RFEWorkflow = types.RFEWorkflow
  // ... etc
  ```

**Verified:** ✅ Compiles successfully with zero logic changes

---

## Implementation Tasks Remaining

### 1. Add `parentOutcome` Field to RFEWorkflow

**File:** `types/rfe.go`
```go
type RFEWorkflow struct {
    // ... existing fields
    JiraLinks       []WorkflowJiraLink `json:"jiraLinks,omitempty"`
    ParentOutcome   *string            `json:"parentOutcome,omitempty"` // NEW
}
```

**File:** `types/rfe.go`
```go
type CreateRFEWorkflowRequest struct {
    // ... existing fields
    ParentOutcome   *string            `json:"parentOutcome,omitempty"` // NEW
}
```

**File:** `main.go` (or future `k8s/resources.go`)
Update `rfeWorkflowToCRObject()` to serialize `parentOutcome` to CR spec.

### 2. GitHub API Helper (Read File Content)

**File:** `git.go` (add function)
```go
// readGitHubFile reads a file from a GitHub repo using Contents API
func readGitHubFile(ctx context.Context, owner, repo, branch, path, token string) ([]byte, error) {
    apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
        owner, repo, path, branch)

    req, _ := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Accept", "application/vnd.github.v3.raw") // Get raw content

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("GitHub API error: %s (body: %s)", resp.Status, string(body))
    }

    return io.ReadAll(resp.Body)
}
```

**Usage pattern:**
```go
owner, repo, _ := parseGitHubURL(wf.UmbrellaRepo.URL) // Already exists
branch := "main"
if wf.UmbrellaRepo.Branch != nil {
    branch = *wf.UmbrellaRepo.Branch
}
content, err := readGitHubFile(ctx, owner, repo, branch, "specs/foo/spec.md", githubToken)
```

### 3. Restore `publishWorkflowFileToJira`

**File:** `handlers.go` (lines 2879-2952)

**Current state:** Returns `410 Gone` with "workspace API removed"

**New implementation:**
```go
func publishWorkflowFileToJira(c *gin.Context) {
    project := c.Param("projectName")
    id := c.Param("id")

    var req struct {
        Path string `json:"path" binding:"required"` // e.g., "specs/foo/spec.md"
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
        return
    }

    // 1. Load RFE workflow
    _, reqDyn := getK8sClientsForRequest(c)
    reqK8s, _ := getK8sClientsForRequest(c)
    gvrWf := getRFEWorkflowResource()
    item, err := reqDyn.Resource(gvrWf).Namespace(project).Get(c.Request.Context(), id, v1.GetOptions{})
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
        return
    }
    wf := rfeFromUnstructured(item)

    // 2. Get GitHub token
    userID, _ := c.Get("userID")
    userIDStr, _ := userID.(string)
    githubToken, err := getGitHubToken(c.Request.Context(), reqK8s, reqDyn, project, userIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 3. Read file from GitHub
    owner, repo, _ := parseGitHubURL(wf.UmbrellaRepo.URL)
    branch := "main"
    if wf.UmbrellaRepo.Branch != nil {
        branch = *wf.UmbrellaRepo.Branch
    }
    content, err := readGitHubFile(c.Request.Context(), owner, repo, branch, req.Path, githubToken)
    if err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read file from GitHub", "details": err.Error()})
        return
    }

    // 4. Extract title from markdown
    title := extractTitleFromContent(string(content)) // Already exists in handlers.go
    if title == "" {
        title = wf.Title // Fallback to workflow title
    }

    // 5. Load Jira configuration from runner secrets
    secretName := "ambient-runner-secrets"
    if obj, err := reqDyn.Resource(getProjectSettingsResource()).Namespace(project).Get(c.Request.Context(), "projectsettings", v1.GetOptions{}); err == nil {
        if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
            if v, ok := spec["runnerSecretsName"].(string); ok && strings.TrimSpace(v) != "" {
                secretName = strings.TrimSpace(v)
            }
        }
    }

    sec, err := reqK8s.CoreV1().Secrets(project).Get(c.Request.Context(), secretName, v1.GetOptions{})
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read runner secret", "details": err.Error()})
        return
    }

    get := func(k string) string {
        if b, ok := sec.Data[k]; ok {
            return string(b)
        }
        return ""
    }

    jiraURL := strings.TrimSpace(get("JIRA_URL"))
    jiraProject := strings.TrimSpace(get("JIRA_PROJECT"))
    jiraToken := strings.TrimSpace(get("JIRA_API_TOKEN"))

    if jiraURL == "" || jiraProject == "" || jiraToken == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Missing Jira configuration in runner secret (JIRA_URL, JIRA_PROJECT, JIRA_API_TOKEN required)"})
        return
    }

    // 6. Check if Jira link already exists for this path
    existingKey := ""
    for _, jl := range wf.JiraLinks {
        if strings.TrimSpace(jl.Path) == strings.TrimSpace(req.Path) {
            existingKey = jl.JiraKey
            break
        }
    }

    // 7. Determine auth header (Cloud vs Server)
    authHeader := ""
    if strings.Contains(jiraURL, "atlassian.net") {
        // Jira Cloud - assume token is email:api_token
        encoded := base64.StdEncoding.EncodeToString([]byte(jiraToken))
        authHeader = "Basic " + encoded
    } else {
        // Jira Server/Data Center
        authHeader = "Bearer " + jiraToken
    }

    // 8. Create or update Jira issue
    jiraBase := strings.TrimRight(jiraURL, "/")
    var httpReq *http.Request

    if existingKey == "" {
        // CREATE new issue
        jiraEndpoint := fmt.Sprintf("%s/rest/api/2/issue", jiraBase)

        fields := map[string]interface{}{
            "project":     map[string]string{"key": jiraProject},
            "summary":     title,
            "description": string(content),
            "issuetype":   map[string]string{"name": "Feature"},
        }

        // Add parent Outcome if specified
        if wf.ParentOutcome != nil && *wf.ParentOutcome != "" {
            fields["parent"] = map[string]string{"key": *wf.ParentOutcome}
        }

        reqBody := map[string]interface{}{"fields": fields}
        payload, _ := json.Marshal(reqBody)
        httpReq, _ = http.NewRequest("POST", jiraEndpoint, bytes.NewReader(payload))
    } else {
        // UPDATE existing issue
        jiraEndpoint := fmt.Sprintf("%s/rest/api/2/issue/%s", jiraBase, url.PathEscape(existingKey))
        reqBody := map[string]interface{}{
            "fields": map[string]interface{}{
                "summary":     title,
                "description": string(content),
            },
        }
        payload, _ := json.Marshal(reqBody)
        httpReq, _ = http.NewRequest("PUT", jiraEndpoint, bytes.NewReader(payload))
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", authHeader)

    httpClient := &http.Client{Timeout: 30 * time.Second}
    httpResp, httpErr := httpClient.Do(httpReq)
    if httpErr != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "Jira request failed", "details": httpErr.Error()})
        return
    }
    defer httpResp.Body.Close()

    respBody, _ := io.ReadAll(httpResp.Body)
    if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
        c.Data(httpResp.StatusCode, "application/json", respBody)
        return
    }

    // 9. Extract Jira key from response
    var outKey string
    if existingKey == "" {
        var created struct {
            Key string `json:"key"`
        }
        _ = json.Unmarshal(respBody, &created)
        if strings.TrimSpace(created.Key) == "" {
            c.JSON(http.StatusBadGateway, gin.H{"error": "Jira creation returned no key"})
            return
        }
        outKey = created.Key
    } else {
        outKey = existingKey
    }

    // 10. Update RFEWorkflow CR with Jira link
    obj := item.DeepCopy()
    spec, _ := obj.Object["spec"].(map[string]interface{})
    if spec == nil {
        spec = map[string]interface{}{}
        obj.Object["spec"] = spec
    }

    var links []interface{}
    if existing, ok := spec["jiraLinks"].([]interface{}); ok {
        links = existing
    }

    // Add or update link
    found := false
    for _, li := range links {
        if m, ok := li.(map[string]interface{}); ok {
            if fmt.Sprintf("%v", m["path"]) == req.Path {
                m["jiraKey"] = outKey
                found = true
                break
            }
        }
    }
    if !found {
        links = append(links, map[string]interface{}{"path": req.Path, "jiraKey": outKey})
    }
    spec["jiraLinks"] = links

    if _, err := reqDyn.Resource(gvrWf).Namespace(project).Update(c.Request.Context(), obj, v1.UpdateOptions{}); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update workflow with Jira link", "details": err.Error()})
        return
    }

    // 11. Return success
    c.JSON(http.StatusOK, gin.H{
        "key": outKey,
        "url": fmt.Sprintf("%s/browse/%s", jiraBase, outKey),
    })
}
```

**Note:** `getWorkflowJira` endpoint (line 3069) already works - it fetches existing Jira issues.

---

## Testing Plan

1. **Set up runner secrets:**
   ```bash
   kubectl create secret generic ambient-runner-secrets \
     --from-literal=JIRA_URL=https://issues.redhat.com \
     --from-literal=JIRA_PROJECT=RHASTRAT \
     --from-literal=JIRA_API_TOKEN=your_token \
     -n your-project
   ```

2. **Create RFE with optional Outcome:**
   ```bash
   POST /api/projects/my-project/rfe-workflows
   {
     "title": "Test Feature",
     "description": "...",
     "umbrellaRepo": {"url": "https://github.com/org/repo"},
     "parentOutcome": "RHASTRAT-456"  // Optional
   }
   ```

3. **Publish spec to Jira:**
   ```bash
   POST /api/projects/my-project/rfe-workflows/rfe-123/jira
   {
     "path": "specs/test-feature/spec.md"
   }
   ```

4. **Verify:**
   - Feature created in Jira project
   - Linked to Outcome (if provided)
   - `RFEWorkflow.jiraLinks` updated with key
   - Can fetch via `GET /api/projects/my-project/rfe-workflows/rfe-123/jira?path=specs/test-feature/spec.md`

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

## File Changes Summary

### Created:
- ✅ `types/common.go` - Common type definitions
- ✅ `types/session.go` - Session-related types
- ✅ `types/rfe.go` - RFE workflow types (with `ParentOutcome` field)
- ✅ `types/project.go` - Project types
- ✅ `jira.go` - Jira integration with GitHub file reading

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

## Next Steps: Code Organization Refactor

### Current State (2025-01-10)
- ✅ `handlers.go`: 3835 lines (needs breakdown)
- ✅ `main.go`: 497 lines (needs simplification to ~20-30 lines)
- ✅ All Jira integration complete
- ✅ Types package created (`types/common.go`, `types/session.go`, `types/rfe.go`, `types/project.go`)
- ✅ Build clean

### Refactor Goals

**Goal 1: Simplify main.go (Standard Go Practice)**
- Current: 497 lines with routing, initialization, helpers
- Target: ~20-30 lines (just initialize and run)
- Move logic to:
  - `server/server.go` - Server setup, routing, middleware
  - `server/k8s.go` - Kubernetes client initialization
  - Keep `main.go` minimal (industry standard)

**Goal 2: Break Down handlers.go**
- Current: 3835 lines (too large)
- Strategy: **One handler at a time, one commit each**
- Target structure:
  ```
  handlers/
  ├── middleware.go    - Auth, validation, K8s client helpers
  ├── sessions.go      - AgenticSession CRUD + lifecycle
  ├── projects.go      - Project CRUD
  ├── permissions.go   - RBAC + access keys
  ├── rfe.go           - RFEWorkflow CRUD + seeding
  ├── secrets.go       - Runner secrets management
  └── helpers.go       - Shared utility functions
  ```

### Incremental Refactor Strategy

**Phase 1: Simplify main.go**
1. Create `server/` package
2. Move server setup to `server/server.go`
3. Move K8s initialization to `server/k8s.go`
4. Reduce `main.go` to ~20-30 lines
5. Commit: "refactor: simplify main.go following Go best practices"

**Phase 2: Extract handlers.go (One Handler Per Commit)**

**Commit 1:** Health check handler
- Extract health endpoint
- Commit: "refactor: extract health handler"

**Commit 2:** Content service handlers
- Extract content service endpoints (contentWrite, contentRead, etc.)
- Create `handlers/content.go`
- Commit: "refactor: extract content handlers to handlers/content.go"

**Commit 3:** GitHub auth handlers
- Extract GitHub auth endpoints
- Create `handlers/github_auth.go`
- Commit: "refactor: extract GitHub auth handlers"

**Commit 4:** Project handlers
- Extract project CRUD endpoints
- Create `handlers/projects.go`
- Commit: "refactor: extract project handlers"

**Commit 5:** Permissions handlers
- Extract RBAC + access key endpoints
- Create `handlers/permissions.go`
- Commit: "refactor: extract permissions handlers"

**Commit 6:** Secrets handlers
- Extract runner secrets endpoints
- Create `handlers/secrets.go`
- Commit: "refactor: extract secrets handlers"

**Commit 7:** Session handlers
- Extract agentic session endpoints
- Create `handlers/sessions.go`
- Commit: "refactor: extract session handlers"

**Commit 8:** RFE workflow handlers
- Extract RFE workflow endpoints
- Create `handlers/rfe.go`
- Commit: "refactor: extract RFE workflow handlers"

**Commit 9:** Repo browsing handlers
- Extract repo tree/blob endpoints
- Create `handlers/repo.go`
- Commit: "refactor: extract repo browsing handlers"

**Commit 10:** WebSocket handlers
- Extract WebSocket endpoints
- Create `handlers/websocket.go` (or move to websocket_messaging.go)
- Commit: "refactor: extract WebSocket handlers"

**Commit 11:** Middleware and helpers
- Extract middleware functions
- Extract shared helpers
- Create `handlers/middleware.go` and `handlers/helpers.go`
- Commit: "refactor: extract middleware and helpers"

**Commit 12:** Remove old handlers.go
- Delete empty handlers.go
- Update imports in all files
- Commit: "refactor: remove old handlers.go after migration complete"

### Benefits
- ✅ Small, reviewable commits (easier to debug if issues arise)
- ✅ Git history shows exactly what moved where
- ✅ Can test after each commit
- ✅ Easy to revert individual changes if needed
- ✅ Clear separation of concerns
- ✅ Better code navigation and maintenance
- ✅ Follows Go community best practices

### Verification After Each Commit
```bash
go build -v  # Must compile
go test ./...  # Run tests if available
git add -A
git commit -m "refactor: <specific change>"
```

### Notes
- Branch: `jira-backend-refactor`
- All upstream/main logic preserved
- Build verified clean after each change
- Ready to start Phase 1
