import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

// GET /api/projects/[name]/runner-secrets/config
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/runner-secrets/config`, { headers });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error getting runner secrets config:', error);
    return Response.json({ error: 'Failed to get runner secrets config' }, { status: 500 });
  }
}

// PUT /api/projects/[name]/runner-secrets/config
export async function PUT(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const body = await request.text();
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/runner-secrets/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...headers },
      body,
    });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error updating runner secrets config:', error);
    return Response.json({ error: 'Failed to update runner secrets config' }, { status: 500 });
  }
}


