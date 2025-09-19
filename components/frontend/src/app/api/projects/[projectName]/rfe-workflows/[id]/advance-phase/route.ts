import { BACKEND_URL } from '@/lib/config'

type RouteContext = { params: { projectName: string, id: string } }

export async function POST(request: Request, { params }: RouteContext) {
  const { projectName, id } = params
  const body = await request.json().catch(() => ({}))
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/rfe-workflows/${encodeURIComponent(id)}/advance-phase`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const data = await resp.json().catch(() => ({}))
  return Response.json(data, { status: resp.status })
}


