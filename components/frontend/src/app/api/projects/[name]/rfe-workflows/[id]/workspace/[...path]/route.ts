import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; id: string; path: string[] }> },
) {
  const { name, id, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const rel = path.join('/')
  const resp = await fetch(
    `${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}/workspace/${encodeURIComponent(rel)}`,
    { headers },
  )
  const contentType = resp.headers.get('content-type') || 'application/octet-stream'
  const buf = await resp.arrayBuffer()
  return new Response(buf, { status: resp.status, headers: { 'Content-Type': contentType } })
}


export async function PUT(
  request: Request,
  { params }: { params: Promise<{ name: string; id: string; path: string[] }> },
) {
  const { name, id, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const rel = path.join('/')
  const contentType = request.headers.get('content-type') || 'application/octet-stream'
  const body = await request.arrayBuffer()
  const resp = await fetch(
    `${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}/workspace/${encodeURIComponent(rel)}`,
    { method: 'PUT', headers: { ...headers, 'Content-Type': contentType }, body }
  )
  const respBody = await resp.text()
  return new Response(respBody, { status: resp.status, headers: { 'Content-Type': resp.headers.get('content-type') || 'application/json' } })
}


