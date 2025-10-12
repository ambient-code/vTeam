"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { WorkflowPhase } from "@/types/agentic-session";
import { ArrowLeft } from "lucide-react";
import RepoBrowser from "@/components/RepoBrowser";
import type { GitHubFork } from "@/types";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { RfeSessionsTable } from "./rfe-sessions-table";
import { RfePhaseCards } from "./rfe-phase-cards";
import { RfeWorkspaceCard } from "./rfe-workspace-card";
import { RfeHeader } from "./rfe-header";
import { RfeAgentsCard } from "./rfe-agents-card";
import { useRfeWorkflow, useRfeWorkflowSessions, useDeleteRfeWorkflow } from "@/services/queries";

export default function ProjectRFEDetailPage() {
  const params = useParams();
  const router = useRouter();
  const project = params?.name as string;
  const id = params?.id as string;

  // React Query hooks
  const { data: workflow, isLoading: loading, refetch: load } = useRfeWorkflow(project, id);
  const { data: rfeSessions = [], refetch: loadSessions } = useRfeWorkflowSessions(project, id);
  const deleteWorkflowMutation = useDeleteRfeWorkflow();

  const [error, setError] = useState<string | null>(null);
  // const [advancing, _setAdvancing] = useState(false);
  const [startingPhase, setStartingPhase] = useState<WorkflowPhase | null>(null);
  // Workspace (PVC) removed: Git remote is source of truth
  const [activeTab, setActiveTab] = useState<string>("overview");
  const [selectedFork] = useState<GitHubFork | undefined>(undefined);
 
  // const [specBaseRelPath, _setSpecBaseRelPath] = useState<string>("specs");
  const [publishingPhase, setPublishingPhase] = useState<WorkflowPhase | null>(null);

  const [rfeDoc, setRfeDoc] = useState<{ exists: boolean; content: string }>({ exists: false, content: "" });
  const [firstFeaturePath, setFirstFeaturePath] = useState<string>("");
  const [specKitDir, setSpecKitDir] = useState<{
    spec: {
      exists: boolean;
      content: string;
    },
    plan: {
      exists: boolean;
      content: string;
    },
    tasks: {
      exists: boolean;
      content: string;
    }
  }>({
    spec: {
      exists: false,
      content: "",
    },
    plan: {
      exists: false,
      content: "",
    },
    tasks: {
      exists: false,
      content: "",
    }
  });

  const [seeding, setSeeding] = useState<boolean>(false);
  const [seedingStatus, setSeedingStatus] = useState<{ checking: boolean; isSeeded: boolean; error?: string }>({
    checking: true,
    isSeeded: false,
  });
  const [selectedAgents, setSelectedAgents] = useState<string[]>([]);

  const checkSeeding = useCallback(async () => {
    if (!project || !id || !workflow?.umbrellaRepo) return;
    try {
      setSeedingStatus({ checking: true, isSeeded: false });
      const resp = await fetch(`/api/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}/check-seeding`);
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      const data = await resp.json();
      setSeedingStatus({ checking: false, isSeeded: data.isSeeded });
    } catch (e) {
      setSeedingStatus({ 
        checking: false, 
        isSeeded: false, 
        error: e instanceof Error ? e.message : 'Failed to check seeding' 
      });
    }
  }, [project, id, workflow?.umbrellaRepo]);

  const checkPhaseDocuments = useCallback(async () => {
    if (!project || !id || !workflow?.umbrellaRepo) return;

    try {
      const repo = workflow.umbrellaRepo.url.replace(/^https?:\/\/(?:www\.)?github.com\//i, '').replace(/\.git$/i, '');
      const ref = workflow.umbrellaRepo.branch || 'main';

      // Check for rfe.md
      const rfeResp = await fetch(`/api/projects/${encodeURIComponent(project)}/repo/blob?repo=${encodeURIComponent(repo)}&ref=${encodeURIComponent(ref)}&path=${encodeURIComponent('rfe.md')}`);
      setRfeDoc({ exists: rfeResp.ok, content: '' });

      // Try to find specs directory structure
      const specsTreeResp = await fetch(`/api/projects/${encodeURIComponent(project)}/repo/tree?repo=${encodeURIComponent(repo)}&ref=${encodeURIComponent(ref)}&path=${encodeURIComponent('specs')}`);

      if (specsTreeResp.ok) {
        const specsTree = await specsTreeResp.json();
        const entries = specsTree.entries || [];
        console.log('[checkPhaseDocuments] specsTree response:', specsTree);
        console.log('[checkPhaseDocuments] entries:', entries);

        // Check for spec.md, plan.md, tasks.md directly in specs/
        const specResp = await fetch(`/api/projects/${encodeURIComponent(project)}/repo/blob?repo=${encodeURIComponent(repo)}&ref=${encodeURIComponent(ref)}&path=${encodeURIComponent('specs/spec.md')}`);
        const planResp = await fetch(`/api/projects/${encodeURIComponent(project)}/repo/blob?repo=${encodeURIComponent(repo)}&ref=${encodeURIComponent(ref)}&path=${encodeURIComponent('specs/plan.md')}`);
        const tasksResp = await fetch(`/api/projects/${encodeURIComponent(project)}/repo/blob?repo=${encodeURIComponent(repo)}&ref=${encodeURIComponent(ref)}&path=${encodeURIComponent('specs/tasks.md')}`);

        let specExists = specResp.ok;
        let planExists = planResp.ok;
        let tasksExists = tasksResp.ok;

        // If not found directly, check first subdirectory
        if (!specExists && !planExists && !tasksExists) {
          const firstDir = entries.find((e: { type: string; name?: string }) => e.type === 'tree');
          if (firstDir && firstDir.name) {
            const subPath = `specs/${firstDir.name}`;
            setFirstFeaturePath(subPath);
            const subSpecResp = await fetch(`/api/projects/${encodeURIComponent(project)}/repo/blob?repo=${encodeURIComponent(repo)}&ref=${encodeURIComponent(ref)}&path=${encodeURIComponent(`${subPath}/spec.md`)}`);
            const subPlanResp = await fetch(`/api/projects/${encodeURIComponent(project)}/repo/blob?repo=${encodeURIComponent(repo)}&ref=${encodeURIComponent(ref)}&path=${encodeURIComponent(`${subPath}/plan.md`)}`);
            const subTasksResp = await fetch(`/api/projects/${encodeURIComponent(project)}/repo/blob?repo=${encodeURIComponent(repo)}&ref=${encodeURIComponent(ref)}&path=${encodeURIComponent(`${subPath}/tasks.md`)}`);
            specExists = subSpecResp.ok;
            planExists = subPlanResp.ok;
            tasksExists = subTasksResp.ok;
          }
        } else {
          setFirstFeaturePath('specs');
        }

        setSpecKitDir({
          spec: { exists: specExists, content: '' },
          plan: { exists: planExists, content: '' },
          tasks: { exists: tasksExists, content: '' }
        });
      }
    } catch (e) {
      // Silently fail - we only need this to discover file paths for Jira integration
      // Button visibility is determined by session completion status
      console.debug('Failed to check phase documents:', e);
    }
  }, [project, id, workflow?.umbrellaRepo]);

  useEffect(() => { if (workflow) { checkSeeding(); checkPhaseDocuments(); } }, [workflow, checkSeeding, checkPhaseDocuments]);

  // Workspace probing removed

  // Workspace browse handlers removed

  const openJiraForPath = useCallback(async (relPath: string) => {
    try {
      const resp = await fetch(`/api/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}/jira?path=${encodeURIComponent(relPath)}`);
      if (!resp.ok) return;
      const data = await resp.json().catch(() => null);
      if (!data) return;
      const selfUrl = typeof data.self === 'string' ? data.self : '';
      const key = typeof data.key === 'string' ? data.key : '';
      if (selfUrl && key) {
        const origin = (() => { try { return new URL(selfUrl).origin; } catch { return ''; } })();
        if (origin) window.open(`${origin}/browse/${encodeURIComponent(key)}`, '_blank');
      }
    } catch {
      // noop
    }
  }, [project, id]);

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
    try {
      setSeeding(true);
      const resp = await fetch(`/api/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}/seed`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data?.error || `HTTP ${resp.status}`);
      }
      await checkSeeding(); // Re-check seeding status
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start seeding');
    } finally {
      setSeeding(false);
    }
  }, [project, id, checkSeeding]);


  if (loading) return <div className="container mx-auto py-8">Loading…</div>;
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

  // Seeding status is checked on-the-fly
  const isSeeded = seedingStatus.isSeeded;
  const seedingError = seedingStatus.error;

  return (
    <div className="container mx-auto py-8">
      <div className="max-w-6xl mx-auto space-y-8">
        <Breadcrumbs
          items={[
            { label: 'Projects', href: '/projects' },
            { label: project, href: `/projects/${project}` },
            { label: 'RFE Workspaces', href: `/projects/${project}/rfe` },
            { label: workflow.title },
          ]}
          className="mb-4"
        />
        <RfeHeader
          workflow={workflow}
          projectName={project}
          deleting={deleteWorkflowMutation.isPending}
          onDelete={deleteWorkflow}
        />

     

        <RfeWorkspaceCard
          workflow={workflow}
          workflowWorkspace={workflowWorkspace}
          isSeeded={isSeeded}
          seedingStatus={seedingStatus}
          seedingError={seedingError}
          seeding={seeding}
          onSeedWorkflow={seedWorkflow}
        />

        <RfeAgentsCard
          projectName={project}
          workflowId={id}
          selectedAgents={selectedAgents}
          onAgentsChange={setSelectedAgents}
        />

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
              defaultRef={selectedFork?.default_branch || workflow.umbrellaRepo?.branch || "main"}
            />
          </TabsContent>
        </Tabs>

      </div>
    </div>
  );
}
