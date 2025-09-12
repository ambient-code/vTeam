import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

// GET /api/projects/[name]/secrets - List Opaque secrets in a project
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/secrets`, { headers });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error listing secrets:', error);
    return Response.json({ error: 'Failed to list secrets' }, { status: 500 });
  }
}


