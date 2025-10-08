"use client";

import React from "react";
import { Button } from "@/components/ui/button";
import { RefreshCw } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { FileTree, type FileTreeNode } from "@/components/file-tree";
import type { AgenticSession } from "@/types/agentic-session";

export type WorkspaceTabProps = {
  session: AgenticSession;
  wsLoading: boolean;
  wsUnavailable: boolean;
  wsTree: FileTreeNode[];
  wsSelectedPath?: string;
  wsFileContent: string;
  onRefresh: (background?: boolean) => void;
  onSelect: (node: FileTreeNode) => void;
  onToggle: (node: FileTreeNode) => void;
  onSave: (path: string, content: string) => Promise<void>;
  setWsFileContent: (v: string) => void;
};

const WorkspaceTab: React.FC<WorkspaceTabProps> = ({ session, wsLoading, wsUnavailable, wsTree, wsSelectedPath, wsFileContent, onRefresh, onSelect, onToggle, onSave, setWsFileContent }) => {
  if (wsLoading) {
    return (
      <div className="flex items-center justify-center h-32 text-sm text-muted-foreground">
        <RefreshCw className="animate-spin h-4 w-4 mr-2" /> Loading workspace...
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
    <div className="grid grid-cols-1 md:grid-cols-2 gap-0">
      <div className="border rounded-md overflow-hidden">
        <div className="p-3 border-b flex items-center justify-between">
          <div>
            <h3 className="font-medium text-sm">Files</h3>
            <p className="text-xs text-muted-foreground">{wsTree.length} items</p>
          </div>
          <Button size="sm" variant="outline" onClick={() => onRefresh(false)} disabled={wsLoading} className="h-8">
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
        <div className="p-2">
          <FileTree nodes={wsTree} selectedPath={wsSelectedPath} onSelect={onSelect} onToggle={onToggle} />
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
                    <Button size="sm" onClick={async () => { await onSave(wsSelectedPath, wsFileContent); }}>Save</Button>
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
  );
};

export default WorkspaceTab;


