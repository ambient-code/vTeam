# vTeam Platform Feedback

Early feedback from the Ambient Code Platform team.

---

## IMPORTANT: Scope Constraint for AI Agents

**CRITICAL INSTRUCTION FOR CLAUDE CODE AND CURSOR:**

**ALL changes related to this feedback MUST be constrained to the static prototype ONLY.**

- **Working Directory**: `static-prototype/`
- **DO NOT modify**: `components/frontend/`, `components/backend/`, `components/operator/`, or any production code
- **Scope**: This feedback is for design exploration and prototyping only
- **Purpose**: Iterate on UX/UI concepts before implementing in production

When implementing feedback items:
1. All file paths mentioned refer to their equivalent locations in `static-prototype/`
2. Make changes to HTML/CSS/JS prototype files only
3. Do NOT create or modify TypeScript, Go, or Kubernetes resources
4. Focus on visual design, layout, and user flow changes
5. Document prototype changes in `static-prototype/CHANGELOG.md` or similar

**Example Path Mapping**:
- Feedback mentions: `components/frontend/src/app/projects/[name]/layout.tsx`
- Actually modify: `static-prototype/[equivalent-html-file].html`

If you need to make production changes based on this feedback, create a separate issue or task list and consult with the team first.

---

## Resources

**Video Walkthrough**: https://drive.google.com/file/d/18O98fAN1alz2XuZWBr9L9MrlXB5NxwIO/view?usp=sharing

**Meeting Notes (Gemini)**: [Chat UI Feedback - 2025_10_30 15_56 EDT - Notes by Gemini.md](./Chat%20UI%20Feedback%20-%202025_10_30%2015_56%20EDT%20-%20Notes%20by%20Gemini.md)

**Version 1 Prototype Feedback**: [my_notes.txt](./my_notes.txt)

**Session 2 Notes (Gemini)**: [Chat UI feedback session 2 - 2025_10_31 14_29 EDT - Notes by Gemini.md](./Chat%20UI%20feedback%20session%202%20-%202025_10_31%2014_29%20EDT%20-%20Notes%20by%20Gemini.md)

**Session 2 Transcript**: [Chat UI feedback session 2 - 2025_10_31 14_29 EDT - transcript.md](./Chat%20UI%20feedback%20session%202%20-%202025_10_31%2014_29%20EDT%20-%20transcript.md)

---

## Feedback Status Guide

- **Actionable**: Should be addressed
- **Ignore**: Explicitly marked to skip
- **Needs Clarification**: Requires more information

---

## Security & Permissions

### [Actionable] GitHub Integration Access Control (Gage)
**Issue**: Moving GitHub integration out of user's own access opens us up to situations where a user may now get to see repos they don't have actual access to.

**Component**: `components/backend/handlers/github.go`, `components/frontend/src/app/integrations/`

**Action Required**:
- Review OAuth scope implementation
- Ensure repository access respects user's GitHub permissions
- Add RBAC checks for repository visibility

---

## UX & Terminology

### [Actionable] Session vs Workflow Confusion (Gage)
**Issue**: Confused a bit on the role of a session and workflow - it looks like sessions ARE workflows.

**Component**: Documentation, Frontend UI labels

**Action Required**:
- Clarify terminology in documentation
- Update UI labels for consistency
- Consider renaming or better differentiation

### [Ignore] Workflow as Collection of Sessions (Gage)
**Status**: IGNORE - Design decision already made

**Note**: Today workflows are a collection of sessions in part so other people can collaborate on a single workflow.

### [Actionable] API Keys Navigation (Jeremy Eder / Version 1 Feedback)
**Issue**: We have API keys on the left sidebar and then fields for other API keys in the settings tab. I know they have different uses, but isn't it confusing?

**Additional Context**: Hide API keys from the project view for now until we have Jira integration.

**Component**: `components/frontend/src/app/projects/[name]/layout.tsx` (sidebar), Settings page

**Action Required**:
- Hide API keys from project view navigation until Jira integration is implemented
- When re-introduced, rename one section to be more specific (e.g., "API Keys" → "Service Keys" or "Integration Keys")
- Add clarifying descriptions to both sections
- Consider consolidating if they serve similar purposes

### [Actionable] Project Information Tab (Jeremy Eder / Version 1 Feedback)
**Issue**: Project information - either drop it or put something useful there. Maybe we create a metrics tab or something instead.

**Additional Context**: Move Project info to overview page.

**Component**: `components/frontend/src/app/projects/[name]/page.tsx`

**Action Required**:
- Move existing project information content to the overview page
- Evaluate current project info page content
- Option 1: Add useful metrics (session count, cost analytics, activity timeline)
- Option 2: Remove tab entirely if no valuable content (preferred after moving to overview)
- Recommended: Consolidate into overview page with metrics/analytics dashboard

### [Ignore] Chat Interface Complexity (Jeremy Eder)
**Status**: IGNORE - Too subjective, needs user testing first

**Note**: Chat interface - I'm torn. I think we are surfacing too much detail overall. There are too many choices.

### [Actionable] "Associated Repository" Terminology (Jeremy Eder)
**Issue**: What does "associated repository" mean? I need a clearer word used there.

**Component**: Frontend session creation forms, documentation

**Action Required**:
- Replace "Associated Repository" with clearer term
- Suggestions: "Source Repository", "Code Repository", "Git Repository", "Working Repository"
- Update all instances in UI and docs

### [Actionable] Remove "Headless Session" Terminology (Session 2 - Bill Murdock, Sally O'Malley, Gage Krumbach)
**Issue**: The term "headless session" is confusing for new users. They don't understand what it means or why they would choose it.

**Timestamp**: 00:17:52 - 00:18:45

**Component**: Session creation UI

**Action Required**:
- Remove "Headless Session" as a separate session type
- All sessions should be interactive by default
- Headless capabilities should be contained within workflows, not exposed as a session creation choice
- Note: When a headless session is restarted, it already becomes interactive, so this simplifies the UX

### [Actionable] Rename "RF.md" to "Idea.md" (Session 2 - Sally O'Malley, Bill Murdock)
**Issue**: The "RF" (Request for Feature) terminology is confusing and too prescriptive. Users don't understand what RF means.

**Timestamp**: 00:26:46

**Component**: Workflow artifacts, UI labels

**Action Required**:
- Rename `RF.md` to `Idea.md` throughout the UI
- Update workflow step descriptions to use "Idea" instead of "RF"
- Consider renaming the workflow from "Create RF" to "Create Specification" or similar
- Make the artifact format more generic so it can be used in various contexts (Jira, email, epic description, etc.)

### [Actionable] Add Spec Repo Education Tooltip (Session 2 - Sally O'Malley)
**Issue**: Users don't understand what a "spec repo" is or why they need one.

**Timestamp**: 00:22:38

**Component**: Session creation UI, spec repo configuration

**Action Required**:
- Add tooltip or popup with "What is a spec repo?" explanation
- Educate users on the spec repo concept before requiring them to set one up
- Consider inline help text in the form

### [Actionable] Move Integrations to User Menu (Session 2 - Gage Krumbach)
**Issue**: Integrations should be in the user dropdown menu to reinforce that it's user-scoped, not project-scoped.

**Timestamp**: 00:10:24

**Component**: Navigation structure

**Action Required**:
- Move "Integrations" from main navigation to user dropdown menu
- This reinforces that integrations are per-user settings, not per-project

### [Actionable] Improve Agent Selection UI (Session 2 - Dana Gutride, Gage Krumbach)
**Issue**: The agent selection UI looks like users must make a choice, but Claude auto-selects agents anyway.

**Timestamp**: 00:24:42

**Component**: Agent selection interface

**Action Required**:
- Hardcode "Auto-select agents" as the default behavior
- Add an info button explaining that Claude automatically picks agents based on context
- Don't make it look like a required choice
- Consider disabling the selection and just showing "Agents will be automatically selected based on your request"

### [Actionable] Live Markdown Rendering (Jeremy Eder)
**Issue**: Is it possible to render the markdown in the status pane live, as it's being generated?

**Component**: `components/frontend/src/app/projects/[name]/sessions/[sessionName]/page.tsx`

**Action Required**:
- Implement streaming markdown renderer
- Use WebSocket updates from backend
- Progressive rendering as content arrives
- Consider using React Markdown with streaming support

**References**:
- WebSocket implementation: `components/backend/websocket/websocket_messaging.go`
- Session status updates

### [Needs Clarification] Task Popup Suggestions (Jeremy Eder)
**Issue**: I like how the chat had popups. I wonder if we can keep raising popups of the next most likely task. These popups give us the opportunity to gain advantages from prompt templating like what SpecKit is doing in the background.

**Component**: New feature - AI-powered task suggestions

**Action Required**:
- Research SpecKit prompt templating approach
- Design popup/modal component for task suggestions
- Integrate with Claude API for next-task predictions
- Create prompt templates for common workflows

**Status**: Needs product/design review before implementation

---

## Navigation & Layout

### [Actionable] Remove Workflows from Project Page (Session 1)
**Issue**: Remove Workflows from the project page.

**Component**: `components/frontend/src/app/projects/[name]/layout.tsx`

**Action Required**:
- Remove "Workflows" from sidebar navigation items
- Verify no broken links remain
- Update any documentation referencing this section

**Implementation**:
```tsx
// File: components/frontend/src/app/projects/[name]/layout.tsx
// Remove this item from the items array:
// { href: `${base}/workflows`, label: "Workflows", icon: WorkflowIcon }
```

---

## Workflow Behavior & UX

### [Actionable] Workflow Step Dependencies (Version 1 Feedback)
**Issue**: Each step in a workflow depends on the successful completion of the step before it. For example, if the /spec command needs /ideate to have successfully run. A successful run of a step is determined by whether or not it generated an artifact. For example, if there is no rfe.md generated then /spec is disabled.

**Component**: Workflow step logic, RFE workflow implementation

**Action Required**:
- Implement sequential step dependency validation
- Add artifact existence checks before enabling subsequent steps
- Disable steps that depend on missing artifacts
- Provide clear visual feedback on why a step is disabled
- Example dependency chain:
  - `/ideate` must complete and generate `rfe.md`
  - `/spec` requires `rfe.md` to exist before it can run
  - `/plan` requires spec artifacts before it can run
- Display artifact status indicators for each step

### [Actionable] Workflow Status Message Updates (Version 1 Feedback)
**Issue**: The workflow descriptions in each step should change to status messages when the step is running. For example, when a user runs /ideate then 'Create or update rfe.md file' should become 'Generating rfe.md'.

**Component**: Workflow UI, step status display

**Action Required**:
- Implement dynamic status message updates during execution
- Replace static descriptions with active status messages when running
- Examples:
  - Before: "Create or update rfe.md file"
  - During: "Generating rfe.md..."
  - After: "Generated rfe.md" (with timestamp)
- Use progressive verb forms during execution ("Generating", "Creating", "Analyzing")
- Add visual indicators (spinner, progress bar) alongside status messages
- Persist completion messages after step finishes

### [Actionable] Separate RFE and Specification Workflows (Session 2 - Dana Gutride, Gage Krumbach)
**Issue**: The current workflow conflates RFE (for PMs) and Specification (for engineers). These should be distinct workflows for different user types.

**Timestamp**: 00:27:42 - 00:31:23

**Component**: Workflow design, user flows

**Action Required**:
- **PM Workflow**: Focus on iterating on ideas/RFEs
  - PMs should primarily work on the "Idea" (formerly RF.md)
  - After PM completes idea, they trigger a "Generate Specification" workflow
  - PM workflow should be a guided journey with prompts and questions, not manual commands
- **Engineer Workflow**: Starts from a completed specification
  - Engineers receive the spec.md output from the PM workflow
  - Engineers can then run spec kit commands themselves if needed
  - Engineers are "power users" who can run specific commands
- Create clear handoff point between PM idea generation and engineering specification
- Don't teach PMs spec kit - they just iterate on ideas
- Engineers need to understand spec kit for refinement

**User Types Identified**:
1. **Power Users** (Engineers): Run specific commands, need full control
2. **Workflow Runners** (PMs): Taken through guided journey, don't need command-level access

### [Actionable] Bot-Guided Repo Setup (Session 2 - Bill Murdock)
**Issue**: The form-based spec repo setup is cumbersome. The chatbot should guide users through this process.

**Timestamp**: 00:21:43

**Component**: Spec repo configuration, chat interface

**Action Required**:
- Move spec repo setup into chat-based flow
- Let the chatbot ask for GitHub username and personal access token through conversation
- Bot can intelligently guide the user: "I need access to your GitHub repos. Please provide your GitHub username and a personal access token"
- Bot can be smarter about validation (e.g., not requiring `.git` suffix)
- Reduce friction compared to traditional form fields

---

## Workflow Design Philosophy

### [Actionable] Generic Workflow Platform Approach (Session 2 - Andy Braren, Gage Krumbach, Bill Murdock)
**Issue**: The platform is being designed too specifically around RFE creation. It should be a generic workflow platform.

**Timestamp**: 00:37:06 - 00:39:05

**Component**: Overall platform design philosophy

**Action Required**:
- Stop positioning this as an "RFE builder"
- Position as a **generic workflow platform** where teams can define and run whatever workflows they want
- The "Create RFE" workflow is just the first example/use case
- Teams should be able to:
  - Define their own workflow steps
  - Customize artifact names and formats
  - Use different integrations (not just Jira)
  - Adapt the platform to their team's processes
- Ensure UI and documentation reflect this generic, flexible approach
- Consider creating a workflow editor/builder for future phases

**Design Principle**:
> "We're making a generic platform to help users and help internal teams run whatever workflows they desire, augmented by AI. The RF workflow is just the first stab at that."

---

## Open Questions & Discussion Items

### [Needs Clarification] File Editing in UI (Version 1 Feedback)
**Question**: Do we need file editing in the Ambient Code UI for MVP?

**Context**: Determining scope for MVP release

**Discussion Points**:
- What types of files would users need to edit?
- Can users edit artifacts (rfe.md, spec.md, etc.) directly in the UI?
- Is read-only view sufficient for MVP?
- Should editing be delegated to external editors/IDEs?
- What's the expected workflow: view → edit externally → re-upload?

**Status**: Needs product/design review

### [Needs Clarification] Artifact Storage Architecture (Version 1 Feedback)
**Question**: Can we abstract the idea of a spec repo for artifacts and config, or can we have a default storage area on the container we are running in?

**Context**: Determining artifact storage strategy

**Discussion Points**:
- Current approach uses separate "spec repo" for artifacts
- Alternative: Default storage on job container filesystem
- Pros of spec repo: Version control, collaboration, persistence
- Pros of container storage: Simpler setup, no git dependency
- Hybrid approach: Default to container, optional git sync
- Impact on artifact sharing across sessions
- Impact on artifact versioning and history

**Status**: Needs architecture review

### [Needs Clarification] Collaboration Features (Version 1 Feedback)
**Question**: How should collaboration work in workflows?

**Context**: Multi-user workflow participation

**Discussion Points**:
- Can multiple users work on the same workflow simultaneously?
- How are conflicts resolved when multiple users edit artifacts?
- Do we need real-time collaboration (like Google Docs)?
- Is async collaboration (like GitHub PRs) sufficient?
- How do permissions work for shared workflows?
- Should workflows be owned by projects or individual users?

**Status**: Needs product/design review

### [Needs Clarification] Artifact Iteration Through Chat (Session 2 - Bill Murdock, Sally O'Malley)
**Question**: Can users iterate on generated artifacts through chat instead of manual file editing?

**Timestamp**: 00:41:09 - 00:43:43

**Context**: User workflow preference

**Resolution**: This already works! Users can iterate on artifacts (RF.md, spec.md) through the chat interface while the session is alive.

**Action**:
- Ensure this capability is prominently documented
- Make it clear in the UI that artifacts can be refined through conversation
- Example: After RF.md is generated, user can say "Remove the stuff about WAGG 2.1, I don't think that's appropriate"

**Status**: Feature exists, needs documentation/visibility

### [Future Enhancement] Artifact Viewer in Right Sidebar (Session 2 - Andy Braren)
**Issue**: It would be helpful to preview generated artifacts without leaving the interface.

**Timestamp**: 00:42:08

**Context**: Emerging pattern in Claude web interface, Cursor, and other AI tools

**Proposal**:
- Add artifact browser/viewer in right-hand sidebar
- After generating artifacts, users can click on one to see preview
- Could show live PR, code files, or markdown preview
- Follows pattern emerging in modern AI interfaces

**Priority**: Future enhancement, not MVP

**Status**: Document for future consideration

### [Future Enhancement] Auto-Create Spec Repo (Session 2 - Bill Murdock)
**Issue**: Requiring users to manually create a spec repo adds friction.

**Timestamp**: 00:23:38

**Proposal**:
- If user doesn't already have a spec repo, the system could create one for them
- Currently: User must create empty GitHub repo, then seed it
- Future: System creates the repo and seeds it in one step

**Priority**: Nice to have, not MVP critical

**Status**: Document for future consideration

### [Future Enhancement] Workspace/Team Hierarchy (Session 2 - Andy Braren)
**Issue**: Need another organizational level above projects for PM teams to collectively manage workflows.

**Timestamp**: 00:52:59 - 00:54:04

**Context**: PM teams need shared workflow definitions

**Proposal**:
- Add "Workspace" or "Team" level above projects
- PM team could collectively own and edit the "Create RFE" workflow definition
- Workflows defined at workspace level are available to all projects in that workspace
- Enables team-wide workflow management

**Priority**: Phase 2 feature

**Status**: Document for future architecture planning

---

## Summary of Actionable Items

**High Priority** (Session 2 additions marked with 🆕):
1. Remove Workflows from project page navigation (Version 1)
2. Hide API Keys from project view until Jira integration (Version 1)
3. Move Project Info to overview page (Version 1)
4. Implement workflow step dependencies based on artifact generation (Version 1)
5. Add dynamic status messages for workflow steps (Version 1)
6. 🆕 Remove "Headless Session" option - all sessions are interactive (Session 2)
7. 🆕 Rename "RF.md" to "Idea.md" throughout UI (Session 2)
8. 🆕 Add spec repo education tooltip/popup (Session 2)
9. 🆕 Move Integrations to user menu (Session 2)
10. 🆕 Improve agent selection UI - show auto-select as default (Session 2)
11. 🆕 Separate RFE and Specification workflows for different user types (Session 2)
12. GitHub integration access control review

**Medium Priority**:
13. 🆕 Bot-guided repo setup through chat (Session 2)
14. 🆕 Generic workflow platform positioning (Session 2)
15. Clarify "Associated Repository" terminology
16. Session vs Workflow terminology clarity
17. Project Information tab content evaluation (after moving to overview)
18. Live markdown rendering implementation

**Low Priority / Needs Discussion**:
19. Task popup suggestions (requires design review)
20. File editing in UI for MVP (requires product review)
21. Artifact storage architecture (requires architecture review)
22. Collaboration features design (requires product review)

**Future Enhancements** (Document but defer):
23. 🆕 Artifact viewer in right sidebar (Session 2)
24. 🆕 Auto-create spec repo (Session 2)
25. 🆕 Workspace/Team hierarchy above projects (Session 2)

---

## Notes for AI Agents

When implementing changes:
- Follow vTeam frontend standards in `components/frontend/DESIGN_GUIDELINES.md`
- Use Shadcn UI components only
- Ensure TypeScript strict mode compliance (no `any` types)
- Add loading and error states for all new features
- Update tests and documentation
- Follow the backend standards in `CLAUDE.md` for Go changes
