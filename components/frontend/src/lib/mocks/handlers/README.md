# Mock API Handlers

This directory contains centralized mock handlers for all API endpoints. Each handler file corresponds to a specific API domain.

## Structure

```
handlers/
├── auth.ts          # Authentication endpoints (/api/me)
├── github.ts        # GitHub integration (/api/auth/github/*)
├── projects.ts      # Project management (/api/projects/*)
├── sessions.ts      # Agentic sessions (/api/projects/[name]/agentic-sessions)
├── rfe.ts          # RFE workflows (/api/projects/[name]/rfe-workflows)
├── cluster.ts       # System endpoints (/api/cluster-info, /api/version)
└── index.ts         # Centralized exports
```

## Usage

### In API Routes

Instead of implementing mock logic directly in route files, import and use the centralized handlers:

```typescript
// src/app/api/me/route.ts
import { USE_MOCKS } from '@/lib/mock-config';
import { handleGetMe } from '@/lib/mocks/handlers';

export async function GET(request: Request) {
  if (USE_MOCKS) {
    return handleGetMe();
  }
  
  // Real backend logic...
}
```

### Adding New Mock Handlers

1. **Create handler function** in the appropriate file (e.g., `auth.ts`):

```typescript
export async function handleNewEndpoint(): Promise<NextResponse> {
  return createMockResponse('/new-endpoint', () => mockData);
}
```

2. **Export from index.ts** (if not using wildcard export):

```typescript
export { handleNewEndpoint } from './auth';
```

3. **Use in API route**:

```typescript
import { USE_MOCKS } from '@/lib/mock-config';
import { handleNewEndpoint } from '@/lib/mocks/handlers';

export async function GET() {
  if (USE_MOCKS) {
    return handleNewEndpoint();
  }
  // ...
}
```

## Benefits

### ✅ Centralized Logic
- All mock implementations in one place
- Easy to find and maintain
- Consistent patterns across handlers

### ✅ Separation of Concerns
- API routes focus on routing and real backend calls
- Mock logic isolated from route implementation
- Clean interface between mocking and routing

### ✅ Reusability
- Handlers can be reused across multiple routes
- Common patterns extracted to utilities
- Easy to test independently

### ✅ Type Safety
- Full TypeScript support
- Type-checked request/response data
- IntelliSense support in IDE

## Handler Patterns

### Simple GET Handler

```typescript
export async function handleGetResource(): Promise<NextResponse> {
  return createMockResponse('/endpoint', () => mockData);
}
```

### Handler with Path Parameters

```typescript
export async function handleGetResource(id: string): Promise<NextResponse> {
  const resource = getMockResource(id);
  if (!resource) {
    return createMockError(`/endpoint/${id}`, 'Not found', 404);
  }
  return createMockResponse(`/endpoint/${id}`, () => resource);
}
```

### Handler with Request Body

```typescript
export async function handleCreateResource(
  request: NextRequest
): Promise<NextResponse> {
  try {
    const body = await request.json() as CreateResourceRequest;
    const newResource = {
      id: generateId(),
      ...body,
      createdAt: new Date().toISOString(),
    };
    mockResources.push(newResource);
    return createMockResponse('/endpoint', () => newResource, 201);
  } catch (error) {
    return NextResponse.json({ error: 'Invalid request' }, { status: 400 });
  }
}
```

### Handler with State Mutation

```typescript
export async function handleDeleteResource(id: string): Promise<NextResponse> {
  const index = mockResources.findIndex(r => r.id === id);
  if (index === -1) {
    return createMockError(`/endpoint/${id}`, 'Not found', 404);
  }
  mockResources.splice(index, 1);
  return createMockResponse(`/endpoint/${id}`, () => ({ 
    message: 'Deleted successfully' 
  }));
}
```

## Files Overview

### auth.ts
Handles authentication and user session endpoints:
- `handleGetMe()` - Returns current user authentication status

### github.ts
Handles GitHub integration endpoints:
- `handleGetGitHubStatus()` - Returns GitHub connection status
- `handleInstallGitHub(request)` - Simulates GitHub App installation

### projects.ts
Handles project management endpoints:
- `handleListProjects()` - Returns list of projects
- `handleCreateProject(request)` - Creates new project
- `handleGetProject(name)` - Returns single project
- `handleDeleteProject(name)` - Deletes project

### sessions.ts
Handles agentic session endpoints:
- `handleListSessions(projectName)` - Returns sessions for project

### rfe.ts
Handles RFE workflow endpoints:
- `handleListRFEWorkflows(projectName)` - Returns RFE workflows for project

### cluster.ts
Handles system and cluster endpoints:
- `handleGetClusterInfo()` - Returns cluster information
- `handleGetVersion()` - Returns application version

## Migration Guide

If you find mock logic inline in an API route file, migrate it to a handler:

### Before (Inline in Route)
```typescript
// src/app/api/resource/route.ts
export async function GET() {
  if (USE_MOCKS) {
    return createMockResponse('/resource', () => mockData);
  }
  // backend logic...
}
```

### After (Using Handler)
```typescript
// src/lib/mocks/handlers/resource.ts
export async function handleGetResource(): Promise<NextResponse> {
  return createMockResponse('/resource', () => mockData);
}

// src/app/api/resource/route.ts
import { handleGetResource } from '@/lib/mocks/handlers';

export async function GET() {
  if (USE_MOCKS) {
    return handleGetResource();
  }
  // backend logic...
}
```

## Testing

Handlers can be tested independently:

```typescript
import { handleGetResource } from '@/lib/mocks/handlers';

test('handleGetResource returns mock data', async () => {
  const response = await handleGetResource();
  const data = await response.json();
  expect(data).toBeDefined();
});
```

## Related Documentation

- [Main Mocking Guide](../../MOCKING_GUIDE.md) - Complete mock system guide
- [Mock Data Files](../) - Mock data definitions
- [API Handler Utilities](../api-handler.ts) - Helper functions

