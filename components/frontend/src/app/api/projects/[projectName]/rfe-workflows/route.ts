import { BACKEND_URL } from '@/lib/config'

type RouteContext = {
  params: { projectName: string }
}

export async function GET(_request: Request, { params }: RouteContext) {
  const { projectName } = params
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/rfe-workflows`)
  const data = await resp.json().catch(() => ({}))
  return Response.json(data, { status: resp.status })
}

export async function POST(request: Request, { params }: RouteContext) {
  const { projectName } = params
  const body = await request.json()
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/rfe-workflows`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const data = await resp.json().catch(() => ({}))
  return Response.json(data, { status: resp.status })
}


