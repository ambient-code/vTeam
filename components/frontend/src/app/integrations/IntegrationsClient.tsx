'use client'

import React from 'react'
import { GitHubConnectionCard } from '@/components/github-connection-card'

type Props = { appSlug?: string }

export default function IntegrationsClient({ appSlug }: Props) {
  return (
    <div className="max-w-3xl mx-auto p-6 space-y-6">
      <h1 className="text-2xl font-semibold">Integrations</h1>
      <GitHubConnectionCard appSlug={appSlug} showManageButton={true} />
    </div>
  )
}


