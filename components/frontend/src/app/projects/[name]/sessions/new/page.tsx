"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Loader2, Info } from "lucide-react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { getApiUrl } from "@/lib/config";
import type { CreateAgenticSessionRequest, AgentPersona as AgentSummary } from "@/types/agentic-session";
import { Checkbox } from "@/components/ui/checkbox";

const formSchema = z.object({
  prompt: z.string().min(10, "Prompt must be at least 10 characters long"),
  model: z.string().min(1, "Please select a model"),
  temperature: z.number().min(0).max(2),
  maxTokens: z.number().min(100).max(8000),
  timeout: z.number().min(60).max(1800),
  interactive: z.boolean().default(false),
  // Git configuration fields
  gitUserName: z.string().optional(),
  gitUserEmail: z.string().email().optional().or(z.literal("")),
  gitRepoUrl: z.string().url().optional().or(z.literal("")),
  // storage paths are not user-configurable anymore
  agentPersona: z.string().optional(),
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
  const [projectName, setProjectName] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [agents, setAgents] = useState<AgentSummary[]>([]);

  useEffect(() => {
    params.then(({ name }) => setProjectName(name));
  }, [params]);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      prompt: "",
      model: "claude-3-7-sonnet-latest",
      temperature: 0.7,
      maxTokens: 4000,
      timeout: 300,
      interactive: false,
      gitUserName: "",
      gitUserEmail: "",
      gitRepoUrl: "",
      agentPersona: "",
      
    },
  });

  useEffect(() => {
    const loadAgents = async () => {
      if (!projectName) return;
      try {
        const apiUrl = getApiUrl();
        const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agents`);
        if (res.ok) {
          const data = await res.json();
          setAgents(Array.isArray(data) ? data : []);
        }
      } catch {
        // ignore
      }
    };
    loadAgents();
  }, [projectName]);

  const onSubmit = async (values: FormValues) => {
    if (!projectName) return;
    setIsSubmitting(true);
    setError(null);

    try {
      const request: CreateAgenticSessionRequest = {
        prompt: values.prompt,
        llmSettings: {
          model: values.model,
          temperature: values.temperature,
          maxTokens: values.maxTokens,
        },
        timeout: values.timeout,
        interactive: values.interactive,
      };

      // Add Git configuration if provided
      if (values.gitUserName || values.gitUserEmail || values.gitRepoUrl) {
        request.gitConfig = {};

        if (values.gitUserName && values.gitUserEmail) {
          request.gitConfig.user = {
            name: values.gitUserName,
            email: values.gitUserEmail,
          };
        }

        if (values.gitRepoUrl) {
          request.gitConfig.repositories = [
            {
              url: values.gitRepoUrl,
              branch: "main",
            },
          ];
        }
      }

      // No user-configurable storage paths; backend/operator provide defaults
      if (values.agentPersona) {
        request.environmentVariables = {
          ...(request.environmentVariables || {}),
          AGENT_PERSONA: values.agentPersona,
        };
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
    <div className="container mx-auto p-6 max-w-2xl">
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

              <FormField
                control={form.control}
                name="agentPersona"
                render={({ field }) => {
                  const selected = agents.find((a) => a.persona === field.value);
                  return (
                    <FormItem>
                      <FormLabel className="flex items-center gap-2">
                        Agent Persona
                        {selected && (
                          <span className="relative group inline-block">
                            <Info className="w-4 h-4 text-muted-foreground" />
                            <div className="absolute z-10 hidden group-hover:block bg-popover text-popover-foreground border rounded-md p-3 shadow-md w-72 left-1/2 -translate-x-1/2 mt-2">
                              <div className="text-sm font-medium mb-1">{selected.name}</div>
                              <div className="text-xs text-muted-foreground mb-2">{selected.role}</div>
                              <ul className="list-disc pl-5 space-y-1 text-sm">
                                {selected.expertise?.map((e, i) => (
                                  <li key={i}>{e}</li>
                                ))}
                              </ul>
                            </div>
                          </span>
                        )}
                      </FormLabel>
                      <Select onValueChange={field.onChange} defaultValue={field.value}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Select an agent (optional)" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {agents.map((a) => (
                            <SelectItem key={a.persona} value={a.persona}>
                              <div className="flex flex-col">
                                <span>{a.name}</span>
                                <span className="text-xs text-muted-foreground">{a.role}</span>
                              </div>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <FormDescription>
                        Optionally inject a persona’s knowledge into the session at start.
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  );
                }}
              />

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

              {/* Git Configuration Section */}
              <div className="space-y-4">
                <h3 className="text-lg font-medium">Git Configuration (Optional)</h3>
                <div className="grid grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="gitUserName"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Git User Name</FormLabel>
                        <FormControl>
                          <Input
                            placeholder="Your Name"
                            {...field}
                          />
                        </FormControl>
                        <FormDescription>Name for Git commits</FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="gitUserEmail"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Git User Email</FormLabel>
                        <FormControl>
                          <Input
                            placeholder="your.email@example.com"
                            type="email"
                            {...field}
                          />
                        </FormControl>
                        <FormDescription>Email for Git commits</FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
                <FormField
                  control={form.control}
                  name="gitRepoUrl"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Git Repository URL</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="https://github.com/username/repo.git"
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>Git repository to clone and work with</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
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