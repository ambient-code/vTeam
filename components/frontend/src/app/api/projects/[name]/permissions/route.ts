import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

// GET /api/projects/[name]/permissions - List project permissions (users & groups)
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/projects/${name}/permissions`, { headers });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      return Response.json(errorData, { status: response.status });
    }
    const data = await response.json();
    return Response.json(data);
  } catch (error) {
    console.error('Error fetching project permissions:', error);
    return Response.json({ error: 'Failed to fetch project permissions' }, { status: 500 });
  }
}

// POST /api/projects/[name]/permissions - Add permission assignment
export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const body = await request.json();
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/projects/${name}/permissions`, {
      method: 'POST',
      headers,
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      return Response.json(errorData, { status: response.status });
    }

    const data = await response.json();
    return Response.json(data, { status: 201 });
  } catch (error) {
    console.error('Error adding project permission:', error);
    return Response.json({ error: 'Failed to add project permission' }, { status: 500 });
  }
}


