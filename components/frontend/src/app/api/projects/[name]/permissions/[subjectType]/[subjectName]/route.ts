import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

// DELETE /api/projects/[name]/permissions/[subjectType]/[subjectName] - Remove permission
export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ name: string; subjectType: string; subjectName: string }> }
) {
  try {
    const { name, subjectType, subjectName } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/projects/${name}/permissions/${encodeURIComponent(subjectType)}/${encodeURIComponent(subjectName)}`, {
      method: 'DELETE',
      headers,
    });

    if (!response.ok && response.status !== 204) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      return Response.json(errorData, { status: response.status });
    }

    return new Response(null, { status: 204 });
  } catch (error) {
    console.error('Error removing project permission:', error);
    return Response.json({ error: 'Failed to remove project permission' }, { status: 500 });
  }
}


