import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; id: string; path: string[] }> },
) {
  const { name, id, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const rel = path.join('/')
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}/artifacts/${encodeURIComponent(rel)}`, { headers })
  const contentType = resp.headers.get('content-type') || 'text/plain'
  const body = await resp.text()
  return new Response(body, { status: resp.status, headers: { 'Content-Type': contentType } })
}

export async function PUT(
  request: Request,
  { params }: { params: Promise<{ name: string; id: string; path: string[] }> },
) {
  const { name, id, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const rel = path.join('/')
  const text = await request.text()
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}/artifacts/${encodeURIComponent(rel)}`, {
    method: 'PUT',
    headers: { ...headers, 'Content-Type': 'text/plain' },
    body: text,
  })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


