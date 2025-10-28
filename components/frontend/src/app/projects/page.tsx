'use client';

import { useState } from 'react';
import Link from 'next/link';
import { formatDistanceToNow } from 'date-fns';
import { Plus, RefreshCw, Trash2, FolderOpen } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useProjects, useDeleteProject } from '@/services/queries';
import { PageHeader } from '@/components/page-header';
import { EmptyState } from '@/components/empty-state';
import { ErrorMessage } from '@/components/error-message';
import { DestructiveConfirmationDialog } from '@/components/confirmation-dialog';
import { successToast, errorToast } from '@/hooks/use-toast';
import type { Project } from '@/types/api';

export default function ProjectsPage() {
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [projectToDelete, setProjectToDelete] = useState<Project | null>(null);

  // React Query hooks
  const { data: projects = [], isLoading, error, refetch } = useProjects();
  const deleteProjectMutation = useDeleteProject();

  const handleRefreshClick = () => {
    refetch();
  };

  const openDeleteDialog = (project: Project) => {
    setProjectToDelete(project);
    setShowDeleteDialog(true);
  };

  const closeDeleteDialog = () => {
    setShowDeleteDialog(false);
    setProjectToDelete(null);
  };

  const confirmDelete = async () => {
    if (!projectToDelete) return;

    deleteProjectMutation.mutate(projectToDelete.name, {
      onSuccess: () => {
        successToast(`Project "${projectToDelete.displayName || projectToDelete.name}" deleted successfully`);
        closeDeleteDialog();
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to delete project');
      },
    });
  };

  // Special handling for 403 errors on vanilla Kubernetes (user lacks cluster-wide namespace list permission)
  const is403Error = error && (error as any).message?.includes('403');

  // Loading state
  if (isLoading) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="h-8 w-8 animate-spin" />
          <span className="ml-2">Loading projects...</span>
        </div>
      </div>
    );
  }

  // Error state (non-403 errors)
  if (error && !is403Error) {
    return (
      <div className="container mx-auto p-6">
        <ErrorMessage error={error} onRetry={() => refetch()} />
      </div>
    );
  }

  return (
    <div className="container mx-auto p-0">
      {/* Sticky header */}
      <div className="sticky top-0 z-20 bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b">
        <div className="px-6 py-4">
          <PageHeader
            title="Projects"
            description="Manage your Ambient AI projects and configurations"
            actions={
              <>
                <Button
                  variant="outline"
                  onClick={handleRefreshClick}
                  disabled={isLoading}
                >
                  <RefreshCw
                    className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`}
                  />
                  Refresh
                </Button>
                <Link href="/projects/new">
                  <Button>
                    <Plus className="w-4 h-4 mr-2" />
                    New Project
                  </Button>
                </Link>
              </>
            }
          />
        </div>
      </div>

      {/* Special 403 error state for vanilla Kubernetes */}
      {is403Error && (
        <div className="px-6 pt-4">
          <Card className="border-amber-200 bg-amber-50">
            <CardContent className="pt-6">
              <div className="flex items-start gap-3">
                <div className="text-amber-600">
                  <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                </div>
                <div className="flex-1">
                  <h3 className="font-semibold text-amber-900">Insufficient Permissions</h3>
                  <p className="text-sm text-amber-800 mt-1">
                    You don't have permissions to list all namespaces in the cluster. 
                    On vanilla Kubernetes, listing projects requires cluster-wide namespace list permissions.
                    Please contact your administrator to grant you access or create a project using the button above.
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Content */}
      <div className="px-6 pt-4">
        <Card>
          <CardHeader>
            <CardTitle>Ambient Projects</CardTitle>
            <CardDescription>
              Configure and manage project settings, resource limits, and access
              controls
            </CardDescription>
          </CardHeader>
          <CardContent>
            {projects.length === 0 ? (
              <EmptyState
                icon={FolderOpen}
                title="No projects found"
                description="Get started by creating your first project"
                action={{
                  label: 'Create Project',
                  onClick: () => (window.location.href = '/projects/new'),
                }}
              />
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="min-w-[200px]">Name</TableHead>
                      <TableHead className="hidden md:table-cell">
                        Description
                      </TableHead>
                      <TableHead className="hidden lg:table-cell">
                        Created
                      </TableHead>
                      <TableHead className="w-[50px]">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {projects.map((project) => (
                      <TableRow key={project.name}>
                        <TableCell className="font-medium min-w-[200px]">
                          <Link
                            href={`/projects/${project.name}`}
                            className="text-blue-600 hover:underline hover:text-blue-800 transition-colors block"
                          >
                            <div>
                              <div className="font-medium">
                                {project.displayName || project.name}
                              </div>
                              <div className="text-xs text-gray-500 font-normal">
                                {project.name}
                              </div>
                            </div>
                          </Link>
                        </TableCell>
                        <TableCell className="hidden md:table-cell max-w-[200px]">
                          <span
                            className="truncate block"
                            title={project.description || '—'}
                          >
                            {project.description || '—'}
                          </span>
                        </TableCell>
                        <TableCell className="hidden lg:table-cell">
                          {project.creationTimestamp &&
                            formatDistanceToNow(
                              new Date(project.creationTimestamp),
                              { addSuffix: true }
                            )}
                        </TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 w-8 p-0"
                            onClick={() => openDeleteDialog(project)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
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

      {/* Delete confirmation dialog */}
      <DestructiveConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        onConfirm={confirmDelete}
        title="Delete project"
        description={`Are you sure you want to delete project "${projectToDelete?.name}"? This will permanently remove the project and all related resources. This action cannot be undone.`}
        confirmText="Delete"
        loading={deleteProjectMutation.isPending}
      />
    </div>
  );
}
