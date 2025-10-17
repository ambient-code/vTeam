"use client";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Edit2, Plus, Trash2, ArrowRight } from "lucide-react";
import { EmptyState } from "@/components/empty-state";
import { FolderGit2 } from "lucide-react";

type Repo = {
  input: { url: string; branch: string };
  output?: { url: string; branch: string };
};

type RepositoryListProps = {
  repos: Repo[];
  mainRepoIndex: number;
  onAddRepo: () => void;
  onEditRepo: (index: number) => void;
  onRemoveRepo: (index: number) => void;
  onSetMainRepo: (index: number) => void;
};

export function RepositoryList({
  repos,
  mainRepoIndex,
  onAddRepo,
  onEditRepo,
  onRemoveRepo,
  onSetMainRepo,
}: RepositoryListProps) {
  if (!repos || repos.length === 0) {
    return (
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <label className="text-sm font-medium">Repositories</label>
          <Button type="button" variant="outline" size="sm" onClick={onAddRepo}>
            <Plus className="w-4 h-4 mr-1" />
            Add Repository
          </Button>
        </div>
        <EmptyState
          icon={FolderGit2}
          title="No repositories configured"
          description="Add at least one repository for Claude to work with."
          action={{
            label: "Add Your First Repository",
            onClick: onAddRepo,
          }}
        />
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <label className="text-sm font-medium">Repositories</label>
        <Button type="button" variant="outline" size="sm" onClick={onAddRepo}>
          <Plus className="w-4 h-4 mr-1" />
          Add Repository
        </Button>
      </div>
      <div className="space-y-2">
        {repos.map((repo, idx) => (
          <div key={idx} className="border rounded p-3 space-y-2">
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1 space-y-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">Input:</span>
                  <code className="text-xs bg-muted px-1.5 py-0.5 rounded">{repo.input.url}</code>
                  {repo.input.branch && (
                    <Badge variant="outline" className="text-xs">
                      {repo.input.branch}
                    </Badge>
                  )}
                </div>
                {repo.output?.url && (
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">Output:</span>
                    <ArrowRight className="w-3 h-3" />
                    <code className="text-xs bg-muted px-1.5 py-0.5 rounded">{repo.output.url}</code>
                    {repo.output.branch && (
                      <Badge variant="outline" className="text-xs">
                        {repo.output.branch || "auto"}
                      </Badge>
                    )}
                  </div>
                )}
              </div>
              <div className="flex items-center gap-1">
                {idx === mainRepoIndex && (
                  <Badge className="text-xs">Working Directory</Badge>
                )}
                {idx !== mainRepoIndex && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => onSetMainRepo(idx)}
                    title="Set as working directory"
                  >
                    <span className="text-xs">Set as Working Directory</span>
                  </Button>
                )}
                <Button type="button" variant="ghost" size="sm" onClick={() => onEditRepo(idx)}>
                  <Edit2 className="w-4 h-4" />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => onRemoveRepo(idx)}
                  disabled={repos.length === 1}
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </div>
          </div>
        ))}
      </div>
      <p className="text-xs text-muted-foreground">
        The {repos[mainRepoIndex]?.input?.url || "selected"} repo is Claude&apos;s working directory. Other
        repos are available as add_dirs.
      </p>
    </div>
  );
}
