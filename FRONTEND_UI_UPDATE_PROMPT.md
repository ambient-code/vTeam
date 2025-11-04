# Frontend UI Update Prompt for Claude 4.5 Sonnet

## Objective
Update the current vTeam Next.js/React frontend UI (located at `components/frontend`) to match the design and user experience of the static HTML prototype (located at `static-prototype`).

---

## Context

### Current Frontend
- **Location**: `/Users/abraren/acorn/local/code/vTeam/components/frontend`
- **Tech Stack**: Next.js 15, React, TypeScript, TailwindCSS, Shadcn/ui, React Query
- **Architecture**: App Router with Server/Client Components

### Static Prototype Reference
- **Location**: `/Users/abraren/acorn/local/code/vTeam/static-prototype`
- **Files**: HTML pages with inline CSS and vanilla JavaScript
- **Purpose**: Design reference and UX flow demonstration

### Design Guidelines
- **MUST READ**: `components/frontend/DESIGN_GUIDELINES.md`
- **MUST READ**: `components/frontend/COMPONENT_PATTERNS.md`

---

## Key Design Changes to Implement

### 1. **Branding & Header**
**Prototype Reference**: `static-prototype/index.html` (lines 10-54)

**Changes Needed**:
- Change branding from "vTeam" to **"Ambient Code Platform"**
- Update header logo/title styling to match prototype
- Implement user dropdown menu in header with:
  - User email display
  - User initials avatar bubble
  - Dropdown arrow animation
  - "Integrations" link
  - "Logout" button
- Use prototype's header styling: white background, subtle border-bottom

**Current Location**: `components/frontend/src/components/navigation.tsx`

---

### 2. **Terminology Changes**
**Prototype shows different terminology throughout:**

| Current Term | New Term (Prototype) |
|--------------|---------------------|
| "Projects" | "Workspaces" |
| "RFE Workflows" page | Integrated into workspace tabs |
| Page title on projects list | "Workspaces" |
| Create button | "New Workspace" |

**Affected Files**:
- `src/app/projects/page.tsx`
- `src/components/navigation.tsx`
- `src/components/page-header.tsx`
- All project-related page titles and descriptions

---

### 3. **Projects/Workspaces List Page**
**Prototype Reference**: `static-prototype/index.html`

**Changes Needed**:

#### Header Section
- Title: "Workspaces"
- Description: "Select a workspace or create a new one to get started"
- Actions: "Refresh" button + "New Workspace" button

#### Table Design
- **Columns**: Name, Description, Sessions, Created, Actions
- **Special column**: "Sessions" shows:
  - Total session count
  - Badge with running session count (if any) with spinner icon
  - Example: `4` + `<Badge>1 Running</Badge>`
- **Actions column**: Single delete button (trash icon)
- **Name column**: Link styled in blue (`#1e40af`)
- Clean white card with subtle borders

#### Modal for Creation
- **Modal Title**: "Create New Workspace"
- **Fields**:
  - Workspace Name (with validation message)
  - Display Name
  - Description (textarea)
- Pre-filled with auto-incrementing defaults (e.g., `demo-user-workspace-1`)
- Modal styling matches prototype's clean design

**Current Location**: `src/app/projects/page.tsx`

---

### 4. **Workspace Detail Page (Project Detail)**
**Prototype Reference**: `static-prototype/projects/sample-workspace/page.html`

**Major Redesign Needed**:

#### Layout
- **Left Sidebar Navigation** (vertical menu with icons):
  - Sessions (star icon) - Active by default
  - Sharing (users icon) - Previously "Permissions"
  - Workspace Settings (settings icon)
  
#### Breadcrumbs
- Format: `Workspaces / {workspace-name}`
- Located above page title

#### Page Header
- Show workspace name as title
- Show workspace description below title

#### Main Content Area - Tabbed Interface
The prototype shows different content sections that switch via sidebar navigation:

**A. Sessions Tab (Default)**
- Title: "Agentic Sessions"
- Actions: Refresh + "New Session" button
- Table columns:
  - Session Name (with description underneath)
  - Status (badge with icon: Running/Completed)
  - Mode (interactive/headless)
  - Model (e.g., claude-3.5-sonnet)
  - Created (relative time)
- Clickable rows that navigate to session detail
- Status badges with SVG icons (spinning loader for "Running", checkmark for "Completed")

**B. Sharing Tab** (Previously "Permissions")
- Title: "Sharing"
- Description: "Users and groups with access to this workspace and their roles"
- Table columns:
  - Users/Groups (name + subtitle)
  - Type (User/Group badge)
  - Role (Admin/View badge)
  - Actions (delete button)
- Action: "Grant Permission" button

**C. Workspace Settings Tab**
Multiple cards with different settings sections:

1. **General Settings Card**
   - Display Name (editable)
   - Workspace Name (read-only with help text)
   - Description (textarea)
   - "Save Changes" button

2. **Runner Secrets Card**
   - Runner Secret dropdown selector
   - Anthropic API Key (required field with asterisk)
   - Additional Secrets section:
     - Dynamic key/value pairs
     - "Add Secret" button
     - Grid layout: Key | Value | Delete button
   - "Save Secrets" button

3. **Resource Limits Card**
   - Max Concurrent Sessions
   - Max RFE Workspaces
   - Storage Limit (GB)
   - "Update Limits" button

4. **API Keys Card** (embedded in settings)
   - Table of API keys
   - Columns: Name, Created, Last Used, Role, Actions
   - Actions: Refresh + "Create Key" button

**Current Locations**:
- `src/app/projects/[name]/page.tsx`
- `src/app/projects/[name]/layout.tsx`
- `src/app/projects/[name]/permissions/page.tsx`
- `src/app/projects/[name]/settings/page.tsx`

---

### 5. **Session Creation Modal**
**Prototype Reference**: `static-prototype/projects/sample-workspace/page.html` (lines 698-776)

**Features to Implement**:

#### Basic Configuration
- **Model Dropdown**:
  - Claude Sonnet 3.7 (default)
  - Claude Opus 4.1
  - Claude Opus 4
  - Claude Sonnet 4
  - Claude Haiku 3.5

#### Headless-Specific Field
- **Agentic Prompt** textarea (only shown for headless mode)
- Help text: "Provide detailed instructions for the AI to execute without user interaction"

#### Advanced Settings (Collapsible Accordion)
- **Button**: "Change Default Model Settings" with chevron
- **Fields when expanded**:
  - Temperature (0.0 - 2.0, step 0.1)
  - Timeout (60-1800 seconds)
  - Max Output Tokens (100-8000)
  - **Bring Your Own Key section**:
    - Anthropic API Key input (password field)
    - "Save key for future sessions" checkbox
    - Help text: "Optional: Use your own Anthropic API key for this session"

#### Modal Actions
- Cancel button
- "Create Session" button

**Implementation**:
- Create new component: `src/components/session-config-dialog.tsx`
- Use Shadcn Dialog component
- Use Accordion component for advanced settings
- Integrate with session creation flow

---

### 6. **Sessions List Page**
**Prototype Reference**: `static-prototype/projects/sample-workspace/sessions/page.html`

**Design**:
- Breadcrumbs: `Workspaces / {workspace-name} / Sessions`
- Title: "Agentic Sessions"
- Description: "AI-powered coding sessions for your project"
- Left sidebar with workspace navigation
- Table with sessions (same as on workspace detail page)

**Current Location**: `src/app/projects/[name]/sessions/page.tsx`

---

### 7. **Visual Design System**

#### Colors (from `static-prototype/styles.css`)
```css
Primary Blue: #1e40af (links, primary buttons)
Hover Blue: #1d4ed8
Background: #f8fafc (page background)
Card Background: white
Borders: #e2e8f0
Text Primary: #333
Text Secondary: #64748b
Text Muted: #374151
Success Green: #166534 (background: #dcfce7)
Warning Yellow: #92400e (background: #fef3c7)
Error Red: #991b1b (background: #fee2e2)
Info Blue: #1e40af (background: #dbeafe)
```

#### Typography
- Font Family: `-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif`
- Page Title: `2rem`, `font-weight: bold`
- Card Title: `1.25rem`, `font-weight: 600`
- Body: `1rem`, `line-height: 1.6`

#### Spacing & Layout
- Container: `max-width: 1200px`, centered
- Card Border Radius: `0.5rem`
- Button Border Radius: `0.375rem`
- Standard Padding: Card header/content `1.5rem`

#### Buttons
- **Primary**: Blue background (`#1e40af`), white text
- **Secondary**: White background, gray border (`#d1d5db`), gray text
- **Icons**: Inline SVG, 14-16px size
- **Hover States**: Subtle background color changes

#### Tables
- Header Background: `#f8fafc`
- Border: `#e2e8f0`
- Row Hover: `#f8fafc`
- Cell Padding: `0.75rem`

#### Badges
- Border Radius: `9999px` (pill shape)
- Padding: `0.25rem 0.75rem`
- Font Size: `0.75rem`, `font-weight: 500`
- With inline SVG icons (12px)

#### Modals
- Backdrop: `rgba(0, 0, 0, 0.5)`
- Content: White, rounded corners, shadow
- Max Width: `500-600px`
- Close button: `×` character

---

## Implementation Guidelines

### Architecture Requirements

1. **Follow Existing Patterns**:
   - Use `type` over `interface` (per COMPONENT_PATTERNS.md)
   - No `any` types allowed
   - Use Shadcn/ui components as foundation
   - Maintain Server/Client Component separation
   - Use React Query for all data fetching

2. **File Organization**:
   - Keep single-use components colocated with pages
   - Reusable components go in `src/components/`
   - Follow existing directory structure

3. **Component Standards**:
   - Loading states: Use `loading.tsx` with Skeleton components
   - Error states: Use `error.tsx` with Error boundary
   - Empty states: Use `EmptyState` component
   - Forms: Use Shadcn Form components with validation

4. **Data Fetching**:
   - Use existing React Query hooks from `src/services/queries/`
   - Create new query hooks if needed following same pattern
   - Use API service layer from `src/services/api/`

---

## Step-by-Step Implementation Plan

### Phase 1: Global Changes
1. Update branding from "vTeam" to "Ambient Code Platform"
2. Update navigation component with user dropdown
3. Change "Projects" terminology to "Workspaces" everywhere
4. Update color scheme to match prototype

### Phase 2: Workspaces List Page
1. Update page title and description
2. Modify table columns to match prototype
3. Add "Sessions" column with running count
4. Update create workspace modal
5. Style improvements to match prototype

### Phase 3: Workspace Detail Page
1. Restructure layout with sidebar navigation
2. Implement tabbed interface (Sessions/Sharing/Settings)
3. Update breadcrumbs format
4. Move content to appropriate tabs

### Phase 4: Sessions Management
1. Create session configuration modal component
2. Add model selection dropdown
3. Implement advanced settings accordion
4. Add BYOK (Bring Your Own Key) section
5. Update session creation flow

### Phase 5: Settings & Permissions
1. Rename "Permissions" to "Sharing"
2. Restructure settings page with multiple cards
3. Add Runner Secrets configuration
4. Add Resource Limits section
5. Move API Keys to settings page

### Phase 6: Polish & Testing
1. Fine-tune spacing and colors
2. Verify all interactions work
3. Test responsive behavior
4. Ensure accessibility standards
5. Validate with design guidelines

---

## Critical Requirements

### DO:
✅ Use Shadcn/ui components exclusively
✅ Follow DESIGN_GUIDELINES.md strictly
✅ Maintain TypeScript strict typing (no `any`)
✅ Use React Query for data fetching
✅ Implement proper loading/error states
✅ Use Tailwind CSS classes
✅ Keep existing API integration working
✅ Preserve functionality while updating UI
✅ Use semantic HTML
✅ Ensure accessibility (ARIA labels, keyboard navigation)

### DON'T:
❌ Create custom UI components from scratch
❌ Use `any` types
❌ Break existing API integrations
❌ Remove error handling
❌ Ignore loading states
❌ Skip empty states
❌ Use inline styles (use Tailwind classes)
❌ Create new API endpoints (use existing backend)

---

## Testing Checklist

After implementation, verify:

- [ ] All pages render without errors
- [ ] Branding updated throughout
- [ ] Terminology consistent ("Workspaces" not "Projects")
- [ ] User dropdown works in header
- [ ] Workspace list shows correct columns
- [ ] Session count displays properly
- [ ] Create workspace modal functions
- [ ] Workspace detail page has sidebar navigation
- [ ] Settings page has all sections
- [ ] Sharing page displays correctly
- [ ] Session creation modal works
- [ ] All forms validate properly
- [ ] Delete confirmations work
- [ ] Loading states display
- [ ] Error states display
- [ ] Empty states display
- [ ] Responsive design works
- [ ] Colors match prototype
- [ ] Typography matches prototype
- [ ] Icons display correctly
- [ ] Badges styled correctly
- [ ] Tables styled correctly
- [ ] Buttons work and are styled correctly

---

## Reference Files

### Must Read Before Starting:
1. `/Users/abraren/acorn/local/code/vTeam/components/frontend/DESIGN_GUIDELINES.md`
2. `/Users/abraren/acorn/local/code/vTeam/components/frontend/COMPONENT_PATTERNS.md`
3. `/Users/abraren/acorn/local/code/vTeam/static-prototype/README.md`

### Key Prototype Files:
- `/Users/abraren/acorn/local/code/vTeam/static-prototype/index.html` - Workspaces list
- `/Users/abraren/acorn/local/code/vTeam/static-prototype/styles.css` - Design system
- `/Users/abraren/acorn/local/code/vTeam/static-prototype/projects/sample-workspace/page.html` - Workspace detail
- `/Users/abraren/acorn/local/code/vTeam/static-prototype/projects/sample-workspace/sessions/page.html` - Sessions list

### Current Frontend Structure:
```
components/frontend/src/
├── app/
│   ├── projects/
│   │   ├── page.tsx                    # Main workspaces list
│   │   ├── new/page.tsx               # Create workspace
│   │   └── [name]/
│   │       ├── page.tsx               # Workspace detail
│   │       ├── layout.tsx             # Workspace layout
│   │       ├── sessions/page.tsx      # Sessions list
│   │       ├── permissions/page.tsx   # Permissions (→ Sharing)
│   │       └── settings/page.tsx      # Settings
├── components/
│   ├── ui/                            # Shadcn components
│   ├── navigation.tsx                 # Header navigation
│   ├── page-header.tsx               # Page headers
│   ├── breadcrumbs.tsx               # Breadcrumbs
│   └── ...                           # Other components
├── services/
│   ├── api/                          # API client layer
│   └── queries/                      # React Query hooks
└── types/                            # TypeScript types
```

---

## Success Criteria

The implementation is complete when:

1. **Visual Consistency**: Frontend UI matches static prototype design
2. **Terminology**: All "Projects" references changed to "Workspaces"
3. **Branding**: "Ambient Code Platform" used throughout
4. **Functionality**: All existing features work with new UI
5. **Code Quality**: Follows all guidelines in DESIGN_GUIDELINES.md
6. **Type Safety**: No TypeScript errors, no `any` types
7. **User Experience**: Smooth interactions, proper loading/error states
8. **Responsive**: Works on desktop and mobile
9. **Accessible**: Keyboard navigation, ARIA labels, semantic HTML

---

## Questions to Consider

Before starting, review:
- What existing API endpoints are available? (Check `src/services/api/`)
- What data types are defined? (Check `src/types/`)
- What React Query hooks exist? (Check `src/services/queries/`)
- What Shadcn components are installed? (Check `src/components/ui/`)
- What's the current routing structure? (Check `src/app/`)

---

## Getting Started

1. **Read the design guidelines**: Start with DESIGN_GUIDELINES.md and COMPONENT_PATTERNS.md
2. **Explore the prototype**: Open static-prototype/index.html in a browser
3. **Review current frontend**: Familiarize yourself with the existing structure
4. **Plan your approach**: Break down work into small, testable changes
5. **Test incrementally**: Verify each change before moving to the next
6. **Commit frequently**: Make atomic commits for easy rollback

---

## Additional Resources

- **Shadcn/ui Docs**: https://ui.shadcn.com/
- **Next.js App Router**: https://nextjs.org/docs/app
- **React Query**: https://tanstack.com/query/latest
- **Tailwind CSS**: https://tailwindcss.com/docs

---

Good luck! Remember: **Preserve functionality while updating the UI** - this is a visual refresh, not a rewrite.

