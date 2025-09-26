import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params
  const headers = await buildForwardHeadersAsync(request)
  // Per-job content service (sidecar) name
  const contentBase = `http://ambient-content-${encodeURIComponent(sessionName)}.${encodeURIComponent(name)}.svc.cluster.local:8080`
  const url = new URL(request.url)
  const subpath = url.searchParams.get('path')
  // Match backend resolveWorkspaceAbsPath semantics: /sessions/<session>/workspace[/subpath]
  const pathRoot = `/sessions/${encodeURIComponent(sessionName)}/workspace${subpath ? `/${subpath}` : ''}`
  const resp = await fetch(
    `${contentBase}/content/list?path=${encodeURIComponent(pathRoot)}`,
    { headers },
  )
  const contentType = resp.headers.get('content-type') || 'application/json'
  const body = await resp.text()
  return new Response(body, { status: resp.status, headers: { 'Content-Type': contentType } })
}


