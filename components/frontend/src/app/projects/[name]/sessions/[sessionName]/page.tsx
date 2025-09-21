"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import Link from "next/link";
import { formatDistanceToNow, format } from "date-fns";
import {
  ArrowLeft,
  RefreshCw,
  ExternalLink,
  Clock,
  Globe,
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
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  AgenticSession,
  AgenticSessionPhase,
} from "@/types/agentic-session";
import type { MessageObject, ToolUseBlock, ToolUseMessages, ToolResultBlock } from "@/types/agentic-session";
import { CloneSessionDialog } from "@/components/clone-session-dialog";

import { getApiUrl } from "@/lib/config";
import { Project } from "@/types/project";

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


  useEffect(() => {
    params.then(({ name, sessionName }) => {
      setProjectName(name);
      setSessionName(sessionName);
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
        `${apiUrl}/projects/${encodeURIComponent(projectName)}/sessions/${encodeURIComponent(sessionName)}/messages`
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

 
  const handleRefresh = () => {
    setLoading(true);
    fetchSession();
  };

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
      window.location.href = `/projects/${encodeURIComponent(projectName)}?tab=sessions`;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete session");
      setActionLoading(null);
    }
  };
  

  const [agenticMessages, toolUseMessages] = useMemo(() => {
    const toolUseBlocks: ToolUseBlock[] = [];
    const toolResultBlocks: ToolResultBlock[] = [];
    const agenticMessages: MessageObject[] = [];
    
    for (const message of messages) {
      if (message.type === "assistant_message" || message.type === "user_message") {
        if (typeof message.content === "object" && message.content.type === "tool_use_block") {
          toolUseBlocks.push(message.content);
        } else if (typeof message.content === "object" && message.content.type === "tool_result_block") {
          toolResultBlocks.push(message.content);
        } else {
          agenticMessages.push(message);
        }
      } else {
        agenticMessages.push(message);
      }
    }

    // Merge tool use blocks with their corresponding result blocks
    const toolUseMessages: ToolUseMessages[] = [];
    for (const toolUseBlock of toolUseBlocks) {
      const resultBlock = toolResultBlocks.find(result => result.tool_use_id === toolUseBlock.id);
      if (resultBlock) {
        toolUseMessages.push({
          type: "tool_use_messages",
          toolUseBlock: toolUseBlock,
          resultBlock: resultBlock,
        });
      }
    }

    return [agenticMessages, toolUseMessages];
  }, [messages]);

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
          <Link href={`/projects/${encodeURIComponent(projectName)}/sessions`}>
            <Button variant="ghost" size="sm">
              <ArrowLeft className="w-4 h-4 mr-2" />
              Back to Sessions
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
    <div className="container mx-auto p-6 max-w-4xl">
      <div className="flex items-center justify-between mb-6">
        <Link href={`/projects/${encodeURIComponent(projectName)}/sessions`}>
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to Sessions
          </Button>
        </Link>
        <div className="flex items-center gap-4">
          <Button variant="outline" onClick={handleRefresh} disabled={loading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>

          {/* Action buttons based on session status */}
          {session && (() => {
            const phase = session.status?.phase || "Pending";

            if (actionLoading) {
              return (
                <Button variant="outline" disabled>
                  <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                  {actionLoading}
                </Button>
              );
            }

            const buttons = [] as React.ReactNode[];

            // Clone button for all sessions
            buttons.push(
              <CloneSessionDialog
                key="clone"
                session={session}
                onSuccess={() => fetchSession()}
                trigger={
                  <Button variant="outline">
                    <Copy className="w-4 h-4 mr-2" />
                    Clone
                  </Button>
                }
              />
            );

            // Stop button for Pending/Creating/Running sessions
            if (phase === "Pending" || phase === "Creating" || phase === "Running") {
              buttons.push(
                <Button key="stop" variant="secondary" onClick={handleStop}>
                  <Square className="w-4 h-4 mr-2" />
                  Stop
                </Button>
              );
            }

            // Delete button for all sessions (except running/creating)
            if (phase !== "Running" && phase !== "Creating") {
              buttons.push(
                <Button key="delete" variant="destructive" onClick={handleDelete}>
                  <Trash2 className="w-4 h-4 mr-2" />
                  Delete
                </Button>
              );
            }

            return buttons;
          })()}
        </div>
      </div>

      <div className="space-y-6">
        {/* Header */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="text-2xl">
                  {session.spec.displayName || session.metadata.name}
                </CardTitle>
                {session.spec.displayName && (
                  <div className="text-sm text-gray-500 mb-1">{session.metadata.name}</div>
                )}
                <CardDescription>
                  Created {formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}
                </CardDescription>
              </div>
              <Badge className={getPhaseColor(session.status?.phase || "Pending")}>
                {session.status?.phase || "Pending"}
              </Badge>
            </div>
          </CardHeader>
        </Card>

        {/* Session Details */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Brain className="w-5 h-5 mr-2" />
                Agentic Prompt
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="whitespace-pre-wrap text-sm">{session.spec.prompt}</p>
            </CardContent>
          </Card>

        </div>

        {/* Configuration */}
        <Card>
          <CardHeader>
            <CardTitle>Configuration</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
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
          </CardContent>
        </Card>

        {/* Status Information */}
        {session.status && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Clock className="w-5 h-5 mr-2" />
                Execution Status
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {session.status.message && (
                  <div>
                    <p className="font-semibold text-sm">Status Message</p>
                    <p className="text-sm text-muted-foreground">{session.status.message}</p>
                  </div>
                )}

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
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
                      <p className="font-semibold">Kubernetes Job</p>
                      <p className="text-muted-foreground font-mono text-xs">{session.status.jobName}</p>
                    </div>
                  )}

                  {session.status.cost && (
                    <div>
                      <p className="font-semibold">Cost</p>
                      <p className="text-muted-foreground">${session.status.cost.toFixed(4)}</p>
                    </div>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Real-time Messages Progress*/}
        {(messages.length > 0 ||
          session.status?.phase === "Running" ||
          session.status?.phase === "Pending" ||
          session.status?.phase === "Creating") && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <span>Agentic Progress</span>
                <Badge variant="secondary">
                  {toolUseMessages.length + agenticMessages.length} message
                </Badge>
              </CardTitle>
              <CardDescription>Live analysis from Claude AI</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="max-h-96 overflow-y-auto space-y-4 bg-gray-50 rounded-lg p-4">
                {[...toolUseMessages, ...agenticMessages].map((message, index) => (
                  <StreamMessage key={`msg-${index}`} message={message} />
                ))}

                {/* Show loading message if still processing */}
                {(session.status?.phase === "Running" ||
                  session.status?.phase === "Pending" ||
                  session.status?.phase === "Creating") && (
                  <Message
                    role="bot"
                    content={
                      session.status?.phase === "Pending"
                        ? "Agentic session is queued and waiting to start..."
                        : session.status?.phase === "Creating"
                        ? "Creating agentic environment..."
                        : "Analyzing the website and generating insights..."
                    }
                    name="Claude AI"
                    isLoading={true}
                  />
                )}

                {/* Show empty state if no messages yet */}
                {(messages.length === 0) &&
                  session.status?.phase !== "Running" &&
                  session.status?.phase !== "Pending" &&
                  session.status?.phase !== "Creating" && (
                    <div className="text-center py-8 text-gray-500">
                      <Brain className="w-8 h-8 mx-auto mb-2 opacity-50" />
                      <p>No messages yet</p>
                    </div>
                  )}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Agentic Results */}
        {session.status?.message && (
          <Card>
            <CardHeader>
              <CardTitle>Agentic Results</CardTitle>
              <CardDescription>Claude&apos;s analysis of the target website</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="bg-white rounded-lg border p-6 prose prose-sm max-w-none prose-headings:text-gray-900 prose-p:text-gray-700 prose-strong:text-gray-900 prose-code:bg-gray-100 prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-pre:bg-gray-900 prose-pre:text-gray-100">
                <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]} components={outputComponents}>
                  {session.status.finalOutput}
                </ReactMarkdown>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
