"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getApiUrl } from "@/lib/config";
import { MoreVertical, Plus, RefreshCw, Square, RefreshCcw, Trash2 } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { ProjectSubpageHeader } from "@/components/project-subpage-header";
import { AgenticSession } from "@/types/agentic-session";

export default function ProjectSessionsListPage({ params }: { params: Promise<{ name: string }> }) {
  const [projectName, setProjectName] = useState<string>("");
  const [sessions, setSessions] = useState<AgenticSession[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [actionLoading, setActionLoading] = useState<Record<string, string>>({});
  const [changesMap, setChangesMap] = useState<Record<string, { label: string; count: number }>>({});
  // Keep a signature of sessions to avoid redundant diff calls on re-renders
  const lastDiffSignatureRef = useRef<string>("");
  const diffInFlightRef = useRef<boolean>(false);
  const lastDiffRunAtRef = useRef<number>(0);

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
  }, [projectName]);

  // Compute changes summary per session (sum across repos via diff API)
  useEffect(() => {
    if (!sessions || sessions.length === 0 || !projectName) return;
    if (typeof document !== 'undefined' && document.hidden) return;
    // Build a stable signature of sessions that affects diffs
    const signature = JSON.stringify(
      sessions.map((s) => ({
        n: s.metadata?.name,
        rs: Array.isArray(s.spec?.repos) ? (s.spec!.repos as any[]).map((r: any) => r?.status || "") : [],
      }))
    );
    if (signature === lastDiffSignatureRef.current) return; // No meaningful change
    // Throttle to at most once every 30s
    const now = Date.now();
    if (now - lastDiffRunAtRef.current < 30000) return;
    if (diffInFlightRef.current) return;
    lastDiffSignatureRef.current = signature;

    const run = async () => {
      diffInFlightRef.current = true;
      const apiUrl = getApiUrl();
      const next: Record<string, { label: string; count: number }> = {};
      await Promise.all((sessions || []).map(async (s) => {
        try {
          const sessionName = s.metadata.name;
          const repos = Array.isArray(s.spec?.repos) ? (s.spec!.repos as any[]) : ([] as any[]);
          const statuses = repos.map((r:any)=> (r?.status as string)||"");
          const allPushed = statuses.length>0 && statuses.every((st:string)=> st==='pushed');
          if (allPushed) {
            next[sessionName] = statuses.some((st:string)=> st==='pushed') ? { label: 'pushed', count: 0 } : { label: 'no changes', count: 0 };
            return;
          }
          const counts = await Promise.all(repos.map(async (r, idx) => {
            const url = (r?.input?.url as string) || "";
            const status = (r?.status as string) || '';
            if (!url) return 0;
            // Only compute diffs for repos without a final status; 'pushed'/'abandoned' are final
            if (status === 'pushed' || status === 'abandoned') return 0;
            const folder = (() => {
              try {
                const cleaned = url.replace(/^git@([^:]+):/, "https://$1/");
                const u = new URL(cleaned);
                const segs = u.pathname.split('/').filter(Boolean);
                const last = segs[segs.length-1] || "repo";
                return last.replace(/\.git$/i, "");
              } catch {
                const parts = url.split('/');
                const last = parts[parts.length-1] || "repo";
                return last.replace(/\.git$/i, "");
              }
            })();
            const qs = new URLSearchParams({ repoIndex: String(idx), repoPath: `/sessions/${sessionName}/workspace/${folder}` });
            const resp = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/github/diff?${qs.toString()}`);
            if (!resp.ok) return 0;
            const data = await resp.json();
            const total = Number(data.added||0)+Number(data.modified||0)+Number(data.deleted||0)+Number(data.renamed||0)+Number(data.untracked||0);
            return total;
          }));
          const total = counts.reduce((a,b)=>a+b,0);
          if (total > 0) next[s.metadata.name] = { label: `diff ${total}`, count: total };
          else if (allPushed && statuses.some((st:string)=> st==='pushed')) next[s.metadata.name] = { label: "pushed", count: 0 };
          else next[s.metadata.name] = { label: "no changes", count: 0 };
        } catch { /* ignore per-row errors */ }
      }));
      setChangesMap(next);
      lastDiffRunAtRef.current = Date.now();
      diffInFlightRef.current = false;
    };

    run();
  }, [sessions, projectName]);

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
                    <TableHead>Changes</TableHead>
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
                        {(() => {
                          const sName = session.metadata.name;
                          const ch = changesMap[sName];
                          if (!ch) return <span className="text-xs text-muted-foreground">—</span>;
                          if (ch.label === 'no changes') return <span className="text-xs text-muted-foreground">no changes</span>;
                          if (ch.label === 'pushed') return <span className="text-xs px-2 py-0.5 rounded border bg-green-50 text-green-700">pushed</span>;
                          return <span className="text-xs px-2 py-0.5 rounded border">{ch.label}</span>;
                        })()}
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


