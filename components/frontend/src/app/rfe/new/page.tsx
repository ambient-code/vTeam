"use client";

import { useState, useCallback } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import Link from "next/link";
import { ArrowLeft, Loader2, GitBranch, Users } from "lucide-react";

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
import { AgentSelection } from "@/components/agent-selection";
import { CreateRFEWorkflowRequest } from "@/types/agentic-session";
import { DEFAULT_AGENT_SELECTIONS } from "@/lib/agents";
import { getApiUrl } from "@/lib/config";

const formSchema = z.object({
  title: z.string().min(5, "Title must be at least 5 characters long"),
  description: z.string().min(20, "Description must be at least 20 characters long"),
  targetRepoUrl: z.string().url("Please enter a valid repository URL"),
  targetRepoBranch: z.string().min(1, "Branch is required"),
  selectedAgents: z.array(z.string()).min(1, "At least one agent must be selected"),
  // Git configuration fields
  gitUserName: z.string().optional(),
  gitUserEmail: z.union([
    z.literal(""),
    z.string().email("Please enter a valid email")
  ]).optional(),
});

type FormValues = z.infer<typeof formSchema>;

export default function NewRFEWorkflowPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const project = searchParams.get("project");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onBlur", // Only validate on blur, not on every change
    defaultValues: {
      title: "",
      description: "",
      targetRepoUrl: "",
      targetRepoBranch: "main",
      selectedAgents: DEFAULT_AGENT_SELECTIONS.BALANCED,
      gitUserName: "",
      gitUserEmail: "",
    },
  });

  // Stable callback for agent selection changes
  const handleAgentSelectionChange = useCallback((agents: string[]) => {
    try {
      console.log('handleAgentSelectionChange called with:', agents);

      // Validate that agents is an array
      if (!Array.isArray(agents)) {
        console.error('agents parameter is not an array:', agents);
        return;
      }

      // Validate that all agents are strings
      const invalidAgents = agents.filter(agent => typeof agent !== 'string');
      if (invalidAgents.length > 0) {
        console.error('Some agents are not strings:', invalidAgents);
        return;
      }

      form.setValue("selectedAgents", agents, { shouldValidate: false, shouldDirty: true });
      console.log('Successfully set form value for selectedAgents');
    } catch (error) {
      console.error('Error in handleAgentSelectionChange:', error);
    }
  }, [form]);

  const onSubmit = async (values: FormValues) => {
    console.log("Form submission started with values:", values);

    // Validate that we have selected agents
    if (!values.selectedAgents || values.selectedAgents.length === 0) {
      setError("Please select at least one agent");
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      const request: CreateRFEWorkflowRequest = {
        title: values.title,
        description: values.description,
        targetRepoUrl: values.targetRepoUrl,
        targetRepoBranch: values.targetRepoBranch,
        selectedAgents: values.selectedAgents,
        gitUserName: values.gitUserName || undefined,
        gitUserEmail: values.gitUserEmail || undefined,
      };

      console.log("Creating RFE workflow with request:", request);
      console.log("Selected agents:", values.selectedAgents);

      const url = project
        ? `${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows`
        : `${getApiUrl()}/rfe-workflows`;
      const response = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        const errorMessage = errorData.error || errorData.message || `HTTP ${response.status}: ${response.statusText}`;
        throw new Error(errorMessage);
      }

      const result = await response.json();
      console.log("RFE workflow created successfully:", result);

      // Redirect to the RFE workflow detail page
      const next = project ? `/rfe/${result.id}?project=${encodeURIComponent(project)}` : `/rfe/${result.id}`;
      router.push(next);
    } catch (err) {
      console.error("Error creating RFE workflow:", err);
      setError(
        err instanceof Error
          ? err.message
          : "An unexpected error occurred while creating the RFE workflow"
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="container mx-auto py-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex items-center gap-4 mb-8">
          <Link href="/rfe">
            <Button variant="ghost" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to RFE Workflows
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Create RFE Workflow</h1>
            <p className="text-muted-foreground">
              Set up a new Request for Enhancement workflow with AI agents
            </p>
          </div>
        </div>

        <Form {...form}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              form.handleSubmit(onSubmit)(e);
            }}
            className="space-y-8"
          >
            {/* Basic Information */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <GitBranch className="h-5 w-5" />
                  RFE Details
                </CardTitle>
                <CardDescription>
                  Provide basic information about the feature or enhancement
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <FormField
                  control={form.control}
                  name="title"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>RFE Title</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="e.g., User Authentication System"
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        A concise title that describes the feature or enhancement
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="description"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Description</FormLabel>
                      <FormControl>
                        <Textarea
                          placeholder="Describe the feature requirements, goals, and context..."
                          rows={4}
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        Detailed description of what needs to be built and why
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </CardContent>
            </Card>

            {/* Target Repository */}
            <Card>
              <CardHeader>
                <CardTitle>Target Repository</CardTitle>
                <CardDescription>
                  Specify where the RFE artifacts and implementation will be stored
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <FormField
                  control={form.control}
                  name="targetRepoUrl"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Repository URL</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="https://github.com/your-org/project.git"
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        Git repository where specs and implementation will be stored
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="targetRepoBranch"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Branch</FormLabel>
                      <FormControl>
                        <Input placeholder="main" {...field} />
                      </FormControl>
                      <FormDescription>
                        Target branch for the RFE workflow
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </CardContent>
            </Card>

            {/* Agent Selection */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Users className="h-5 w-5" />
                  Agent Selection
                </CardTitle>
                <CardDescription>
                  Choose AI agents to participate in the RFE workflow
                </CardDescription>
              </CardHeader>
              <CardContent>
                <FormField
                  control={form.control}
                  name="selectedAgents"
                  render={({ field, fieldState }) => (
                    <FormItem>
                      <FormControl>
                        <AgentSelection
                          selectedAgents={Array.isArray(field.value) ? field.value : []}
                          onSelectionChange={handleAgentSelectionChange}
                          maxAgents={8}
                          disabled={isSubmitting}
                        />
                      </FormControl>
                      {fieldState.error && (
                        <FormMessage>{fieldState.error.message}</FormMessage>
                      )}
                    </FormItem>
                  )}
                />
              </CardContent>
            </Card>

            {/* Git Configuration (Optional) */}
            <Card>
              <CardHeader>
                <CardTitle>Git Configuration (Optional)</CardTitle>
                <CardDescription>
                  Configure Git user information for commits made during the workflow
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
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
                          Name to use for Git commits
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
                          <Input
                            type="email"
                            placeholder="your.email@example.com"
                            {...field}
                          />
                        </FormControl>
                        <FormDescription>
                          Email to use for Git commits
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </CardContent>
            </Card>

            {/* Error Display */}
            {error && (
              <Card className="border-red-200 bg-red-50">
                <CardContent className="pt-6">
                  <p className="text-red-600 text-sm">{error}</p>
                </CardContent>
              </Card>
            )}

            {/* Submit Button */}
            <div className="flex justify-end gap-4">
              <Link href="/rfe">
                <Button variant="outline" disabled={isSubmitting}>
                  Cancel
                </Button>
              </Link>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating RFE Workflow...
                  </>
                ) : (
                  "Create RFE Workflow"
                )}
              </Button>
            </div>
          </form>
        </Form>
      </div>
    </div>
  );
}