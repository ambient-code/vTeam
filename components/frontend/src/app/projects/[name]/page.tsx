'use client';

import { useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { formatDistanceToNow } from 'date-fns';
import { RefreshCw } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { ProjectSubpageHeader } from '@/components/project-subpage-header';
import { ErrorMessage } from '@/components/error-message';
import { Breadcrumbs } from '@/components/breadcrumbs';

import { useProject } from '@/services/queries';

export default function ProjectDetailsPage() {
  const params = useParams();
  const router = useRouter();
  const projectName = params?.name as string;

  // React Query hook replaces all manual state management
  const { data: project, isLoading, error, refetch } = useProject(projectName);

  const handleRefresh = useCallback(() => {
    refetch();
  }, [refetch]);

  // Loading state
  if (!projectName || (isLoading && !project)) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="animate-spin h-8 w-8" />
          <span className="ml-2">Loading project...</span>
        </div>
      </div>
    );
  }

  // Error state (no project loaded)
  if (error && !project) {
    return (
      <div className="container mx-auto p-6">
        <Card className="border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <p className="text-red-700">{error instanceof Error ? error.message : 'Failed to load project'}</p>
            <div className="mt-4 flex gap-4">
              <Button onClick={() => router.push('/projects')} variant="outline">
                Back to Projects
              </Button>
              <Button onClick={handleRefresh}>Try Again</Button>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!project) return null;

  return (
    <div className="container mx-auto p-6">
      <Breadcrumbs
        items={[
          { label: 'Projects', href: '/projects' },
          { label: project.displayName || project.name },
        ]}
        className="mb-4"
      />
      <ProjectSubpageHeader
        title={<>{project.displayName || project.name}</>}
        description={<>{projectName}</>}
        actions={
          <Button variant="outline" onClick={handleRefresh} disabled={isLoading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        }
      />

      {/* Error state (with project loaded) */}
      {error && project && (
        <div className="px-6">
          <ErrorMessage error={error} onRetry={handleRefresh} />
        </div>
      )}

      <div className="pt-2">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Project Info */}
          <Card>
            <CardHeader>
              <CardTitle>Project Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label className="text-sm font-medium">Description</Label>
                <p className="text-sm text-muted-foreground">
                  {project.description || 'No description provided'}
                </p>
              </div>
              <div>
                <Label className="text-sm font-medium">Created</Label>
                <p className="text-sm text-muted-foreground">
                  {project.creationTimestamp &&
                    formatDistanceToNow(new Date(project.creationTimestamp), {
                      addSuffix: true,
                    })}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
