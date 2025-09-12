import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

// DELETE /api/projects/[name]/keys/[keyId] - Delete project access key
export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ name: string; keyId: string }> }
) {
  try {
    const { name, keyId } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/projects/${name}/keys/${encodeURIComponent(keyId)}`, {
      method: 'DELETE',
      headers,
    });

    if (!response.ok && response.status !== 204) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      return Response.json(errorData, { status: response.status });
    }

    return new Response(null, { status: 204 });
  } catch (error) {
    console.error('Error deleting project key:', error);
    return Response.json({ error: 'Failed to delete project key' }, { status: 500 });
  }
}


