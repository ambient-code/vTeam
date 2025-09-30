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
  ChevronRight,
  ChevronDown,
  CheckCircle2,
  XCircle,
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
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { FileTree, type FileTreeNode } from "@/components/file-tree";
import {
  AgenticSession,
  AgenticSessionPhase,
} from "@/types/agentic-session";
import type { MessageObject, ToolUseBlock, ToolUseMessages, ToolResultBlock, ResultMessage } from "@/types/agentic-session";
import { CloneSessionDialog } from "@/components/clone-session-dialog";

import { getApiUrl } from "@/lib/config";
import { cn } from "@/lib/utils";

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

export default function ProjectSessionDetailPage({ params }: { params: Promise<{ name: string; sessionName: string }> }) {
  const [projectName, setProjectName] = useState<string>("");
  const [sessionName, setSessionName] = useState<string>("");

  const [session, setSession] = useState<AgenticSession | null>(null);
  const [messages, setMessages] = useState<MessageObject[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [hasWorkspace, setHasWorkspace] = useState<boolean | null>(null);
  const [activeTab, setActiveTab] = useState<string>("overview");

  // Embedded workspace state
  const [wsTree, setWsTree] = useState<FileTreeNode[]>([]);
  const [wsSelectedPath, setWsSelectedPath] = useState<string | undefined>(undefined);
  const [wsFileContent, setWsFileContent] = useState<string>("");
  const [wsLoading, setWsLoading] = useState<boolean>(false);
  const [usageExpanded, setUsageExpanded] = useState(false);
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
      // Fetch messages from proxy endpoint to ensure latest content
      const msgResp = await fetch(
        `${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`
      );
      if (msgResp.ok) {
        const msgData = await msgResp.json();
        if (Array.isArray(msgData)) setMessages(msgData);
      }


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
    } catch {}
  }, [chatInput, projectName, sessionName, fetchSession])

  useEffect(() => {
    if (projectName && sessionName) {
      fetchSession();
      // Poll for updates every 5 seconds if the session is still running
      const interval = setInterval(() => {
        if (
          session?.status?.phase === "Pending" ||
          session?.status?.phase === "Running"
        ) {
          fetchSession();
        }
      }, 5000);
      return () => clearInterval(interval);
    }
  }, [projectName, sessionName, session?.status?.phase, fetchSession]);


  const workspaceBasePath = session?.spec?.paths?.workspace || `/agentic-sessions/${encodeURIComponent(sessionName)}/workspace`

    
  const probeWorkspace = useCallback(async () => {
    // Probe workspace existence via API proxy
    try {
      const apiUrl = getApiUrl();
      const wsResp = await fetch(
        `${apiUrl}/projects/${encodeURIComponent(projectName)}${workspaceBasePath}`
      );
      setHasWorkspace(wsResp.ok);
    } catch {
      setHasWorkspace(false);
    }
  }, [projectName, workspaceBasePath]);

  useEffect(() => {
    if (projectName && sessionName) {
      probeWorkspace();
    }
  }, [projectName, sessionName, probeWorkspace]);


 
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


  

  const allMessages = useMemo(() => {
    const toolUseBlocks: { block: ToolUseBlock; timestamp: string }[] = [];
    const toolResultBlocks: { block: ToolResultBlock; timestamp: string }[] = [];
    const agenticMessages: MessageObject[] = [];
    
    for (const message of messages) {
      if (message.type === "assistant_message" || message.type === "user_message") {
        if (typeof message.content === "object" && message.content.type === "tool_use_block") {
          toolUseBlocks.push({ block: message.content, timestamp: message.timestamp });
        } else if (typeof message.content === "object" && message.content.type === "tool_result_block") {
          toolResultBlocks.push({ block: message.content, timestamp: message.timestamp });
        } else {
          agenticMessages.push(message);
        }
      } else {
        agenticMessages.push(message);
      }
    }

    // Merge tool use blocks with their corresponding result blocks
    const toolUseMessages: ToolUseMessages[] = [];
    for (const toolUseItem of toolUseBlocks) {
      const resultItem = toolResultBlocks.find(result => result.block.tool_use_id === toolUseItem.block.id);
      if (resultItem) {
        toolUseMessages.push({
          type: "tool_use_messages",
          timestamp: toolUseItem.timestamp,
          toolUseBlock: toolUseItem.block,
          resultBlock: resultItem.block,
        });
      }
    }

    const all = [...agenticMessages, ...toolUseMessages]
    return session?.spec?.interactive ? all.filter((m) => m.type !== "result_message") : all;
  }, [messages, session?.spec?.interactive]);

  const latestDisplayMessage = useMemo(() => {
    if (allMessages.length === 0) return null;
    const sorted = [...allMessages].sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
    return sorted[sorted.length - 1];
  }, [allMessages]);

  // Stats: derive latest result metrics and duration
  const latestResult: ResultMessage | null = useMemo(() => {
    const results = messages.filter((m) => m.type === "result_message");
    return results.length > 0 ? (results[results.length - 1] as any) : null;
  }, [messages]);

  const durationMs = useMemo(() => {
    const start = session?.status?.startTime
      ? new Date(session.status.startTime).getTime()
      : undefined;
    const end = session?.status?.completionTime
      ? new Date(session.status.completionTime).getTime()
      : Date.now();
    return start ? Math.max(0, end - start) : undefined;
  }, [session?.status?.startTime, session?.status?.completionTime]);

  // Subagent aggregation from tool_use messages
  const subagentStats = useMemo(() => {
    const counts: Record<string, number> = {};
    const orderedTypes: string[] = [];
    for (const message of messages) {
      if (message.type === "assistant_message") {
        const content = message.content;
        if (content && typeof content === "object" && (content as any).type === "tool_use_block") {
          const block = content as ToolUseBlock;
          const input = block.input as unknown as Record<string, unknown> | undefined;
          const subagentType = (input?.subagent_type as string) || undefined;
          if (block.name === "Task" && typeof subagentType === "string" && subagentType.trim().length > 0) {
            counts[subagentType] = (counts[subagentType] || 0) + 1;
            if (!orderedTypes.includes(subagentType)) orderedTypes.push(subagentType);
          }
        }
      } else if (message.type === "user_message") {
        const content = message.content;
        if (content && typeof content === "object" && (content as any).type === "tool_use_block") {
          const block = content as ToolUseBlock;
          const input = block.input as unknown as Record<string, unknown> | undefined;
          const subagentType = (input?.subagent_type as string) || undefined;
          if (block.name === "Task" && typeof subagentType === "string" && subagentType.trim().length > 0) {
            counts[subagentType] = (counts[subagentType] || 0) + 1;
            if (!orderedTypes.includes(subagentType)) orderedTypes.push(subagentType);
          }
        }
      }
    }
    return { uniqueCount: orderedTypes.length, orderedTypes, counts };
  }, [messages]);

  // Workspace helpers (loaded when Workspace tab opens)
  type ListItem = { name: string; path: string; isDir: boolean; size: number; modifiedAt: string };
  const listWsPath = useCallback(async (relPath?: string) => {
    const url = new URL(`${getApiUrl()}/projects/${encodeURIComponent(projectName)}${workspaceBasePath}`, window.location.origin);
    if (relPath) url.searchParams.set("path", relPath);
    const resp = await fetch(url.toString());
    if (!resp.ok) throw new Error("Failed to list workspace");
    const data = await resp.json();
    const items: ListItem[] = Array.isArray(data.items) ? data.items : [];
    return items;
  }, [projectName, sessionName, workspaceBasePath]);

  const readWsFile = useCallback(async (rel: string) => {
    const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(projectName)}${workspaceBasePath}/${encodeURIComponent(rel)}`);
    if (!resp.ok) throw new Error("Failed to fetch file");
    const text = await resp.text();
    return text;
  }, [projectName, sessionName, workspaceBasePath]);

  const writeWsFile = useCallback(async (rel: string, content: string) => {
    const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(projectName)}${workspaceBasePath}/${encodeURIComponent(rel)}`, {
      method: "PUT",
      headers: { "Content-Type": "text/plain; charset=utf-8" },
      body: content,
    });
    if (!resp.ok) throw new Error("Failed to save file");
  }, [projectName, workspaceBasePath]);

  const buildWsRoot = useCallback(async () => {
    if (!hasWorkspace) return;
    setWsLoading(true);
    try {
      const items = await listWsPath();
      const children: FileTreeNode[] = items.map((it) => ({
        name: it.name,
        path: it.path.replace(/^\/+/, ""),
        type: it.isDir ? "folder" : "file",
        expanded: it.isDir,
        sizeKb: it.isDir ? undefined : it.size / 1024,
      }));
      setWsTree(children);
    } finally {
      setWsLoading(false);
    }
  }, [hasWorkspace, listWsPath]);

  const onWsToggle = useCallback(async (node: FileTreeNode) => {
    if (node.type !== "folder") return;
    // Lazy-load folder children when expanding
    const items = await listWsPath(node.path);
    const children: FileTreeNode[] = items.map((it) => ({
      name: it.name,
      path: `${node.path}/${it.name}`.replace(/^\/+/, ""),
      type: it.isDir ? "folder" : "file",
      expanded: false,
      sizeKb: it.isDir ? undefined : it.size / 1024,
    }));
    node.children = children;
    setWsTree((prev) => [...prev]);
  }, [listWsPath]);

  const onWsSelect = useCallback(async (node: FileTreeNode) => {
    if (node.type !== "file") return;
    setWsSelectedPath(node.path);
    const text = await readWsFile(node.path);
    setWsFileContent(text);
  }, [readWsFile]);

  useEffect(() => {
    if (activeTab === "workspace" && wsTree.length === 0) {
      buildWsRoot();
    }
  }, [activeTab, wsTree.length, buildWsRoot]);

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
            <CardContent>
              <div className="text-xs text-muted-foreground">Cost</div>
              <div className="text-lg font-semibold">{typeof session.status?.total_cost_usd === "number" ? `$${session.status.total_cost_usd.toFixed(4)}` : "-"}</div>
            </CardContent>
          </Card>
          <Card className="py-4">
            <CardContent >
              <div className="text-xs text-muted-foreground">Duration</div>
              <div className="text-lg font-semibold">{typeof durationMs === "number" ? `${durationMs} ms` : "-"}</div>
            </CardContent>
          </Card>
          <Card className="py-4">
            <CardContent >
              <div className="text-xs text-muted-foreground">Messages</div>
              <div className="text-lg font-semibold">{allMessages.length}</div>
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
            {hasWorkspace ? <TabsTrigger value="workspace">Workspace</TabsTrigger> : null}
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
                  {latestDisplayMessage ? (
                    <div className="space-y-4">
                      <StreamMessage message={latestDisplayMessage} onGoToResults={() => setActiveTab("results")} />
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
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
          </TabsContent>

          {/* Messages */}
          <TabsContent value="messages">
            <div className="flex flex-col gap-4 max-h-[60vh] overflow-y-auto pr-1">
              {allMessages
                .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())
                .map((message, index) => (
                  <StreamMessage key={`msg-${index}`} message={message} onGoToResults={() => setActiveTab("results")} />
              ))}


              {(session.status?.phase === "Running" ||
                session.status?.phase === "Pending" ||
                session.status?.phase === "Creating") && (
                <Message
                  role="bot"
                  content={session.status.message || (() => {
                    const messages = [
                      "Pretending to be productive...",
                      "Downloading more RAM...",
                      "Consulting the magic 8-ball...",
                      "Teaching bugs to behave...",
                      "Brewing digital coffee...",
                      "Rolling for initiative...",
                      "Surfing the data waves...",
                      "Juggling bits and bytes...",
                      "Tipping my fedora...",
                    ];
                    return messages[Math.floor(Math.random() * messages.length)];
                  })()}
                  name="Claude AI"
                  isLoading={true}
                />
              )}

              {(messages.length === 0) &&
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
                        placeholder="Type a message to the agent... (use /end to finish)"
                        value={chatInput}
                        onChange={(e) => setChatInput(e.target.value)}
                        rows={3}
                      />
                      <div className="flex items-center justify-between">
                        <div className="text-xs text-muted-foreground">Type <span className="font-mono">/end</span> to end the session</div>
                        <div className="flex gap-2">
                          <Button
                            variant="secondary"
                            size="sm"
                            onClick={async () => {
                              setChatInput("/end")
                              await sendChat()
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

          {/* Workspace */}
          {hasWorkspace ? (
            <TabsContent value="workspace">
              {wsLoading && (
                <div className="flex items-center justify-center h-32 text-sm text-muted-foreground">
                  <RefreshCw className="animate-spin h-4 w-4 mr-2" /> Loading workspace...
                </div>
              )}
              {!wsLoading && (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-0">
                  <div className="border rounded-md overflow-hidden">
                    <div className="p-3 border-b">
                      <h3 className="font-medium text-sm">Files</h3>
                      <p className="text-xs text-muted-foreground">{wsTree.length} items</p>
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
                                  try {
                                    await writeWsFile(wsSelectedPath, wsFileContent);
                                  } catch {
                                    // noop for now
                                  }
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
          ) : null}

          {/* Results */}
          <TabsContent value="results">
            {latestResult ? (
              <Card>
                <CardHeader>
                  <CardTitle>Agent Results</CardTitle>
                  <CardDescription>
                    <Badge variant={latestResult.is_error ? "destructive" : "secondary"} className="text-xs">
                      {latestResult.is_error ? (
                        <span className="inline-flex items-center"><XCircle className="w-3 h-3 mr-1" /> Error</span>
                      ) : (
                        <span className="inline-flex items-center"><CheckCircle2 className="w-3 h-3 mr-1" /> Success</span>
                      )}
                    </Badge>
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="mb-4">
                    <div className="flex flex-col justify-between mb-2">
                      <div className="flex justify-between w-full gap-2 text-xs text-gray-700">
                        <div><span className="font-medium">Duration:</span> {latestResult.duration_ms} ms</div>
                        <div><span className="font-medium">API:</span> {latestResult.duration_api_ms} ms</div>
                        <div><span className="font-medium">Turns:</span> {latestResult.num_turns}</div>
                        {typeof latestResult.total_cost_usd === "number" && <div><span className="font-medium">Cost:</span> ${latestResult.total_cost_usd.toFixed(4)}</div>}
                      </div>

                      {latestResult.usage && (
                        <div className="mt-2">
                          <div className="flex flex-col justify-between mb-1">
                            <div className="text-[11px] text-gray-500">Usage</div>
                            <button
                              className="text-xs text-blue-600 hover:underline inline-flex items-center gap-1"
                              onClick={() => setUsageExpanded((e) => !e)}
                              aria-expanded={usageExpanded}
                            >
                              {usageExpanded ? "Hide" : "Show"} details
                              {usageExpanded ? (
                                <ChevronDown className="w-3 h-3 text-gray-500" />
                              ) : (
                                <ChevronRight className="w-3 h-3 text-gray-500" />
                              )}
                            </button>
                          </div>

                          {!usageExpanded && (
                            <div className="text-xs text-gray-600 italic">Usage details hidden</div>
                          )}

                          {usageExpanded && (
                            <pre className="bg-gray-50 border rounded p-2 whitespace-pre-wrap break-words text-xs text-gray-800">
                              {JSON.stringify(latestResult.usage, null, 2)}
                            </pre>
                          )}
                        </div>
                      )}
                    </div>
                  </div>

                  {session.status?.result && (
                    <div className="bg-white rounded-lg prose prose-sm max-w-none prose-headings:text-gray-900 prose-p:text-gray-700 prose-strong:text-gray-900 prose-code:bg-gray-100 prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-pre:bg-gray-900 prose-pre:text-gray-100">
                      <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]} components={outputComponents}>
                        {session.status.result}
                      </ReactMarkdown>
                    </div>
                  )}
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
