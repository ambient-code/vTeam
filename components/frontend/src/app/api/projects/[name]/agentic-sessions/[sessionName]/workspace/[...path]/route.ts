import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string; path: string[] }> },
) {
  const { name, sessionName, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const contentBase = `http://ambient-content-${encodeURIComponent(sessionName)}.${encodeURIComponent(name)}.svc.cluster.local:8080`
  const rel = path.join('/')
  const fsPath = `/sessions/${encodeURIComponent(sessionName)}/workspace/${rel}`
  const resp = await fetch(`${contentBase}/content/file?path=${encodeURIComponent(fsPath)}`, { headers })
  const contentType = resp.headers.get('content-type') || 'application/octet-stream'
  const buf = await resp.arrayBuffer()
  return new Response(buf, { status: resp.status, headers: { 'Content-Type': contentType } })
}


export async function PUT(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string; path: string[] }> },
) {
  const { name, sessionName, path } = await params
  const headers = await buildForwardHeadersAsync(request)
  const rel = path.join('/')
  const contentBase = `http://ambient-content-${encodeURIComponent(sessionName)}.${encodeURIComponent(name)}.svc.cluster.local:8080`
  const fsPath = `/sessions/${encodeURIComponent(sessionName)}/workspace/${rel}`
  const contentType = request.headers.get('content-type') || 'text/plain; charset=utf-8'
  const textBody = await request.text()
  const resp = await fetch(`${contentBase}/content/write`, {
    method: 'POST',
    headers: { ...headers, 'Content-Type': 'application/json' },
    body: JSON.stringify({ path: fsPath, content: textBody, encoding: 'utf8' }),
  })
  const respBody = await resp.text()
  return new Response(respBody, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}


