import { NextRequest, NextResponse } from 'next/server'

/**
 * OAuth callback endpoint for MCP servers (e.g., Atlassian MCP).
 * This receives the OAuth redirect from the MCP provider and forwards
 * the authorization code to the completion page, which communicates
 * back to the Claude Code CLI running in the session pod.
 */
export async function GET(request: NextRequest) {
  const code = request.nextUrl.searchParams.get('code')
  const state = request.nextUrl.searchParams.get('state')
  const error = request.nextUrl.searchParams.get('error')
  const errorDescription = request.nextUrl.searchParams.get('error_description')

  // Build redirect URL to completion page with all OAuth params
  const completionUrl = new URL('/oauth/mcp/complete', request.url)

  if (error) {
    completionUrl.searchParams.set('error', error)
    if (errorDescription) {
      completionUrl.searchParams.set('error_description', errorDescription)
    }
  } else if (code) {
    completionUrl.searchParams.set('code', code)
    if (state) {
      completionUrl.searchParams.set('state', state)
    }
  } else {
    completionUrl.searchParams.set('error', 'invalid_request')
    completionUrl.searchParams.set('error_description', 'Missing authorization code')
  }

  return NextResponse.redirect(completionUrl)
}
