import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

// GET /api/projects/[name]/agentic-sessions - List sessions in a project
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions`, { headers });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error listing agentic sessions:', error);
    return Response.json({ error: 'Failed to list agentic sessions' }, { status: 500 });
  }
}

// POST /api/projects/[name]/agentic-sessions - Create a new session in a project
export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const body = await request.text();
    const headers = await buildForwardHeadersAsync(request);
    
    console.log('[API Route] Creating session for project:', name);
    console.log('[API Route] Auth headers present:', {
      hasUser: !!headers['X-Forwarded-User'],
      hasUsername: !!headers['X-Forwarded-Preferred-Username'],
      hasToken: !!headers['X-Forwarded-Access-Token'],
      hasEmail: !!headers['X-Forwarded-Email'],
    });
    
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions`, {
      method: 'POST',
      headers,
      body,
    });
    
    const text = await response.text();
    console.log('[API Route] Backend response status:', response.status);
    if (!response.ok) {
      console.error('[API Route] Backend error:', text);
    }
    
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error creating agentic session:', error);
    return Response.json({ error: 'Failed to create agentic session', details: error instanceof Error ? error.message : String(error) }, { status: 500 });
  }
}


