import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

// GET /api/projects/[name]/keys - List project access keys
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/projects/${name}/keys`, { headers });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      return Response.json(errorData, { status: response.status });
    }
    const data = await response.json();
    return Response.json(data);
  } catch (error) {
    console.error('Error fetching project keys:', error);
    return Response.json({ error: 'Failed to fetch project keys' }, { status: 500 });
  }
}

// POST /api/projects/[name]/keys - Create project access key
export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const body = await request.json();
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/projects/${name}/keys`, {
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
    console.error('Error creating project key:', error);
    return Response.json({ error: 'Failed to create project key' }, { status: 500 });
  }
}


