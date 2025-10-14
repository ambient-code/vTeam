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

    // Resolve repo/ref for this workflow to fetch content from GitHub via backend
    // 1) Load workflow to get repositories and canonical branch if present
    const wfResp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}`, { headers })
    if (!wfResp.ok) {
      const errText = await wfResp.text()
      return new Response(errText, { status: wfResp.status, headers: { 'Content-Type': 'application/json' } })
    }
    const wf = await wfResp.json()
    // Use umbrellaRepo as the main repository
    const umbrellaUrl: string | undefined = wf?.umbrellaRepo?.url
    const umbrellaBranch: string | undefined = wf?.umbrellaRepo?.branch
    const repo: string | undefined = umbrellaUrl?.replace(/^https?:\/\/(?:www\.)?github.com\//i, '').replace(/\.git$/i, '')
    const ref: string = (umbrellaBranch || 'main')
    if (!repo) {
      return Response.json({ error: 'Workflow umbrellaRepo not configured' }, { status: 400 })
    }

    // 2) Fetch file content via backend repo blob proxy
    const blobUrl = `${BACKEND_URL}/projects/${encodeURIComponent(name)}/repo/blob?repo=${encodeURIComponent(repo)}&ref=${encodeURIComponent(ref)}&path=${encodeURIComponent(path)}`
    const blobResp = await fetch(blobUrl, { headers })
    if (!blobResp.ok) {
      const errText = await blobResp.text()
      return new Response(errText, { status: blobResp.status, headers: { 'Content-Type': 'application/json' } })
    }
    const blobData = await blobResp.json().catch(async () => ({ content: await blobResp.text() }))

    // 3) Delegate to backend to create Jira and update CR (now that content can be validated server-side if needed)
    const backendResp = await fetch(`${BACKEND_URL}/projects/${encodeURIComponent(name)}/rfe-workflows/${encodeURIComponent(id)}/jira`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...headers },
      body: JSON.stringify({ path, phase })
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


