"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import Link from "next/link";
import { formatDistanceToNow, format } from "date-fns";
import {
  ArrowLeft,
  RefreshCw,
  Clock,
  Brain,
  Square,
  Trash2,
  Copy,
} from "lucide-react";

// Custom components
import { Message } from "@/components/ui/message";
import { StreamMessage } from "@/components/ui/stream-message";

// Markdown rendering
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import type { Components } from "react-markdown";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  AgenticSession,
  AgenticSessionPhase,
} from "@/types/agentic-session";
import { CloneSessionDialog } from "@/components/clone-session-dialog";
import { FileTree, type FileTreeNode } from "@/components/file-tree";

import { getApiUrl } from "@/lib/config";
import { cn } from "@/lib/utils";
import type { SessionMessage } from "@/types";
import type { MessageObject, ToolUseMessages, ToolUseBlock, ToolResultBlock } from "@/types/agentic-session";

const getPhaseColor = (phase: AgenticSessionPhase) => {
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

type CodeBlockProps = React.HTMLAttributes<HTMLElement> & {
  inline?: boolean;
  className?: string;
  children?: React.ReactNode;
};

// Markdown components for final output
const CodeBlock = ({ inline, className, children, ...props }: CodeBlockProps) => {
  const match = /language-(\w+)/.exec(className || "");
  return !inline && match ? (
    <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg overflow-x-auto">
      <code className={className} {...(props as React.HTMLAttributes<HTMLElement>)}>
        {children}
      </code>
    </pre>
  ) : (
    <code className="bg-gray-100 px-1 py-0.5 rounded text-sm" {...(props as React.HTMLAttributes<HTMLElement>)}>
      {children}
    </code>
  );
};

const outputComponents: Components = {
  code: CodeBlock,
  h1: ({ children }) => (
    <h1 className="text-2xl font-bold text-gray-900 mb-4 mt-6 border-b pb-2">
      {children}
    </h1>
  ),
  h2: ({ children }) => (
    <h2 className="text-xl font-semibold text-gray-800 mb-3 mt-5">{children}</h2>
  ),
  h3: ({ children }) => (
    <h3 className="text-lg font-medium text-gray-800 mb-2 mt-4">{children}</h3>
  ),
  blockquote: ({ children }) => (
    <blockquote className="border-l-4 border-blue-500 pl-4 py-2 bg-blue-50 italic text-gray-700 my-4">
      {children}
    </blockquote>
  ),
  ul: ({ children }) => (
    <ul className="list-disc list-inside space-y-1 my-3 text-gray-700">{children}</ul>
  ),
  ol: ({ children }) => (
    <ol className="list-decimal list-inside space-y-1 my-3 text-gray-700">{children}</ol>
  ),
  p: ({ children }) => (
    <p className="text-gray-700 leading-relaxed mb-3">{children}</p>
  ),
};

export default function ProjectSessionDetailPage({ params }: { params: Promise<{ name: string; sessionName: string }>}) {
  const [projectName, setProjectName] = useState<string>("");
  const [sessionName, setSessionName] = useState<string>("");

  const [session, setSession] = useState<AgenticSession | null>(null);
  const [liveMessages, setLiveMessages] = useState<SessionMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<string>("overview");

  // const [usageExpanded, setUsageExpanded] = useState(false);
  const [promptExpanded, setPromptExpanded] = useState(false);

  const [chatInput, setChatInput] = useState("")

  // Optional back link support via URL search params: backLabel, backHref
  const [backHref, setBackHref] = useState<string | null>(null);
  const [backLabel, setBackLabel] = useState<string | null>(null);

  useEffect(() => {
    params.then(({ name, sessionName }) => {
      setProjectName(name);
      setSessionName(sessionName);
      try {
        const url = new URL(window.location.href);
        const bh = url.searchParams.get("backHref");
        const bl = url.searchParams.get("backLabel");
        setBackHref(bh);
        setBackLabel(bl);
      } catch {}
    });
  }, [params]);
  // Adapter: convert transport SessionMessage shape into StreamMessage model
  type RawWireMessage = SessionMessage & { payload?: unknown };
  type InnerEnvelope = { type?: string; timestamp?: string; payload?: Record<string, unknown>; partial?: { id: string; index: number; total: number; data: string }; seq?: number };

  const streamMessages: Array<MessageObject | ToolUseMessages> = useMemo(() => {
    const toolUseBlocks: { block: ToolUseBlock; timestamp: string }[] = [];
    const toolResultBlocks: { block: ToolResultBlock; timestamp: string }[] = [];
    const agenticMessages: MessageObject[] = [];

    for (const raw of liveMessages as RawWireMessage[]) {
      // Some backends wrap the actual message under payload.payload (and partial under payload.partial)
      const envelope = ((raw?.payload as InnerEnvelope) ?? (raw as unknown as InnerEnvelope)) || {};
      const innerType: string = envelope.type || (raw as unknown as InnerEnvelope)?.type || "";
      // Always use the top-level timestamp for ordering/rendering
      const innerTs: string = (raw as any)?.timestamp || new Date().toISOString();
      const payloadAny = (envelope as any).payload;
      const innerPayload: Record<string, unknown> = (payloadAny && typeof payloadAny === 'object')
        ? (payloadAny as Record<string, unknown>)
        : ((typeof envelope === 'object' ? (envelope as unknown as Record<string, unknown>) : {}) as Record<string, unknown>);
      const partial = (envelope.partial as InnerEnvelope["partial"]) || ((raw as any)?.partial as InnerEnvelope["partial"]) || undefined;

      switch (innerType) {
        case "message.partial": {
          const text = partial?.data || "";
          if (text) {
            agenticMessages.push({
              type: "agent_message",
              content: { type: "text_block", text },
              model: "claude",
              timestamp: innerTs,
            });
          }
          break;
        }
        case "agent.message": {
          // Show partial text if present on agent messages
          if (partial?.data) {
            const text = String(partial.data || "");
            if (text) {
              agenticMessages.push({
                type: "agent_message",
                content: { type: "text_block", text },
                model: "claude",
                timestamp: innerTs,
              });
              break;
            }
          }

          const toolName = (innerPayload?.tool as string | undefined);
          const toolInput = (innerPayload?.input as Record<string, unknown> | undefined) || {};
          const providedId = (innerPayload?.id as string | undefined);
          const result = innerPayload?.tool_result as unknown as { tool_use_id?: string; content?: unknown; is_error?: boolean } | undefined;
          if (toolName) {
            const id = providedId ? String(providedId) : String(envelope?.seq ?? `${toolName}-${toolUseBlocks.length}`);
            toolUseBlocks.push({
              block: {
                type: "tool_use_block",
                id,
                name: toolName,
                input: toolInput,
              },
              timestamp: innerTs,
            });
          } else if (result?.tool_use_id) {
            toolResultBlocks.push({
              block: {
                type: "tool_result_block",
                tool_use_id: String(result.tool_use_id),
                content: (result.content as string | Array<Record<string, unknown>> | null | undefined) ?? null,
                is_error: Boolean(result.is_error),
              },
              timestamp: innerTs,
            });
          } else if ((innerPayload as any)?.type === 'result.message') {
            // Unwrap nested payloads: may be payload.payload
            let rp: any = (innerPayload as any).payload || {};
            if (rp && typeof rp === 'object' && 'payload' in rp && rp.payload) {
              rp = rp.payload;
            }
            agenticMessages.push({
              type: "result_message",
              subtype: String(rp.subtype || ""),
              duration_ms: Number(rp.duration_ms || 0),
              duration_api_ms: Number(rp.duration_api_ms || 0),
              is_error: Boolean(rp.is_error || false),
              num_turns: Number(rp.num_turns || 0),
              session_id: String(rp.session_id || ""),
              total_cost_usd: (typeof rp.total_cost_usd === 'number' ? rp.total_cost_usd : null),
              usage: (typeof rp.usage === 'object' && rp.usage ? rp.usage as Record<string, any> : null),
              result: (typeof rp.result === 'string' ? rp.result : null),
              timestamp: innerTs,
            } as any);
            if (typeof rp.result === 'string' && rp.result.trim()) {
              agenticMessages.push({
                type: "agent_message",
                content: { type: "text_block", text: String(rp.result) },
                model: "claude",
                timestamp: innerTs,
              } as any);
            }
          } else {
            // Free-form agent message text
            const text = (typeof (envelope as any).payload === 'string')
              ? String((envelope as any).payload)
              : (
                  // Prefer structured content block
                  (typeof (innerPayload as any)?.content?.text === 'string' ? String((innerPayload as any).content.text) : undefined)
                  || (typeof (innerPayload as any)?.message === 'string' ? String((innerPayload as any).message) : undefined)
                  || (typeof (innerPayload as any)?.payload?.content?.text === 'string' ? String((innerPayload as any).payload.content.text) : '')
                );
            if (text) {
              agenticMessages.push({
                type: "agent_message",
                content: { type: "text_block", text },
                model: "claude",
                timestamp: innerTs,
              });
            }
          }
          break;
        }
        case "system.message": {
          const text = (typeof (envelope as any).payload === 'string')
            ? String((envelope as any).payload)
            : "";
          if (text) {
            agenticMessages.push({
              type: "system_message",
              subtype: "system.message",
              data: { message: text },
              timestamp: innerTs,
            });
          }
          break;
        }
        case "user.message":
        case "user_message": {
          const text = (innerPayload?.content as string | undefined) || "";
          if (text) {
            agenticMessages.push({
              type: "user_message",
              content: { type: "text_block", text },
              timestamp: innerTs,
            });
          }
          break;
        }
        case "agent.running": {
          agenticMessages.push({
            type: "agent_running",
            timestamp: innerTs,
          });
          break;
        }
        case "agent.waiting": {
          agenticMessages.push({
            type: "agent_waiting",
            timestamp: innerTs,
          });
          break;
        }
      
        default: {
          agenticMessages.push({
            type: "system_message",
            subtype: innerType || "unknown",
            data: innerPayload || {},
            timestamp: innerTs,
          });
        }
      }
    }

    // Merge tool use blocks with their corresponding result blocks
    const toolUseMessages: ToolUseMessages[] = [];
    for (const tu of toolUseBlocks) {
      const match = toolResultBlocks.find((tr) => tr.block.tool_use_id === tu.block.id);
      if (match) {
        toolUseMessages.push({
          type: "tool_use_messages",
          timestamp: tu.timestamp,
          toolUseBlock: tu.block,
          resultBlock: match.block,
        });
      } else {
        // No result yet; show as loading
        toolUseMessages.push({
          type: "tool_use_messages",
          timestamp: tu.timestamp,
          toolUseBlock: tu.block,
          resultBlock: { type: "tool_result_block", tool_use_id: tu.block.id, content: null, is_error: false },
        });
      }
    }

    const all = [...agenticMessages, ...toolUseMessages];
    // Sort by timestamp (ascending)
    const sorted = all.sort((a, b) => {
      const at = new Date((a as any).timestamp || 0).getTime();
      const bt = new Date((b as any).timestamp || 0).getTime();
      return at - bt;
    });
    return session?.spec?.interactive ? sorted.filter((m) => m.type !== "result_message") : sorted;
  }, [liveMessages, session?.spec?.interactive]);



  const fetchSession = useCallback(async () => {
    if (!projectName || !sessionName) return;
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(
        `${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}`
      );
      if (!response.ok) {
        if (response.status === 404) {
          throw new Error("Agentic session not found");
        }
        throw new Error("Failed to fetch agentic session");
      }
      const data = await response.json();
      setSession(data);


    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [projectName, sessionName]);

  const sendChat = useCallback(async () => {
    if (!chatInput.trim() || !projectName || !sessionName) return
    try {
      const apiUrl = getApiUrl()
      await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: chatInput.trim() })
      })
      setChatInput("")
      await fetchSession()
      setActiveTab('messages')
      // Refresh live messages after sending
      try {
        const resp = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`)
        if (resp.ok) {
          const data = await resp.json()
          setLiveMessages(Array.isArray(data.messages) ? data.messages : [])
        }
      } catch {}
    } catch {}
  }, [chatInput, projectName, sessionName, fetchSession])

  useEffect(() => {
    if (projectName && sessionName) {
      fetchSession();
      // Initial load of live messages
      const apiUrl = getApiUrl()
      ;(async () => {
        try {
          const resp = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`)
          if (resp.ok) {
            const data = await resp.json()
            setLiveMessages(Array.isArray(data.messages) ? data.messages : [])
          }
        } catch {}
      })()
      // Poll for updates every 3 seconds while session is active
      const interval = setInterval(() => {
        if (
          session?.status?.phase === "Pending" ||
          session?.status?.phase === "Running" ||
          session?.status?.phase === "Creating"
        ) {
          fetchSession();
          ;(async () => {
            try {
              const resp = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`)
              if (resp.ok) {
                const data = await resp.json()
                setLiveMessages(Array.isArray(data.messages) ? data.messages : [])
              }
            } catch {}
          })()
        }
      }, 3000);
      return () => clearInterval(interval);
    }
  }, [projectName, sessionName, session?.status?.phase, fetchSession]);



  // Workspace (PVC) browser state
  const [wsTree, setWsTree] = useState<FileTreeNode[]>([]);
  const [wsSelectedPath, setWsSelectedPath] = useState<string | undefined>(undefined);
  const [wsFileContent, setWsFileContent] = useState<string>("");
  const [wsLoading, setWsLoading] = useState<boolean>(false);
  const [wsUnavailable, setWsUnavailable] = useState<boolean>(false);

  type ListItem = { name: string; path: string; isDir: boolean; size: number; modifiedAt: string };
  const listWsPath = useCallback(async (relPath?: string) => {
    try {
      const url = new URL(`${getApiUrl()}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace`, window.location.origin);
      if (relPath) url.searchParams.set("path", relPath);
      const resp = await fetch(url.toString());
      if (!resp.ok) {
        // Treat service errors as unavailable; caller will render empty state
        setWsUnavailable(true);
        return [] as ListItem[];
      }
      const data = await resp.json();
      const items: ListItem[] = Array.isArray(data.items) ? data.items : [];
      setWsUnavailable(false);
      return items;
    } catch {
      setWsUnavailable(true);
      return [] as ListItem[];
    }
  }, [projectName, sessionName]);

  const readWsFile = useCallback(async (rel: string) => {
    const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${encodeURIComponent(rel)}`);
    if (!resp.ok) throw new Error("Failed to fetch file");
    const text = await resp.text();
    return text;
  }, [projectName, sessionName]);

  const writeWsFile = useCallback(async (rel: string, content: string) => {
    const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${encodeURIComponent(rel)}`, {
      method: "PUT",
      headers: { "Content-Type": "text/plain; charset=utf-8" },
      body: content,
    });
    if (!resp.ok) throw new Error("Failed to save file");
  }, [projectName, sessionName]);

  const buildWsRoot = useCallback(async () => {
    setWsLoading(true);
    try {
      const items = await listWsPath();
      // Strip the backend's /sessions/<sessionName>/workspace/ prefix from paths
      const prefix = `/sessions/${sessionName}/workspace/`;
      const children: FileTreeNode[] = items.map((it) => ({
        name: it.name,
        path: it.path.startsWith(prefix) ? it.path.slice(prefix.length) : it.name,
        type: it.isDir ? "folder" : "file",
        expanded: it.isDir,
        sizeKb: it.isDir ? undefined : it.size / 1024,
      }));
      setWsTree(children);
    } finally {
      setWsLoading(false);
    }
  }, [listWsPath, sessionName]);

  const onWsToggle = useCallback(async (node: FileTreeNode) => {
    if (node.type !== "folder") return;
    const items = await listWsPath(node.path);
    const prefix = `/sessions/${sessionName}/workspace/`;
    const children: FileTreeNode[] = items.map((it) => ({
      name: it.name,
      path: it.path.startsWith(prefix) ? it.path.slice(prefix.length) : `${node.path}/${it.name}`,
      type: it.isDir ? "folder" : "file",
      expanded: false,
      sizeKb: it.isDir ? undefined : it.size / 1024,
    }));
    node.children = children;
    setWsTree((prev) => [...prev]);
  }, [listWsPath, sessionName]);

  const onWsSelect = useCallback(async (node: FileTreeNode) => {
    if (node.type !== "file") return;
    setWsSelectedPath(node.path);
    const text = await readWsFile(node.path);
    setWsFileContent(text);
  }, [readWsFile]);

  // Initial load when switching to workspace tab
  useEffect(() => {
    if (activeTab === "workspace" && wsTree.length === 0 && projectName && sessionName) {
      buildWsRoot();
    }
  }, [activeTab, wsTree.length, projectName, sessionName, buildWsRoot]);

  // Poll workspace while tab is active and session is running
  useEffect(() => {
    if (activeTab === "workspace" && projectName && sessionName && session?.status?.phase === "Running") {
      const interval = setInterval(() => {
        buildWsRoot();
      }, 5000); // Poll every 5 seconds
      return () => clearInterval(interval);
    }
  }, [activeTab, projectName, sessionName, session?.status?.phase, buildWsRoot]);

 
  const handleStop = async () => {
    if (!session || !projectName) return;
    setActionLoading("stopping");
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(
        `${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/stop`,
        { method: "POST" }
      );
      if (!response.ok) {
        throw new Error("Failed to stop session");
      }
      await fetchSession(); // Refresh the session data
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to stop session");
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async () => {
    if (!session || !projectName) return;

    const displayName = session.spec.displayName || session.metadata.name;
    if (
      !confirm(
        `Are you sure you want to delete agentic session "${displayName}"? This action cannot be undone.`
      )
    ) {
      return;
    }

    setActionLoading("deleting");
    try {
      const apiUrl = getApiUrl();
      const response = await fetch(
        `${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}`,
        { method: "DELETE" }
      );
      if (!response.ok) {
        throw new Error("Failed to delete session");
      }
      // Redirect back to project sessions after successful deletion
      window.location.href = backHref || `/projects/${encodeURIComponent(projectName)}?tab=sessions`;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete session");
      setActionLoading(null);
    }
  };


  

  // Latest live message (from polled backend messages)
  const latestLiveMessage = useMemo(() => {
    if (liveMessages.length === 0) return null;
    return liveMessages[liveMessages.length - 1];
  }, [liveMessages]);

  const durationMs = useMemo(() => {
    const start = session?.status?.startTime
      ? new Date(session.status.startTime).getTime()
      : undefined;
    const end = session?.status?.completionTime
      ? new Date(session.status.completionTime).getTime()
      : Date.now();
    return start ? Math.max(0, end - start) : undefined;
  }, [session?.status?.startTime, session?.status?.completionTime]);

  // Subagent aggregation not available without structured tool messages; show empty
  const subagentStats = useMemo(() => ({ uniqueCount: 0, orderedTypes: [], counts: {} as Record<string, number> }), []);


  if (loading) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="animate-spin h-8 w-8" />
          <span className="ml-2">Loading agentic session...</span>
        </div>
      </div>
    );
  }

  if (error || !session) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center mb-6">
          <Link href={backHref || `/projects/${encodeURIComponent(projectName)}/sessions`}>
            <Button variant="ghost" size="sm">
              <ArrowLeft className="w-4 h-4 mr-2" />
              {backLabel || "Back to Sessions"}
            </Button>
          </Link>
        </div>
        <Card className="border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <p className="text-red-700">Error: {error || "Session not found"}</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6">
      <div className="flex items-center justify-start mb-6">
        <Link href={backHref || `/projects/${encodeURIComponent(projectName)}/sessions`}>
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-2" />
            {backLabel || "Back to Sessions"}
          </Button>
        </Link>
      </div>

      <div className="space-y-6">
        {/* Title & phase */}
        <div className="flex items-start justify-between ">
          <div>
            <h1 className="text-2xl font-semibold flex items-center gap-2">
              <span>{session.spec.displayName || session.metadata.name}</span>
              <Badge className={getPhaseColor(session.status?.phase || "Pending")}>
                {session.status?.phase || "Pending"}
              </Badge>
            </h1>
            {session.spec.displayName && (
              <div className="text-sm text-gray-500">{session.metadata.name}</div>
            )}
            <div className="text-xs text-gray-500 mt-1">
              Created {formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <CloneSessionDialog
              session={session}
              onSuccess={() => fetchSession()}
              trigger={
                <Button variant="outline">
                  <Copy className="w-4 h-4 mr-2" />
                  Clone
                </Button>
              }
            />

            {session.status?.phase !== "Running" && session.status?.phase !== "Creating" && (
              <Button variant="destructive" onClick={handleDelete} disabled={!!actionLoading}>
                <Trash2 className="w-4 h-4 mr-2" />
                {actionLoading === "deleting" ? "Deleting..." : "Delete"}
              </Button>
            )}

            {session.status?.phase === "Pending" || session.status?.phase === "Creating" || session.status?.phase === "Running" && (
              <div>
                <Button variant="secondary" onClick={handleStop} disabled={!!actionLoading}>
                  <Square className="w-4 h-4 mr-2" />
                  {actionLoading === "stopping" ? "Stopping..." : "Stop"}
                </Button>
              </div>
            )}
          </div>
        </div>

        {/* Top compact stat cards */}
        <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
          <Card className="py-4">
            <CardContent >
              <div className="text-xs text-muted-foreground">Duration</div>
              <div className="text-lg font-semibold">{typeof durationMs === "number" ? `${durationMs} ms` : "-"}</div>
            </CardContent>
          </Card>
          <Card className="py-4">
            <CardContent >
              <div className="text-xs text-muted-foreground">Messages</div>
              <div className="text-lg font-semibold">{liveMessages.length}</div>
            </CardContent>
          </Card>
          <Card className="py-4">
            <CardContent>
              <div className="text-xs text-muted-foreground">Agents</div>
              <div className="text-lg font-semibold">{subagentStats.uniqueCount > 0 ? subagentStats.uniqueCount : "-"}</div>
              {subagentStats.orderedTypes.length > 0 ? (
                <div className="text-xs text-muted-foreground mt-1 truncate" title={subagentStats.orderedTypes.join(", ")}>{subagentStats.orderedTypes.join(", ")}</div>
              ) : null}
            </CardContent>
          </Card>
        </div>

        {/* Tabbed content */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="messages">Messages</TabsTrigger>
            <TabsTrigger value="workspace">Workspace</TabsTrigger>
            {!session.spec.interactive ? <TabsTrigger value="results">Results</TabsTrigger> : null}
          </TabsList>

          {/* Overview */}
          <TabsContent value="overview" className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Prompt */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center">
                    <Brain className="w-5 h-5 mr-2" />
                    Initial Prompt
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  {(() => {
                    const promptText = session.spec.prompt || "";
                    const promptIsLong = promptText.length > 400;
                    return (
                      <>
                        <div
                          className={cn(
                            "relative",
                            !promptExpanded && promptIsLong ? "max-h-40 overflow-hidden" : ""
                          )}
                        >
                          <p className="whitespace-pre-wrap text-sm">{promptText}</p>
                          {!promptExpanded && promptIsLong ? (
                            <div className="absolute inset-x-0 bottom-0 h-12 bg-gradient-to-t from-white to-transparent pointer-events-none" />
                          ) : null}
                        </div>
                        {promptIsLong && (
                          <button
                            className="mt-2 text-xs text-blue-600 hover:underline"
                            onClick={() => setPromptExpanded((e) => !e)}
                            aria-expanded={promptExpanded}
                            aria-controls="initial-prompt"
                          >
                            {promptExpanded ? "View less" : "View more"}
                          </button>
                        )}
                      </>
                    );
                  })()}
                </CardContent>
              </Card>
              {/* Latest Message */}
              <Card>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle>Latest Message</CardTitle>
                    <button className="text-xs text-blue-600 hover:underline" onClick={() => setActiveTab("messages")}>Go to messages</button>
                  </div>
                </CardHeader>
                <CardContent>
                  {latestLiveMessage ? (
                    <div className="space-y-2 text-sm">
                      <div className="flex items-center gap-2">
                        <Badge variant="outline" className="text-xs">{latestLiveMessage.type}</Badge>
                        <span className="text-xs text-gray-500">{new Date(latestLiveMessage.timestamp).toLocaleTimeString()}</span>
                      </div>
                      <pre className="whitespace-pre-wrap break-words bg-gray-50 rounded p-2 text-xs text-gray-800">{JSON.stringify(latestLiveMessage.payload, null, 2)}</pre>
                    </div>
                  ) : (
                    <div className="text-sm text-gray-500">No messages yet</div>
                  )}
                </CardContent>
              </Card>
            </div>
            <div className="grid grid-cols-1 gap-6">
              {/* System Status + Configuration (merged) */}
              {session.status && (
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center">
                      <Clock className="w-5 h-5 mr-2" />
                      System Status & Configuration
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4 text-sm">
                      <div>
                        <div className="text-xs font-semibold text-muted-foreground mb-2">Runtime</div>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        {session.status.message && (
                            <div>
                              <p className="font-semibold">Status</p>
                              <p className="text-muted-foreground">{session.status.message}</p>
                            </div>
                          )}
                          {session.status.startTime && (
                            <div>
                              <p className="font-semibold">Started</p>
                              <p className="text-muted-foreground">{format(new Date(session.status.startTime), "PPp")}</p>
                            </div>
                          )}
                          {session.status.completionTime && (
                            <div>
                              <p className="font-semibold">Completed</p>
                              <p className="text-muted-foreground">{format(new Date(session.status.completionTime), "PPp")}</p>
                            </div>
                          )}
                          {session.status.jobName && (
                            <div>
                              <p className="font-semibold">K8s Job</p>
                              <div className="flex items-center gap-2">
                                <p className="text-muted-foreground font-mono text-xs">{session.status.jobName}</p>
                                <Badge variant="outline" className={session.spec?.interactive ? "bg-green-50 text-green-700 border-green-200" : "bg-gray-50 text-gray-700 border-gray-200"}>
                                  {session.spec?.interactive ? "Interactive" : "Headless"}
                                </Badge>
                              </div>
                            </div>
                          )}
                        </div>
                      </div>

                      <div>
                        <div className="text-xs font-semibold text-muted-foreground mb-2">LLM Config</div>
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                          <div>
                            <p className="font-semibold">Model</p>
                            <p className="text-muted-foreground">{session.spec.llmSettings.model}</p>
                          </div>
                          <div>
                            <p className="font-semibold">Temperature</p>
                            <p className="text-muted-foreground">{session.spec.llmSettings.temperature}</p>
                          </div>
                          <div>
                            <p className="font-semibold">Max Tokens</p>
                            <p className="text-muted-foreground">{session.spec.llmSettings.maxTokens}</p>
                          </div>
                          <div>
                            <p className="font-semibold">Timeout</p>
                            <p className="text-muted-foreground">{session.spec.timeout}s</p>
                          </div>
                        </div>
                      </div>

                      <div>
                        <div className="text-xs font-semibold text-muted-foreground mb-2">Repositories</div>
                        {session.spec.repos && session.spec.repos.length > 0 ? (
                          <div className="space-y-2">
                            {session.spec.repos.map((repo, idx) => {
                              const isMain = session.spec.mainRepoIndex === idx;
                              return (
                                <div key={idx} className="flex items-center gap-2 text-sm font-mono">
                                  {isMain && <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded font-sans">MAIN</span>}
                                  <span className="text-muted-foreground break-all">{repo.input.url}</span>
                                  <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded font-sans">{repo.input.branch || "main"}</span>
                                  <span className="text-muted-foreground">→</span>
                                  <span className="text-muted-foreground break-all">{repo.output?.url || "(no push)"}</span>
                                  {repo.output?.url && (
                                    <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded font-sans">{repo.output?.branch || "auto"}</span>
                                  )}
                                </div>
                              );
                            })}
                          </div>
                        ) : (
                          <p className="text-muted-foreground">No repositories configured</p>
                        )}
                        {Array.isArray((session.spec as any)?.inputRepos) && ((session.spec as any)?.inputRepos?.length > 0) ? (
                          <div className="mt-3">
                            <div className="text-xs font-semibold text-muted-foreground mb-2">Additional Input Repos</div>
                            <div className="space-y-1 text-sm">
                              {((session.spec as any).inputRepos as Array<{ name?: string; url?: string; branch?: string }>).map((r, i) => (
                                <div key={`inrepo-${i}`} className="text-muted-foreground break-all">
                                  <span className="font-medium">{r.name || `repo-${i+1}`}:</span> {r.url || "—"}{r.branch ? <span className="text-gray-500"> @{r.branch}</span> : null}
                                </div>
                              ))}
                            </div>
                          </div>
                        ) : null}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
          </TabsContent>

          {/* Messages */}
          <TabsContent value="messages">
            <div className="flex flex-col gap-2 max-h-[60vh] overflow-y-auto pr-1">
              {streamMessages.map((m, idx) => (
                <StreamMessage key={`sm-${idx}`} message={m} isNewest={idx === streamMessages.length - 1} />
              ))}

             

              {(liveMessages.length === 0) &&
                session.status?.phase !== "Running" &&
                session.status?.phase !== "Pending" &&
                session.status?.phase !== "Creating" && (
                  <div className="text-center py-8 text-gray-500">
                    <Brain className="w-8 h-8 mx-auto mb-2 opacity-50" />
                    <p>No messages yet</p>
                  </div>
                )}

                {/* Chat composer (shown only when interactive) */}
              {session.spec?.interactive && (
                <div className="sticky bottom-0 border-t bg-white">
                  <div className="p-3">
                    <div className="border rounded-md p-3 space-y-2 bg-white">
                      <textarea
                        className="w-full border rounded p-2 text-sm"
                        placeholder="Type a message to the agent..."
                        value={chatInput}
                        onChange={(e) => setChatInput(e.target.value)}
                        rows={3}
                      />
                      <div className="flex items-center justify-between">
                        <div className="text-xs text-muted-foreground">Interactive session</div>
                        <div className="flex gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={async () => {
                              // Interrupt current agent activity
                              try {
                                const apiUrl = getApiUrl();
                                await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`, {
                                  method: 'POST',
                                  headers: { 'Content-Type': 'application/json' },
                                  body: JSON.stringify({ type: 'interrupt' })
                                })
                              } catch {}
                            }}
                          >
                            Interrupt agent
                          </Button>
                          <Button
                            variant="secondary"
                            size="sm"
                            onClick={async () => {
                              // End interactive session
                              try {
                                const apiUrl = getApiUrl();
                                await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`, {
                                  method: 'POST',
                                  headers: { 'Content-Type': 'application/json' },
                                  body: JSON.stringify({ type: 'end_session' })
                                })
                              } catch {}
                            }}
                          >
                            End session
                          </Button>
                          <Button size="sm" onClick={sendChat} disabled={!chatInput.trim()}>
                            Send
                          </Button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </TabsContent>

          {/* Workspace (PVC) */}
          <TabsContent value="workspace">
            {wsLoading && (
              <div className="flex items-center justify-center h-32 text-sm text-muted-foreground">
                <RefreshCw className="animate-spin h-4 w-4 mr-2" /> Loading workspace...
              </div>
            )}
            {!wsLoading && wsUnavailable && (
              <div className="flex items-center justify-center h-32 text-sm text-muted-foreground text-center">
                {session.status?.phase === "Pending" || session.status?.phase === "Creating" ? (
                  <div>
                    <div className="flex items-center justify-center"><RefreshCw className="animate-spin h-4 w-4 mr-2" /> Service not ready</div>
                    <div className="mt-2">{session.status?.message || "Preparing session workspace..."}</div>
                  </div>
                ) : (
                  <div>
                    <div className="font-medium">Workspace unavailable</div>
                    <div className="mt-1">Access to the PVC is not available when the session is {session.status?.phase || "Unavailable"}.</div>
                  </div>
                )}
              </div>
            )}
            {!wsLoading && !wsUnavailable && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-0">
                <div className="border rounded-md overflow-hidden">
                  <div className="p-3 border-b flex items-center justify-between">
                    <div>
                      <h3 className="font-medium text-sm">Files</h3>
                      <p className="text-xs text-muted-foreground">{wsTree.length} items</p>
                    </div>
                    <Button 
                      size="sm" 
                      variant="outline" 
                      onClick={buildWsRoot}
                      disabled={wsLoading}
                      className="h-8"
                    >
                      <RefreshCw className="h-4 w-4" />
                    </Button>
                  </div>
                  <div className="p-2">
                    <FileTree nodes={wsTree} selectedPath={wsSelectedPath} onSelect={onWsSelect} onToggle={onWsToggle} />
                  </div>
                </div>
                <div className="overflow-auto">
                  <Card className="m-3">
                    <CardContent className="p-4">
                      {wsSelectedPath ? (
                        <>
                          <div className="flex items-center justify-between mb-2">
                            <div className="text-sm">
                              <span className="font-medium">{wsSelectedPath.split('/').pop()}</span>
                              <Badge variant="outline" className="ml-2">{wsSelectedPath}</Badge>
                            </div>
                            <div className="flex items-center gap-2">
                              <Button size="sm" onClick={async () => {
                                try { await writeWsFile(wsSelectedPath, wsFileContent); } catch {}
                              }}>Save</Button>
                            </div>
                          </div>
                          <textarea
                            className="w-full h-[60vh] bg-gray-900 text-gray-100 p-4 rounded overflow-auto text-sm font-mono"
                            value={wsFileContent}
                            onChange={(e) => setWsFileContent(e.target.value)}
                          />
                        </>
                      ) : (
                        <div className="text-sm text-muted-foreground p-4">Select a file to preview</div>
                      )}
                    </CardContent>
                  </Card>
                </div>
              </div>
            )}
          </TabsContent>

          {/* Results */}
          <TabsContent value="results">
            {session.status?.result ? (
              <Card>
                <CardHeader>
                  <CardTitle>Agent Results</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="bg-white rounded-lg prose prose-sm max-w-none prose-headings:text-gray-900 prose-p:text-gray-700 prose-strong:text-gray-900 prose-code:bg-gray-100 prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-pre:bg-gray-900 prose-pre:text-gray-100">
                    <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]} components={outputComponents}>
                      {session.status.result}
                    </ReactMarkdown>
                  </div>
                </CardContent>
              </Card>
            ) : (
              <div className="text-sm text-muted-foreground">No results yet</div>
            )}
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
