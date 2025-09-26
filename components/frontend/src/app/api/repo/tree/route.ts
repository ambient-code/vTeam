import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

// GET /api/repo/tree - proxy to backend global repo tree
export async function GET(request: Request) {
  const headers = await buildForwardHeadersAsync(request)
  const url = new URL(request.url)
  const qs = url.search
  const resp = await fetch(`${BACKEND_URL}/repo/tree${qs}`, { headers })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


