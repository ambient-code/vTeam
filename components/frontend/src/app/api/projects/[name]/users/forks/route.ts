import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> },
) {
  const { name } = await params
  const headers = await buildForwardHeadersAsync(request)
  const url = new URL(request.url)
  const qs = url.search
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/users/forks${qs}`, { headers, cache: 'no-store' })
  // Pass through upstream response body and content-type
  const contentType = resp.headers.get('content-type') || 'application/json'
  return new Response(resp.body, { status: resp.status, headers: { 'Content-Type': contentType, 'Cache-Control': 'no-store' } })
}

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string }> },
) {
  const { name } = await params
  const headers = await buildForwardHeadersAsync(request)
  const body = await request.text()
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/users/forks`, {
    method: 'POST',
    headers,
    body,
  })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


