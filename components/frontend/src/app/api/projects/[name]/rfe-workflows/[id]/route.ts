import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; id: string }> },
) {
  const { name, id } = await params
  const headers = await buildForwardHeadersAsync(request)
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}`, { headers })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}

export async function PUT(
  request: Request,
  { params }: { params: Promise<{ name: string; id: string }> },
) {
  const { name, id } = await params
  const headers = await buildForwardHeadersAsync(request)
  const body = await request.text()
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { ...headers, 'Content-Type': 'application/json' },
    body
  })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ name: string; id: string }> },
) {
  const { name, id } = await params
  const headers = await buildForwardHeadersAsync(request)
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}`, { method: 'DELETE', headers })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


