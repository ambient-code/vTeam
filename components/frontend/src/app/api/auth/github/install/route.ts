import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'
import { USE_MOCKS } from '@/lib/mock-config'
import { handleInstallGitHub } from '@/lib/mocks/handlers'

export async function POST(request: Request) {
  // Return mock data if enabled
  if (USE_MOCKS) {
    return handleInstallGitHub(request);
  }

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


