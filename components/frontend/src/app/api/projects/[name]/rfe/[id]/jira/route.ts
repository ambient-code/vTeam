import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

type PublishRequestBody = {
  phase?: 'specify' | 'plan' | 'tasks'
  path?: string
  issueTypeName?: string
}

function getExpectedPathForPhase(phase: string): string {
  if (phase === 'specify') return 'specs/spec.md'
  if (phase === 'plan') return 'specs/plan.md'
  return 'specs/tasks.md'
}

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string; id: string }> }
) {
  try {
    const { name, id } = await params
    const headers = await buildForwardHeadersAsync(request)

    const bodyText = await request.text()
    const body: PublishRequestBody = bodyText ? JSON.parse(bodyText) : {}
    const phase = body.phase || 'specify'
    const path = body.path || getExpectedPathForPhase(phase)

    // Delegate to backend which will create Jira and update CR
    const backendResp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}/jira`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...headers },
      body: JSON.stringify({ path })
    })
    const text = await backendResp.text()
    return new Response(text, { status: backendResp.status, headers: { 'Content-Type': 'application/json' } })
  } catch (error) {
    console.error('Error publishing to Jira:', error)
    return Response.json({ error: 'Failed to publish to Jira' }, { status: 500 })
  }
}

// GET /api/projects/[name]/rfe/[id]/jira?path=...
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; id: string }> }
) {
  try {
    const { name, id } = await params
    const headers = await buildForwardHeadersAsync(request)
    const url = new URL(request.url)
    const pathParam = url.searchParams.get('path') || ''
    const backendResp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}/jira?path=${encodeURIComponent(pathParam)}`, { headers })
    const text = await backendResp.text()
    return new Response(text, { status: backendResp.status, headers: { 'Content-Type': 'application/json' } })
  } catch (error) {
    console.error('Error fetching Jira issue:', error)
    return Response.json({ error: 'Failed to fetch Jira issue' }, { status: 500 })
  }
}


