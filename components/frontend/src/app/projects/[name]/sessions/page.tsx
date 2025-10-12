"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getApiUrl } from "@/lib/config";
import { MoreVertical, Plus, RefreshCw, Square, RefreshCcw, Trash2, Upload } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { ProjectSubpageHeader } from "@/components/project-subpage-header";
import { AgenticSession } from "@/types/agentic-session";

export default function ProjectSessionsListPage({ params }: { params: Promise<{ name: string }> }) {
  const [projectName, setProjectName] = useState<string>("");
  const [sessions, setSessions] = useState<AgenticSession[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [actionLoading, setActionLoading] = useState<Record<string, string>>({});
  const [jiraSyncing, setJiraSyncing] = useState<Record<string, boolean>>({});

  const fetchSessions = async () => {
    if (!projectName) return;
    try {
      setLoading(true);
      const apiUrl = getApiUrl();
      const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions`);
      if (!res.ok) throw new Error("Failed to fetch sessions");
      const data = await res.json();
      setSessions(Array.isArray(data?.items) ? data.items : []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    params.then(({ name }) => setProjectName(name));
  }, [params]);

  useEffect(() => {
    if (projectName) {
      fetchSessions();
      const i = setInterval(fetchSessions, 10000);
      return () => clearInterval(i);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectName]);

  const handleStop = async (sessionName: string) => {
    setActionLoading((prev) => ({ ...prev, [sessionName]: "stopping" }));
    try {
      const apiUrl = getApiUrl();
      const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/stop`, { method: 'POST' });
      if (!res.ok) throw new Error("Failed to stop session");
      await fetchSessions();
    } finally {
      setActionLoading((prev) => {
        const { [sessionName]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleRestart = async (sessionName: string) => {
    setActionLoading((prev) => ({ ...prev, [sessionName]: "restarting" }));
    try {
      const apiUrl = getApiUrl();
      const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/start`, { method: 'POST' });
      if (!res.ok) throw new Error("Failed to restart session");
      await fetchSessions();
    } finally {
      setActionLoading((prev) => {
        const { [sessionName]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleDelete = async (sessionName: string) => {
    if (!confirm(`Delete agentic session "${sessionName}"? This action cannot be undone.`)) return;
    setActionLoading((prev) => ({ ...prev, [sessionName]: "deleting" }));
    try {
      const apiUrl = getApiUrl();
      const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}`, { method: 'DELETE' });
      if (!res.ok) throw new Error("Failed to delete session");
      await fetchSessions();
    } finally {
      setActionLoading((prev) => {
        const { [sessionName]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleJiraSync = async (session: AgenticSession) => {
    const sessionName = session.metadata?.name;
    if (!sessionName) return;

    const labels = session.metadata?.labels as Record<string, string> | undefined;
    const rfeWorkflowId = labels?.["rfe-workflow"];
    const rfePhase = labels?.["rfe-phase"];

    if (!rfeWorkflowId || !rfePhase) {
      alert("This session is not linked to an RFE workflow phase");
      return;
    }

    // Determine expected path based on phase
    let path = "";
    if (rfePhase === "specify") {
      path = "spec.md";
    } else if (rfePhase === "plan") {
      path = "plan.md";
    } else if (rfePhase === "tasks") {
      path = "tasks.md";
    } else {
      alert(`Cannot sync phase "${rfePhase}" to Jira`);
      return;
    }

    setJiraSyncing((prev) => ({ ...prev, [sessionName]: true }));
    try {
      const apiUrl = getApiUrl();
      const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/rfe-workflows/${encodeURIComponent(rfeWorkflowId)}/jira`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path, phase: rfePhase }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data?.error || `HTTP ${res.status}`);
      }
      alert(`Successfully synced ${rfePhase} to Jira`);
      await fetchSessions();
    } catch (e) {
      alert(`Failed to sync to Jira: ${e instanceof Error ? e.message : 'Unknown error'}`);
    } finally {
      setJiraSyncing((prev) => {
        const { [sessionName]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  if (!projectName || (loading && sessions.length === 0)) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="animate-spin h-8 w-8" />
          <span className="ml-2">Loading sessions...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <ProjectSubpageHeader
        title={<>Agentic Sessions</>}
        description={<>Sessions scoped to this project</>}
        actions={
          <>
            <Link href={`/projects/${encodeURIComponent(projectName)}/sessions/new`}><Button><Plus className="w-4 h-4 mr-2" />New Session</Button></Link>
            <Button variant="outline" onClick={fetchSessions} disabled={loading}>
              <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
              Refresh
            </Button>
          </>
        }
      />
      <Card>
        <CardHeader>
          <CardTitle>Agentic Sessions ({sessions?.length || 0})</CardTitle>
          <CardDescription>Sessions scoped to this project</CardDescription>
        </CardHeader>
        <CardContent>
          {!sessions || sessions.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-muted-foreground mb-4">No sessions found</p>
              <Link href={`/projects/${encodeURIComponent(projectName)}/sessions/new`}>
                <Button>
                  <Plus className="w-4 h-4 mr-2" />
                  Create your first session
                </Button>
              </Link>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="min-w-[180px]">Name</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Mode</TableHead>
                        <TableHead className="hidden md:table-cell">Model</TableHead>
                    <TableHead className="hidden lg:table-cell">Created</TableHead>
                    <TableHead className="hidden xl:table-cell">Cost</TableHead>
                    <TableHead className="w-[50px]">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {[...sessions]
                    .sort((a, b) => {
                      const aTime = a?.metadata?.creationTimestamp ? new Date(a.metadata.creationTimestamp).getTime() : 0;
                      const bTime = b?.metadata?.creationTimestamp ? new Date(b.metadata.creationTimestamp).getTime() : 0;
                      return bTime - aTime;
                    })
                    .map((session) => (
                    <TableRow key={session.metadata?.uid || session.metadata?.name}>
                      <TableCell className="font-medium min-w-[180px]">
                        <Link href={`/projects/${projectName}/sessions/${session.metadata.name}`} className="text-blue-600 hover:underline hover:text-blue-800 transition-colors block">
                          <div>
                            <div className="font-medium">{session.spec.displayName || session.metadata.name}</div>
                            {session.spec.displayName && (
                              <div className="text-xs text-gray-500 font-normal">{session.metadata.name}</div>
                            )}
                          </div>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">{session.status?.phase || "Pending"}</span>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs px-2 py-1 rounded border bg-gray-50">
                          {session.spec?.interactive ? "Interactive" : "Headless"}
                        </span>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <span className="text-sm text-gray-600 truncate max-w-[120px] block">{session.spec.llmSettings.model}</span>
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        {session.metadata?.creationTimestamp && formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}
                      </TableCell>
                      <TableCell className="hidden xl:table-cell">
                        {session.status?.total_cost_usd ? (
                          <span className="text-sm font-mono">${session.status.total_cost_usd.toFixed(4)}</span>
                        ) : (
                          <span className="text-sm text-gray-400">—</span>
                        )}
                      </TableCell>
                      <TableCell>
                        {(() => {
                          const sessionName = session.metadata.name as string;
                          const currentAction = actionLoading[sessionName];
                          const phase = session.status?.phase || "Pending";

                          if (currentAction) {
                            return (
                              <Button variant="ghost" size="sm" className="h-8 w-8 p-0" disabled>
                                <RefreshCw className="h-4 w-4 animate-spin" />
                              </Button>
                            );
                          }

                          type RowAction = { key: string; label: string; onClick: () => void; icon: React.ReactNode; className?: string };
                          const actions: RowAction[] = [];

                          if (phase === "Pending" || phase === "Creating" || phase === "Running") {
                            actions.push({ key: "stop", label: "Stop", onClick: () => handleStop(sessionName), icon: <Square className="h-4 w-4" /> , className: "text-orange-600" });
                          }

                          if (phase === "Completed" || phase === "Failed" || phase === "Stopped" || phase === "Error") {
                            actions.push({ key: "restart", label: "Restart", onClick: () => handleRestart(sessionName), icon: <RefreshCcw className="h-4 w-4" />, className: "text-blue-600" });
                          }

                          // Add Jira sync option if session is linked to an RFE workflow and phase is completed
                          const labels = session.metadata?.labels as Record<string, string> | undefined;
                          const rfePhase = labels?.["rfe-phase"];
                          if ((phase === "Completed") && rfePhase && (rfePhase === "specify" || rfePhase === "plan" || rfePhase === "tasks")) {
                            actions.push({ key: "jira", label: "Push to Jira", onClick: () => handleJiraSync(session), icon: <Upload className="h-4 w-4" />, className: "text-green-600" });
                          }

                          if (phase !== "Running" && phase !== "Creating") {
                            actions.push({ key: "delete", label: "Delete", onClick: () => handleDelete(sessionName), icon: <Trash2 className="h-4 w-4" />, className: "text-red-600" });
                          }

                          if (actions.length <= 1) {
                            const a = actions[0];
                            return (
                              <Button variant="ghost" size="sm" className="h-8 w-8 p-0" onClick={a?.onClick} disabled={!a}>
                                {a ? a.icon : <MoreVertical className="h-4 w-4" />}
                              </Button>
                            );
                          }

                          return (
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                                  <MoreVertical className="h-4 w-4" />
                                  <span className="sr-only">Open menu</span>
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                {actions.map((a) => (
                                  <DropdownMenuItem key={a.key} onClick={a.onClick} className={a.className}>
                                    {a.label}
                                  </DropdownMenuItem>
                                ))}
                              </DropdownMenuContent>
                            </DropdownMenu>
                          );
                        })()}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}


