# Frontend Development Context

**When to load:** Working on NextJS application, UI components, or React Query integration

## Quick Reference

- **Framework:** Next.js 14 (App Router)
- **UI Library:** Shadcn UI (built on Radix UI primitives)
- **Styling:** Tailwind CSS
- **Data Fetching:** TanStack React Query
- **Primary Directory:** `components/frontend/src/`

## Critical Rules (Zero Tolerance)

### 1. Zero `any` Types

**FORBIDDEN:**

```typescript
// ❌ BAD
function processData(data: any) { ... }
```

**REQUIRED:**

```typescript
// ✅ GOOD - use proper types
function processData(data: AgenticSession) { ... }

// ✅ GOOD - use unknown if type truly unknown
function processData(data: unknown) {
  if (isAgenticSession(data)) { ... }
}
```

### 2. Shadcn UI Components Only

**FORBIDDEN:** Creating custom UI components from scratch for buttons, inputs, dialogs, etc.

**REQUIRED:** Use `@/components/ui/*` components

```typescript
// ❌ BAD
<button className="px-4 py-2 bg-blue-500">Click</button>

// ✅ GOOD
import { Button } from "@/components/ui/button"
<Button>Click</Button>
```

**Available Shadcn components:** button, card, dialog, form, input, select, table, toast, etc.
**Check:** `components/frontend/src/components/ui/` for full list

### 3. React Query for ALL Data Operations

**FORBIDDEN:** Manual `fetch()` calls in components

**REQUIRED:** Use hooks from `@/services/queries/*`

```typescript
// ❌ BAD
const [sessions, setSessions] = useState([])
useEffect(() => {
  fetch('/api/sessions').then(r => r.json()).then(setSessions)
}, [])

// ✅ GOOD
import { useSessions } from "@/services/queries/sessions"
const { data: sessions, isLoading } = useSessions(projectName)
```

### 4. Use `type` Over `interface`

**REQUIRED:** Always prefer `type` for type definitions

```typescript
// ❌ AVOID
interface User { name: string }

// ✅ PREFERRED
type User = { name: string }
```

### 5. Colocate Single-Use Components

**FORBIDDEN:** Creating components in shared directories if only used once

**REQUIRED:** Keep page-specific components with their pages

```
app/
  projects/
    [projectName]/
      sessions/
        _components/        # Components only used in sessions pages
          session-card.tsx
        page.tsx           # Uses session-card
```

## Common Patterns

### Page Structure

```typescript
// app/projects/[projectName]/sessions/page.tsx
import { useSessions } from "@/services/queries/sessions"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"

export default function SessionsPage({
  params,
}: {
  params: { projectName: string }
}) {
  const { data: sessions, isLoading, error } = useSessions(params.projectName)

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>
  if (!sessions?.length) return <div>No sessions found</div>

  return (
    <div>
      {sessions.map(session => (
        <Card key={session.metadata.name}>
          {/* ... */}
        </Card>
      ))}
    </div>
  )
}
```

### React Query Hook Pattern

```typescript
// services/queries/sessions.ts
import { useQuery, useMutation } from "@tanstack/react-query"
import { sessionApi } from "@/services/api/sessions"

export function useSessions(projectName: string) {
  return useQuery({
    queryKey: ["sessions", projectName],
    queryFn: () => sessionApi.list(projectName),
  })
}

export function useCreateSession(projectName: string) {
  return useMutation({
    mutationFn: (data: CreateSessionRequest) =>
      sessionApi.create(projectName, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sessions", projectName] })
    },
  })
}
```

## Pre-Commit Checklist

- [ ] Zero `any` types (or justified with eslint-disable)
- [ ] All UI uses Shadcn components
- [ ] All data operations use React Query
- [ ] Components under 200 lines
- [ ] Single-use components colocated
- [ ] All buttons have loading states
- [ ] All lists have empty states
- [ ] All nested pages have breadcrumbs
- [ ] `npm run build` passes with 0 errors, 0 warnings
- [ ] All types use `type` instead of `interface`

## Key Files

- `components/frontend/DESIGN_GUIDELINES.md` - Comprehensive patterns
- `components/frontend/COMPONENT_PATTERNS.md` - Architecture patterns
- `src/components/ui/` - Shadcn UI components
- `src/services/queries/` - React Query hooks
- `src/services/api/` - API client layer

## Recent Issues & Learnings

- **2024-11-18:** Migrated all data fetching to React Query - no more manual fetch calls
- **2024-11-15:** Enforced Shadcn UI only - removed custom button components
- **2024-11-10:** Added breadcrumb pattern for nested pages
