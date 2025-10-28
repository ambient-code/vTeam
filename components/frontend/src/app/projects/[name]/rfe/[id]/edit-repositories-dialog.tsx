"use client";

import React from "react";
import { useForm, useFieldArray } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Loader2, Plus, Trash2 } from "lucide-react";

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
import { Input } from "@/components/ui/input";
import type { RFEWorkflow } from "@/types/agentic-session";

const repoSchema = z.object({
  url: z.string().url("Please enter a valid repository URL"),
  branch: z.string().min(1),
});

const formSchema = z
  .object({
    umbrellaRepo: repoSchema,
    supportingRepos: z.array(repoSchema),
  })
  .refine(
    (data) => {
      // Check for duplicate repositories
      const allUrls: string[] = [];

      if (data.umbrellaRepo?.url) {
        allUrls.push(normalizeRepoUrl(data.umbrellaRepo.url));
      }

      const supportingUrls = (data.supportingRepos || [])
        .filter((r) => r?.url)
        .map((r) => normalizeRepoUrl(r.url));

      allUrls.push(...supportingUrls);

      const uniqueUrls = new Set(allUrls);
      return uniqueUrls.size === allUrls.length;
    },
    {
      message:
        "Duplicate repository URLs are not allowed. Each repository must be unique.",
      path: ["supportingRepos"],
    }
  );

type FormValues = z.infer<typeof formSchema>;

// Normalize repository URL for comparison (remove trailing slash and .git)
function normalizeRepoUrl(url: string): string {
  return url.trim().toLowerCase().replace(/\.git$/, "").replace(/\/$/, "");
}

type EditRepositoriesDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workflow: RFEWorkflow;
  onSave: (data: {
    umbrellaRepo: { url: string; branch?: string };
    supportingRepos: { url: string; branch?: string }[];
  }) => Promise<void>;
  isSaving: boolean;
};

export function EditRepositoriesDialog({
  open,
  onOpenChange,
  workflow,
  onSave,
  isSaving,
}: EditRepositoriesDialogProps) {
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onBlur",
    defaultValues: {
      umbrellaRepo: {
        url: workflow.umbrellaRepo?.url || "",
        branch: workflow.umbrellaRepo?.branch || "main",
      },
      supportingRepos: (workflow.supportingRepos || []).map(r => ({
        url: r.url,
        branch: r.branch || "main",
      })),
    },
  });

  // Reset form when dialog opens with new workflow data
  React.useEffect(() => {
    if (open) {
      form.reset({
        umbrellaRepo: {
          url: workflow.umbrellaRepo?.url || "",
          branch: workflow.umbrellaRepo?.branch || "main",
        },
        supportingRepos: (workflow.supportingRepos || []).map(r => ({
          url: r.url,
          branch: r.branch || "main",
        })),
      });
    }
  }, [open, workflow, form]);

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "supportingRepos",
  });

  const onSubmit = async (values: FormValues) => {
    await onSave({
      umbrellaRepo: {
        url: values.umbrellaRepo.url.trim(),
        branch: values.umbrellaRepo.branch?.trim() || "main",
      },
      supportingRepos: (values.supportingRepos || [])
        .filter((r) => r && r.url && r.url.trim() !== "")
        .map((r) => ({ url: r.url.trim(), branch: r.branch?.trim() || "main" })),
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Edit Repositories</DialogTitle>
          <DialogDescription>
            Update the repository URLs and branches. Make sure you have push access
            to all repositories.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
            {/* Spec Repository */}
            <div className="space-y-4">
              <div className="font-semibold text-sm">Spec Repository (Required)</div>

              <FormField
                control={form.control}
                name="umbrellaRepo.url"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Repository URL</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="https://github.com/username/repo"
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      Main repository for specs and planning documents
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="umbrellaRepo.branch"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Base Branch</FormLabel>
                    <FormControl>
                      <Input placeholder="main" {...field} />
                    </FormControl>
                    <FormDescription>
                      Feature branch will be created from this base branch
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            {/* Supporting Repositories */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="font-semibold text-sm">
                  Supporting Repositories (Optional)
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => append({ url: "", branch: "main" })}
                >
                  <Plus className="h-4 w-4 mr-1" />
                  Add Repository
                </Button>
              </div>

              {fields.length === 0 && (
                <p className="text-sm text-muted-foreground">
                  No supporting repositories. Click &ldquo;Add Repository&rdquo; to add one.
                </p>
              )}

              {fields.map((field, index) => (
                <div key={field.id} className="border rounded-lg p-4 space-y-4">
                  <div className="flex items-center justify-between">
                    <div className="font-medium text-sm">
                      Supporting Repository {index + 1}
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => remove(index)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>

                  <FormField
                    control={form.control}
                    name={`supportingRepos.${index}.url`}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Repository URL</FormLabel>
                        <FormControl>
                          <Input
                            placeholder="https://github.com/username/repo"
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name={`supportingRepos.${index}.branch`}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Base Branch</FormLabel>
                        <FormControl>
                          <Input placeholder="main" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              ))}
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={isSaving}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isSaving}>
                {isSaving ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Saving...
                  </>
                ) : (
                  "Save Changes"
                )}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
