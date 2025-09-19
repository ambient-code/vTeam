import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ name: string; id: string; sessionName: string }> },
) {
  const { name, id, sessionName } = await params
  const headers = await buildForwardHeadersAsync(request)
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}/sessions/${encodeURIComponent(sessionName)}`, {
    method: 'DELETE',
    headers,
  })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


