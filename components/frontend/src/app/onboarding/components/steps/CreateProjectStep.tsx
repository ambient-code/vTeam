'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { AlertCircle, Loader2 } from 'lucide-react';
import { useCreateProject } from '@/services/queries';
import { useEffect } from 'react';

const projectSchema = z.object({
  name: z
    .string()
    .min(3, 'Project name must be at least 3 characters')
    .max(63, 'Project name must be at most 63 characters')
    .regex(/^[a-z0-9-]+$/, 'Project name must be lowercase alphanumeric with dashes')
    .regex(/^[a-z]/, 'Project name must start with a letter')
    .regex(/[a-z0-9]$/, 'Project name must end with a letter or number'),
  displayName: z
    .string()
    .min(1, 'Display name is required')
    .max(100, 'Display name must be at most 100 characters'),
  description: z.string().max(500, 'Description must be at most 500 characters').optional(),
});

type ProjectFormData = z.infer<typeof projectSchema>;

type CreateProjectStepProps = {
  onProjectCreated: (projectName: string) => void;
  initialData?: {
    name: string;
    displayName: string;
    description: string;
  };
};

export function CreateProjectStep({ onProjectCreated, initialData }: CreateProjectStepProps) {
  const createProjectMutation = useCreateProject();

  const form = useForm<ProjectFormData>({
    resolver: zodResolver(projectSchema),
    defaultValues: {
      name: initialData?.name || '',
      displayName: initialData?.displayName || '',
      description: initialData?.description || '',
    },
  });

  const onSubmit = (data: ProjectFormData) => {
    createProjectMutation.mutate(
      {
        name: data.name,
        displayName: data.displayName,
        description: data.description || '',
      },
      {
        onSuccess: () => {
          onProjectCreated(data.name);
        },
      }
    );
  };

  // Auto-generate project name from display name
  const watchDisplayName = form.watch('displayName');
  useEffect(() => {
    if (watchDisplayName && !form.formState.dirtyFields.name) {
      const kebabCase = watchDisplayName
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-+|-+$/g, '');
      form.setValue('name', kebabCase, { shouldValidate: true });
    }
  }, [watchDisplayName, form]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Create Your First Project</CardTitle>
        <CardDescription>
          Projects provide isolated namespaces for your agentic sessions and configurations
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
            <FormField
              control={form.control}
              name="displayName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Display Name</FormLabel>
                  <FormControl>
                    <Input placeholder="My First Project" {...field} />
                  </FormControl>
                  <FormDescription>A human-readable name for your project</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Project Name</FormLabel>
                  <FormControl>
                    <Input placeholder="my-first-project" {...field} />
                  </FormControl>
                  <FormDescription>
                    Unique identifier (lowercase, alphanumeric, dashes only)
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
                  <FormLabel>Description (Optional)</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder="Describe what you'll use this project for..."
                      className="resize-none"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Help your team understand the purpose of this project
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {createProjectMutation.isError && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  {createProjectMutation.error instanceof Error
                    ? createProjectMutation.error.message
                    : 'Failed to create project. Please try again.'}
                </AlertDescription>
              </Alert>
            )}

            <Button type="submit" disabled={createProjectMutation.isPending} className="w-full">
              {createProjectMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              {createProjectMutation.isPending ? 'Creating Project...' : 'Create Project'}
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  );
}

