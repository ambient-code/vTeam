import { BACKEND_URL } from '@/lib/config'

type RouteContext = {
  params: { projectName: string, id: string }
}

export async function GET(_request: Request, { params }: RouteContext) {
  const { projectName, id } = params
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/rfe-workflows/${encodeURIComponent(id)}`)
  const data = await resp.json().catch(() => ({}))
  return Response.json(data, { status: resp.status })
}

export async function DELETE(_request: Request, { params }: RouteContext) {
  const { projectName, id } = params
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/rfe-workflows/${encodeURIComponent(id)}`, { method: 'DELETE' })
  const data = await resp.json().catch(() => ({}))
  return Response.json(data, { status: resp.status })
}


