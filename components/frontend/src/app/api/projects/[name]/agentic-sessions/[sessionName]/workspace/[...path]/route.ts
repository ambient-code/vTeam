import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string; path: string[] }> },
) {
  const { name, sessionName, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const rel = path.join('/')
  const resp = await fetch(
    `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${encodeURIComponent(rel)}`,
    { headers },
  )
  const contentType = resp.headers.get('content-type') || 'application/octet-stream'
  const buf = await resp.arrayBuffer()
  return new Response(buf, { status: resp.status, headers: { 'Content-Type': contentType } })
}


