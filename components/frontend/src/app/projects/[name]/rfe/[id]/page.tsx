"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { getApiUrl } from "@/lib/config";
import { formatDistanceToNow } from "date-fns";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { AgenticSession, CreateAgenticSessionRequest, RFEWorkflow, WorkflowPhase } from "@/types/agentic-session";
import { WORKFLOW_PHASE_LABELS } from "@/lib/agents";
import { ArrowLeft, Play, Loader2, FolderTree, Plus, Trash2 } from "lucide-react";
import { Upload, CheckCircle2 } from "lucide-react";
import { FileTree, type FileTreeNode } from "@/components/file-tree";

export default function ProjectRFEDetailPage() {
  const params = useParams();
  const project = params?.name as string;
  const id = params?.id as string;

  const [workflow, setWorkflow] = useState<RFEWorkflow | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [advancing, setAdvancing] = useState(false);
  const [startingPhase, setStartingPhase] = useState<WorkflowPhase | null>(null);
  const [rfeSessions, setRfeSessions] = useState<AgenticSession[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [hasWorkspace, setHasWorkspace] = useState<boolean | null>(null);
  const [wsTree, setWsTree] = useState<FileTreeNode[]>([]);
  const [wsSelectedPath, setWsSelectedPath] = useState<string | undefined>(undefined);
  const [wsFileContent, setWsFileContent] = useState<string>("");
  const [wsLoading, setWsLoading] = useState<boolean>(false);
  const [activeTab, setActiveTab] = useState<string>("overview");
 
  const [specBaseRelPath, setSpecBaseRelPath] = useState<string>("specs");
  const [publishingPhase, setPublishingPhase] = useState<WorkflowPhase | null>(null);
  const [deleting, setDeleting] = useState<boolean>(false);

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

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}`);
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      const wf: RFEWorkflow = await resp.json();
      setWorkflow(wf);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [project, id]);

  const loadSessions = useCallback(async () => {
    if (!project || !id) return;
    try {
      setSessionsLoading(true);
      const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}/sessions`);
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      const data = await resp.json();
      setRfeSessions(Array.isArray(data.sessions) ? data.sessions : []);
    } catch {
      setRfeSessions([]);
    } finally {
      setSessionsLoading(false);
    }
  }, [project, id]);

  useEffect(() => { if (project && id) { load(); loadSessions(); } }, [project, id, load, loadSessions]);

  const listWsPath = useCallback(async (subpath?: string) => {
    if (!project || !id) return [] as Array<{ name: string; path: string; isDir: boolean; size?: number }>;
    try {
      const qs = subpath ? `?path=${encodeURIComponent(subpath)}` : "";
      const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}/workspace${qs}`);
      if (!resp.ok) return [];
      const txt = await resp.text();
      try {
        const data = JSON.parse(txt);
        return Array.isArray(data.items) ? data.items : [];
      } catch {
        return [];
      }
    } catch {
      return [];
    }
  }, [project, id]);

  const fetchWsFile = useCallback(async (relPath: string): Promise<{ exists: boolean; content: string }> => {
    if (!project || !id) return { exists: false, content: "" };
    try {
      const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}/workspace/${encodeURIComponent(relPath)}`);
      if (!resp.ok) {
        if (resp.status === 404) return { exists: false, content: "" };
        return { exists: false, content: "" };
      }
      const contentType = resp.headers.get("content-type") || "";
      if (contentType.startsWith("application/json")) {
        const data = await resp.json();
        return { exists: true, content: JSON.stringify(data, null, 2) };
      }
      const text = await resp.text();
      return { exists: true, content: text };
    } catch {
      return { exists: false, content: "" };
    }
  }, [project, id]);

  const saveWsFile = useCallback(async (relPath: string, content: string) => {
    if (!project || !id) return false;
    try {
      const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}/workspace/${encodeURIComponent(relPath)}`, {
        method: "PUT",
        headers: { "Content-Type": "text/plain; charset=utf-8" },
        body: content,
      });
      return resp.ok;
    } catch {
      return false;
    }
  }, [project, id]);

  const probeWorkspaceAndPhase = useCallback(async () => {
    // Probe ideation doc at workspace root
    const rfe = await fetchWsFile("rfe.md");
    setRfeDoc(rfe);

    const features = await listWsPath("specs");
    const firstFeature = features && features.length > 0 ? features[0] : undefined;

    if (!firstFeature || !firstFeature.path) {
      setSpecBaseRelPath("specs");
      setFirstFeaturePath("");
      setSpecKitDir({
        spec: { exists: false, content: "" },
        plan: { exists: false, content: "" },
        tasks: { exists: false, content: "" },
      });
      return;
    }

    setSpecBaseRelPath(String(firstFeature.path));
    setFirstFeaturePath(firstFeature.path);
    // Probe spec.md, plan.md, tasks.md by fetching files
    const [spec, plan, tasks] = await Promise.all([
      fetchWsFile(`${firstFeature.path}/spec.md`),
      fetchWsFile(`${firstFeature.path}/plan.md`),
      fetchWsFile(`${firstFeature.path}/tasks.md`),
    ]);

    setSpecKitDir({
      spec: {
        exists: spec.exists,
        content: spec.content,
      },
      plan: {
        exists: plan.exists,
        content: plan.content,
      },
      tasks: {
        exists: tasks.exists,
        content: tasks.content,
      },
    });


  }, [listWsPath, fetchWsFile]);

  useEffect(() => {
    (async () => {
      if (!project || !id) return;
      const items = await listWsPath();
      setHasWorkspace(items.length >= 0);
      const children: FileTreeNode[] = items.map((it: any) => ({
        name: it.name,
        path: (it.path || it.name).replace(/^\/+/, ""),
        type: it.isDir ? "folder" : "file",
        expanded: false,
        sizeKb: typeof it.size === "number" ? it.size / 1024 : undefined,
      }));
      setWsTree(children);
      await probeWorkspaceAndPhase();
    })();
  }, [project, id, listWsPath, probeWorkspaceAndPhase]);

  const updateChildrenByPath = useCallback((nodes: FileTreeNode[], targetPath: string, children: FileTreeNode[]): FileTreeNode[] => {
    return nodes.map((n) => {
      if (n.path === targetPath) {
        return { ...n, children };
      }
      if (n.type === "folder" && n.children && n.children.length > 0) {
        return { ...n, children: updateChildrenByPath(n.children, targetPath, children) };
      }
      return n;
    });
  }, []);

  const onWsToggle = useCallback(async (node: FileTreeNode) => {
    if (node.type !== "folder") return;
    const items = await listWsPath(node.path);
    const children: FileTreeNode[] = items.map((it: any) => ({
      name: it.name,
      path: `${node.path}/${it.name}`.replace(/^\/+/, ""),
      type: it.isDir ? "folder" : "file",
      expanded: false,
      sizeKb: typeof it.size === "number" ? it.size / 1024 : undefined,
    }));
    setWsTree(prev => updateChildrenByPath(prev, node.path, children));
  }, [listWsPath, updateChildrenByPath]);

  const onWsSelect = useCallback(async (node: FileTreeNode) => {
    if (node.type !== "file") return;
    setWsSelectedPath(node.path);
    setWsLoading(true);
    try {
      const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}/workspace/${encodeURIComponent(node.path)}`);
      if (!resp.ok) { setWsFileContent("Failed to load file"); return; }
      const contentType = resp.headers.get("content-type") || "application/octet-stream";
      if (contentType.startsWith("application/json")) {
        const data = await resp.json();
        setWsFileContent(JSON.stringify(data, null, 2));
      } else {
        const text = await resp.text();
        setWsFileContent(text);
      }
    } catch {
      setWsFileContent("Failed to load file");
    } finally {
      setWsLoading(false);
    }
  }, [project, id]);

  const openJiraForPath = useCallback(async (relPath: string) => {
    try {
      const resp = await fetch(`/api/projects/${encodeURIComponent(project)}/rfe/${encodeURIComponent(id)}/jira?path=${encodeURIComponent(relPath)}`);
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
    try {
      setDeleting(true);
      const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      });
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      // Navigate back to RFE list after successful deletion
      window.location.href = `/projects/${encodeURIComponent(project)}/rfe`;
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete workflow');
    } finally {
      setDeleting(false);
    }
  }, [project, id]);

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

  const currentPhase: WorkflowPhase = (!rfeDoc.exists) ? "pre" : (!specKitDir.spec.exists ? "specify" : (!specKitDir.plan.exists ? "plan" : (!specKitDir.tasks.exists ? "tasks" : "completed")));
  const workflowWorkspace = workflow.workspacePath || `/rfe-workflows/${id}/workspace`;
  const phases: WorkflowPhase[] = ["pre", "ideate", "specify", "plan", "tasks"];
  const idx = phases.indexOf(currentPhase);
  const totalCostUsd = (rfeSessions || []).reduce((sum, s) => sum + (typeof s.status?.total_cost_usd === "number" ? s.status.total_cost_usd : 0), 0);
  const workflowSessions = (rfeSessions || []);
  const totalSessionsCount = workflowSessions.length;
  const completedSessionsCount = workflowSessions.filter(s => (s.status?.phase || '').toLowerCase() === 'completed').length;
  const runningSessionsCount = workflowSessions.filter(s => (s.status?.phase || '').toLowerCase() === 'running').length;
  const failedSessionsCount = workflowSessions.filter(s => (s.status?.phase || '').toLowerCase() === 'failed').length;

  const phaseProgress = (phase: WorkflowPhase): number => {
    // If the expected document for the phase exists, treat as complete
    if (phase === "ideate" && rfeDoc.exists) return 100;
    if (phase === "specify" && specKitDir.spec.exists) return 100;
    if (phase === "plan" && specKitDir.plan.exists) return 100;
    if (phase === "tasks" && specKitDir.tasks.exists) return 100;
    // For pre phase, nothing produced yet
    if (phase === "pre") return 0;

    // Otherwise, use the session status for that phase if present
    const sessionForPhase = rfeSessions.find(s => (s.metadata.labels)?.["rfe-phase"] === phase);
    const status = sessionForPhase?.status?.phase;
    if (status === "Completed") return 100;
    if (status === "Running" || status === "Creating") return 50;
    if (status === "Failed" || status === "Error" || status === "Stopped") return 0;
    return 0;
  };

  return (
    <div className="container mx-auto py-8">
      <div className="max-w-6xl mx-auto space-y-8">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
            <Link href={`/projects/${encodeURIComponent(project)}/rfe`}>
              <Button variant="ghost" size="sm"><ArrowLeft className="h-4 w-4 mr-2" />Back to RFE Workspaces</Button>
            </Link>
            <div>
              <h1 className="text-3xl font-bold">{workflow.title}</h1>
              <p className="text-muted-foreground mt-1">{workflow.description}</p>
            </div>
          </div>
          <Button
            variant="destructive"
            size="sm"
            onClick={deleteWorkflow}
            disabled={deleting}
          >
            {deleting ? (
              <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Deleting…</>
            ) : (
              <><Trash2 className="mr-2 h-4 w-4" />Delete Workflow</>
            )}
          </Button>
        </div>

        <div className="grid gap-6 md:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Current Phase</CardTitle>
              <Badge className="bg-blue-100 text-blue-800">{WORKFLOW_PHASE_LABELS[currentPhase]}</Badge>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{phaseProgress(currentPhase).toFixed(0)}%</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Sessions</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{totalSessionsCount}</div>
              <div className="text-xs text-muted-foreground mt-1">
                Completed: {completedSessionsCount} • Running: {runningSessionsCount} • Failed: {failedSessionsCount}
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Phase Files</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{(specKitDir.spec.exists ? 1 : 0) + (specKitDir.plan.exists ? 1 : 0) + (specKitDir.tasks.exists ? 1 : 0)}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Cost</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">${totalCostUsd.toFixed(4)}</div>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2"><FolderTree className="h-5 w-5" />Workspace & Repositories</CardTitle>
            <CardDescription>Shared workspace for this workflow and optional repos</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="text-sm text-muted-foreground">Workspace: {workflowWorkspace}</div>
            {(workflow.repositories || []).length > 0 && (
              <div className="mt-2 space-y-1">
                {(workflow.repositories || []).map((r, i) => (
                  <div key={i} className="text-sm">
                    <span className="font-medium">{r.url}</span>
                    {r.branch && <span className="text-muted-foreground"> @ {r.branch}</span>}
                    {r.clonePath && <span className="text-muted-foreground"> → {r.clonePath}</span>}
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="sessions">Sessions</TabsTrigger>
            {hasWorkspace ? <TabsTrigger value="workspace">Workspace</TabsTrigger> : null}
          </TabsList>

          <TabsContent value="overview">
            <Card>
              <CardHeader>
                <CardTitle>Phase Documents</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {(() => {
                    const phaseList = ["ideate","specify","plan","tasks"] as const;
                    return phaseList.map(phase => {
                      const expected = (() => {
                        if (phase === "ideate") return "rfe.md";
                        if (!firstFeaturePath) {
                          // Fallback to just filename if no feature path found
                          if (phase === "specify") return "spec.md";
                          if (phase === "plan") return "plan.md";
                          return "tasks.md";
                        }
                        // Use full path with subdirectory
                        if (phase === "specify") return `${firstFeaturePath}/spec.md`;
                        if (phase === "plan") return `${firstFeaturePath}/plan.md`;
                        return `${firstFeaturePath}/tasks.md`;
                      })();
                      const exists = phase === "ideate" ? rfeDoc.exists : (phase === "specify" ? specKitDir.spec.exists : phase === "plan" ? specKitDir.plan.exists : specKitDir.tasks.exists);
                      const linkedKey = Array.isArray((workflow as any).jiraLinks) ? ((workflow as any).jiraLinks as Array<{ path: string; jiraKey: string }>).find(l => l.path === expected)?.jiraKey : undefined;
                      const sessionForPhase = rfeSessions.find(s => (s.metadata.labels)?.["rfe-phase"] === phase);
                     
                      const prerequisitesMet = phase === "ideate" ? true : phase === "specify" ? true : phase === "plan" ? specKitDir.spec.exists : (specKitDir.spec.exists && specKitDir.plan.exists);
                      const sessionDisplay = ((sessionForPhase as any)?.spec?.displayName) || sessionForPhase?.metadata.name;
                      return (
                        <div key={phase} className={`p-4 rounded-lg border flex items-center justify-between ${exists ? "bg-green-50 border-green-200" : ""}`}>
                          <div className="flex flex-col gap-1">
                            <div className="flex items-center gap-3">
                              <Badge variant="outline">{WORKFLOW_PHASE_LABELS[phase]}</Badge>
                              <span className="text-sm text-muted-foreground">{expected}</span>
                            </div>
                            {sessionForPhase && (
                              <div className="flex items-center gap-2">
                                <Link href={{
                                  pathname: `/projects/${encodeURIComponent(project)}/sessions/${encodeURIComponent(sessionForPhase.metadata.name)}`,
                                  query: {
                                    backHref: `/projects/${encodeURIComponent(project)}/rfe/${encodeURIComponent(id)}?tab=overview`,
                                    backLabel: `Back to RFE`
                                  }
                                } as any}>
                                  <Button variant="link" size="sm" className="px-0 h-auto">{sessionDisplay}</Button>
                                </Link>
                                {sessionForPhase?.status?.phase && <Badge variant="outline">{sessionForPhase.status.phase}</Badge>}
                              </div>
                            )}
                          </div>
                          <div className="flex items-center gap-3">
                            {exists ? (
                              <div className="flex items-center gap-2 text-green-700">
                                <CheckCircle2 className="h-5 w-5 text-green-600" />
                                <span className="text-sm font-medium">Ready</span>
                              </div>
                            ) : (
                              <Badge variant="secondary">{prerequisitesMet ? "Missing" : "Blocked"}</Badge>
                            )}
                            {!exists && (
                              phase === "ideate"
                                ? (
                                  (sessionForPhase && (sessionForPhase.status?.phase === "Running" || sessionForPhase.status?.phase === "Creating"))
                                    ? (
                                      <Link href={{
                                        pathname: `/projects/${encodeURIComponent(project)}/sessions/${encodeURIComponent(sessionForPhase.metadata.name)}`,
                                        query: {
                                          backHref: `/projects/${encodeURIComponent(project)}/rfe/${encodeURIComponent(id)}?tab=overview`,
                                          backLabel: `Back to RFE`
                                        }
                                      } as any}>
                                        <Button size="sm" variant="default">
                                          Enter Chat
                                        </Button>
                                      </Link>
                                    )
                                    : (
                                      <Button size="sm" onClick={async () => {
                                        try {
                                          setStartingPhase(phase);
                                          const prompt = `IMPORTANT: The result of this interactive chat session MUST produce rfe.md at the workspace root. The rfe.md should be formatted as markdown in the following way:\n\n# Feature Title\n\n**Feature Overview:**  \n*An elevator pitch (value statement) that describes the Feature in a clear, concise way. ie: Executive Summary of the user goal or problem that is being solved, why does this matter to the user? The \"What & Why\"...* \n\n* Text\n\n**Goals:**\n\n*Provide high-level goal statement, providing user context and expected user outcome(s) for this Feature. Who benefits from this Feature, and how? What is the difference between today's current state and a world with this Feature?*\n\n* Text\n\n**Out of Scope:**\n\n*High-level list of items or personas that are out of scope.*\n\n* Text\n\n**Requirements:**\n\n*A list of specific needs, capabilities, or objectives that a Feature must deliver to satisfy the Feature. Some requirements will be flagged as MVP. If an MVP gets shifted, the Feature shifts. If a non MVP requirement slips, it does not shift the feature.*\n\n* Text\n\n**Done - Acceptance Criteria:**\n\n*Acceptance Criteria articulates and defines the value proposition - what is required to meet the goal and intent of this Feature. The Acceptance Criteria provides a detailed definition of scope and the expected outcomes - from a users point of view*\n\n* Text\n\n**Use Cases - i.e. User Experience & Workflow:**\n\n*Include use case diagrams, main success scenarios, alternative flow scenarios.*\n\n* Text\n\n**Documentation Considerations:**\n\n*Provide information that needs to be considered and planned so that documentation will meet customer needs. If the feature extends existing functionality, provide a link to its current documentation..*\n\n* Text\n\n**Questions to answer:**\n\n*Include a list of refinement / architectural questions that may need to be answered before coding can begin.*\n\n* Text\n\n**Background & Strategic Fit:**\n\n*Provide any additional context is needed to frame the feature.*\n\n* Text\n\n**Customer Considerations**\n\n*Provide any additional customer-specific considerations that must be made when designing and delivering the Feature.*\n\n* Text`;
                                          const payload: CreateAgenticSessionRequest = {
                                            prompt,
                                            displayName: `${workflow.title} - ${phase}`,
                                            interactive: true,
                                            workspacePath: workflowWorkspace,
                                            environmentVariables: {
                                              WORKFLOW_PHASE: phase,
                                              PARENT_RFE: workflow.id,
                                            },
                                            labels: {
                                              project,
                                              "rfe-workflow": workflow.id,
                                              "rfe-phase": phase,
                                            },
                                            annotations: {
                                              "rfe-expected": expected,
                                            },
                                          };
                                          const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/agentic-sessions`, {
                                            method: "POST",
                                            headers: { "Content-Type": "application/json" },
                                            body: JSON.stringify(payload),
                                          });
                                          if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
                                          await Promise.all([load(), loadSessions()]);
                                          await probeWorkspaceAndPhase();
                                        } catch (e) {
                                          setError(e instanceof Error ? e.message : "Failed to start session");
                                        } finally {
                                          setStartingPhase(null);
                                        }
                                      }} disabled={startingPhase === phase || !prerequisitesMet}>
                                        {startingPhase === phase ? (<><Loader2 className="mr-2 h-4 w-4 animate-spin" />Starting…</>) : (<><Play className="mr-2 h-4 w-4" />Start Chat</>)}
                                      </Button>
                                    )
                                )
                                : (
                                    <Button size="sm" onClick={async () => {
                                      try {
                                        setStartingPhase(phase);
                                        const isSpecify = phase === "specify";
                                        const prompt = isSpecify
                                          ? (rfeDoc.exists
                                              ? "/specify Develop a new feature on top of the projects in /repos based on rfe.md"
                                              : `/specify Develop a new feature on top of the projects in /repos. Feature requirements: ${workflow.description}`)
                                          : `/${phase} ${workflow.description}`;
                                        const payload: CreateAgenticSessionRequest = {
                                          prompt,
                                          displayName: `${workflow.title} - ${phase}`,
                                          interactive: false,
                                          workspacePath: workflowWorkspace,
                                          environmentVariables: {
                                            WORKFLOW_PHASE: phase,
                                            PARENT_RFE: workflow.id,
                                          },
                                          labels: {
                                            project,
                                            "rfe-workflow": workflow.id,
                                            "rfe-phase": phase,
                                          },
                                          annotations: {
                                            "rfe-expected": expected,
                                          },
                                        };
                                        const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/agentic-sessions`, {
                                          method: "POST",
                                          headers: { "Content-Type": "application/json" },
                                          body: JSON.stringify(payload),
                                        });
                                        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
                                        await Promise.all([load(), loadSessions()]);
                                        await probeWorkspaceAndPhase();
                                      } catch (e) {
                                        setError(e instanceof Error ? e.message : "Failed to start session");
                                      } finally {
                                        setStartingPhase(null);
                                      }
                                    }} disabled={startingPhase === phase || !prerequisitesMet}>
                                      {startingPhase === phase ? (<><Loader2 className="mr-2 h-4 w-4 animate-spin" />Starting…</>) : (<><Play className="mr-2 h-4 w-4" />Generate</>)}
                                    </Button>
                                )
                            )}
                            {exists && phase !== "ideate" && (
                              <Button size="sm" variant="secondary" onClick={async () => {
                                try {
                                  setPublishingPhase(phase);
                                  const resp = await fetch(`/api/projects/${encodeURIComponent(project)}/rfe/${encodeURIComponent(id)}/jira`, {
                                    method: "POST",
                                    headers: { "Content-Type": "application/json" },
                                    body: JSON.stringify({ path: expected }),
                                  });
                                  const data = await resp.json().catch(() => ({}));
                                  if (!resp.ok) throw new Error(data?.error || `HTTP ${resp.status}`);
                                  await load();
                                } catch (e) {
                                  setError(e instanceof Error ? e.message : "Failed to publish to Jira");
                                } finally {
                                  setPublishingPhase(null);
                                }
                              }} disabled={publishingPhase === phase}>
                                {publishingPhase === phase ? (
                                  <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Publishing…</>
                                ) : (
                                  <><Upload className="mr-2 h-4 w-4" />{linkedKey ? 'Resync with Jira' : 'Publish to Jira'}</>
                                )}
                              </Button>
                            )}
                            {exists && linkedKey && phase !== "ideate" && (
                              <div className="flex items-center gap-2">
                                <Badge variant="outline">{linkedKey}</Badge>
                                <Button variant="link" size="sm" className="px-0 h-auto" onClick={() => openJiraForPath(expected)}>Open in Jira</Button>
                              </div>
                            )}
                          </div>
                        </div>
                      );
                    });
                  })()}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="sessions">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>Agentic Sessions ({rfeSessions.length})</CardTitle>
                    <CardDescription>Sessions scoped to this RFE</CardDescription>
                  </div>
                  <Link href={`/projects/${encodeURIComponent(project)}/sessions/new?workspacePath=${encodeURIComponent(workflowWorkspace)}&rfeWorkflow=${encodeURIComponent(workflow.id)}`}>
                    <Button variant="default" size="sm">
                      <Plus className="w-4 h-4 mr-2" />
                      Create Session
                    </Button>
                  </Link>
                </div>
              </CardHeader>
              <CardContent>
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="min-w-[220px]">Name</TableHead>
                        <TableHead>Stage</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead className="hidden md:table-cell">Model</TableHead>
                        <TableHead className="hidden lg:table-cell">Created</TableHead>
                        <TableHead className="hidden xl:table-cell">Cost</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {rfeSessions.length === 0 ? (
                        <TableRow><TableCell colSpan={6} className="py-6 text-center text-muted-foreground">No agent sessions yet</TableCell></TableRow>
                      ) : (
                        rfeSessions.map((s) => {
                          const labels = (s.metadata.labels || {}) as Record<string, unknown>;
                          const name = s.metadata.name;
                          const display = s.spec?.displayName || name;
                          const rfePhase = typeof labels["rfe-phase"] === "string" ? String(labels["rfe-phase"]) : '';
                          const model = s.spec?.llmSettings?.model;
                          const created = s.metadata?.creationTimestamp ? formatDistanceToNow(new Date(s.metadata.creationTimestamp), { addSuffix: true }) : '';
                          const cost = s.status?.total_cost_usd;
                          return (
                            <TableRow key={name}>
                              <TableCell className="font-medium min-w-[180px]">
                                <Link href={{
                                  pathname: `/projects/${encodeURIComponent(project)}/sessions/${encodeURIComponent(name)}`,
                                  query: {
                                    backHref: `/projects/${encodeURIComponent(project)}/rfe/${encodeURIComponent(id)}?tab=sessions`,
                                    backLabel: `Back to RFE`
                                  }
                                } as any} className="text-blue-600 hover:underline hover:text-blue-800 transition-colors block">
                                  <div className="font-medium">{display}</div>
                                  {display !== name && (<div className="text-xs text-gray-500">{name}</div>)}
                                </Link>
                              </TableCell>
                              <TableCell>{WORKFLOW_PHASE_LABELS[rfePhase as WorkflowPhase] || rfePhase || '—'}</TableCell>
                              <TableCell><span className="text-sm">{s.status?.phase || 'Pending'}</span></TableCell>
                              <TableCell className="hidden md:table-cell"><span className="text-sm text-gray-600 truncate max-w-[160px] block">{model || '—'}</span></TableCell>
                              <TableCell className="hidden lg:table-cell">{created || <span className="text-gray-400">—</span>}</TableCell>
                              <TableCell className="hidden xl:table-cell">{cost ? <span className="text-sm font-mono">${cost.toFixed?.(4) ?? cost}</span> : <span className="text-gray-400">—</span>}</TableCell>
                            </TableRow>
                          );
                        })
                      )}
                    </TableBody>
                  </Table>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="workspace">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2"><FolderTree className="h-5 w-5" />Workspace</CardTitle>
                <CardDescription>Browse generated files and cloned repos</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="md:col-span-1 border rounded-lg overflow-hidden">
                    <div className="p-2">
                      <FileTree nodes={wsTree} selectedPath={wsSelectedPath} onSelect={onWsSelect} onToggle={onWsToggle} />
                    </div>
                  </div>
                  <div className="md:col-span-2 border rounded-lg p-3 min-h-[300px]">
                    {wsSelectedPath ? (
                      wsLoading ? (
                        <div className="text-sm text-muted-foreground">Loading file…</div>
                      ) : (
                        <div className="space-y-2">
                          <div className="flex items-center justify-between">
                            <div className="text-xs text-muted-foreground">{wsSelectedPath}</div>
                            <Button size="sm" onClick={async () => { await saveWsFile(wsSelectedPath!, wsFileContent); }}>Save</Button>
                          </div>
                          <textarea
                            className="w-full h-[60vh] text-sm font-mono border rounded p-3"
                            value={wsFileContent}
                            onChange={(e) => setWsFileContent(e.target.value)}
                          />
                        </div>
                      )
                    ) : (
                      <div className="text-sm text-muted-foreground">Select a file to view its contents</div>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>

      </div>
    </div>
  );
}
