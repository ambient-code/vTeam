'use client'

import React from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { useGitHubStatus, useDisconnectGitHub } from '@/services/queries'
import { successToast, errorToast } from '@/hooks/use-toast'

type Props = { 
  appSlug?: string
  showManageButton?: boolean
}

export function GitHubConnectionCard({ appSlug, showManageButton = true }: Props) {
  const { data: status, isLoading, refetch } = useGitHubStatus()
  const disconnectMutation = useDisconnectGitHub()

  const handleConnect = () => {
    if (!appSlug) return
    const setupUrl = new URL('/integrations/github/setup', window.location.origin)
    const redirectUri = encodeURIComponent(setupUrl.toString())
    const url = `https://github.com/apps/${appSlug}/installations/new?redirect_uri=${redirectUri}`
    window.location.href = url
  }

  const handleDisconnect = async () => {
    disconnectMutation.mutate(undefined, {
      onSuccess: () => {
        successToast('GitHub disconnected successfully')
        refetch()
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to disconnect GitHub')
      },
    })
  }

  const handleManage = () => {
    window.open('https://github.com/settings/installations', '_blank')
  }

  return (
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
          {showManageButton && (
            <Button variant="ghost" onClick={handleManage} disabled={isLoading || disconnectMutation.isPending}>
              Manage in GitHub
            </Button>
          )}
          {status?.installed ? (
            <Button variant="destructive" onClick={handleDisconnect} disabled={isLoading || disconnectMutation.isPending}>
              Disconnect
            </Button>
          ) : (
            <Button onClick={handleConnect} disabled={isLoading || !appSlug}>
              Connect
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

