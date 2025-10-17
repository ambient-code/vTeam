# Frontend Design Guidelines

## Table of Contents
1. [Component Architecture](#component-architecture)
2. [TypeScript & Type Safety](#typescript--type-safety)
3. [API Layer & Data Fetching](#api-layer--data-fetching)
4. [Next.js App Router Patterns](#nextjs-app-router-patterns)
5. [File Organization](#file-organization)
6. [UX Standards](#ux-standards)
7. [Component Composition](#component-composition)
8. [State Management](#state-management)

---

## Component Architecture

### Always Use Shadcn Components as Foundation

**Rule:** All UI components MUST be built on top of Shadcn components when possible.

**Why:** Shadcn provides:
- Accessible, WAI-ARIA compliant components
- Consistent design system
- Pre-built Radix UI primitives
- Full customization control

**Examples:**

```tsx
// ✅ GOOD: Extend Shadcn components
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';

type SuccessAlertProps = {
  message: string;
  onDismiss?: () => void;
};

export const SuccessAlert = ({ message, onDismiss }: SuccessAlertProps) => {
  return (
    <Alert variant="default" className="border-green-500 bg-green-50">
      <AlertDescription>{message}</AlertDescription>
      {onDismiss && (
        <Button variant="ghost" size="sm" onClick={onDismiss}>
          Dismiss
        </Button>
      )}
    </Alert>
  );
};

// ❌ BAD: Creating custom components from scratch
export const SuccessAlert = ({ message }: { message: string }) => {
  return (
    <div className="border rounded p-4 bg-green-50">
      <p>{message}</p>
    </div>
  );
};
```

### Component Variants & Customization

Derive customizations using the component's variant props or composition:

```tsx
// ✅ GOOD: Use variants
<Button variant="destructive">Delete</Button>
<Button variant="outline">Cancel</Button>
<Button variant="ghost">Close</Button>

// ✅ GOOD: Compose new variants
import { buttonVariants } from '@/components/ui/button';

const successButton = buttonVariants({
  variant: 'default',
  className: 'bg-green-600 hover:bg-green-700'
});
```

---

## TypeScript & Type Safety

### No `any` Types - Ever

**Rule:** The use of `any` is STRICTLY FORBIDDEN. Use proper types, `unknown`, or generic constraints.

```tsx
// ❌ BAD
const handleData = (data: any) => {
  console.log(data.name);
};

// ✅ GOOD: Use proper types
type UserData = {
  name: string;
  email: string;
};

const handleData = (data: UserData) => {
  console.log(data.name);
};

// ✅ GOOD: Use unknown for truly unknown data
const handleData = (data: unknown) => {
  if (isUserData(data)) {
    console.log(data.name);
  }
};

const isUserData = (data: unknown): data is UserData => {
  return (
    typeof data === 'object' &&
    data !== null &&
    'name' in data &&
    'email' in data
  );
};
```

### Define Shared Types

**Rule:** Create shared type definitions that match backend Go structs.

**Structure:**
```
src/types/
├── api/              # API request/response types
│   ├── projects.ts
│   ├── sessions.ts
│   ├── rfe.ts
│   └── common.ts
├── models/           # Domain models
│   ├── project.ts
│   ├── session.ts
│   └── user.ts
├── components/       # Component-specific types
│   └── forms.ts
└── index.ts          # Public exports
```

**Example:**

```tsx
// src/types/api/projects.ts
export type ProjectStatus = 'active' | 'archived' | 'pending';

export type Project = {
  name: string;
  displayName: string;
  description?: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  creationTimestamp: string;
  status: ProjectStatus;
};

export type CreateProjectRequest = {
  name: string;
  displayName: string;
  description?: string;
  labels?: Record<string, string>;
};

export type CreateProjectResponse = {
  project: Project;
};

// src/types/api/common.ts
export type ApiResponse<T> = {
  data: T;
  error?: never;
};

export type ApiError = {
  error: string;
  code?: string;
  details?: Record<string, unknown>;
};

export type ApiResult<T> = ApiResponse<T> | ApiError;
```

### Use `type` over `interface`

**Rule:** Prefer `type` declarations over `interface` (per user preference).

```tsx
// ✅ GOOD
type ButtonProps = {
  variant?: 'primary' | 'secondary';
  size?: 'sm' | 'md' | 'lg';
  disabled?: boolean;
  onClick?: () => void;
};

// ❌ AVOID
interface ButtonProps {
  variant?: 'primary' | 'secondary';
  // ...
}
```

---

## API Layer & Data Fetching

### Data Fetching Strategy

Our application uses a hybrid approach leveraging Next.js capabilities:

1. **Server Components (SSR/SSG)**: Use Next.js `fetch` API for initial data loading
2. **Client Components**: Use TanStack React Query for dynamic/interactive data
3. **Mutations**: Use Next.js Server Actions for POST/PUT/DELETE operations

### Next.js Fetch API (Server Components)

**Rule:** Use Next.js extended `fetch` API in Server Components for initial page data.

**Why:** Next.js `fetch` provides:
- Automatic request deduplication
- Built-in caching strategies
- Server-side rendering benefits
- No client-side JavaScript needed for initial load

**Caching Strategies:**

```tsx
// Force cache (default) - Cache indefinitely until revalidated
fetch(url, { cache: 'force-cache' });

// No store - Fresh data on every request
fetch(url, { cache: 'no-store' });

// Revalidate - Cache with time-based revalidation
fetch(url, { next: { revalidate: 3600 } }); // Revalidate every hour

// Tag-based revalidation - Cache with on-demand revalidation
fetch(url, { next: { tags: ['projects'] } });
```

**Example Server Component:**

```tsx
// app/projects/page.tsx (Server Component)
import type { Project } from '@/types/api/projects';
import { ProjectsList } from './components/projects-list';

async function getProjects(): Promise<Project[]> {
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/projects`, {
    next: { revalidate: 60, tags: ['projects'] }, // Revalidate every 60 seconds
  });

  if (!res.ok) {
    throw new Error('Failed to fetch projects');
  }

  const data = await res.json();
  return data.projects;
}

export default async function ProjectsPage() {
  const projects = await getProjects();

  return (
    <div>
      <h1>Projects</h1>
      <ProjectsList initialProjects={projects} />
    </div>
  );
}
```

**Error Handling:**

```tsx
// app/projects/page.tsx
import { notFound } from 'next/navigation';
import type { Project } from '@/types/api/projects';

async function getProject(name: string): Promise<Project | null> {
  const res = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/projects/${name}`,
    {
      next: { revalidate: 60, tags: ['projects', `project-${name}`] },
    }
  );

  if (res.status === 404) {
    return null;
  }

  if (!res.ok) {
    throw new Error('Failed to fetch project');
  }

  const data = await res.json();
  return data.project;
}

export default async function ProjectPage({ params }: { params: { name: string } }) {
  const project = await getProject(params.name);

  if (!project) {
    notFound(); // Renders not-found.tsx
  }

  return (
    <div>
      <h1>{project.displayName}</h1>
      {/* ... */}
    </div>
  );
}
```

**Parallel Data Fetching:**

```tsx
// app/projects/[name]/page.tsx
async function getProject(name: string) {
  const res = await fetch(`/api/projects/${name}`, {
    next: { tags: [`project-${name}`] },
  });
  if (!res.ok) throw new Error('Failed to fetch project');
  return res.json();
}

async function getSessions(projectName: string) {
  const res = await fetch(`/api/projects/${projectName}/sessions`, {
    next: { tags: [`project-${projectName}-sessions`] },
  });
  if (!res.ok) throw new Error('Failed to fetch sessions');
  return res.json();
}

async function getRfeWorkflows(projectName: string) {
  const res = await fetch(`/api/projects/${projectName}/rfe-workflows`, {
    next: { tags: [`project-${projectName}-rfe`] },
  });
  if (!res.ok) throw new Error('Failed to fetch RFE workflows');
  return res.json();
}

export default async function ProjectDashboard({ params }: { params: { name: string } }) {
  // Fetch all data in parallel
  const [projectData, sessionsData, rfeData] = await Promise.all([
    getProject(params.name),
    getSessions(params.name),
    getRfeWorkflows(params.name),
  ]);

  return (
    <div>
      <h1>{projectData.project.displayName}</h1>
      <SessionsList sessions={sessionsData.sessions} />
      <RfeList workflows={rfeData.workflows} />
    </div>
  );
}
```

### React Query for Mutations

**Rule:** Use React Query mutations for ALL data mutations (POST, PUT, DELETE operations).

**Why:** React Query provides:
- Automatic error handling and retry logic
- Optimistic updates
- Automatic cache invalidation
- TypeScript type safety
- Built-in loading and error states
- Better client-side state management

See the API Service Layer section above for implementation examples.

### Use TanStack React Query (Client Components)

**Rule:** Use React Query for dynamic, client-side data fetching in Client Components.

**When to use React Query:**
- Real-time data that needs frequent updates
- User-specific data
- Data that changes based on user interaction
- Polling or WebSocket fallback
- Optimistic updates
- Complex client-side caching needs

**Setup:**

```tsx
// src/lib/query-client.ts
import { QueryClient } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

// src/app/layout.tsx
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from '@/lib/query-client';

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html>
      <body>
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      </body>
    </html>
  );
}
```

### API Service Layer

**Rule:** Create a separate, reusable API service layer.

**Structure:**
```
src/services/
├── api/
│   ├── client.ts          # Base API client
│   ├── projects.ts        # Project endpoints
│   ├── sessions.ts        # Session endpoints
│   ├── rfe.ts            # RFE endpoints
│   └── auth.ts           # Auth endpoints
├── queries/
│   ├── use-projects.ts    # Project queries & mutations
│   ├── use-sessions.ts    # Session queries & mutations
│   └── use-rfe.ts        # RFE queries & mutations
└── index.ts
```

**Example:**

```tsx
// src/services/api/client.ts
import type { ApiError } from '@/types/api/common';

export class ApiClient {
  private baseUrl = '/api';

  async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      const error: ApiError = await response.json().catch(() => ({
        error: `HTTP ${response.status}: ${response.statusText}`,
      }));
      throw new ApiError(error.error, error.code);
    }

    return response.json();
  }

  get<T>(endpoint: string, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: 'GET' });
  }

  post<T>(endpoint: string, data?: unknown, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  put<T>(endpoint: string, data?: unknown, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  delete<T>(endpoint: string, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: 'DELETE' });
  }
}

export class ApiError extends Error {
  constructor(message: string, public code?: string) {
    super(message);
    this.name = 'ApiError';
  }
}

export const apiClient = new ApiClient();

// src/services/api/projects.ts
import { apiClient } from './client';
import type { Project, CreateProjectRequest, CreateProjectResponse } from '@/types/api/projects';

export const projectsApi = {
  list: () => apiClient.get<{ projects: Project[] }>('/projects'),
  
  get: (name: string) => 
    apiClient.get<{ project: Project }>(`/projects/${name}`),
  
  create: (data: CreateProjectRequest) => 
    apiClient.post<CreateProjectResponse>('/projects', data),
  
  delete: (name: string) => 
    apiClient.delete(`/projects/${name}`),
};

// src/services/queries/use-projects.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { projectsApi } from '@/services/api/projects';
import type { CreateProjectRequest } from '@/types/api/projects';

const projectKeys = {
  all: ['projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: (filters?: string) => [...projectKeys.lists(), filters] as const,
  details: () => [...projectKeys.all, 'detail'] as const,
  detail: (name: string) => [...projectKeys.details(), name] as const,
};

export const useProjects = () => {
  return useQuery({
    queryKey: projectKeys.lists(),
    queryFn: () => projectsApi.list(),
    select: (data) => data.projects,
  });
};

export const useProject = (name: string) => {
  return useQuery({
    queryKey: projectKeys.detail(name),
    queryFn: () => projectsApi.get(name),
    select: (data) => data.project,
    enabled: !!name,
  });
};

export const useCreateProject = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: CreateProjectRequest) => projectsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
};

export const useDeleteProject = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (name: string) => projectsApi.delete(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
};

// Usage in components
'use client';

import { useProjects, useCreateProject } from '@/services/queries/use-projects';
import { Button } from '@/components/ui/button';

export const ProjectsList = () => {
  const { data: projects, isLoading, error } = useProjects();
  const createProject = useCreateProject();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      {projects?.map((project) => (
        <div key={project.name}>{project.displayName}</div>
      ))}
      <Button
        onClick={() => createProject.mutate({ name: 'new-project', displayName: 'New Project' })}
        disabled={createProject.isPending}
      >
        {createProject.isPending ? 'Creating...' : 'Create Project'}
      </Button>
    </div>
  );
};
```

---

## Next.js App Router Patterns

### Use App Router Features

**Rule:** Leverage all Next.js App Router capabilities for better UX and code organization.

#### Required Files Per Route

Each route should have:
- `page.tsx` - Main page component
- `layout.tsx` - Shared layout (if needed)
- `loading.tsx` - Loading UI
- `error.tsx` - Error boundary
- `not-found.tsx` - 404 UI (for dynamic routes)

```
app/projects/[name]/
├── layout.tsx          # Shared layout with sidebar
├── page.tsx            # Project dashboard
├── loading.tsx         # Loading skeleton
├── error.tsx           # Error boundary
├── not-found.tsx       # Project not found
├── components/         # Page-specific components
│   ├── project-header.tsx
│   ├── stats-card.tsx
│   └── activity-feed.tsx
├── lib/               # Page-specific utilities
│   ├── utils.ts
│   └── constants.ts
└── hooks/             # Page-specific hooks
    └── use-project-data.ts
```

**Examples:**

```tsx
// app/projects/[name]/loading.tsx
import { Skeleton } from '@/components/ui/skeleton';

export default function Loading() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-12 w-full" />
      <Skeleton className="h-64 w-full" />
    </div>
  );
}

// app/projects/[name]/error.tsx
'use client';

import { useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';

type ErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function Error({ error, reset }: ErrorProps) {
  useEffect(() => {
    console.error('Project error:', error);
  }, [error]);

  return (
    <div className="flex items-center justify-center min-h-screen">
      <Alert variant="destructive" className="max-w-md">
        <AlertDescription>
          <h2 className="font-semibold mb-2">Something went wrong</h2>
          <p className="text-sm mb-4">{error.message}</p>
          <Button onClick={reset}>Try again</Button>
        </AlertDescription>
      </Alert>
    </div>
  );
}

// app/projects/[name]/not-found.tsx
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/empty-state';

export default function NotFound() {
  return (
    <EmptyState
      icon="folder-x"
      title="Project not found"
      description="The project you're looking for doesn't exist or you don't have access to it."
      action={
        <Button asChild>
          <Link href="/projects">View all projects</Link>
        </Button>
      }
    />
  );
}
```

---

## File Organization

### Component Colocation

**Rule:** Single-use components should be colocated with their page. Reusable components go in `src/components`.

```
✅ GOOD Structure:
src/
├── app/
│   └── projects/
│       └── [name]/
│           ├── sessions/
│           │   ├── [sessionName]/
│           │   │   ├── page.tsx
│           │   │   ├── loading.tsx
│           │   │   ├── components/
│           │   │   │   ├── session-header.tsx    # Only used here
│           │   │   │   └── message-list.tsx      # Only used here
│           │   │   └── hooks/
│           │   │       └── use-session-messages.ts
│           │   └── page.tsx
├── components/
│   ├── ui/                    # Shadcn components
│   ├── empty-state.tsx       # Reusable across app
│   ├── breadcrumbs.tsx       # Reusable across app
│   └── loading-button.tsx    # Reusable across app
├── hooks/
│   └── use-toast.tsx         # Reusable hook
└── lib/
    ├── utils.ts              # Shared utilities
    └── constants.ts          # Shared constants

❌ BAD Structure:
src/
├── components/
│   ├── session-header.tsx    # Only used in one page
│   ├── message-list.tsx      # Only used in one page
│   └── stats-card.tsx        # Only used in one page
└── app/
    └── projects/[name]/sessions/[sessionName]/page.tsx
```

### Extract Reusable Logic

**Rule:** Identify and extract reusable components and hooks.

```tsx
// ❌ BAD: Repeated logic in multiple components
const ComponentA = () => {
  const [isLoading, setIsLoading] = useState(false);
  
  const handleSubmit = async () => {
    setIsLoading(true);
    try {
      await fetch('/api/data');
    } finally {
      setIsLoading(false);
    }
  };
  
  return <Button disabled={isLoading}>Submit</Button>;
};

// ✅ GOOD: Extract into reusable hook
// src/hooks/use-async-action.ts
export const useAsyncAction = <T,>(
  action: () => Promise<T>
) => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const execute = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await action();
      return result;
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Unknown error');
      setError(error);
      throw error;
    } finally {
      setIsLoading(false);
    }
  };

  return { execute, isLoading, error };
};

// Usage
const ComponentA = () => {
  const { execute, isLoading } = useAsyncAction(() => fetch('/api/data'));
  return <Button disabled={isLoading} onClick={execute}>Submit</Button>;
};
```

---

## UX Standards

### Button States

**Rule:** ALL buttons MUST have consistent loading and disabled states.

```tsx
// ✅ GOOD: Consistent button with loading state
import { Button } from '@/components/ui/button';
import { Loader2 } from 'lucide-react';

type LoadingButtonProps = React.ComponentProps<typeof Button> & {
  isLoading?: boolean;
  loadingText?: string;
};

export const LoadingButton = ({
  isLoading,
  loadingText,
  children,
  disabled,
  ...props
}: LoadingButtonProps) => {
  return (
    <Button disabled={disabled || isLoading} {...props}>
      {isLoading ? (
        <>
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          {loadingText || children}
        </>
      ) : (
        children
      )}
    </Button>
  );
};

// Usage
const MyForm = () => {
  const mutation = useCreateProject();
  
  return (
    <LoadingButton
      isLoading={mutation.isPending}
      loadingText="Creating..."
      onClick={() => mutation.mutate(data)}
    >
      Create Project
    </LoadingButton>
  );
};
```

### Empty States

**Rule:** ALL lists and data displays MUST have proper empty states.

```tsx
// src/components/empty-state.tsx
import { LucideIcon } from 'lucide-react';
import * as Icons from 'lucide-react';

type EmptyStateProps = {
  icon?: keyof typeof Icons;
  title: string;
  description: string;
  action?: React.ReactNode;
};

export const EmptyState = ({
  icon = 'inbox',
  title,
  description,
  action,
}: EmptyStateProps) => {
  const Icon = Icons[icon] as LucideIcon;

  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <div className="rounded-full bg-muted p-3 mb-4">
        <Icon className="h-6 w-6 text-muted-foreground" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{title}</h3>
      <p className="text-sm text-muted-foreground mb-4 max-w-md">
        {description}
      </p>
      {action && <div>{action}</div>}
    </div>
  );
};

// Usage
const ProjectsList = () => {
  const { data: projects } = useProjects();

  if (!projects?.length) {
    return (
      <EmptyState
        icon="folder-open"
        title="No projects yet"
        description="Get started by creating your first project."
        action={
          <Button asChild>
            <Link href="/projects/new">Create Project</Link>
          </Button>
        }
      />
    );
  }

  return <div>{/* render projects */}</div>;
};
```

### Breadcrumbs

**Rule:** All nested pages MUST display breadcrumbs for navigation context.

```tsx
// src/components/breadcrumbs.tsx
import Link from 'next/link';
import { ChevronRight } from 'lucide-react';

type BreadcrumbItem = {
  label: string;
  href?: string;
};

type BreadcrumbsProps = {
  items: BreadcrumbItem[];
};

export const Breadcrumbs = ({ items }: BreadcrumbsProps) => {
  return (
    <nav aria-label="Breadcrumb" className="flex items-center space-x-2 text-sm text-muted-foreground">
      {items.map((item, index) => (
        <div key={index} className="flex items-center">
          {index > 0 && <ChevronRight className="h-4 w-4 mx-2" />}
          {item.href ? (
            <Link
              href={item.href}
              className="hover:text-foreground transition-colors"
            >
              {item.label}
            </Link>
          ) : (
            <span className="text-foreground font-medium">{item.label}</span>
          )}
        </div>
      ))}
    </nav>
  );
};

// Usage in page
const ProjectSessionPage = ({ params }: { params: { name: string; sessionName: string } }) => {
  return (
    <div>
      <Breadcrumbs
        items={[
          { label: 'Projects', href: '/projects' },
          { label: params.name, href: `/projects/${params.name}` },
          { label: 'Sessions', href: `/projects/${params.name}/sessions` },
          { label: params.sessionName },
        ]}
      />
      {/* rest of page */}
    </div>
  );
};
```

### Layout & Sidebar

**Rule:** Use consistent layouts with proper sidebar/content separation.

```tsx
// app/projects/[name]/layout.tsx
import { Sidebar } from './components/sidebar';
import { Breadcrumbs } from '@/components/breadcrumbs';

type LayoutProps = {
  children: React.ReactNode;
  params: { name: string };
};

export default function ProjectLayout({ children, params }: LayoutProps) {
  return (
    <div className="flex h-screen">
      <Sidebar projectName={params.name} />
      <div className="flex-1 flex flex-col">
        <header className="border-b p-4">
          <Breadcrumbs
            items={[
              { label: 'Projects', href: '/projects' },
              { label: params.name },
            ]}
          />
        </header>
        <main className="flex-1 overflow-auto p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
```

---

## Component Composition

### Break Down Large Components

**Rule:** Components over 200 lines MUST be broken down into smaller sub-components.

```tsx
// ❌ BAD: 600+ line component
export function SessionPage() {
  // 600 lines of mixed concerns
  return (
    <div>
      {/* header */}
      {/* tabs */}
      {/* messages */}
      {/* workspace */}
      {/* results */}
    </div>
  );
}

// ✅ GOOD: Broken into focused components
// app/projects/[name]/sessions/[sessionName]/page.tsx
export default function SessionPage({ params }: PageProps) {
  return (
    <div className="space-y-6">
      <SessionHeader sessionName={params.sessionName} />
      <SessionTabs sessionName={params.sessionName} />
    </div>
  );
}

// app/projects/[name]/sessions/[sessionName]/components/session-header.tsx
export function SessionHeader({ sessionName }: { sessionName: string }) {
  // 50 lines
}

// app/projects/[name]/sessions/[sessionName]/components/session-tabs.tsx
export function SessionTabs({ sessionName }: { sessionName: string }) {
  // 80 lines
}
```

---

## State Management

### Server State vs Client State

**Rule:** Use React Query for server state, React state for UI-only state.

```tsx
// ✅ GOOD: Clear separation
'use client';

import { useState } from 'react';
import { useProject } from '@/services/queries/use-projects';

export const ProjectPage = ({ params }: { params: { name: string } }) => {
  // Server state - managed by React Query
  const { data: project, isLoading } = useProject(params.name);
  
  // Client state - managed by React state
  const [selectedTab, setSelectedTab] = useState('overview');
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  // ...
};
```

---

## Summary Checklist

### Component Architecture
- [ ] All components use Shadcn as foundation
- [ ] Component variants derived from Shadcn base components
- [ ] No components over 200 lines

### TypeScript & Type Safety
- [ ] Zero `any` types in codebase
- [ ] Proper TypeScript types throughout
- [ ] Use `type` over `interface`
- [ ] Shared types match backend Go structs
- [ ] Type guards for runtime validation

### Data Fetching & API
- [ ] React Query for all data fetching (queries and mutations)
- [ ] API service layer separated from components
- [ ] Proper error handling in all data fetching
- [ ] Automatic cache invalidation with React Query

### Next.js App Router
- [ ] All routes have loading.tsx
- [ ] All routes have error.tsx
- [ ] Dynamic routes have not-found.tsx
- [ ] React Query hooks for all data operations

### File Organization
- [ ] Single-use components colocated with pages
- [ ] Reusable components in src/components
- [ ] Custom hooks extracted where appropriate
- [ ] Page-specific utilities in colocated lib/ folders

### UX Standards
- [ ] All buttons have loading states
- [ ] All lists have empty states
- [ ] Breadcrumbs on all nested pages
- [ ] Consistent layout with sidebar
- [ ] Proper loading skeletons
- [ ] User-friendly error messages
- [ ] Success feedback (toasts/alerts)

