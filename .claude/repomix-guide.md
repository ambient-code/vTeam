# Repomix Context Switching Guide

**Purpose:** Quick reference for loading the right repomix view based on the task.

## Available Views

The `repomix-analysis/` directory contains 7 pre-generated codebase views optimized for different scenarios:

| File | Size | Use When |
|------|------|----------|
| `01-full-context.xml` | 2.1MB | Deep dive into specific component implementation |
| `02-production-optimized.xml` | 4.2MB | General development work, most common use case |
| `03-architecture-only.xml` | 737KB | Understanding system design, new team member onboarding |
| `04-backend-focused.xml` | 403KB | Backend API work (Go handlers, K8s integration) |
| `05-frontend-focused.xml` | 767KB | UI development (NextJS, React Query, Shadcn) |
| `06-ultra-compressed.xml` | 10MB | Quick overview, exploring unfamiliar areas |
| `07-metadata-rich.xml` | 849KB | File structure analysis, refactoring planning |

## Usage Patterns

### Scenario 1: Backend Development

**Task:** Adding a new API endpoint for project settings

**Command:**

```
"Claude, reference the backend-focused repomix view (04-backend-focused.xml) and help me add a new endpoint for updating project settings."
```

**Why this view:**

- Contains all backend handlers and types
- Includes K8s client patterns
- Focused context without frontend noise

### Scenario 2: Frontend Development

**Task:** Creating a new UI component for RFE workflows

**Command:**

```
"Claude, load the frontend-focused repomix view (05-frontend-focused.xml) and help me create a new component for displaying RFE workflow steps."
```

**Why this view:**

- All React components and pages
- Shadcn UI patterns
- React Query hooks

### Scenario 3: Architecture Understanding

**Task:** Explaining the system to a new team member

**Command:**

```
"Claude, using the architecture-only repomix view (03-architecture-only.xml), explain how the operator watches for AgenticSession creation and spawns jobs."
```

**Why this view:**

- High-level component structure
- CRD definitions
- Component relationships
- No implementation details

### Scenario 4: Cross-Component Analysis

**Task:** Tracing a request from frontend through backend to operator

**Command:**

```
"Claude, use the production-optimized repomix view (02-production-optimized.xml) and trace the flow of creating an AgenticSession from UI click to Job creation."
```

**Why this view:**

- Balanced coverage of all components
- Includes key implementation files
- Not overwhelmed with test files

### Scenario 5: Quick Exploration

**Task:** Finding where a specific feature is implemented

**Command:**

```
"Claude, use the ultra-compressed repomix view (06-ultra-compressed.xml) to help me find where multi-repo support is implemented."
```

**Why this view:**

- Fast to process
- Good for keyword searches
- Covers entire codebase breadth

### Scenario 6: Refactoring Planning

**Task:** Planning to break up large handlers/sessions.go file

**Command:**

```
"Claude, analyze the metadata-rich repomix view (07-metadata-rich.xml) and suggest how to split handlers/sessions.go into smaller modules."
```

**Why this view:**

- File size and structure metadata
- Module boundaries
- Import relationships

### Scenario 7: Deep Implementation Dive

**Task:** Debugging a complex operator reconciliation issue

**Command:**

```
"Claude, load the full-context repomix view (01-full-context.xml) and help me understand why the operator is creating duplicate jobs for the same session."
```

**Why this view:**

- Complete implementation details
- All edge case handling
- Full operator logic

## Best Practices

### Start Broad, Then Narrow

1. **First pass:** Use `03-architecture-only.xml` to understand where the feature lives
2. **Second pass:** Use component-specific view (`04-backend` or `05-frontend`)
3. **Deep dive:** Use `01-full-context.xml` for specific implementation details

### Combine with Context Files

For even better results, combine repomix views with context files:

```
"Claude, load the backend-focused repomix view (04) and the backend-development context file, then help me add user token authentication to the new endpoint."
```

### Regenerate Periodically

Repomix views are snapshots in time. Regenerate monthly (or after major changes):

```bash
# Full regeneration
cd repomix-analysis
./regenerate-all.sh  # If you create this script

# Or manually
repomix --output 02-production-optimized.xml --config repomix-production.json
```

**Tip:** Add to monthly maintenance calendar.

## Quick Reference Table

| Task Type | Repomix View | Context File |
|-----------|--------------|--------------|
| Backend API work | 04-backend-focused | backend-development.md |
| Frontend UI work | 05-frontend-focused | frontend-development.md |
| Security review | 02-production-optimized | security-standards.md |
| Architecture overview | 03-architecture-only | - |
| Quick exploration | 06-ultra-compressed | - |
| Refactoring | 07-metadata-rich | - |
| Deep debugging | 01-full-context | (component-specific) |

## Maintenance

**When to regenerate:**

- After major architectural changes
- Monthly (scheduled)
- Before major refactoring efforts
- When views feel "stale" (>2 months old)

**How to regenerate:**
See `.repomixignore` for exclusion patterns. Adjust as needed to balance completeness with token efficiency.
