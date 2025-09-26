'use client'

import React, { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'

export default function GitHubSetupPage() {
  const [message, setMessage] = useState<string>('Finalizing GitHub connection...')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const url = new URL(window.location.href)
    const installationId = url.searchParams.get('installation_id')
    const setupAction = url.searchParams.get('setup_action')

    if (!installationId) {
      setMessage('No installation was detected.')
      return
    }

    const link = async () => {
      try {
        const resp = await fetch('/api/auth/github/install', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ installationId: Number(installationId) }),
        })
        if (!resp.ok) {
          const t = await resp.text().catch(() => '')
          throw new Error(t || `Link failed (${resp.status})`)
        }
        setMessage('GitHub connected. Redirecting...')
        setTimeout(() => {
          window.location.replace('/integrations')
        }, 800)
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to complete setup')
      }
    }
    link()
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


