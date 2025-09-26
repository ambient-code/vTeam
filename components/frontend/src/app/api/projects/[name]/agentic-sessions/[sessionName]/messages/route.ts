import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params
  const headers = await buildForwardHeadersAsync(request)
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/sessions/${encodeURIComponent(sessionName)}/messages`, {
    method: 'GET',
    headers,
  })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params
  const headers = await buildForwardHeadersAsync(request)
  const body = await request.text()
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/sessions/${encodeURIComponent(sessionName)}/messages`, {
    method: 'POST',
    headers: { ...headers, 'Content-Type': 'application/json' },
    body,
  })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}
