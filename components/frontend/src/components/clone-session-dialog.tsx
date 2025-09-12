"use client";

import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Copy, Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { AgenticSession } from "@/types/agentic-session";
import type { Project } from "@/types/project";
import { getApiUrl } from "@/lib/config";

const formSchema = z.object({
  project: z.string().min(1, "Please select a project"),
});

type FormValues = z.infer<typeof formSchema>;

type CloneSessionDialogProps = {
  session: AgenticSession;
  trigger: React.ReactNode;
  onSuccess?: () => void;
  projectName?: string; // when provided, hide selector and use this project
}

export function CloneSessionDialog({
  session,
  trigger,
  onSuccess,
  projectName,
}: CloneSessionDialogProps) {
  const [open, setOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [projects, setProjects] = useState<Project[]>([]);
  const [loadingProjects, setLoadingProjects] = useState(!projectName);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      project: projectName || session.spec.project || "",
    },
  });

  // Fetch projects when dialog opens, unless projectName is fixed
  useEffect(() => {
    if (open && !projectName) {
      const fetchProjects = async () => {
        try {
          const apiUrl = getApiUrl();
          const response = await fetch(`${apiUrl}/projects`);
          if (response.ok) {
            const data = await response.json();
            setProjects(Array.isArray(data.items) ? data.items : []);
          } else {
            console.error('Failed to fetch projects');
            setError('Failed to load projects');
          }
        } catch (err) {
          console.error('Error fetching projects:', err);
          setError('Failed to load projects');
        } finally {
          setLoadingProjects(false);
        }
      };

      fetchProjects();
    }
  }, [open, projectName]);

  const onSubmit = async (values: FormValues) => {
    setIsSubmitting(true);
    setError(null);

    try {
      const cloneRequest = {
        targetProject: projectName || values.project,
        newSessionName: session.metadata.name,
      };

      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName || values.project)}/agentic-sessions/${encodeURIComponent(session.metadata.name)}/clone`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(cloneRequest),
      });

      if (!response.ok) {
        const errorData = await response
          .json()
          .catch(() => ({ message: "Unknown error" }));
        throw new Error(
          errorData.message || "Failed to clone agentic session"
        );
      }

      // Success - close dialog and notify parent
      setOpen(false);
      onSuccess?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
    if (!newOpen) {
      // Reset form and state when closing
      form.reset();
      setError(null);
      setLoadingProjects(!projectName);
    }
  };

  const handleTriggerClick = () => {
    setOpen(true);
  };

  return (
    <>
      <div onClick={handleTriggerClick}>{trigger}</div>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle className="flex items-center">
              <Copy className="w-5 h-5 mr-2" />
              Clone Session
            </DialogTitle>
            <DialogDescription>
              {projectName
                ? `Clone "${session.spec.displayName || session.metadata.name}" into this project.`
                : `Clone "${session.spec.displayName || session.metadata.name}" to a target project.`}
            </DialogDescription>
          </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            {!projectName && (
            <FormField
              control={form.control}
              name="project"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Target Project</FormLabel>
                  <Select
                    onValueChange={field.onChange}
                    defaultValue={field.value}
                    disabled={loadingProjects || isSubmitting}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue
                          placeholder={
                            loadingProjects
                              ? "Loading projects..."
                              : "Select a project"
                          }
                        />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {projects.map((project) => (
                      <SelectItem key={project.name} value={project.name}>
                          {project.displayName || project.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    Select the project where the cloned session will be created
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />)}

            {error && (
              <div className="bg-red-50 border border-red-200 rounded-md p-3">
                <p className="text-red-700 text-sm">{error}</p>
              </div>
            )}

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setOpen(false)}
                disabled={isSubmitting}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting || (!projectName && loadingProjects)}>
                {isSubmitting && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                {isSubmitting ? "Cloning..." : "Clone Session"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
        </DialogContent>
      </Dialog>
    </>
  );
}