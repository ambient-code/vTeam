"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
// Tabs removed for sessions list
import { getApiUrl } from "@/lib/config";
import { formatDistanceToNow } from "date-fns";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { RFEWorkflow, WorkflowPhase } from "@/types/agentic-session";
import { WORKFLOW_PHASE_LABELS } from "@/lib/agents";
import { ArrowLeft, Edit, Upload, Play, Loader2, GitBranch, FileText, RefreshCw } from "lucide-react";

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

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${encodeURIComponent(id)}`);
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      setWorkflow(await resp.json());
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

  const currentPhase: WorkflowPhase = workflow.currentPhase ?? "pre";
  const artifacts = workflow.artifacts ?? [];
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
              <Button variant="outline" size="sm"><Edit className="mr-2 h-4 w-4" />Edit Artifacts</Button>
            </Link>
            {artifacts.length > 0 && (
              <Button variant="outline" size="sm"><Upload className="mr-2 h-4 w-4" />Push to Git</Button>
            )}
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
              <CardTitle className="text-sm font-medium">Artifacts</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{artifacts.length}</div>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2"><GitBranch className="h-5 w-5" />Target Repository</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-sm text-muted-foreground">{workflow.targetRepoUrl}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Phase Documents</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {(() => {
                const expectedPaths: Record<string, string> = {
                  specify: "specs/spec.md",
                  plan: "specs/plan.md",
                  tasks: "specs/tasks.md",
                };
                const has = (p: string) => artifacts.some(a => a.path.endsWith(p) || a.name === p.split('/').pop());
                const specExists = has(expectedPaths.specify);
                const planExists = has(expectedPaths.plan);
                const tasksExists = has(expectedPaths.tasks);
                const phaseList = ["specify","plan","tasks"] as WorkflowPhase[];
                return phaseList.map(phase => {
                  const expected = expectedPaths[phase as keyof typeof expectedPaths];
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

        {artifacts.length > 0 && (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2"><FileText className="h-5 w-5" />Generated Artifacts</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                {artifacts.map(a => (
                  <div key={a.path} className="p-3 border rounded-lg">
                    <div className="flex items-center justify-between">
                      <p className="font-medium text-sm">{a.name}</p>
                      <Badge variant="outline" className="text-xs">{a.agent || a.phase}</Badge>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
