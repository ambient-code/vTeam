"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useRepoBranches } from "@/services/queries";

type Repo = {
  url: string;
  branch?: string;
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
  // Fetch branches for the repository
  const { data: branchesData, isLoading: branchesLoading } = useRepoBranches(
    projectName,
    repo.url,
    { enabled: !!repo.url && open }
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Edit Repository" : "Add Repository"}</DialogTitle>
          <DialogDescription>Configure repository URL and branch</DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Repository URL</label>
            <Input
              placeholder="https://github.com/org/repo.git"
              value={repo.url}
              onChange={(e) => onRepoChange({ ...repo, url: e.target.value })}
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">Branch</label>
            <Select
              value={repo.branch || "main"}
              onValueChange={(value) => onRepoChange({ ...repo, branch: value })}
            >
              <SelectTrigger>
                <SelectValue placeholder={branchesLoading ? "Loading branches..." : "Select branch"} />
              </SelectTrigger>
              <SelectContent>
                {branchesLoading ? (
                  <SelectItem value="loading" disabled>Loading branches...</SelectItem>
                ) : branchesData?.branches && branchesData.branches.length > 0 ? (
                  branchesData.branches.map((branch) => (
                    <SelectItem key={branch.name} value={branch.name}>
                      {branch.name}
                    </SelectItem>
                  ))
                ) : (
                  <>
                    <SelectItem value="main">main</SelectItem>
                    <SelectItem value="master">master</SelectItem>
                    <SelectItem value="develop">develop</SelectItem>
                  </>
                )}
              </SelectContent>
            </Select>
            {!repo.url && (
              <p className="text-xs text-muted-foreground">Enter repository URL first to load branches</p>
            )}
          </div>
        </div>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            onClick={() => {
              if (!repo.url) return;
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
