"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { CreateProjectRequest } from "@/types/project";
import { Save, Loader2, Info } from "lucide-react";
import { successToast, errorToast } from "@/hooks/use-toast";
import { useCreateProject } from "@/services/queries";
import { useClusterInfo } from "@/hooks/use-cluster-info";
import { Alert, AlertDescription } from "@/components/ui/alert";

type CreateWorkspaceDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function CreateWorkspaceDialog({
  open,
  onOpenChange,
}: CreateWorkspaceDialogProps) {
  const router = useRouter();
  const createProjectMutation = useCreateProject();
  const { isOpenShift, isLoading: clusterLoading } = useClusterInfo();
  const [error, setError] = useState<string | null>(null);
  const [formData, setFormData] = useState<CreateProjectRequest>({
    name: "",
    displayName: "",
    description: "",
  });

  const [nameError, setNameError] = useState<string | null>(null);

  const validateProjectName = (name: string) => {
    // Validate name pattern: ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
    const namePattern = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;

    if (!name) {
      return "Workspace name is required";
    }

    if (name.length > 63) {
      return "Workspace name must be 63 characters or less";
    }

    if (!namePattern.test(name)) {
      return "Workspace name must be lowercase alphanumeric with hyphens (cannot start or end with hyphen)";
    }

    return null;
  };

  const handleNameChange = (name: string) => {
    setFormData((prev) => ({ ...prev, name }));
    setNameError(validateProjectName(name));
  };

  const resetForm = () => {
    setFormData({
      name: "",
      displayName: "",
      description: "",
    });
    setNameError(null);
    setError(null);
  };

  const handleClose = () => {
    if (!createProjectMutation.isPending) {
      resetForm();
      onOpenChange(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validate required fields
    const nameValidationError = validateProjectName(formData.name);
    if (nameValidationError) {
      setNameError(nameValidationError);
      return;
    }

    setError(null);

    // Prepare the request payload
    const payload: CreateProjectRequest = {
      name: formData.name,
      // Only include displayName and description on OpenShift
      ...(isOpenShift &&
        formData.displayName?.trim() && {
          displayName: formData.displayName.trim(),
        }),
      ...(isOpenShift &&
        formData.description?.trim() && {
          description: formData.description.trim(),
        }),
    };

    createProjectMutation.mutate(payload, {
      onSuccess: (project) => {
        successToast(
          `Workspace "${formData.displayName || formData.name}" created successfully`
        );
        resetForm();
        onOpenChange(false);
        router.push(`/projects/${encodeURIComponent(project.name)}`);
      },
      onError: (err) => {
        const message =
          err instanceof Error ? err.message : "Failed to create workspace";
        setError(message);
        errorToast(message);
      },
    });
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create New Workspace</DialogTitle>
          <DialogDescription>
            Create a new Ambient AI workspace with custom settings and resource
            configurations
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Cluster info banner */}
          {!clusterLoading && !isOpenShift && (
            <Alert>
              <Info className="h-4 w-4" />
              <AlertDescription>
                Running on vanilla Kubernetes. Display name and description
                fields are not available.
              </AlertDescription>
            </Alert>
          )}

          {/* Basic Information */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold">Basic Information</h3>

            <div className="space-y-2">
              <Label htmlFor="name">Workspace Name *</Label>
              <Input
                id="name"
                value={formData.name}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder="my-research-workspace"
                className={nameError ? "border-red-500" : ""}
              />
              {nameError && <p className="text-sm text-red-600">{nameError}</p>}
              <p className="text-sm text-gray-600">
                Lowercase alphanumeric with hyphens. Will be used as the
                Kubernetes namespace.
              </p>
            </div>

            {/* OpenShift-only fields */}
            {isOpenShift && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="displayName">Display Name</Label>
                  <Input
                    id="displayName"
                    value={formData.displayName}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        displayName: e.target.value,
                      }))
                    }
                    placeholder="My Research Workspace"
                    maxLength={100}
                  />
                  <p className="text-sm text-gray-600">
                    Human-readable name for the workspace (max 100 characters).
                    Defaults to workspace name if empty.
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="description">Description</Label>
                  <Textarea
                    id="description"
                    value={formData.description}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        description: e.target.value,
                      }))
                    }
                    placeholder="Description of the workspace purpose and goals..."
                    maxLength={500}
                    rows={3}
                  />
                  <p className="text-sm text-gray-600">
                    Optional description (max 500 characters)
                  </p>
                </div>
              </>
            )}
          </div>

          {error && (
            <div className="p-4 bg-red-50 border border-red-200 rounded-md">
              <p className="text-red-700">{error}</p>
            </div>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={createProjectMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={createProjectMutation.isPending || !!nameError}
            >
              {createProjectMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <Save className="w-4 h-4 mr-2" />
                  Create Workspace
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

