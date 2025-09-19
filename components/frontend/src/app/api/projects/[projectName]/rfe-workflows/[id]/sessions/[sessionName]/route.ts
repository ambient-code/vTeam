import { BACKEND_URL } from '@/lib/config'

type RouteContext = { params: { projectName: string, id: string, sessionName: string } }

export async function DELETE(_request: Request, { params }: RouteContext) {
  const { projectName, id, sessionName } = params
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/rfe-workflows/${encodeURIComponent(id)}/sessions/${encodeURIComponent(sessionName)}`, { method: 'DELETE' })
  const data = await resp.json().catch(() => ({}))
  return Response.json(data, { status: resp.status })
}


