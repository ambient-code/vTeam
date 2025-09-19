"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useSearchParams } from "next/navigation";
import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import {
  RFEWorkflow,
  WorkflowPhase,
  AgenticSessionPhase,
} from "@/types/agentic-session";
import {
  ArrowLeft,
  GitBranch,
  Users,
  FileText,
  Play,
  Edit,
  Upload,
  Clock,
  CheckCircle,
  XCircle,
  Loader2,
  ExternalLink,
} from "lucide-react";
import { getApiUrl } from "@/lib/config";
import { WORKFLOW_PHASE_LABELS, WORKFLOW_PHASE_DESCRIPTIONS, getAgentByPersona } from "@/lib/agents";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

const getSessionStatusIcon = (status: string) => {
  switch (status.toLowerCase()) {
    case "completed":
      return <CheckCircle className="h-4 w-4 text-green-600" />;
    case "failed":
    case "error":
      return <XCircle className="h-4 w-4 text-red-600" />;
    case "running":
      return <Loader2 className="h-4 w-4 text-blue-600 animate-spin" />;
    default:
      return <Clock className="h-4 w-4 text-gray-400" />;
  }
};

const getSessionStatusColor = (status: string) => {
  switch (status.toLowerCase()) {
    case "completed":
      return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300";
    case "failed":
    case "error":
      return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300";
    case "running":
      return "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300";
    case "pending":
    case "creating":
      return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300";
    default:
      return "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300";
  }
};

function calculatePhaseProgress(workflow: RFEWorkflow, phase: WorkflowPhase): number {
  const safeSessions = workflow.agentSessions || [];
  const phaseSessions = safeSessions.filter(s => s.phase === phase);
  if (phaseSessions.length === 0) return 0;

  const completedSessions = phaseSessions.filter(s => s.status.toLowerCase() === "completed").length;
  return (completedSessions / phaseSessions.length) * 100;
}

export default function RFEWorkflowDetailPage() {
  const params = useParams();
  const workflowId = params.id as string;
  const searchParams = useSearchParams();
  const project = searchParams.get("project");

  const [workflow, setWorkflow] = useState<RFEWorkflow | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isStartingPhase, setIsStartingPhase] = useState(false);
  const [isUpdatingDynamicData, setIsUpdatingDynamicData] = useState(false);

  // Initial fetch for complete workflow data
  const fetchWorkflow = useCallback(async () => {
    try {
      setIsLoading(true);
      const url = project
        ? `${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${workflowId}`
        : `${getApiUrl()}/rfe-workflows/${workflowId}`;
      const response = await fetch(url);

      if (!response.ok) {
        if (response.status === 404) {
          throw new Error("RFE workflow not found");
        }
        throw new Error(`Failed to fetch RFE workflow: ${response.status}`);
      }

      const data = await response.json();
      setWorkflow(data);
      setError(null);
    } catch (err) {
      console.error("Error fetching RFE workflow:", err);
      setError(
        err instanceof Error ? err.message : "Failed to load RFE workflow"
      );
    } finally {
      setIsLoading(false);
    }
  }, [workflowId]);

  // Fetch only dynamic data that changes frequently
  const fetchDynamicUpdates = useCallback(async () => {
    if (!workflow) return;

    try {
      setIsUpdatingDynamicData(true);
      const url = project
        ? `${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${workflowId}`
        : `${getApiUrl()}/rfe-workflows/${workflowId}`;
      const response = await fetch(url);
      if (!response.ok) return;

      const data = await response.json();

      // Only update if there are actual changes to prevent unnecessary re-renders
      const hasSessionChanges = JSON.stringify(workflow.agentSessions) !== JSON.stringify(data.agentSessions);
      const hasArtifactChanges = JSON.stringify(workflow.artifacts) !== JSON.stringify(data.artifacts);
      const hasPhaseChange = workflow.currentPhase !== data.currentPhase;

      if (hasSessionChanges || hasArtifactChanges || hasPhaseChange) {
        setWorkflow(prevWorkflow => ({
          ...prevWorkflow!,
          agentSessions: data.agentSessions,
          artifacts: data.artifacts,
          currentPhase: data.currentPhase,
        }));
      }
    } catch (err) {
      console.error("Error fetching dynamic updates:", err);
    } finally {
      setIsUpdatingDynamicData(false);
    }
  }, [workflowId, workflow]);

  useEffect(() => {
    fetchWorkflow();
  }, [workflowId, fetchWorkflow]);

  // Set up polling for dynamic updates only after initial load
  useEffect(() => {
    if (!workflow) return;

    const interval = setInterval(fetchDynamicUpdates, 3000); // More frequent updates for dynamic data
    return () => clearInterval(interval);
  }, [workflow, fetchDynamicUpdates]);

  const handleStartNextPhase = async () => {
    if (!workflow) return;

    const phases: WorkflowPhase[] = ["specify", "plan", "tasks", "review"];
    const currentIndex = phases.indexOf(workflow.currentPhase);
    const nextPhase = phases[currentIndex + 1];

    if (!nextPhase) return;

    setIsStartingPhase(true);
    try {
      const url = project
        ? `${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${workflowId}/advance-phase`
        : `${getApiUrl()}/rfe-workflows/${workflowId}/advance-phase`;
      const response = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ nextPhase }),
      });

      if (response.ok) {
        fetchWorkflow();
      } else {
        throw new Error("Failed to start next phase");
      }
    } catch (err) {
      console.error("Error starting next phase:", err);
    } finally {
      setIsStartingPhase(false);
    }
  };

  const handlePushToGit = async () => {
    if (!workflow) return;

    try {
      const url = project
        ? `${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${workflowId}/push-to-git`
        : `${getApiUrl()}/rfe-workflows/${workflowId}/push-to-git`;
      const response = await fetch(url, {
        method: "POST",
      });

      if (response.ok) {
        fetchWorkflow();
      } else {
        throw new Error("Failed to push to git");
      }
    } catch (err) {
      console.error("Error pushing to git:", err);
    }
  };

  if (isLoading) {
    return (
      <div className="container mx-auto py-8">
        <div className="max-w-6xl mx-auto">
          <div className="animate-pulse space-y-6">
            <div className="h-8 bg-gray-200 rounded w-1/3"></div>
            <div className="h-4 bg-gray-200 rounded w-2/3"></div>
            <div className="grid gap-4 md:grid-cols-3">
              {[...Array(3)].map((_, i) => (
                <div key={i} className="h-32 bg-gray-200 rounded"></div>
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (error || !workflow) {
    return (
      <div className="container mx-auto py-8">
        <Card className="border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <p className="text-red-600">Error: {error}</p>
            <div className="mt-4 space-x-2">
              <Button onClick={fetchWorkflow} variant="outline">
                Retry
              </Button>
              <Link href="/rfe">
                <Button variant="outline">
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back to RFE Workflows
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const phases: WorkflowPhase[] = ["specify", "plan", "tasks", "review"];
  const currentPhaseIndex = phases.indexOf(workflow.currentPhase);
  const isPhaseComplete = calculatePhaseProgress(workflow, workflow.currentPhase) === 100;
  const canAdvancePhase = isPhaseComplete && currentPhaseIndex < phases.length - 1;

  return (
    <div className="container mx-auto py-8">
      <div className="max-w-6xl mx-auto space-y-8">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
              <Link href={project ? `/rfe?project=${encodeURIComponent(project)}` : "/rfe"}>
              <Button variant="ghost" size="sm">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to RFE Workflows
              </Button>
            </Link>
            <div>
              <h1 className="text-3xl font-bold">{workflow.title}</h1>
              <p className="text-muted-foreground mt-1">{workflow.description}</p>
            </div>
          </div>

          <div className="flex gap-2">
            <Link href={project ? `/rfe/${workflowId}/edit?project=${encodeURIComponent(project)}` : `/rfe/${workflowId}/edit`}>
              <Button variant="outline" size="sm">
                <Edit className="mr-2 h-4 w-4" />
                Edit Artifacts
              </Button>
            </Link>
            {(workflow.artifacts || []).length > 0 && (
              <Button onClick={handlePushToGit} variant="outline" size="sm">
                <Upload className="mr-2 h-4 w-4" />
                Push to Git
              </Button>
            )}
          </div>
        </div>

        {/* Workflow Overview */}
        <div className="grid gap-6 md:grid-cols-3">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Current Phase</CardTitle>
              <Badge className="bg-blue-100 text-blue-800">
                {WORKFLOW_PHASE_LABELS[workflow.currentPhase]}
              </Badge>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {calculatePhaseProgress(workflow, workflow.currentPhase).toFixed(0)}%
              </div>
              <p className="text-xs text-muted-foreground">
                {WORKFLOW_PHASE_DESCRIPTIONS[workflow.currentPhase]}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Agent Progress</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {(workflow.agentSessions || []).filter(s => s.status.toLowerCase() === "completed").length}/{(workflow.agentSessions || []).length}
              </div>
              <p className="text-xs text-muted-foreground">
                Sessions completed across all phases
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Artifacts</CardTitle>
              <FileText className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{(workflow.artifacts || []).length}</div>
              <p className="text-xs text-muted-foreground">
                Generated specifications and plans
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Repository Info */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GitBranch className="h-5 w-5" />
              Target Repository
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium">{workflow.targetRepoUrl}</p>
                <p className="text-sm text-muted-foreground">
                  Branch: {workflow.targetRepoBranch || "main"}
                </p>
              </div>
              <a
                href={workflow.targetRepoUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-blue-600 hover:text-blue-800"
              >
                <ExternalLink className="h-4 w-4" />
              </a>
            </div>
          </CardContent>
        </Card>

        {/* Phase Progress */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Workflow Progress</CardTitle>
              {canAdvancePhase && (
                <Button onClick={handleStartNextPhase} disabled={isStartingPhase}>
                  {isStartingPhase ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Play className="mr-2 h-4 w-4" />
                  )}
                  Start {WORKFLOW_PHASE_LABELS[phases[currentPhaseIndex + 1]]} Phase
                </Button>
              )}
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {phases.map((phase) => {
                const progress = calculatePhaseProgress(workflow, phase);
                const isCurrent = phase === workflow.currentPhase;
                const isCompleted = false; // TODO: Implement phase completion tracking

                return (
                  <div key={phase} className={`p-4 rounded-lg border ${
                    isCurrent ? 'border-blue-200 bg-blue-50' :
                    isCompleted ? 'border-green-200 bg-green-50' :
                    'border-gray-200'
                  }`}>
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <Badge variant={isCurrent ? "default" : isCompleted ? "secondary" : "outline"}>
                          {WORKFLOW_PHASE_LABELS[phase]}
                        </Badge>
                        {isCurrent && <Badge variant="outline">Current</Badge>}
                        {isCompleted && <CheckCircle className="h-4 w-4 text-green-600" />}
                      </div>
                      <span className="text-sm font-medium">{Math.round(progress)}%</span>
                    </div>
                    <Progress value={progress} className="mb-2" />
                    <p className="text-xs text-muted-foreground">
                      {WORKFLOW_PHASE_DESCRIPTIONS[phase]}
                    </p>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>

        {/* Agent Sessions by Phase */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Agent Sessions</CardTitle>
                <CardDescription>
                  Track progress of individual agent executions across workflow phases
                </CardDescription>
              </div>
              {isUpdatingDynamicData && (
                <Loader2 className="h-4 w-4 text-muted-foreground animate-spin" />
              )}
            </div>
          </CardHeader>
          <CardContent>
            <Tabs defaultValue={workflow.currentPhase} className="w-full">
              <TabsList className="grid w-full grid-cols-4">
                {phases.map(phase => (
                  <TabsTrigger key={phase} value={phase}>
                    {WORKFLOW_PHASE_LABELS[phase]}
                  </TabsTrigger>
                ))}
              </TabsList>

              {phases.map(phase => (
                <TabsContent key={phase} value={phase} className="space-y-4">
                  <div className="grid gap-4">
                    {(workflow.agentSessions || [])
                      .filter(session => session.phase === phase)
                      .map(session => {
                        const agent = getAgentByPersona(session.agentPersona);
                        return (
                          <div key={`${session.agentPersona}-${phase}`} className="flex items-center justify-between p-4 border rounded-lg">
                            <div className="flex items-center gap-3">
                              {getSessionStatusIcon(session.status)}
                              <div>
                                <p className="font-medium">{agent?.name || session.agentPersona}</p>
                                <p className="text-sm text-muted-foreground">{agent?.role || 'Agent'}</p>
                              </div>
                            </div>

                            <div className="flex items-center gap-3">
                              <Badge className={getSessionStatusColor(session.status)}>
                                {session.status}
                              </Badge>

                              {session.completedAt && !isNaN(new Date(session.completedAt).getTime()) && (
                                <span className="text-xs text-muted-foreground">
                                  {formatDistanceToNow(new Date(session.completedAt), { addSuffix: true })}
                                </span>
                              )}

                              {session.cost && session.cost > 0 && (
                                <span className="text-xs text-muted-foreground">
                                  ${session.cost.toFixed(4)}
                                </span>
                              )}
                            </div>
                          </div>
                        );
                      })}

                    {(workflow.agentSessions || []).filter(s => s.phase === phase).length === 0 && (
                      <div className="text-center py-8 text-muted-foreground">
                        No agent sessions for this phase yet
                      </div>
                    )}
                  </div>
                </TabsContent>
              ))}
            </Tabs>
          </CardContent>
        </Card>

        {/* Generated Artifacts */}
        {(workflow.artifacts || []).length > 0 && (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <FileText className="h-5 w-5" />
                  Generated Artifacts
                </CardTitle>
                {isUpdatingDynamicData && (
                  <Loader2 className="h-4 w-4 text-muted-foreground animate-spin" />
                )}
              </div>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                {workflow.artifacts.map(artifact => (
                  <div key={artifact.path} className="p-3 border rounded-lg">
                    <div className="flex items-center justify-between">
                      <p className="font-medium text-sm">{artifact.name}</p>
                      <Badge variant="outline" className="text-xs">
                        {artifact.agent || artifact.phase}
                      </Badge>
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {(artifact.size / 1024).toFixed(1)} KB • {artifact.lastModified && !isNaN(new Date(artifact.lastModified).getTime())
                        ? formatDistanceToNow(new Date(artifact.lastModified), { addSuffix: true })
                        : 'recently'
                      }
                    </p>
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