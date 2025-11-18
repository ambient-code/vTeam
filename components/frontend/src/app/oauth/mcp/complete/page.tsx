'use client'

import { useEffect, useState } from 'react'
import { useSearchParams } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { CheckCircle2, XCircle, Loader2 } from 'lucide-react'

/**
 * OAuth completion page for MCP servers.
 * This page receives the OAuth callback parameters and posts them back to
 * the Claude Code CLI window/iframe that initiated the OAuth flow.
 */
export default function MCPOAuthComplete() {
  const searchParams = useSearchParams()
  const [status, setStatus] = useState<'processing' | 'success' | 'error'>('processing')
  const [errorMessage, setErrorMessage] = useState<string>('')

  useEffect(() => {
    const code = searchParams.get('code')
    const state = searchParams.get('state')
    const error = searchParams.get('error')
    const errorDescription = searchParams.get('error_description')

    if (error) {
      setStatus('error')
      setErrorMessage(errorDescription || error)
      return
    }

    if (!code) {
      setStatus('error')
      setErrorMessage('No authorization code received')
      return
    }

    // Post message to opener window (Claude Code CLI or parent session page)
    const messageData = {
      type: 'oauth-callback',
      provider: 'mcp',
      code,
      state,
    }

    // Try to communicate with opener (popup scenario)
    if (window.opener && !window.opener.closed) {
      try {
        window.opener.postMessage(messageData, window.location.origin)
        setStatus('success')

        // Auto-close after 2 seconds
        setTimeout(() => {
          window.close()
        }, 2000)
      } catch (err) {
        console.error('Failed to post message to opener:', err)
        setStatus('error')
        setErrorMessage('Failed to communicate with parent window')
      }
    }
    // Try parent (iframe scenario)
    else if (window.parent && window.parent !== window) {
      try {
        window.parent.postMessage(messageData, window.location.origin)
        setStatus('success')
      } catch (err) {
        console.error('Failed to post message to parent:', err)
        setStatus('error')
        setErrorMessage('Failed to communicate with parent window')
      }
    }
    // Standalone page (shouldn't happen in normal flow)
    else {
      setStatus('error')
      setErrorMessage('No parent window found to communicate with')
    }
  }, [searchParams])

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            {status === 'processing' && (
              <>
                <Loader2 className="h-5 w-5 animate-spin" />
                Processing Authentication
              </>
            )}
            {status === 'success' && (
              <>
                <CheckCircle2 className="h-5 w-5 text-green-600" />
                Authentication Complete
              </>
            )}
            {status === 'error' && (
              <>
                <XCircle className="h-5 w-5 text-red-600" />
                Authentication Failed
              </>
            )}
          </CardTitle>
          <CardDescription>
            {status === 'processing' && 'Completing MCP server authentication...'}
            {status === 'success' && 'You can close this window now.'}
            {status === 'error' && 'There was a problem with the authentication.'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {status === 'processing' && (
            <p className="text-sm text-muted-foreground">
              Communicating with Claude Code session...
            </p>
          )}
          {status === 'success' && (
            <p className="text-sm text-muted-foreground">
              MCP server authentication was successful. This window will close automatically.
            </p>
          )}
          {status === 'error' && (
            <div className="space-y-2">
              <p className="text-sm text-red-600 font-medium">
                {errorMessage}
              </p>
              <p className="text-sm text-muted-foreground">
                Please close this window and try again.
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
