"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { RefreshCw, ArrowLeft, Download } from "lucide-react";
import { FileTree, type FileTreeNode } from "@/components/file-tree";
import { getApiUrl } from "@/lib/config";

type ListItem = { name: string; path: string; isDir: boolean; size: number; modifiedAt: string };

export default function SessionWorkspacePage() {
  const params = useParams();
  const projectName = params.name as string;
  const sessionName = params.sessionName as string;

  const [tree, setTree] = useState<FileTreeNode[]>([]);
  const [selectedPath, setSelectedPath] = useState<string | undefined>(undefined);
  const [fileContent, setFileContent] = useState<string>("");
  const [loading, setLoading] = useState<boolean>(true);

  const listPath = useCallback(async (relPath?: string) => {
    const url = new URL(`${getApiUrl()}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace`, window.location.origin);
    if (relPath) url.searchParams.set("path", relPath);
    const resp = await fetch(url.toString());
    if (!resp.ok) throw new Error("Failed to list workspace");
    const data = await resp.json();
    const items: ListItem[] = Array.isArray(data.items) ? data.items : [];
    return items;
  }, [projectName, sessionName]);

  const readFile = useCallback(async (rel: string) => {
    const resp = await fetch(`${getApiUrl()}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${encodeURIComponent(rel)}`);
    if (!resp.ok) throw new Error("Failed to fetch file");
    const text = await resp.text();
    return text;
  }, [projectName, sessionName]);

  const buildRoot = useCallback(async () => {
    setLoading(true);
    try {
      const items = await listPath();
      const children: FileTreeNode[] = items.map((it) => ({
        name: it.name,
        path: it.path.replace(/^\/+/, ""),
        type: it.isDir ? "folder" : "file",
        expanded: it.isDir,
        sizeKb: it.isDir ? undefined : it.size / 1024,
      }));
      setTree(children);
    } finally {
      setLoading(false);
    }
  }, [listPath]);

  useEffect(() => {
    buildRoot();
  }, [buildRoot]);

  const onToggle = useCallback(async (node: FileTreeNode) => {
    if (node.type !== "folder") return;
    // If already populated, skip
    if (node.children && node.children.length > 0) return;
    const items = await listPath(node.path);
    node.children = items.map((it) => ({
      name: it.name,
      path: it.path.replace(/^\/+/, ""),
      type: it.isDir ? "folder" : "file",
      expanded: false,
      sizeKb: it.isDir ? undefined : it.size / 1024,
    }));
    setTree((prev) => [...prev]);
  }, [listPath]);

  const onSelect = useCallback(async (node: FileTreeNode) => {
    if (node.type !== "file") return;
    setSelectedPath(node.path);
    const text = await readFile(node.path);
    setFileContent(text);
  }, [readFile]);

  const downloadSelected = useCallback(() => {
    if (!selectedPath) return;
    const blob = new Blob([fileContent], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = selectedPath.split("/").pop() || "download";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [fileContent, selectedPath]);

  return (
    <div className="h-screen flex flex-col">
      <div className="border-b bg-background px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link href={`/projects/${encodeURIComponent(projectName)}/sessions/${encodeURIComponent(sessionName)}`}>
              <Button variant="ghost" size="sm">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Session
              </Button>
            </Link>
            <div>
              <h1 className="text-xl font-semibold">Workspace</h1>
              <p className="text-sm text-muted-foreground">{sessionName}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button onClick={buildRoot} variant="ghost" size="sm">
              <RefreshCw className="h-4 w-4" />
            </Button>
            <Button onClick={downloadSelected} disabled={!selectedPath} size="sm" variant="outline">
              <Download className="mr-2 h-4 w-4" />
              Download
            </Button>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-hidden">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-0 h-full">
          <div className="border-r overflow-auto">
            <div className="p-4 border-b">
              <h3 className="font-medium">Files</h3>
              <p className="text-sm text-muted-foreground">{tree.length} items</p>
            </div>
            <div className="p-2">
              <FileTree nodes={tree} selectedPath={selectedPath} onSelect={onSelect} onToggle={onToggle} />
            </div>
          </div>
          <div className="overflow-auto">
            <Card className="m-4">
              <CardHeader>
                <CardTitle>{selectedPath ? selectedPath.split("/").pop() : ""}</CardTitle>
                <CardDescription>{selectedPath || ""}</CardDescription>
              </CardHeader>
              <CardContent className="p-4">
                {!selectedPath && (
                  <div className="text-sm text-muted-foreground p-4">Select a file to preview</div>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}


