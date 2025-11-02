'use client';

import React from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useForm, useFieldArray } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import Link from 'next/link';
import { Loader2, GitBranch } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { ErrorMessage } from '@/components/error-message';

import { useCreateRfeWorkflow } from '@/services/queries';
import { successToast, errorToast } from '@/hooks/use-toast';
import { Breadcrumbs } from '@/components/breadcrumbs';
import type { CreateRFEWorkflowRequest } from '@/types/api';

const repoSchema = z.object({
  url: z.string().url('Please enter a valid repository URL'),
  branch: z.string().min(1, 'Branch is required').default('main'),
});

const formSchema = z.object({
  title: z.string().min(5, 'Title must be at least 5 characters long'),
  description: z.string().min(20, 'Description must be at least 20 characters long'),
  branchName: z.string().min(1, 'Branch name is required'),
  workspacePath: z.string().optional(),
  parentOutcome: z.string().optional(),
  umbrellaRepo: repoSchema,
  supportingRepos: z.array(repoSchema).optional().default([]),
}).refine(
  (data) => {
    // Check for duplicate repositories
    const allUrls: string[] = [];

    // Add umbrella repo URL if present
    if (data.umbrellaRepo?.url) {
      allUrls.push(normalizeRepoUrl(data.umbrellaRepo.url));
    }

    // Add supporting repo URLs if present
    const supportingUrls = (data.supportingRepos || [])
      .filter(r => r?.url)
      .map(r => normalizeRepoUrl(r.url));

    allUrls.push(...supportingUrls);

    // Check for duplicates
    const uniqueUrls = new Set(allUrls);
    return uniqueUrls.size === allUrls.length;
  },
  {
    message: 'Duplicate repository URLs are not allowed. Each repository must be unique.',
    path: ['supportingRepos'],
  }
);

type FormValues = z.input<typeof formSchema>;

// Normalize repository URL for comparison (remove trailing slash and .git)
function normalizeRepoUrl(url: string): string {
  return url.trim().toLowerCase().replace(/\.git$/, '').replace(/\/$/, '');
}

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
      umbrellaRepo: { url: '', branch: 'main' },
      supportingRepos: [],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: 'supportingRepos',
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
    const request: CreateRFEWorkflowRequest = {
      title: values.title,
      description: values.description,
      branchName: values.branchName.trim(),
      workspacePath: values.workspacePath || undefined,
      parentOutcome: values.parentOutcome?.trim() || undefined,
      umbrellaRepo: {
        url: values.umbrellaRepo.url.trim(),
        branch: (values.umbrellaRepo.branch || 'main').trim(),
      },
      supportingRepos: (values.supportingRepos || [])
        .filter((r) => r && r.url && r.url.trim() !== '')
        .map((r) => ({ url: r.url.trim(), branch: r.branch?.trim() || 'main' })),
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
        <div className="mb-8">
          <h1 className="text-3xl font-bold">Create RFE Workspace</h1>
          <p className="text-muted-foreground">Set up a new Request for Enhancement workflow with AI agents</p>
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
                  Set the spec repo and optional supporting repos. Base branch is the branch from which the feature branch{branchName && ` (${branchName})`} will be set up. All modifications will be made to the feature branch.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4 items-end">
                  <div className="md:col-span-3">
                    <FormField
                      control={form.control}
                      name={`umbrellaRepo.url`}
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Spec Repo URL</FormLabel>
                          <FormControl>
                            <Input placeholder="https://github.com/org/repo.git" {...field} />
                          </FormControl>
                          <FormDescription>
                            The spec repository contains your feature specifications, planning documents, and agent configurations for this RFE workspace
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                  <div className="md:col-span-1">
                    <FormField
                      control={form.control}
                      name={`umbrellaRepo.branch`}
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
                </div>

                <div className="pt-2">
                  <div className="text-sm font-medium">Supporting Repositories (optional)</div>
                </div>
                {fields.map((field, index) => (
                  <div key={field.id} className="grid grid-cols-1 md:grid-cols-4 gap-4 items-end">
                    <div className="md:col-span-3">
                      <FormField
                        control={form.control}
                        name={`supportingRepos.${index}.url`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Repository URL</FormLabel>
                            <FormControl>
                              <Input placeholder="https://github.com/org/repo.git" {...field} />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>
                    <div className="md:col-span-1">
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

                    <div className="md:col-span-6 flex justify-end">
                      <Button type="button" variant="outline" size="sm" onClick={() => remove(index)}>
                        Remove
                      </Button>
                    </div>
                  </div>
                ))}
                <div className="space-y-2">
                  <Button type="button" variant="secondary" size="sm" onClick={() => append({ url: '', branch: 'main' })}>
                    Add supporting repo
                  </Button>
                  {form.formState.errors.supportingRepos?.message && (
                    <p className="text-sm font-medium text-destructive">
                      {form.formState.errors.supportingRepos.message}
                    </p>
                  )}
                </div>
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
