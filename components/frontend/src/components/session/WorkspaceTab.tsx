"use client";

import React from "react";
import { Button } from "@/components/ui/button";
import { RefreshCw, FolderOpen, HardDrive } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { FileTree, type FileTreeNode } from "@/components/file-tree";
import type { AgenticSession } from "@/types/agentic-session";
import { EmptyState } from "@/components/empty-state";

export type WorkspaceTabProps = {
  session: AgenticSession;
  wsLoading: boolean;
  wsUnavailable: boolean;
  wsTree: FileTreeNode[];
  wsSelectedPath?: string;
  onRefresh: (background?: boolean) => void;
  onSelect: (node: FileTreeNode) => void;
  onToggle: (node: FileTreeNode) => void;
  k8sResources?: {
    pvcName?: string;
    pvcExists?: boolean;
    pvcSize?: string;
  };
  contentPodError?: string | null;
  onRetrySpawn?: () => void;
};

const WorkspaceTab: React.FC<WorkspaceTabProps> = ({ session, wsLoading, wsUnavailable, wsTree, wsSelectedPath, onRefresh, onSelect, onToggle, k8sResources, contentPodError, onRetrySpawn }) => {
  if (wsLoading) {
    return (
      <div className="flex items-center justify-center h-32 text-sm text-muted-foreground">
        <RefreshCw className="animate-spin h-4 w-4 mr-2" /> Loading workspace...
      </div>
    );
  }
  
  // Show error with retry button if content pod failed to spawn
  if (contentPodError) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-sm text-center p-6">
        <div className="text-destructive font-medium mb-2">Workspace Viewer Error</div>
        <div className="text-muted-foreground mb-4 max-w-md">{contentPodError}</div>
        {onRetrySpawn && (
          <Button onClick={onRetrySpawn} variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-2" /> Retry
          </Button>
        )}
      </div>
    );
  }
  
  if (wsUnavailable) {
    return (
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
    );
  }
  return (
    <div className="grid grid-cols-1 gap-0">
      <div className="border rounded-md overflow-hidden">
        <div className="p-3 border-b flex items-center justify-between">
          <div className="flex-1">
            {k8sResources?.pvcName ? (
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-xs">
                  <HardDrive className="w-3 h-3 mr-1" />
                  PVC
                </Badge>
                <span className="font-mono text-xs text-muted-foreground">{k8sResources.pvcName}</span>
                <Badge className={`text-xs ${k8sResources.pvcExists ? 'bg-green-100 text-green-800 border-green-300 dark:bg-green-700 dark:text-white dark:border-green-700' : 'bg-red-100 text-red-800 border-red-300 dark:bg-red-700 dark:text-white dark:border-red-700'}`}>
                  {k8sResources.pvcExists ? 'Exists' : 'Not Found'}
                </Badge>
                {k8sResources.pvcSize && (
                  <span className="text-xs text-muted-foreground">{k8sResources.pvcSize}</span>
                )}
              </div>
            ) : (
              <p className="text-xs text-muted-foreground">{wsTree.length} items</p>
            )}
          </div>
          <Button size="sm" variant="outline" onClick={() => onRefresh(false)} disabled={wsLoading} className="h-8">
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
        <div className="p-2">
          {wsTree.length === 0 ? (
            <EmptyState
              icon={FolderOpen}
              title="No files yet"
              description="The workspace is empty. Files will appear here as the session progresses."
            />
          ) : (
            <FileTree nodes={wsTree} selectedPath={wsSelectedPath} onSelect={onSelect} onToggle={onToggle} />
          )}
        </div>
      </div>
      {/* TODO: Artifact/File Viewer - Temporarily hidden until Artifact Viewer feature is implemented
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
                    <Button size="sm" onClick={async () => { await onSave(wsSelectedPath, wsFileContent); }}>Save</Button>
                  </div>
                </div>
                <textarea
                  className="w-full h-[60vh] bg-slate-950 dark:bg-black text-slate-50 p-4 rounded overflow-auto text-sm font-mono"
                  value={wsFileContent}
                  onChange={(e) => setWsFileContent(e.target.value)}
                />
              </>
            ) : (
              <EmptyState
                icon={FileText}
                title="No file selected"
                description="Select a file from the tree to view and edit its contents."
              />
            )}
          </CardContent>
        </Card>
      </div>
      */}
    </div>
  );
};

export default WorkspaceTab;


