'use client';

import { useCallback } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { Plus, RefreshCw, MoreVertical, FileText } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { ProjectSubpageHeader } from '@/components/project-subpage-header';
import { EmptyState } from '@/components/empty-state';
import { ErrorMessage } from '@/components/error-message';
import { Breadcrumbs } from '@/components/breadcrumbs';

import { useRfeWorkflows } from '@/services/queries';
import type { WorkflowPhase } from '@/types/api';

const phaseLabel: Record<WorkflowPhase, string> = {
  pre: 'Pre',
  ideate: 'Ideate',
  specify: 'Specify',
  plan: 'Plan',
  tasks: 'Tasks',
  implement: 'Implement',
  review: 'Review',
  completed: 'Completed',
};

export default function ProjectRFEListPage() {
  const params = useParams();
  const projectName = params?.name as string;

  // React Query hook
  const { data: workflows = [], isLoading, error, refetch } = useRfeWorkflows(projectName);

  const handleRefresh = useCallback(() => {
    refetch();
  }, [refetch]);

  // Loading state
  if (!projectName || (isLoading && workflows.length === 0)) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="animate-spin h-8 w-8" />
          <span className="ml-2">Loading workflows...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Breadcrumbs
        items={[
          { label: 'Projects', href: '/projects' },
          { label: projectName, href: `/projects/${projectName}` },
          { label: 'RFE Workspaces' },
        ]}
        className="mb-4"
      />
      <ProjectSubpageHeader
        title={<>RFE Workspaces</>}
        description={<>Feature refinement workflows scoped to this project</>}
        actions={
          <>
            <Link href={`/projects/${encodeURIComponent(projectName)}/rfe/new`}>
              <Button>
                <Plus className="w-4 h-4 mr-2" />
                New Workspace
              </Button>
            </Link>
            <Button
              variant="outline"
              onClick={handleRefresh}
              disabled={isLoading}
            >
              <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
          </>
        }
      />

      {/* Error state */}
      {error && (
        <ErrorMessage error={error} onRetry={handleRefresh} />
      )}

      <Card>
        <CardHeader>
          <CardTitle>RFE Workspaces ({workflows?.length || 0})</CardTitle>
          <CardDescription>Workflows scoped to this project</CardDescription>
        </CardHeader>
        <CardContent>
          {workflows.length === 0 ? (
            <EmptyState
              icon={FileText}
              title="No RFE workspaces yet"
              description="Create your first feature refinement workflow"
              action={{
                label: 'Create Workflow',
                onClick: () =>
                  (window.location.href = `/projects/${encodeURIComponent(projectName)}/rfe/new`),
              }}
            />
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="min-w-[220px]">Name</TableHead>
                    <TableHead>Phase</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="hidden xl:table-cell">Created</TableHead>
                    <TableHead className="w-[50px]">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {workflows.map((workflow) => (
                    <TableRow key={workflow.id}>
                      <TableCell className="font-medium min-w-[220px]">
                        <Link
                          href={`/projects/${encodeURIComponent(projectName)}/rfe/${workflow.id}`}
                          className="text-blue-600 hover:underline hover:text-blue-800 transition-colors block"
                        >
                          <div>
                            <div className="font-medium">{workflow.title}</div>
                            <div className="text-xs text-gray-500 font-normal">
                              {workflow.id}
                            </div>
                          </div>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">
                          {phaseLabel[(workflow.currentPhase || 'pre') as WorkflowPhase]}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">
                          {workflow.status || '—'}
                        </span>
                      </TableCell>
                      <TableCell className="hidden xl:table-cell">
                        {workflow.createdAt
                          ? formatDistanceToNow(new Date(workflow.createdAt), {
                              addSuffix: true,
                            })
                          : '—'}
                      </TableCell>
                      <TableCell>
                        <Link
                          href={`/projects/${encodeURIComponent(projectName)}/rfe/${workflow.id}`}
                        >
                          <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                            <MoreVertical className="h-4 w-4" />
                          </Button>
                        </Link>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
