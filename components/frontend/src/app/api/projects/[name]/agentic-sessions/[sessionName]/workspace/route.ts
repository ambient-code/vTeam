import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params
  const headers = await buildForwardHeadersAsync(request)
  const url = new URL(request.url)
  const subpath = url.searchParams.get('path')
  const qs = subpath ? `?path=${encodeURIComponent(subpath)}` : ''
  const resp = await fetch(
    `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace${qs}`,
    { headers },
  )
  const contentType = resp.headers.get('content-type') || 'application/json'
  const body = await resp.text()
  return new Response(body, { status: resp.status, headers: { 'Content-Type': contentType } })
}


