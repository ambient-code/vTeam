"use client";

import { useState, useCallback } from "react";
import { errorToast, successToast } from "@/hooks/use-toast";
import { readWorkspaceFile } from "@/services/api/workspace";
import type { FileTreeNode } from "@/components/file-tree";

type ViewingFile = {
  path: string;
  content: string;
};

type UseFileOperationsProps = {
  projectName: string;
  sessionName: string;
  basePath: string;
};

export function useFileOperations({
  projectName,
  sessionName,
  basePath,
}: UseFileOperationsProps) {
  const [currentSubPath, setCurrentSubPath] = useState<string>("");
  const [viewingFile, setViewingFile] = useState<ViewingFile | null>(null);
  const [loadingFile, setLoadingFile] = useState(false);

  // Navigate into folder or load file content
  const handleFileOrFolderSelect = useCallback(async (node: FileTreeNode) => {
    if (node.type === 'folder') {
      // Navigate into folder
      const newSubPath = currentSubPath ? `${currentSubPath}/${node.name}` : node.name;
      setCurrentSubPath(newSubPath);
      setViewingFile(null);
    } else {
      // Load file content inline
      setLoadingFile(true);
      try {
        const fullPath = currentSubPath 
          ? `${basePath}/${currentSubPath}/${node.name}`
          : `${basePath}/${node.name}`;
        
        const content = await readWorkspaceFile(projectName, sessionName, fullPath);
        setViewingFile({ path: node.name, content });
      } catch (error) {
        console.error("Failed to load file:", error);
        errorToast(error instanceof Error ? error.message : 'Failed to load file');
      } finally {
        setLoadingFile(false);
      }
    }
  }, [projectName, sessionName, basePath, currentSubPath]);

  // Download the currently viewing file
  const handleDownloadFile = useCallback(() => {
    if (!viewingFile) return;

    try {
      const fullPath = currentSubPath
        ? `${basePath}/${currentSubPath}/${viewingFile.path}`
        : `${basePath}/${viewingFile.path}`;

      const downloadUrl = `/api/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${encodeURIComponent(fullPath)}`;

      // Create a hidden link and click it to trigger download
      const link = document.createElement('a');
      link.href = downloadUrl;
      link.download = viewingFile.path;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);

      successToast(`Downloading ${viewingFile.path}...`);
    } catch (err) {
      errorToast(err instanceof Error ? err.message : "Failed to download file");
    }
  }, [viewingFile, currentSubPath, basePath, projectName, sessionName]);

  // Navigate back to parent folder
  const navigateBack = useCallback(() => {
    if (viewingFile) {
      // Go back to file tree
      setViewingFile(null);
    } else if (currentSubPath) {
      // Go back to parent folder
      const pathParts = currentSubPath.split('/');
      pathParts.pop();
      setCurrentSubPath(pathParts.join('/'));
    }
  }, [viewingFile, currentSubPath]);

  // Reset to root
  const resetToRoot = useCallback(() => {
    setCurrentSubPath("");
    setViewingFile(null);
  }, []);

  return {
    currentSubPath,
    viewingFile,
    loadingFile,
    handleFileOrFolderSelect,
    handleDownloadFile,
    navigateBack,
    resetToRoot,
    setCurrentSubPath,
    setViewingFile,
  };
}

