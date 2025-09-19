import { BACKEND_URL } from '@/lib/config'

type RouteContext = { params: { projectName: string, id: string } }

export async function POST(_request: Request, { params }: RouteContext) {
  const { projectName, id } = params
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/rfe-workflows/${encodeURIComponent(id)}/resume`, { method: 'POST' })
  const data = await resp.json().catch(() => ({}))
  return Response.json(data, { status: resp.status })
}


