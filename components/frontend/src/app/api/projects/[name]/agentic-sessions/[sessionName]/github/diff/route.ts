import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params
  const headers = await buildForwardHeadersAsync(request)
  const url = new URL(request.url)
  const repoIndex = url.searchParams.get('repoIndex')
  const repoPath = url.searchParams.get('repoPath')
  const qs = new URLSearchParams()
  if (repoIndex) qs.set('repoIndex', repoIndex)
  if (repoPath) qs.set('repoPath', repoPath)
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/github/diff?${qs.toString()}`, { headers })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}
