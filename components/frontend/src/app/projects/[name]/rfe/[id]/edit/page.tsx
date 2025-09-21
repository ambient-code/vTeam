"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { RFEWorkflow, ArtifactFile } from "@/types/agentic-session";
import { ArrowLeft, Save, Upload, FileText, Download, RefreshCw, Loader2, AlertCircle } from "lucide-react";
import { FileTree, type FileTreeNode } from "@/components/file-tree";
import { getApiUrl } from "@/lib/config";
import { getAgentByPersona } from "@/lib/agents";
import { Textarea } from "@/components/ui/textarea";
import { useToast } from "@/hooks/use-toast";

function buildFileTree(artifacts: ArtifactFile[]): FileTreeNode[] {
  const tree: FileTreeNode[] = [];
  const nodeMap = new Map<string, FileTreeNode>();

  // Create root folders
  const phases = new Set((artifacts || []).map(a => a.phase).filter(Boolean));
  phases.forEach(phase => {
    if (phase) {
      const node: FileTreeNode = {
        name: phase,
        path: phase,
        type: "folder",
        children: [],
        expanded: true,
      };
      tree.push(node);
      nodeMap.set(phase, node);
    }
  });

  // Add artifacts to their respective folders
  artifacts.forEach(artifact => {
    const phase = artifact.phase || "other";
    const phaseNode = nodeMap.get(phase);

    if (phaseNode) {
      const fileNode: FileTreeNode = {
        name: artifact.name,
        path: artifact.path,
        type: "file",
        data: artifact,
      };
      phaseNode.children!.push(fileNode);
    }
  });

  return tree;
}

export default function ArtifactEditPage() {
  const params = useParams();

  const { toast } = useToast();
  const projectName = params.name as string;
  const workflowId = params.id as string;

  const [workflow, setWorkflow] = useState<RFEWorkflow | null>(null);
  const [selectedFile, setSelectedFile] = useState<ArtifactFile | null>(null);
  const [fileContent, setFileContent] = useState("");
  const [isDirty, setIsDirty] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [fileTree, setFileTree] = useState<FileTreeNode[]>([]);
  const [lastSaved, setLastSaved] = useState<Date | null>(null);

  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const fetchWorkflow = useCallback(async () => {
    try {
      const response = await fetch(`${getApiUrl()}/projects/${projectName}/rfe-workflows/${workflowId}`);
      if (!response.ok) throw new Error("Failed to fetch workflow");

      const data = await response.json();
      setWorkflow(data);
      setFileTree(buildFileTree(data.artifacts));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load workflow");
    } finally {
      setIsLoading(false);
    }
  }, [workflowId, projectName]);

  const selectFile = useCallback(async (artifact: ArtifactFile) => {
    if (isDirty) {
      const confirm = window.confirm("You have unsaved changes. Do you want to continue without saving?");
      if (!confirm) return;
    }

    try {
      // Fetch file content from PVC
      const response = await fetch(`${getApiUrl()}/projects/${projectName}/rfe-workflows/${workflowId}/artifacts/${encodeURIComponent(artifact.path)}`);
      if (!response.ok) throw new Error("Failed to fetch file content");

      const content = await response.text();
      setSelectedFile(artifact);
      setFileContent(content);
      setIsDirty(false);
    } catch {
      toast({
        title: "Error",
        description: "Failed to load file content",
        variant: "destructive",
      });
    }
  }, [isDirty, workflowId, projectName, toast]);

  const saveFile = useCallback(async () => {
    if (!selectedFile || !isDirty) return;

    setIsSaving(true);
    try {
      const response = await fetch(`${getApiUrl()}/projects/${projectName}/rfe-workflows/${workflowId}/artifacts/${encodeURIComponent(selectedFile.path)}`, {
        method: "PUT",
        headers: { "Content-Type": "text/plain" },
        body: fileContent,
      });

      if (!response.ok) throw new Error("Failed to save file");

      setIsDirty(false);
      setLastSaved(new Date());

      toast({
        title: "Success",
        description: "File saved successfully",
      });

      // Refresh workflow to update artifact info
      fetchWorkflow();
    } catch {
      toast({
        title: "Error",
        description: "Failed to save file",
        variant: "destructive",
      });
    } finally {
      setIsSaving(false);
    }
  }, [selectedFile, isDirty, workflowId, projectName, fileContent, toast, fetchWorkflow]);

  const pushToGit = async () => {
    try {
      const response = await fetch(`${getApiUrl()}/projects/${projectName}/rfe-workflows/${workflowId}/push-to-git`, {
        method: "POST",
      });

      if (!response.ok) throw new Error("Failed to push to git");

      toast({
        title: "Success",
        description: "Artifacts pushed to Git repository",
      });

      fetchWorkflow();
    } catch {
      toast({
        title: "Error",
        description: "Failed to push to Git",
        variant: "destructive",
      });
    }
  };

  const downloadFile = () => {
    if (!selectedFile) return;

    const blob = new Blob([fileContent], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = selectedFile.name;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  useEffect(() => {
    fetchWorkflow();
  }, [fetchWorkflow]);

  // Auto-select first file when workflow loads
  useEffect(() => {
    if (workflow && !selectedFile && workflow.artifacts && workflow.artifacts.length > 0) {
      selectFile(workflow.artifacts[0]);
    }
  }, [workflow, selectedFile, selectFile]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "s") {
        e.preventDefault();
        saveFile();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [saveFile]);

  if (isLoading) {
    return (
      <div className="container mx-auto py-8">
        <div className="flex items-center justify-center h-64">
          <Loader2 className="h-8 w-8 animate-spin" />
        </div>
      </div>
    );
  }

  if (error || !workflow) {
    return (
      <div className="container mx-auto py-8">
        <Card className="border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <p className="text-red-600">Error: {error}</p>
            <Link href={`/projects/${projectName}/rfe/${workflowId}`}>
              <Button className="mt-4" variant="outline">
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back to RFE Details
              </Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col">
      {/* Header */}
      <div className="border-b bg-background px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link href={`/projects/${projectName}/rfe/${workflowId}`}>
              <Button variant="ghost" size="sm">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to RFE Details
              </Button>
            </Link>
            <div>
              <h1 className="text-xl font-semibold">Edit Artifacts</h1>
              <p className="text-sm text-muted-foreground">{workflow.title}</p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {isDirty && (
              <Badge variant="outline" className="text-orange-600">
                <AlertCircle className="mr-1 h-3 w-3" />
                Unsaved Changes
              </Badge>
            )}

            {lastSaved && (
              <span className="text-sm text-muted-foreground">
                Last saved: {lastSaved.toLocaleTimeString()}
              </span>
            )}

            <Button
              onClick={saveFile}
              disabled={!selectedFile || !isDirty || isSaving}
              size="sm"
            >
              {isSaving ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Save className="mr-2 h-4 w-4" />
              )}
              Save
            </Button>

            {selectedFile && (
              <Button onClick={downloadFile} variant="outline" size="sm">
                <Download className="mr-2 h-4 w-4" />
                Download
              </Button>
            )}

            <Button onClick={pushToGit} variant="outline" size="sm">
              <Upload className="mr-2 h-4 w-4" />
              Push to Git
            </Button>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 overflow-hidden">
        <ResizablePanelGroup direction="horizontal">
          {/* File Tree */}
          <ResizablePanel defaultSize={25} minSize={20}>
            <div className="h-full border-r bg-muted/30">
              <div className="p-4 border-b">
                <h3 className="font-medium">Artifacts</h3>
                <p className="text-sm text-muted-foreground">
                  {(workflow.artifacts || []).length} files
                </p>
              </div>

              <div className="p-2 space-y-1 overflow-auto">
                <FileTree
                  nodes={fileTree}
                  selectedPath={selectedFile?.path}
                  onSelect={(node) => {
                    const art = (node.data as ArtifactFile) || undefined;
                    if (art) selectFile(art);
                  }}
                />

                {(workflow.artifacts || []).length === 0 && (
                  <div className="text-center py-8 text-muted-foreground">
                    <FileText className="mx-auto h-8 w-8 mb-2" />
                    <p className="text-sm">No artifacts generated yet</p>
                  </div>
                )}
              </div>
            </div>
          </ResizablePanel>

          <ResizableHandle />

          {/* Editor */}
          <ResizablePanel defaultSize={75}>
            <div className="h-full flex flex-col">
              {selectedFile ? (
                <>
                  {/* File Header */}
                  <div className="border-b p-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <h3 className="font-medium">{selectedFile.name}</h3>
                        <div className="flex items-center gap-2 mt-1">
                          {selectedFile.agent && (
                            <Badge variant="outline">
                              {getAgentByPersona(selectedFile.agent)?.name || selectedFile.agent}
                            </Badge>
                          )}
                          {selectedFile.phase && (
                            <Badge variant="outline">{selectedFile.phase}</Badge>
                          )}
                          <span className="text-xs text-muted-foreground">
                            {(selectedFile.size / 1024).toFixed(1)} KB
                          </span>
                        </div>
                      </div>

                      <div className="flex items-center gap-2">
                        <Button onClick={fetchWorkflow} variant="ghost" size="sm">
                          <RefreshCw className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </div>

                  {/* Editor */}
                  <div className="flex-1 p-4">
                    <Textarea
                      ref={textareaRef}
                      value={fileContent}
                      onChange={(e) => {
                        setFileContent(e.target.value);
                        setIsDirty(true);
                      }}
                      className="w-full h-full resize-none font-mono text-sm"
                      placeholder="Start editing your artifact..."
                    />
                  </div>
                </>
              ) : (
                <div className="flex-1 flex items-center justify-center">
                  <div className="text-center text-muted-foreground">
                    <FileText className="mx-auto h-12 w-12 mb-4" />
                    <h3 className="text-lg font-medium mb-2">No file selected</h3>
                    <p>Select a file from the tree to start editing</p>
                  </div>
                </div>
              )}
            </div>
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>
    </div>
  );
}