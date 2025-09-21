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
import { RFEWorkflow, WorkflowPhase } from "@/types/agentic-session";
import { WORKFLOW_PHASE_LABELS } from "@/lib/agents";
import { ArrowLeft, Edit, Play, Loader2, RefreshCw, FolderTree } from "lucide-react";
import { FileTree, type FileTreeNode } from "@/components/file-tree";

function phaseProgress(w: RFEWorkflow, phase: WorkflowPhase) {
  const inPhase = (w.agentSessions || []).filter(s => s.phase === phase);
  if (!inPhase.length) return 0;
  const done = inPhase.filter(s => s.status.toLowerCase() === "completed").length;
  return (done / inPhase.length) * 100;
}

export default function ProjectRFEDetailPage() {
  const params = useParams();
  const project = params?.name as string;
  const id = params?.id as string;

  const [workflow, setWorkflow] = useState<RFEWorkflow | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [advancing, setAdvancing] = useState(false);
  const [startingPhase, setStartingPhase] = useState<WorkflowPhase | null>(null);
  const [rfeSessions, setRfeSessions] = useState<Array<{ name: string; phase?: string; labels?: Record<string, unknown> }>>([]);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [hasWorkspace, setHasWorkspace] = useState<boolean | null>(null);
  const [wsTree, setWsTree] = useState<FileTreeNode[]>([]);
  const [wsSelectedPath, setWsSelectedPath] = useState<string | undefined>(undefined);
  const [wsFileContent, setWsFileContent] = useState<string>("");
  const [wsLoading, setWsLoading] = useState<boolean>(false);
  const [activeTab, setActiveTab] = useState<string>("overview");
  const [specExists, setSpecExists] = useState<boolean>(false);
  const [planExists, setPlanExists] = useState<boolean>(false);
  const [tasksExists, setTasksExists] = useState<boolean>(false);

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

  const probeWorkspaceAndPhase = useCallback(async () => {
    const items = await listWsPath("specs");
    const names = new Set(items.map((i: any) => i.name));
    setSpecExists(names.has("spec.md"));
    setPlanExists(names.has("plan.md"));
    setTasksExists(names.has("tasks.md"));
  }, [listWsPath]);

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
    setWsTree(prev => prev.map(n => n.path === node.path ? { ...n, children } : n));
  }, [listWsPath]);

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

  const advancePhase = useCallback(async () => {
    try {
      setAdvancing(true);
      const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}/advance-phase`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to advance phase");
    } finally {
      setAdvancing(false);
    }
  }, [project, id, load]);

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

  const currentPhase: WorkflowPhase = (!specExists && !planExists && !tasksExists) ? "pre" : (!specExists ? "specify" : (!planExists ? "plan" : (!tasksExists ? "tasks" : "completed")));
  const workflowWorkspace = workflow.workspacePath || `/rfe-workflows/${id}/workspace`;
  const phases: WorkflowPhase[] = ["pre", "specify", "plan", "tasks"];
  const idx = phases.indexOf(currentPhase);
  const canAdvance = (currentPhase === "pre") || (phaseProgress(workflow, currentPhase) === 100 && idx < phases.length - 1);

  return (
    <div className="container mx-auto py-8">
      <div className="max-w-6xl mx-auto space-y-8">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
            <Link href={`/projects/${encodeURIComponent(project)}/rfe`}>
              <Button variant="ghost" size="sm"><ArrowLeft className="h-4 w-4 mr-2" />Back to RFE Workflows</Button>
            </Link>
            <div>
              <h1 className="text-3xl font-bold">{workflow.title}</h1>
              <p className="text-muted-foreground mt-1">{workflow.description}</p>
            </div>
          </div>
          <div className="flex gap-2">
            <Link href={`/projects/${encodeURIComponent(project)}/rfe/${encodeURIComponent(id)}/edit`}>
              <Button variant="outline" size="sm"><Edit className="mr-2 h-4 w-4" />Edit</Button>
            </Link>
          </div>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Current Phase</CardTitle>
              <Badge className="bg-blue-100 text-blue-800">{WORKFLOW_PHASE_LABELS[currentPhase]}</Badge>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{phaseProgress(workflow, currentPhase).toFixed(0)}%</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Agent Progress</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{(workflow.agentSessions || []).filter(s => s.status.toLowerCase() === "completed").length}/{(workflow.agentSessions || []).length}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Phase Files</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{(specExists ? 1 : 0) + (planExists ? 1 : 0) + (tasksExists ? 1 : 0)}</div>
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
            <TabsTrigger value="workspace">Workspace</TabsTrigger>
          </TabsList>

          <TabsContent value="overview">
            <Card>
              <CardHeader>
                <CardTitle>Phase Documents</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {(() => {
                    const phaseList = ["specify","plan","tasks"] as WorkflowPhase[];
                    return phaseList.map(phase => {
                      const expected = (() => {
                        // Derive expected path under specs/. If subfolder exists, prefer it.
                        // We only know existence booleans here, so use conventional relative names.
                        // The backend summary already handles subfolder detection for existence.
                        if (phase === "specify") return "specs/spec.md";
                        if (phase === "plan") return "specs/plan.md";
                        return "specs/tasks.md";
                      })();
                      const exists = phase === "specify" ? specExists : phase === "plan" ? planExists : tasksExists;
                      const sessionForPhase = rfeSessions.find(s => (s.labels as any)?.["rfe-phase"] === phase);
                      const running = (sessionForPhase?.phase || "").toLowerCase() === "running";
                      const completed = (sessionForPhase?.phase || "").toLowerCase() === "completed";
                      const prerequisitesMet = phase === "specify" ? true : phase === "plan" ? specExists : (specExists && planExists);
                      return (
                        <div key={phase} className="p-4 rounded-lg border flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <Badge variant="outline">{WORKFLOW_PHASE_LABELS[phase]}</Badge>
                            <span className="text-sm text-muted-foreground">{expected}</span>
                          </div>
                          <div className="flex items-center gap-3">
                            <Badge variant={exists ? "outline" : "secondary"}>{exists ? "Exists" : (prerequisitesMet ? "Missing" : "Blocked")}</Badge>
                            {running && <Badge variant="outline">Running</Badge>}
                            {completed && <Badge variant="outline">Completed</Badge>}
                            {!exists && !running && (
                              <Button size="sm" onClick={async () => {
                                try {
                                  setStartingPhase(phase);
                              const payload = {
                                    prompt: `/${phase} ${workflow.description}`,
                                    displayName: `${workflow.title} - ${phase}`,
                                interactive: false,
                                sharedWorkspace: workflowWorkspace,
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
                  <Button variant="outline" size="sm" onClick={loadSessions} disabled={sessionsLoading}>
                    <RefreshCw className={`w-4 h-4 mr-2 ${sessionsLoading ? "animate-spin" : ""}`} />
                    Refresh
                  </Button>
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
                        rfeSessions.map((s: any) => {
                          const labels = (s.labels || {}) as Record<string, unknown>;
                          const name = s.name;
                          const display = s.spec?.displayName || name;
                          const rfePhase = typeof labels["rfe-phase"] === "string" ? String(labels["rfe-phase"]) : '';
                          const model = s.spec?.llmSettings?.model;
                          const created = s.metadata?.creationTimestamp ? formatDistanceToNow(new Date(s.metadata.creationTimestamp), { addSuffix: true }) : '';
                          const cost = s.status?.cost;
                          return (
                            <TableRow key={name}>
                              <TableCell className="font-medium min-w-[180px]">
                                <Link href={`/projects/${encodeURIComponent(project)}/sessions/${encodeURIComponent(name)}`} className="text-blue-600 hover:underline hover:text-blue-800 transition-colors block">
                                  <div className="font-medium">{display}</div>
                                  {display !== name && (<div className="text-xs text-gray-500">{name}</div>)}
                                </Link>
                              </TableCell>
                              <TableCell>{WORKFLOW_PHASE_LABELS[rfePhase as WorkflowPhase] || rfePhase || '—'}</TableCell>
                              <TableCell><span className="text-sm">{s.phase || 'Pending'}</span></TableCell>
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
                        <pre className="text-sm whitespace-pre-wrap break-words">{wsFileContent}</pre>
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

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Agentic Sessions ({rfeSessions.length})</CardTitle>
                <CardDescription>Sessions scoped to this RFE</CardDescription>
              </div>
              <Button variant="outline" size="sm" onClick={loadSessions} disabled={sessionsLoading}>
                <RefreshCw className={`w-4 h-4 mr-2 ${sessionsLoading ? "animate-spin" : ""}`} />
                Refresh
              </Button>
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
                    rfeSessions.map((s: any) => {
                      const labels = (s.labels || {}) as Record<string, unknown>;
                      const name = s.name;
                      const display = s.spec?.displayName || name;
                      const rfePhase = typeof labels["rfe-phase"] === "string" ? String(labels["rfe-phase"]) : '';
                      const model = s.spec?.llmSettings?.model;
                      const created = s.metadata?.creationTimestamp ? formatDistanceToNow(new Date(s.metadata.creationTimestamp), { addSuffix: true }) : '';
                      const cost = s.status?.cost;
                      return (
                        <TableRow key={name}>
                          <TableCell className="font-medium min-w-[180px]">
                            <Link href={`/projects/${encodeURIComponent(project)}/sessions/${encodeURIComponent(name)}`} className="text-blue-600 hover:underline hover:text-blue-800 transition-colors block">
                              <div className="font-medium">{display}</div>
                              {display !== name && (<div className="text-xs text-gray-500">{name}</div>)}
                            </Link>
                          </TableCell>
                          <TableCell>{WORKFLOW_PHASE_LABELS[rfePhase as WorkflowPhase] || rfePhase || '—'}</TableCell>
                          <TableCell><span className="text-sm">{s.phase || 'Pending'}</span></TableCell>
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

        {/* Artifacts grid omitted; use Workspace tab */}
      </div>
    </div>
  );
}
