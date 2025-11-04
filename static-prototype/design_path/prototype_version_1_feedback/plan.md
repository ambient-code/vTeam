# Static Prototype Update Plan

This plan outlines concrete changes to the static prototype HTML files to address actionable feedback from curated_feedback.md.

**Last Updated**: After Session 2 feedback integration

**New Updates from Session 2**: Updates 8-13 added below

## Files to Modify

### Core Project Pages
- `static-prototype/projects/sample-project/page.html` - Main project page with sidebar navigation
- `static-prototype/projects/sample-project/info/page.html` - Project information page
- `static-prototype/projects/sample-project/rfe/page.html` - RFE Workspaces page (workflow page)
- `static-prototype/projects/sample-project/sessions/page.html` - Sessions list page
- `static-prototype/projects/sample-project/keys/page.html` - API Keys page
- `static-prototype/projects/sample-project/permissions/page.html` - Permissions page
- `static-prototype/projects/sample-project/settings/page.html` - Settings page

### Individual RFE/Workflow Pages (examples)
- `static-prototype/projects/sample-project/create-api-tests/page.html`
- `static-prototype/projects/sample-project/implement-login-flow/page.html`
- Other workflow instance pages

---

## Update 1: Remove Workflows from Sidebar Navigation

**Priority**: High

**Files**: All project pages (page.html, sessions/page.html, rfe/page.html, keys/page.html, permissions/page.html, settings/page.html, info/page.html)

**Changes**:
1. In each file's sidebar (`<aside class="project-sidebar">`), remove the "Workflows" navigation item
2. Remove this block:
   ```html
   <li><a href="../rfe/page.html">
       <svg>...</svg>
       Workflows
   </a></li>
   ```
3. Update active states to ensure remaining nav items work correctly

**Note**: The actual RFE workspace pages can remain for now, just remove from navigation.

---

## Update 2: Hide API Keys from Sidebar Navigation

**Priority**: High

**Files**: All project pages (page.html, sessions/page.html, rfe/page.html, keys/page.html, permissions/page.html, settings/page.html, info/page.html)

**Changes**:
1. In each file's sidebar, comment out or remove the "API Keys" navigation item
2. Remove this block:
   ```html
   <li><a href="../keys/page.html">
       <svg>...</svg>
       API Keys
   </a></li>
   ```
3. Add HTML comment indicating it will return when Jira integration is implemented

**Note**: The keys/page.html file itself can remain, just hidden from navigation.

---

## Update 3: Move Project Info to Overview Page

**Priority**: High

**Files**:
- `static-prototype/projects/sample-project/page.html` (overview page)
- `static-prototype/projects/sample-project/info/page.html` (project info page)

**Changes**:

### In page.html (overview):
1. Add a new section in the main content area
2. Copy relevant project information content from info/page.html
3. Add project metadata display:
   - Project name
   - Description
   - Created date
   - Owner
   - Any other relevant project details
4. Consider adding a stats/metrics section (placeholder for future):
   - Total sessions
   - Active workflows
   - Recent activity

### In info/page.html:
1. Add deprecation notice at top of page
2. Redirect users to overview page
3. Or simply add content indicating this has moved to Overview

---

## Update 4: Remove Project Information from Sidebar Navigation

**Priority**: High

**Files**: All project pages (rfe/page.html, sessions/page.html, keys/page.html, permissions/page.html, settings/page.html)

**Changes**:
1. Remove "Project Information" navigation item from sidebar
2. Remove this block:
   ```html
   <li><a href="../info/page.html">
       <svg>...</svg>
       Project Information
   </a></li>
   ```

**Note**: After Update 3, this navigation item is redundant since info moved to overview.

---

## Update 5: Add Workflow Step Dependencies (Visual Only)

**Priority**: High

**Files**: Individual workflow pages (create-api-tests/page.html, implement-login-flow/page.html, etc.)

**Changes**:

1. For each workflow step in the step list, add visual indicators:
   - **Completed steps**: Green checkmark icon, normal text
   - **Current available step**: Blue highlight, enabled button
   - **Locked steps** (missing dependencies): Gray out, add lock icon, disable button

2. Add tooltip or help text explaining why step is locked:
   ```html
   <div class="step-item locked" title="Requires rfe.md from previous step">
       <div class="step-icon">🔒</div>
       <div class="step-content">
           <h4>/spec - Create Specification</h4>
           <p class="step-status">Waiting for /ideate to complete</p>
       </div>
       <button class="btn btn-secondary" disabled>Run</button>
   </div>
   ```

3. Add artifact status badges next to each step:
   ```html
   <div class="artifact-status">
       <span class="badge badge-success">✓ rfe.md</span>
   </div>
   ```

4. Example dependency chain to implement:
   - `/ideate` → must generate `rfe.md`
   - `/spec` → requires `rfe.md`, generates `spec.md`
   - `/plan` → requires `spec.md`, generates `plan.md`
   - `/tasks` → requires `plan.md`, generates `tasks.md`

---

## Update 6: Add Dynamic Status Messages to Workflow Steps

**Priority**: High

**Files**: Individual workflow pages (create-api-tests/page.html, implement-login-flow/page.html, etc.)

**Changes**:

1. Add multiple status message states for each step (use CSS classes to toggle visibility):

   ```html
   <div class="step-item" data-step-status="pending">
       <div class="step-content">
           <h4>/ideate - Ideation</h4>

           <!-- Pending state -->
           <p class="step-description" data-status="pending">
               Create or update rfe.md file
           </p>

           <!-- Running state -->
           <p class="step-description" data-status="running" style="display: none;">
               <span class="spinner"></span> Generating rfe.md...
           </p>

           <!-- Completed state -->
           <p class="step-description" data-status="completed" style="display: none;">
               ✓ Generated rfe.md <span class="timestamp">2 minutes ago</span>
           </p>
       </div>
   </div>
   ```

2. Add CSS for status indicators:
   - Pending: Normal gray text
   - Running: Blue text with spinner animation
   - Completed: Green text with checkmark and timestamp

3. Add example workflow states to demonstrate:
   - Create one workflow page showing pending state
   - Create one workflow page showing running state (with "Generating..." messages)
   - Create one workflow page showing completed state (with checkmarks and timestamps)

4. Use progressive verb forms for running states:
   - "Generating rfe.md..."
   - "Creating specification..."
   - "Analyzing requirements..."
   - "Building task list..."

---

## Update 7: Consolidated Sidebar Navigation Order

**Priority**: Medium

**After all changes, the final sidebar navigation should be:**

```html
<aside class="project-sidebar">
    <ul>
        <li><a href="../sessions/page.html">Sessions</a></li>
        <li><a href="../page.html" class="active">Overview</a></li>
        <!-- Workflows - REMOVED -->
        <!-- API Keys - REMOVED (until Jira integration) -->
        <!-- Project Information - REMOVED (moved to Overview) -->
        <li><a href="../permissions/page.html">Sharing</a></li>
        <li><a href="../settings/page.html">Settings</a></li>
    </ul>
</aside>
```

**Clean and simple**: Sessions, Overview, Sharing, Settings

---

## Update 8: Remove "Headless Session" Option (Session 2)

**Priority**: High

**Source**: Session 2 feedback (00:17:52 - 00:18:45)

**Files**:
- Any session creation pages/forms
- New session modal/dialog components

**Changes**:

1. Remove the "Headless Session" vs "Interactive Session" choice from session creation UI
2. All sessions should be created as interactive by default
3. Remove any toggle, radio buttons, or dropdown that lets users choose session type
4. Update UI text to simply say "New Session" instead of "New Interactive Session"

**Example - Before**:
```html
<select name="session-type">
    <option value="interactive">Interactive Session</option>
    <option value="headless">Headless Session</option>
</select>
```

**Example - After**:
```html
<!-- Session type selection removed entirely -->
<h3>Create New Session</h3>
<p>Start an interactive session with Claude</p>
```

**Rationale**: Users don't understand what "headless" means. Headless capabilities should be contained within workflows, not exposed as a session creation choice.

---

## Update 9: Rename "RF.md" to "Idea.md" (Session 2)

**Priority**: High

**Source**: Session 2 feedback (00:26:46)

**Files**: Individual workflow pages showing artifacts

**Changes**:

1. Replace all instances of "RF.md" with "Idea.md" in workflow UIs
2. Update workflow step titles:
   - Old: "Create RF" → New: "Create Idea" or "Ideation"
   - Old: "Update RF.md" → New: "Update Idea.md"
3. Update artifact listings and file previews
4. Update any workflow descriptions that mention "RF"

**Example Changes**:
```html
<!-- Before -->
<h4>/ideate - Create RF</h4>
<p>Create or update RF.md file</p>
<div class="artifact">
    <a href="#">RF.md</a>
</div>

<!-- After -->
<h4>/ideate - Ideation</h4>
<p>Create or update Idea.md file</p>
<div class="artifact">
    <a href="#">Idea.md</a>
</div>
```

**Rationale**: "RF" (Request for Feature) is organizational jargon that's confusing. "Idea" is more generic and can be used in various contexts (Jira cards, emails, epics, etc.).

---

## Update 10: Add Spec Repo Education Tooltip (Session 2)

**Priority**: High

**Source**: Session 2 feedback (00:22:38)

**Files**: Any pages with spec repo configuration (session creation, settings)

**Changes**:

1. Add an info icon (ⓘ) or help icon next to "Spec Repository" label
2. On hover or click, show tooltip/popup explaining what a spec repo is
3. Add inline help text under the spec repo input field

**Example Implementation**:
```html
<label for="spec-repo">
    Spec Repository
    <button class="info-icon" onclick="showSpecRepoHelp()">
        <svg><!-- info icon --></svg>
    </button>
</label>

<div id="spec-repo-help" class="tooltip" style="display: none;">
    <h4>What is a Spec Repository?</h4>
    <p>A spec repository is where workflow artifacts (ideas, specifications, plans) are stored.
    It's a GitHub repository that serves as version-controlled storage for all your workflow outputs.</p>
    <p>You can use an existing empty repository or create a new one specifically for this purpose.</p>
</div>

<input type="text" id="spec-repo" placeholder="username/my-spec-repo">
<small class="help-text">The GitHub repository where artifacts will be stored</small>
```

**CSS Addition**:
```css
.info-icon {
    background: none;
    border: none;
    color: #6b7280;
    cursor: pointer;
    padding: 0;
    margin-left: 0.25rem;
}

.info-icon:hover {
    color: #374151;
}

.tooltip {
    position: absolute;
    background: white;
    border: 1px solid #e5e7eb;
    border-radius: 0.375rem;
    padding: 1rem;
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
    max-width: 300px;
    z-index: 1000;
}
```

---

## Update 11: Move Integrations to User Menu (Session 2)

**Priority**: High

**Source**: Session 2 feedback (00:10:24)

**Files**:
- `static-prototype/projects/sample-project/page.html` (and all other pages with main navigation)
- User dropdown menu component

**Changes**:

1. Remove "Integrations" from main top navigation bar
2. Add "Integrations" to user dropdown menu (next to logout)

**Example - Before**:
```html
<nav class="nav">
    <a href="../../index.html">Projects</a>
    <a href="../../integrations/page.html">Integrations</a>
</nav>
```

**Example - After**:
```html
<!-- Main nav - Integrations removed -->
<nav class="nav">
    <a href="../../index.html">Projects</a>
</nav>

<!-- User dropdown menu - Integrations added -->
<div class="user-dropdown-menu">
    <div class="menu-section">
        <a href="../../integrations/page.html">
            <svg><!-- settings icon --></svg>
            Integrations
        </a>
    </div>
    <div class="menu-section">
        <button onclick="logout()">
            <svg><!-- logout icon --></svg>
            Logout
        </button>
    </div>
</div>
```

**Rationale**: Integrations are user-scoped settings (GitHub tokens, etc.), not project-scoped. Placing them in the user menu reinforces this.

---

## Update 12: Improve Agent Selection UI (Session 2)

**Priority**: High

**Source**: Session 2 feedback (00:24:42)

**Files**: Session pages with agent selection interface

**Changes**:

1. Change agent selection from appearing like a required choice to showing it's automatic
2. Add info button explaining auto-selection behavior
3. Disable the selection UI and show default message

**Example - Before**:
```html
<label>Select Agents</label>
<select name="agents" multiple>
    <option value="">-- Select agents --</option>
    <option value="pm">Product Manager</option>
    <option value="architect">Architect</option>
    <option value="engineer">Staff Engineer</option>
</select>
```

**Example - After**:
```html
<div class="agent-selection-info">
    <label>
        Agent Selection
        <button class="info-icon" onclick="showAgentInfo()">
            <svg><!-- info icon --></svg>
        </button>
    </label>
    <div class="auto-select-message">
        <svg class="check-icon"><!-- checkmark --></svg>
        <span>Agents automatically selected based on your request</span>
    </div>
    <div id="agent-info-tooltip" class="tooltip" style="display: none;">
        <p>Claude automatically chooses the most appropriate agents based on your prompt and workflow context.
        You don't need to manually select them.</p>
    </div>
</div>
```

**CSS Addition**:
```css
.agent-selection-info {
    margin: 1rem 0;
}

.auto-select-message {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem;
    background: #f0fdf4;
    border: 1px solid #86efac;
    border-radius: 0.375rem;
    color: #166534;
    font-size: 0.875rem;
}

.check-icon {
    width: 16px;
    height: 16px;
    color: #16a34a;
}
```

---

## Update 13: Rename Workflow to Reflect Generic Platform (Session 2)

**Priority**: Medium

**Source**: Session 2 feedback (00:37:06 - 00:39:05)

**Files**: All pages that reference workflows

**Changes**:

1. Update page titles and descriptions to reflect generic workflow capability
2. Avoid overly prescriptive language like "RFE Builder"
3. Show workflows as examples, not the only use case

**Example - Before**:
```html
<h1>RFE Workflow Builder</h1>
<p>Create and manage Request for Enhancement workflows</p>
```

**Example - After**:
```html
<h1>Workflow Management</h1>
<p>Create and run custom AI-powered workflows tailored to your team's processes</p>

<div class="workflow-examples">
    <h3>Example Workflows</h3>
    <ul>
        <li>Product Idea Refinement (PM workflow)</li>
        <li>Specification Generation (Engineering workflow)</li>
        <li>Architecture Review</li>
        <li>Custom workflows defined by your team</li>
    </ul>
</div>
```

**Key Messaging Changes**:
- "RFE Builder" → "Workflow Platform"
- "Create RFE" → "Create Idea" or "Product Refinement Workflow"
- Emphasize customization and team-specific workflows
- Show RFE as one example among many possible workflows

---

## CSS Additions Needed

Add to `static-prototype/styles.css`:

```css
/* Locked step styling */
.step-item.locked {
    opacity: 0.5;
    cursor: not-allowed;
}

.step-item.locked button {
    opacity: 0.5;
    pointer-events: none;
}

/* Status message states */
.step-description[data-status="running"] {
    color: #3b82f6;
    font-weight: 500;
}

.step-description[data-status="completed"] {
    color: #10b981;
}

/* Spinner animation */
.spinner {
    display: inline-block;
    width: 12px;
    height: 12px;
    border: 2px solid #3b82f6;
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
}

@keyframes spin {
    to { transform: rotate(360deg); }
}

/* Artifact status badges */
.artifact-status {
    margin-top: 0.5rem;
}

.badge {
    display: inline-block;
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    border-radius: 0.25rem;
    font-weight: 500;
}

.badge-success {
    background-color: #d1fae5;
    color: #065f46;
}

.badge-warning {
    background-color: #fef3c7;
    color: #92400e;
}

/* Timestamp styling */
.timestamp {
    color: #6b7280;
    font-size: 0.75rem;
    font-weight: 400;
}
```

---

## Testing Checklist

After making changes:

1. **Navigation**:
   - [ ] Verify "Workflows" removed from all sidebars
   - [ ] Verify "API Keys" removed from all sidebars
   - [ ] Verify "Project Information" removed from all sidebars
   - [ ] Verify remaining nav items work and highlight correctly

2. **Overview Page**:
   - [ ] Project information visible on overview page
   - [ ] Project metadata displays correctly
   - [ ] Layout is clean and organized

3. **Workflow Steps**:
   - [ ] Locked steps show lock icon and are disabled
   - [ ] Status messages change appropriately (pending → running → completed)
   - [ ] Artifact badges display correctly
   - [ ] Dependency chain is visually clear

4. **Visual Consistency**:
   - [ ] All pages use consistent sidebar navigation
   - [ ] All styling is consistent across pages
   - [ ] No broken links remain

---

## Out of Scope for This Plan

The following feedback items are NOT included because they require further discussion, backend implementation, or are unclear:

### Requires Backend/Infrastructure Changes:
- **Live markdown rendering** - Requires WebSocket implementation (not feasible in static prototype)
- **Bot-guided repo setup** - Requires actual chat AI integration (Session 2 feedback)
- **GitHub integration access control** - Backend security concern, not prototype change
- **Artifact storage architecture** - Needs architecture review
- **Separate RFE and Specification workflows** - Requires workflow engine changes (Session 2 feedback)

### Needs Product/Design Decision:
- **Task popup suggestions** - Needs product/design review first
- **File editing in UI** - Open question for MVP scope
- **Collaboration features** - Needs product/design review
- **Session vs Workflow terminology** - Needs terminology decision first
- **"Associated Repository" renaming** - Needs terminology decision on exact wording

### Future Enhancements (Documented but Deferred):
- **Artifact viewer in right sidebar** - Phase 2 feature (Session 2)
- **Auto-create spec repo** - Nice to have (Session 2)
- **Workspace/Team hierarchy** - Phase 2 organizational feature (Session 2)
- **Artifact iteration through chat** - Feature already exists, needs documentation (Session 2)

These items should be addressed separately after decisions are made or in future phases.

---

## Implementation Order

### Phase 1 - Navigation Cleanup (High Priority)
**Updates**: 1, 2, 4, 7, 11
- Remove "Workflows" from sidebar
- Remove "API Keys" from sidebar
- Remove "Project Information" from sidebar
- Move "Integrations" to user menu
- Final consolidated navigation structure

**Estimated Time**: 30 minutes

### Phase 2 - Content Consolidation (High Priority)
**Updates**: 3
- Move project info to overview page

**Estimated Time**: 45 minutes

### Phase 3 - Session 2 UX Improvements (High Priority)
**Updates**: 8, 9, 10, 12
- Remove "Headless Session" option
- Rename "RF.md" to "Idea.md"
- Add spec repo education tooltips
- Improve agent selection UI

**Estimated Time**: 1 hour

### Phase 4 - Workflow Enhancements (High Priority)
**Updates**: 5, 6
- Add workflow step dependencies (visual)
- Add dynamic status messages

**Estimated Time**: 1.5 hours

### Phase 5 - Messaging & Polish (Medium Priority)
**Updates**: 13, CSS
- Update to generic workflow platform messaging
- Add all CSS for new components
- Final testing and cleanup

**Estimated Time**: 1 hour

**Total Estimated Effort**: 4.5-5 hours for a developer familiar with the prototype structure.

---

## Updated Testing Checklist

After making changes:

### 1. Navigation (Updates 1, 2, 4, 7, 11)
- [ ] "Workflows" removed from all project sidebars
- [ ] "API Keys" removed from all project sidebars
- [ ] "Project Information" removed from all sidebars
- [ ] "Integrations" moved to user dropdown menu
- [ ] Remaining nav items work and highlight correctly
- [ ] Final sidebar shows: Sessions, Overview, Sharing, Settings

### 2. Overview Page (Update 3)
- [ ] Project information visible on overview page
- [ ] Project metadata displays correctly
- [ ] Layout is clean and organized

### 3. Session Creation (Update 8)
- [ ] "Headless Session" option removed
- [ ] Session creation just says "New Session"
- [ ] No confusing session type terminology

### 4. Workflow Artifacts (Update 9)
- [ ] All "RF.md" references changed to "Idea.md"
- [ ] Workflow step titles updated ("Ideation" not "Create RF")
- [ ] Artifact listings show "Idea.md"

### 5. Educational Elements (Update 10, 12)
- [ ] Spec repo has info tooltip explaining what it is
- [ ] Agent selection shows "auto-select" message
- [ ] Info buttons work and display helpful content
- [ ] Tooltips are positioned correctly

### 6. Workflow Steps (Updates 5, 6)
- [ ] Locked steps show lock icon and are disabled
- [ ] Status messages change: pending → running → completed
- [ ] Artifact badges display correctly
- [ ] Dependency chain is visually clear
- [ ] Spinner animations work for running states
- [ ] Timestamps show on completed states

### 7. Generic Platform Messaging (Update 13)
- [ ] No "RFE Builder" language remains
- [ ] Workflows shown as examples, not prescriptive
- [ ] Generic platform positioning is clear

### 8. Visual Consistency
- [ ] All pages use consistent sidebar navigation
- [ ] All styling is consistent across pages
- [ ] New tooltips and info icons match design system
- [ ] No broken links remain
- [ ] All CSS additions work correctly
