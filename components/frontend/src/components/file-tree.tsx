"use client";

import { useState } from "react";
import { Folder, FolderOpen, FileText } from "lucide-react";

export type FileTreeNode = {
  name: string;
  path: string;
  type: "file" | "folder";
  children?: FileTreeNode[];
  expanded?: boolean;
  sizeKb?: number;
  data?: unknown;
};

export type FileTreeProps = {
  nodes: FileTreeNode[];
  selectedPath?: string;
  onSelect: (node: FileTreeNode) => void;
  onToggle?: (node: FileTreeNode) => Promise<void> | void;
  className?: string;
};

export function FileTree({ nodes, selectedPath, onSelect, onToggle, className }: FileTreeProps) {
  return (
    <div className={className}>
      {nodes.map((node) => (
        <FileTreeItem
          key={node.path}
          node={node}
          selectedPath={selectedPath}
          onSelect={onSelect}
          onToggle={onToggle}
        />
      ))}
    </div>
  );
}

type ItemProps = {
  node: FileTreeNode;
  selectedPath?: string;
  onSelect: (node: FileTreeNode) => void;
  onToggle?: (node: FileTreeNode) => Promise<void> | void;
  depth?: number;
};

function FileTreeItem({ node, selectedPath, onSelect, onToggle, depth = 0 }: ItemProps) {
  const [expanded, setExpanded] = useState<boolean>(node.expanded ?? true);
  const isSelected = node.path === selectedPath;

  return (
    <div>
      <div
        className={`flex items-center gap-2 px-2 py-1 text-sm rounded cursor-pointer hover:bg-muted ${
          isSelected ? "bg-muted" : ""
        }`}
        style={{ paddingLeft: `${(depth + 1) * 12}px` }}
        onClick={async () => {
          if (node.type === "folder") {
            // If folder has children, toggle expand/collapse
            if (node.children && node.children.length > 0) {
              const next = !expanded;
              setExpanded(next);
              if (next && onToggle) {
                await onToggle(node);
              }
            } else {
              // If folder has no children (flat listing), call onSelect to navigate
              onSelect(node);
            }
          } else {
            onSelect(node);
          }
        }}
      >
        {node.type === "folder" ? (
          expanded ? (
            <FolderOpen className="h-4 w-4 text-blue-600" />
          ) : (
            <Folder className="h-4 w-4 text-blue-600" />
          )
        ) : (
          <FileText className="h-4 w-4 text-gray-600" />
        )}

        <span className={`flex-1 ${isSelected ? "font-medium" : ""}`}>{node.name}</span>

        {typeof node.sizeKb === "number" && (
          <span className="text-xs text-muted-foreground">{node.sizeKb.toFixed(1)}K</span>
        )}
      </div>

      {node.type === "folder" && expanded && node.children && node.children.length > 0 && (
        <div>
          {node.children.map((child) => (
            <FileTreeItem
              key={child.path}
              node={child}
              selectedPath={selectedPath}
              onSelect={onSelect}
              onToggle={onToggle}
              depth={depth + 1}
            />
          ))}
        </div>
      )}
    </div>
  );
}


