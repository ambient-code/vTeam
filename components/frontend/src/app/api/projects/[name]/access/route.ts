import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const headers = await buildForwardHeadersAsync(request);

    const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/access`, { headers });
    const data = await resp.json().catch(() => ({}));
    return Response.json(data, { status: resp.status });
  } catch (error) {
    console.error('Error performing access check:', error);
    return Response.json({ error: 'Failed to perform access check' }, { status: 500 });
  }
}


