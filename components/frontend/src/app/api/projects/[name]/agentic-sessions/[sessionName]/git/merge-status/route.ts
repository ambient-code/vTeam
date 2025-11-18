import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params;
  const { searchParams } = new URL(request.url);
  const path = searchParams.get('path') || 'artifacts';
  const branch = searchParams.get('branch') || 'main';
  
  const headers = await buildForwardHeadersAsync(request);
  const resp = await fetch(
    `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/git/merge-status?path=${encodeURIComponent(path)}&branch=${encodeURIComponent(branch)}`,
    { headers }
  );
  const data = await resp.text();
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } });
}

