import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(request: Request) {
  const headers = await buildForwardHeadersAsync(request)
  const url = new URL(request.url)
  const resp = await fetch(`${BACKEND_URL}/auth/github/user/callback${url.search}`, {
    method: 'GET',
    headers,
    redirect: 'manual',
  })

  // Forward redirects from backend (e.g., to /integrations)
  const location = resp.headers.get('location')
  if (location && [301, 302, 303, 307, 308].includes(resp.status)) {
    return new Response(null, { status: resp.status, headers: { Location: location } })
  }

  const text = await resp.text()
  return new Response(text, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


