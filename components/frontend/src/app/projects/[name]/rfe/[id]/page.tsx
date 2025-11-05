"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { WorkflowPhase } from "@/types/agentic-session";
import { ArrowLeft, Loader2, Bot } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import RepoBrowser from "@/components/RepoBrowser";
import type { GitHubFork } from "@/types";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { RfeSessionsTable } from "./rfe-sessions-table";
import { RfePhaseCards } from "./rfe-phase-cards";
import { RfeWorkspaceCard } from "./rfe-workspace-card";
import { RfeHeader } from "./rfe-header";
import { RfeAgentsCard } from "./rfe-agents-card";
import { AVAILABLE_AGENTS } from "@/lib/agents";
import { useRfeWorkflow, useRfeWorkflowSessions, useDeleteRfeWorkflow, useRfeWorkflowSeeding, useSeedRfeWorkflow, useUpdateRfeWorkflow, useRepoBlob, useRepoTree, useOpenJiraIssue, useRfeWorkflowAgents } from "@/services/queries";

export default function ProjectRFEDetailPage() {
  const params = useParams();
  const router = useRouter();
  const project = params?.name as string;
  const id = params?.id as string;

  // React Query hooks
  const { data: workflow, isLoading: loading, refetch: load } = useRfeWorkflow(project, id);
  const { data: rfeSessions = [], refetch: loadSessions } = useRfeWorkflowSessions(project, id);
  const deleteWorkflowMutation = useDeleteRfeWorkflow();
  const { data: seedingData, isLoading: checkingSeeding, error: seedingQueryError, refetch: refetchSeeding } = useRfeWorkflowSeeding(project, id);
  const seedWorkflowMutation = useSeedRfeWorkflow();
  const updateWorkflowMutation = useUpdateRfeWorkflow();
  const { openJiraForPath } = useOpenJiraIssue(project, id);
  const { data: repoAgents = AVAILABLE_AGENTS, isLoading: loadingAgents } = useRfeWorkflowAgents(project, id);

  // Extract repo info from workflow
  const repo = workflow?.umbrellaRepo?.url.replace(/^https?:\/\/(?:www\.)?github.com\//i, '').replace(/\.git$/i, '') || '';
  // Use feature branch if available, otherwise fall back to base branch
  const ref = workflow?.branchName || workflow?.umbrellaRepo?.branch || 'main';
  const hasRepoInfo = !!workflow?.umbrellaRepo && !!repo;

  // Fetch rfe.md
  const { data: rfeBlob } = useRepoBlob(
    project,
    { repo, ref, path: 'rfe.md' },
    { enabled: hasRepoInfo }
  );

  // Fetch specs directory tree
  const { data: specsTree } = useRepoTree(
    project,
    { repo, ref, path: 'specs' },
    { enabled: hasRepoInfo }
  );

  // Find first subdirectory in specs tree
  const firstSubDir = specsTree?.entries?.find((e: { type: string; name?: string }) => e.type === 'tree')?.name || '';
  const subPath = firstSubDir ? `specs/${firstSubDir}` : '';
  
  // Fetch spec files from subdirectory (files are always in subdirs, never in root specs/)
  const { data: specBlob } = useRepoBlob(
    project,
    { repo, ref, path: subPath ? `${subPath}/spec.md` : '' },
    { enabled: hasRepoInfo && !!subPath }
  );
  const { data: planBlob } = useRepoBlob(
    project,
    { repo, ref, path: subPath ? `${subPath}/plan.md` : '' },
    { enabled: hasRepoInfo && !!subPath }
  );
  const { data: tasksBlob } = useRepoBlob(
    project,
    { repo, ref, path: subPath ? `${subPath}/tasks.md` : '' },
    { enabled: hasRepoInfo && !!subPath }
  );

  const [error, setError] = useState<string | null>(null);
  // const [advancing, _setAdvancing] = useState(false);
  const [startingPhase, setStartingPhase] = useState<WorkflowPhase | null>(null);
  // Workspace (PVC) removed: Git remote is source of truth
  const [activeTab, setActiveTab] = useState<string>("overview");
  const [selectedFork] = useState<GitHubFork | undefined>(undefined);
 
  // const [specBaseRelPath, _setSpecBaseRelPath] = useState<string>("specs");
  const [publishingPhase, setPublishingPhase] = useState<WorkflowPhase | null>(null);

  const [selectedAgents, setSelectedAgents] = useState<string[]>([]);

  // Process rfe.md blob data
  const [rfeDoc, setRfeDoc] = useState<{ exists: boolean; content: string }>({ exists: false, content: "" });
  useEffect(() => {
    if (!rfeBlob) return;
    
    (async () => {
      if (rfeBlob.ok) {
        const content = await rfeBlob.clone().text();
        setRfeDoc({ exists: true, content });
      } else {
        setRfeDoc({ exists: false, content: '' });
      }
    })();
  }, [rfeBlob]);

  // Process spec kit blobs from subdirectory
  const [specKitDir, setSpecKitDir] = useState<{
    spec: { exists: boolean; content: string; },
    plan: { exists: boolean; content: string; },
    tasks: { exists: boolean; content: string; }
  }>({
    spec: { exists: false, content: '' },
    plan: { exists: false, content: '' },
    tasks: { exists: false, content: '' }
  });

  useEffect(() => {
    (async () => {
      const specData = specBlob?.ok
        ? { exists: true, content: await specBlob.clone().text() }
        : { exists: false, content: '' };
      
      const planData = planBlob?.ok
        ? { exists: true, content: await planBlob.clone().text() }
        : { exists: false, content: '' };
      
      const tasksData = tasksBlob?.ok
        ? { exists: true, content: await tasksBlob.clone().text() }
        : { exists: false, content: '' };

      setSpecKitDir({ spec: specData, plan: planData, tasks: tasksData });
    })();
  }, [specBlob, planBlob, tasksBlob]);

  const firstFeaturePath = subPath;


  const deleteWorkflow = useCallback(async () => {
    if (!confirm('Are you sure you want to delete this RFE workflow? This action cannot be undone.')) {
      return;
    }
    return new Promise<void>((resolve, reject) => {
      deleteWorkflowMutation.mutate(
        { projectName: project, workflowId: id },
        {
          onSuccess: () => {
            router.push(`/projects/${encodeURIComponent(project)}/rfe`);
            resolve();
          },
          onError: (err) => {
            setError(err.message || 'Failed to delete workflow');
            reject(err);
          },
        }
      );
    });
  }, [project, id, deleteWorkflowMutation, router]);

  const seedWorkflow = useCallback(async () => {
    return new Promise<void>((resolve, reject) => {
      seedWorkflowMutation.mutate(
        { projectName: project, workflowId: id },
        {
          onSuccess: () => {
            resolve();
          },
          onError: (err) => {
            // Don't set page-level error - let RfeWorkspaceCard show the inline error
            // The error is available via seedWorkflowMutation.error
            reject(err);
          },
        }
      );
    });
  }, [project, id, seedWorkflowMutation]);

  const updateRepositories = useCallback(async (data: { umbrellaRepo: { url: string; branch?: string }; supportingRepos: { url: string; branch?: string }[] }) => {
    return new Promise<void>((resolve, reject) => {
      updateWorkflowMutation.mutate(
        {
          projectName: project,
          workflowId: id,
          data: {
            umbrellaRepo: data.umbrellaRepo,
            supportingRepos: data.supportingRepos,
          },
        },
        {
          onSuccess: () => {
            // Refetch workflow to get updated data
            load();
            // Also refetch seeding status to clear any errors
            refetchSeeding();
            // Clear any previous seeding errors
            seedWorkflowMutation.reset();
            resolve();
          },
          onError: (err) => {
            setError(err.message || 'Failed to update repositories');
            reject(err);
          },
        }
      );
    });
  }, [project, id, updateWorkflowMutation, load, refetchSeeding, seedWorkflowMutation]);


  if (loading) return (
    <div className="flex items-center justify-center min-h-screen">
      <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
    </div>
  );
  if (error || !workflow) return (
    <div className="container mx-auto py-8">
      <Card className="border-red-200 bg-red-50">
        <CardContent className="pt-6">
          <p className="text-red-600">{error || "Not found"}</p>
          <Link href={`/projects/${encodeURIComponent(project)}/rfe`}>
            <Button variant="outline" className="mt-4"><ArrowLeft className="mr-2 h-4 w-4" />Back</Button>
          </Link>
        </CardContent>
      </Card>
    </div>
  );

  const workflowWorkspace = workflow.workspacePath || `/rfe-workflows/${id}/workspace`;
  const upstreamRepo = workflow?.umbrellaRepo?.url || "";

  // Seeding status from React Query
  const isSeeded = seedingData?.isSeeded || false;
  // Combine seed mutation error with check-seeding query error
  const seedingError = seedWorkflowMutation.error?.message || seedingQueryError?.message;
  // Track if we've completed the initial seeding check
  const hasCheckedSeeding = seedingData !== undefined || !!seedingQueryError;
  const seedingStatus = {
    checking: checkingSeeding,
    isSeeded,
    error: seedingError,
    hasChecked: hasCheckedSeeding,
  };

  return (
    <div className="container mx-auto py-8">
      <div className="max-w-6xl mx-auto space-y-8">
        <Breadcrumbs
          items={[
            { label: 'Workspaces', href: '/projects' },
            { label: project, href: `/projects/${project}` },
            { label: 'RFE Workspaces', href: `/projects/${project}/rfe` },
            { label: workflow.title },
          ]}
          className="mb-4"
        />
        <RfeHeader
          workflow={workflow}
          deleting={deleteWorkflowMutation.isPending}
          onDelete={deleteWorkflow}
        />

        <RfeWorkspaceCard
          workflow={workflow}
          workflowWorkspace={workflowWorkspace}
          isSeeded={isSeeded}
          seedingStatus={seedingStatus}
          seedingError={seedingError}
          seeding={seedWorkflowMutation.isPending}
          onSeedWorkflow={seedWorkflow}
          onUpdateRepositories={updateRepositories}
          updating={updateWorkflowMutation.isPending}
        />

        {/* Two Column Layout */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Left Column - Tabs */}
          <div className="lg:col-span-1">
            <Tabs value={activeTab} onValueChange={setActiveTab}>
              <TabsList>
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="sessions">Sessions</TabsTrigger>
                {upstreamRepo ? <TabsTrigger value="browser">Repository</TabsTrigger> : null}
              </TabsList>

              <TabsContent value="overview">
                <RfePhaseCards
                  workflow={workflow}
                  rfeSessions={rfeSessions}
                  rfeDoc={rfeDoc}
                  specKitDir={specKitDir}
                  firstFeaturePath={firstFeaturePath}
                  projectName={project}
                  rfeId={id}
                  workflowWorkspace={workflowWorkspace}
                  isSeeded={isSeeded}
                  startingPhase={startingPhase}
                  publishingPhase={publishingPhase}
                  selectedAgents={selectedAgents}
                  onStartPhase={setStartingPhase}
                  onPublishPhase={setPublishingPhase}
                  onLoad={async () => { await load(); }}
                  onLoadSessions={async () => { await loadSessions(); }}
                  onError={setError}
                  onOpenJira={openJiraForPath}
                />
              </TabsContent>

              <TabsContent value="sessions">
                <RfeSessionsTable
                  sessions={rfeSessions}
                  projectName={project}
                  rfeId={id}
                  workspacePath={workflowWorkspace}
                  workflowId={workflow.id}
                />
              </TabsContent>

              <TabsContent value="browser">
                <RepoBrowser
                  projectName={project}
                  repoUrl={selectedFork?.url || upstreamRepo}
                  defaultRef={selectedFork?.default_branch || workflow.branchName || workflow.umbrellaRepo?.branch || "main"}
                />
              </TabsContent>
            </Tabs>
          </div>

          {/* Right Column - Agents Accordion */}
          <div className="lg:col-span-1">
            <Card>
              <CardContent className="p-4">
                <Accordion type="multiple" className="w-full">
                  <AccordionItem value="agents" className="border-none">
                    <AccordionTrigger className="text-lg font-semibold hover:no-underline px-0">
                      <div className="flex items-center gap-2">
                        <Bot className="h-5 w-5" />
                        Agents
                      </div>
                    </AccordionTrigger>
                    <AccordionContent className="pt-4">
                      {loadingAgents ? (
                        <div className="flex items-center justify-center py-8">
                          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                        </div>
                      ) : repoAgents.length === 0 ? (
                        <div className="text-center py-8 text-muted-foreground">
                          <Bot className="h-12 w-12 mx-auto mb-2 opacity-50" />
                          <p>No agents found in repository .claude/agents directory</p>
                          <p className="text-xs mt-1">Seed the repository to add agent definitions</p>
                        </div>
                      ) : (
                        <>
                          <div className="grid grid-cols-1 gap-3">
                            {repoAgents.map((agent) => {
                              const isSelected = selectedAgents.includes(agent.persona);
                              return (
                                <div
                                  key={agent.persona}
                                  className={`p-3 rounded-lg border transition-colors ${
                                    isSelected ? 'bg-primary/5 border-primary' : 'bg-background border-border hover:border-primary/50'
                                  }`}
                                >
                                  <label className="flex items-start gap-3 cursor-pointer">
                                    <Checkbox
                                      checked={isSelected}
                                      onCheckedChange={(checked) => {
                                        setSelectedAgents(
                                          checked
                                            ? [...selectedAgents, agent.persona]
                                            : selectedAgents.filter(p => p !== agent.persona)
                                        );
                                      }}
                                      className="mt-0.5"
                                    />
                                    <div className="flex-1 min-w-0">
                                      <div className="font-medium text-sm">{agent.name}</div>
                                      <div className="text-xs text-muted-foreground mt-0.5">{agent.role}</div>
                                    </div>
                                  </label>
                                </div>
                              );
                            })}
                          </div>
                          {selectedAgents.length > 0 && (
                            <div className="mt-4 pt-4 border-t">
                              <div className="text-sm font-medium mb-2">Selected Agents ({selectedAgents.length})</div>
                              <div className="flex flex-wrap gap-2">
                                {selectedAgents.map(persona => {
                                  const agent = repoAgents.find(a => a.persona === persona);
                                  return agent ? (
                                    <Badge key={persona} variant="secondary">
                                      {agent.name}
                                    </Badge>
                                  ) : null;
                                })}
                              </div>
                            </div>
                          )}
                        </>
                      )}
                    </AccordionContent>
                  </AccordionItem>
                </Accordion>
              </CardContent>
            </Card>
          </div>
        </div>

      </div>
    </div>
  );
}
