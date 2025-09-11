"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams } from "next/navigation";
import { formatDistanceToNow, format } from "date-fns";
import Link from "next/link";
import {
  ArrowLeft,
  RefreshCw,
  ExternalLink,
  Clock,
  Globe,
  Brain,
  Square,
  Trash2,
} from "lucide-react";

// Custom components
import { Message } from "@/components/ui/message";
import { ToolMessage } from "@/components/ui/tool-message";

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
  ResearchSession,
  ResearchSessionPhase,
} from "@/types/research-session";

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

// Markdown components for final output
const outputComponents: Components = {
  code: ({
    inline,
    className,
    children,
    ...props
  }: {
    inline?: boolean;
    className?: string;
    children?: React.ReactNode;
  } & React.HTMLAttributes<HTMLElement>) => {
    const match = /language-(\w+)/.exec(className || "");
    return !inline && match ? (
      <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg overflow-x-auto">
        <code
          className={className}
          {...(props as React.HTMLAttributes<HTMLElement>)}
        >
          {children}
        </code>
      </pre>
    ) : (
      <code
        className="bg-gray-100 px-1 py-0.5 rounded text-sm"
        {...(props as React.HTMLAttributes<HTMLElement>)}
      >
        {children}
      </code>
    );
  },
  h1: ({ children }) => (
    <h1 className="text-2xl font-bold text-gray-900 mb-4 mt-6 border-b pb-2">
      {children}
    </h1>
  ),
  h2: ({ children }) => (
    <h2 className="text-xl font-semibold text-gray-800 mb-3 mt-5">
      {children}
    </h2>
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
    <ul className="list-disc list-inside space-y-1 my-3 text-gray-700">
      {children}
    </ul>
  ),
  ol: ({ children }) => (
    <ol className="list-decimal list-inside space-y-1 my-3 text-gray-700">
      {children}
    </ol>
  ),
  p: ({ children }) => (
    <p className="text-gray-700 leading-relaxed mb-3">{children}</p>
  ),
};

export default function SessionDetailPage() {
  const params = useParams();
  const sessionName = params.name as string;

  const [session, setSession] = useState<ResearchSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const fetchSession = useCallback(async () => {
    try {
      const response = await fetch(
        `${getApiUrl()}/research-sessions/${sessionName}`
      );
      if (!response.ok) {
        if (response.status === 404) {
          throw new Error("Research session not found");
        }
        throw new Error("Failed to fetch research session");
      }
      const data = await response.json();
      setSession(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [sessionName]);

  useEffect(() => {
    if (sessionName) {
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
  }, [sessionName, session?.status?.phase, fetchSession]);

  const handleRefresh = () => {
    setLoading(true);
    fetchSession();
  };

  const handleStop = async () => {
    if (!session) return;
    setActionLoading("stopping");
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
      await fetchSession(); // Refresh the session data
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to stop session");
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async () => {
    if (!session) return;

    const displayName = session.spec.displayName || session.metadata.name;
    if (
      !confirm(
        `Are you sure you want to delete research session "${displayName}"? This action cannot be undone.`
      )
    ) {
      return;
    }

    setActionLoading("deleting");
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
      // Redirect back to home after successful deletion
      window.location.href = "/";
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete session");
      setActionLoading(null);
    }
  };

  if (loading) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="animate-spin h-8 w-8" />
          <span className="ml-2">Loading research session...</span>
        </div>
      </div>
    );
  }

  if (error || !session) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center mb-6">
          <Link href="/">
            <Button variant="ghost" size="sm">
              <ArrowLeft className="w-4 h-4 mr-2" />
              Back to Sessions
            </Button>
          </Link>
        </div>
        <Card className="border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <p className="text-red-700">
              Error: {error || "Session not found"}
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6 max-w-4xl">
      <div className="flex items-center justify-between mb-6">
        <Link href="/">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to Sessions
          </Button>
        </Link>
        <div className="flex items-center gap-4">
          <Button variant="outline" onClick={handleRefresh} disabled={loading}>
            <RefreshCw
              className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`}
            />
            Refresh
          </Button>

          {/* Action buttons based on session status */}
          {session &&
            (() => {
              const phase = session.status?.phase || "Pending";

              if (actionLoading) {
                return (
                  <Button variant="outline" disabled>
                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                    {actionLoading}
                  </Button>
                );
              }

              const buttons = [];

              // Stop button for Pending/Creating/Running sessions
              if (
                phase === "Pending" ||
                phase === "Creating" ||
                phase === "Running"
              ) {
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
                  <Button
                    key="delete"
                    variant="destructive"
                    onClick={handleDelete}
                  >
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
                  <div className="text-sm text-gray-500 mb-1">
                    {session.metadata.name}
                  </div>
                )}
                <CardDescription>
                  Created{" "}
                  {formatDistanceToNow(
                    new Date(session.metadata.creationTimestamp),
                    { addSuffix: true }
                  )}
                </CardDescription>
              </div>
              <Badge
                className={getPhaseColor(session.status?.phase || "Pending")}
              >
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
                Research Prompt
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="whitespace-pre-wrap text-sm">
                {session.spec.prompt}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Globe className="w-5 h-5 mr-2" />
                Target Website
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center space-x-2">
                <div className="flex-1 min-w-0">
                  <a
                    href={session.spec.websiteURL}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center text-blue-600 hover:underline"
                  >
                    <span className="truncate block max-w-full">
                      {session.spec.websiteURL}
                    </span>
                    <ExternalLink className="w-4 h-4 ml-2 flex-shrink-0 text-blue-600 hover:text-blue-800" />
                  </a>
                </div>
              </div>
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
                <p className="text-muted-foreground">
                  {session.spec.llmSettings.model}
                </p>
              </div>
              <div>
                <p className="font-semibold">Temperature</p>
                <p className="text-muted-foreground">
                  {session.spec.llmSettings.temperature}
                </p>
              </div>
              <div>
                <p className="font-semibold">Max Tokens</p>
                <p className="text-muted-foreground">
                  {session.spec.llmSettings.maxTokens}
                </p>
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
                    <p className="text-sm text-muted-foreground">
                      {session.status.message}
                    </p>
                  </div>
                )}

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                  {session.status.startTime && (
                    <div>
                      <p className="font-semibold">Started</p>
                      <p className="text-muted-foreground">
                        {format(new Date(session.status.startTime), "PPp")}
                      </p>
                    </div>
                  )}

                  {session.status.completionTime && (
                    <div>
                      <p className="font-semibold">Completed</p>
                      <p className="text-muted-foreground">
                        {format(new Date(session.status.completionTime), "PPp")}
                      </p>
                    </div>
                  )}

                  {session.status.jobName && (
                    <div>
                      <p className="font-semibold">Kubernetes Job</p>
                      <p className="text-muted-foreground font-mono text-xs">
                        {session.status.jobName}
                      </p>
                    </div>
                  )}

                  {session.status.cost && (
                    <div>
                      <p className="font-semibold">Cost</p>
                      <p className="text-muted-foreground">
                        ${session.status.cost.toFixed(4)}
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Real-time Messages Progress*/}
        {((session.status?.messages && session.status.messages.length > 0) ||
          session.status?.phase === "Running" ||
          session.status?.phase === "Pending" ||
          session.status?.phase === "Creating") && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <span>Research Progress</span>
                <Badge variant="secondary">
                  {session.status?.messages?.length || 0} message
                  {(session.status?.messages?.length || 0) !== 1 ? "s" : ""}
                </Badge>
              </CardTitle>
              <CardDescription>Live analysis from Claude AI</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="max-h-96 overflow-y-auto space-y-4 bg-gray-50 rounded-lg p-4">
                {/* Display all existing messages */}
                {session.status?.messages?.map((message, index) => {
                  // Check if this is a tool-related message
                  const isToolMessage =
                    message.tool_use_id || message.tool_use_name;

                  if (isToolMessage) {
                    return (
                      <ToolMessage
                        key={`tool-${index}-${message.tool_use_id}`}
                        message={message}
                      />
                    );
                  } else {
                    // Regular text message
                    return (
                      <Message
                        key={`text-${index}`}
                        role="bot"
                        content={message.content || ""}
                        name="Claude AI"
                      />
                    );
                  }
                })}

                {/* Show loading message if still processing */}
                {(session.status?.phase === "Running" ||
                  session.status?.phase === "Pending" ||
                  session.status?.phase === "Creating") && (
                  <Message
                    role="bot"
                    content={
                      session.status?.phase === "Pending"
                        ? "Research session is queued and waiting to start..."
                        : session.status?.phase === "Creating"
                        ? "Creating research environment..."
                        : "Analyzing the website and generating insights..."
                    }
                    name="Claude AI"
                    isLoading={true}
                  />
                )}

                {/* Show empty state if no messages yet */}
                {(!session.status?.messages ||
                  session.status.messages.length === 0) &&
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

        {/* Research Results */}
        {session.status?.finalOutput && (
          <Card>
            <CardHeader>
              <CardTitle>Research Results</CardTitle>
              <CardDescription>
                Claude&apos;s analysis of the target website
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="bg-white rounded-lg border p-6 prose prose-sm max-w-none prose-headings:text-gray-900 prose-p:text-gray-700 prose-strong:text-gray-900 prose-code:bg-gray-100 prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-pre:bg-gray-900 prose-pre:text-gray-100">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  rehypePlugins={[rehypeHighlight]}
                  components={outputComponents}
                >
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
