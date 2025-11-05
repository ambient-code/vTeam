import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'
import { USE_MOCKS } from '@/lib/mock-config'
import { handleListRFEWorkflows } from '@/lib/mocks/handlers'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> },
) {
  const { name } = await params

  // Return mock data if enabled
  if (USE_MOCKS) {
    return handleListRFEWorkflows(name);
  }

  const headers = await buildForwardHeadersAsync(request)
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows`, { headers })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string }> },
) {
  const { name } = await params
  const headers = await buildForwardHeadersAsync(request)
  const body = await request.text()
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows`, {
    method: 'POST',
    headers,
    body,
  })
  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


