"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Plus } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { AgenticSession, AgenticSessionPhase, WorkflowPhase } from "@/types/agentic-session";
import { WORKFLOW_PHASE_LABELS } from "@/lib/agents";
import { getPhaseColor } from "@/utils/session-helpers";

type RfeSessionsTableProps = {
  sessions: AgenticSession[];
  projectName: string;
  rfeId: string;
  workspacePath: string;
  workflowId: string;
};

export function RfeSessionsTable({
  sessions,
  projectName,
  rfeId,
  workspacePath,
  workflowId,
}: RfeSessionsTableProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Agentic Sessions ({sessions.length})</CardTitle>
            <CardDescription>Sessions scoped to this RFE</CardDescription>
          </div>
          <Link
            href={`/projects/${encodeURIComponent(projectName)}/sessions/new?workspacePath=${encodeURIComponent(workspacePath)}&rfeWorkflow=${encodeURIComponent(workflowId)}`}
          >
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
              {sessions.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-6 text-center text-muted-foreground">
                    No agent sessions yet
                  </TableCell>
                </TableRow>
              ) : (
                sessions.map((s) => {
                  const labels = (s.metadata.labels || {}) as Record<string, unknown>;
                  const name = s.metadata.name;
                  const display = s.spec?.displayName || name;
                  const rfePhase =
                    typeof labels["rfe-phase"] === "string" ? String(labels["rfe-phase"]) : "";
                  const model = s.spec?.llmSettings?.model;
                  const created = s.metadata?.creationTimestamp
                    ? formatDistanceToNow(new Date(s.metadata.creationTimestamp), { addSuffix: true })
                    : "";
                  const cost = s.status?.total_cost_usd;
                  return (
                    <TableRow key={name}>
                      <TableCell className="font-medium min-w-[180px]">
                        <Link
                          href={
                            {
                              pathname: `/projects/${encodeURIComponent(projectName)}/sessions/${encodeURIComponent(name)}`,
                              query: {
                                backHref: `/projects/${encodeURIComponent(projectName)}/rfe/${encodeURIComponent(rfeId)}?tab=sessions`,
                                backLabel: `Back to RFE`,
                              },
                            } as unknown as { pathname: string; query: Record<string, string> }
                          }
                          className="text-blue-600 hover:underline hover:text-blue-800 transition-colors block"
                        >
                          <div className="font-medium">{display}</div>
                          {display !== name && <div className="text-xs text-gray-500">{name}</div>}
                        </Link>
                      </TableCell>
                      <TableCell>
                        {WORKFLOW_PHASE_LABELS[rfePhase as WorkflowPhase] || rfePhase || "—"}
                      </TableCell>
                      <TableCell>
                        <Badge className={getPhaseColor(s.status?.phase as AgenticSessionPhase || "Pending")}>
                          {s.status?.phase || "Pending"}
                        </Badge>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <span className="text-sm text-gray-600 truncate max-w-[160px] block">
                          {model || "—"}
                        </span>
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        {created || <span className="text-gray-400">—</span>}
                      </TableCell>
                      <TableCell className="hidden xl:table-cell">
                        {cost ? (
                          <span className="text-sm font-mono">
                            ${cost.toFixed?.(4) ?? cost}
                          </span>
                        ) : (
                          <span className="text-gray-400">—</span>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}
