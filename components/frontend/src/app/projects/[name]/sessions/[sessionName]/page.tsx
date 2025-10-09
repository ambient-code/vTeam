"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import {
  ArrowLeft,
  RefreshCw,
  
  Square,
  Trash2,
  Copy,
} from "lucide-react";

// Custom components
import OverviewTab from "@/components/session/OverviewTab";
import MessagesTab from "@/components/session/MessagesTab";
import WorkspaceTab from "@/components/session/WorkspaceTab";
import ResultsTab from "@/components/session/ResultsTab";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  AgenticSession,
  AgenticSessionPhase,
} from "@/types/agentic-session";
import { CloneSessionDialog } from "@/components/clone-session-dialog";
import type { FileTreeNode } from "@/components/file-tree";

import { getApiUrl } from "@/lib/config";
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
      const innerType: string = (raw as unknown as InnerEnvelope)?.type || envelope.type || "";
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

  // Derive the most recent final result text from live messages (fallback for interactive sessions)
  const latestResultText = useMemo(() => {
    const unwrapPayload = (obj: any): any => {
      let cur = obj;
      let guard = 0;
      while (cur && typeof cur === 'object' && 'payload' in cur && (cur as any).payload && guard < 5) {
        cur = (cur as any).payload;
        guard++;
      }
      return cur;
    };

    for (let i = liveMessages.length - 1; i >= 0; i--) {
      const raw: any = liveMessages[i] as any;
      const envelope: any = (raw?.payload ?? raw) || {};
      const innerType: string = envelope?.type || (raw as any)?.type || "";
      const innerPayload: any = (envelope && typeof envelope === 'object' && 'payload' in envelope) ? (envelope as any).payload : null;

      if (innerType === 'agent.message') {
        const unwrapped = unwrapPayload(innerPayload);
        const isResultMessage = (innerPayload && (innerPayload as any).type === 'result.message') || (unwrapped && typeof (unwrapped as any).result === 'string');
        if (isResultMessage) {
          const result = typeof (unwrapped as any)?.result === 'string' ? (unwrapped as any).result : null;
          if (result && result.trim()) return result;
        }
      }

      if (innerType === 'result.message') {
        const unwrapped = unwrapPayload(innerPayload ?? envelope);
        const result = typeof (unwrapped as any)?.result === 'string' ? (unwrapped as any).result : null;
        if (result && result.trim()) return result;
      }
    }
    return null as string | null;
  }, [liveMessages]);

  // Prefer status.result if it is a non-empty string; otherwise use the latest result.message text
  const resultForResultsTab = useMemo(() => {
    const s = session?.status?.result;
    if (typeof s === 'string' && s.trim()) return s;
    return latestResultText || null;
  }, [session?.status?.result, latestResultText]);

  type ResultMeta = {
    subtype?: string;
    duration_ms?: number;
    duration_api_ms?: number;
    is_error?: boolean;
    num_turns?: number;
    session_id?: string;
    total_cost_usd?: number | null;
    usage?: Record<string, unknown> | null;
  };

  const latestResultMeta: ResultMeta | null = useMemo(() => {
    const unwrapPayload = (obj: any): any => {
      let cur = obj;
      let guard = 0;
      while (cur && typeof cur === 'object' && 'payload' in cur && (cur as any).payload && guard < 5) {
        cur = (cur as any).payload;
        guard++;
      }
      return cur;
    };
    for (let i = liveMessages.length - 1; i >= 0; i--) {
      const raw: any = liveMessages[i] as any;
      const envelope: any = (raw?.payload ?? raw) || {};
      const innerType: string = envelope?.type || (raw as any)?.type || "";
      const innerPayload: any = (envelope && typeof envelope === 'object' && 'payload' in envelope) ? (envelope as any).payload : null;
      if (innerType === 'agent.message') {
        const unwrapped = unwrapPayload(innerPayload);
        const data = (typeof unwrapped === 'object' && unwrapped) ? unwrapped as any : null;
        if (data && typeof data.result === 'string') {
          return {
            subtype: typeof data.subtype === 'string' ? data.subtype : undefined,
            duration_ms: typeof data.duration_ms === 'number' ? data.duration_ms : undefined,
            duration_api_ms: typeof data.duration_api_ms === 'number' ? data.duration_api_ms : undefined,
            is_error: Boolean(data.is_error),
            num_turns: typeof data.num_turns === 'number' ? data.num_turns : undefined,
            session_id: typeof data.session_id === 'string' ? data.session_id : undefined,
            total_cost_usd: typeof data.total_cost_usd === 'number' ? data.total_cost_usd : null,
            usage: (typeof data.usage === 'object' && data.usage) ? data.usage as Record<string, unknown> : null,
          } as ResultMeta;
        }
      }
      if (innerType === 'result.message') {
        const data = unwrapPayload(innerPayload ?? envelope);
        if (data && typeof (data as any).result === 'string') {
          const d: any = data;
          return {
            subtype: typeof d.subtype === 'string' ? d.subtype : undefined,
            duration_ms: typeof d.duration_ms === 'number' ? d.duration_ms : undefined,
            duration_api_ms: typeof d.duration_api_ms === 'number' ? d.duration_api_ms : undefined,
            is_error: Boolean(d.is_error),
            num_turns: typeof d.num_turns === 'number' ? d.num_turns : undefined,
            session_id: typeof d.session_id === 'string' ? d.session_id : undefined,
            total_cost_usd: typeof d.total_cost_usd === 'number' ? d.total_cost_usd : null,
            usage: (typeof d.usage === 'object' && d.usage) ? d.usage as Record<string, unknown> : null,
          } as ResultMeta;
        }
      }
    }
    // Fallback to status fields if present
    const st = session?.status;
    if (st) {
      return {
        subtype: st.subtype,
        duration_ms: undefined,
        duration_api_ms: undefined,
        is_error: st.is_error,
        num_turns: st.num_turns,
        session_id: st.session_id,
        total_cost_usd: st.total_cost_usd ?? null,
        usage: st.usage ?? null,
      } as ResultMeta;
    }
    return null;
  }, [liveMessages, session?.status]);



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
      // Only update if content changed to avoid rerender flicker
      setSession((prev) => {
        try {
          const a = JSON.stringify(prev);
          const b = JSON.stringify(data);
          return a === b ? prev : data;
        } catch {
          return data;
        }
      });


    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [projectName, sessionName]);

  // Workspace (PVC) browser state
  const [wsTree, setWsTree] = useState<FileTreeNode[]>([]);
  const [wsSelectedPath, setWsSelectedPath] = useState<string | undefined>(undefined);
  const [wsFileContent, setWsFileContent] = useState<string>("");
  const [wsLoading, setWsLoading] = useState<boolean>(false);
  const [wsUnavailable, setWsUnavailable] = useState<boolean>(false);
  // Simple exponential backoff for workspace polling when content service is not reachable
  const wsErrCountRef = useRef<number>(0);
  const wsBackoffUntilRef = useRef<number>(0);
  // Keep a ref of the latest tree to avoid including wsTree in callback deps (prevents effect thrash)
  const wsTreeRef = useRef<FileTreeNode[]>([]);
  useEffect(() => { wsTreeRef.current = wsTree; }, [wsTree]);

  type ListItem = { name: string; path: string; isDir: boolean; size: number; modifiedAt: string };
  const listWsPath = useCallback(async (relPath?: string) => {
    const now = Date.now();
    if (wsBackoffUntilRef.current && now < wsBackoffUntilRef.current) {
      return [] as ListItem[];
    }
    try {
      const url = new URL(`${getApiUrl()}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace`, window.location.origin);
      if (relPath) url.searchParams.set("path", relPath);
      const resp = await fetch(url.toString());
      if (!resp.ok) {
        // Treat service errors as unavailable; caller will render empty state
        setWsUnavailable(true);
        // Backoff on errors
        wsErrCountRef.current = Math.min(wsErrCountRef.current + 1, 8);
        const delayMs = Math.min(60000, Math.pow(2, Math.max(0, wsErrCountRef.current - 1)) * 2000);
        wsBackoffUntilRef.current = Date.now() + delayMs;
        return [] as ListItem[];
      }
      const data = await resp.json();
      const items: ListItem[] = Array.isArray(data.items) ? data.items : [];
      setWsUnavailable(false);
      // Reset backoff on success
      wsErrCountRef.current = 0;
      wsBackoffUntilRef.current = 0;
      return items;
    } catch {
      setWsUnavailable(true);
      wsErrCountRef.current = Math.min(wsErrCountRef.current + 1, 8);
      const delayMs = Math.min(60000, Math.pow(2, Math.max(0, wsErrCountRef.current - 1)) * 2000);
      wsBackoffUntilRef.current = Date.now() + delayMs;
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

  // Preserve expansion state across refreshes to avoid flicker
  const collectExpanded = useCallback((nodes: FileTreeNode[], base = ""): Record<string, boolean> => {
    const map: Record<string, boolean> = {};
    for (const n of nodes) {
      const p = base ? `${base}/${n.name}` : n.path || n.name;
      if (n.type === 'folder') {
        map[p] = !!n.expanded;
        if (n.children && n.children.length) {
          Object.assign(map, collectExpanded(n.children, p));
        }
      }
    }
    return map;
  }, []);

  const buildWsRoot = useCallback(async (background = false) => {
    if (!background) setWsLoading(true);
    try {
      const prevExpanded = collectExpanded(wsTreeRef.current);
      const items = await listWsPath();
      // Strip the backend's /sessions/<sessionName>/workspace/ prefix from paths
      const prefix = `/sessions/${sessionName}/workspace/`;
      const children: FileTreeNode[] = items.map((it) => {
        const displayPath = it.path.startsWith(prefix) ? it.path.slice(prefix.length) : it.name;
        const nodePathKey = displayPath;
        return {
          name: it.name,
          path: displayPath,
          type: it.isDir ? "folder" : "file",
          // Keep previous expansion state if present
          expanded: it.isDir ? (prevExpanded[nodePathKey] ?? false) : false,
          sizeKb: it.isDir ? undefined : it.size / 1024,
        } as FileTreeNode;
      });
      // Only update state if changed length or names differ to avoid unnecessary re-renders
      const currentTree = wsTreeRef.current;
      const sameLength = currentTree.length === children.length;
      const sameNames = sameLength && currentTree.every((n, i) => n.name === children[i].name && n.type === children[i].type);
      if (!sameLength || !sameNames) setWsTree(children);
    } finally {
      if (!background) setWsLoading(false);
    }
  }, [listWsPath, sessionName, collectExpanded]);

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

  // Derive repo folder name in workspace from a Git URL (last path segment, no .git)
  const deriveRepoFolderFromUrl = useCallback((url: string): string => {
    try {
      // Handle SSH urls like git@github.com:owner/repo.git and https
      const cleaned = url.replace(/^git@([^:]+):/, "https://$1/")
      const u = new URL(cleaned)
      const segs = u.pathname.split('/').filter(Boolean)
      const last = segs[segs.length - 1] || "repo"
      return last.replace(/\.git$/i, "")
    } catch {
      const parts = url.split('/')
      const last = parts[parts.length - 1] || "repo"
      return last.replace(/\.git$/i, "")
    }
  }, [])

  // Fetch functions for polling
  const fetchMessages = useCallback(async () => {
    if (!projectName || !sessionName) return;
    try {
      const apiUrl = getApiUrl();
      const resp = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`);
      if (resp.ok) {
        const data = await resp.json();
        const newMsgs = Array.isArray(data.messages) ? data.messages : [];
        setLiveMessages((prev) => {
          // Skip update only if messages are actually identical
          // (timestamps alone are insufficient since backend uses second-precision RFC3339)
          if (prev.length === newMsgs.length && prev.length > 0) {
            try {
              // Deep comparison of message content to catch same-second updates
              const prevJson = JSON.stringify(prev);
              const newJson = JSON.stringify(newMsgs);
              if (prevJson === newJson) {
                return prev;  // Truly identical, skip update
              }
            } catch {
              // JSON.stringify failed, update to be safe
            }
          }
          return newMsgs;
        });
      }
    } catch {}
  }, [projectName, sessionName]);

  const fetchWorkspace = useCallback(async () => {
    if (!projectName || !sessionName || activeTab !== 'workspace') return;
    await buildWsRoot(true);
  }, [projectName, sessionName, activeTab, buildWsRoot]);

  const fetchRepoDiffs = useCallback(async () => {
    if (!projectName || !sessionName || !session?.spec?.repos || !Array.isArray(session.spec.repos)) return;
    try {
      const apiUrl = getApiUrl();
      const repos = session.spec.repos as any[];
      const counts = await Promise.all(repos.map(async (r: any, idx: number) => {
        const url = (r?.input?.url as string) || "";
        if (!url) return { added: 0, modified: 0, deleted: 0, renamed: 0, untracked: 0 };
        const folder = deriveRepoFolderFromUrl(url);
        const qs = new URLSearchParams({ repoIndex: String(idx), repoPath: `/sessions/${sessionName}/workspace/${folder}` });
        const resp = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/github/diff?${qs.toString()}`);
        if (!resp.ok) return { added: 0, modified: 0, deleted: 0, renamed: 0, untracked: 0 };
        const data = await resp.json();
        return { added: Number(data.added||0), modified: Number(data.modified||0), deleted: Number(data.deleted||0), renamed: Number(data.renamed||0), untracked: Number(data.untracked||0) };
      }));
      const nextTotals: Record<number, { added: number; modified: number; deleted: number; renamed: number; untracked: number }> = {};
      counts.forEach((t, i) => { nextTotals[i] = t as any });
      setDiffTotals(nextTotals);
    } catch {}
  }, [projectName, sessionName, session?.spec?.repos, deriveRepoFolderFromUrl]);

  const sendChat = useCallback(async () => {
    if (!chatInput.trim() || !projectName || !sessionName) return;
    try {
      const apiUrl = getApiUrl();
      await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: chatInput.trim() })
      });
      setChatInput("");
      await fetchSession();
      await fetchMessages();
      setActiveTab('messages');
    } catch {}
  }, [chatInput, projectName, sessionName, fetchSession, fetchMessages]);

  // Single unified polling effect
  useEffect(() => {
    if (!projectName || !sessionName) return;
    
    // Initial fetch
    fetchSession();
    fetchMessages();
    fetchWorkspace();
    fetchRepoDiffs();

    // Poll for updates every 3 seconds while session is active
    const interval = setInterval(() => {
      const isActive = session?.status?.phase === "Pending" ||
                       session?.status?.phase === "Running" ||
                       session?.status?.phase === "Creating";
      
      if (isActive) {
        fetchSession();
        fetchMessages();
        fetchWorkspace();
        fetchRepoDiffs();
      }
    }, 3000);

    return () => clearInterval(interval);
  }, [projectName, sessionName, session?.status?.phase, fetchSession, fetchMessages, fetchWorkspace, fetchRepoDiffs]);

 
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

  // Track per-repo diff breakdown and action states
  const [diffTotals, setDiffTotals] = useState<Record<number, { added: number; modified: number; deleted: number; renamed: number; untracked: number }>>({})
  const [busyRepo, setBusyRepo] = useState<Record<number, 'push' | 'abandon' | null>>({})

  const buildGithubCompareUrl = useCallback((inputUrl: string, inputBranch?: string, outputUrl?: string, outputBranch?: string): string | null => {
    if (!inputUrl || !outputUrl) return null
    const parseOwner = (url: string): { owner: string; repo: string } | null => {
      try {
        const cleaned = url.replace(/^git@([^:]+):/, "https://$1/")
        const u = new URL(cleaned)
        const segs = u.pathname.split('/').filter(Boolean)
        if (segs.length >= 2) return { owner: segs[segs.length-2], repo: segs[segs.length-1].replace(/\.git$/i, "") }
        return null
      } catch { return null }
    }
    const inOrg = parseOwner(inputUrl)
    const outOrg = parseOwner(outputUrl)
    if (!inOrg || !outOrg) return null
    const base = inputBranch && inputBranch.trim() ? inputBranch : 'main'
    const head = outputBranch && outputBranch.trim() ? outputBranch : null
    if (!head) return null
    return `https://github.com/${inOrg.owner}/${inOrg.repo}/compare/${encodeURIComponent(base)}...${encodeURIComponent(outOrg.owner + ':' + head)}`
  }, [])


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
            <TabsTrigger value="results">Results</TabsTrigger>
          </TabsList>

          {/* Overview */}
          <TabsContent value="overview" className="space-y-6">
            <OverviewTab
              session={session}
              promptExpanded={promptExpanded}
              setPromptExpanded={setPromptExpanded}
              latestLiveMessage={latestLiveMessage}
              subagentStats={{ uniqueCount: 0, orderedTypes: [] }}
              diffTotals={diffTotals}
              onPush={async (idx) => {
                try {
                  setBusyRepo((b) => ({ ...b, [idx]: 'push' }));
                                            const apiUrl = getApiUrl();
                  const repo = session.spec.repos?.[idx];
                  if (!repo) return;
                  const folder = deriveRepoFolderFromUrl(repo.input.url);
                  const resp = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/github/push`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ repoIndex: idx, repoPath: `/sessions/${sessionName}/workspace/${folder}` }) });
                  if (resp.ok) setDiffTotals((m) => ({ ...m, [idx]: { added: 0, modified: 0, deleted: 0, renamed: 0, untracked: 0 } }));
                                            await fetchSession();
                } catch {} finally { setBusyRepo((b) => ({ ...b, [idx]: null })); }
              }}
              onAbandon={async (idx) => {
                try {
                  setBusyRepo((b) => ({ ...b, [idx]: 'abandon' }));
                                            const apiUrl = getApiUrl();
                  const repo = session.spec.repos?.[idx];
                  if (!repo) return;
                  const folder = deriveRepoFolderFromUrl(repo.input.url);
                  const resp = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/github/abandon`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ repoIndex: idx, repoPath: `/sessions/${sessionName}/workspace/${folder}` }) });
                  if (resp.ok) setDiffTotals((m) => ({ ...m, [idx]: { added: 0, modified: 0, deleted: 0, renamed: 0, untracked: 0 } }));
                                            await fetchSession();
                } catch {} finally { setBusyRepo((b) => ({ ...b, [idx]: null })); }
              }}
              busyRepo={busyRepo}
              buildGithubCompareUrl={buildGithubCompareUrl}
            />
          </TabsContent>

          {/* Messages */}
          <TabsContent value="messages">
            <MessagesTab
              session={session}
              streamMessages={streamMessages}
              chatInput={chatInput}
              setChatInput={setChatInput}
              onSendChat={sendChat}
              onInterrupt={async () => { try { const apiUrl = getApiUrl(); await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ type: 'interrupt' }) }) } catch {} }}
              onEndSession={async () => { try { const apiUrl = getApiUrl(); await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ type: 'end_session' }) }) } catch {} }}
              onGoToResults={() => setActiveTab('results')}
            />
          </TabsContent>

          {/* Workspace (PVC) */}
          <TabsContent value="workspace">
            <WorkspaceTab
              session={session}
              wsLoading={wsLoading}
              wsUnavailable={wsUnavailable}
              wsTree={wsTree}
              wsSelectedPath={wsSelectedPath}
              wsFileContent={wsFileContent}
              onRefresh={(bg) => void buildWsRoot(Boolean(bg))}
              onSelect={onWsSelect}
              onToggle={onWsToggle}
              onSave={writeWsFile}
              setWsFileContent={setWsFileContent}
            />
          </TabsContent>

          {/* Results */}
          <TabsContent value="results">
            <ResultsTab result={resultForResultsTab} meta={latestResultMeta} />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
