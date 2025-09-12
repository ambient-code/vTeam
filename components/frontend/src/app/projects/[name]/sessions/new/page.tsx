"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Loader2 } from "lucide-react";
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
import type { CreateAgenticSessionRequest } from "@/types/agentic-session";

const formSchema = z.object({
  prompt: z.string().min(10, "Prompt must be at least 10 characters long"),
  websiteURL: z.string().url("Please enter a valid URL"),
  model: z.string().min(1, "Please select a model"),
  temperature: z.number().min(0).max(2),
  maxTokens: z.number().min(100).max(8000),
  timeout: z.number().min(60).max(1800),
  // Git configuration fields
  gitUserName: z.string().optional(),
  gitUserEmail: z.string().email().optional().or(z.literal("")),
  gitRepoUrl: z.string().url().optional().or(z.literal("")),
});

type FormValues = z.infer<typeof formSchema>;

const models = [
  { value: "claude-3-5-sonnet-20241022", label: "Claude 3.5 Sonnet" },
  { value: "claude-3-haiku-20240307", label: "Claude 3 Haiku" },
  { value: "claude-3-opus-20240229", label: "Claude 3 Opus" },
];

export default function NewProjectSessionPage({ params }: { params: Promise<{ name: string }> }) {
  const router = useRouter();
  const [projectName, setProjectName] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    params.then(({ name }) => setProjectName(name));
  }, [params]);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      prompt: "",
      websiteURL: "",
      model: "claude-3-5-sonnet-20241022",
      temperature: 0.7,
      maxTokens: 4000,
      timeout: 300,
      gitUserName: "",
      gitUserEmail: "",
      gitRepoUrl: "",
    },
  });

  const onSubmit = async (values: FormValues) => {
    if (!projectName) return;
    setIsSubmitting(true);
    setError(null);

    try {
      const request: CreateAgenticSessionRequest = {
        prompt: values.prompt,
        websiteURL: values.websiteURL,
        llmSettings: {
          model: values.model,
          temperature: values.temperature,
          maxTokens: values.maxTokens,
        },
        timeout: values.timeout,
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

      router.push(`/projects/${encodeURIComponent(projectName)}/sessions`);
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

              <FormField
                control={form.control}
                name="websiteURL"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Website URL</FormLabel>
                    <FormControl>
                      <Input placeholder="https://example.com" {...field} />
                    </FormControl>
                    <FormDescription>The website that Claude will analyze</FormDescription>
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