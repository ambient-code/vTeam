import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

// GET /api/projects/[name] - Get project by name
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/projects/${name}`, { headers });
    if (!response.ok) {
      throw new Error(`Backend responded with status: ${response.status}`);
    }
    const data = await response.json();
    return Response.json(data);
  } catch (error) {
    console.error('Error fetching project:', error);
    return Response.json({ error: 'Failed to fetch project' }, { status: 500 });
  }
}

// PUT /api/projects/[name] - Update project
export async function PUT(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const body = await request.json();
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/projects/${name}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      return Response.json(errorData, { status: response.status });
    }

    const data = await response.json();
    return Response.json(data);
  } catch (error) {
    console.error('Error updating project:', error);
    return Response.json({ error: 'Failed to update project' }, { status: 500 });
  }
}

// DELETE /api/projects/[name] - Delete project
export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/projects/${name}`, {
      method: 'DELETE',
      headers,
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      return Response.json(errorData, { status: response.status });
    }

    if (response.status === 204) {
      return new Response(null, { status: 204 });
    }

    const data = await response.json();
    return Response.json(data);
  } catch (error) {
    console.error('Error deleting project:', error);
    return Response.json({ error: 'Failed to delete project' }, { status: 500 });
  }
}