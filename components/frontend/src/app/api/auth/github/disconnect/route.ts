import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function POST(request: Request) {
  const headers = await buildForwardHeadersAsync(request)
  const resp = await fetch(`${BACKEND_URL}/auth/github/disconnect`, {
    method: 'POST',
    headers,
  })
  const text = await resp.text()
  return new Response(text, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


