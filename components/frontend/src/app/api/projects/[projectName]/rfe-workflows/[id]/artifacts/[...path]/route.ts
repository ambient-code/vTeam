import { BACKEND_URL } from '@/lib/config'

type RouteContext = { params: { projectName: string, id: string, path: string[] } }

export async function GET(_request: Request, { params }: RouteContext) {
  const { projectName, id, path } = params
  const joined = path.join('/')
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/rfe-workflows/${encodeURIComponent(id)}/artifacts/${joined}`)
  const text = await resp.text()
  return new Response(text, { status: resp.status, headers: { 'Content-Type': 'text/plain' } })
}

export async function PUT(request: Request, { params }: RouteContext) {
  const { projectName, id, path } = params
  const joined = path.join('/')
  const body = await request.text()
  const resp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/rfe-workflows/${encodeURIComponent(id)}/artifacts/${joined}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'text/plain' },
    body,
  })
  const data = await resp.json().catch(() => ({}))
  return Response.json(data, { status: resp.status })
}


