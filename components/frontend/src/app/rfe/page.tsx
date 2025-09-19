"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
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
import {
  RFEWorkflow,
  WorkflowPhase,
} from "@/types/agentic-session";
import {
  Plus,
  RefreshCw,
  GitBranch,
  Users,
  FileText,
  MoreHorizontal,
  ExternalLink,
  Trash2,
  Play,
  Pause,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { getApiUrl } from "@/lib/config";
import { WORKFLOW_PHASE_LABELS } from "@/lib/agents";
import { Progress } from "@/components/ui/progress";

const getPhaseColor = (phase: WorkflowPhase) => {
  switch (phase) {
    case "specify":
      return "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300";
    case "plan":
      return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300";
    case "tasks":
      return "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300";
    case "review":
      return "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300";
    case "completed":
      return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300";
    default:
      return "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300";
  }
};

const getStatusColor = (status: string) => {
  switch (status) {
    case "active":
      return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300";
    case "completed":
      return "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300";
    case "failed":
      return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300";
    case "paused":
      return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300";
    default:
      return "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300";
  }
};

function calculateProgress(workflow: RFEWorkflow): number {
  const phases = ["specify", "plan", "tasks", "review"];
  const currentIndex = phases.indexOf(workflow.currentPhase);

  if (workflow.status === "completed") return 100;
  if (currentIndex === -1) return 0;

  // Calculate progress within current phase based on agent completion
  const safeSessions = workflow.agentSessions || [];
  const agentProgress = safeSessions
    .filter(s => s.phase === workflow.currentPhase)
    .reduce((completed, session) => {
      return completed + (session.status === "Completed" ? 1 : 0);
    }, 0);

  const totalAgentsInPhase = safeSessions.filter(s => s.phase === workflow.currentPhase).length;
  const phaseProgress = totalAgentsInPhase > 0 ? agentProgress / totalAgentsInPhase : 0;

  // Each phase is 25% of total progress
  const baseProgress = currentIndex * 25;
  const currentPhaseProgress = phaseProgress * 25;

  return Math.min(baseProgress + currentPhaseProgress, 100);
}

export default function RFEWorkflowsPage() {
  const searchParams = useSearchParams();
  const project = searchParams.get("project");
  const [workflows, setWorkflows] = useState<RFEWorkflow[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchWorkflows = async () => {
    try {
      setIsLoading(true);
      const base = project
        ? `${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows`
        : `${getApiUrl()}/rfe-workflows`;
      const response = await fetch(base);

      if (!response.ok) {
        throw new Error(`Failed to fetch RFE workflows: ${response.status}`);
      }

      const data = await response.json();
      setWorkflows(data.workflows || []);
      setError(null);
    } catch (err) {
      console.error("Error fetching RFE workflows:", err);
      setError(
        err instanceof Error
          ? err.message
          : "Failed to load RFE workflows"
      );
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchWorkflows();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project]);

  const handleDeleteWorkflow = async (workflowId: string) => {
    if (!confirm("Are you sure you want to delete this RFE workflow?")) {
      return;
    }

    try {
      const url = project
        ? `${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${workflowId}`
        : `${getApiUrl()}/rfe-workflows/${workflowId}`;
      const response = await fetch(url, {
        method: "DELETE",
      });

      if (!response.ok) {
        throw new Error(`Failed to delete workflow: ${response.status}`);
      }

      setWorkflows(workflows.filter(w => w.id !== workflowId));
    } catch (err) {
      console.error("Error deleting workflow:", err);
      // Could show a toast notification here
    }
  };

  const handlePauseWorkflow = async (workflowId: string) => {
    try {
      const url = project
        ? `${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${workflowId}/pause`
        : `${getApiUrl()}/rfe-workflows/${workflowId}/pause`;
      const response = await fetch(url, {
        method: "POST",
      });

      if (response.ok) {
        fetchWorkflows();
      }
    } catch (err) {
      console.error("Error pausing workflow:", err);
    }
  };

  const handleResumeWorkflow = async (workflowId: string) => {
    try {
      const url = project
        ? `${getApiUrl()}/projects/${encodeURIComponent(project)}/rfe-workflows/${workflowId}/resume`
        : `${getApiUrl()}/rfe-workflows/${workflowId}/resume`;
      const response = await fetch(url, {
        method: "POST",
      });

      if (response.ok) {
        fetchWorkflows();
      }
    } catch (err) {
      console.error("Error resuming workflow:", err);
    }
  };

  if (error) {
    return (
      <div className="container mx-auto py-8">
        <Card className="border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <p className="text-red-600">Error: {error}</p>
            <Button onClick={fetchWorkflows} className="mt-4" variant="outline">
              <RefreshCw className="mr-2 h-4 w-4" />
              Retry
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold">RFE Workflows</h1>
          <p className="text-muted-foreground">
            Manage Request for Enhancement workflows with AI agents
          </p>
        </div>
        <div className="flex gap-2">
          <Button onClick={fetchWorkflows} variant="outline" disabled={isLoading}>
            <RefreshCw className={`mr-2 h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          <Link href={project ? `/rfe/new?project=${encodeURIComponent(project)}` : "/rfe/new"}>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Create RFE Workflow
            </Button>
          </Link>
        </div>
      </div>

      {/* Workflows Grid */}
      {isLoading ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(6)].map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader>
                <div className="h-4 bg-gray-200 rounded w-3/4"></div>
                <div className="h-3 bg-gray-200 rounded w-1/2"></div>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <div className="h-3 bg-gray-200 rounded"></div>
                  <div className="h-3 bg-gray-200 rounded w-2/3"></div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : workflows.length === 0 ? (
        <div className="text-center py-12">
          <GitBranch className="mx-auto h-12 w-12 text-gray-400 mb-4" />
          <h3 className="text-lg font-semibold mb-2">No RFE Workflows</h3>
          <p className="text-muted-foreground mb-4">
            Get started by creating your first RFE workflow
          </p>
          <Link href={project ? `/rfe/new?project=${encodeURIComponent(project)}` : "/rfe/new"}>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Create RFE Workflow
            </Button>
          </Link>
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {workflows.map((workflow) => (
            <RFEWorkflowCard
              key={workflow.id}
              workflow={workflow}
              onDelete={() => handleDeleteWorkflow(workflow.id)}
              onPause={() => handlePauseWorkflow(workflow.id)}
              onResume={() => handleResumeWorkflow(workflow.id)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface RFEWorkflowCardProps {
  workflow: RFEWorkflow;
  onDelete: () => void;
  onPause: () => void;
  onResume: () => void;
}

function RFEWorkflowCard({ workflow, onDelete, onPause, onResume }: RFEWorkflowCardProps) {
  const progress = calculateProgress(workflow);
  const completedSessions = (workflow.agentSessions || []).filter(s => s.status === "Completed").length;

  return (
    <Card className="hover:shadow-lg transition-shadow">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <CardTitle className="text-lg">
              <Link href={project ? `/rfe/${workflow.id}?project=${encodeURIComponent(project)}` : `/rfe/${workflow.id}`} className="hover:underline">
                {workflow.title}
              </Link>
            </CardTitle>
            <CardDescription className="line-clamp-2 mt-1">
              {workflow.description}
            </CardDescription>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem asChild>
                <Link href={project ? `/rfe/${workflow.id}?project=${encodeURIComponent(project)}` : `/rfe/${workflow.id}`}>
                  <ExternalLink className="mr-2 h-4 w-4" />
                  View Details
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem asChild>
                <Link href={project ? `/rfe/${workflow.id}/edit?project=${encodeURIComponent(project)}` : `/rfe/${workflow.id}/edit`}>
                  <FileText className="mr-2 h-4 w-4" />
                  Edit Artifacts
                </Link>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              {workflow.status === "active" ? (
                <DropdownMenuItem onClick={onPause}>
                  <Pause className="mr-2 h-4 w-4" />
                  Pause Workflow
                </DropdownMenuItem>
              ) : workflow.status === "paused" ? (
                <DropdownMenuItem onClick={onResume}>
                  <Play className="mr-2 h-4 w-4" />
                  Resume Workflow
                </DropdownMenuItem>
              ) : null}
              <DropdownMenuItem onClick={onDelete} className="text-red-600">
                <Trash2 className="mr-2 h-4 w-4" />
                Delete Workflow
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {/* Status and Phase Badges */}
        <div className="flex gap-2 mt-3">
          <Badge className={getStatusColor(workflow.status)}>
            {workflow.status}
          </Badge>
          <Badge className={getPhaseColor(workflow.currentPhase)}>
            {WORKFLOW_PHASE_LABELS[workflow.currentPhase]}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Progress Bar */}
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span>Progress</span>
            <span>{Math.round(progress)}%</span>
          </div>
          <Progress value={progress} className="h-2" />
        </div>

        {/* Agent Progress */}
        <div className="flex items-center gap-2">
          <Users className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm text-muted-foreground">
            {completedSessions}/{(workflow.agentSessions || []).length} agent sessions completed
          </span>
        </div>

        {/* Repository Info */}
        <div className="flex items-center gap-2">
          <GitBranch className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm text-muted-foreground truncate">
            {workflow.targetRepoUrl.replace(/^https?:\/\//, "")}
          </span>
        </div>

        {/* Artifacts Count */}
        <div className="flex items-center gap-2">
          <FileText className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm text-muted-foreground">
            {(workflow.artifacts || []).length} artifacts generated
          </span>
        </div>

        {/* Last Activity */}
        <div className="text-xs text-muted-foreground">
          Updated {workflow.updatedAt && !isNaN(new Date(workflow.updatedAt).getTime())
            ? formatDistanceToNow(new Date(workflow.updatedAt), { addSuffix: true })
            : 'recently'
          }
        </div>
      </CardContent>
    </Card>
  );
}