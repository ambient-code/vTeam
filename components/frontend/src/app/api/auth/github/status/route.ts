import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(request: Request) {
  const headers = await buildForwardHeadersAsync(request)

  const resp = await fetch(`${BACKEND_URL}/auth/github/status`, {
    method: 'GET',
    headers,
  })

  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


