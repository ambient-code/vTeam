"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
// ProjectPhase removed from UI
import type { Project } from "@/types/agentic-session";
import { Plus, RefreshCw, Trash2 } from "lucide-react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { getApiUrl } from "@/lib/config";

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<{ [key: string]: string }>(
    {}
  );
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [projectPendingDelete, setProjectPendingDelete] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  // Search and phase filter removed per requirements

  const fetchProjects = async () => {
    try {
      const apiUrl = getApiUrl();
      const url = `${apiUrl}/projects`;

      const response = await fetch(url);
      if (!response.ok) {
        throw new Error("Failed to fetch projects");
      }
      const data: any = await response.json();
      const items = Array.isArray(data.items) ? data.items : [];
      setProjects(items);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProjects();
    // Poll for updates every 30 seconds (less frequent than sessions)
    const interval = setInterval(fetchProjects, 30000);
    return () => clearInterval(interval);
  }, []);

  const handleRefresh = () => {
    setLoading(true);
    fetchProjects();
  };

  const handleDelete = async (projectName: string) => {
    setActionLoading((prev) => ({ ...prev, [projectName]: "deleting" }));
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/projects/${projectName}`, {
        method: "DELETE",
      });
      if (!response.ok) {
        throw new Error("Failed to delete project");
      }
      await fetchProjects(); // Refresh the list
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete project");
    } finally {
      setActionLoading((prev) => {
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { [projectName]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const openDeleteDialog = (projectName: string) => {
    setProjectPendingDelete(projectName);
    setShowDeleteDialog(true);
  };

  const confirmDelete = async () => {
    if (!projectPendingDelete) return;
    setDeleting(true);
    try {
      await handleDelete(projectPendingDelete);
      setShowDeleteDialog(false);
      setProjectPendingDelete(null);
    } finally {
      setDeleting(false);
    }
  };

  if (loading && (!projects || projects.length === 0)) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="animate-spin h-8 w-8" />
          <span className="ml-2">Loading projects...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-0">
      <div className="sticky top-0 z-20 bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b">
        <div className="px-6 py-4 flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold">Projects</h1>
            <p className="text-sm text-muted-foreground">Manage your Ambient AI projects and configurations</p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={handleRefresh} disabled={loading}>
              <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
              Refresh
            </Button>
            <Link href="/projects/new">
              <Button>
                <Plus className="w-4 h-4 mr-2" />
                New Project
              </Button>
            </Link>
          </div>
        </div>
      </div>

      {/* Filters removed per requirements */}

      {error && (
        <div className="px-6">
          <Card className="mb-4 border-red-200 bg-red-50">
            <CardContent className="pt-6">
              <p className="text-red-700">Error: {error}</p>
            </CardContent>
          </Card>
        </div>
      )}

      <div className="px-6">
      <Card>
        <CardHeader>
          <CardTitle>Ambient Projects</CardTitle>
          <CardDescription>
            Configure and manage project settings, resource limits, and access controls
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!projects || projects.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-muted-foreground mb-4">
                No projects found
              </p>
              <Link href="/projects/new">
                <Button>
                  <Plus className="w-4 h-4 mr-2" />
                  Create your first project
                </Button>
              </Link>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="min-w-[200px]">Name</TableHead>
                    {/* Status column removed */}
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
                  {projects.map((project: Project) => (
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
                        <span className="truncate block" title={"—"}>
                          {project.description || "—"}
                        </span>
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        {project.creationTimestamp && (
                          formatDistanceToNow(
                            new Date(project.creationTimestamp),
                            {
                              addSuffix: true,
                            }
                          )
                        )}
                      </TableCell>
                     
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0"
                          onClick={() => openDeleteDialog(project.name)}
                          disabled={!!actionLoading[project.name]}
                        >
                          {actionLoading[project.name] ? (
                            <RefreshCw className="h-4 w-4 animate-spin" />
                          ) : (
                            <Trash2 className="h-4 w-4" />
                          )}
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
      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete project</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete project &quot;{projectPendingDelete}&quot;? This will permanently remove the project and all related resources. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDeleteDialog(false)} disabled={deleting}>Cancel</Button>
            <Button variant="destructive" onClick={confirmDelete} disabled={deleting}>
              {deleting ? <><RefreshCw className="w-4 h-4 mr-2 animate-spin" />Deleting...</> : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}