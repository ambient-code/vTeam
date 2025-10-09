import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

// GET /api/repo/blob - proxy to backend project-scoped repo blob
export async function GET(request: Request) {
  const headers = await buildForwardHeadersAsync(request)
  const url = new URL(request.url)
  const projectName = url.searchParams.get('project')

  if (!projectName) {
    return new Response(JSON.stringify({ error: 'project parameter is required' }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' }
    })
  }

  // Remove project from query params before forwarding
  url.searchParams.delete('project')
  const qs = url.search

  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/repo/blob${qs}`, { headers })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


