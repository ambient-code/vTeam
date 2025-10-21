"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CheckCircle2, Loader2, Play, Upload } from "lucide-react";
import type { AgenticSession, CreateAgenticSessionRequest, RFEWorkflow, WorkflowPhase } from "@/types/agentic-session";
import { WORKFLOW_PHASE_LABELS, AVAILABLE_AGENTS } from "@/lib/agents";
import { useCreateSession, usePublishToJira } from "@/services/queries";

type RfePhaseCardsProps = {
  workflow: RFEWorkflow;
  rfeSessions: AgenticSession[];
  rfeDoc: { exists: boolean; content: string };
  specKitDir: {
    spec: { exists: boolean; content: string };
    plan: { exists: boolean; content: string };
    tasks: { exists: boolean; content: string };
  };
  firstFeaturePath: string;
  projectName: string;
  rfeId: string;
  workflowWorkspace: string;
  isSeeded: boolean;
  startingPhase: WorkflowPhase | null;
  publishingPhase: WorkflowPhase | null;
  selectedAgents: string[];
  onStartPhase: (phase: WorkflowPhase | null) => void;
  onPublishPhase: (phase: WorkflowPhase | null) => void;
  onLoad: () => Promise<void>;
  onLoadSessions: () => Promise<void>;
  onError: (error: string) => void;
  onOpenJira: (path: string) => void;
};

export function RfePhaseCards({
  workflow,
  rfeSessions,
  rfeDoc,
  specKitDir,
  firstFeaturePath,
  projectName,
  rfeId,
  workflowWorkspace,
  isSeeded,
  startingPhase,
  publishingPhase,
  selectedAgents,
  onStartPhase,
  onPublishPhase,
  onLoad,
  onLoadSessions,
  onError,
  onOpenJira,
}: RfePhaseCardsProps) {
  const createSessionMutation = useCreateSession();
  const publishToJiraMutation = usePublishToJira();
  const phaseList = ["ideate", "specify", "plan", "tasks", "implement"] as const;

  // Helper function to generate agent instructions based on selected agents
  const getAgentInstructions = () => {
    if (selectedAgents.length === 0) return '';

    const selectedAgentDetails = selectedAgents
      .map(persona => AVAILABLE_AGENTS.find(a => a.persona === persona))
      .filter(Boolean);

    if (selectedAgentDetails.length === 0) return '';

    const agentList = selectedAgentDetails
      .map(agent => `- ${agent!.name} (${agent!.role})`)
      .join('\n');

    return `\n\nIMPORTANT - Selected Agents for this workflow:
The following agents have been selected to participate in this workflow. Invoke them by name to get their specialized perspectives:

${agentList}

You can invoke agents by using their name in your prompts. For example: "Let's get input from ${selectedAgentDetails[0]!.name} on this approach."`;
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Phase Documents</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {phaseList.map((phase) => {
            const expected = (() => {
              if (phase === "ideate") return "rfe.md";
              if (phase === "implement") return "implement";
              if (!firstFeaturePath) {
                if (phase === "specify") return "spec.md";
                if (phase === "plan") return "plan.md";
                return "tasks.md";
              }
              if (phase === "specify") return `${firstFeaturePath}/spec.md`;
              if (phase === "plan") return `${firstFeaturePath}/plan.md`;
              return `${firstFeaturePath}/tasks.md`;
            })();

            const exists =
              phase === "ideate"
                ? rfeDoc.exists
                : phase === "specify"
                  ? specKitDir.spec.exists
                  : phase === "plan"
                    ? specKitDir.plan.exists
                    : phase === "tasks"
                      ? specKitDir.tasks.exists
                      : false;

            const linkedKey = Array.isArray(
              (workflow as unknown as { jiraLinks?: Array<{ path: string; jiraKey: string }> })
                .jiraLinks
            )
              ? (
                  (
                    workflow as unknown as {
                      jiraLinks?: Array<{ path: string; jiraKey: string }>;
                    }
                  ).jiraLinks || []
                ).find((l) => l.path === expected)?.jiraKey
              : undefined;

            const sessionForPhase = rfeSessions.find(
              (s) => s.metadata.labels?.["rfe-phase"] === phase
            );
            const sessionDisplay =
              sessionForPhase && typeof sessionForPhase.spec?.displayName === "string"
                ? String(sessionForPhase.spec.displayName)
                : sessionForPhase?.metadata.name;

            return (
              <div
                key={phase}
                className={`p-4 rounded-lg border flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 ${
                  exists ? "bg-green-50 border-green-200" : ""
                }`}
              >
                <div className="flex flex-col gap-1 flex-1 min-w-0">
                  <div className="flex items-center gap-3">
                    <Badge variant="outline">{WORKFLOW_PHASE_LABELS[phase]}</Badge>
                    <span className="text-sm text-muted-foreground">{expected}</span>
                  </div>
                  {sessionForPhase && (
                    <div className="flex items-center gap-2">
                      <Link
                        href={
                          {
                            pathname: `/projects/${encodeURIComponent(projectName)}/sessions/${encodeURIComponent(sessionForPhase.metadata.name)}`,
                            query: {
                              backHref: `/projects/${encodeURIComponent(projectName)}/rfe/${encodeURIComponent(rfeId)}?tab=overview`,
                              backLabel: `Back to RFE`,
                            },
                          } as unknown as { pathname: string; query: Record<string, string> }
                        }
                      >
                        <Button variant="link" size="sm" className="px-0 h-auto">
                          {sessionDisplay}
                        </Button>
                      </Link>
                      {sessionForPhase?.status?.phase && (
                        <Badge variant="outline">{sessionForPhase.status.phase}</Badge>
                      )}
                    </div>
                  )}
                </div>
                <div className="flex items-center flex-wrap gap-3">
                  {exists ? (
                    <div className="flex items-center gap-2 text-green-700">
                      <CheckCircle2 className="h-5 w-5 text-green-600" />
                      <span className="text-sm font-medium">Ready</span>
                    </div>
                  ) : (
                    <span className="text-xs text-muted-foreground italic">
                      {phase === "plan"
                        ? "requires spec.md"
                        : phase === "tasks"
                          ? "requires plan.md"
                          : phase === "implement"
                            ? "requires tasks.md"
                            : ""}
                    </span>
                  )}
                  {!exists &&
                    (phase === "ideate" ? (
                      sessionForPhase &&
                      (sessionForPhase.status?.phase === "Running" ||
                        sessionForPhase.status?.phase === "Creating") ? (
                        <Link
                          href={
                            {
                              pathname: `/projects/${encodeURIComponent(projectName)}/sessions/${encodeURIComponent(sessionForPhase.metadata.name)}`,
                              query: {
                                backHref: `/projects/${encodeURIComponent(projectName)}/rfe/${encodeURIComponent(rfeId)}?tab=overview`,
                                backLabel: `Back to RFE`,
                              },
                            } as unknown as { pathname: string; query: Record<string, string> }
                          }
                        >
                          <Button size="sm" variant="default">
                            Enter Chat
                          </Button>
                        </Link>
                      ) : (
                        <Button
                          size="sm"
                          onClick={async () => {
                            try {
                              onStartPhase(phase);
                              const basePrompt = `IMPORTANT: The result of this interactive chat session MUST produce rfe.md at the workspace root. The rfe.md should be formatted as markdown in the following way:\n\n# Feature Title\n\n**Feature Overview:**  \n*An elevator pitch (value statement) that describes the Feature in a clear, concise way. ie: Executive Summary of the user goal or problem that is being solved, why does this matter to the user? The "What & Why"...* \n\n* Text\n\n**Goals:**\n\n*Provide high-level goal statement, providing user context and expected user outcome(s) for this Feature. Who benefits from this Feature, and how? What is the difference between today's current state and a world with this Feature?*\n\n* Text\n\n**Out of Scope:**\n\n*High-level list of items or personas that are out of scope.*\n\n* Text\n\n**Requirements:**\n\n*A list of specific needs, capabilities, or objectives that a Feature must deliver to satisfy the Feature. Some requirements will be flagged as MVP. If an MVP gets shifted, the Feature shifts. If a non MVP requirement slips, it does not shift the feature.*\n\n* Text\n\n**Done - Acceptance Criteria:**\n\n*Acceptance Criteria articulates and defines the value proposition - what is required to meet the goal and intent of this Feature. The Acceptance Criteria provides a detailed definition of scope and the expected outcomes - from a users point of view*\n\n* Text\n\n**Use Cases - i.e. User Experience & Workflow:**\n\n*Include use case diagrams, main success scenarios, alternative flow scenarios.*\n\n* Text\n\n**Documentation Considerations:**\n\n*Provide information that needs to be considered and planned so that documentation will meet customer needs. If the feature extends existing functionality, provide a link to its current documentation..*\n\n* Text\n\n**Questions to answer:**\n\n*Include a list of refinement / architectural questions that may need to be answered before coding can begin.*\n\n* Text\n\n**Background & Strategic Fit:**\n\n*Provide any additional context is needed to frame the feature.*\n\n* Text\n\n**Customer Considerations**\n\n*Provide any additional customer-specific considerations that must be made when designing and delivering the Feature.*\n\n* Text`;
                              const prompt = basePrompt + getAgentInstructions();
                              const payload: CreateAgenticSessionRequest = {
                                prompt,
                                displayName: `${workflow.title} - ${phase}`,
                                interactive: true,
                                workspacePath: workflowWorkspace,
                                autoPushOnComplete: true,
                                environmentVariables: {
                                  WORKFLOW_PHASE: phase,
                                  PARENT_RFE: workflow.id,
                                },
                                labels: {
                                  project: projectName,
                                  "rfe-workflow": workflow.id,
                                  "rfe-phase": phase,
                                },
                                annotations: {
                                  "rfe-expected": expected,
                                },
                              };
                              if (workflow.umbrellaRepo) {
                                const repos = [
                                  {
                                    input: {
                                      url: workflow.umbrellaRepo.url,
                                      branch: workflow.umbrellaRepo.branch,
                                    },
                                    output: {
                                      url: workflow.umbrellaRepo.url,
                                      branch: workflow.umbrellaRepo.branch,
                                    },
                                  },
                                  ...((workflow.supportingRepos || []).map((r) => ({
                                    input: { url: r.url, branch: r.branch },
                                    output: { url: r.url, branch: r.branch },
                                  }))),
                                ];
                                payload.repos = repos;
                                payload.mainRepoIndex = 0;
                                payload.environmentVariables = {
                                  ...(payload.environmentVariables || {}),
                                  REPOS_JSON: JSON.stringify(repos),
                                  MAIN_REPO_INDEX: "0",
                                };
                              }
                              createSessionMutation.mutate(
                                { projectName, data: payload as CreateAgenticSessionRequest },
                                {
                                  onSuccess: async () => {
                                    try {
                                      await Promise.all([onLoad(), onLoadSessions()]);
                                    } finally {
                                      onStartPhase(null);
                                    }
                                  },
                                  onError: (err) => {
                                    onError(err.message || "Failed to start session");
                                    onStartPhase(null);
                                  },
                                }
                              );
                            } catch (e) {
                              onError(e instanceof Error ? e.message : "Failed to start session");
                              onStartPhase(null);
                            }
                          }}
                          disabled={startingPhase === phase || !isSeeded}
                        >
                          {startingPhase === phase ? (
                            <>
                              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                              Starting…
                            </>
                          ) : (
                            <>
                              <Play className="mr-2 h-4 w-4" />
                              Start Chat
                            </>
                          )}
                        </Button>
                      )
                    ) : (
                      <Button
                        size="sm"
                        onClick={async () => {
                          try {
                            onStartPhase(phase);
                            const isSpecify = phase === "specify";
                            const basePrompt = isSpecify
                              ? `/speckit.specify Develop a new feature based on rfe.md or if that does not exist, follow these feature requirements: ${workflow.description}`
                              : `/speckit.${phase}`;
                            const prompt = basePrompt + getAgentInstructions();
                            const payload: CreateAgenticSessionRequest = {
                              prompt,
                              displayName: `${workflow.title} - ${phase}`,
                              interactive: false,
                              workspacePath: workflowWorkspace,
                              autoPushOnComplete: true,
                              environmentVariables: {
                                WORKFLOW_PHASE: phase,
                                PARENT_RFE: workflow.id,
                              },
                              labels: {
                                project: projectName,
                                "rfe-workflow": workflow.id,
                                "rfe-phase": phase,
                              },
                              annotations: {
                                "rfe-expected": expected,
                              },
                            };
                            if (workflow.umbrellaRepo) {
                              const repos = [
                                {
                                  input: {
                                    url: workflow.umbrellaRepo.url,
                                    branch: workflow.umbrellaRepo.branch,
                                  },
                                  output: {
                                    url: workflow.umbrellaRepo.url,
                                    branch: workflow.umbrellaRepo.branch,
                                  },
                                },
                                ...((workflow.supportingRepos || []).map((r) => ({
                                  input: { url: r.url, branch: r.branch },
                                  output: { url: r.url, branch: r.branch },
                                }))),
                              ];
                              payload.repos = repos;
                              payload.mainRepoIndex = 0;
                              payload.environmentVariables = {
                                ...(payload.environmentVariables || {}),
                                REPOS_JSON: JSON.stringify(repos),
                                MAIN_REPO_INDEX: "0",
                              };
                            }
                            createSessionMutation.mutate(
                              { projectName, data: payload as CreateAgenticSessionRequest },
                              {
                                onSuccess: async () => {
                                  try {
                                    await Promise.all([onLoad(), onLoadSessions()]);
                                  } finally {
                                    onStartPhase(null);
                                  }
                                },
                                onError: (err) => {
                                  onError(err.message || "Failed to start session");
                                  onStartPhase(null);
                                },
                              }
                            );
                          } catch (e) {
                            onError(e instanceof Error ? e.message : "Failed to start session");
                            onStartPhase(null);
                          }
                        }}
                        disabled={startingPhase === phase || !isSeeded}
                      >
                        {startingPhase === phase ? (
                          <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Starting…
                          </>
                        ) : (
                          <>
                            <Play className="mr-2 h-4 w-4" />
                            Generate
                          </>
                        )}
                      </Button>
                    ))}
                  {exists && phase !== "ideate" && (
                    <Button
                      size="sm"
                      variant="secondary"
                      onClick={() => {
                        onPublishPhase(phase);
                        publishToJiraMutation.mutate(
                          { projectName, workflowId: rfeId, path: expected },
                          {
                            onSuccess: async () => {
                              try {
                                await onLoad();
                              } finally {
                                onPublishPhase(null);
                              }
                            },
                            onError: (err) => {
                              onError(err.message || "Failed to publish to Jira");
                              onPublishPhase(null);
                            },
                          }
                        );
                      }}
                      disabled={publishingPhase === phase}
                    >
                      {publishingPhase === phase ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Publishing…
                        </>
                      ) : (
                        <>
                          <Upload className="mr-2 h-4 w-4" />
                          {linkedKey ? "Resync with Jira" : "Publish to Jira"}
                        </>
                      )}
                    </Button>
                  )}
                  {exists && linkedKey && phase !== "ideate" && (
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">{linkedKey}</Badge>
                      <Button
                        variant="link"
                        size="sm"
                        className="px-0 h-auto"
                        onClick={() => onOpenJira(expected)}
                      >
                        Open in Jira
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
