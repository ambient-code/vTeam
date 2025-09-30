"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { formatDistanceToNow } from "date-fns";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
// Status/phase UI removed
import { Label } from "@/components/ui/label";
// Removed inline edit form; editing moved to Settings page
import { Project } from "@/types/project";
import { RefreshCw } from "lucide-react";
import { getApiUrl } from "@/lib/config";
import { ProjectSubpageHeader } from "@/components/project-subpage-header";

// Project type selection removed

export default function ProjectDetailsPage({ params }: { params: Promise<{ name: string }> }) {
  const router = useRouter();
  // Sessions state
  const [sessions, setSessions] = useState<any[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState<boolean>(true);
  const [sessionActionLoading, setSessionActionLoading] = useState<{ [key: string]: string }>({});
  const [projectName, setProjectName] = useState<string>('');
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // tabs removed; this page is Overview only

  // Form state for editing
  const [formData, setFormData] = useState({
    displayName: "",
    description: "",
    // projectType removed
  });

  const fetchProject = async () => {
    if (!projectName) return;
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/projects/${projectName}`);
      if (!response.ok) {
        if (response.status === 404) {
          throw new Error("Project not found");
        }
        throw new Error("Failed to fetch project");
      }
      const data: Project = await response.json();
      setProject(data);

      // Update form data
      setFormData({
        displayName: data.displayName || "",
        description: data.description || "",
        // projectType removed
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    params.then(({ name }) => setProjectName(name));
  }, [params]);

  // tabs removed

  useEffect(() => {
    if (projectName) {
      fetchProject();
      // Poll for updates every 30 seconds
      const interval = setInterval(fetchProject, 30000);
      return () => clearInterval(interval);
    }
  }, [projectName]);

  // Sessions fetching
  const fetchSessions = async () => {
    if (!projectName) return;
    try {
      setSessionsLoading(true);
      const apiUrl = getApiUrl();
      const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions`);
      if (!res.ok) throw new Error("Failed to fetch sessions");
      const data = await res.json();
      setSessions(Array.isArray(data?.items) ? data.items : []);
    } catch (e) {
      // keep project error separate; optionally show in tab content
    } finally {
      setSessionsLoading(false);
    }
  };

  useEffect(() => {
    if (projectName) {
      fetchSessions();
      const i = setInterval(fetchSessions, 10000);
      return () => clearInterval(i);
    }
  }, [projectName]);

  const handleSessionStop = async (sessionName: string) => {
    setSessionActionLoading((prev) => ({ ...prev, [sessionName]: "stopping" }));
    try {
      const apiUrl = getApiUrl();
      const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/stop`, { method: 'POST' });
      if (!res.ok) throw new Error("Failed to stop session");
      await fetchSessions();
    } finally {
      setSessionActionLoading((prev) => {
        const { [sessionName]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleSessionRestart = async (sessionName: string) => {
    setSessionActionLoading((prev) => ({ ...prev, [sessionName]: "restarting" }));
    try {
      const apiUrl = getApiUrl();
      const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/start`, { method: 'POST' });
      if (!res.ok) throw new Error("Failed to restart session");
      await fetchSessions();
    } finally {
      setSessionActionLoading((prev) => {
        const { [sessionName]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleSessionDelete = async (sessionName: string) => {
    if (!confirm(`Delete agentic session "${sessionName}"? This action cannot be undone.`)) return;
    setSessionActionLoading((prev) => ({ ...prev, [sessionName]: "deleting" }));
    try {
      const apiUrl = getApiUrl();
      const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}`, { method: 'DELETE' });
      if (!res.ok) throw new Error("Failed to delete session");
      await fetchSessions();
    } finally {
      setSessionActionLoading((prev) => {
        const { [sessionName]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleSave = async () => {
    if (!project) return;

    setSaving(true);
    setError(null);

    try {
      const apiUrl = getApiUrl();

      // Prepare the update payload
      const updatedProject = {
        name: project.name,
        displayName: formData.displayName.trim(),
        description: formData.description.trim() || undefined,
        annotations: {
          ...project.annotations,
        },
      } as Partial<Project> & { name: string };

      const response = await fetch(`${apiUrl}/projects/${projectName}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(updatedProject),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: "Unknown error" }));
        throw new Error(errorData.message || errorData.error || "Failed to update project");
      }

      const updatedData = await response.json();
      setProject(updatedData);

      // Show success feedback (you could add a toast notification here)
      console.log("Project updated successfully");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update project");
    } finally {
      setSaving(false);
    }
  };

  const handleRefresh = () => {
    setLoading(true);
    fetchProject();
  };

  if (!projectName || (loading && !project)) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="animate-spin h-8 w-8" />
          <span className="ml-2">Loading project...</span>
        </div>
      </div>
    );
  }

  if (error && !project) {
    return (
      <div className="container mx-auto p-6">
        <Card className="border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <p className="text-red-700">{error}</p>
            <div className="mt-4 flex gap-4">
              <Button onClick={() => router.push("/projects")} variant="outline">
                Back to Projects
              </Button>
              <Button onClick={fetchProject}>
                Try Again
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!project) return null;

  return (
    <div className="container mx-auto p-6">
      <ProjectSubpageHeader
        title={<>{project.displayName || project.name}</>}
        description={<>{projectName}</>}
        actions={
          <Button variant="outline" onClick={handleRefresh} disabled={loading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        }
      />

      {error && (
        <div className="px-6">
          <Card className="mb-4 border-red-200 bg-red-50">
            <CardContent className="pt-6">
              <p className="text-red-700">Error: {error}</p>
            </CardContent>
          </Card>
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
                    {project.description || "No description provided"}
                  </p>
                </div>
                {/* Project type removed */}
                <div>
                  <Label className="text-sm font-medium">Created</Label>
                  <p className="text-sm text-muted-foreground">
                    {project.creationTimestamp &&
                      formatDistanceToNow(new Date(project.creationTimestamp), {
                        addSuffix: true,
                      })}
                  </p>
                </div>
                {/* Last Reconciled not available in DTO */}
              </CardContent>
            </Card>

          
            {/* Status Conditions removed */}
          </div>

          {/* Editing moved to Settings page */}
      </div>
    </div>
  );
}