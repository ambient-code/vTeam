"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Loader2 } from "lucide-react";
import { useForm, useFieldArray } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Textarea } from "@/components/ui/textarea";
import type { CreateAgenticSessionRequest } from "@/types/agentic-session";
import { Checkbox } from "@/components/ui/checkbox";
import { successToast, errorToast } from "@/hooks/use-toast";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { RepositoryDialog } from "./repository-dialog";
import { RepositoryList } from "./repository-list";
import { ModelConfiguration } from "./model-configuration";
import { useCreateSession } from "@/services/queries/use-sessions";

const formSchema = z
  .object({
    prompt: z.string(),
    model: z.string().min(1, "Please select a model"),
    temperature: z.number().min(0).max(2),
    maxTokens: z.number().min(100).max(8000),
    timeout: z.number().min(60).max(1800),
    interactive: z.boolean().default(false),
    // Unified multi-repo array
    repos: z
      .array(z.object({
        input: z.object({ url: z.string().url(), branch: z.string().optional() }),
        output: z.object({ url: z.string().url().optional().or(z.literal("")), branch: z.string().optional() }).optional(),
      }))
      .optional()
      .default([]),
    mainRepoIndex: z.number().optional().default(0),
    // Runner behavior
    autoPushOnComplete: z.boolean().default(false),
    // storage paths are not user-configurable anymore
    agentPersona: z.string().optional(),
  })
  .superRefine((data, ctx) => {
    const isInteractive = Boolean(data.interactive);
    const promptLength = (data.prompt || "").trim().length;
    if (!isInteractive && promptLength < 10) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["prompt"],
        message: "Prompt must be at least 10 characters long",
      });
    }
  });

type FormValues = z.input<typeof formSchema>;

export default function NewProjectSessionPage({ params }: { params: Promise<{ name: string }> }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [projectName, setProjectName] = useState<string>("");
  const [prefillWorkspacePath, setPrefillWorkspacePath] = useState<string | undefined>(undefined);
  const [editingRepoIndex, setEditingRepoIndex] = useState<number | null>(null);
  const [repoDialogOpen, setRepoDialogOpen] = useState(false);
  const [tempRepo, setTempRepo] = useState<{ input: { url: string; branch: string }; output?: { url: string; branch: string } }>({ input: { url: "", branch: "main" } });

  // React Query hooks
  const createSessionMutation = useCreateSession();

  useEffect(() => {
    params.then(({ name }) => setProjectName(name));
  }, [params]);

  useEffect(() => {
    const ws = searchParams?.get("workspacePath");
    if (ws) setPrefillWorkspacePath(ws);
  }, [searchParams]);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      prompt: "",
      model: "claude-sonnet-4-5",
      temperature: 0.7,
      maxTokens: 4000,
      timeout: 300,
      interactive: false,
      autoPushOnComplete: false,
      agentPersona: "",
      repos: [],
      mainRepoIndex: 0,
    },
  });

  // Field arrays for multi-repo configuration
  const { fields: reposFields, append: appendRepo, remove: removeRepo, update: updateRepo } = useFieldArray({ control: form.control, name: "repos" });

  // Watch interactive to adjust prompt field hints
  const isInteractive = form.watch("interactive");



  

  const onSubmit = async (values: FormValues) => {
    if (!projectName) return;

    const promptToSend = values.interactive && !values.prompt.trim()
      ? "Greet the user and briefly explain the workspace capabilities: they can select workflows, add code repositories for context, use commands, and you'll help with software engineering tasks. Keep it friendly and concise."
      : values.prompt;
    const request: CreateAgenticSessionRequest = {
      prompt: promptToSend,
      llmSettings: {
        model: values.model,
        temperature: values.temperature,
        maxTokens: values.maxTokens,
      },
      timeout: values.timeout,
      interactive: values.interactive,
      autoPushOnComplete: values.autoPushOnComplete,
      };

      if (prefillWorkspacePath) {
        request.workspacePath = prefillWorkspacePath;
      }

      // Apply labels if projectName is present
      if (projectName) {
        request.labels = {
          ...(request.labels || {}),
          project: projectName,
        };
      }


      // Multi-repo configuration
      type RepoConfig = { input: { url: string; branch?: string }; output?: { url: string; branch?: string } };
      const repos = (values.repos as RepoConfig[] | undefined) || [];
      if (Array.isArray(repos) && repos.length > 0) {
        const filteredRepos = repos.filter(r => r && r.input && r.input.url);
        (request as CreateAgenticSessionRequest & { repos?: RepoConfig[]; mainRepoIndex?: number }).repos = filteredRepos;
        (request as CreateAgenticSessionRequest & { repos?: RepoConfig[]; mainRepoIndex?: number }).mainRepoIndex = values.mainRepoIndex || 0;

        // Ensure runner env receives repos JSON + main repo index for immediate compatibility
        request.environmentVariables = {
          ...(request.environmentVariables || {}),
          REPOS_JSON: JSON.stringify(filteredRepos),
          MAIN_REPO_INDEX: String(values.mainRepoIndex || 0),
        };
      }

    createSessionMutation.mutate(
      { projectName, data: request },
      {
        onSuccess: (session) => {
          const sessionName = session.metadata.name;
          successToast(`Session "${sessionName}" created successfully`);
          router.push(`/projects/${encodeURIComponent(projectName)}/sessions/${sessionName}`);
        },
        onError: (error) => {
          errorToast(error.message || "Failed to create session");
        },
      }
    );
  };

  return (
    <div className="container mx-auto p-6">
      <Breadcrumbs
        items={[
          { label: 'Projects', href: '/projects' },
          { label: projectName, href: `/projects/${projectName}` },
          { label: 'Sessions', href: `/projects/${projectName}/sessions` },
          { label: 'New Session' },
        ]}
        className="mb-4"
      />

      <Card>
        <CardHeader>
          <CardTitle>New Agentic Session</CardTitle>
          <CardDescription>Create a new agentic session that will analyze a website</CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              <FormField
                control={form.control}
                name="interactive"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-3">
                    <FormControl>
                      <Checkbox checked={field.value} onCheckedChange={(v) => field.onChange(Boolean(v))} />
                    </FormControl>
                    <div className="space-y-1 leading-none">
                      <FormLabel>Interactive chat</FormLabel>
                      <FormDescription>
                        When enabled, the session runs in chat mode. You can send messages and receive streamed responses.
                      </FormDescription>
                    </div>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {!isInteractive && (
                <FormField
                  control={form.control}
                  name="prompt"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Agentic Prompt</FormLabel>
                      <FormControl>
                        <Textarea placeholder="Describe what you want Claude to analyze on the website..." className="min-h-[100px]" {...field} />
                      </FormControl>
                      <FormDescription>Provide a detailed prompt about what you want Claude to analyze on the website</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}


              <ModelConfiguration control={form.control} />

              {/* Multi-agent selection */}
              <div className="space-y-2">
                <FormLabel>Select Agents (optional)</FormLabel>
                <FormDescription>
                  Choose one or more agents to inject their knowledge into the session at start.
                </FormDescription>
              </div>

              {/* Repositories (Optional) */}
              <RepositoryList
                repos={(form.watch("repos") || []) as Array<{ input: { url: string; branch: string }; output?: { url: string; branch: string } }>}
                mainRepoIndex={form.watch("mainRepoIndex") || 0}
                onAddRepo={() => {
                  setTempRepo({ input: { url: "", branch: "main" } });
                  setEditingRepoIndex(null);
                  setRepoDialogOpen(true);
                }}
                onEditRepo={(index) => {
                  const repo = form.getValues(`repos.${index}`) as { input: { url: string; branch: string }; output?: { url: string; branch: string } } | undefined;
                  if (repo) {
                    setTempRepo(repo);
                    setEditingRepoIndex(index);
                    setRepoDialogOpen(true);
                  }
                }}
                onRemoveRepo={(index) => {
                  removeRepo(index);
                  const currentMain = form.getValues("mainRepoIndex") || 0;
                  if (currentMain >= reposFields.length - 1) {
                    form.setValue("mainRepoIndex", Math.max(0, reposFields.length - 2));
                  }
                }}
                onSetMainRepo={(index) => form.setValue("mainRepoIndex", index)}
              />

              <RepositoryDialog
                open={repoDialogOpen}
                onOpenChange={setRepoDialogOpen}
                repo={tempRepo}
                onRepoChange={setTempRepo}
                onSave={() => {
                  if (!tempRepo.input.url) return;
                  if (editingRepoIndex !== null) {
                    updateRepo(editingRepoIndex, tempRepo);
                  } else {
                    appendRepo(tempRepo);
                  }
                }}
                isEditing={editingRepoIndex !== null}
                projectName={projectName}
              />

              {/* Runner behavior */}
              <FormField
                control={form.control}
                name="autoPushOnComplete"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-3">
                    <FormControl>
                      <Checkbox checked={field.value} onCheckedChange={(v) => field.onChange(Boolean(v))} />
                    </FormControl>
                    <div className="space-y-1 leading-none">
                      <FormLabel>Auto-push to Git on completion</FormLabel>
                      <FormDescription>
                        When enabled, the runner will commit and push changes automatically after it finishes.
                      </FormDescription>
                    </div>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Storage paths are managed automatically by the backend/operator */}

              {createSessionMutation.isError && (
                <div className="bg-red-50 border border-red-200 rounded-md p-3 dark:bg-red-950/50 dark:border-red-800">
                  <p className="text-red-700 dark:text-red-300 text-sm dark:text-red-300">{createSessionMutation.error?.message || "Failed to create session"}</p>
                </div>
              )}

              <div className="flex gap-4">
                <Button type="submit" disabled={createSessionMutation.isPending}>
                  {createSessionMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {createSessionMutation.isPending ? "Creating Session..." : "Create Agentic Session"}
                </Button>
                <Link href={`/projects/${encodeURIComponent(projectName)}/sessions`}>
                  <Button type="button" variant="link" disabled={createSessionMutation.isPending}>Cancel</Button>
                </Link>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  );
}