"use client";

import { useState, useEffect, useMemo, useCallback, useRef } from "react";
import { Play, Loader2, FolderTree, AlertCircle, GitBranch, Edit, RefreshCw, Folder, Sparkles, X, CloudUpload, CloudDownload, MoreVertical, Link, Cloud, FolderSync, Download, Workflow, ChevronDown, ChevronRight } from "lucide-react";
import { useRouter } from "next/navigation";

// Custom components
import MessagesTab from "@/components/session/MessagesTab";
import { FileTree, type FileTreeNode } from "@/components/file-tree";
import { EmptyState } from "@/components/empty-state";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from "@/components/ui/dropdown-menu";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { PageHeader } from "@/components/page-header";
import { SessionHeader } from "./session-header";

import type { SessionMessage } from "@/types";
import type { MessageObject, ToolUseMessages, ToolUseBlock, ToolResultBlock } from "@/types/agentic-session";

// React Query hooks
import {
  useSession,
  useSessionMessages,
  useStopSession,
  useDeleteSession,
  useSendChatMessage,
  useSendControlMessage,
  useSessionK8sResources,
  useContinueSession,
} from "@/services/queries";
import { useWorkspaceList, useGitMergeStatus, useGitPull, useGitPush, useGitListBranches } from "@/services/queries/use-workspace";
import { successToast, errorToast } from "@/hooks/use-toast";
import { useOOTBWorkflows, useWorkflowMetadata } from "@/services/queries/use-workflows";
import { useMutation } from "@tanstack/react-query";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

export default function ProjectSessionDetailPage({
  params,
}: {
  params: Promise<{ name: string; sessionName: string }>;
}) {
  const router = useRouter();
  const [projectName, setProjectName] = useState<string>("");
  const [sessionName, setSessionName] = useState<string>("");
  const [chatInput, setChatInput] = useState("");
  const [backHref, setBackHref] = useState<string | null>(null);
  const [contentPodSpawning, setContentPodSpawning] = useState(false);
  const [contentPodReady, setContentPodReady] = useState(false);
  const [contentPodError, setContentPodError] = useState<string | null>(null);
  const [selectedAgents, setSelectedAgents] = useState<string[]>([]);
  const [selectedWorkflow, setSelectedWorkflow] = useState<string>("none");
  const [openAccordionItems, setOpenAccordionItems] = useState<string[]>(["workflows"]);
  const [contextModalOpen, setContextModalOpen] = useState(false);
  const [contextUrl, setContextUrl] = useState("");
  const [contextBranch, setContextBranch] = useState("main");
  const [commandsScrollTop, setCommandsScrollTop] = useState(false);
  const [commandsScrollBottom, setCommandsScrollBottom] = useState(true);
  const [customWorkflowDialogOpen, setCustomWorkflowDialogOpen] = useState(false);
  const [customWorkflowUrl, setCustomWorkflowUrl] = useState("");
  const [customWorkflowBranch, setCustomWorkflowBranch] = useState("main");
  const [customWorkflowPath, setCustomWorkflowPath] = useState("");
  const [pendingWorkflow, setPendingWorkflow] = useState<{ id: string; name: string; description: string; gitUrl: string; branch: string; path?: string; enabled: boolean } | null>(null);
  const [activeWorkflow, setActiveWorkflow] = useState<string | null>(null);
  const [workflowActivating, setWorkflowActivating] = useState(false);
  const [repoChanging, setRepoChanging] = useState(false);
  const [autoSelectAgents, setAutoSelectAgents] = useState(true);
  const [showAgentsList, setShowAgentsList] = useState(false);
  const [showCommandsList, setShowCommandsList] = useState(false);
  
  // Directory browser state (unified for artifacts, repos, and workflow)
  const [selectedDirectory, setSelectedDirectory] = useState<{
    type: 'artifacts' | 'repo' | 'workflow';
    name: string;
    path: string;
  }>({
    type: 'artifacts',
    name: 'Shared Artifacts',
    path: 'artifacts'
  });
  const [directoryRemotes, setDirectoryRemotes] = useState<Record<string, {url: string; branch: string}>>({});
  const [remoteDialogOpen, setRemoteDialogOpen] = useState(false);
  const [remoteUrl, setRemoteUrl] = useState("");
  const [remoteBranch, setRemoteBranch] = useState("main");
  const [synchronizing, setSynchronizing] = useState(false);
  const [newBranchName, setNewBranchName] = useState("");
  const [showCreateBranch, setShowCreateBranch] = useState(false);
  const [currentSubPath, setCurrentSubPath] = useState<string>("");
  const [inlineViewingFile, setInlineViewingFile] = useState<{path: string; content: string} | null>(null);
  const [commitModalOpen, setCommitModalOpen] = useState(false);
  const [commitMessage, setCommitMessage] = useState("");
  const [committing, setCommitting] = useState(false);
  const [loadingInlineFile, setLoadingInlineFile] = useState(false);
  const [gitStatus, setGitStatus] = useState<{
    initialized: boolean;
    hasChanges: boolean;
    uncommittedFiles: number;
    filesAdded: number;
    filesRemoved: number;
    totalAdded: number;
    totalRemoved: number;
  } | null>(null);

  // Extract params
  useEffect(() => {
    params.then(({ name, sessionName: sName }) => {
      setProjectName(name);
      setSessionName(sName);
      try {
        const url = new URL(window.location.href);
        setBackHref(url.searchParams.get("backHref"));
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
  const stopMutation = useStopSession();
  const deleteMutation = useDeleteSession();
  const continueMutation = useContinueSession();
  const sendChatMutation = useSendChatMessage();
  const sendControlMutation = useSendControlMessage();
  
  // Repo management mutations
  const addRepoMutation = useMutation({
    mutationFn: async (repo: { url: string; branch: string; output?: { url: string; branch: string } }) => {
      setRepoChanging(true);
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/repos`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(repo),
        }
      );
      if (!response.ok) throw new Error('Failed to add repository');
      const result = await response.json();
      return { ...result, inputRepo: repo }; // Include original repo for remote setup
    },
    onSuccess: async (data) => {
      successToast('Repository cloning...');
      // Wait for clone and restart to complete
      await new Promise(resolve => setTimeout(resolve, 3000));
      await refetchSession();
      
      // Auto-configure and persist remote for the new repo
      if (data.name && data.inputRepo) {
        try {
          // Call backend to persist the remote to annotations
          await fetch(
            `/api/projects/${projectName}/agentic-sessions/${sessionName}/git/configure-remote`,
            {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                path: data.name,
                remoteUrl: data.inputRepo.url,
                branch: data.inputRepo.branch || 'main',
              }),
            }
          );
          
          // Update local state
          const newRemotes = {...directoryRemotes};
          newRemotes[data.name] = {
            url: data.inputRepo.url,
            branch: data.inputRepo.branch || 'main'
          };
          setDirectoryRemotes(newRemotes);
        } catch (err) {
          console.error('Failed to configure remote:', err);
          // Non-fatal - repo still works, just no remote configured
        }
      }
      
      setRepoChanging(false);
      successToast('Repository added successfully');
    },
    onError: (error: Error) => {
      setRepoChanging(false);
      errorToast(error.message || 'Failed to add repository');
    },
  });

  const removeRepoMutation = useMutation({
    mutationFn: async (repoName: string) => {
      setRepoChanging(true);
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/repos/${repoName}`,
        { method: 'DELETE' }
      );
      if (!response.ok) throw new Error('Failed to remove repository');
      return response.json();
    },
    onSuccess: async () => {
      successToast('Repository removing...');
      // Wait for removal and restart to complete
      await new Promise(resolve => setTimeout(resolve, 2000));
      await refetchSession();
      setRepoChanging(false);
      successToast('Repository removed successfully');
    },
    onError: (error: Error) => {
      setRepoChanging(false);
      errorToast(error.message || 'Failed to remove repository');
    },
  });
  
  // Fetch OOTB workflows from backend (pass project for user's GitHub token)
  const { data: ootbWorkflows = [] } = useOOTBWorkflows(projectName);
  
  // Fetch workflow metadata (commands and agents) when workflow is active
  const { data: workflowMetadata } = useWorkflowMetadata(
    projectName,
    sessionName,
    !!activeWorkflow && !workflowActivating
  );
  
  // Git operations for selected directory
  const currentRemote = directoryRemotes[selectedDirectory.path];
  const { data: mergeStatus, refetch: refetchMergeStatus } = useGitMergeStatus(
    projectName,
    sessionName,
    selectedDirectory.path,
    currentRemote?.branch || 'main',
    !!currentRemote
  );
  const { data: remoteBranches = [] } = useGitListBranches(
    projectName,
    sessionName,
    selectedDirectory.path,
    !!currentRemote
  );
  const gitPullMutation = useGitPull();
  const gitPushMutation = useGitPush();
  
  // Fetch directory file listing (with subpath support)
  const fullDirectoryPath = currentSubPath 
    ? `${selectedDirectory.path}/${currentSubPath}` 
    : selectedDirectory.path;
  
  const { data: directoryFiles = [], refetch: refetchDirectoryFiles } = useWorkspaceList(
    projectName,
    sessionName,
    fullDirectoryPath,
    { enabled: openAccordionItems.includes("directories") }
  );
  
  // Reset subpath and inline file view when directory changes
  useEffect(() => {
    setCurrentSubPath("");
    setInlineViewingFile(null);
  }, [selectedDirectory.path, selectedDirectory.type]);
  
  // Track if we've already initialized from session to prevent infinite loops
  const initializedFromSessionRef = useRef(false);
  
  // Load active workflow from session spec if present
  useEffect(() => {
    // Only initialize once from session data, but wait for both session and ootbWorkflows to be ready
    if (initializedFromSessionRef.current || !session) return;
    
    // If session has an active workflow, wait for ootbWorkflows to load before initializing
    if (session.spec?.activeWorkflow && ootbWorkflows.length === 0) {
      return; // Wait for ootbWorkflows to load
    }
    
    if (session.spec?.activeWorkflow) {
      // Derive workflow ID from gitUrl if possible
      const gitUrl = session.spec.activeWorkflow.gitUrl;
      const matchingWorkflow = ootbWorkflows.find(w => w.gitUrl === gitUrl);
      if (matchingWorkflow) {
        setActiveWorkflow(matchingWorkflow.id);
        setSelectedWorkflow(matchingWorkflow.id);
      } else {
        setActiveWorkflow("custom");
        setSelectedWorkflow("custom");
      }
    }
    
    // Load remotes for all directories from annotations
    const annotations = session.metadata?.annotations || {};
    const remotes: Record<string, {url: string; branch: string}> = {};
    
    Object.keys(annotations).forEach(key => {
      if (key.startsWith('ambient-code.io/remote-') && key.endsWith('-url')) {
        // Decode path: backend uses :: as separator (e.g., "workflows::spec-kit" -> "workflows/spec-kit")
        const path = key.replace('ambient-code.io/remote-', '').replace('-url', '').replace(/::/g, '/');
        const branchKey = key.replace('-url', '-branch');
        remotes[path] = {
          url: annotations[key],
          branch: annotations[branchKey] || 'main'
        };
      }
    });
    
    setDirectoryRemotes(remotes);
    
    initializedFromSessionRef.current = true;
  }, [session, ootbWorkflows]);

  // Handler for inline file viewing and folder navigation
  const handleInlineFileOrFolderSelect = useCallback(async (node: FileTreeNode) => {
    if (node.type === 'folder') {
      // Navigate into folder
      const newSubPath = currentSubPath ? `${currentSubPath}/${node.name}` : node.name;
      setCurrentSubPath(newSubPath);
      setInlineViewingFile(null);
    } else {
      // Load file content inline
      setLoadingInlineFile(true);
      try {
        const fullPath = currentSubPath 
          ? `${selectedDirectory.path}/${currentSubPath}/${node.name}`
          : `${selectedDirectory.path}/${node.name}`;
        
        const response = await fetch(
          `/api/projects/${projectName}/agentic-sessions/${sessionName}/workspace/${encodeURIComponent(fullPath)}`
        );
        
        if (response.ok) {
          const content = await response.text();
          setInlineViewingFile({ path: node.name, content });
        } else {
          errorToast('Failed to load file');
        }
      } catch {
        errorToast('Failed to load file');
      } finally {
        setLoadingInlineFile(false);
      }
    }
  }, [projectName, sessionName, selectedDirectory.path, currentSubPath]);

  // Handler for downloading a single file
  const handleDownloadFile = useCallback(() => {
    if (!inlineViewingFile) return;

    try {
      const fullPath = currentSubPath
        ? `${selectedDirectory.path}/${currentSubPath}/${inlineViewingFile.path}`
        : `${selectedDirectory.path}/${inlineViewingFile.path}`;

      const downloadUrl = `/api/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${encodeURIComponent(fullPath)}`;

      // Create a hidden link and click it to trigger download
      const link = document.createElement('a');
      link.href = downloadUrl;
      link.download = inlineViewingFile.path;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);

      successToast(`Downloading ${inlineViewingFile.path}...`);
    } catch (err) {
      errorToast(err instanceof Error ? err.message : "Failed to download file");
    }
  }, [inlineViewingFile, currentSubPath, selectedDirectory.path, projectName, sessionName]);

  // Compute directory options from session data
  const directoryOptions = useMemo(() => {
    type DirectoryOption = {
      type: 'artifacts' | 'repo' | 'workflow';
      name: string;
      path: string;
    };
    
    const options: DirectoryOption[] = [
      { type: 'artifacts', name: 'Shared Artifacts', path: 'artifacts' }
    ];
    
    // Add repos from spec
    if (session?.spec?.repos) {
      session.spec.repos.forEach((repo, idx) => {
        const repoName = repo.input.url.split('/').pop()?.replace('.git', '') || `repo-${idx}`;
        options.push({
          type: 'repo',
          name: repoName,
          path: repoName
        });
      });
    }
    
    // Add active workflow
    if (activeWorkflow && session?.spec?.activeWorkflow) {
      const workflowName = session.spec.activeWorkflow.gitUrl.split('/').pop()?.replace('.git', '') || 'workflow';
      options.push({
        type: 'workflow',
        name: `Workflow: ${workflowName}`,
        path: `workflows/${workflowName}`
      });
    }
    
    return options;
  }, [session, activeWorkflow]);

  // Workspace state - removed unused tree/file management code

  // Handler for workflow selection (just sets pending, doesn't activate)
  const handleWorkflowChange = (value: string) => {
    setSelectedWorkflow(value);
    
    if (value === "none") {
      setPendingWorkflow(null);
      return;
    }
    
    if (value === "custom") {
      setCustomWorkflowDialogOpen(true);
      return;
    }
    
    // Find the selected workflow from OOTB workflows
    const workflow = ootbWorkflows.find(w => w.id === value);
    if (!workflow) {
      errorToast(`Workflow ${value} not found`);
      return;
    }
    
    if (!workflow.enabled) {
      errorToast(`Workflow ${workflow.name} is not yet available`);
      return;
    }
    
    // Set as pending (user must click Activate)
    setPendingWorkflow(workflow);
  };
  
  // Handler to activate the pending workflow
  const handleActivateWorkflow = async () => {
    if (!pendingWorkflow) return;
    
    setWorkflowActivating(true);
    
    try {
      // 1. Update CR with workflow configuration
      const response = await fetch(`/api/projects/${projectName}/agentic-sessions/${sessionName}/workflow`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          gitUrl: pendingWorkflow.gitUrl,
          branch: pendingWorkflow.branch,
          path: pendingWorkflow.path || "",
        }),
      });
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to update workflow");
      }
      
      // 2. Send WebSocket message to trigger workflow clone and restart
      await fetch(`/api/projects/${projectName}/agentic-sessions/${sessionName}/messages`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          type: "workflow_change",
          gitUrl: pendingWorkflow.gitUrl,
          branch: pendingWorkflow.branch,
          path: pendingWorkflow.path || "",
        }),
      });
      
      successToast(`Activating workflow: ${pendingWorkflow.name}`);
      setActiveWorkflow(pendingWorkflow.id);
      setPendingWorkflow(null);
      
      // Wait for restart to complete (give runner time to clone and restart)
      await new Promise(resolve => setTimeout(resolve, 3000));
      
      await refetchSession();
      successToast("Workflow activated successfully");
      
    } catch (error) {
      console.error("Failed to activate workflow:", error);
      errorToast(error instanceof Error ? error.message : "Failed to activate workflow");
    } finally {
      setWorkflowActivating(false);
    }
  };

  // Handler for custom workflow submission
  const handleCustomWorkflowSubmit = () => {
    if (!customWorkflowUrl.trim()) {
      errorToast("Git URL is required");
      return;
    }
    
    // Set as pending custom workflow
    setPendingWorkflow({
      id: "custom",
      name: "Custom Workflow",
      description: `Custom workflow from ${customWorkflowUrl.trim()}`,
      gitUrl: customWorkflowUrl.trim(),
      branch: customWorkflowBranch.trim() || "main",
      path: customWorkflowPath.trim() || "",
      enabled: true,
    });
    
    setCustomWorkflowDialogOpen(false);
    setSelectedWorkflow("custom");
  };
  
  // Fetch git status for selected directory
  const fetchGitStatus = useCallback(async () => {
    if (!projectName || !sessionName) return;
    
    try {
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/git/status?path=${encodeURIComponent(selectedDirectory.path)}`
      );
      
      if (response.ok) {
        const data = await response.json();
        setGitStatus(data);
      }
    } catch (error) {
      console.error("Failed to fetch git status:", error);
    }
  }, [projectName, sessionName, selectedDirectory.path]);
  
  // Poll git status when directories section is open
  useEffect(() => {
    if (openAccordionItems.includes("directories")) {
      fetchGitStatus();
      const interval = setInterval(fetchGitStatus, 30000); // Every 30s
      return () => clearInterval(interval);
    }
  }, [openAccordionItems, fetchGitStatus]);
  
  // Handler to configure directory remote
  const handleConfigureRemote = async () => {
    if (!remoteUrl.trim()) {
      errorToast("Repository URL is required");
      return;
    }
    
    try {
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/git/configure-remote`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            path: selectedDirectory.path,
            remoteUrl: remoteUrl.trim(),
            branch: remoteBranch.trim() || "main",
          }),
        }
      );
      
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Failed to configure remote");
      }
      
      const newRemotes = {...directoryRemotes};
      newRemotes[selectedDirectory.path] = {
        url: remoteUrl.trim(),
        branch: remoteBranch.trim() || "main",
      };
      setDirectoryRemotes(newRemotes);
      
      setRemoteDialogOpen(false);
      successToast("Remote configured successfully");
      await fetchGitStatus();
      refetchMergeStatus();
      
    } catch (error) {
      errorToast(error instanceof Error ? error.message : "Failed to configure remote");
    }
  };
  
  // Old synchronize handler - replaced by handleGitSynchronize below
  // Kept for backwards compatibility during transition

  // Handler to pull changes
  const handleGitPull = () => {
    if (!currentRemote) return;

    gitPullMutation.mutate(
      {
        projectName,
        sessionName,
        path: selectedDirectory.path,
        branch: currentRemote.branch,
      },
      {
        onSuccess: () => {
          successToast("Changes pulled successfully");
          fetchGitStatus();
          refetchMergeStatus();
        },
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to pull changes"),
      }
    );
  };

  // Handler to push changes
  const handleGitPush = () => {
    if (!currentRemote) return;

      const timestamp = new Date().toISOString();
      const message = `Workflow progress - ${timestamp}`;
      
    gitPushMutation.mutate(
      {
        projectName,
        sessionName,
        path: selectedDirectory.path,
            branch: currentRemote.branch,
        message,
      },
      {
        onSuccess: () => {
          successToast("Changes pushed successfully");
          fetchGitStatus();
          refetchMergeStatus();
        },
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to push changes"),
      }
    );
  };

  // Handler to synchronize (pull then push)
  const handleGitSynchronize = async () => {
    if (!currentRemote) return;

    try {
      setSynchronizing(true);
      
      // Pull first
      await gitPullMutation.mutateAsync({
        projectName,
        sessionName,
        path: selectedDirectory.path,
        branch: currentRemote.branch,
      });
      
      // Then push
      const timestamp = new Date().toISOString();
      const message = `Workflow progress - ${timestamp}`;
      
      await gitPushMutation.mutateAsync({
        projectName,
        sessionName,
        path: selectedDirectory.path,
        branch: currentRemote.branch,
        message,
      });
      
      successToast("Changes synchronized successfully");
      fetchGitStatus();
      refetchMergeStatus();
    } catch (error) {
      errorToast(error instanceof Error ? error.message : "Failed to synchronize");
    } finally {
      setSynchronizing(false);
    }
  };

  // Handler to commit changes without pushing
  const handleCommit = async () => {
    if (!commitMessage.trim()) {
      errorToast("Commit message is required");
      return;
    }

    setCommitting(true);
    try {
      // Use the synchronize endpoint but it will detect there's nothing on remote to pull
      // So it just commits locally
      const response = await fetch(
        `/api/projects/${projectName}/agentic-sessions/${sessionName}/git/synchronize`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            path: selectedDirectory.path,
            message: commitMessage.trim(),
            branch: currentRemote?.branch || 'main',
          }),
        }
      );

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to commit');
      }

      successToast('Changes committed successfully');
      setCommitMessage("");
      setCommitModalOpen(false);
      fetchGitStatus();
      refetchMergeStatus();
    } catch (error) {
      errorToast(error instanceof Error ? error.message : 'Failed to commit');
    } finally {
      setCommitting(false);
    }
  };

  // Removed: deriveRepoFolderFromUrl - No longer needed
  // Removed: useAllSessionGitHubDiffs - Old diff endpoint not used, git/status provides current info

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
    if (!chatInput.trim() && !selectedAgents.length && !autoSelectAgents) return;

    // Build message with agent prepend if needed
    let finalMessage = chatInput.trim();

    if (autoSelectAgents) {
      finalMessage = "You MUST use relevant sub-agents when needed based on the task at hand. " + finalMessage;
    } else if (selectedAgents.length > 0) {
      const agentNames = selectedAgents
        .map(id => workflowMetadata?.agents?.find(a => a.id === id))
        .filter(Boolean)
        .map(agent => agent!.name)
        .join(', ');

      finalMessage = `You MUST collaborate with these agents: ${agentNames}. ` + finalMessage;
    }

    sendChatMutation.mutate(
      { projectName, sessionName, content: finalMessage },
      {
        onSuccess: () => {
          setChatInput("");
          // Clear agent selection after sending
          setSelectedAgents([]);
          setAutoSelectAgents(false);
        },
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to send message"),
      }
    );
  };

  const handleCommandClick = (slashCommand: string) => {
    // Build message with agent prepend if needed (same logic as sendChat)
    let finalMessage = slashCommand;

    if (autoSelectAgents) {
      finalMessage = "You MUST use relevant sub-agents when needed based on the task at hand. " + finalMessage;
    } else if (selectedAgents.length > 0) {
      const agentNamesStr = selectedAgents
        .map(id => workflowMetadata?.agents?.find(a => a.id === id))
        .filter(Boolean)
        .map(agent => agent!.name)
        .join(', ');

      finalMessage = `You MUST collaborate with these agents: ${agentNamesStr}. ` + finalMessage;
    }

    sendChatMutation.mutate(
      { projectName, sessionName, content: finalMessage },
      {
        onSuccess: () => {
          successToast(`Command ${slashCommand} sent`);
          // Clear agent selection after sending
          setSelectedAgents([]);
          setAutoSelectAgents(false);
        },
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to send command"),
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


  const durationMs = useMemo(() => {
    const start = session?.status?.startTime ? new Date(session.status.startTime).getTime() : undefined;
    const end = session?.status?.completionTime ? new Date(session.status.completionTime).getTime() : Date.now();
    return start ? Math.max(0, end - start) : undefined;
  }, [session?.status?.startTime, session?.status?.completionTime]);

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
          <SessionHeader
                      session={session}
            projectName={projectName}
            actionLoading={
              stopMutation.isPending ? "stopping" :
              deleteMutation.isPending ? "deleting" :
              null
            }
            onRefresh={refetchSession}
            onStop={handleStop}
            onDelete={handleDelete}
            durationMs={durationMs}
            k8sResources={k8sResources}
            messageCount={messages.length}
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
                  <div className="flex items-center gap-2">
                    <Workflow className="h-4 w-4" />
                    <span>Workflows</span>
                    {activeWorkflow && !openAccordionItems.includes("workflows") && (
                      <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                        {ootbWorkflows.find(w => w.id === activeWorkflow)?.name || "Custom Workflow"}
                      </Badge>
                    )}
                  </div>
                </AccordionTrigger>
                <AccordionContent className="pt-2 pb-3">
                  {(session?.status?.phase === 'Stopped' || session?.status?.phase === 'Error' || session?.status?.phase === 'Completed') ? (
                    <EmptyState
                      icon={Play}
                      title="Session not running"
                      description="You need to restart this session to use agents, commands, or switch workflows."
                      className="py-8"
                    />
                  ) : (
                  <div className="space-y-3">
                    
                    {/* Workflow selector - always visible except when activating */}
                    {!workflowActivating && (
                      <>
                        <p className="text-sm text-muted-foreground">
                          Workflows provide agents with pre-defined context and structured steps to follow.
                        </p>
                        
                        <div>
                          {/*
                          <label className="text-sm font-medium mb-1.5 block">
                            Workflows
                          </label>
                          */}
                          <Select value={selectedWorkflow} onValueChange={handleWorkflowChange} disabled={workflowActivating}>
                            <SelectTrigger className="w-full h-auto py-8">
                              <SelectValue placeholder="Generic chat" />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="none">
                                <div className="flex flex-col items-start gap-0.5 py-1">
                                  <span>General chat</span>
                                  <span className="text-xs text-muted-foreground font-normal">
                                      A general chat session with no structured workflow.
                                    </span>
                                </div>
                              </SelectItem>
                              {ootbWorkflows.map((workflow) => (
                                <SelectItem 
                                  key={workflow.id} 
                                  value={workflow.id}
                                  disabled={!workflow.enabled}
                                >
                                  <div className="flex flex-col items-start gap-0.5 py-1">
                                    <span>{workflow.name}</span>
                                    <span className="text-xs text-muted-foreground font-normal">
                                      {workflow.description}
                                    </span>
                                  </div>
                                </SelectItem>
                              ))}
                              <SelectItem value="custom">
                                <div className="flex flex-col items-start gap-0.5 py-1">
                                  <span>Custom Workflow...</span>
                                  <span className="text-xs text-muted-foreground font-normal">
                                    Load a workflow from a custom Git repository
                                  </span>
                                </div>
                              </SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        
                        {/* Show workflow preview and activate/switch button */}
                        {pendingWorkflow && (
                          <Alert className="bg-blue-50 border-blue-200">
                            <AlertCircle className="h-4 w-4 text-blue-600" />
                            <AlertTitle className="text-blue-900">
                              Reload required
                            </AlertTitle>
                            <AlertDescription className="text-blue-800">
                              <div className="space-y-2 mt-2">
                                <p className="text-sm">
                                  Claude will {activeWorkflow ? 'restart and switch to' : 'pause briefly to load'} the workflow. Your chat history will be preserved.
                                </p>
                                <Button 
                                  onClick={handleActivateWorkflow}
                                  className="w-full mt-3"
                                  size="sm"
                                >
                                  <Play className="mr-2 h-4 w-4" />
                                  Load new workflow
                                </Button>
                              </div>
                            </AlertDescription>
                          </Alert>
                        )}
                      </>
                    )}
                    
                    {/* Show active workflow info */}
                    {activeWorkflow && !workflowActivating && (
                      <>
                      
                    {/* Commands Section */}
                    {workflowMetadata?.commands && workflowMetadata.commands.length > 0 && (
                      <div className="space-y-2">
                        {/* View commands expandable section */}
                        <div>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="w-full justify-between h-8 px-2"
                            onClick={() => setShowCommandsList(!showCommandsList)}
                          >
                            <span className="text-xs font-medium">
                              {showCommandsList ? 'Hide' : 'Show'} {workflowMetadata.commands.length} available command{workflowMetadata.commands.length !== 1 ? 's' : ''}
                            </span>
                            {showCommandsList ? (
                              <ChevronDown className="h-3 w-3" />
                            ) : (
                              <ChevronRight className="h-3 w-3" />
                            )}
                          </Button>

                          {showCommandsList && (
                            <div className="relative mt-2">
                              {commandsScrollTop && (
                                <div className="absolute top-0 left-0 right-0 h-8 bg-gradient-to-b from-white to-transparent pointer-events-none z-10" />
                              )}
                              <div 
                                className="max-h-[400px] overflow-y-auto space-y-2 pr-1"
                                onScroll={(e) => {
                                  const target = e.currentTarget;
                                  const isScrolledFromTop = target.scrollTop > 10;
                                  const isScrolledToBottom = target.scrollHeight - target.scrollTop <= target.clientHeight + 10;
                                  setCommandsScrollTop(isScrolledFromTop);
                                  setCommandsScrollBottom(!isScrolledToBottom);
                                }}
                              >
                                {workflowMetadata.commands.map((cmd) => {
                                  // Extract command name after last dot and capitalize
                                  const commandTitle = cmd.name.includes('.') 
                                    ? cmd.name.split('.').pop() 
                                    : cmd.name;
                                  
                                  return (
                                    <div
                                      key={cmd.id}
                                      className="p-3 rounded-md border bg-muted/30"
                                    >
                                      <div className="flex items-center justify-between mb-1">
                                        <h3 className="text-sm font-bold capitalize">
                                          {commandTitle}
                                        </h3>
                                        <Button
                                          variant="outline"
                                          size="sm"
                                          className="flex-shrink-0 h-7 text-xs"
                                          onClick={() => handleCommandClick(cmd.slashCommand)}
                                        >
                                          Run {cmd.slashCommand.replace(/^\/speckit\./, '/')}
                                        </Button>
                                      </div>
                                      {cmd.description && (
                                        <p className="text-xs text-muted-foreground">
                                          {cmd.description}
                                        </p>
                                      )}
                                    </div>
                                  );
                                })}
                              </div>
                              {commandsScrollBottom && (
                                <div className="absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-white to-transparent pointer-events-none z-10" />
                              )}
                            </div>
                          )}
                        </div>
                      </div>
                    )}

                        {workflowMetadata?.commands?.length === 0 && (
                          <p className="text-xs text-muted-foreground text-center py-2">
                            No commands found in this workflow
                          </p>
                        )}

                        {/* Agents Section */}
                        {workflowMetadata?.agents && workflowMetadata.agents.length > 0 && (
                          <div className="space-y-2">
                            {/* View agents expandable section */}
                            <div>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="w-full justify-between h-8 px-2"
                                onClick={() => setShowAgentsList(!showAgentsList)}
                              >
                                <span className="text-xs font-medium">
                                  {showAgentsList ? 'Hide' : 'Show'} {workflowMetadata.agents.length} available agent{workflowMetadata.agents.length !== 1 ? 's' : ''}
                                </span>
                                {showAgentsList ? (
                                  <ChevronDown className="h-3 w-3" />
                                ) : (
                                  <ChevronRight className="h-3 w-3" />
                                )}
                              </Button>

                              {showAgentsList && (
                                <TooltipProvider>
                                  <div className="space-y-2 max-h-48 overflow-y-auto mt-2 pt-2 m-3">
                                    <div className="flex items-center space-x-2 pb-2">
                                      <Checkbox
                                        id="auto-select-agents"
                                        checked={autoSelectAgents}
                                        onCheckedChange={(checked) => {
                                          setAutoSelectAgents(!!checked);
                                          if (checked) setSelectedAgents([]);
                                        }}
                                      />
                                      <Sparkles className="h-3 w-3 text-purple-500" />
                                      <Label htmlFor="auto-select-agents" className="text-sm font-normal cursor-pointer">
                                        Automatically select recommended agents for each task
                                      </Label>
                                    </div>
                                    <div className="space-y-1 space-x-6">
                                      {workflowMetadata.agents.map((agent) => (
                                        <Tooltip key={agent.id}>
                                          <TooltipTrigger asChild>
                                            <div className="flex items-center space-x-2">
                                              <Checkbox
                                                id={`agent-${agent.id}`}
                                                checked={selectedAgents.includes(agent.id)}
                                                disabled={autoSelectAgents}
                                                onCheckedChange={(checked) => {
                                                  if (checked) {
                                                    setSelectedAgents([...selectedAgents, agent.id]);
                                                  } else {
                                                    setSelectedAgents(selectedAgents.filter(id => id !== agent.id));
                                                  }
                                                }}
                                              />
                                              <Label
                                                htmlFor={`agent-${agent.id}`}
                                                className="text-sm font-normal cursor-pointer flex-1"
                                              >
                                                {agent.name}
                                              </Label>
                                            </div>
                                          </TooltipTrigger>
                                          <TooltipContent>
                                            <p className="max-w-xs">{agent.description}</p>
                                          </TooltipContent>
                                        </Tooltip>
                                      ))}
                                    </div>
                                  </div>
                                </TooltipProvider>
                              )}
                            </div>

                            {/* Temporarily disabled */}
                            {/* {(selectedAgents.length > 0 || autoSelectAgents) && (
                              <div className="bg-blue-50 border border-blue-200 rounded-md px-3 py-1.5 flex items-center gap-2">
                                <Info className="h-3 w-3 text-blue-600 flex-shrink-0" />
                                <span className="text-xs text-blue-800">
                                  Next message will include agent instructions
                                </span>
                              </div>
                            )} */}
                          </div>
                        )}

                        {workflowMetadata?.agents?.length === 0 && (
                          <p className="text-xs text-muted-foreground text-center py-2">
                            No agents found in this workflow
                          </p>
                        )}
                      </>
                    )}
                    
                    {/* Show activating/switching state */}
                    {workflowActivating && (
                      <Alert>
                        <Loader2 className="h-4 w-4 animate-spin" />
                        <AlertTitle>{activeWorkflow ? 'Switching Workflow...' : 'Activating Workflow...'}</AlertTitle>
                        <AlertDescription>
                          <div className="space-y-2">
                            <p>Claude is restarting with the new workflow.</p>
                            <ul className="text-sm space-y-1 mt-2 list-disc list-inside">
                              <li>Cloning workflow repository</li>
                              <li>Setting up workspace structure</li>
                              <li>Restarting Claude Code</li>
                            </ul>
                            <p className="text-xs text-muted-foreground mt-2">
                              This may take 10-30 seconds...
                            </p>
                          </div>
                        </AlertDescription>
                      </Alert>
                    )}
                    
                  </div>
                  )}
                </AccordionContent>
              </AccordionItem>

              {/* Context - Add/Remove Repositories and other context sources */}
              <AccordionItem value="context" className="border rounded-lg px-3 bg-white">
                <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                  <div className="flex items-center gap-2">
                    <Link className="h-4 w-4" />
                    <span>Context</span>
                    {session?.spec?.repos && session.spec.repos.length > 0 && (
                      <Badge variant="secondary" className="ml-auto mr-2">
                        {session.spec.repos.length}
                      </Badge>
                    )}
                  </div>
                </AccordionTrigger>
                <AccordionContent className="pt-2 pb-3">
                    <div className="space-y-3">
                    <p className="text-sm text-muted-foreground">
                      Add additional context to enhance the AI's understanding
                    </p>
                    
                    {/* Repository List */}
                    {!session?.spec?.repos || session.spec.repos.length === 0 ? (
                      <div className="text-center py-6">
                        <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-gray-100 mb-2">
                          <GitBranch className="h-5 w-5 text-gray-400" />
                        </div>
                        <p className="text-sm text-muted-foreground mb-3">No repositories added</p>
                        <Button size="sm" variant="outline" onClick={() => setContextModalOpen(true)}>
                          <GitBranch className="mr-2 h-3 w-3" />
                          Add Repository
                          </Button>
                        </div>
                    ) : (
                      <div className="space-y-2">
                        {session.spec.repos.map((repo, idx) => {
                          const repoName = repo.input.url.split('/').pop()?.replace('.git', '') || `repo-${idx}`;
                          return (
                            <div key={idx} className="flex items-center gap-2 p-2 border rounded bg-muted/30 hover:bg-muted/50 transition-colors">
                              <GitBranch className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                              <div className="flex-1 min-w-0">
                                <div className="text-sm font-medium truncate">{repoName}</div>
                                <div className="text-xs text-muted-foreground truncate">{repo.input.url}</div>
                          </div>
                          <Button 
                                variant="ghost"
                            size="sm" 
                                className="h-7 w-7 p-0 flex-shrink-0"
                                onClick={() => {
                                  if (confirm(`Remove repository ${repoName}?`)) {
                                    removeRepoMutation.mutate(repoName);
                                  }
                                }}
                              >
                                <X className="h-3 w-3" />
                          </Button>
                                  </div>
                                );
                              })}
                        <Button onClick={() => setContextModalOpen(true)} variant="outline" className="w-full" size="sm">
                          <GitBranch className="mr-2 h-3 w-3" />
                        Add Repository
                      </Button>
                                </div>
                              )}
                    
                    {/* Future: Files and URLs would go here */}
                    <div className="border-t pt-3">
                      <p className="text-xs text-muted-foreground text-center">
                        Additional context types (file imports, Jira, Google drive) coming soon
                      </p>
                            </div>
                          </div>
                </AccordionContent>
              </AccordionItem>

              {/* File Explorer (unified for artifacts, repos, and workflow) */}
              <AccordionItem value="directories" className="border rounded-lg px-3 bg-white">
                <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                  <div className="flex items-center gap-2 w-full">
                    <Folder className="h-4 w-4" />
                    <span>File Explorer</span>
                    {gitStatus?.hasChanges && (
                      <div className="flex gap-1 ml-auto mr-2">
                        {gitStatus.totalAdded > 0 && (
                          <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                            +{gitStatus.totalAdded}
                          </Badge>
                        )}
                        {gitStatus.totalRemoved > 0 && (
                          <Badge variant="outline" className="bg-red-50 text-red-700 border-red-200">
                            -{gitStatus.totalRemoved}
                      </Badge>
                        )}
                      </div>
                    )}
                  </div>
                </AccordionTrigger>
                <AccordionContent className="pt-2 pb-3">
                  <div className="space-y-3">
                    {/* Panel Description */}
                    <p className="text-sm text-muted-foreground">
                      Browse, view, and manage files in your workspace directories. Track changes and sync with Git for version control.
                    </p>
                    
                    {/* Directory Selector */}
                    <div className="flex items-center justify-between gap-2">
                      <Label className="text-xs text-muted-foreground">Directory:</Label>
                      <Select
                        value={`${selectedDirectory.type}:${selectedDirectory.path}`}
                        onValueChange={(value) => {
                          const [type, ...pathParts] = value.split(':');
                          const path = pathParts.join(':');
                          const option = directoryOptions.find(
                            opt => opt.type === type && opt.path === path
                          );
                          if (option) setSelectedDirectory(option);
                        }}
                      >
                        <SelectTrigger className="w-[250px] h-8">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {directoryOptions.map(opt => (
                            <SelectItem key={`${opt.type}:${opt.path}`} value={`${opt.type}:${opt.path}`}>
                              <div className="flex items-center gap-2">
                                {opt.type === 'artifacts' && <Folder className="h-3 w-3" />}
                                {opt.type === 'repo' && <GitBranch className="h-3 w-3" />}
                                {opt.type === 'workflow' && <Sparkles className="h-3 w-3" />}
                                <span className="text-xs">{opt.name}</span>
                              </div>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    
                    {/* File Browser for Selected Directory */}
                    <div className="border rounded-lg overflow-hidden">
                      {/* Header with breadcrumbs and actions */}
                      <div className="px-2 py-1.5 border-b flex items-center justify-between bg-muted/30">
                        <div className="flex items-center gap-1 text-xs text-muted-foreground min-w-0 flex-1">
                          {/* Back button when in subfolder or viewing file */}
                          {(currentSubPath || inlineViewingFile) && (
                            <Button 
                              variant="ghost" 
                              size="sm" 
                              onClick={() => {
                                if (inlineViewingFile) {
                                  // Go back to file tree
                                  setInlineViewingFile(null);
                                } else if (currentSubPath) {
                                  // Go back to parent folder
                                  const pathParts = currentSubPath.split('/');
                                  pathParts.pop();
                                  setCurrentSubPath(pathParts.join('/'));
                                }
                              }}
                              className="h-6 px-1.5 mr-1"
                            >
                              ← Back
                            </Button>
                          )}
                          
                          {/* Breadcrumb path */}
                          <Folder className="inline h-3 w-3 mr-1 flex-shrink-0" />
                          <code className="bg-muted px-1 py-0.5 rounded text-xs truncate">
                            {selectedDirectory.path}
                            {currentSubPath && `/${currentSubPath}`}
                            {inlineViewingFile && `/${inlineViewingFile.path}`}
                          </code>
                        </div>

                        {/* Action buttons */}
                        {inlineViewingFile ? (
                          /* Download button when viewing file */
                          <div className="flex items-center gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={handleDownloadFile}
                              className="h-6 px-2 flex-shrink-0"
                              title="Download file"
                            >
                              <Download className="h-3 w-3" />
                            </Button>
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="sm" className="h-6 px-2 flex-shrink-0">
                                  <MoreVertical className="h-3 w-3" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem disabled className="text-xs text-muted-foreground">
                                  Sync to Jira - Coming soon
                                </DropdownMenuItem>
                                <DropdownMenuItem disabled className="text-xs text-muted-foreground">
                                  Sync to GDrive - Coming soon
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </div>
                        ) : (
                          /* Refresh button when not viewing file */
                          <Button variant="ghost" size="sm" onClick={() => refetchDirectoryFiles()} className="h-6 px-2 flex-shrink-0">
                            <FolderSync className="h-3 w-3" />
                        </Button>
                        )}
                      </div>
                      
                      {/* Content area */}
                      <div className="p-2 max-h-64 overflow-y-auto">
                        {loadingInlineFile ? (
                          <div className="flex items-center justify-center py-8">
                            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                          </div>
                        ) : inlineViewingFile ? (
                          /* File content view */
                          <div className="text-xs">
                            <pre className="bg-muted/50 p-2 rounded overflow-x-auto">
                              <code>{inlineViewingFile.content}</code>
                            </pre>
                          </div>
                        ) : directoryFiles.length === 0 ? (
                          /* Empty state */
                          <div className="text-center py-4 text-sm text-muted-foreground">
                            <FolderTree className="h-8 w-8 mx-auto mb-2 opacity-30" />
                            <p>No files yet</p>
                            <p className="text-xs mt-1">Files will appear here</p>
                          </div>
                        ) : (
                          /* File tree */
                          <FileTree 
                            nodes={directoryFiles.map((item): FileTreeNode => ({
                              name: item.name,
                              path: item.path,
                              type: item.isDir ? 'folder' : 'file',
                              sizeKb: item.size ? item.size / 1024 : undefined,
                            }))}
                            onSelect={handleInlineFileOrFolderSelect}
                          />
                        )}
                      </div>
                    </div>
                    
                    {/* Remote Configuration Status */}
                    {!currentRemote ? (
                      <div className="border border-blue-200 bg-blue-50 rounded-md px-3 py-2 flex items-center justify-between">
                        <span className="text-sm text-blue-800">Set up Git remote for version control</span>
                        <Button onClick={() => {
                          setRemoteUrl("");
                          setRemoteBranch("main");
                          setRemoteDialogOpen(true);
                        }} size="sm" variant="outline">
                          <GitBranch className="mr-2 h-3 w-3" />
                          Configure
                          </Button>
                      </div>
                    ) : (
                      <div className="border rounded-md px-2 py-1.5">
                        {/* Single-line git status bar */}
                        <div className="flex items-center gap-2 text-xs">
                          {/* Remote info */}
                          <div className="flex items-center gap-1.5 text-muted-foreground">
                            <Cloud className="h-3 w-3" />
                            <span className="truncate max-w-[200px]">
                              {currentRemote?.url?.split('/').slice(-2).join('/').replace('.git', '') || ''}/{currentRemote?.branch || 'main'}
                            </span>
                          </div>
                          
                          <div className="flex-1" />
                          
                          {/* Status indicator */}
                          {mergeStatus && !mergeStatus.canMergeClean ? (
                            <div className="flex items-center gap-1 text-red-600">
                              <X className="h-3 w-3" />
                              <span className="font-medium">conflict</span>
                            </div>
                          ) : (gitStatus?.hasChanges || mergeStatus?.remoteCommitsAhead) ? (
                            <div className="flex items-center gap-1.5 text-muted-foreground text-xs">
                              {mergeStatus?.remoteCommitsAhead ? (
                                <span>↓{mergeStatus.remoteCommitsAhead}</span>
                              ) : null}
                              {gitStatus?.hasChanges ? (
                                <span className="font-normal">{gitStatus.uncommittedFiles} uncommitted</span>
                              ) : null}
                            </div>
                          ) : null}
                          
                          {/* Sync button - only enabled when NO uncommitted changes */}
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                              <Button 
                                size="sm"
                                  variant="ghost"
                                  onClick={handleGitSynchronize}
                                  disabled={!mergeStatus?.canMergeClean || synchronizing || gitStatus?.hasChanges}
                                  className="h-6 w-6 p-0"
                                >
                                  {synchronizing ? (
                                  <Loader2 className="h-3 w-3 animate-spin" />
                                ) : (
                                    <RefreshCw className="h-3 w-3" />
                                )}
                              </Button>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>{gitStatus?.hasChanges ? 'Commit changes first' : `Sync with origin/${currentRemote?.branch || 'main'}`}</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>

                          {/* More options */}
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                                <MoreVertical className="h-3 w-3" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem
                                onClick={() => {
                                  setRemoteUrl(currentRemote?.url || '');
                                  setRemoteBranch(currentRemote?.branch || 'main');
                                  setShowCreateBranch(false);
                                  setRemoteDialogOpen(true);
                                }}
                              >
                                <Edit className="mr-2 h-3 w-3" />
                                Manage Remote
                              </DropdownMenuItem>
                              <DropdownMenuSeparator />
                              <DropdownMenuItem
                                onClick={() => {
                                  setCommitMessage(`Update ${selectedDirectory.name} - ${new Date().toLocaleString()}`);
                                  setCommitModalOpen(true);
                                }}
                                disabled={!gitStatus?.hasChanges}
                              >
                                <Edit className="mr-2 h-3 w-3" />
                                Commit Changes
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={handleGitPull}
                                disabled={!mergeStatus?.canMergeClean || gitPullMutation.isPending}
                              >
                                <CloudDownload className="mr-2 h-3 w-3" />
                                Pull
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={handleGitPush}
                                disabled={!mergeStatus?.canMergeClean || gitPushMutation.isPending || gitStatus?.hasChanges}
                              >
                                <CloudUpload className="mr-2 h-3 w-3" />
                                Push
                              </DropdownMenuItem>
                              <DropdownMenuSeparator />
                              <DropdownMenuItem
                                onClick={() => {
                                  const newRemotes = {...directoryRemotes};
                                  delete newRemotes[selectedDirectory.path];
                                  setDirectoryRemotes(newRemotes);
                                  successToast("Git remote disconnected");
                                }}
                              >
                                <X className="mr-2 h-3 w-3 text-red-600" />
                                Disconnect
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </div>
                      </div>
                    )}
                    
                    
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </div>

          {/* Right Column - Messages (Always Visible) */}
          <div>
            <Card className="relative">
              <CardContent className="p-3">
                {/* Workflow activation overlay */}
                {workflowActivating && (
                  <div className="absolute inset-0 bg-white/90 backdrop-blur-sm z-10 flex items-center justify-center rounded-lg">
                    <Alert className="max-w-md mx-4">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      <AlertTitle>Activating Workflow...</AlertTitle>
                      <AlertDescription>
                        <p>Claude is restarting with the new workflow. Please wait...</p>
                      </AlertDescription>
                    </Alert>
                  </div>
                )}
                
                {/* Repository change overlay */}
                {repoChanging && (
                  <div className="absolute inset-0 bg-white/90 backdrop-blur-sm z-10 flex items-center justify-center rounded-lg">
                    <Alert className="max-w-md mx-4">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      <AlertTitle>Updating Repositories...</AlertTitle>
                      <AlertDescription>
                        <div className="space-y-2">
                          <p>Claude is paused while repositories are being updated.</p>
                          <ul className="text-sm space-y-1 mt-2 list-disc list-inside">
                            <li>Cloning repository to workspace</li>
                            <li>Restarting Claude Code</li>
                          </ul>
                          <p className="text-xs text-muted-foreground mt-2">
                            This may take 10-20 seconds...
                          </p>
                        </div>
                      </AlertDescription>
                    </Alert>
                  </div>
                )}
                
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
                  selectedAgents={selectedAgents}
                  autoSelectAgents={autoSelectAgents}
                  agentNames={selectedAgents
                    .map(id => workflowMetadata?.agents?.find(a => a.id === id))
                    .filter(Boolean)
                    .map(agent => agent!.name)}
                />
              </CardContent>
            </Card>
          </div>
        </div>
        </div>
      </div>
    </div>

    {/* Add Context Modal */}
    <Dialog open={contextModalOpen} onOpenChange={setContextModalOpen}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Add Context</DialogTitle>
          <DialogDescription>
            Add additional context to enhance the AI's understanding
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="context-url">Repository URL</Label>
            <Input
              id="context-url"
              placeholder="https://github.com/org/repo"
              value={contextUrl}
              onChange={(e) => setContextUrl(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Currently supports GitHub repositories for code context
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="context-branch">Branch (optional)</Label>
            <Input
              id="context-branch"
              placeholder="main"
              value={contextBranch}
              onChange={(e) => setContextBranch(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Leave empty to use the default branch
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              setContextUrl("");
              setContextBranch("main");
              setContextModalOpen(false);
            }}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={async () => {
              if (!contextUrl.trim()) return;
              
              try {
                // Add repository to session
                await addRepoMutation.mutateAsync({
                  url: contextUrl.trim(),
                  branch: contextBranch.trim() || 'main',
                });
                
                successToast('Repository added successfully');
                setContextUrl("");
                setContextBranch("main");
                setContextModalOpen(false);
              } catch (err) {
                errorToast(err instanceof Error ? err.message : 'Failed to add repository');
              }
            }}
            disabled={!contextUrl.trim() || addRepoMutation.isPending}
          >
            {addRepoMutation.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Adding...
              </>
            ) : (
              'Add'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    {/* Custom Workflow Dialog */}
    <Dialog open={customWorkflowDialogOpen} onOpenChange={setCustomWorkflowDialogOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Load Custom Workflow</DialogTitle>
          <DialogDescription>
            Enter the Git repository URL and optional path for your custom workflow.
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="workflow-url">Git Repository URL *</Label>
            <Input
              id="workflow-url"
              placeholder="https://github.com/org/workflow-repo.git"
              value={customWorkflowUrl}
              onChange={(e) => setCustomWorkflowUrl(e.target.value)}
              disabled={workflowActivating}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="workflow-branch">Branch</Label>
            <Input
              id="workflow-branch"
              placeholder="main"
              value={customWorkflowBranch}
              onChange={(e) => setCustomWorkflowBranch(e.target.value)}
              disabled={workflowActivating}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="workflow-path">Path (optional)</Label>
            <Input
              id="workflow-path"
              placeholder="workflows/my-workflow"
              value={customWorkflowPath}
              onChange={(e) => setCustomWorkflowPath(e.target.value)}
              disabled={workflowActivating}
            />
            <p className="text-xs text-muted-foreground">
              Optional subdirectory within the repository containing the workflow
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setCustomWorkflowDialogOpen(false)}
            disabled={workflowActivating}
          >
            Cancel
          </Button>
          <Button
            onClick={handleCustomWorkflowSubmit}
            disabled={!customWorkflowUrl.trim() || workflowActivating}
          >
            {workflowActivating ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Loading...
              </>
            ) : (
              'Load Workflow'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    {/* Manage Remote Dialog - Combined URL and Branch Management */}
    <Dialog open={remoteDialogOpen} onOpenChange={setRemoteDialogOpen}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Manage Remote for {selectedDirectory.name}</DialogTitle>
          <DialogDescription>
            Configure repository URL and select branch for git operations
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="remote-repo-url">Repository URL *</Label>
            <Input
              id="remote-repo-url"
              placeholder="https://github.com/org/my-repo.git"
              value={remoteUrl}
              onChange={(e) => setRemoteUrl(e.target.value)}
            />
          </div>

          {!showCreateBranch ? (
          <div className="space-y-2">
              <Label htmlFor="remote-branch">Branch</Label>
              <div className="flex gap-2">
                <Select 
                  value={remoteBranch} 
                  onValueChange={(branch) => setRemoteBranch(branch)}
                >
                  <SelectTrigger className="flex-1">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {!remoteBranches.includes(remoteBranch) && remoteBranch && (
                      <SelectItem value={remoteBranch}>{remoteBranch} (current)</SelectItem>
                    )}
                    {remoteBranches.map(b => (
                      <SelectItem key={b} value={b}>{b}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button 
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setShowCreateBranch(true);
                    setNewBranchName(`session-${sessionName.substring(0, 20)}`);
                  }}
                >
                  New
                </Button>
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              <Label>Create New Branch</Label>
            <Input
                placeholder="branch-name"
                value={newBranchName}
                onChange={e => setNewBranchName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newBranchName.trim()) {
                    setRemoteBranch(newBranchName.trim());
                    setShowCreateBranch(false);
                  }
                }}
                autoFocus
              />
              <div className="flex gap-2">
                <Button 
                  size="sm"
                  className="flex-1"
                  onClick={() => {
                    setRemoteBranch(newBranchName.trim());
                    setShowCreateBranch(false);
                  }}
                  disabled={!newBranchName.trim()}
                >
                  Set Branch
                </Button>
                <Button 
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setShowCreateBranch(false);
                    setNewBranchName("");
                  }}
                >
                  Cancel
                </Button>
          </div>
            </div>
          )}
          
          {mergeStatus && !showCreateBranch && (
            <div className="text-xs text-muted-foreground border-t pt-2">
              {mergeStatus.canMergeClean ? (
                <span className="text-green-600">✓ Can merge cleanly</span>
              ) : (
                <span className="text-amber-600">⚠️ {mergeStatus.conflictingFiles.length} conflicts</span>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => {
              setRemoteDialogOpen(false);
              setShowCreateBranch(false);
            }}
          >
            Cancel
          </Button>
          <Button
            onClick={() => {
              handleConfigureRemote();
              setShowCreateBranch(false);
            }}
            disabled={!remoteUrl.trim()}
          >
            Save Remote
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
    
    {/* Commit Changes Dialog */}
    <Dialog open={commitModalOpen} onOpenChange={setCommitModalOpen}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Commit Changes</DialogTitle>
          <DialogDescription>
            Commit {gitStatus?.uncommittedFiles || 0} files to {selectedDirectory.name}
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="commit-message">Commit Message *</Label>
            <Input
              id="commit-message"
              placeholder="Update feature specification"
              value={commitMessage}
              onChange={(e) => setCommitMessage(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && commitMessage.trim()) {
                  handleCommit();
                }
              }}
              autoFocus
            />
          </div>
          
          {gitStatus && (
            <div className="text-xs text-muted-foreground bg-muted p-2 rounded">
              <div className="font-medium mb-1">Changes to commit:</div>
              <div className="space-y-0.5">
                <div>Files: {gitStatus.uncommittedFiles}</div>
                <div className="text-green-600">+{gitStatus.totalAdded} lines</div>
                {gitStatus.totalRemoved > 0 && (
                  <div className="text-red-600">-{gitStatus.totalRemoved} lines</div>
                )}
              </div>
            </div>
          )}
        </div>
        
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => {
              setCommitModalOpen(false);
              setCommitMessage("");
            }}
          >
            Cancel
          </Button>
          <Button
            onClick={handleCommit}
            disabled={!commitMessage.trim() || committing}
          >
            {committing ? (
              <>
                <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                Committing...
              </>
            ) : (
              'Commit'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
    </>
  );
}

