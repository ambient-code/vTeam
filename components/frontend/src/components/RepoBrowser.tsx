'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { GitBranch } from 'lucide-react';
import { apiClient } from '@/utils/api';
import { RepoEntry, RepoBlob } from '@/types';
import { FileTree, type FileTreeNode } from '@/components/file-tree';

interface RepoBrowserProps {
  projectName: string;
  repoUrl: string;
  defaultRef?: string;
  onFileSelect?: (path: string, content: string) => void;
}

// Breadcrumb UI removed in favor of FileTree-based layout

export default function RepoBrowser({
  projectName,
  repoUrl,
  defaultRef = 'main',
  onFileSelect,
}: RepoBrowserProps) {
  const [currentRef] = useState(defaultRef);
  const [nodes, setNodes] = useState<FileTreeNode[]>([]);
  const [selectedPath, setSelectedPath] = useState<string | undefined>(undefined);
  const [fileContent, setFileContent] = useState<RepoBlob | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const entryToNode = (entry: RepoEntry, basePath: string = ''): FileTreeNode => {
    const nodePath = basePath ? `${basePath}/${entry.name}` : entry.name;
    return {
      name: entry.name,
      path: nodePath,
      type: entry.type === 'tree' ? 'folder' : 'file',
      expanded: false,
      sizeKb: typeof entry.size === 'number' ? entry.size / 1024 : undefined,
    };
  };

  const updateChildrenByPath = useCallback((nodesIn: FileTreeNode[], targetPath: string, children: FileTreeNode[]): FileTreeNode[] => {
    return nodesIn.map((n) => {
      if (n.path === targetPath) {
        return { ...n, children };
      }
      if (n.type === 'folder' && n.children && n.children.length > 0) {
        return { ...n, children: updateChildrenByPath(n.children, targetPath, children) };
      }
      return n;
    });
  }, []);

  const loadRoot = useCallback(async () => {
    setLoading(true);
    setError(null);
    setFileContent(null);
    setSelectedPath(undefined);
    try {
      const response = await apiClient.getRepoTree(repoUrl, currentRef, '');
      const rootNodes = (response.entries || []).map((e) => entryToNode(e));
      setNodes(rootNodes);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load repository tree';
      setError(errorMessage);
      setNodes([]);
    } finally {
      setLoading(false);
    }
  }, [projectName, repoUrl, currentRef]);

  useEffect(() => {
    loadRoot();
  }, [loadRoot]);

  const onToggle = useCallback(async (node: FileTreeNode) => {
    if (node.type !== 'folder') return;
    try {
      const response = await apiClient.getRepoTree(repoUrl, currentRef, node.path);
      const children = (response.entries || []).map((e) => entryToNode(e, node.path));
      setNodes((prev) => updateChildrenByPath(prev, node.path, children));
    } catch {
      // ignore toggle error; keep previous state
    }
  }, [repoUrl, currentRef, updateChildrenByPath]);

  const onSelect = useCallback(async (node: FileTreeNode) => {
    // Only handle file selection
    setSelectedPath(node.path);
    setLoading(true);
    setError(null);
    try {
      const response = await apiClient.getRepoBlob(repoUrl, currentRef, node.path);
      setFileContent(response);
      if (onFileSelect) {
        onFileSelect(node.path, response.content);
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load file content';
      setError(errorMessage);
      setFileContent(null);
    } finally {
      setLoading(false);
    }
  }, [repoUrl, currentRef, onFileSelect]);

  // Directory previews in the right pane are intentionally omitted

  return (
    <Card className="h-full">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <GitBranch className="w-5 h-5" />
          Repository Browser
        </CardTitle>
        <div className="flex items-center gap-2 text-sm text-gray-600">
          <span>{repoUrl}</span>
          <span>@</span>
          <span className="font-mono bg-gray-100 px-2 py-1 rounded">
            {currentRef}
          </span>
        </div>
      </CardHeader>
      <CardContent>
        {error && (
          <Alert variant="destructive" className="mb-4">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="md:col-span-1 border rounded-lg overflow-hidden">
            <div className="p-2">
              {loading && nodes.length === 0 ? (
                <div className="text-sm text-muted-foreground p-2">Loading…</div>
              ) : (
                <FileTree nodes={nodes} selectedPath={selectedPath} onSelect={onSelect} onToggle={onToggle} />
              )}
            </div>
          </div>
          <div className="md:col-span-2 border rounded-lg p-3 min-h-[300px]">
            {fileContent ? (
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <div className="text-xs text-muted-foreground">{selectedPath}</div>
                  <div className="text-xs text-muted-foreground">
                    Size: {formatFileSize(fileContent.size)} | Encoding: {fileContent.encoding}
                  </div>
                </div>
                <div className="bg-gray-50 rounded-lg p-4 overflow-auto max-h-[60vh]">
                  <pre className="text-sm whitespace-pre-wrap break-words">{fileContent.content}</pre>
                </div>
              </div>
            ) : loading ? (
              <div className="text-sm text-muted-foreground">Loading…</div>
            ) : (
              <div className="text-sm text-muted-foreground">Select a file to view its contents</div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}