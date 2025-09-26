'use client'

import React, { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

type GitHubStatus = {
  installed: boolean
  installationId?: number
  githubUserId?: string
  userId?: string
  host?: string
  updatedAt?: string
}

type Props = { appSlug?: string }

export default function IntegrationsClient({ appSlug }: Props) {
  const [status, setStatus] = useState<GitHubStatus | null>(null)
  const [loading, setLoading] = useState(false)

  const refresh = async () => {
    setLoading(true)
    try {
      const resp = await fetch('/api/auth/github/status', { cache: 'no-store' })
      const data = await resp.json().catch(() => ({}))
      const mapped: GitHubStatus = {
        installed: !!data.installed,
        installationId: data.installationId,
        githubUserId: data.githubUserId,
        userId: data.userId,
        host: data.host,
        updatedAt: data.updatedAt,
      }
      setStatus(mapped)
    } catch {
      setStatus({ installed: false })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  const handleConnect = () => {
    if (!appSlug) return
    const setupUrl = new URL('/integrations/github/setup', window.location.origin)
    const redirectUri = encodeURIComponent(setupUrl.toString())
    const url = `https://github.com/apps/${appSlug}/installations/new?redirect_uri=${redirectUri}`
    window.location.href = url
  }

  const handleDisconnect = async () => {
    setLoading(true)
    try {
      await fetch('/api/auth/github/disconnect', { method: 'POST' })
      await refresh()
    } finally {
      setLoading(false)
    }
  }

  const handleManage = () => {
    window.open('https://github.com/settings/installations', '_blank')
  }

  return (
    <div className="max-w-3xl mx-auto p-6 space-y-6">
      <h1 className="text-2xl font-semibold">Integrations</h1>

      <Card>
        <CardHeader>
          <CardTitle>GitHub</CardTitle>
          <CardDescription>Connect GitHub to enable forks, PRs, and repo browsing</CardDescription>
        </CardHeader>
        <CardContent className="flex items-center justify-between gap-4">
          <div className="text-sm">
            {status?.installed ? (
              <div>
                Connected{status.githubUserId ? ` as ${status.githubUserId}` : ''}
                {status.updatedAt ? (
                  <span className="text-gray-500"> · updated {new Date(status.updatedAt).toLocaleString()}</span>
                ) : null}
              </div>
            ) : (
              <div>Not connected</div>
            )}
          </div>
          <div className="flex gap-2">
            <Button variant="ghost" onClick={handleManage} disabled={loading}>Manage in GitHub</Button>
            {status?.installed ? (
              <Button variant="destructive" onClick={handleDisconnect} disabled={loading}>Disconnect</Button>
            ) : (
              <Button onClick={handleConnect} disabled={loading || !appSlug}>Connect</Button>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}


