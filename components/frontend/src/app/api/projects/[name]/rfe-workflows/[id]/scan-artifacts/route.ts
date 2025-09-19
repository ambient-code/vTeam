import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string; id: string }> },
) {
  const { name, id } = await params
  const headers = await buildForwardHeadersAsync(request)
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}/scan-artifacts`, {
    method: 'POST',
    headers,
  })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


