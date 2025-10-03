import { buildForwardHeadersAsync } from '@/lib/auth'
import { BACKEND_URL } from '@/lib/config';

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string; path: string[] }> },
) {
  const { name, sessionName, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const rel = path.join('/')
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${encodeURIComponent(rel)}`, { headers })
  const contentType = resp.headers.get('content-type') || 'application/octet-stream'
  const buf = await resp.arrayBuffer()
  return new Response(buf, { status: resp.status, headers: { 'Content-Type': contentType } })
}


export async function PUT(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string; path: string[] }> },
) {
  const { name, sessionName, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const rel = path.join('/')
  const contentType = request.headers.get('content-type') || 'text/plain; charset=utf-8'
  const textBody = await request.text()
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${encodeURIComponent(rel)}`, {
    method: 'PUT',
    headers: { ...headers, 'Content-Type': contentType },
    body: textBody,
  })
  const respBody = await resp.text()
  return new Response(respBody, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


