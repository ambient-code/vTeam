# React Query Usage Patterns

Standard patterns for data fetching, mutations, and cache management in the frontend.

## Core Principles

1. **ALL data fetching uses React Query** - No manual `fetch()` in components
2. **Queries for reads** - `useQuery` for GET operations
3. **Mutations for writes** - `useMutation` for POST/PUT/DELETE
4. **Cache invalidation** - Invalidate queries after mutations
5. **Optimistic updates** - Update UI before server confirms

## File Structure

```
src/services/
├── api/                    # API client layer (pure functions)
│   ├── sessions.ts         # sessionApi.list(), .create(), .delete()
│   ├── projects.ts
│   └── common.ts           # Shared fetch logic, error handling
└── queries/                # React Query hooks
    ├── sessions.ts         # useSessions(), useCreateSession()
    ├── projects.ts
    └── common.ts           # Query client config
```

**Separation of concerns:**

- `api/`: Pure API functions (no React, no hooks)
- `queries/`: React Query hooks that use API functions

## Pattern 1: Query Hook (List Resources)

```typescript
// services/queries/sessions.ts
import { useQuery } from "@tanstack/react-query"
import { sessionApi } from "@/services/api/sessions"

export function useSessions(projectName: string) {
  return useQuery({
    queryKey: ["sessions", projectName],
    queryFn: () => sessionApi.list(projectName),
    staleTime: 5000,          // Consider data fresh for 5s
    refetchInterval: 10000,   // Poll every 10s for updates
  })
}
```

**Usage in component:**

```typescript
// app/projects/[projectName]/sessions/page.tsx
'use client'

import { useSessions } from "@/services/queries/sessions"

export function SessionsList({ projectName }: { projectName: string }) {
  const { data: sessions, isLoading, error } = useSessions(projectName)

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>
  if (!sessions?.length) return <div>No sessions found</div>

  return (
    <div>
      {sessions.map(session => (
        <SessionCard key={session.metadata.name} session={session} />
      ))}
    </div>
  )
}
```

**Key points:**

- `queryKey` includes all parameters that affect the query
- `staleTime` prevents unnecessary refetches
- `refetchInterval` for polling (optional)
- Destructure `data`, `isLoading`, `error` for UI states

## Pattern 2: Query Hook (Single Resource)

```typescript
// services/queries/sessions.ts
export function useSession(projectName: string, sessionName: string) {
  return useQuery({
    queryKey: ["sessions", projectName, sessionName],
    queryFn: () => sessionApi.get(projectName, sessionName),
    enabled: !!sessionName,  // Only run if sessionName provided
    staleTime: 3000,
  })
}
```

**Key points:**

- `enabled: !!sessionName` prevents query if parameter missing
- More specific queryKey for targeted cache invalidation

## Pattern 3: Create Mutation with Optimistic Update

```typescript
// services/queries/sessions.ts
import { useMutation, useQueryClient } from "@tanstack/react-query"

export function useCreateSession(projectName: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateSessionRequest) =>
      sessionApi.create(projectName, data),

    // Optimistic update: show immediately before server confirms
    onMutate: async (newSession) => {
      // Cancel any outgoing refetches (prevent overwriting optimistic update)
      await queryClient.cancelQueries({
        queryKey: ["sessions", projectName]
      })

      // Snapshot current value
      const previousSessions = queryClient.getQueryData([
        "sessions",
        projectName
      ])

      // Optimistically update cache
      queryClient.setQueryData(
        ["sessions", projectName],
        (old: AgenticSession[] | undefined) => [
          ...(old || []),
          {
            metadata: { name: newSession.name },
            spec: newSession,
            status: { phase: "Pending" },  // Optimistic status
          },
        ]
      )

      // Return context with snapshot
      return { previousSessions }
    },

    // Rollback on error
    onError: (err, variables, context) => {
      queryClient.setQueryData(
        ["sessions", projectName],
        context?.previousSessions
      )

      // Show error toast/notification
      console.error("Failed to create session:", err)
    },

    // Refetch after success (get real data from server)
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["sessions", projectName]
      })
    },
  })
}
```

**Usage:**

```typescript
// components/sessions/create-session-dialog.tsx
'use client'

import { useCreateSession } from "@/services/queries/sessions"
import { Button } from "@/components/ui/button"

export function CreateSessionDialog({ projectName }: { projectName: string }) {
  const createSession = useCreateSession(projectName)

  const handleSubmit = (data: CreateSessionRequest) => {
    createSession.mutate(data)
  }

  return (
    <form onSubmit={handleSubmit}>
      {/* form fields */}
      <Button
        type="submit"
        disabled={createSession.isPending}
      >
        {createSession.isPending ? "Creating..." : "Create Session"}
      </Button>
    </form>
  )
}
```

**Key points:**

- `onMutate`: Optimistic update (runs before server call)
- `onError`: Rollback on failure
- `onSuccess`: Invalidate queries to refetch real data
- Use `isPending` for loading states

## Pattern 4: Delete Mutation

```typescript
// services/queries/sessions.ts
export function useDeleteSession(projectName: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (sessionName: string) =>
      sessionApi.delete(projectName, sessionName),

    // Optimistic delete
    onMutate: async (sessionName) => {
      await queryClient.cancelQueries({
        queryKey: ["sessions", projectName]
      })

      const previousSessions = queryClient.getQueryData([
        "sessions",
        projectName
      ])

      // Remove from cache
      queryClient.setQueryData(
        ["sessions", projectName],
        (old: AgenticSession[] | undefined) =>
          old?.filter(s => s.metadata.name !== sessionName) || []
      )

      return { previousSessions }
    },

    onError: (err, sessionName, context) => {
      queryClient.setQueryData(
        ["sessions", projectName],
        context?.previousSessions
      )
    },

    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["sessions", projectName]
      })
    },
  })
}
```

## Pattern 5: Polling Until Condition Met

```typescript
// services/queries/sessions.ts
export function useSessionWithPolling(
  projectName: string,
  sessionName: string
) {
  return useQuery({
    queryKey: ["sessions", projectName, sessionName],
    queryFn: () => sessionApi.get(projectName, sessionName),
    refetchInterval: (query) => {
      const session = query.state.data

      // Stop polling if completed or error
      if (session?.status.phase === "Completed" ||
          session?.status.phase === "Error") {
        return false  // Stop polling
      }

      return 3000  // Poll every 3s while running
    },
  })
}
```

**Key points:**

- Dynamic `refetchInterval` based on query data
- Return `false` to stop polling
- Return number (ms) to continue polling

## API Client Layer Pattern

```typescript
// services/api/sessions.ts
import { API_BASE_URL } from "@/config"
import type { AgenticSession, CreateSessionRequest } from "@/types/session"

async function fetchWithAuth(url: string, options: RequestInit = {}) {
  const token = getAuthToken()  // From auth context or storage

  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${token}`,
      ...options.headers,
    },
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.message || "Request failed")
  }

  return response.json()
}

export const sessionApi = {
  list: async (projectName: string): Promise<AgenticSession[]> => {
    const data = await fetchWithAuth(
      `${API_BASE_URL}/projects/${projectName}/agentic-sessions`
    )
    return data.items || []
  },

  get: async (
    projectName: string,
    sessionName: string
  ): Promise<AgenticSession> => {
    return fetchWithAuth(
      `${API_BASE_URL}/projects/${projectName}/agentic-sessions/${sessionName}`
    )
  },

  create: async (
    projectName: string,
    data: CreateSessionRequest
  ): Promise<AgenticSession> => {
    return fetchWithAuth(
      `${API_BASE_URL}/projects/${projectName}/agentic-sessions`,
      {
        method: "POST",
        body: JSON.stringify(data),
      }
    )
  },

  delete: async (projectName: string, sessionName: string): Promise<void> => {
    return fetchWithAuth(
      `${API_BASE_URL}/projects/${projectName}/agentic-sessions/${sessionName}`,
      {
        method: "DELETE",
      }
    )
  },
}
```

**Key points:**

- Shared `fetchWithAuth` for token injection
- Pure functions (no React, no hooks)
- Type-safe inputs and outputs
- Centralized error handling

## Anti-Patterns (DO NOT USE)

### ❌ Manual fetch() in Components

```typescript
// NEVER DO THIS
const [sessions, setSessions] = useState([])

useEffect(() => {
  fetch('/api/sessions')
    .then(r => r.json())
    .then(setSessions)
}, [])
```

**Why wrong:** No caching, no automatic refetching, manual state management.
**Use instead:** React Query hooks.

### ❌ Not Using Query Keys Properly

```typescript
// BAD: Same query key for different data
useQuery({
  queryKey: ["sessions"],  // Missing projectName!
  queryFn: () => sessionApi.list(projectName),
})
```

**Why wrong:** Cache collisions, wrong data shown.
**Use instead:** Include all parameters in query key.

## Quick Reference

| Pattern | Hook | When to Use |
|---------|------|-------------|
| List resources | `useQuery` | GET /resources |
| Get single resource | `useQuery` | GET /resources/:id |
| Create resource | `useMutation` | POST /resources |
| Update resource | `useMutation` | PUT /resources/:id |
| Delete resource | `useMutation` | DELETE /resources/:id |
| Polling | `useQuery` + `refetchInterval` | Real-time updates |
| Optimistic update | `onMutate` | Instant UI feedback |
| Dependent query | `enabled` | Query depends on another |

## Validation Checklist

Before merging frontend code:

- [ ] All data fetching uses React Query (no manual fetch)
- [ ] Query keys include all relevant parameters
- [ ] Mutations invalidate related queries
- [ ] Loading and error states handled
- [ ] Optimistic updates for create/delete (where appropriate)
- [ ] API client layer is pure functions (no hooks)
