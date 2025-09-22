import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

// PUT /api/projects/[name]/runner-secrets/trigger-sync
export async function PUT(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const response = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/runner-secrets/trigger-sync`, {
      method: 'PUT',
      headers,
    });
    const text = await response.text();
    return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
  } catch (error) {
    console.error('Error triggering secret sync:', error);
    return Response.json({ error: 'Failed to trigger secret sync' }, { status: 500 });
  }
}