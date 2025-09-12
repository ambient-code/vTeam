import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

type Ctx = { params: Promise<{ name: string; sessionName: string }> };

// GET /api/projects/[name]/agentic-sessions/[sessionName]
export async function GET(request: Request, { params }: Ctx) {
  try {
    const { name, sessionName } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}`, { headers });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error fetching agentic session:', error);
    return Response.json({ error: 'Failed to fetch agentic session' }, { status: 500 });
  }
}

// PUT /api/projects/[name]/agentic-sessions/[sessionName]
export async function PUT(request: Request, { params }: Ctx) {
  try {
    const { name, sessionName } = await params;
    const body = await request.text();
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...headers },
      body,
    });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error updating agentic session:', error);
    return Response.json({ error: 'Failed to update agentic session' }, { status: 500 });
  }
}

// DELETE /api/projects/[name]/agentic-sessions/[sessionName]
export async function DELETE(request: Request, { params }: Ctx) {
  try {
    const { name, sessionName } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}`, {
      method: 'DELETE',
      headers,
    });
    if (response.status === 204) return new Response(null, { status: 204 });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error deleting agentic session:', error);
    return Response.json({ error: 'Failed to delete agentic session' }, { status: 500 });
  }
}


