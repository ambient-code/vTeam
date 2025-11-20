"use client";

import { useState } from "react";
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
import { useProjects, useCloneSession } from "@/services/queries";

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
  const [error, setError] = useState<string | null>(null);

  const { data: projects = [], isLoading: loadingProjects } = useProjects();
  const cloneSessionMutation = useCloneSession();

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      project: projectName || session.spec.project || "",
    },
  });

  const onSubmit = async (values: FormValues) => {
    setError(null);
    const targetProject = projectName || values.project;

    cloneSessionMutation.mutate(
      {
        projectName: targetProject,
        sessionName: session.metadata.name,
        data: {
          targetProject: targetProject,
          newSessionName: `${session.metadata.name}-clone-${Date.now()}`,
        },
      },
      {
        onSuccess: () => {
          setOpen(false);
          onSuccess?.();
        },
        onError: (err) => {
          setError(err.message || "Failed to clone session");
        },
      }
    );
  };

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
    if (!newOpen) {
      // Reset form and state when closing
      form.reset();
      setError(null);
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
                    disabled={loadingProjects || cloneSessionMutation.isPending}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue
                          placeholder={
                            loadingProjects
                              ? "Loading workspaces..."
                              : "Select a workspace"
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
                <p className="text-red-700 dark:text-red-300 text-sm">{error}</p>
              </div>
            )}

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setOpen(false)}
                disabled={cloneSessionMutation.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={cloneSessionMutation.isPending || (!projectName && loadingProjects)}>
                {cloneSessionMutation.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                {cloneSessionMutation.isPending ? "Cloning..." : "Clone Session"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
        </DialogContent>
      </Dialog>
    </>
  );
}