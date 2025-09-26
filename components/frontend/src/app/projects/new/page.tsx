"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { CreateProjectRequest } from "@/types/project";
import { ArrowLeft, Save, Loader2 } from "lucide-react";
import { getApiUrl } from "@/lib/config";

// Project type selection removed

export default function NewProjectPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [formData, setFormData] = useState<CreateProjectRequest>({
    name: "",
    displayName: "",
    description: "",
    // resourceQuota removed
  });

  const [nameError, setNameError] = useState<string | null>(null);

  const validateProjectName = (name: string) => {
    // Validate name pattern: ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
    const namePattern = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;

    if (!name) {
      return "Project name is required";
    }

    if (name.length > 63) {
      return "Project name must be 63 characters or less";
    }

    if (!namePattern.test(name)) {
      return "Project name must be lowercase alphanumeric with hyphens (cannot start or end with hyphen)";
    }

    return null;
  };

  const handleNameChange = (name: string) => {
    setFormData(prev => ({ ...prev, name }));
    setNameError(validateProjectName(name));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validate required fields
    const nameValidationError = validateProjectName(formData.name);
    if (nameValidationError) {
      setNameError(nameValidationError);
      return;
    }

    if (!formData.displayName.trim()) {
      setError("Display name is required");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const apiUrl = getApiUrl();

      // Prepare the request payload
      const payload: CreateProjectRequest = {
        name: formData.name,
        displayName: formData.displayName.trim(),
        ...(formData.description?.trim() && { description: formData.description.trim() }),
        // projectType and resourceQuota removed
      };

      const response = await fetch(`${apiUrl}/projects`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: "Unknown error" }));
        throw new Error(errorData.message || errorData.error || "Failed to create project");
      }

      const project = await response.json();
      const projectName = project?.name;
      if (!projectName) {
        throw new Error("Create project succeeded but response didn't include a project name");
      }
      router.push(`/projects/${projectName}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create project");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="container mx-auto p-6 max-w-2xl">
      <div className="flex items-center gap-4 mb-6">
        <Link href="/projects">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to Projects
          </Button>
        </Link>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Create New Project</CardTitle>
          <CardDescription>
            Create a new Ambient AI project with custom settings and resource configurations
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Basic Information */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold">Basic Information</h3>

              <div className="space-y-2">
                <Label htmlFor="name">Project Name *</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  placeholder="my-research-project"
                  className={nameError ? "border-red-500" : ""}
                />
                {nameError && (
                  <p className="text-sm text-red-600">{nameError}</p>
                )}
                <p className="text-sm text-gray-600">
                  Lowercase alphanumeric with hyphens. Will be used as the OpenShift namespace.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="displayName">Display Name *</Label>
                <Input
                  id="displayName"
                  value={formData.displayName}
                  onChange={(e) => setFormData(prev => ({ ...prev, displayName: e.target.value }))}
                  placeholder="My Research Project"
                  maxLength={100}
                />
                <p className="text-sm text-gray-600">
                  Human-readable name for the project (max 100 characters)
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={formData.description}
                  onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                  placeholder="Description of the project purpose and goals..."
                  maxLength={500}
                  rows={3}
                />
                <p className="text-sm text-gray-600">
                  Optional description (max 500 characters)
                </p>
              </div>

            {/* Project type removed */}
            </div>

            {/* Resource quota inputs removed */}

            {error && (
              <div className="p-4 bg-red-50 border border-red-200 rounded-md">
                <p className="text-red-700">{error}</p>
              </div>
            )}

            <div className="flex gap-4 pt-4">
              <Button type="submit" disabled={loading || !!nameError}>
                {loading ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Creating...
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4 mr-2" />
                    Create Project
                  </>
                )}
              </Button>
              <Link href="/projects">
                <Button type="button" variant="outline" disabled={loading}>
                  Cancel
                </Button>
              </Link>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}