'use client';

import { useState, useEffect } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import { Star, Settings, Users, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { PageHeader } from '@/components/page-header';
import { Breadcrumbs } from '@/components/breadcrumbs';

import { SessionsSection } from '@/components/workspace-sections/sessions-section';
import { SharingSection } from '@/components/workspace-sections/sharing-section';
import { SettingsSection } from '@/components/workspace-sections/settings-section';
import { useProject } from '@/services/queries/use-projects';

type Section = 'sessions' | 'sharing' | 'settings';

export default function ProjectDetailsPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const projectName = params?.name as string;
  
  // Fetch project data for display name and description
  const { data: project, isLoading: projectLoading } = useProject(projectName);
  
  // Initialize active section from query parameter or default to 'sessions'
  const initialSection = (searchParams.get('section') as Section) || 'sessions';
  const [activeSection, setActiveSection] = useState<Section>(initialSection);

  // Update active section when query parameter changes
  useEffect(() => {
    const sectionParam = searchParams.get('section') as Section;
    if (sectionParam && ['sessions', 'sharing', 'settings'].includes(sectionParam)) {
      setActiveSection(sectionParam);
    }
  }, [searchParams]);

  const navItems = [
    { id: 'sessions' as Section, label: 'Sessions', icon: Star },
    { id: 'sharing' as Section, label: 'Sharing', icon: Users },
    { id: 'settings' as Section, label: 'Workspace Settings', icon: Settings },
  ];

  // Loading state
  if (!projectName || projectLoading) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <Alert className="max-w-md mx-4">
            <Loader2 className="h-4 w-4 animate-spin" />
            <AlertTitle>Loading Workspace...</AlertTitle>
            <AlertDescription>
              <p>Please wait while the workspace is loading...</p>
            </AlertDescription>
          </Alert>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Sticky header */}
      <div className="sticky top-0 z-20 bg-card border-b">
        <div className="px-6 py-4">
          <Breadcrumbs
            items={[
              { label: 'Workspaces', href: '/projects' },
              { label: projectName },
            ]}
          />
        </div>
      </div>

      <div className="container mx-auto p-0">
        {/* Title and Description */}
        <div className="px-6 pt-6 pb-4">
          <PageHeader
            title={project?.displayName || projectName}
            description={project?.description || 'Workspace details and configuration'}
          />
        </div>

        {/* Divider */}
        <hr className="border-t mx-6 mb-6" />

        {/* Content */}
        <div className="px-6 flex gap-6">
          {/* Sidebar Navigation */}
          <aside className="w-56 shrink-0">
            <Card>
              <CardHeader>
                <CardTitle>Workspace</CardTitle>
              </CardHeader>
              <CardContent className="px-4 pb-4 pt-2">
                <div className="space-y-1">
                  {navItems.map((item) => {
                    const isActive = activeSection === item.id;
                    const Icon = item.icon;
                    return (
                      <Button
                        key={item.id}
                        variant={isActive ? "secondary" : "ghost"}
                        className={cn("w-full justify-start", isActive && "font-semibold")}
                        onClick={() => setActiveSection(item.id)}
                      >
                        <Icon className="w-4 h-4 mr-2" />
                        {item.label}
                      </Button>
                    );
                  })}
                </div>
              </CardContent>
            </Card>
          </aside>

          {/* Main Content */}
          {activeSection === 'sessions' && <SessionsSection projectName={projectName} />}
          {activeSection === 'sharing' && <SharingSection projectName={projectName} />}
          {activeSection === 'settings' && <SettingsSection projectName={projectName} />}
        </div>
      </div>
    </div>
  );
}
