"use client";

import { useState, useEffect, useMemo, useCallback } from "react";
import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { ArrowLeft, Square, Trash2, Copy } from "lucide-react";
import { useRouter } from "next/navigation";

// Custom components
import OverviewTab from "@/components/session/OverviewTab";
import MessagesTab from "@/components/session/MessagesTab";
import WorkspaceTab from "@/components/session/WorkspaceTab";
import ResultsTab from "@/components/session/ResultsTab";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CloneSessionDialog } from "@/components/clone-session-dialog";
import { Breadcrumbs } from "@/components/breadcrumbs";
import type { FileTreeNode } from "@/components/file-tree";

import type { SessionMessage } from "@/types";
import type { MessageObject, ToolUseMessages, ToolUseBlock, ToolResultBlock } from "@/types/agentic-session";
import { getPhaseColor } from "@/utils/session-helpers";

// React Query hooks
import {
  useSession,
  useSessionMessages,
  useStopSession,
  useDeleteSession,
  useSendChatMessage,
  useSendControlMessage,
  usePushSessionToGitHub,
  useAbandonSessionChanges,
  useWorkspaceList,
  useWriteWorkspaceFile,
  useAllSessionGitHubDiffs,
  workspaceKeys,
} from "@/services/queries";
import { successToast, errorToast } from "@/hooks/use-toast";
import { workspaceApi } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";

export default function ProjectSessionDetailPage({
  params,
}: {
  params: Promise<{ name: string; sessionName: string }>;
}) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [projectName, setProjectName] = useState<string>("");
  const [sessionName, setSessionName] = useState<string>("");
  const [activeTab, setActiveTab] = useState<string>("overview");
  const [promptExpanded, setPromptExpanded] = useState(false);
  const [chatInput, setChatInput] = useState("");
  const [backHref, setBackHref] = useState<string | null>(null);
  const [backLabel, setBackLabel] = useState<string | null>(null);

  // Extract params
  useEffect(() => {
    params.then(({ name, sessionName: sName }) => {
      setProjectName(name);
      setSessionName(sName);
      try {
        const url = new URL(window.location.href);
        setBackHref(url.searchParams.get("backHref"));
        setBackLabel(url.searchParams.get("backLabel"));
      } catch {}
    });
  }, [params]);

  // React Query hooks
  const { data: session, isLoading, error, refetch: refetchSession } = useSession(projectName, sessionName);
  const { data: messages = [] } = useSessionMessages(projectName, sessionName);
  const stopMutation = useStopSession();
  const deleteMutation = useDeleteSession();
  const sendChatMutation = useSendChatMessage();
  const sendControlMutation = useSendControlMessage();
  const pushToGitHubMutation = usePushSessionToGitHub();
  const abandonChangesMutation = useAbandonSessionChanges();
  const writeWorkspaceFileMutation = useWriteWorkspaceFile();

  // Workspace state
  const [wsSelectedPath, setWsSelectedPath] = useState<string | undefined>();
  const [wsFileContent, setWsFileContent] = useState<string>("");
  const [wsTree, setWsTree] = useState<FileTreeNode[]>([]);
  
  // Helper to convert absolute workspace path to relative path
  const toRelativePath = useCallback((absPath: string): string => {
    // Strip /sessions/<sessionName>/workspace/ prefix to get relative path
    const prefix = `/sessions/${sessionName}/workspace/`;
    if (absPath.startsWith(prefix)) {
      return absPath.substring(prefix.length);
    }
    // If no prefix, assume it's already relative
    return absPath;
  }, [sessionName]);
  
  // Fetch workspace root directory
  const { data: workspaceItems = [], isLoading: wsLoading } = useWorkspaceList(
    projectName,
    sessionName,
    undefined,
    { enabled: activeTab === 'workspace' }
  );

  // Update tree when workspace items change
  useEffect(() => {
    if (workspaceItems.length > 0) {
      const treeNodes: FileTreeNode[] = workspaceItems.map(item => ({
        name: item.name,
        path: item.path, // Keep the original path for display/reference
        type: item.isDir ? 'folder' : 'file',
        expanded: false,
        sizeKb: item.isDir ? undefined : item.size / 1024,
      }));
      setWsTree(treeNodes);
    }
  }, [workspaceItems]);

  const wsUnavailable = false;

  // GitHub diff state
  const [busyRepo, setBusyRepo] = useState<Record<number, 'push' | 'abandon' | null>>({});
  
  // Helper to derive repo folder from URL
  const deriveRepoFolderFromUrl = useCallback((url: string): string => {
    try {
      const cleaned = url.replace(/^git@([^:]+):/, "https://$1/");
      const u = new URL(cleaned);
      const segs = u.pathname.split('/').filter(Boolean);
      const last = segs[segs.length - 1] || "repo";
      return last.replace(/\.git$/i, "");
    } catch {
      const parts = url.split('/');
      const last = parts[parts.length - 1] || "repo";
      return last.replace(/\.git$/i, "");
    }
  }, []);

  // Fetch all repo diffs using React Query hook
  const { data: diffTotals = {}, refetch: refetchDiffs } = useAllSessionGitHubDiffs(
    projectName,
    sessionName,
    session?.spec?.repos as Array<{ input: { url: string; branch: string }; output?: { url: string; branch: string } }> | undefined,
    deriveRepoFolderFromUrl,
    { 
      enabled: !!session?.spec?.repos && activeTab === 'overview',
      sessionPhase: session?.status?.phase 
    }
  );

  // Adapter: convert SessionMessage to StreamMessage
  type RawWireMessage = SessionMessage & { payload?: unknown; timestamp?: string };
  type InnerEnvelope = {
    type?: string;
    timestamp?: string;
    payload?: Record<string, unknown> | string;
    partial?: { id: string; index: number; total: number; data: string };
    seq?: number;
  };

  const streamMessages: Array<MessageObject | ToolUseMessages> = useMemo(() => {
    const toolUseBlocks: { block: ToolUseBlock; timestamp: string }[] = [];
    const toolResultBlocks: { block: ToolResultBlock; timestamp: string }[] = [];
    const agenticMessages: MessageObject[] = [];

    for (const raw of messages as RawWireMessage[]) {
      const envelope: InnerEnvelope = ((raw?.payload as InnerEnvelope) ?? (raw as unknown as InnerEnvelope)) || {};
      const innerType: string = (raw as unknown as InnerEnvelope)?.type || envelope.type || "";
      const innerTs: string = raw?.timestamp || envelope.timestamp || new Date().toISOString();
      const payloadValue = envelope.payload;
      const innerPayload: Record<string, unknown> = (payloadValue && typeof payloadValue === 'object' && !Array.isArray(payloadValue))
        ? (payloadValue as Record<string, unknown>)
        : ((typeof envelope === 'object' ? (envelope as unknown as Record<string, unknown>) : {}) as Record<string, unknown>);
      const partial = (envelope.partial as InnerEnvelope["partial"]) || ((raw as unknown as { partial?: InnerEnvelope["partial"] })?.partial) || undefined;

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
              block: { type: "tool_use_block", id, name: toolName, input: toolInput },
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
          } else if ((innerPayload as Record<string, unknown>)?.type === 'result.message') {
            let rp: Record<string, unknown> = (innerPayload.payload as Record<string, unknown>) || {};
            if (rp && typeof rp === 'object' && 'payload' in rp && rp.payload && typeof rp.payload === 'object') {
              rp = rp.payload as Record<string, unknown>;
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
              usage: (typeof rp.usage === 'object' && rp.usage ? rp.usage as Record<string, unknown> : null),
              result: (typeof rp.result === 'string' ? rp.result : null),
              timestamp: innerTs,
            });
            if (typeof rp.result === 'string' && rp.result.trim()) {
              agenticMessages.push({
                type: "agent_message",
                content: { type: "text_block", text: String(rp.result) },
                model: "claude",
                timestamp: innerTs,
              });
            }
          } else {
            const envelopePayload = envelope.payload;
            const contentText = (innerPayload.content as Record<string, unknown> | undefined)?.text;
            const messageText = innerPayload.message;
            const nestedContentText = (innerPayload.payload as Record<string, unknown> | undefined)?.content as Record<string, unknown> | undefined;
            const text = (typeof envelopePayload === 'string')
              ? String(envelopePayload)
              : (
                  (typeof contentText === 'string' ? String(contentText) : undefined)
                  || (typeof messageText === 'string' ? String(messageText) : undefined)
                  || (typeof nestedContentText?.text === 'string' ? String(nestedContentText.text) : '')
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
          const text = (typeof envelope.payload === 'string') ? String(envelope.payload) : "";
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
          agenticMessages.push({ type: "agent_running", timestamp: innerTs });
          break;
        }
        case "agent.waiting": {
          agenticMessages.push({ type: "agent_waiting", timestamp: innerTs });
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
        toolUseMessages.push({
          type: "tool_use_messages",
          timestamp: tu.timestamp,
          toolUseBlock: tu.block,
          resultBlock: { type: "tool_result_block", tool_use_id: tu.block.id, content: null, is_error: false },
        });
      }
    }

    const all = [...agenticMessages, ...toolUseMessages];
    const sorted = all.sort((a, b) => {
      const at = new Date(a.timestamp || 0).getTime();
      const bt = new Date(b.timestamp || 0).getTime();
      return at - bt;
    });
    return session?.spec?.interactive ? sorted.filter((m) => m.type !== "result_message") : sorted;
  }, [messages, session?.spec?.interactive]);

  // Handlers
  const handleStop = () => {
    stopMutation.mutate(
      { projectName, sessionName },
      {
        onSuccess: () => successToast("Session stopped successfully"),
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to stop session"),
      }
    );
  };

  const handleDelete = () => {
    const displayName = session?.spec.displayName || session?.metadata.name;
    if (!confirm(`Are you sure you want to delete agentic session "${displayName}"? This action cannot be undone.`)) {
      return;
    }

    deleteMutation.mutate(
      { projectName, sessionName },
      {
        onSuccess: () => {
          router.push(backHref || `/projects/${encodeURIComponent(projectName)}/sessions`);
        },
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to delete session"),
      }
    );
  };

  const sendChat = () => {
    if (!chatInput.trim()) return;

    sendChatMutation.mutate(
      { projectName, sessionName, content: chatInput.trim() },
      {
        onSuccess: () => {
          setChatInput("");
          setActiveTab('messages');
        },
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to send message"),
      }
    );
  };

  const handleInterrupt = () => {
    sendControlMutation.mutate({ projectName, sessionName, type: 'interrupt' });
  };

  const handleEndSession = () => {
    sendControlMutation.mutate({ projectName, sessionName, type: 'end_session' });
  };

  // Workspace operations - using React Query with queryClient for imperative fetching
  const onWsToggle = useCallback(async (node: FileTreeNode) => {
    if (node.type !== "folder") return;
    
    // Toggle expansion
    node.expanded = !node.expanded;
    
    // If expanding, fetch children using React Query
    if (node.expanded && !node.children) {
      try {
        // Convert to relative path for API call
        const relativePath = toRelativePath(node.path);
        const items = await queryClient.fetchQuery({
          queryKey: workspaceKeys.list(projectName, sessionName, relativePath),
          queryFn: () => workspaceApi.listWorkspace(projectName, sessionName, relativePath),
        });
        node.children = items.map(item => ({
          name: item.name,
          path: item.path,
          type: item.isDir ? 'folder' : 'file',
          expanded: false,
          sizeKb: item.isDir ? undefined : item.size / 1024,
        }));
      } catch {
        errorToast('Failed to load folder contents');
      }
    }
    
    setWsTree([...wsTree]);
  }, [wsTree, projectName, sessionName, queryClient, toRelativePath]);

  const onWsSelect = useCallback(async (node: FileTreeNode) => {
    if (node.type !== "file") return;
    setWsSelectedPath(node.path);
    
    try {
      // Convert to relative path for API call
      const relativePath = toRelativePath(node.path);
      const content = await queryClient.fetchQuery({
        queryKey: workspaceKeys.file(projectName, sessionName, relativePath),
        queryFn: () => workspaceApi.readWorkspaceFile(projectName, sessionName, relativePath),
      });
      setWsFileContent(content);
    } catch {
      errorToast('Failed to read file');
    }
  }, [projectName, sessionName, queryClient, toRelativePath]);

  const writeWsFile = useCallback(async (path: string, content: string) => {
    // Convert to relative path for API call
    const relativePath = toRelativePath(path);
    writeWorkspaceFileMutation.mutate(
      { projectName, sessionName, path: relativePath, content },
      {
        onSuccess: () => {
          setWsFileContent(content);
          successToast('File saved successfully');
        },
        onError: (err) => {
          errorToast(err instanceof Error ? err.message : 'Failed to save file');
        },
      }
    );
  }, [projectName, sessionName, writeWorkspaceFileMutation, toRelativePath]);

  const buildGithubCompareUrl = useCallback((inputUrl: string, inputBranch?: string, outputUrl?: string, outputBranch?: string): string | null => {
    if (!inputUrl || !outputUrl) return null;
    const parseOwner = (url: string): { owner: string; repo: string } | null => {
      try {
        const cleaned = url.replace(/^git@([^:]+):/, "https://$1/");
        const u = new URL(cleaned);
        const segs = u.pathname.split('/').filter(Boolean);
        if (segs.length >= 2) return { owner: segs[segs.length-2], repo: segs[segs.length-1].replace(/\.git$/i, "") };
        return null;
      } catch { return null; }
    };
    const inOrg = parseOwner(inputUrl);
    const outOrg = parseOwner(outputUrl);
    if (!inOrg || !outOrg) return null;
    const base = inputBranch && inputBranch.trim() ? inputBranch : 'main';
    const head = outputBranch && outputBranch.trim() ? outputBranch : null;
    if (!head) return null;
    return `https://github.com/${inOrg.owner}/${inOrg.repo}/compare/${encodeURIComponent(base)}...${encodeURIComponent(outOrg.owner + ':' + head)}`;
  }, []);


  const latestLiveMessage = useMemo(() => {
    if (messages.length === 0) return null;
    return messages[messages.length - 1];
  }, [messages]);

  const durationMs = useMemo(() => {
    const start = session?.status?.startTime ? new Date(session.status.startTime).getTime() : undefined;
    const end = session?.status?.completionTime ? new Date(session.status.completionTime).getTime() : Date.now();
    return start ? Math.max(0, end - start) : undefined;
  }, [session?.status?.startTime, session?.status?.completionTime]);

  const subagentStats = useMemo(() => ({ uniqueCount: 0, orderedTypes: [], counts: {} as Record<string, number> }), []);

  // Loading state
  if (isLoading) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full" />
          <span className="ml-2">Loading session...</span>
        </div>
      </div>
    );
  }

  // Error state
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
            <p className="text-red-700">Error: {error instanceof Error ? error.message : "Session not found"}</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6">
      <Breadcrumbs
        items={[
          { label: 'Projects', href: '/projects' },
          { label: projectName, href: `/projects/${projectName}` },
          { label: 'Sessions', href: `/projects/${projectName}/sessions` },
          { label: session.spec.displayName || session.metadata.name },
        ]}
        className="mb-4"
      />
      <div className="flex items-center justify-start mb-6">
        <Link href={backHref || `/projects/${encodeURIComponent(projectName)}/sessions`}>
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-2" />
            {backLabel || "Back to Sessions"}
          </Button>
        </Link>
      </div>

      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-start justify-between">
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
              onSuccess={() => refetchSession()}
              trigger={
                <Button variant="outline">
                  <Copy className="w-4 h-4 mr-2" />
                  Clone
                </Button>
              }
            />

            {session.status?.phase !== "Running" && session.status?.phase !== "Creating" && (
              <Button
                variant="destructive"
                onClick={handleDelete}
                disabled={deleteMutation.isPending}
              >
                <Trash2 className="w-4 h-4 mr-2" />
                {deleteMutation.isPending ? "Deleting..." : "Delete"}
              </Button>
            )}

            {(session.status?.phase === "Pending" || session.status?.phase === "Creating" || session.status?.phase === "Running") && (
              <Button
                variant="secondary"
                onClick={handleStop}
                disabled={stopMutation.isPending}
              >
                  <Square className="w-4 h-4 mr-2" />
                {stopMutation.isPending ? "Stopping..." : "Stop"}
                </Button>
            )}
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
          <Card className="py-4">
            <CardContent>
              <div className="text-xs text-muted-foreground">Duration</div>
              <div className="text-lg font-semibold">{typeof durationMs === "number" ? `${durationMs} ms` : "-"}</div>
            </CardContent>
          </Card>
          <Card className="py-4">
            <CardContent>
              <div className="text-xs text-muted-foreground">Messages</div>
              <div className="text-lg font-semibold">{messages.length}</div>
            </CardContent>
          </Card>
          <Card className="py-4">
            <CardContent>
              <div className="text-xs text-muted-foreground">Agents</div>
              <div className="text-lg font-semibold">{subagentStats.uniqueCount > 0 ? subagentStats.uniqueCount : "-"}</div>
            </CardContent>
          </Card>
        </div>

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="messages">Messages</TabsTrigger>
            <TabsTrigger value="workspace">Workspace</TabsTrigger>
            <TabsTrigger value="results">Results</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-6">
            <OverviewTab
              session={session}
              promptExpanded={promptExpanded}
              setPromptExpanded={setPromptExpanded}
              latestLiveMessage={latestLiveMessage as SessionMessage | null}
              diffTotals={diffTotals}
              onPush={async (idx) => {
                  const repo = session.spec.repos?.[idx];
                  if (!repo) return;
                
                setBusyRepo((b) => ({ ...b, [idx]: 'push' }));
                  const folder = deriveRepoFolderFromUrl(repo.input.url);
                const repoPath = `/sessions/${sessionName}/workspace/${folder}`;
                
                pushToGitHubMutation.mutate(
                  { projectName, sessionName, repoIndex: idx, repoPath },
                  {
                    onSuccess: () => {
                      refetchDiffs();
                      successToast('Changes pushed to GitHub');
                    },
                    onError: (err) => errorToast(err instanceof Error ? err.message : 'Failed to push changes'),
                    onSettled: () => setBusyRepo((b) => ({ ...b, [idx]: null })),
                  }
                );
              }}
              onAbandon={async (idx) => {
                  const repo = session.spec.repos?.[idx];
                  if (!repo) return;
                
                setBusyRepo((b) => ({ ...b, [idx]: 'abandon' }));
                  const folder = deriveRepoFolderFromUrl(repo.input.url);
                const repoPath = `/sessions/${sessionName}/workspace/${folder}`;
                
                abandonChangesMutation.mutate(
                  { projectName, sessionName, repoIndex: idx, repoPath },
                  {
                    onSuccess: () => {
                      refetchDiffs();
                      successToast('Changes abandoned');
                    },
                    onError: (err) => errorToast(err instanceof Error ? err.message : 'Failed to abandon changes'),
                    onSettled: () => setBusyRepo((b) => ({ ...b, [idx]: null })),
                  }
                );
              }}
              busyRepo={busyRepo}
              buildGithubCompareUrl={buildGithubCompareUrl}
              onRefreshDiff={async () => { await refetchDiffs(); }}
            />
          </TabsContent>

          <TabsContent value="messages">
            <MessagesTab
              session={session}
              streamMessages={streamMessages}
              chatInput={chatInput}
              setChatInput={setChatInput}
              onSendChat={() => Promise.resolve(sendChat())}
              onInterrupt={() => Promise.resolve(handleInterrupt())}
              onEndSession={() => Promise.resolve(handleEndSession())}
              onGoToResults={() => setActiveTab('results')}
            />
          </TabsContent>

          <TabsContent value="workspace">
            <WorkspaceTab
              session={session}
              wsLoading={wsLoading}
              wsUnavailable={wsUnavailable}
              wsTree={wsTree}
              wsSelectedPath={wsSelectedPath}
              wsFileContent={wsFileContent}
              onRefresh={() => Promise.resolve()}
              onSelect={onWsSelect}
              onToggle={onWsToggle}
              onSave={writeWsFile}
              setWsFileContent={setWsFileContent}
            />
          </TabsContent>

          <TabsContent value="results">
            <ResultsTab result={null} meta={null} />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}

