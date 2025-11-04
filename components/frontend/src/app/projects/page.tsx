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
        successToast(`Workspace "${projectToDelete.displayName || projectToDelete.name}" deleted successfully`);
        closeDeleteDialog();
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to delete workspace');
      },
    });
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="h-8 w-8 animate-spin" />
          <span className="ml-2">Loading workspaces...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-0">
      {/* Sticky header */}
      <div className="sticky top-0 z-20 bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b">
        <div className="px-6 py-4">
          <PageHeader
            title="Workspaces"
            description="Select a workspace or create a new one to get started"
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
                    New Workspace
                  </Button>
                </Link>
              </>
            }
          />
        </div>
      </div>

      {/* Error state */}
      {error && (
        <div className="px-6 pt-4">
          <ErrorMessage error={error} onRetry={() => refetch()} />
        </div>
      )}

      {/* Content */}
      <div className="px-6 pt-4">
        <Card>
          <CardHeader>
            <CardTitle>Workspaces</CardTitle>
            <CardDescription>
              Configure and manage workspace settings, resource limits, and access
              controls
            </CardDescription>
          </CardHeader>
          <CardContent>
            {projects.length === 0 ? (
              <EmptyState
                icon={FolderOpen}
                title="No workspaces found"
                description="Get started by creating your first workspace"
                action={{
                  label: 'Create Workspace',
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
                      <TableHead className="hidden sm:table-cell">
                        Sessions
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
                        <TableCell className="hidden sm:table-cell">
                          <div className="flex items-center gap-2">
                            <span className="text-muted-foreground">—</span>
                          </div>
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
        title="Delete workspace"
        description={`Are you sure you want to delete workspace "${projectToDelete?.name}"? This will permanently remove the workspace and all related resources. This action cannot be undone.`}
        confirmText="Delete"
        loading={deleteProjectMutation.isPending}
      />
    </div>
  );
}
