"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useGitHubForks } from "@/services/queries";

type Repo = {
  input: { url: string; branch: string };
  output?: { url: string; branch: string };
};

type RepositoryDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  repo: Repo;
  onRepoChange: (repo: Repo) => void;
  onSave: () => void;
  isEditing: boolean;
  projectName: string;
};

export function RepositoryDialog({
  open,
  onOpenChange,
  repo,
  onRepoChange,
  onSave,
  isEditing,
  projectName,
}: RepositoryDialogProps) {
  const [forkOptions, setForkOptions] = useState<Array<{ fullName: string; url: string }>>([]);
  const [outputBranchMode, setOutputBranchMode] = useState<"same" | "auto">("auto");

  // Fetch forks using React Query - only when we have an input URL
  const { data: forksData } = useGitHubForks(projectName, repo.input.url);
  
  useEffect(() => {
    if (open && repo.input.url && forksData) {
      // Filter forks based on the input URL
      const filtered = forksData.filter(fork => {
        // Match fork URL with input URL
        return fork.url === repo.input.url || fork.fullName.includes(repo.input.url.split('/').pop() || '');
      });
      setForkOptions(filtered);
    }
  }, [open, repo.input.url, forksData]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Edit Repository" : "Add Repository"}</DialogTitle>
          <DialogDescription>Configure input and optional output repository settings</DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Input Repository URL</label>
            <Input
              placeholder="https://github.com/org/repo.git"
              value={repo.input.url}
              onChange={(e) => onRepoChange({ ...repo, input: { ...repo.input, url: e.target.value } })}
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">Input Branch</label>
            <Input
              placeholder="main"
              value={repo.input.branch}
              onChange={(e) => onRepoChange({ ...repo, input: { ...repo.input, branch: e.target.value } })}
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">Output Repository (optional)</label>
            <Select
              value={repo.output?.url || "__none__"}
              onValueChange={(val) => {
                if (val === "__none__") {
                  onRepoChange({ ...repo, output: undefined });
                } else {
                  onRepoChange({
                    ...repo,
                    output: { url: val, branch: outputBranchMode === "same" ? repo.input.branch : "" },
                  });
                }
              }}
            >
              <SelectTrigger>
                <SelectValue
                  placeholder={repo.input.url ? "Select fork or same as input" : "Enter input repo first"}
                />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__none__">No output (don&apos;t push)</SelectItem>
                {repo.input.url && <SelectItem value={repo.input.url}>Same as input</SelectItem>}
                {forkOptions.map((f) => (
                  <SelectItem key={f.fullName} value={f.url}>
                    {f.fullName}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">Must be upstream or one of your forks</p>
          </div>
          {repo.output?.url && (
            <div className="space-y-2">
              <label className="text-sm font-medium">Output Branch</label>
              <Select
                value={outputBranchMode}
                onValueChange={(val: "same" | "auto") => {
                  setOutputBranchMode(val);
                  if (val === "same") {
                    onRepoChange({ ...repo, output: { ...repo.output!, branch: repo.input.branch } });
                  } else {
                    onRepoChange({ ...repo, output: { ...repo.output!, branch: "" } });
                  }
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select output branch mode" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="same">Same as input branch</SelectItem>
                  <SelectItem value="auto">Auto-generate sessions/&#123;&#123;session_id&#125;&#125;</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">To avoid conflicts, custom branches are not allowed</p>
            </div>
          )}
        </div>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            onClick={() => {
              if (!repo.input.url) return;
              onSave();
              onOpenChange(false);
            }}
          >
            {isEditing ? "Update" : "Add"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
