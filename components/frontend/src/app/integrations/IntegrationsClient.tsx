'use client'

import React from 'react'
import { GitHubConnectionCard } from '@/components/github-connection-card'
import { PageHeader } from '@/components/page-header'

type Props = { appSlug?: string }

export default function IntegrationsClient({ appSlug }: Props) {
  return (
    <div className="min-h-screen bg-[#f8fafc]">
      {/* Sticky header */}
      <div className="sticky top-0 z-20 bg-white border-b">
        <div className="container mx-auto px-6 py-6">
          <PageHeader
            title="Integrations"
            description="Connect Ambient Code Platform with your favorite tools and services"
          />
        </div>
      </div>

      <div className="container mx-auto p-0">
        {/* Content */}
        <div className="px-6 pt-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <GitHubConnectionCard appSlug={appSlug} showManageButton={true} />
          </div>
        </div>
      </div>
    </div>
  )
}


