'use client';

import React from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import Link from 'next/link';
import { ArrowLeft, Loader2, GitBranch } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { ErrorMessage } from '@/components/error-message';

import { useCreateRfeWorkflow, useProjectSettings } from '@/services/queries';
import { successToast, errorToast } from '@/hooks/use-toast';
import { Breadcrumbs } from '@/components/breadcrumbs';
import type { CreateRFEWorkflowRequest } from '@/types/api';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Info } from 'lucide-react';

const formSchema = z.object({
  title: z.string().min(5, 'Title must be at least 5 characters long'),
  description: z.string().min(20, 'Description must be at least 20 characters long'),
  branchName: z.string().min(1, 'Branch name is required'),
  workspacePath: z.string().optional(),
  parentOutcome: z.string().optional(),
  umbrellaRepoName: z.string().min(1, 'Please select an umbrella repository'),
  supportingRepoNames: z.array(z.string()).optional().default([]),
});

type FormValues = z.input<typeof formSchema>;

// Generate branch name from title (ambient-first-three-words)
function generateBranchName(title: string): string {
  const normalized = title.toLowerCase().trim();
  const words = normalized
    .split(/[^a-z0-9]+/)
    .filter((w) => w.length > 0)
    .slice(0, 3);
  return words.length > 0 ? `ambient-${words.join('-')}` : '';
}

export default function ProjectNewRFEWorkflowPage() {
  const router = useRouter();
  const params = useParams();
  const projectName = params?.name as string;

  // Fetch ProjectSettings to get available repos
  const { data: projectSettings, isLoading: settingsLoading, error: settingsError } = useProjectSettings(projectName);

  // React Query mutation replaces manual fetch
  const createWorkflowMutation = useCreateRfeWorkflow();

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: 'onBlur',
    defaultValues: {
      title: '',
      description: '',
      branchName: '',
      workspacePath: '',
      parentOutcome: '',
      umbrellaRepoName: '',
      supportingRepoNames: [],
    },
  });

  // Watch the title field and auto-populate branchName
  const title = form.watch('title');
  const branchName = form.watch('branchName');

  // Auto-populate branch name when title changes
  // This will only update if user hasn't manually edited the branch name
  React.useEffect(() => {
    const generatedName = generateBranchName(title);
    const currentBranchName = form.getValues('branchName');

    // Only auto-populate if:
    // 1. There's a generated name
    // 2. Current branch name is empty or matches the previously auto-generated name
    if (generatedName && (!currentBranchName || currentBranchName.startsWith('ambient-'))) {
      form.setValue('branchName', generatedName, { shouldValidate: false, shouldDirty: false });
    }
  }, [title, form]);

  const onSubmit = async (values: FormValues) => {
    if (!projectSettings?.repos || projectSettings.repos.length === 0) {
      errorToast('No repositories configured for this project. Please configure repositories in Project Settings first.');
      return;
    }

    // Find the umbrella repo from project settings
    const umbrellaRepo = projectSettings.repos.find(r => r.name === values.umbrellaRepoName);
    if (!umbrellaRepo) {
      errorToast('Selected umbrella repository not found');
      return;
    }

    // Find supporting repos from project settings
    const supportingRepos = (values.supportingRepoNames || [])
      .map(name => projectSettings.repos?.find(r => r.name === name))
      .filter(r => r !== undefined)
      .map(r => ({ url: r!.url, branch: r!.defaultBranch || 'main' }));

    const request: CreateRFEWorkflowRequest = {
      title: values.title,
      description: values.description,
      branchName: values.branchName.trim(),
      workspacePath: values.workspacePath || undefined,
      parentOutcome: values.parentOutcome?.trim() || undefined,
      umbrellaRepo: {
        url: umbrellaRepo.url,
        branch: umbrellaRepo.defaultBranch || 'main',
      },
      supportingRepos,
    };

    createWorkflowMutation.mutate(
      { projectName, data: request },
      {
        onSuccess: (workflow) => {
          successToast(`RFE workspace "${values.title}" created successfully`);
          router.push(`/projects/${encodeURIComponent(projectName)}/rfe/${encodeURIComponent(workflow.id)}`);
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to create RFE workflow');
        },
      }
    );
  };

  return (
    <div className="container mx-auto py-8">
      <div className="max-w-4xl mx-auto">
        <Breadcrumbs
          items={[
            { label: 'Projects', href: '/projects' },
            { label: projectName, href: `/projects/${projectName}` },
            { label: 'RFE Workspaces', href: `/projects/${projectName}/rfe` },
            { label: 'New Workspace' },
          ]}
          className="mb-4"
        />
        <div className="flex items-center gap-4 mb-8">
          <Link href={`/projects/${encodeURIComponent(projectName)}/rfe`}>
            <Button variant="ghost" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to RFE Workspaces
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Create RFE Workspace</h1>
            <p className="text-muted-foreground">Set up a new Request for Enhancement workflow with AI agents</p>
          </div>
        </div>

        {/* Error state from mutation */}
        {createWorkflowMutation.isError && (
          <div className="mb-6">
            <ErrorMessage error={createWorkflowMutation.error} />
          </div>
        )}

        <Form {...form}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              form.handleSubmit(onSubmit)(e);
            }}
            className="space-y-8"
          >
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <GitBranch className="h-5 w-5" />
                  RFE Details
                </CardTitle>
                <CardDescription>Provide basic information about the feature or enhancement</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <FormField
                  control={form.control}
                  name="title"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>RFE Title</FormLabel>
                      <FormControl>
                        <Input placeholder="e.g., User Authentication System" {...field} />
                      </FormControl>
                      <FormDescription>
                        A concise title that describes the feature or enhancement.{' '}
                        <span className="font-medium text-foreground">This title will be used to generate the feature branch name.</span>
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
                        <Textarea placeholder="Describe the feature requirements, goals, and context..." rows={4} {...field} />
                      </FormControl>
                      <FormDescription>Detailed description of what needs to be built and why</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="branchName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Feature Branch</FormLabel>
                      <FormControl>
                        <Input placeholder="ambient-feature-name" {...field} />
                      </FormControl>
                      <FormDescription>
                        This feature branch will be created for all repositories configured in this RFE. Below, configure the Base Branch for each repository from which the feature branch will be created.
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="parentOutcome"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        Jira Outcome <span className="text-muted-foreground font-normal">(optional)</span>
                      </FormLabel>
                      <FormControl>
                        <Input placeholder="e.g., RHASTRAT-456" {...field} />
                      </FormControl>
                      <FormDescription>Jira Outcome key that Features created from this RFE will link to</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <GitBranch className="h-5 w-5" />
                  Repositories
                </CardTitle>
                <CardDescription>
                  Select repositories from your project settings. Feature branch{branchName && ` (${branchName})`} will be created from the default branch of each repository.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {settingsLoading ? (
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Loading project repositories...
                  </div>
                ) : settingsError ? (
                  <Alert>
                    <Info className="h-4 w-4" />
                    <AlertDescription>
                      Failed to load project settings. Please refresh the page or contact support.
                    </AlertDescription>
                  </Alert>
                ) : !projectSettings?.repos || projectSettings.repos.length === 0 ? (
                  <Alert>
                    <Info className="h-4 w-4" />
                    <AlertDescription>
                      No repositories configured for this project. Please configure repositories in Project Settings first.
                      <Link href={`/projects/${projectName}/settings`} className="ml-2 underline">
                        Go to Settings
                      </Link>
                    </AlertDescription>
                  </Alert>
                ) : (
                  <>
                    <FormField
                      control={form.control}
                      name="umbrellaRepoName"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Spec Repository</FormLabel>
                          <Select onValueChange={field.onChange} defaultValue={field.value}>
                            <FormControl>
                              <SelectTrigger>
                                <SelectValue placeholder="Select a repository" />
                              </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              {projectSettings.repos?.map((repo) => (
                                <SelectItem key={repo.name} value={repo.name}>
                                  {repo.name} ({repo.url})
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                          <FormDescription>
                            The spec repository contains your feature specifications, planning documents, and agent configurations
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="supportingRepoNames"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Supporting Repositories (optional)</FormLabel>
                          <FormDescription className="mb-2">
                            Select additional repositories that will be cloned alongside the spec repository
                          </FormDescription>
                          <div className="space-y-2">
                            {projectSettings.repos
                              ?.filter(r => r.name !== form.watch('umbrellaRepoName'))
                              .map((repo) => (
                                <label key={repo.name} className="flex items-center gap-2 cursor-pointer">
                                  <input
                                    type="checkbox"
                                    checked={field.value?.includes(repo.name) || false}
                                    onChange={(e) => {
                                      const currentValue = field.value || [];
                                      if (e.target.checked) {
                                        field.onChange([...currentValue, repo.name]);
                                      } else {
                                        field.onChange(currentValue.filter(name => name !== repo.name));
                                      }
                                    }}
                                    className="h-4 w-4"
                                  />
                                  <span className="text-sm">{repo.name} ({repo.url})</span>
                                </label>
                              ))}
                          </div>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </>
                )}
              </CardContent>
            </Card>

            <div className="flex justify-end gap-4">
              <Link href={`/projects/${encodeURIComponent(projectName)}/rfe`}>
                <Button variant="outline" disabled={createWorkflowMutation.isPending}>
                  Cancel
                </Button>
              </Link>
              <Button type="submit" disabled={createWorkflowMutation.isPending}>
                {createWorkflowMutation.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating RFE Workspace...
                  </>
                ) : (
                  'Create RFE Workspace'
                )}
              </Button>
            </div>
          </form>
        </Form>
      </div>
    </div>
  );
}
