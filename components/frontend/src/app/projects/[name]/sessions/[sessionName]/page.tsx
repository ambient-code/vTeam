"use client";

import { useState, useEffect, useMemo, useRef } from "react";
import { Loader2, FolderTree, GitBranch, Edit, RefreshCw, Folder, Sparkles, X, CloudUpload, CloudDownload, MoreVertical, Cloud, FolderSync, Download, LibraryBig, MessageSquare } from "lucide-react";
import { useRouter } from "next/navigation";

// Custom components
import MessagesTab from "@/components/session/MessagesTab";
import { FileTree, type FileTreeNode } from "@/components/file-tree";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Label } from "@/components/ui/label";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { SessionHeader } from "./session-header";

// Extracted components
import { AddContextModal } from "./components/modals/add-context-modal";
import { CustomWorkflowDialog } from "./components/modals/custom-workflow-dialog";
import { ManageRemoteDialog } from "./components/modals/manage-remote-dialog";
import { CommitChangesDialog } from "./components/modals/commit-changes-dialog";
import { WorkflowsAccordion } from "./components/accordions/workflows-accordion";
import { RepositoriesAccordion } from "./components/accordions/repositories-accordion";
import { ArtifactsAccordion } from "./components/accordions/artifacts-accordion";

// Extracted hooks and utilities
import { useGitOperations } from "./hooks/use-git-operations";
import { useWorkflowManagement } from "./hooks/use-workflow-management";
import { useFileOperations } from "./hooks/use-file-operations";
import { adaptSessionMessages } from "./lib/message-adapter";
import type { DirectoryOption, DirectoryRemote } from "./lib/types";

import type { SessionMessage } from "@/types";
import type { MessageObject, ToolUseMessages } from "@/types/agentic-session";

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
import { useWorkspaceList, useGitMergeStatus, useGitListBranches } from "@/services/queries/use-workspace";
import { successToast, errorToast } from "@/hooks/use-toast";
import { useOOTBWorkflows, useWorkflowMetadata } from "@/services/queries/use-workflows";
import { useMutation } from "@tanstack/react-query";

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
  const [autoSelectAgents, setAutoSelectAgents] = useState(true);
  const [openAccordionItems, setOpenAccordionItems] = useState<string[]>(["workflows"]);
  const [contextModalOpen, setContextModalOpen] = useState(false);
  const [repoChanging, setRepoChanging] = useState(false);
  const [firstMessageLoaded, setFirstMessageLoaded] = useState(false);
  
  // Directory browser state (unified for artifacts, repos, and workflow)
  const [selectedDirectory, setSelectedDirectory] = useState<DirectoryOption>({
    type: 'artifacts',
    name: 'Shared Artifacts',
    path: 'artifacts'
  });
  const [directoryRemotes, setDirectoryRemotes] = useState<Record<string, DirectoryRemote>>({});
  const [remoteDialogOpen, setRemoteDialogOpen] = useState(false);
  const [commitModalOpen, setCommitModalOpen] = useState(false);
  const [customWorkflowDialogOpen, setCustomWorkflowDialogOpen] = useState(false);

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

  // React Query hooks
  const { data: session, isLoading, error, refetch: refetchSession } = useSession(projectName, sessionName);
  const { data: messages = [] } = useSessionMessages(projectName, sessionName, session?.status?.phase);
  const { data: k8sResources } = useSessionK8sResources(projectName, sessionName);
  const stopMutation = useStopSession();
  const deleteMutation = useDeleteSession();
  const continueMutation = useContinueSession();
  const sendChatMutation = useSendChatMessage();
  const sendControlMutation = useSendControlMessage();
  
  // Workflow management hook
  const workflowManagement = useWorkflowManagement({
    projectName,
    sessionName,
    onWorkflowActivated: refetchSession,
  });
  
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
      return { ...result, inputRepo: repo };
    },
    onSuccess: async (data) => {
      successToast('Repository cloning...');
      await new Promise(resolve => setTimeout(resolve, 3000));
      await refetchSession();
      
      if (data.name && data.inputRepo) {
        try {
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
          
          const newRemotes = {...directoryRemotes};
          newRemotes[data.name] = {
            url: data.inputRepo.url,
            branch: data.inputRepo.branch || 'main'
          };
          setDirectoryRemotes(newRemotes);
        } catch (err) {
          console.error('Failed to configure remote:', err);
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
  
  // Fetch OOTB workflows
  const { data: ootbWorkflows = [] } = useOOTBWorkflows(projectName);
  
  // Fetch workflow metadata
  const { data: workflowMetadata } = useWorkflowMetadata(
    projectName,
    sessionName,
    !!workflowManagement.activeWorkflow && !workflowManagement.workflowActivating
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
  
  // Git operations hook
  const gitOps = useGitOperations({
    projectName,
    sessionName,
    directoryPath: selectedDirectory.path,
    remoteBranch: currentRemote?.branch || 'main',
  });
  
  // File operations for directory explorer
  const fileOps = useFileOperations({
    projectName,
    sessionName,
    basePath: selectedDirectory.path,
  });
  
  const { data: directoryFiles = [], refetch: refetchDirectoryFiles } = useWorkspaceList(
    projectName,
    sessionName,
    fileOps.currentSubPath ? `${selectedDirectory.path}/${fileOps.currentSubPath}` : selectedDirectory.path,
    { enabled: openAccordionItems.includes("directories") }
  );
  
  // Artifacts file operations
  const artifactsOps = useFileOperations({
    projectName,
    sessionName,
    basePath: 'artifacts',
  });
  
  const { data: artifactsFiles = [], refetch: refetchArtifactsFiles } = useWorkspaceList(
    projectName,
    sessionName,
    artifactsOps.currentSubPath ? `artifacts/${artifactsOps.currentSubPath}` : 'artifacts',
    { enabled: openAccordionItems.includes("artifacts") }
  );
  
  // Track if we've already initialized from session
  const initializedFromSessionRef = useRef(false);
  
  // Track when first message loads
  useEffect(() => {
    if (messages && messages.length > 0 && !firstMessageLoaded) {
      setFirstMessageLoaded(true);
    }
  }, [messages, firstMessageLoaded]);
  
  // Load active workflow and remotes from session
  useEffect(() => {
    if (initializedFromSessionRef.current || !session) return;
    
    if (session.spec?.activeWorkflow && ootbWorkflows.length === 0) {
      return;
    }
    
    if (session.spec?.activeWorkflow) {
      const gitUrl = session.spec.activeWorkflow.gitUrl;
      const matchingWorkflow = ootbWorkflows.find(w => w.gitUrl === gitUrl);
      if (matchingWorkflow) {
        workflowManagement.setActiveWorkflow(matchingWorkflow.id);
        workflowManagement.setSelectedWorkflow(matchingWorkflow.id);
      } else {
        workflowManagement.setActiveWorkflow("custom");
        workflowManagement.setSelectedWorkflow("custom");
      }
    }
    
    // Load remotes from annotations
    const annotations = session.metadata?.annotations || {};
    const remotes: Record<string, DirectoryRemote> = {};
    
    Object.keys(annotations).forEach(key => {
      if (key.startsWith('ambient-code.io/remote-') && key.endsWith('-url')) {
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
  }, [session, ootbWorkflows, workflowManagement]);

  // Compute directory options
  const directoryOptions = useMemo<DirectoryOption[]>(() => {
    const options: DirectoryOption[] = [
      { type: 'artifacts', name: 'Shared Artifacts', path: 'artifacts' }
    ];
    
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
    
    if (workflowManagement.activeWorkflow && session?.spec?.activeWorkflow) {
      const workflowName = session.spec.activeWorkflow.gitUrl.split('/').pop()?.replace('.git', '') || 'workflow';
      options.push({
        type: 'workflow',
        name: `Workflow: ${workflowName}`,
        path: `workflows/${workflowName}`
      });
    }
    
    return options;
  }, [session, workflowManagement.activeWorkflow]);

  // Workflow change handler
  const handleWorkflowChange = (value: string) => {
    workflowManagement.handleWorkflowChange(
      value,
      ootbWorkflows,
      () => setCustomWorkflowDialogOpen(true)
    );
  };

  // Convert messages using extracted adapter
  const streamMessages: Array<MessageObject | ToolUseMessages> = useMemo(() => {
    return adaptSessionMessages(messages as SessionMessage[], session?.spec?.interactive || false);
  }, [messages, session?.spec?.interactive]);

  // Session action handlers
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
          setSelectedAgents([]);
        },
        onError: (err) => errorToast(err instanceof Error ? err.message : "Failed to send message"),
      }
    );
  };

  const handleCommandClick = (slashCommand: string) => {
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
          setSelectedAgents([]);
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

  // Auto-spawn content pod on completed session
  const sessionCompleted = (
    session?.status?.phase === 'Completed' ||
    session?.status?.phase === 'Failed' ||
    session?.status?.phase === 'Stopped'
  );

  useEffect(() => {
    if (sessionCompleted && !contentPodReady && !contentPodSpawning && !contentPodError) {
      spawnContentPodAsync();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionCompleted, contentPodReady, contentPodSpawning, contentPodError]);

  const spawnContentPodAsync = async () => {
    if (!projectName || !sessionName) return;
    
    setContentPodSpawning(true);
    setContentPodError(null);
    
    try {
      const { spawnContentPod, getContentPodStatus } = await import('@/services/api/sessions');
      
      const spawnResult = await spawnContentPod(projectName, sessionName);
      
      if (spawnResult.status === 'exists' && spawnResult.ready) {
        setContentPodReady(true);
        setContentPodSpawning(false);
        setContentPodError(null);
        return;
      }
      
      let attempts = 0;
      const maxAttempts = 30;
      
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

  // Loading state
  if (isLoading || !projectName || !sessionName) {
    return (
      <div className="absolute inset-0 top-16 overflow-hidden bg-[#f8fafc] flex items-center justify-center">
        <div className="flex items-center">
          <div className="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full" />
          <span className="ml-2">Loading session...</span>
        </div>
      </div>
    );
  }

  // Error state
  if (error || !session) {
    return (
      <div className="absolute inset-0 top-16 overflow-hidden bg-[#f8fafc] flex flex-col">
        <div className="flex-shrink-0 bg-white border-b">
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
          </div>
        </div>
        <div className="flex-grow overflow-hidden">
          <div className="h-full container mx-auto px-6 py-6">
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
      <div className="absolute inset-0 top-16 overflow-hidden bg-[#f8fafc] flex flex-col">
        {/* Fixed header */}
        <div className="flex-shrink-0 bg-white border-b">
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
                continueMutation.isPending ? "resuming" :
                null
              }
              onRefresh={refetchSession}
              onStop={handleStop}
              onContinue={handleContinue}
              onDelete={handleDelete}
              durationMs={durationMs}
              k8sResources={k8sResources}
              messageCount={messages.length}
            />
          </div>
        </div>

        {/* Main content area */}
        <div className="flex-grow overflow-hidden">
          <div className="h-full container mx-auto px-6 py-6">
            <div className="h-full flex gap-6">
              {/* Left Column - Accordions */}
              <div className="w-2/5 flex flex-col min-w-0 relative">
                {/* Blocking overlay when first message hasn't loaded and session is pending */}
                {!firstMessageLoaded && session?.status?.phase === 'Pending' && (
                  <div className="absolute inset-0 bg-white/60 backdrop-blur-sm rounded-lg z-20 flex items-center justify-center">
                    <div className="flex flex-col items-center justify-center text-center text-muted-foreground">
                      <LibraryBig className="w-8 h-8 mx-auto mb-2 opacity-50" />
                      <div className="flex items-center gap-2">
                        <Loader2 className="h-4 w-4 animate-spin text-blue-600" />
                        <p className="text-sm">No context yet</p>
                      </div>
                      <p className="text-xs mt-1">Context will appear once the session starts...</p>
                    </div>
                  </div>
                )}
                <div className={`overflow-y-auto flex-grow pb-6 ${!firstMessageLoaded && session?.status?.phase === 'Pending' ? 'pointer-events-none opacity-50' : ''}`}>
                  <Accordion type="multiple" value={openAccordionItems} onValueChange={setOpenAccordionItems} className="w-full space-y-3">
                    <WorkflowsAccordion
                      sessionPhase={session?.status?.phase}
                      activeWorkflow={workflowManagement.activeWorkflow}
                      selectedWorkflow={workflowManagement.selectedWorkflow}
                      pendingWorkflow={workflowManagement.pendingWorkflow}
                      workflowActivating={workflowManagement.workflowActivating}
                      workflowMetadata={workflowMetadata}
                      ootbWorkflows={ootbWorkflows}
                      selectedAgents={selectedAgents}
                      autoSelectAgents={autoSelectAgents}
                      isExpanded={openAccordionItems.includes("workflows")}
                      onWorkflowChange={handleWorkflowChange}
                      onActivateWorkflow={workflowManagement.activateWorkflow}
                      onCommandClick={handleCommandClick}
                      onSetSelectedAgents={setSelectedAgents}
                      onSetAutoSelectAgents={setAutoSelectAgents}
                      onResume={handleContinue}
                    />

                    <RepositoriesAccordion
                      repositories={session?.spec?.repos || []}
                      onAddRepository={() => setContextModalOpen(true)}
                      onRemoveRepository={(repoName) => removeRepoMutation.mutate(repoName)}
                    />

                    <ArtifactsAccordion
                      files={artifactsFiles}
                      currentSubPath={artifactsOps.currentSubPath}
                      viewingFile={artifactsOps.viewingFile}
                      isLoadingFile={artifactsOps.loadingFile}
                      onFileOrFolderSelect={artifactsOps.handleFileOrFolderSelect}
                      onRefresh={refetchArtifactsFiles}
                      onDownloadFile={artifactsOps.handleDownloadFile}
                      onNavigateBack={artifactsOps.navigateBack}
                    />

                    {/* Experimental - File Explorer */}
                    <AccordionItem value="experimental" className="border rounded-lg px-3 bg-white">
                      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                        <div className="flex items-center gap-2">
                          <Sparkles className="h-4 w-4" />
                          <span>Experimental</span>
                        </div>
                      </AccordionTrigger>
                      <AccordionContent className="pt-2 pb-3">
                        <div className="space-y-3">
                          <Accordion 
                            type="multiple" 
                            value={openAccordionItems} 
                            onValueChange={setOpenAccordionItems}
                          >
                            <AccordionItem value="directories" className="border rounded-lg px-3 bg-muted/10">
                              <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
                                <div className="flex items-center gap-2 w-full">
                                  <Folder className="h-4 w-4" />
                                  <span>File Explorer</span>
                                  {gitOps.gitStatus?.hasChanges && (
                                    <div className="flex gap-1 ml-auto mr-2">
                                      {(gitOps.gitStatus?.totalAdded ?? 0) > 0 && (
                                        <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                                          +{gitOps.gitStatus.totalAdded}
                                        </Badge>
                                      )}
                                      {(gitOps.gitStatus?.totalRemoved ?? 0) > 0 && (
                                        <Badge variant="outline" className="bg-red-50 text-red-700 border-red-200">
                                          -{gitOps.gitStatus.totalRemoved}
                                        </Badge>
                                      )}
                                    </div>
                                  )}
                                </div>
                              </AccordionTrigger>
                              <AccordionContent className="pt-2 pb-3">
                                <div className="space-y-3">
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
                                  
                                  {/* File Browser */}
                                  <div className="border rounded-lg overflow-hidden">
                                    <div className="px-2 py-1.5 border-b flex items-center justify-between bg-muted/30">
                                      <div className="flex items-center gap-1 text-xs text-muted-foreground min-w-0 flex-1">
                                        {(fileOps.currentSubPath || fileOps.viewingFile) && (
                                          <Button 
                                            variant="ghost" 
                                            size="sm" 
                                            onClick={fileOps.navigateBack}
                                            className="h-6 px-1.5 mr-1"
                                          >
                                            ← Back
                                          </Button>
                                        )}
                                        
                                        <Folder className="inline h-3 w-3 mr-1 flex-shrink-0" />
                                        <code className="bg-muted px-1 py-0.5 rounded text-xs truncate">
                                          {selectedDirectory.path}
                                          {fileOps.currentSubPath && `/${fileOps.currentSubPath}`}
                                          {fileOps.viewingFile && `/${fileOps.viewingFile.path}`}
                                        </code>
                                      </div>

                                      {fileOps.viewingFile ? (
                                        <div className="flex items-center gap-1">
                                          <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={fileOps.handleDownloadFile}
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
                                        <Button variant="ghost" size="sm" onClick={() => refetchDirectoryFiles()} className="h-6 px-2 flex-shrink-0">
                                          <FolderSync className="h-3 w-3" />
                                        </Button>
                                      )}
                                    </div>
                                    
                                    <div className="p-2 max-h-64 overflow-y-auto">
                                      {fileOps.loadingFile ? (
                                        <div className="flex items-center justify-center py-8">
                                          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                                        </div>
                                      ) : fileOps.viewingFile ? (
                                        <div className="text-xs">
                                          <pre className="bg-muted/50 p-2 rounded overflow-x-auto">
                                            <code>{fileOps.viewingFile.content}</code>
                                          </pre>
                                        </div>
                                      ) : directoryFiles.length === 0 ? (
                                        <div className="text-center py-4 text-sm text-muted-foreground">
                                          <FolderTree className="h-8 w-8 mx-auto mb-2 opacity-30" />
                                          <p>No files yet</p>
                                          <p className="text-xs mt-1">Files will appear here</p>
                                        </div>
                                      ) : (
                                        <FileTree 
                                          nodes={directoryFiles.map((item): FileTreeNode => ({
                                            name: item.name,
                                            path: item.path,
                                            type: item.isDir ? 'folder' : 'file',
                                            sizeKb: item.size ? item.size / 1024 : undefined,
                                          }))}
                                          onSelect={fileOps.handleFileOrFolderSelect}
                                        />
                                      )}
                                    </div>
                                  </div>
                                  
                                  {/* Remote Configuration */}
                                  {!currentRemote ? (
                                    <div className="border border-blue-200 bg-blue-50 rounded-md px-3 py-2 flex items-center justify-between">
                                      <span className="text-sm text-blue-800">Set up Git remote for version control</span>
                                      <Button onClick={() => setRemoteDialogOpen(true)} size="sm" variant="outline">
                                        <GitBranch className="mr-2 h-3 w-3" />
                                        Configure
                                      </Button>
                                    </div>
                                  ) : (
                                    <div className="border rounded-md px-2 py-1.5">
                                      <div className="flex items-center gap-2 text-xs">
                                        <div className="flex items-center gap-1.5 text-muted-foreground">
                                          <Cloud className="h-3 w-3" />
                                          <span className="truncate max-w-[200px]">
                                            {currentRemote?.url?.split('/').slice(-2).join('/').replace('.git', '') || ''}/{currentRemote?.branch || 'main'}
                                          </span>
                                        </div>
                                        
                                        <div className="flex-1" />
                                        
                                        {mergeStatus && !mergeStatus.canMergeClean ? (
                                          <div className="flex items-center gap-1 text-red-600">
                                            <X className="h-3 w-3" />
                                            <span className="font-medium">conflict</span>
                                          </div>
                                        ) : (gitOps.gitStatus?.hasChanges || mergeStatus?.remoteCommitsAhead) ? (
                                          <div className="flex items-center gap-1.5 text-muted-foreground text-xs">
                                            {mergeStatus?.remoteCommitsAhead ? (
                                              <span>↓{mergeStatus.remoteCommitsAhead}</span>
                                            ) : null}
                                            {gitOps.gitStatus?.hasChanges ? (
                                              <span className="font-normal">{gitOps.gitStatus?.uncommittedFiles ?? 0} uncommitted</span>
                                            ) : null}
                                          </div>
                                        ) : null}
                                        
                                        <TooltipProvider>
                                          <Tooltip>
                                            <TooltipTrigger asChild>
                                              <Button 
                                                size="sm"
                                                variant="ghost"
                                                onClick={() => gitOps.handleGitSynchronize(refetchMergeStatus)}
                                                disabled={!mergeStatus?.canMergeClean || gitOps.synchronizing || gitOps.gitStatus?.hasChanges}
                                                className="h-6 w-6 p-0"
                                              >
                                                {gitOps.synchronizing ? (
                                                  <Loader2 className="h-3 w-3 animate-spin" />
                                                ) : (
                                                  <RefreshCw className="h-3 w-3" />
                                                )}
                                              </Button>
                                            </TooltipTrigger>
                                            <TooltipContent>
                                              <p>{gitOps.gitStatus?.hasChanges ? 'Commit changes first' : `Sync with origin/${currentRemote?.branch || 'main'}`}</p>
                                            </TooltipContent>
                                          </Tooltip>
                                        </TooltipProvider>

                                        <DropdownMenu>
                                          <DropdownMenuTrigger asChild>
                                            <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                                              <MoreVertical className="h-3 w-3" />
                                            </Button>
                                          </DropdownMenuTrigger>
                                          <DropdownMenuContent align="end">
                                            <DropdownMenuItem onClick={() => setRemoteDialogOpen(true)}>
                                              <Edit className="mr-2 h-3 w-3" />
                                              Manage Remote
                                            </DropdownMenuItem>
                                            <DropdownMenuSeparator />
                                            <DropdownMenuItem
                                              onClick={() => setCommitModalOpen(true)}
                                              disabled={!gitOps.gitStatus?.hasChanges}
                                            >
                                              <Edit className="mr-2 h-3 w-3" />
                                              Commit Changes
                                            </DropdownMenuItem>
                                            <DropdownMenuItem
                                              onClick={() => gitOps.handleGitPull(refetchMergeStatus)}
                                              disabled={!mergeStatus?.canMergeClean || gitOps.isPulling}
                                            >
                                              <CloudDownload className="mr-2 h-3 w-3" />
                                              Pull
                                            </DropdownMenuItem>
                                            <DropdownMenuItem
                                              onClick={() => gitOps.handleGitPush(refetchMergeStatus)}
                                              disabled={!mergeStatus?.canMergeClean || gitOps.isPushing || gitOps.gitStatus?.hasChanges}
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
                      </AccordionContent>
                    </AccordionItem>
                  </Accordion>
                </div>
              </div>

              {/* Right Column - Messages */}
              <div className="flex-1 flex flex-col min-w-0">
                <Card className="relative flex-1 flex flex-col overflow-hidden py-4">
                  <CardContent className="px-3 pt-3 pb-0 flex-1 flex flex-col overflow-hidden">
                    {/* Workflow activation overlay */}
                    {workflowManagement.workflowActivating && (
                      <div className="absolute inset-0 bg-white/90 backdrop-blur-sm z-10 flex items-center justify-center rounded-lg">
                        <Alert className="max-w-md mx-4">
                          <Loader2 className="h-4 w-4 animate-spin" />
                          <AlertTitle>Activating Workflow...</AlertTitle>
                          <AlertDescription>
                            <p>The new workflow is being loaded. Please wait...</p>
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
                              <p>Please wait while repositories are being updated. This may take 10-20 seconds...</p>
                            </div>
                          </AlertDescription>
                        </Alert>
                      </div>
                    )}
                    
                    {/* Session starting overlay */}
                    {!firstMessageLoaded && session?.status?.phase === 'Pending' && (
                      <div className="absolute inset-0 bg-white/60 backdrop-blur-sm rounded-lg z-20 flex items-center justify-center">
                        <div className="flex flex-col items-center justify-center text-center text-muted-foreground">
                          <MessageSquare className="w-8 h-8 mx-auto mb-2 opacity-50" />
                          <div className="flex items-center gap-2">
                            <Loader2 className="h-4 w-4 animate-spin text-blue-600" />
                            <p className="text-sm">No messages yet</p>
                          </div>
                          <p className="text-xs mt-1">Messages will appear once the session starts...</p>
                        </div>
                      </div>
                    )}
                    
                    <div className={`flex flex-col flex-1 overflow-hidden ${!firstMessageLoaded && session?.status?.phase === 'Pending' ? 'pointer-events-none opacity-50' : ''}`}>
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
                        workflowMetadata={workflowMetadata}
                        onSetSelectedAgents={setSelectedAgents}
                        onSetAutoSelectAgents={setAutoSelectAgents}
                        onCommandClick={handleCommandClick}
                      />
                    </div>
                  </CardContent>
                </Card>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Modals */}
      <AddContextModal
        open={contextModalOpen}
        onOpenChange={setContextModalOpen}
        onAddRepository={async (url, branch) => {
          await addRepoMutation.mutateAsync({ url, branch });
          setContextModalOpen(false);
        }}
        isLoading={addRepoMutation.isPending}
      />

      <CustomWorkflowDialog
        open={customWorkflowDialogOpen}
        onOpenChange={setCustomWorkflowDialogOpen}
        onSubmit={(url, branch, path) => {
          workflowManagement.setCustomWorkflow(url, branch, path);
          setCustomWorkflowDialogOpen(false);
        }}
        isActivating={workflowManagement.workflowActivating}
      />

      <ManageRemoteDialog
        open={remoteDialogOpen}
        onOpenChange={setRemoteDialogOpen}
        onSave={async (url, branch) => {
          const success = await gitOps.configureRemote(url, branch);
          if (success) {
            const newRemotes = {...directoryRemotes};
            newRemotes[selectedDirectory.path] = { url, branch };
            setDirectoryRemotes(newRemotes);
            setRemoteDialogOpen(false);
            refetchMergeStatus();
          }
        }}
        directoryName={selectedDirectory.name}
        currentUrl={currentRemote?.url}
        currentBranch={currentRemote?.branch}
        remoteBranches={remoteBranches}
        mergeStatus={mergeStatus}
        isLoading={gitOps.isConfiguringRemote}
      />

      <CommitChangesDialog
        open={commitModalOpen}
        onOpenChange={setCommitModalOpen}
        onCommit={async (message) => {
          const success = await gitOps.handleCommit(message);
          if (success) {
            setCommitModalOpen(false);
            refetchMergeStatus();
          }
        }}
        gitStatus={gitOps.gitStatus ?? null}
        directoryName={selectedDirectory.name}
        isCommitting={gitOps.committing}
      />
    </>
  );
}
