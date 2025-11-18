'use client'

import React from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
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
    <Card className="bg-white border border-gray-200 shadow-sm">
      <div className="p-6">
        {/* Header section with icon and title */}
        <div className="flex items-start gap-4 mb-6">
          <div className="flex-shrink-0 w-16 h-16 bg-gray-900 rounded-lg flex items-center justify-center">
            <svg className="w-8 h-8 text-white" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
            </svg>
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-gray-900 mb-1">GitHub</h3>
            <p className="text-gray-600">Connect to GitHub repositories</p>
          </div>
        </div>

        {/* Status section */}
        <div className="mb-4">
          <div className="flex items-center gap-2 mb-2">
            <span className={`w-2 h-2 rounded-full ${status?.installed ? 'bg-green-500' : 'bg-gray-400'}`}></span>
            <span className="text-sm font-medium text-gray-700">
              {status?.installed ? (
                <>Connected{status.githubUserId ? ` as ${status.githubUserId}` : ''}</>
              ) : (
                'Not Connected'
              )}
            </span>
          </div>
          <p className="text-gray-600">
            Connect to GitHub to manage repositories and create pull requests
          </p>
        </div>

        {/* Action buttons */}
        <div className="flex gap-3">
          {status?.installed ? (
            <>
              {showManageButton && (
                <Button 
                  variant="outline" 
                  onClick={handleManage} 
                  disabled={isLoading || disconnectMutation.isPending}
                >
                  Manage in GitHub
                </Button>
              )}
              <Button 
                variant="destructive" 
                onClick={handleDisconnect} 
                disabled={isLoading || disconnectMutation.isPending}
              >
                Disconnect
              </Button>
            </>
          ) : (
            <Button 
              onClick={handleConnect} 
              disabled={isLoading || !appSlug}
              className="bg-blue-600 hover:bg-blue-700 text-white"
            >
              Connect GitHub
            </Button>
          )}
        </div>
      </div>
    </Card>
  )
}

