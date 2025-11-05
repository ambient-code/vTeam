import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'
import { USE_MOCKS } from '@/lib/mock-config'
import { handleGetGitHubStatus } from '@/lib/mocks/handlers'

export async function GET(request: Request) {
  // Return mock data if enabled
  if (USE_MOCKS) {
    return handleGetGitHubStatus();
  }

  const headers = await buildForwardHeadersAsync(request)

  const resp = await fetch(`${BACKEND_URL}/auth/github/status`, {
    method: 'GET',
    headers,
  })

  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


