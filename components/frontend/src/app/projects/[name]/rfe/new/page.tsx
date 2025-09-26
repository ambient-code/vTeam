"use client";

import { useState } from "react";
import { useRouter, useParams } from "next/navigation";
import { useForm, useFieldArray } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import Link from "next/link";
import { ArrowLeft, Loader2, GitBranch } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { CreateRFEWorkflowRequest } from "@/types/agentic-session";
import { getApiUrl } from "@/lib/config";

const repoSchema = z.object({
  url: z.string().url("Please enter a valid repository URL"),
  branch: z.string().min(1, "Branch is required").default("main"),
});

const formSchema = z.object({
  title: z.string().min(5, "Title must be at least 5 characters long"),
  description: z.string().min(20, "Description must be at least 20 characters long"),
  workspacePath: z.string().optional(),
  umbrellaRepo: repoSchema,
  supportingRepos: z.array(repoSchema).optional().default([]),
});

type FormValues = z.input<typeof formSchema>;

export default function ProjectNewRFEWorkflowPage() {
  const router = useRouter();
  const params = useParams();
  const project = params?.name as string;
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onBlur",
    defaultValues: {
      title: "",
      description: "",
      workspacePath: "",
      umbrellaRepo: { url: "", branch: "main" },
      supportingRepos: [],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "supportingRepos",
  });

  const onSubmit = async (values: FormValues) => {
    setIsSubmitting(true);
    setError(null);

    try {
      const request: CreateRFEWorkflowRequest = {
        title: values.title,
        description: values.description,
        workspacePath: values.workspacePath || undefined,
        umbrellaRepo: {
          url: values.umbrellaRepo.url.trim(),
          branch: (values.umbrellaRepo.branch || "main").trim(),
        },
        supportingRepos: (values.supportingRepos || [])
          .filter(r => r && r.url && r.url.trim() !== "")
          .map(r => ({ url: r.url.trim(), branch: r.branch?.trim() || "main"})),
      };

      const url = `${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows`;
      const response = await fetch(url, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(request) });
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        const message = errorData.error || errorData.message || `HTTP ${response.status}: ${response.statusText}`;
        throw new Error(message);
      }

      const result = await response.json();
      router.push(`/projects/${encodeURIComponent(project)}/rfe/${encodeURIComponent(result.id)}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create RFE workspace");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="container mx-auto py-8">
      <div className="max-w-4xl mx-auto">
        <div className="flex items-center gap-4 mb-8">
          <Link href={`/projects/${encodeURIComponent(project)}/rfe`}>
            <Button variant="ghost" size="sm"><ArrowLeft className="h-4 w-4 mr-2" />Back to RFE Workspaces</Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Create RFE Workspace</h1>
            <p className="text-muted-foreground">Set up a new Request for Enhancement workflow with AI agents</p>
          </div>
        </div>

        <Form {...form}>
          <form onSubmit={(e) => { e.preventDefault(); form.handleSubmit(onSubmit)(e); }} className="space-y-8">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2"><GitBranch className="h-5 w-5" />RFE Details</CardTitle>
                <CardDescription>Provide basic information about the feature or enhancement</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <FormField control={form.control} name="title" render={({ field }) => (
                  <FormItem>
                    <FormLabel>RFE Title</FormLabel>
                    <FormControl><Input placeholder="e.g., User Authentication System" {...field} /></FormControl>
                    <FormDescription>A concise title that describes the feature or enhancement</FormDescription>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={form.control} name="description" render={({ field }) => (
                  <FormItem>
                    <FormLabel>Description</FormLabel>
                    <FormControl><Textarea placeholder="Describe the feature requirements, goals, and context..." rows={4} {...field} /></FormControl>
                    <FormDescription>Detailed description of what needs to be built and why</FormDescription>
                    <FormMessage />
                  </FormItem>
                )} />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2"><GitBranch className="h-5 w-5" />Repositories</CardTitle>
                <CardDescription>Set the umbrella repo and optional supporting repos</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4 items-end">
                  <div className="md:col-span-3">
                    <FormField control={form.control} name={`umbrellaRepo.url`} render={({ field }) => (
                      <FormItem>
                        <FormLabel>Umbrella Repo URL</FormLabel>
                        <FormControl><Input placeholder="https://github.com/org/repo.git" {...field} /></FormControl>
                        <FormMessage />
                      </FormItem>
                    )} />
                  </div>
                  <div className="md:col-span-1">
                    <FormField control={form.control} name={`umbrellaRepo.branch`} render={({ field }) => (
                      <FormItem>
                        <FormLabel>Branch</FormLabel>
                        <FormControl><Input placeholder="main" {...field} /></FormControl>
                        <FormMessage />
                      </FormItem>
                    )} />
                  </div>
                  
                </div>

                <div className="pt-2">
                  <div className="text-sm font-medium">Supporting Repositories (optional)</div>
                </div>
                {fields.map((field, index) => (
                  <div key={field.id} className="grid grid-cols-1 md:grid-cols-4 gap-4 items-end">
                    <div className="md:col-span-3">
                      <FormField control={form.control} name={`supportingRepos.${index}.url`} render={({ field }) => (
                        <FormItem>
                          <FormLabel>Repository URL</FormLabel>
                          <FormControl><Input placeholder="https://github.com/org/repo.git" {...field} /></FormControl>
                          <FormMessage />
                        </FormItem>
                      )} />
                    </div>
                    <div className="md:col-span-1">
                      <FormField control={form.control} name={`supportingRepos.${index}.branch`} render={({ field }) => (
                        <FormItem>
                          <FormLabel>Branch</FormLabel>
                          <FormControl><Input placeholder="main" {...field} /></FormControl>
                          <FormMessage />
                        </FormItem>
                      )} />
                    </div>
                    
                    <div className="md:col-span-6 flex justify-end">
                      <Button type="button" variant="outline" size="sm" onClick={() => remove(index)}>Remove</Button>
                    </div>
                  </div>
                ))}
                <div>
                  <Button type="button" variant="secondary" size="sm" onClick={() => append({ url: "", branch: "main" })}>Add supporting repo</Button>
                </div>
              </CardContent>
            </Card>
            {/* Agent selection omitted in this simplified flow */}

            {error && (
              <Card className="border-red-200 bg-red-50"><CardContent className="pt-6"><p className="text-red-600 text-sm">{error}</p></CardContent></Card>
            )}

            <div className="flex justify-end gap-4">
              <Link href={`/projects/${encodeURIComponent(project)}/rfe`}>
                <Button variant="outline" disabled={isSubmitting}>Cancel</Button>
              </Link>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? (<><Loader2 className="mr-2 h-4 w-4 animate-spin" />Creating RFE Workspace...</>) : ("Create RFE Workspace")}
              </Button>
            </div>
          </form>
        </Form>
      </div>
    </div>
  );
}
