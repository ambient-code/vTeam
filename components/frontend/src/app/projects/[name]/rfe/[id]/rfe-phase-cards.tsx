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
                              ? `MULTI-AGENT COLLABORATIVE SPECIFICATION WORKFLOW:

You MUST follow this structured workflow to create a comprehensive feature specification using multiple agent perspectives.

## CRITICAL REQUIREMENT: Citations and Evidence

**ALL facts, statistics, data points, market research, and claims MUST include proper citations with actual links to sources.**

Citation Requirements:
- Citations MUST be links to actual sources that were reviewed (web pages, documents, files)
- Use WebSearch or WebFetch tools to find and verify sources before citing them
- For internal documents: Link to file path or repository location
- For external sources: Full URL to the actual page/document
- If you cannot find a credible source with a link, DO NOT make up a citation
- If you cannot find a credible source, DO NOT include the claim

**FORBIDDEN**: Making up links, citing sources you didn't actually review, or claiming statistics without verified sources.

Citation Format:
- "Claim or data point" [[Source Title](actual-url-or-file-path)]
- For assumptions based on common practice: Clearly label as [Assumption: reasoning] - no citation needed for explicit assumptions

Examples of CORRECT citations:
- "Next.js App Router supports streaming SSR" [[Next.js Documentation](https://nextjs.org/docs/app/building-your-application/routing)]
- "The current authentication flow requires 5 steps" [[Source: specs/001-auth/spec.md](specs/001-auth/spec.md)]
- "Users expect checkout to complete in under 3 minutes" [Assumption: based on standard e-commerce UX practices]

Examples of INCORRECT/FORBIDDEN:
- "87% of developers prefer dark mode [Source: Stack Overflow Survey 2024]" - NO LINK, FORBIDDEN
- "Studies show users prefer this" - NO SOURCE AT ALL, FORBIDDEN
- "Research from MIT indicates..." [[MIT Study](https://fake-url.com)] - MADE UP LINK, FORBIDDEN

**If you need data to support a claim:**
1. Use WebSearch to find actual sources
2. Use WebFetch to verify the content
3. Include the real URL in your citation
4. If you cannot find a source, remove the claim or mark it as an assumption

## Step 1: PM-Led Initial Outline

Invoke Parker (Product Manager) to create an initial RFE outline based on rfe.md (or if that doesn't exist, use these requirements: ${workflow.description}).

**CRITICAL**: When invoking Parker, you MUST include the complete "CRITICAL REQUIREMENT: Citations and Evidence" section above in Parker's prompt. Parker must follow all citation requirements.

Parker should create an outline containing:
- Executive summary with business justification and market analysis (with proper citations)
- Business impact and customer requirements (with proper citations)
- Technical approach (high-level, no implementation details)
- User experience considerations (with proper citations where applicable)
- Implementation scope (in-scope vs out-of-scope)
- Acceptance criteria
- Risks and mitigation strategies
- Success metrics (with proper citations for benchmarks)

## Step 2: Agent Selection

Parker should analyze the feature type and select TWO agents best suited to review the outline:

**Selection criteria by feature type**:
- Infrastructure/Platform features → Archie (Architect) + Stella (Staff Engineer)
- User-facing UI features → Felix (UX Feature Lead) + Stella (Staff Engineer)
- API/Integration features → Archie (Architect) + Taylor (Team Member)
- Documentation features → Terry (Technical Writer) + Casey (Content Strategist)
- Testing/Quality features → Neil (Test Engineer) + Stella (Staff Engineer)

## Step 3: Parallel Agent Reviews

Launch BOTH selected agents in parallel (single message with multiple Task tool calls) to review Parker's outline.

**CRITICAL**: When invoking each review agent, you MUST include the complete "CRITICAL REQUIREMENT: Citations and Evidence" section in their prompts. Each agent must verify that all claims in Parker's outline have proper citations and flag any unsupported claims.

Each agent should:
- Review from their domain expertise perspective
- Verify all citations are actual links to reviewed sources
- Flag any claims without proper citations or with fabricated links
- Provide structured feedback with specific recommendations
- Identify gaps, risks, architectural concerns, and improvement opportunities

## Step 4: Document Versioning

Save all versions during the refinement process:
1. Save Parker's initial outline as \`specs/[feature-dir]/outline-v1-pm-initial.md\`
2. Save Agent 1's feedback as \`specs/[feature-dir]/feedback-agent1.md\`
3. Save Agent 2's feedback as \`specs/[feature-dir]/feedback-agent2.md\`
4. Save Parker's revised outline as \`specs/[feature-dir]/outline-v2-revised.md\`

## Step 5: Final Spec Generation

Have Parker incorporate all feedback into a comprehensive revised outline, then you MUST run:
\`/speckit.specify Develop a new feature based on the revised outline\`

**CRITICAL REQUIREMENTS**:
- The \`/speckit.specify\` command MUST complete successfully
- It MUST generate the final \`spec.md\` file in the proper location
- NO other file can substitute for spec.md (not outline.md, not rfe.md, only spec.md)
- The spec.md MUST be generated using the collaborative outline as the foundation
- Do NOT skip this step or consider the workflow complete until spec.md exists
- If \`/specify\` fails, debug and retry until it succeeds

---

BEGIN WORKFLOW NOW.`
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
