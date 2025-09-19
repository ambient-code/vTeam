"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { getApiUrl } from "@/lib/config";
import { RFEWorkflow, WorkflowPhase } from "@/types/agentic-session";
import { Plus, RefreshCw } from "lucide-react";

const phaseLabel: Record<WorkflowPhase, string> = {
  specify: "Specify",
  plan: "Plan",
  tasks: "Tasks",
  review: "Review",
  completed: "Completed",
};

function calcProgress(w: RFEWorkflow): number {
  const phases: WorkflowPhase[] = ["specify", "plan", "tasks", "review"];
  if (w.status === "completed") return 100;
  const idx = phases.indexOf(w.currentPhase);
  if (idx < 0) return 0;
  const inPhase = (w.agentSessions || []).filter(s => s.phase === w.currentPhase);
  const done = inPhase.filter(s => s.status === "Completed").length;
  const phasePct = inPhase.length ? (done / inPhase.length) * 25 : 0;
  return Math.min(idx * 25 + phasePct, 100);
}

export default function ProjectRFEListPage() {
  const params = useParams();
  const project = params?.name as string;
  const [items, setItems] = useState<RFEWorkflow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    try {
      setLoading(true);
      const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows`);
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      const data = await resp.json();
      setItems(Array.isArray(data.workflows) ? data.workflows : []);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (project) load();
  }, [project]);

  return (
    <div className="container mx-auto py-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold">RFE Workflows</h1>
          <p className="text-muted-foreground">Project: {project}</p>
        </div>
        <div className="flex gap-2">
          <Button onClick={load} variant="outline" disabled={loading}>
            <RefreshCw className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          <Link href={`/projects/${encodeURIComponent(project)}/rfe/new`}>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Create RFE Workflow
            </Button>
          </Link>
        </div>
      </div>

      {error && (
        <Card className="mb-6 border-red-200 bg-red-50">
          <CardContent className="pt-6 text-red-600">{error}</CardContent>
        </Card>
      )}

      {loading ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader>
                <div className="h-4 bg-gray-200 rounded w-3/4" />
                <div className="h-3 bg-gray-200 rounded w-1/2" />
              </CardHeader>
              <CardContent>
                <div className="h-2 bg-gray-200 rounded" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : items.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">No RFE workflows yet.</div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {items.map(w => (
            <Card key={w.id} className="hover:shadow transition-shadow">
              <CardHeader className="pb-3">
                <CardTitle className="text-lg">
                  <Link href={`/projects/${encodeURIComponent(project)}/rfe/${w.id}`} className="hover:underline">
                    {w.title}
                  </Link>
                </CardTitle>
                <CardDescription className="line-clamp-2">{w.description}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex gap-2">
                  <Badge variant="outline">{phaseLabel[w.currentPhase]}</Badge>
                  <Badge variant="outline">{w.status}</Badge>
                </div>
                <div>
                  <Progress value={calcProgress(w)} className="h-2" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
