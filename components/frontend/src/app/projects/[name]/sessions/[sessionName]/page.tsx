"use client";

import { useState, useEffect, useMemo, useCallback } from "react";
import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { ArrowLeft, Square, Trash2, Copy, Play, MoreVertical, Bot, Loader2, FolderTree, AlertCircle, Sprout, CheckCircle2, GitBranch, Edit, Info } from "lucide-react";
import { useRouter } from "next/navigation";

// Custom components
import OverviewTab from "@/components/session/OverviewTab";
import MessagesTab from "@/components/session/MessagesTab";
import WorkspaceTab from "@/components/session/WorkspaceTab";
import ResultsTab from "@/components/session/ResultsTab";
import { EditRepositoriesDialog } from "../../rfe/[id]/edit-repositories-dialog";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { CloneSessionDialog } from "@/components/clone-session-dialog";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { PageHeader } from "@/components/page-header";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from "@/components/ui/dropdown-menu";
import { GitHubConnectionCard } from "@/components/github-connection-card";
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
  useSessionK8sResources,
  useContinueSession,
  useRfeWorkflow,
  useRfeWorkflowAgents,
  useRfeWorkflowSeeding,
  useSeedRfeWorkflow,
  useUpdateRfeWorkflow,
  useGitHubStatus,
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
  const [promptExpanded, setPromptExpanded] = useState(false);
  const [chatInput, setChatInput] = useState("");
  const [backHref, setBackHref] = useState<string | null>(null);
  const [backLabel, setBackLabel] = useState<string | null>(null);
  const [contentPodSpawning, setContentPodSpawning] = useState(false);
  const [contentPodReady, setContentPodReady] = useState(false);
  const [contentPodError, setContentPodError] = useState<string | null>(null);
  const [selectedAgents, setSelectedAgents] = useState<string[]>([]);
  const [editRepoDialogOpen, setEditRepoDialogOpen] = useState(false);
  const [selectedWorkflow, setSelectedWorkflow] = useState<string>("none");
  const [githubModalOpen, setGithubModalOpen] = useState(false);
  const [specRepoUrl, setSpecRepoUrl] = useState("https://github.com/org/repo.git");
  const [baseBranch, setBaseBranch] = useState("main");
  const [openAccordionItems, setOpenAccordionItems] = useState<string[]>([]);

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

  // Open spec-repository accordion when plan-feature or develop-feature is selected
  useEffect(() => {
    if (selectedWorkflow === "plan-feature" || selectedWorkflow === "develop-feature") {
      setOpenAccordionItems(prev => {
        if (!prev.includes("spec-repository")) {
          return [...prev, "spec-repository"];
        }
        return prev;
      });
    }
  }, [selectedWorkflow]);

  // React Query hooks
  const { data: session, isLoading, error, refetch: refetchSession } = useSession(projectName, sessionName);
  const { data: messages = [] } = useSessionMessages(projectName, sessionName, session?.status?.phase);
  const { data: k8sResources } = useSessionK8sResources(projectName, sessionName);
  const { data: githubStatus } = useGitHubStatus();
  const stopMutation = useStopSession();
  const deleteMutation = useDeleteSession();
  const continueMutation = useContinueSession();
  const sendChatMutation = useSendChatMessage();
  const sendControlMutation = useSendControlMessage();
  const pushToGitHubMutation = usePushSessionToGitHub();
  const abandonChangesMutation = useAbandonSessionChanges();
  const writeWorkspaceFileMutation = useWriteWorkspaceFile();
  
  // Get RFE workflow ID from session if this is an RFE session
  const rfeWorkflowId = session?.metadata?.labels?.['rfe-workflow-id'];
  const { data: rfeWorkflow, refetch: refetchWorkflow } = useRfeWorkflow(projectName, rfeWorkflowId || '', { enabled: !!rfeWorkflowId });
  const { data: repoAgents = [], isLoading: loadingAgents } = useRfeWorkflowAgents(
    projectName,
    rfeWorkflowId || '',
    { enabled: !!rfeWorkflowId }
  );
  const { data: seedingData, isLoading: checkingSeeding, error: seedingQueryError, refetch: refetchSeeding } = useRfeWorkflowSeeding(
    projectName,
    rfeWorkflowId || '',
    { enabled: !!rfeWorkflowId }
  );
  const seedWorkflowMutation = useSeedRfeWorkflow();
  const updateWorkflowMutation = useUpdateRfeWorkflow();

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
    { enabled: true }
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

  // Handler to refresh workspace
  const handleRefreshWorkspace = useCallback(async () => {
    // Invalidate all workspace queries to force fresh fetch
    await queryClient.invalidateQueries({
      queryKey: workspaceKeys.lists(),
    });
    await queryClient.invalidateQueries({
      queryKey: workspaceKeys.files(),
    });
  }, [queryClient]);

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
      enabled: !!session?.spec?.repos,
      sessionPhase: session?.status?.phase 
    }
  );

  // Handler to refresh diffs by invalidating cache first
  const handleRefreshDiff = useCallback(async () => {
    // Invalidate all diff queries to force fresh fetch
    await queryClient.invalidateQueries({
      queryKey: workspaceKeys.diffs(),
    });
    // Then refetch
    await refetchDiffs();
  }, [queryClient, refetchDiffs]);

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
          let text = "";
          let isDebug = false;
          
          // The envelope object might have message/payload at different levels
          // Try envelope.payload first, then fall back to envelope itself
          const envelopeObj = envelope as { message?: string; payload?: string | { message?: string; payload?: string; debug?: boolean }; debug?: boolean };
          
          // Check if envelope.payload is a string
          if (typeof envelopeObj.payload === 'string') {
            text = envelopeObj.payload;
          }
          // Check if envelope.payload is an object with message or payload
          else if (typeof envelopeObj.payload === 'object' && envelopeObj.payload !== null) {
            const payloadObj = envelopeObj.payload as { message?: string; payload?: string; debug?: boolean };
            text = payloadObj.message || (typeof payloadObj.payload === 'string' ? payloadObj.payload : "");
            isDebug = payloadObj.debug === true;
          }
          // Fall back to envelope.message directly
          else if (typeof envelopeObj.message === 'string') {
            text = envelopeObj.message;
          }
          
          if (envelopeObj.debug === true) {
            isDebug = true;
          }
          
          // Always create a system message - show the raw envelope if we couldn't extract text
          agenticMessages.push({
            type: "system_message",
            subtype: "system.message",
            data: { 
              message: text || `[system event: ${JSON.stringify(envelope)}]`,
              debug: isDebug 
            },
            timestamp: innerTs,
          });
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

  const handleContinue = () => {
    continueMutation.mutate(
      { projectName, parentSessionName: sessionName },
      {
        onSuccess: () => {
          successToast("Session restarted successfully");
        },
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to restart session"),
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
        },
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to send message"),
      }
    );
  };

  const handleInterrupt = () => {
    sendControlMutation.mutate(
      { projectName, sessionName, type: 'interrupt' },
      {
        onSuccess: () => successToast("Agent interrupted"),
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to interrupt agent"),
      }
    );
  };

  const handleEndSession = () => {
    sendControlMutation.mutate(
      { projectName, sessionName, type: 'end_session' },
      {
        onSuccess: () => successToast("Session ended successfully"),
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to end session"),
      }
    );
  };

  const handleSeedWorkflow = useCallback(async () => {
    if (!rfeWorkflowId) return;
    return new Promise<void>((resolve, reject) => {
      seedWorkflowMutation.mutate(
        { projectName, workflowId: rfeWorkflowId },
        {
          onSuccess: () => {
            successToast("Repository seeded successfully");
            refetchSeeding();
            resolve();
          },
          onError: (err) => {
            errorToast(err instanceof Error ? err.message : "Failed to seed repository");
            reject(err);
          },
        }
      );
    });
  }, [projectName, rfeWorkflowId, seedWorkflowMutation, refetchSeeding]);

  const handleUpdateRepositories = useCallback(async (data: { umbrellaRepo: { url: string; branch?: string }; supportingRepos: { url: string; branch?: string }[] }) => {
    if (!rfeWorkflowId) return;
    return new Promise<void>((resolve, reject) => {
      updateWorkflowMutation.mutate(
        {
          projectName,
          workflowId: rfeWorkflowId,
          data: {
            umbrellaRepo: data.umbrellaRepo,
            supportingRepos: data.supportingRepos,
          },
        },
        {
          onSuccess: () => {
            successToast("Repositories updated successfully");
            refetchWorkflow();
            refetchSeeding();
            seedWorkflowMutation.reset();
            resolve();
          },
          onError: (err) => {
            errorToast(err instanceof Error ? err.message : "Failed to update repositories");
            reject(err);
          },
        }
      );
    });
  }, [projectName, rfeWorkflowId, updateWorkflowMutation, refetchWorkflow, refetchSeeding, seedWorkflowMutation]);

  // Seeding status from React Query
  const isSeeded = seedingData?.isSeeded || false;
  const seedingError = seedWorkflowMutation.error?.message || seedingQueryError?.message;
  const hasCheckedSeeding = seedingData !== undefined || !!seedingQueryError;
  const seedingStatus = {
    checking: checkingSeeding,
    isSeeded,
    error: seedingError,
    hasChecked: hasCheckedSeeding,
  };
  const workflowWorkspace = rfeWorkflow?.workspacePath || (rfeWorkflowId ? `/rfe-workflows/${rfeWorkflowId}/workspace` : '');

  // Check if session is completed
  const sessionCompleted = (
    session?.status?.phase === 'Completed' ||
    session?.status?.phase === 'Failed' ||
    session?.status?.phase === 'Stopped'
  );

  // Auto-spawn content pod on completed session
  // Don't auto-retry if we already encountered an error - user must explicitly retry
  useEffect(() => {
    if (sessionCompleted && !contentPodReady && !contentPodSpawning && !contentPodError) {
      spawnContentPodAsync();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionCompleted, contentPodReady, contentPodSpawning, contentPodError]);

  const spawnContentPodAsync = async () => {
    if (!projectName || !sessionName) return;
    
    setContentPodSpawning(true);
    setContentPodError(null); // Clear any previous errors
    
    try {
      // Import API function
      const { spawnContentPod, getContentPodStatus } = await import('@/services/api/sessions');
      
      // Spawn pod
      const spawnResult = await spawnContentPod(projectName, sessionName);
      
      // If already exists and ready, we're done
      if (spawnResult.status === 'exists' && spawnResult.ready) {
        setContentPodReady(true);
        setContentPodSpawning(false);
        setContentPodError(null);
        return;
      }
      
      // Poll for readiness
      let attempts = 0;
      const maxAttempts = 30; // 30 seconds
      
      const pollInterval = setInterval(async () => {
        attempts++;
        
        try {
          const status = await getContentPodStatus(projectName, sessionName);
          
          if (status.ready) {
            clearInterval(pollInterval);
            setContentPodReady(true);
            setContentPodSpawning(false);
            setContentPodError(null);
            successToast('Workspace viewer ready');
          }
          
          if (attempts >= maxAttempts) {
            clearInterval(pollInterval);
            setContentPodSpawning(false);
            const errorMsg = 'Workspace viewer failed to start within 30 seconds';
            setContentPodError(errorMsg);
            errorToast(errorMsg);
          }
        } catch {
          // Not found yet, keep polling
          if (attempts >= maxAttempts) {
            clearInterval(pollInterval);
            setContentPodSpawning(false);
            const errorMsg = 'Workspace viewer failed to start';
            setContentPodError(errorMsg);
            errorToast(errorMsg);
          }
        }
      }, 1000);
      
    } catch (error) {
      setContentPodSpawning(false);
      const errorMsg = error instanceof Error ? error.message : 'Failed to spawn workspace viewer';
      setContentPodError(errorMsg);
      errorToast(errorMsg);
    }
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

  const subagentStats = useMemo(() => {
    const agentCounts: Record<string, number> = {};

    // Parse streamMessages for tool_use_messages with subagent_type
    for (const msg of streamMessages) {
      if (msg.type === 'tool_use_messages') {
        const toolUseBlock = msg.toolUseBlock;

        // Only count Task tool uses (not other tools like Bash, Read, Write)
        if (toolUseBlock?.name !== 'Task') continue;

        // Type-safe extraction with runtime checks
        if (toolUseBlock.input && typeof toolUseBlock.input === 'object') {
          const inputData = toolUseBlock.input as Record<string, unknown>;
          const subagentType = inputData.subagent_type;

          if (typeof subagentType === 'string') {
            agentCounts[subagentType] = (agentCounts[subagentType] || 0) + 1;
          }
        }
      }
    }

    const orderedTypes = Object.keys(agentCounts).sort();

    return {
      uniqueCount: orderedTypes.length,
      orderedTypes,
      counts: agentCounts,
    };
  }, [streamMessages]);

  // Loading state - also check if params are loaded
  if (isLoading || !projectName || !sessionName) {
    return (
      <div className="min-h-screen bg-[#f8fafc]">
        <div className="container mx-auto p-6">
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full" />
            <span className="ml-2">Loading session...</span>
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (error || !session) {
    return (
      <div className="min-h-screen bg-[#f8fafc]">
        <div className="sticky top-0 z-20 bg-white border-b">
          <div className="container mx-auto px-6 py-4">
            <Breadcrumbs
              items={[
                { label: 'Workspaces', href: '/projects' },
                { label: projectName, href: `/projects/${projectName}` },
                { label: 'Sessions', href: `/projects/${projectName}/sessions` },
                { label: 'Error' },
              ]}
              className="mb-4"
            />
            <PageHeader
              title="Session Error"
              description="Unable to load session"
            />
          </div>
        </div>
        <div className="container mx-auto p-0">
          <div className="px-6 pt-6">
            <Card className="border-red-200 bg-red-50">
              <CardContent className="pt-6">
                <p className="text-red-700">Error: {error instanceof Error ? error.message : "Session not found"}</p>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      {rfeWorkflow && (
        <EditRepositoriesDialog
          open={editRepoDialogOpen}
          onOpenChange={setEditRepoDialogOpen}
          workflow={rfeWorkflow}
          onSave={async (data) => {
            await handleUpdateRepositories(data);
            setEditRepoDialogOpen(false);
          }}
          isSaving={updateWorkflowMutation.isPending}
        />
      )}
      <div className="min-h-screen bg-[#f8fafc]">
        {/* Sticky header */}
      <div className="sticky top-0 z-20 bg-white border-b">
        <div className="container mx-auto px-6 py-4">
          <Breadcrumbs
            items={[
              { label: 'Workspaces', href: '/projects' },
              { label: projectName, href: `/projects/${projectName}` },
              { label: 'Sessions', href: `/projects/${projectName}/sessions` },
              { label: session.spec.displayName || session.metadata.name },
            ]}
            className="mb-4"
          />
          <PageHeader
            title={
              <div className="flex items-center gap-2">
                <span>{session.spec.displayName || session.metadata.name}</span>
                <Badge className={getPhaseColor(session.status?.phase || "Pending")}>
                  {session.status?.phase || "Pending"}
                </Badge>
              </div>
            }
            description={
              <div>
                {session.spec.displayName && (
                  <div className="text-sm text-gray-500">{session.metadata.name}</div>
                )}
                <div className="text-xs text-gray-500">
                  Created {formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}
                </div>
              </div>
            }
            actions={
              <>
                {/* Continue button for completed sessions */}
                {(session.status?.phase === "Completed" || session.status?.phase === "Failed" || session.status?.phase === "Stopped") && (
                  <Button
                    onClick={handleContinue}
                    disabled={continueMutation.isPending}
                  >
                    <Play className="w-4 h-4 mr-2" />
                    {continueMutation.isPending ? "Starting..." : "Continue"}
                  </Button>
                )}

                {/* Stop button for active sessions */}
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

                {/* Actions dropdown menu */}
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" size="icon">
                      <MoreVertical className="w-4 h-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <CloneSessionDialog
                      session={session}
                      onSuccess={() => refetchSession()}
                      trigger={
                        <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
                          <Copy className="w-4 h-4 mr-2" />
                          Clone
                        </DropdownMenuItem>
                      }
                    />
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={handleDelete}
                      disabled={deleteMutation.isPending}
                      className="text-red-600"
                    >
                      <Trash2 className="w-4 h-4 mr-2" />
                      {deleteMutation.isPending ? "Deleting..." : "Delete"}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </>
            }
          />
        </div>
      </div>

      <div className="container mx-auto p-0">
        <div className="px-6 pt-6">
        {/* Two Column Layout */}
        <div className="grid grid-cols-1 lg:grid-cols-[40%_1fr] gap-6">
          {/* Left Column - Accordions */}
          <div>
            <Accordion type="multiple" value={openAccordionItems} onValueChange={setOpenAccordionItems} className="w-full space-y-3">
              <AccordionItem value="workflows" className="border rounded-lg px-3 bg-white">
                <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                  Workflows
                </AccordionTrigger>
                <AccordionContent className="pt-2 pb-3">
                  <div className="space-y-3">
                    <div>
                      <label className="text-sm font-medium mb-1.5 block">Select a Workflow</label>
                      <Select value={selectedWorkflow} onValueChange={setSelectedWorkflow}>
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder="None selected" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="none">None selected</SelectItem>
                          <SelectItem value="plan-feature">Plan a feature</SelectItem>
                          <SelectItem value="develop-feature">Develop a feature</SelectItem>
                          <SelectItem value="bug-fix">Bug fix</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <p className="text-sm text-muted-foreground">
                      Workflows provide a structured processes for Ambient Code Platform agents to follow and achieve complex goals.
                    </p>
                  </div>
                </AccordionContent>
              </AccordionItem>

              {/* Only show Spec Repository for plan-feature and develop-feature workflows */}
              {(selectedWorkflow === "plan-feature" || selectedWorkflow === "develop-feature") && (
              <AccordionItem value="spec-repository" className="border rounded-lg px-3 bg-white">
                <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                  Spec Repository
                </AccordionTrigger>
                <AccordionContent className="pt-2 pb-3">
                  {!rfeWorkflowId ? (
                    <div className="text-center py-6">
                      <FolderTree className="h-10 w-10 mx-auto mb-3 opacity-50 text-muted-foreground" />
                      <p className="text-sm text-muted-foreground mb-4">
                        A spec repository is required to store agent config and workflow artifacts.
                      </p>
                      <Button onClick={() => setGithubModalOpen(true)}>
                        Add Spec Repository
                      </Button>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      <div className="text-sm text-muted-foreground">Workspace: {workflowWorkspace}</div>

                      {rfeWorkflow?.branchName && (
                        <Alert className="border-blue-200 bg-blue-50">
                          <GitBranch className="h-4 w-4 text-blue-600" />
                          <AlertTitle className="text-blue-900">Feature Branch</AlertTitle>
                          <AlertDescription className="text-blue-800">
                            All modifications will occur on feature branch{' '}
                            <code className="px-2 py-1 bg-blue-100 text-blue-900 rounded font-semibold">
                              {rfeWorkflow.branchName}
                            </code>
                            {' '}for all supplied repositories.
                          </AlertDescription>
                        </Alert>
                      )}

                      {(rfeWorkflow?.umbrellaRepo || (rfeWorkflow?.supportingRepos || []).length > 0) && (
                        <div className="space-y-1">
                          {rfeWorkflow.umbrellaRepo && (
                            <div className="text-sm space-y-1">
                              <div>
                                <span className="font-medium">Spec Repo:</span> {rfeWorkflow.umbrellaRepo.url}
                              </div>
                              {rfeWorkflow.umbrellaRepo.branch && (
                                <div className="ml-4 text-muted-foreground">
                                  Base branch: <code className="text-xs bg-muted px-1 py-0.5 rounded">{rfeWorkflow.umbrellaRepo.branch}</code>
                                  {rfeWorkflow.branchName && (
                                    <span> → Feature branch <code className="text-xs bg-blue-50 text-blue-700 px-1 py-0.5 rounded">{rfeWorkflow.branchName}</code> {isSeeded ? 'set up' : 'will be set up'}</span>
                                  )}
                                </div>
                              )}
                            </div>
                          )}
                          {(rfeWorkflow.supportingRepos || []).map(
                            (r: { url: string; branch?: string }, i: number) => (
                              <div key={i} className="text-sm space-y-1">
                                <div>
                                  <span className="font-medium">Supporting:</span> {r.url}
                                </div>
                                {r.branch && (
                                  <div className="ml-4 text-muted-foreground">
                                    Base branch: <code className="text-xs bg-muted px-1 py-0.5 rounded">{r.branch}</code>
                                    {rfeWorkflow.branchName && (
                                      <span> → Feature branch <code className="text-xs bg-blue-50 text-blue-700 px-1 py-0.5 rounded">{rfeWorkflow.branchName}</code> {isSeeded ? 'set up' : 'will be set up'}</span>
                                    )}
                                  </div>
                                )}
                              </div>
                            )
                          )}
                        </div>
                      )}

                      {!isSeeded && !seedingStatus.checking && seedingStatus.hasChecked && rfeWorkflow?.umbrellaRepo && (
                        <Alert variant="destructive">
                          <AlertCircle className="h-4 w-4" />
                          <AlertTitle>Spec Repository Not Seeded</AlertTitle>
                          <AlertDescription className="mt-2">
                            <p className="mb-3">
                              Before you can start working on phases, the spec repository needs to be seeded.
                              This will:
                            </p>
                            <ul className="list-disc list-inside space-y-1 mb-3 text-sm">
                              <li>Set up the feature branch{rfeWorkflow.branchName && ` (${rfeWorkflow.branchName})`} from the base branch</li>
                              <li>Add Spec-Kit template files for spec-driven development</li>
                              <li>Add agent definition files in the .claude directory</li>
                            </ul>
                            {seedingError && (
                              <div className="mb-3 p-2 bg-red-100 border border-red-300 rounded text-sm text-red-800">
                                <strong>Seeding Failed:</strong> {seedingError}
                              </div>
                            )}
                            <div className="flex gap-2">
                              <Button
                                onClick={() => setEditRepoDialogOpen(true)}
                                disabled={updateWorkflowMutation.isPending}
                                size="sm"
                                variant="outline"
                              >
                                <Edit className="mr-2 h-4 w-4" />
                                Edit Repositories
                              </Button>
                              <Button onClick={handleSeedWorkflow} disabled={seedWorkflowMutation.isPending || updateWorkflowMutation.isPending} size="sm">
                                {seedWorkflowMutation.isPending ? (
                                  <>
                                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                    Seeding Repository...
                                  </>
                                ) : (
                                  <>
                                    <Sprout className="mr-2 h-4 w-4" />
                                    {seedingError ? "Retry Seeding" : "Seed Repository"}
                                  </>
                                )}
                              </Button>
                            </div>
                          </AlertDescription>
                        </Alert>
                      )}

                      {seedingStatus.checking && rfeWorkflow?.umbrellaRepo && (
                        <div className="flex items-center gap-2 text-gray-600 bg-gray-50 p-3 rounded-lg">
                          <Loader2 className="h-5 w-5 animate-spin" />
                          <span className="text-sm">Checking repository seeding status...</span>
                        </div>
                      )}

                      {isSeeded && (
                        <div className="flex items-center justify-between text-green-700 bg-green-50 p-3 rounded-lg">
                          <div className="flex items-center gap-2">
                            <CheckCircle2 className="h-5 w-5 text-green-600" />
                            <span className="text-sm font-medium">Repository seeded and ready</span>
                          </div>
                          <Button
                            onClick={() => setEditRepoDialogOpen(true)}
                            disabled={updateWorkflowMutation.isPending}
                            size="sm"
                            variant="outline"
                          >
                            <Edit className="mr-2 h-4 w-4" />
                            Edit Repositories
                          </Button>
                        </div>
                      )}
                    </div>
                  )}

                  {/* Workspace Content - Always visible for all sessions */}
                  <div className="mt-4 pt-4 border-t">
                    {sessionCompleted && !contentPodReady ? (
                      <Card className="p-8">
                        <div className="text-center space-y-4">
                          {contentPodSpawning ? (
                            <>
                              <div className="flex items-center justify-center">
                                <div className="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full" />
                              </div>
                              <p className="text-sm font-medium">Starting workspace viewer...</p>
                              <p className="text-xs text-gray-500">This may take up to 30 seconds</p>
                            </>
                          ) : (
                            <>
                              <p className="text-sm text-gray-600">
                                Session has completed. To view and edit your workspace files, please start a workspace viewer.
                              </p>
                              <Button onClick={spawnContentPodAsync}>
                                Start Workspace Viewer
                              </Button>
                            </>
                          )}
                        </div>
                      </Card>
                    ) : (
                      <WorkspaceTab
                        session={session}
                        wsLoading={wsLoading}
                        wsUnavailable={wsUnavailable}
                        wsTree={wsTree}
                        wsSelectedPath={wsSelectedPath}
                        wsFileContent={wsFileContent}
                        onRefresh={handleRefreshWorkspace}
                        onSelect={onWsSelect}
                        onToggle={onWsToggle}
                        onSave={writeWsFile}
                        setWsFileContent={setWsFileContent}
                        k8sResources={k8sResources}
                        contentPodError={contentPodError}
                        onRetrySpawn={spawnContentPodAsync}
                      />
                    )}
                  </div>
                </AccordionContent>
              </AccordionItem>
              )}

              <AccordionItem value="agents" className="border rounded-lg px-3 bg-white">
                <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                  Agents
                </AccordionTrigger>
                <AccordionContent className="pt-2 pb-3">
                  {loadingAgents ? (
                    <div className="flex items-center justify-center py-6">
                      <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    </div>
                  ) : !rfeWorkflowId || repoAgents.length === 0 ? (
                    <div className="text-center py-6 text-muted-foreground">
                      <Bot className="h-10 w-10 mx-auto mb-2 opacity-50" />
                      <p className="text-sm">No agents found in repository .claude/agents directory</p>
                      <p className="text-xs mt-1">Seed the repository to add agent definitions</p>
                    </div>
                  ) : (
                    <>
                      <div className="grid grid-cols-1 gap-2">
                        {repoAgents.map((agent) => {
                          const isSelected = selectedAgents.includes(agent.persona);
                          return (
                            <div
                              key={agent.persona}
                              className={`p-2 rounded border transition-colors ${
                                isSelected ? 'bg-primary/5 border-primary' : 'bg-background border-border hover:border-primary/50'
                              }`}
                            >
                              <label className="flex items-start gap-2 cursor-pointer">
                                <Checkbox
                                  checked={isSelected}
                                  onCheckedChange={(checked) => {
                                    setSelectedAgents(
                                      checked
                                        ? [...selectedAgents, agent.persona]
                                        : selectedAgents.filter(p => p !== agent.persona)
                                    );
                                  }}
                                  className="mt-0.5"
                                />
                                <div className="flex-1 min-w-0">
                                  <div className="font-medium text-sm">{agent.name}</div>
                                  <div className="text-xs text-muted-foreground">{agent.role}</div>
                                </div>
                              </label>
                            </div>
                          );
                        })}
                      </div>
                      {selectedAgents.length > 0 && (
                        <div className="mt-3 pt-3 border-t">
                          <div className="text-sm font-medium mb-1.5">Selected Agents ({selectedAgents.length})</div>
                          <div className="flex flex-wrap gap-1.5">
                            {selectedAgents.map(persona => {
                              const agent = repoAgents.find(a => a.persona === persona);
                              return agent ? (
                                <Badge key={persona} variant="secondary" className="text-xs">
                                  {agent.name}
                                </Badge>
                              ) : null;
                            })}
                          </div>
                        </div>
                      )}
                    </>
                  )}
                </AccordionContent>
              </AccordionItem>

              <AccordionItem value="overview" className="border rounded-lg px-3 bg-white">
                <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                  Context
                </AccordionTrigger>
                <AccordionContent className="space-y-4 pt-2 pb-3">
                  <OverviewTab
                    session={session}
                    promptExpanded={promptExpanded}
                    setPromptExpanded={setPromptExpanded}
                    latestLiveMessage={latestLiveMessage as SessionMessage | null}
                    diffTotals={diffTotals}
                    k8sResources={k8sResources}
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
                    onRefreshDiff={handleRefreshDiff}
                  />
                </AccordionContent>
              </AccordionItem>

              <AccordionItem value="results" className="border rounded-lg px-3 bg-white">
                <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                  Results
                </AccordionTrigger>
                <AccordionContent className="pt-2 pb-3">
                  <ResultsTab result={null} meta={null} />
                </AccordionContent>
              </AccordionItem>

              <AccordionItem value="session-details" className="border rounded-lg px-3 bg-white">
                <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                  Session Details
                </AccordionTrigger>
                <AccordionContent className="pt-2 pb-3">
                  <div className="grid grid-cols-3 gap-3">
                    <Card className="py-2">
                      <CardContent className="p-2">
                        <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Duration</div>
                        <div className="text-sm font-semibold">{typeof durationMs === "number" ? `${durationMs} ms` : "-"}</div>
                      </CardContent>
                    </Card>
                    <Card className="py-2">
                      <CardContent className="p-2">
                        <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Messages</div>
                        <div className="text-sm font-semibold">{messages.length}</div>
                      </CardContent>
                    </Card>
                    <Card className="py-2">
                      <CardContent className="p-2">
                        <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Agents</div>
                        <div className="text-sm font-semibold">{subagentStats.uniqueCount > 0 ? subagentStats.uniqueCount : "-"}</div>
                      </CardContent>
                    </Card>
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </div>

          {/* Right Column - Messages (Always Visible) */}
          <div>
            <Card>
              <CardContent className="p-3">
                <MessagesTab
                  session={session}
                  streamMessages={streamMessages}
                  chatInput={chatInput}
                  setChatInput={setChatInput}
                  onSendChat={() => Promise.resolve(sendChat())}
                  onInterrupt={() => Promise.resolve(handleInterrupt())}
                  onEndSession={() => Promise.resolve(handleEndSession())}
                  onGoToResults={() => {}}
                  onContinue={handleContinue}
                />
              </CardContent>
            </Card>
          </div>
        </div>
        </div>
      </div>
    </div>

    {/* Add Spec Repository Modal */}
    <Dialog open={githubModalOpen} onOpenChange={setGithubModalOpen}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Add spec repository</DialogTitle>
          <DialogDescription>
            Set the spec repo and optional supporting repos. Base branch is the branch from which the feature branch will be set up. All modifications will be made to the feature branch.
          </DialogDescription>
        </DialogHeader>
        
        {!githubStatus?.installed && (
          <div className="mb-4">
            <GitHubConnectionCard appSlug="ambient-code-vteam" showManageButton={false} />
          </div>
        )}

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="spec-repo-url">Spec Repo URL</Label>
            <Input
              id="spec-repo-url"
              placeholder="https://github.com/org/repo.git"
              value={specRepoUrl}
              onChange={(e) => setSpecRepoUrl(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              The spec repository contains your feature specifications, planning documents, and agent configurations for this RFE workspace
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="base-branch">Base Branch</Label>
            <Input
              id="base-branch"
              placeholder="main"
              value={baseBranch}
              onChange={(e) => setBaseBranch(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label>Supporting Repositories (optional)</Label>
            <Button variant="outline" className="w-full">
              Add supporting repo
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
    </>
  );
}

