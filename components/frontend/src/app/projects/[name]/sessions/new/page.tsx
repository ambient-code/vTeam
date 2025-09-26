"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Loader2, Sparkle } from "lucide-react";
import { useForm, useFieldArray } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { getApiUrl } from "@/lib/config";
import type { CreateAgenticSessionRequest, AgentPersona as AgentSummary } from "@/types/agentic-session";
import { Checkbox } from "@/components/ui/checkbox";
import MultiAgentSelection from "@/components/multi-agent-selection";
import { Edit2, Plus, Trash2, ArrowRight } from "lucide-react";

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
    // PR options
    createPR: z.boolean().default(false),
    prTargetBranch: z.string().optional().or(z.literal("")),
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
const models = [
  { value: "claude-opus-4-1", label: "Claude Opus 4.1" },
  { value: "claude-opus-4-0", label: "Claude Opus 4" },
  { value: "claude-sonnet-4-0", label: "Claude Sonnet 4" },
  { value: "claude-3-7-sonnet-latest", label: "Claude Sonnet 3.7" },
  { value: "claude-3-5-haiku-latest", label: "Claude Haiku 3.5" },
];

export default function NewProjectSessionPage({ params }: { params: Promise<{ name: string }> }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [projectName, setProjectName] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [prefillWorkspacePath, setPrefillWorkspacePath] = useState<string | undefined>(undefined);
  const [rfeWorkflowId, setRfeWorkflowId] = useState<string | undefined>(undefined);
  const [selectedAgents, setSelectedAgents] = useState<string[]>([]);
  const [editingRepoIndex, setEditingRepoIndex] = useState<number | null>(null);
  const [repoDialogOpen, setRepoDialogOpen] = useState(false);
  const [tempRepo, setTempRepo] = useState<{ input: { url: string; branch: string }; output?: { url: string; branch: string } }>({ input: { url: "", branch: "main" } });
  const [forkOptions, setForkOptions] = useState<Array<{ fullName: string; url: string }>>([]);
  const [outputBranchMode, setOutputBranchMode] = useState<"same" | "auto">("auto");
  const [loadingRfeWorkflow, setLoadingRfeWorkflow] = useState(false);

  useEffect(() => {
    params.then(({ name }) => setProjectName(name));
  }, [params]);

  useEffect(() => {
    const ws = searchParams?.get("workspacePath");
    if (ws) setPrefillWorkspacePath(ws);
    const rfe = searchParams?.get("rfeWorkflow");
    if (rfe) setRfeWorkflowId(rfe);
  }, [searchParams]);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      prompt: "",
      model: "claude-3-7-sonnet-latest",
      temperature: 0.7,
      maxTokens: 4000,
      timeout: 300,
      interactive: false,
      createPR: false,
      prTargetBranch: "",
      agentPersona: "",
      repos: [],
      mainRepoIndex: 0,
    },
  });

  // Field arrays for multi-repo configuration
  const { fields: reposFields, append: appendRepo, remove: removeRepo, update: updateRepo } = useFieldArray({ control: form.control, name: "repos" });

  // Watch interactive to adjust prompt field hints
  const isInteractive = form.watch("interactive");

  // Load RFE workflow and prefill repos
  useEffect(() => {
    const loadRfeWorkflow = async () => {
      if (!projectName || !rfeWorkflowId) return;
      try {
        setLoadingRfeWorkflow(true);
        const apiUrl = getApiUrl();
        const resp = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/rfe-workflows/${encodeURIComponent(rfeWorkflowId)}`);
        if (!resp.ok) return;
        const workflow = await resp.json();
        
        // Prefill repos from RFE workflow (umbrella + supporting)
        if (workflow.umbrellaRepo) {
          const repos = [
            {
              input: { url: workflow.umbrellaRepo.url, branch: workflow.umbrellaRepo.branch || "main" },
              output: { url: workflow.umbrellaRepo.url, branch: workflow.umbrellaRepo.branch || "main" }
            },
            ...((workflow.supportingRepos || []).map((r: { url: string; branch?: string }) => ({
              input: { url: r.url, branch: r.branch || "main" },
              output: { url: r.url, branch: r.branch || "main" }
            })))
          ];
          form.setValue("repos", repos);
          form.setValue("mainRepoIndex", 0); // umbrella is main
        }
      } catch {
        // ignore
      } finally {
        setLoadingRfeWorkflow(false);
      }
    };
    loadRfeWorkflow();
  }, [projectName, rfeWorkflowId, form]);

  // Load forks when tempRepo input URL changes and dialog is open
  useEffect(() => {
    const loadForks = async () => {
      try {
        setForkOptions([]);
        const upstream = (tempRepo.input.url || "").trim();
        if (!projectName || !upstream || !repoDialogOpen) return;
        const apiUrl = getApiUrl();
        const url = `${apiUrl}/projects/${encodeURIComponent(projectName)}/users/forks?upstreamRepo=${encodeURIComponent(upstream)}`;
        const res = await fetch(url);
        if (!res.ok) return;
        const data = await res.json();
        const forks = Array.isArray(data.forks) ? data.forks : [];
        setForkOptions(forks.map((f: any) => ({ fullName: f.fullName || f.name, url: f.url })));
      } catch {
        // ignore
      }
    };
    loadForks();
  }, [projectName, tempRepo.input.url, repoDialogOpen]);

  

  const onSubmit = async (values: FormValues) => {
    if (!projectName) return;
    setIsSubmitting(true);
    setError(null);

    try {
      const promptToSend = values.interactive && !values.prompt.trim()
        ? "Running in interactive mode"
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
      };

      if (prefillWorkspacePath) {
        request.workspacePath = prefillWorkspacePath;
      }

      // Apply labels if rfeWorkflowId is present
      if (rfeWorkflowId || projectName) {
        request.labels = {
          ...(request.labels || {}),
          ...(projectName ? { project: projectName } : {}),
          ...(rfeWorkflowId ? { "rfe-workflow": rfeWorkflowId } : {}),
        };
      }

      // No Git user/auth; repositories managed via input/output env vars only

      // Inject selected agents via environment variables
      if (selectedAgents.length > 0) {
        request.environmentVariables = {
          ...(request.environmentVariables || {}),
          AGENT_PERSONAS: selectedAgents.join(","),
        };
      } else if (values.agentPersona) {
        // Fallback to single-agent support if provided
        request.environmentVariables = {
          ...(request.environmentVariables || {}),
          AGENT_PERSONA: values.agentPersona,
        };
      }

      // Multi-repo configuration
      const repos = (values as any).repos as Array<{ input: { url: string; branch?: string }; output?: { url: string; branch?: string } }> || [];
      if (Array.isArray(repos) && repos.length > 0) {
        const filteredRepos = repos.filter(r => r && r.input && r.input.url);
        (request as any).repos = filteredRepos;
        (request as any).mainRepoIndex = values.mainRepoIndex || 0;

        // Ensure runner env receives repos JSON + main repo index for immediate compatibility
        request.environmentVariables = {
          ...(request.environmentVariables || {}),
          REPOS_JSON: JSON.stringify(filteredRepos),
          MAIN_REPO_INDEX: String(values.mainRepoIndex || 0),
        };
      }

      // Pass PR creation intent as env vars
      if (values.createPR) {
        request.environmentVariables = {
          ...(request.environmentVariables || {}),
          CREATE_PR: "true",
        };
        const target = (values.prTargetBranch || "").trim();
        if (target) {
          request.environmentVariables.PR_TARGET_BRANCH = target;
        }
      }

      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: "Unknown error" }));
        throw new Error(errorData.message || "Failed to create agentic session");
      }

      try {
        const responseData = await response.json();
        const sessionName = responseData.name; 
        router.push(`/projects/${encodeURIComponent(projectName)}/sessions/${sessionName}`);
      } catch (err) {
        router.push(`/projects/${encodeURIComponent(projectName)}/sessions`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="container mx-auto p-6">
      <div className="flex items-center mb-6">
        <Link href={`/projects/${encodeURIComponent(projectName)}/sessions`}>
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to Sessions
          </Button>
        </Link>
      </div>

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


              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="model"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Model</FormLabel>
                      <Select onValueChange={field.onChange} defaultValue={field.value}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Select a model" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {models.map((m) => (
                            <SelectItem key={m.value} value={m.value}>
                              {m.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="temperature"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Temperature</FormLabel>
                      <FormControl>
                        <Input type="number" step="0.1" min="0" max="2" {...field} onChange={(e) => field.onChange(parseFloat(e.target.value))} />
                      </FormControl>
                      <FormDescription>Controls randomness (0.0 - 2.0)</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              {/* Multi-agent selection */}
              <div className="space-y-2">
                <FormLabel>Select Agents (optional)</FormLabel>
                <FormDescription>
                  Choose one or more agents to inject their knowledge into the session at start.
                </FormDescription>
              </div>


              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="maxTokens"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Max Tokens</FormLabel>
                      <FormControl>
                        <Input type="number" min="100" max="8000" {...field} onChange={(e) => field.onChange(parseInt(e.target.value))} />
                      </FormControl>
                      <FormDescription>Maximum response length</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="timeout"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Timeout (seconds)</FormLabel>
                      <FormControl>
                        <Input type="number" min="60" max="1800" {...field} onChange={(e) => field.onChange(parseInt(e.target.value))} />
                      </FormControl>
                      <FormDescription>Maximum execution time</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              

              {/* Repositories (Optional) */}
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-lg font-medium">Repositories (Optional)</h3>
                    <p className="text-sm text-muted-foreground">Add repositories to clone side-by-side</p>
                  </div>
                  <Button 
                    type="button" 
                    variant="outline" 
                    size="sm"
                    onClick={() => {
                      setTempRepo({ input: { url: "", branch: "main" } });
                      setOutputBranchMode("auto");
                      setEditingRepoIndex(null);
                      setRepoDialogOpen(true);
                    }}
                  >
                    <Plus className="h-4 w-4 mr-2" />
                    Add Repository
                  </Button>
                </div>

                {reposFields.length > 0 && (
                  <div className="border rounded-md divide-y">
                    {reposFields.map((field, index) => {
                      const repo = form.getValues(`repos.${index}`) as any;
                      const isEntrypoint = form.watch("mainRepoIndex") === index;
                      return (
                        <div key={field.id} className="p-3 flex items-center justify-between hover:bg-muted/50" style={{ borderLeft: isEntrypoint ? "4px solid #007bff" : "transparent" }}>
                          <div className="flex-1 flex items-center gap-2 text-sm font-mono">
                          <span className="text-muted-foreground">{repo.input?.url || "(empty)"}</span>
                            <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded font-sans">{repo.input?.branch || "main"}</span>
                            <ArrowRight className="h-3 w-3 text-muted-foreground" />
                            <span className="text-muted-foreground">{repo.output?.url || "(no push)"}</span>
                            {repo.output?.url && (
                               <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded font-sans">{repo.output?.branch || (
                                <div className="flex items-center gap-1">
                                <Sparkle
                                className="h-3 w-3 text-muted-foreground"
                                />
                                auto
                                </div>
                               )}</span>
                            )}
                          </div>
                          <div className="flex items-center gap-1">
                            {!isEntrypoint && (
                              <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                onClick={() => form.setValue("mainRepoIndex", index)}
                                title="Set as main repo"
                              >
                                <span className="text-xs">Set as entrypoint</span>
                              </Button>
                            )}
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => {
                                setTempRepo(repo);
                                // Determine output branch mode from existing data
                                if (repo.output?.branch === repo.input?.branch) {
                                  setOutputBranchMode("same");
                                } else {
                                  setOutputBranchMode("auto");
                                }
                                setEditingRepoIndex(index);
                                setRepoDialogOpen(true);
                              }}
                            >
                              <Edit2 className="h-4 w-4" />
                            </Button>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => {
                                removeRepo(index);
                                // Adjust mainRepoIndex if needed
                                const currentMain = form.getValues("mainRepoIndex") || 0;
                                if (currentMain >= reposFields.length - 1) {
                                  form.setValue("mainRepoIndex", Math.max(0, reposFields.length - 2));
                                }
                              }}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}

                {reposFields.length > 0 && (
                  <p className="text-xs text-muted-foreground">
                    {(() => {
                      const repos = form.getValues("repos");
                      const mainRepoIndex = form.getValues("mainRepoIndex") ?? 0;
                      const mainRepo = Array.isArray(repos) && repos[mainRepoIndex] ? repos[mainRepoIndex] : undefined;
                      const mainRepoUrl = mainRepo?.input?.url ?? "";
                      return (
                        <>
                          The {mainRepoUrl || "selected"} repo is Claude&apos;s working directory. Other repos are available as add_dirs.
                        </>
                      );
                    })()}
                  </p>
                )}
              </div>

              {/* Repository Edit Dialog */}
              <Dialog open={repoDialogOpen} onOpenChange={setRepoDialogOpen}>
                <DialogContent className="max-w-2xl">
                  <DialogHeader>
                    <DialogTitle>{editingRepoIndex !== null ? "Edit Repository" : "Add Repository"}</DialogTitle>
                    <DialogDescription>Configure input and optional output repository settings</DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4 py-4">
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Input Repository URL</label>
                      <Input
                        placeholder="https://github.com/org/repo.git"
                        value={tempRepo.input.url}
                        onChange={(e) => setTempRepo({ ...tempRepo, input: { ...tempRepo.input, url: e.target.value } })}
                      />
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Input Branch</label>
                      <Input
                        placeholder="main"
                        value={tempRepo.input.branch}
                        onChange={(e) => setTempRepo({ ...tempRepo, input: { ...tempRepo.input, branch: e.target.value } })}
                      />
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Output Repository (optional)</label>
                        <Select
                        value={tempRepo.output?.url || "__none__"}
                        onValueChange={(val) => {
                          if (val === "__none__") {
                            setTempRepo({ ...tempRepo, output: undefined });
                          } else {
                            setTempRepo({ ...tempRepo, output: { url: val, branch: outputBranchMode === "same" ? tempRepo.input.branch : "" } });
                          }
                        }}
                      >
                            <SelectTrigger>
                          <SelectValue placeholder={tempRepo.input.url ? "Select fork or same as input" : "Enter input repo first"} />
                            </SelectTrigger>
                          <SelectContent>
                          <SelectItem value="__none__">No output (don&apos;t push)</SelectItem>
                          {tempRepo.input.url && (
                            <SelectItem value={tempRepo.input.url}>Same as input</SelectItem>
                            )}
                            {forkOptions.map((f) => (
                              <SelectItem key={f.fullName} value={f.url}>{f.fullName}</SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      <p className="text-xs text-muted-foreground">Must be upstream or one of your forks</p>
                    </div>
                    {tempRepo.output?.url && (
                      <div className="space-y-2">
                        <label className="text-sm font-medium">Output Branch</label>
                        <Select
                          value={outputBranchMode}
                          onValueChange={(val: "same" | "auto") => {
                            setOutputBranchMode(val);
                            if (val === "same") {
                              setTempRepo({ ...tempRepo, output: { ...tempRepo.output!, branch: tempRepo.input.branch } });
                            } else {
                              setTempRepo({ ...tempRepo, output: { ...tempRepo.output!, branch: "" } });
                            }
                          }}
                        >
                            <SelectTrigger>
                              <SelectValue placeholder="Select output branch mode" />
                            </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="same">Same as input branch</SelectItem>
                            <SelectItem value="auto">Auto-generate sessions/{'{'}{'{'}session_id{'}'}{'}'}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                        <p className="text-xs text-muted-foreground">To avoid conflicts, custom branches are not allowed</p>
                      </div>
                    )}
                </div>
                  <div className="flex justify-end gap-2">
                    <Button type="button" variant="outline" onClick={() => setRepoDialogOpen(false)}>Cancel</Button>
                    <Button
                      type="button"
                      onClick={() => {
                        if (!tempRepo.input.url) return;
                        if (editingRepoIndex !== null) {
                          updateRepo(editingRepoIndex, tempRepo);
                        } else {
                          appendRepo(tempRepo);
                        }
                        setRepoDialogOpen(false);
                      }}
                    >
                      {editingRepoIndex !== null ? "Update" : "Add"}
                    </Button>
              </div>
                </DialogContent>
              </Dialog>

              {/* PR creation option */}
              <div className="space-y-4">
                <h3 className="text-lg font-medium">Pull Request (Optional)</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="createPR"
                    render={({ field }) => (
                      <FormItem className="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-3">
                        <FormControl>
                          <input
                            type="checkbox"
                            checked={field.value}
                            onChange={(e) => field.onChange(e.target.checked)}
                          />
                        </FormControl>
                        <div className="space-y-1 leading-none">
                          <FormLabel>Create PR after run</FormLabel>
                          <FormDescription>
                            When enabled, a PR will be created if output branch differs from input branch.
                          </FormDescription>
                        </div>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </div>

              {/* Storage paths are managed automatically by the backend/operator */}

              {error && (
                <div className="bg-red-50 border border-red-200 rounded-md p-3">
                  <p className="text-red-700 text-sm">{error}</p>
                </div>
              )}

              <div className="flex gap-4">
                <Button type="submit" disabled={isSubmitting}>
                  {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {isSubmitting ? "Creating Session..." : "Create Agentic Session"}
                </Button>
                <Link href={`/projects/${encodeURIComponent(projectName)}/sessions`}>
                  <Button type="button" variant="link" disabled={isSubmitting}>Cancel</Button>
                </Link>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  );
}