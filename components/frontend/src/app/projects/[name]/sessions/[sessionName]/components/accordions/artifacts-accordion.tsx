"use client";

import { Folder, NotepadText, Download, FolderSync, Loader2 } from "lucide-react";
import { AccordionItem, AccordionTrigger, AccordionContent } from "@/components/ui/accordion";
import { Button } from "@/components/ui/button";
import { FileTree, type FileTreeNode } from "@/components/file-tree";

type WorkspaceFile = {
  name: string;
  path: string;
  isDir: boolean;
  size?: number;
};

type ArtifactsAccordionProps = {
  files: WorkspaceFile[];
  currentSubPath: string;
  viewingFile: { path: string; content: string } | null;
  isLoadingFile: boolean;
  onFileOrFolderSelect: (node: FileTreeNode) => void;
  onRefresh: () => void;
  onDownloadFile: () => void;
  onNavigateBack: () => void;
};

export function ArtifactsAccordion({
  files,
  currentSubPath,
  viewingFile,
  isLoadingFile,
  onFileOrFolderSelect,
  onRefresh,
  onDownloadFile,
  onNavigateBack,
}: ArtifactsAccordionProps) {
  return (
    <AccordionItem value="artifacts" className="border rounded-lg px-3 bg-card">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <NotepadText className="h-4 w-4" />
          <span>Artifacts</span>
        </div>
      </AccordionTrigger>
      <AccordionContent className="pt-2 pb-3">
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Artifacts created by the AI will be added here.
          </p>
          
          {/* File Browser for Artifacts */}
          <div className="border rounded-lg overflow-hidden">
            {/* Header with breadcrumbs and actions */}
            <div className="px-2 py-1.5 border-b flex items-center justify-between bg-muted/30">
              <div className="flex items-center gap-1 text-xs text-muted-foreground min-w-0 flex-1">
                {/* Back button when in subfolder or viewing file */}
                {(currentSubPath || viewingFile) && (
                  <Button 
                    variant="ghost" 
                    size="sm" 
                    onClick={onNavigateBack}
                    className="h-6 px-1.5 mr-1"
                  >
                    ← Back
                  </Button>
                )}
                
                {/* Breadcrumb path */}
                <Folder className="inline h-3 w-3 mr-1 flex-shrink-0" />
                <code className="bg-muted px-1 py-0.5 rounded text-xs truncate">
                  artifacts
                  {currentSubPath && `/${currentSubPath}`}
                  {viewingFile && `/${viewingFile.path}`}
                </code>
              </div>

              {/* Action buttons */}
              {viewingFile ? (
                /* Download button when viewing file */
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={onDownloadFile}
                  className="h-6 px-2 flex-shrink-0"
                  title="Download file"
                >
                  <Download className="h-3 w-3" />
                </Button>
              ) : (
                /* Refresh button when not viewing file */
                <Button 
                  variant="ghost" 
                  size="sm" 
                  onClick={onRefresh} 
                  className="h-6 px-2 flex-shrink-0"
                >
                  <FolderSync className="h-3 w-3" />
                </Button>
              )}
            </div>
            
            {/* Content area */}
            <div className="p-2 max-h-64 overflow-y-auto">
              {isLoadingFile ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : viewingFile ? (
                /* File content view */
                <div className="text-xs">
                  <pre className="bg-muted/50 p-2 rounded overflow-x-auto">
                    <code>{viewingFile.content}</code>
                  </pre>
                </div>
              ) : files.length === 0 ? (
                /* Empty state */
                <div className="text-center py-4 text-sm text-muted-foreground">
                  <NotepadText className="h-8 w-8 mx-auto mb-2 opacity-30" />
                  <p>No artifacts yet</p>
                  <p className="text-xs mt-1">AI-generated artifacts will appear here</p>
                </div>
              ) : (
                /* File tree */
                <FileTree 
                  nodes={files.map((item): FileTreeNode => ({
                    name: item.name,
                    path: item.path,
                    type: item.isDir ? 'folder' : 'file',
                    sizeKb: item.size ? item.size / 1024 : undefined,
                  }))}
                  onSelect={onFileOrFolderSelect}
                />
              )}
            </div>
          </div>
        </div>
      </AccordionContent>
    </AccordionItem>
  );
}

