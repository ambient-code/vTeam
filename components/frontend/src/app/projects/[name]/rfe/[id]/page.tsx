"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { getApiUrl } from "@/lib/config";
import { RFEWorkflow, WorkflowPhase } from "@/types/agentic-session";
import { WORKFLOW_PHASE_LABELS } from "@/lib/agents";
import { ArrowLeft, Edit, Upload, Play, Loader2, GitBranch, FileText } from "lucide-react";

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

  useEffect(() => { if (project && id) load(); }, [project, id, load]);

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

  const phases: WorkflowPhase[] = ["specify", "plan", "tasks", "review"];
  const idx = phases.indexOf(workflow.currentPhase);
  const canAdvance = phaseProgress(workflow, workflow.currentPhase) === 100 && idx < phases.length - 1;

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
            {(workflow.artifacts || []).length > 0 && (
              <Button variant="outline" size="sm"><Upload className="mr-2 h-4 w-4" />Push to Git</Button>
            )}
          </div>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Current Phase</CardTitle>
              <Badge className="bg-blue-100 text-blue-800">{WORKFLOW_PHASE_LABELS[workflow.currentPhase]}</Badge>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{phaseProgress(workflow, workflow.currentPhase).toFixed(0)}%</div>
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
              <div className="text-2xl font-bold">{(workflow.artifacts || []).length}</div>
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
            <div className="flex items-center justify-between">
              <CardTitle>Workflow Progress</CardTitle>
              {canAdvance && (
                <Button><Play className="mr-2 h-4 w-4" />Start {WORKFLOW_PHASE_LABELS[phases[idx + 1]]} Phase</Button>
              )}
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {phases.map(phase => (
                <div key={phase} className="p-4 rounded-lg border">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">{WORKFLOW_PHASE_LABELS[phase]}</Badge>
                    </div>
                    <span className="text-sm font-medium">{Math.round(phaseProgress(workflow, phase))}%</span>
                  </div>
                  <Progress value={phaseProgress(workflow, phase)} className="mb-2" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Agent Sessions</CardTitle>
                <CardDescription>Track agent executions across phases</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <Tabs defaultValue={workflow.currentPhase} className="w-full">
              <TabsList className="grid w-full grid-cols-4">
                {phases.map(p => (<TabsTrigger key={p} value={p}>{WORKFLOW_PHASE_LABELS[p]}</TabsTrigger>))}
              </TabsList>
              {phases.map(p => (
                <TabsContent key={p} value={p} className="space-y-4">
                  <div className="grid gap-4">
                    {(workflow.agentSessions || []).filter(s => s.phase === p).map(s => (
                      <div key={`${s.agentPersona}-${p}`} className="flex items-center justify-between p-4 border rounded-lg">
                        <div className="flex items-center gap-3">
                          <div className="h-2 w-2 rounded-full bg-gray-400" />
                          <div>
                            <p className="font-medium">{s.agentPersona}</p>
                          </div>
                        </div>
                        <Badge variant="outline">{s.status}</Badge>
                      </div>
                    ))}
                    {(workflow.agentSessions || []).filter(s => s.phase === p).length === 0 && (
                      <div className="text-center py-8 text-muted-foreground">No agent sessions for this phase yet</div>
                    )}
                  </div>
                </TabsContent>
              ))}
            </Tabs>
          </CardContent>
        </Card>

        {(workflow.artifacts || []).length > 0 && (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2"><FileText className="h-5 w-5" />Generated Artifacts</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                {workflow.artifacts.map(a => (
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
