import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function POST(request: Request) {
  const headers = await buildForwardHeadersAsync(request)
  const body = await request.text()

  const resp = await fetch(`${BACKEND_URL}/auth/github/install`, {
    method: 'POST',
    headers,
    body,
  })

  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


