'use client'

import React, { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { useConnectGitHub } from '@/services/queries'

export default function GitHubSetupPage() {
  const [message, setMessage] = useState<string>('Finalizing GitHub connection...')
  const [error, setError] = useState<string | null>(null)
  const connectMutation = useConnectGitHub()

  useEffect(() => {
    const url = new URL(window.location.href)
    const installationId = url.searchParams.get('installation_id')

    if (!installationId) {
      setMessage('No installation was detected.')
      return
    }

    connectMutation.mutate(
      { installationId: Number(installationId) },
      {
        onSuccess: () => {
          setMessage('GitHub connected. Redirecting...')
          setTimeout(() => {
            window.location.replace('/integrations')
          }, 800)
        },
        onError: (err) => {
          setError(err instanceof Error ? err.message : 'Failed to complete setup')
        },
      }
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <div className="max-w-lg mx-auto p-6">
      {error ? (
        <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>
      ) : (
        <div className="text-sm text-foreground/80">{message}</div>
      )}
      <div className="mt-4">
        <Button variant="ghost" onClick={() => window.location.replace('/integrations')}>Back to Integrations</Button>
      </div>
    </div>
  )
}


