"use client";

import { useState, useEffect } from "react";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  ResearchSession,
  ResearchSessionPhase,
} from "@/types/research-session";
import {
  Plus,
  RefreshCw,
  Square,
  RotateCcw,
  Trash2,
  MoreHorizontal,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { getApiUrl } from "@/lib/config";

const getPhaseColor = (phase: ResearchSessionPhase) => {
  switch (phase) {
    case "Pending":
      return "bg-yellow-100 text-yellow-800";
    case "Creating":
      return "bg-blue-100 text-blue-800";
    case "Running":
      return "bg-blue-100 text-blue-800";
    case "Completed":
      return "bg-green-100 text-green-800";
    case "Failed":
      return "bg-red-100 text-red-800";
    case "Stopped":
      return "bg-gray-100 text-gray-800";
    case "Error":
      return "bg-red-100 text-red-800";
    default:
      return "bg-gray-100 text-gray-800";
  }
};

export default function HomePage() {
  const [sessions, setSessions] = useState<ResearchSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<{ [key: string]: string }>(
    {}
  );

  const fetchSessions = async () => {
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/research-sessions`);
      if (!response.ok) {
        throw new Error("Failed to fetch research sessions");
      }
      const data = await response.json();
      setSessions(Array.isArray(data) ? data : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSessions();
    // Poll for updates every 10 seconds
    const interval = setInterval(fetchSessions, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleRefresh = () => {
    setLoading(true);
    fetchSessions();
  };

  const handleStop = async (sessionName: string) => {
    setActionLoading((prev) => ({ ...prev, [sessionName]: "stopping" }));
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(
        `${apiUrl}/research-sessions/${sessionName}/stop`,
        {
          method: "POST",
        }
      );
      if (!response.ok) {
        throw new Error("Failed to stop session");
      }
      await fetchSessions(); // Refresh the list
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to stop session");
    } finally {
      setActionLoading((prev) => {
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { [sessionName]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleRestart = async (sessionName: string) => {
    setActionLoading((prev) => ({ ...prev, [sessionName]: "restarting" }));
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(
        `${apiUrl}/research-sessions/${sessionName}/restart`,
        {
          method: "POST",
        }
      );
      if (!response.ok) {
        throw new Error("Failed to restart session");
      }
      await fetchSessions(); // Refresh the list
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to restart session"
      );
    } finally {
      setActionLoading((prev) => {
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { [sessionName]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleDelete = async (sessionName: string) => {
    if (
      !confirm(
        `Are you sure you want to delete research session "${sessionName}"? This action cannot be undone.`
      )
    ) {
      return;
    }

    setActionLoading((prev) => ({ ...prev, [sessionName]: "deleting" }));
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(
        `${apiUrl}/research-sessions/${sessionName}/delete`,
        {
          method: "DELETE",
        }
      );
      if (!response.ok) {
        throw new Error("Failed to delete session");
      }
      await fetchSessions(); // Refresh the list
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete session");
    } finally {
      setActionLoading((prev) => {
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { [sessionName]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  if (loading && (!sessions || sessions.length === 0)) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="animate-spin h-8 w-8" />
          <span className="ml-2">Loading research sessions...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold">Research Sessions</h1>
          <p className="text-muted-foreground mt-2">
            Manage your Claude research jobs
          </p>
        </div>
        <div className="flex gap-4">
          <Button variant="outline" onClick={handleRefresh} disabled={loading}>
            <RefreshCw
              className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`}
            />
            Refresh
          </Button>
          <Link href="/new">
            <Button>
              <Plus className="w-4 h-4 mr-2" />
              New Research Session
            </Button>
          </Link>
        </div>
      </div>

      {error && (
        <Card className="mb-6 border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <p className="text-red-700">Error: {error}</p>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Active Research Sessions</CardTitle>
          <CardDescription>
            Track the status and results of your research jobs
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!sessions || sessions.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-muted-foreground mb-4">
                No research sessions found
              </p>
              <Link href="/new">
                <Button>
                  <Plus className="w-4 h-4 mr-2" />
                  Create your first research session
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
                    <TableHead className="min-w-[120px]">Website</TableHead>
                    <TableHead className="hidden md:table-cell">
                      Model
                    </TableHead>
                    <TableHead className="hidden lg:table-cell">
                      Created
                    </TableHead>
                    <TableHead className="hidden xl:table-cell">Cost</TableHead>
                    <TableHead className="w-[50px]">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sessions.map((session) => (
                    <TableRow key={session.metadata.uid}>
                      <TableCell className="font-medium min-w-[180px]">
                        <Link
                          href={`/session/${session.metadata.name}`}
                          className="text-blue-600 hover:underline hover:text-blue-800 transition-colors block"
                        >
                          <div>
                            <div className="font-medium">
                              {session.spec.displayName ||
                                session.metadata.name}
                            </div>
                            {session.spec.displayName && (
                              <div className="text-xs text-gray-500 font-normal">
                                {session.metadata.name}
                              </div>
                            )}
                          </div>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <Badge
                          className={getPhaseColor(
                            session.status?.phase || "Pending"
                          )}
                        >
                          {session.status?.phase || "Pending"}
                        </Badge>
                      </TableCell>
                      <TableCell className="min-w-[120px] max-w-[200px]">
                        <a
                          href={session.spec.websiteURL}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-blue-600 hover:underline truncate block w-full"
                          title={session.spec.websiteURL}
                        >
                          {session.spec.websiteURL}
                        </a>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <span className="text-sm text-gray-600 truncate max-w-[120px] block">
                          {session.spec.llmSettings.model}
                        </span>
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        {formatDistanceToNow(
                          new Date(session.metadata.creationTimestamp),
                          {
                            addSuffix: true,
                          }
                        )}
                      </TableCell>
                      <TableCell className="hidden xl:table-cell">
                        {session.status?.cost ? (
                          <span className="text-sm font-mono">
                            ${session.status.cost.toFixed(4)}
                          </span>
                        ) : (
                          <span className="text-sm text-gray-400">—</span>
                        )}
                      </TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-8 w-8 p-0"
                            >
                              {actionLoading[session.metadata.name] ? (
                                <RefreshCw className="h-4 w-4 animate-spin" />
                              ) : (
                                <MoreHorizontal className="h-4 w-4" />
                              )}
                              <span className="sr-only">Open menu</span>
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            {(() => {
                              const sessionName = session.metadata.name;
                              const currentAction = actionLoading[sessionName];
                              const phase = session.status?.phase || "Pending";

                              if (currentAction) {
                                return (
                                  <DropdownMenuItem disabled>
                                    <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                                    {currentAction}
                                  </DropdownMenuItem>
                                );
                              }

                              const items = [];

                              // Stop action for Pending/Creating/Running sessions
                              if (
                                phase === "Pending" ||
                                phase === "Creating" ||
                                phase === "Running"
                              ) {
                                items.push(
                                  <DropdownMenuItem
                                    key="stop"
                                    onClick={() => handleStop(sessionName)}
                                    className="text-orange-600"
                                  >
                                    <Square className="mr-2 h-4 w-4" />
                                    Stop
                                  </DropdownMenuItem>
                                );
                              }

                              // Restart action for Completed/Failed/Stopped/Error sessions
                              if (
                                phase === "Completed" ||
                                phase === "Failed" ||
                                phase === "Stopped" ||
                                phase === "Error"
                              ) {
                                items.push(
                                  <DropdownMenuItem
                                    key="restart"
                                    onClick={() => handleRestart(sessionName)}
                                    className="text-blue-600"
                                  >
                                    <RotateCcw className="mr-2 h-4 w-4" />
                                    Restart
                                  </DropdownMenuItem>
                                );
                              }

                              // Separator before delete if other actions exist
                              if (items.length > 0) {
                                items.push(
                                  <DropdownMenuSeparator key="separator" />
                                );
                              }

                              // Delete action for all sessions (except running/creating)
                              if (phase !== "Running" && phase !== "Creating") {
                                items.push(
                                  <DropdownMenuItem
                                    key="delete"
                                    onClick={() => handleDelete(sessionName)}
                                    className="text-red-600"
                                  >
                                    <Trash2 className="mr-2 h-4 w-4" />
                                    Delete
                                  </DropdownMenuItem>
                                );
                              }

                              return items;
                            })()}
                          </DropdownMenuContent>
                        </DropdownMenu>
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
