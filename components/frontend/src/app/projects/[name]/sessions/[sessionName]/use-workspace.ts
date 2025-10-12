import { useState, useCallback, useRef } from 'react';
import { getApiUrl } from '@/lib/config';
import type { FileTreeNode } from '@/components/file-tree';

type UseWorkspaceReturn = {
  wsTree: FileTreeNode[];
  wsSelectedPath: string | undefined;
  wsFileContent: string;
  wsLoading: boolean;
  wsUnavailable: boolean;
  onWsToggle: (node: FileTreeNode) => Promise<void>;
  onWsSelect: (node: FileTreeNode) => Promise<void>;
  buildWsRoot: (background?: boolean) => Promise<void>;
  readWsFile: (rel: string) => Promise<void>;
  writeWsFile: (rel: string, content: string) => Promise<void>;
};

export function useWorkspace(projectName: string, sessionName: string): UseWorkspaceReturn {
  const [wsTree, setWsTree] = useState<FileTreeNode[]>([]);
  const [wsSelectedPath, setWsSelectedPath] = useState<string | undefined>(undefined);
  const [wsFileContent, setWsFileContent] = useState<string>("");
  const [wsLoading, setWsLoading] = useState<boolean>(false);
  const [wsUnavailable, setWsUnavailable] = useState<boolean>(false);

  const wsErrCountRef = useRef<number>(0);
  const wsBackoffUntilRef = useRef<number>(0);
  const wsTreeRef = useRef<FileTreeNode[]>([]);

  const listWsPath = useCallback(async (relPath?: string) => {
    if (!projectName || !sessionName) return [];
    const apiUrl = getApiUrl();
    let url = `${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace`;
    if (relPath) url += `/${relPath}`;
    const res = await fetch(url);
    if (!res.ok) throw new Error(`Failed to list workspace: ${res.statusText}`);
    const data = await res.json();
    return (data.entries || []) as FileTreeNode[];
  }, [projectName, sessionName]);

  const readWsFile = useCallback(async (rel: string) => {
    const apiUrl = getApiUrl();
    const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${rel}`);
    if (!res.ok) throw new Error('Failed to read file');
    const text = await res.text();
    setWsFileContent(text);
  }, [projectName, sessionName]);

  const writeWsFile = useCallback(async (rel: string, content: string) => {
    const apiUrl = getApiUrl();
    const res = await fetch(`${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${rel}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/octet-stream' },
      body: content,
    });
    if (!res.ok) throw new Error('Failed to write file');
  }, [projectName, sessionName]);

  const collectExpanded = useCallback((nodes: FileTreeNode[], base = ""): Record<string, boolean> => {
    const acc: Record<string, boolean> = {};
    for (const n of nodes) {
      const path = base ? `${base}/${n.name}` : n.name;
      if (n.type === "folder" && n.expanded) {
        acc[path] = true;
        if (n.children) {
          Object.assign(acc, collectExpanded(n.children, path));
        }
      }
    }
    return acc;
  }, []);

  const buildWsRoot = useCallback(async (background = false) => {
    if (!background) setWsLoading(true);
    try {
      const now = Date.now();
      if (now < wsBackoffUntilRef.current) {
        return;
      }
      const entries = await listWsPath();
      wsErrCountRef.current = 0;
      wsBackoffUntilRef.current = 0;
      setWsUnavailable(false);
      setWsTree(entries);
      wsTreeRef.current = entries;
    } catch {
      wsErrCountRef.current += 1;
      if (wsErrCountRef.current >= 3) {
        setWsUnavailable(true);
        wsBackoffUntilRef.current = Date.now() + 60000;
      }
    } finally {
      if (!background) setWsLoading(false);
    }
  }, [listWsPath]);

  const onWsToggle = useCallback(async (node: FileTreeNode) => {
    const newTree = [...wsTreeRef.current];
    const toggleNode = (nodes: FileTreeNode[]): void => {
      for (const n of nodes) {
        if (n.name === node.name && n.type === node.type) {
          n.expanded = !n.expanded;
        }
        if (n.children) toggleNode(n.children);
      }
    };
    toggleNode(newTree);
    setWsTree(newTree);
    wsTreeRef.current = newTree;
  }, []);

  const onWsSelect = useCallback(async (node: FileTreeNode) => {
    setWsSelectedPath(node.path);
    if (node.type === "file") {
      await readWsFile(node.path);
    }
  }, [readWsFile]);

  return {
    wsTree,
    wsSelectedPath,
    wsFileContent,
    wsLoading,
    wsUnavailable,
    onWsToggle,
    onWsSelect,
    buildWsRoot,
    readWsFile,
    writeWsFile,
  };
}
