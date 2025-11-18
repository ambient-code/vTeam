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
  const [manuallyEditedName, setManuallyEditedName] = useState(false);

  const generateWorkspaceName = (displayName: string): string => {
    return displayName
      .toLowerCase()
      .replace(/\s+/g, "-") // Replace spaces with hyphens
      .replace(/[^a-z0-9-]/g, "") // Remove invalid characters
      .replace(/-+/g, "-") // Collapse multiple hyphens
      .replace(/^-+|-+$/g, "") // Remove leading/trailing hyphens
      .slice(0, 63); // Truncate to max length
  };

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

  const handleDisplayNameChange = (displayName: string) => {
    setFormData((prev) => ({
      ...prev,
      displayName,
      // Auto-generate name only if it hasn't been manually edited
      name: manuallyEditedName ? prev.name : generateWorkspaceName(displayName),
    }));
    
    // Validate the auto-generated name
    if (!manuallyEditedName) {
      const generatedName = generateWorkspaceName(displayName);
      setNameError(validateProjectName(generatedName));
    }
  };

  // Commented out - name input field is hidden, auto-generated from displayName
  // const handleNameChange = (name: string) => {
  //   setManuallyEditedName(true);
  //   setFormData((prev) => ({ ...prev, name }));
  //   setNameError(validateProjectName(name));
  // };

  const resetForm = () => {
    setFormData({
      name: "",
      displayName: "",
      description: "",
    });
    setNameError(null);
    setError(null);
    setManuallyEditedName(false);
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
    if (isOpenShift && !formData.displayName?.trim()) {
      setError("Display Name is required");
      return;
    }

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
      <DialogContent className="w-[672px] max-w-[90vw] max-h-[90vh] overflow-y-auto">
        <DialogHeader className="space-y-3">
          <DialogTitle>Create New Workspace</DialogTitle>
          <DialogDescription>
            Set up a new workspace for your team
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-8 pt-2">
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
          <div className="space-y-6">

            {/* OpenShift-only fields */}
            {isOpenShift && (
              <div className="space-y-2">
                <Label htmlFor="displayName">Workspace Name *</Label>
                <Input
                  id="displayName"
                  value={formData.displayName}
                  onChange={(e) => handleDisplayNameChange(e.target.value)}
                  placeholder="e.g. My Research Workspace"
                  maxLength={100}
                />
              </div>
            )}

            {/* Vanilla Kubernetes name field */}
            {!isOpenShift && (
              <div className="space-y-2">
                <Label htmlFor="name">Workspace Name *</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => {
                    const name = e.target.value;
                    setFormData((prev) => ({ ...prev, name }));
                    setNameError(validateProjectName(name));
                  }}
                  placeholder="my-research-workspace"
                  className={nameError ? "border-red-500" : ""}
                />
                {nameError && <p className="text-sm text-red-600">{nameError}</p>}
                <p className="text-sm text-gray-600">
                  Lowercase alphanumeric with hyphens.
                </p>
              </div>
            )}

            {/* OpenShift-only description field */}
            {isOpenShift && (
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
              </div>
            )}
          </div>

          {error && (
            <div className="p-4 bg-red-50 border border-red-200 rounded-md">
              <p className="text-red-700">{error}</p>
            </div>
          )}

          <DialogFooter className="pt-2">
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

