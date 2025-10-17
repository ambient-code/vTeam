# Component Patterns & Architecture Guide

This guide documents the component patterns and architectural decisions made during the frontend modernization.

## File Organization

```
src/
├── app/                    # Next.js 15 App Router
│   ├── projects/
│   │   ├── page.tsx       # Route component
│   │   ├── loading.tsx    # Loading state
│   │   ├── error.tsx      # Error boundary
│   │   └── [name]/        # Dynamic routes
├── components/            # Reusable components
│   ├── ui/               # Shadcn base components
│   ├── layouts/          # Layout components
│   └── *.tsx             # Custom components
├── services/             # API layer
│   ├── api/             # HTTP clients
│   └── queries/         # React Query hooks
├── hooks/               # Custom hooks
├── types/               # TypeScript types
└── lib/                 # Utilities
```

## Naming Conventions

- **Files**: kebab-case (e.g., `empty-state.tsx`)
- **Components**: PascalCase (e.g., `EmptyState`)
- **Hooks**: camelCase with `use` prefix (e.g., `useAsyncAction`)
- **Types**: PascalCase (e.g., `ProjectSummary`)

## Component Patterns

### 1. Type Over Interface

**Guideline**: Always use `type` instead of `interface`

```typescript
// ✅ Good
type ButtonProps = {
  label: string;
  onClick: () => void;
};

// ❌ Bad
interface ButtonProps {
  label: string;
  onClick: () => void;
}
```

### 2. Component Props

**Pattern**: Destructure props with typed parameters

```typescript
type EmptyStateProps = {
  icon?: React.ComponentType<{ className?: string }>;
  title: string;
  description?: string;
  action?: React.ReactNode;
};

export function EmptyState({
  icon: Icon,
  title,
  description,
  action
}: EmptyStateProps) {
  // Implementation
}
```

### 3. Children Props

**Pattern**: Use `React.ReactNode` for children

```typescript
type PageContainerProps = {
  children: React.ReactNode;
  maxWidth?: 'sm' | 'md' | 'lg';
};
```

### 4. Loading States

**Pattern**: Use skeleton components, not spinners

```typescript
// ✅ Good - loading.tsx
import { TableSkeleton } from '@/components/skeletons';

export default function SessionsLoading() {
  return <TableSkeleton rows={10} columns={5} />;
}

// ❌ Bad - inline spinner
if (loading) return <Spinner />;
```

### 5. Error Handling

**Pattern**: Use error boundaries, not inline error states

```typescript
// ✅ Good - error.tsx
'use client';

export default function SessionsError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Failed to load sessions</CardTitle>
        <CardDescription>{error.message}</CardDescription>
      </CardHeader>
      <CardContent>
        <Button onClick={reset}>Try again</Button>
      </CardContent>
    </Card>
  );
}
```

### 6. Empty States

**Pattern**: Use EmptyState component consistently

```typescript
{sessions.length === 0 ? (
  <EmptyState
    icon={Inbox}
    title="No sessions yet"
    description="Create your first session to get started"
    action={
      <Button onClick={handleCreate}>
        <Plus className="w-4 h-4 mr-2" />
        New Session
      </Button>
    }
  />
) : (
  // Render list
)}
```

## React Query Patterns

### 1. Query Hooks

**Pattern**: Create typed query hooks in `services/queries/`

```typescript
export function useProjects() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: () => projectsApi.listProjects(),
    staleTime: 30000, // 30 seconds
  });
}
```

### 2. Mutation Hooks

**Pattern**: Include optimistic updates and cache invalidation

```typescript
export function useDeleteProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => projectsApi.deleteProject(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
    },
  });
}
```

### 3. Page Usage

**Pattern**: Destructure query results

```typescript
export default function ProjectsPage() {
  const { data: projects, isLoading, error } = useProjects();
  const deleteMutation = useDeleteProject();

  // Use loading.tsx for isLoading
  // Use error.tsx for error
  // Render data
}
```

## Layout Patterns

### 1. Page Structure

```typescript
<PageContainer maxWidth="xl">
  <PageHeader
    title="Projects"
    description="Manage your projects"
    actions={<Button>New Project</Button>}
  />

  <PageSection title="Active Projects">
    {/* Content */}
  </PageSection>
</PageContainer>
```

### 2. Sidebar Layout

```typescript
<SidebarLayout
  sidebar={<ProjectNav />}
  sidebarWidth="16rem"
>
  {children}
</SidebarLayout>
```

## Form Patterns

### 1. Form Fields

**Pattern**: Use FormFieldWrapper for consistency

```typescript
<FormFieldsGrid>
  <FormFieldWrapper
    label="Project Name"
    description="Unique identifier"
    error={errors.name}
  >
    <Input {...register('name')} />
  </FormFieldWrapper>
</FormFieldsGrid>
```

### 2. Submit Buttons

**Pattern**: Use LoadingButton for mutations

```typescript
<LoadingButton
  type="submit"
  loading={mutation.isPending}
  disabled={!isValid}
>
  Create Project
</LoadingButton>
```

## Custom Hooks

### 1. Async Actions

```typescript
const { execute, isLoading, error } = useAsyncAction(
  async (data) => {
    return await api.createProject(data);
  }
);

await execute(formData);
```

### 2. Local Storage

```typescript
const [theme, setTheme] = useLocalStorage('theme', 'light');
```

### 3. Clipboard

```typescript
const { copy, copied } = useClipboard();

<Button onClick={() => copy(text)}>
  {copied ? 'Copied!' : 'Copy'}
</Button>
```

## TypeScript Patterns

### 1. No Any Types

```typescript
// ✅ Good
type MessageHandler = (msg: SessionMessage) => void;

// ❌ Bad
type MessageHandler = (msg: any) => void;
```

### 2. Optional Chaining

```typescript
// ✅ Good
const name = project?.displayName ?? project.name;

// ❌ Bad
const name = project ? project.displayName || project.name : '';
```

### 3. Type Guards

```typescript
function isErrorResponse(data: unknown): data is ErrorResponse {
  return typeof data === 'object' &&
         data !== null &&
         'error' in data;
}
```

## Performance Patterns

### 1. Code Splitting

**Pattern**: Use dynamic imports for heavy components

```typescript
const HeavyComponent = dynamic(() => import('./HeavyComponent'), {
  loading: () => <Skeleton />,
});
```

### 2. React Query Caching

**Pattern**: Set appropriate staleTime

```typescript
// Fast-changing data
staleTime: 0

// Slow-changing data
staleTime: 300000 // 5 minutes

// Static data
staleTime: Infinity
```

## Accessibility Patterns

### 1. ARIA Labels

```typescript
<Button aria-label="Delete project">
  <Trash className="w-4 h-4" />
</Button>
```

### 2. Keyboard Navigation

```typescript
<div
  role="button"
  tabIndex={0}
  onKeyDown={(e) => e.key === 'Enter' && handleClick()}
>
  {content}
</div>
```

## Error Message Patterns

```typescript
// ✅ User-friendly
"Failed to load projects. Please try again."

// ❌ Technical
"Error: ECONNREFUSED 127.0.0.1:3000"
```

## Summary

Key patterns:
- Use `type` over `interface`
- Skeleton components for loading
- Error boundaries for errors
- EmptyState for empty lists
- React Query for data fetching
- TypeScript strict mode
- No `any` types
- Proper error messages
