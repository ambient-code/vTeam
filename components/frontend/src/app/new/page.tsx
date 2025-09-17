"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import Link from "next/link";
import { ArrowLeft, Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { CreateAgenticSessionRequest, GitConfig } from "@/types/agentic-session";

import { getApiUrl } from "@/lib/config";

const formSchema = z.object({
  prompt: z.string().min(10, "Prompt must be at least 10 characters long"),
  taskType: z.enum(["website", "repository", "scratch"], {
    required_error: "Please select a task type"
  }),
  websiteURL: z.string().optional(),
  model: z.string().min(1, "Please select a model"),
  temperature: z.number().min(0).max(2),
  maxTokens: z.number().min(100).max(8000),
  timeout: z.number().min(60).max(1800),
  // Git configuration (optional)
  gitUserName: z.string().optional(),
  gitUserEmail: z.string().email().optional().or(z.literal("")),
  gitSshKeySecret: z.string().optional(),
  gitTokenSecret: z.string().optional(),
  gitKnownHostsSecret: z.string().optional(),
  gitRepositoryUrl: z.string().optional(),
  gitRepositoryBranch: z.string().optional(),
  gitRepositoryPath: z.string().optional(),
}).refine((data) => {
  // Website URL is required only for website analysis
  if (data.taskType === "website" && (!data.websiteURL || data.websiteURL.trim() === "")) {
    return false;
  }
  // Git repository URL is required only for repository tasks
  if (data.taskType === "repository" && (!data.gitRepositoryUrl || data.gitRepositoryUrl.trim() === "")) {
    return false;
  }
  // Validate URL formats when provided
  if (data.websiteURL && data.websiteURL.trim() !== "") {
    try {
      new URL(data.websiteURL);
    } catch {
      return false;
    }
  }
  if (data.gitRepositoryUrl && data.gitRepositoryUrl.trim() !== "") {
    // Allow both HTTP/HTTPS and SSH URLs for Git
    const gitUrlPattern = /^(https?:\/\/|git@)/;
    if (!gitUrlPattern.test(data.gitRepositoryUrl)) {
      return false;
    }
  }
  return true;
}, {
  message: "Please provide the required URL for your selected task type",
  path: ["websiteURL"] // Show error on website URL field
});

type FormValues = z.infer<typeof formSchema>;

const models = [
  { value: "claude-3-5-sonnet-20241022", label: "Claude 3.5 Sonnet" },
  { value: "claude-3-haiku-20240307", label: "Claude 3 Haiku" },
  { value: "claude-3-opus-20240229", label: "Claude 3 Opus" },
];

export default function NewAgenticSessionPage() {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      prompt: "",
      taskType: "website",
      websiteURL: "",
      model: "claude-3-5-sonnet-20241022",
      temperature: 0.7,
      maxTokens: 4000,
      timeout: 300,
      // Git defaults
      gitUserName: "",
      gitUserEmail: "",
      gitSshKeySecret: "",
      gitTokenSecret: "",
      gitKnownHostsSecret: "",
      gitRepositoryUrl: "",
      gitRepositoryBranch: "main",
      gitRepositoryPath: "",
    },
  });

  const onSubmit = async (values: FormValues) => {
    setIsSubmitting(true);
    setError(null);

    try {
      // Build Git configuration if provided
      let gitConfig = undefined;

      const hasGitUser = values.gitUserName || values.gitUserEmail;
      const hasGitAuth = values.gitSshKeySecret || values.gitTokenSecret;
      const hasGitRepo = values.gitRepositoryUrl;

      if (hasGitUser || hasGitAuth || hasGitRepo) {
        gitConfig = {} as GitConfig;

        // Add user configuration
        if (hasGitUser) {
          gitConfig.user = {
            name: values.gitUserName || "",
            email: values.gitUserEmail || "",
          };
        }

        // Add authentication
        if (hasGitAuth) {
          gitConfig.authentication = {
            sshKeySecret: values.gitSshKeySecret || undefined,
            tokenSecret: values.gitTokenSecret || undefined,
            knownHostsSecret: values.gitKnownHostsSecret || undefined,
          };
        }

        // Add repository
        if (hasGitRepo && values.gitRepositoryUrl) {
          gitConfig.repositories = [{
            url: values.gitRepositoryUrl,
            ...(values.gitRepositoryBranch && { branch: values.gitRepositoryBranch }),
            ...(values.gitRepositoryPath && { clonePath: values.gitRepositoryPath }),
          }];
        }
      }

      const request: CreateAgenticSessionRequest = {
        prompt: values.prompt,
        websiteURL: values.websiteURL || "https://example.com", // Provide default for backend compatibility
        llmSettings: {
          model: values.model,
          temperature: values.temperature,
          maxTokens: values.maxTokens,
        },
        timeout: values.timeout,
        ...(gitConfig && { gitConfig }),
      };

      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/agentic-sessions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        const errorData = await response
          .json()
          .catch(() => ({ message: "Unknown error" }));
        throw new Error(
          errorData.message || "Failed to create agentic session"
        );
      }

      // Redirect to the main page on success
      router.push("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="container mx-auto p-6 max-w-2xl">
      <div className="flex items-center mb-6">
        <Link href="/">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to Sessions
          </Button>
        </Link>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>New Agentic Session</CardTitle>
          <CardDescription>
            Create a new AI-powered session for website analysis, repository work, or starting from scratch
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              <FormField
                control={form.control}
                name="taskType"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Task Type</FormLabel>
                    <Select onValueChange={field.onChange} defaultValue={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="What would you like Claude to do?" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="website">Analyze Website</SelectItem>
                        <SelectItem value="repository">Work with Git Repository</SelectItem>
                        <SelectItem value="scratch">Start from Scratch</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      Choose the type of task you want Claude to perform
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="prompt"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Agentic Prompt</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder={
                          form.watch("taskType") === "website"
                            ? "Describe what you want Claude to analyze on the website..."
                            : form.watch("taskType") === "repository"
                            ? "Describe what you want Claude to do with the repository..."
                            : "Describe what you want Claude to create or build..."
                        }
                        className="min-h-[100px]"
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      Provide a detailed prompt about what you want Claude to do
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {form.watch("taskType") === "website" && (
                <FormField
                  control={form.control}
                  name="websiteURL"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Website URL</FormLabel>
                      <FormControl>
                        <Input placeholder="https://example.com" {...field} />
                      </FormControl>
                      <FormDescription>
                        The website that Claude will analyze
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {form.watch("taskType") === "repository" && (
                <FormField
                  control={form.control}
                  name="gitRepositoryUrl"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Git Repository URL</FormLabel>
                      <FormControl>
                        <Input placeholder="git@github.com:user/repo.git or https://github.com/user/repo.git" {...field} />
                      </FormControl>
                      <FormDescription>
                        Git repository to clone and work with (SSH or HTTPS format)
                      </FormDescription>
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
                      <Select
                        onValueChange={field.onChange}
                        defaultValue={field.value}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Select a model" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {models.map((model) => (
                            <SelectItem key={model.value} value={model.value}>
                              {model.label}
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
                        <Input
                          type="number"
                          step="0.1"
                          min="0"
                          max="2"
                          {...field}
                          onChange={(e) =>
                            field.onChange(parseFloat(e.target.value))
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        Controls randomness (0.0 - 2.0)
                      </FormDescription>
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
                        <Input
                          type="number"
                          min="100"
                          max="8000"
                          {...field}
                          onChange={(e) =>
                            field.onChange(parseInt(e.target.value))
                          }
                        />
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
                        <Input
                          type="number"
                          min="60"
                          max="1800"
                          {...field}
                          onChange={(e) =>
                            field.onChange(parseInt(e.target.value))
                          }
                        />
                      </FormControl>
                      <FormDescription>Maximum execution time</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              {/* Git Configuration Section */}
              <Card className="p-4">
                <CardHeader className="p-0 pb-4">
                  <CardTitle className="text-lg">
                    Git Configuration (Optional)
                    {form.watch("taskType") === "repository" && " - Required for Repository Tasks"}
                  </CardTitle>
                  <CardDescription>
                    {form.watch("taskType") === "repository"
                      ? "Configure Git credentials and user settings to access your repository"
                      : form.watch("taskType") === "scratch"
                      ? "Configure Git settings if you want Claude to create and push to a new repository"
                      : "Configure Git settings if you want Claude to work with code repositories"}
                  </CardDescription>
                </CardHeader>
                <CardContent className="p-0 space-y-4">
                  {/* Git User Configuration */}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="gitUserName"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Git User Name</FormLabel>
                          <FormControl>
                            <Input placeholder="Your Name" {...field} />
                          </FormControl>
                          <FormDescription>
                            Name for Git commits (git config user.name)
                          </FormDescription>
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
                            <Input placeholder="your.email@company.com" {...field} />
                          </FormControl>
                          <FormDescription>
                            Email for Git commits (git config user.email)
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  {/* Git Authentication */}
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <FormField
                      control={form.control}
                      name="gitSshKeySecret"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>SSH Key Secret</FormLabel>
                          <FormControl>
                            <Input placeholder="my-ssh-key" {...field} />
                          </FormControl>
                          <FormDescription>
                            Kubernetes secret with SSH private key
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="gitTokenSecret"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Token Secret</FormLabel>
                          <FormControl>
                            <Input placeholder="my-github-token" {...field} />
                          </FormControl>
                          <FormDescription>
                            Kubernetes secret with Git access token
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="gitKnownHostsSecret"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Known Hosts Secret</FormLabel>
                          <FormControl>
                            <Input placeholder="my-known-hosts" {...field} />
                          </FormControl>
                          <FormDescription>
                            Kubernetes secret with SSH known_hosts
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  {/* Repository Configuration - Only show for repository tasks */}
                  {form.watch("taskType") === "repository" && (
                    <div className="space-y-4">
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <FormField
                          control={form.control}
                          name="gitRepositoryBranch"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Branch (Optional)</FormLabel>
                              <FormControl>
                                <Input placeholder="main" {...field} />
                              </FormControl>
                              <FormDescription>
                                Git branch to checkout (defaults to main)
                              </FormDescription>
                              <FormMessage />
                            </FormItem>
                          )}
                        />

                        <FormField
                          control={form.control}
                          name="gitRepositoryPath"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Local Directory Name (Optional)</FormLabel>
                              <FormControl>
                                <Input placeholder="my-project" {...field} />
                              </FormControl>
                              <FormDescription>
                                Folder name when cloning (defaults to repository name)
                              </FormDescription>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>

              {error && (
                <div className="bg-red-50 border border-red-200 rounded-md p-3">
                  <p className="text-red-700 text-sm">{error}</p>
                </div>
              )}

              <div className="flex gap-4">
                <Button type="submit" disabled={isSubmitting}>
                  {isSubmitting && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  {isSubmitting
                    ? "Creating Session..."
                    : "Create Agentic Session"}
                </Button>
                <Link href="/">
                  <Button type="button" variant="link" disabled={isSubmitting}>
                    Cancel
                  </Button>
                </Link>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  );
}