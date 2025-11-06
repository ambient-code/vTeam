'use client'

import React, { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { useConnectGitHub } from '@/services/queries'
import { publicEnv } from '@/lib/env'

export default function GitHubSetupPage() {
  const [message, setMessage] = useState<string>('Preparing GitHub connection...')
  const [error, setError] = useState<string | null>(null)
  const connectMutation = useConnectGitHub()

  useEffect(() => {
    const url = new URL(window.location.href)
    const installationId = url.searchParams.get('installation_id')
    const fromOnboarding = url.searchParams.get('from') === 'onboarding'

    // If returning from GitHub with installation_id, complete the connection
    if (installationId) {
      setMessage('Finalizing GitHub connection...')
      connectMutation.mutate(
        { installationId: Number(installationId) },
        {
          onSuccess: () => {
            setMessage('GitHub connected successfully! Redirecting...')
            setTimeout(() => {
              // Redirect back to onboarding or integrations page
              window.location.replace(fromOnboarding ? '/onboarding' : '/integrations')
            }, 800)
          },
          onError: (err) => {
            setError(err instanceof Error ? err.message : 'Failed to complete setup')
          },
        }
      )
    } else {
      // No installation_id, redirect to GitHub OAuth
      const appSlug = publicEnv.GITHUB_APP_SLUG
      
      if (!appSlug) {
        setError('GitHub App is not configured. Please contact your administrator.')
        return
      }

      setMessage('Redirecting to GitHub...')
      
      // Build the redirect URI with from parameter to return to the right place
      const setupUrl = new URL('/integrations/github/setup', window.location.origin)
      if (fromOnboarding) {
        setupUrl.searchParams.set('from', 'onboarding')
      }
      const redirectUri = encodeURIComponent(setupUrl.toString())
      
      // Redirect to GitHub OAuth
      const githubUrl = `https://github.com/apps/${appSlug}/installations/new?redirect_uri=${redirectUri}`
      
      setTimeout(() => {
        window.location.href = githubUrl
      }, 500)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <div className="max-w-lg mx-auto p-6">
      {error ? (
        <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>
      ) : (
        <div className="text-sm text-gray-700">{message}</div>
      )}
      <div className="mt-4">
        <Button variant="ghost" onClick={() => window.location.replace('/integrations')}>Back to Integrations</Button>
      </div>
    </div>
  )
}


