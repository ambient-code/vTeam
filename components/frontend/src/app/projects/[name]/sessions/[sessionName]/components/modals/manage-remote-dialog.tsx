"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Loader2 } from "lucide-react";

type MergeStatus = {
  canMergeClean: boolean;
  conflictingFiles: string[];
  remoteCommitsAhead?: number;
  localCommitsAhead?: number;
};

type ManageRemoteDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (url: string, branch: string) => Promise<void>;
  directoryName: string;
  currentUrl?: string;
  currentBranch?: string;
  remoteBranches?: string[];
  mergeStatus?: MergeStatus | null;
  isLoading?: boolean;
};

export function ManageRemoteDialog({
  open,
  onOpenChange,
  onSave,
  directoryName,
  currentUrl = "",
  currentBranch = "main",
  remoteBranches = [],
  mergeStatus,
  isLoading = false,
}: ManageRemoteDialogProps) {
  const [remoteUrl, setRemoteUrl] = useState(currentUrl);
  const [remoteBranch, setRemoteBranch] = useState(currentBranch);
  const [showCreateBranch, setShowCreateBranch] = useState(false);
  const [newBranchName, setNewBranchName] = useState("");

  // Update local state when props change
  useEffect(() => {
    setRemoteUrl(currentUrl);
    setRemoteBranch(currentBranch);
  }, [currentUrl, currentBranch, open]);

  const handleSave = async () => {
    if (!remoteUrl.trim()) return;
    
    await onSave(remoteUrl.trim(), remoteBranch.trim() || "main");
    setShowCreateBranch(false);
  };

  const handleCancel = () => {
    setShowCreateBranch(false);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Manage Remote for {directoryName}</DialogTitle>
          <DialogDescription>
            Configure repository URL and select branch for git operations
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="remote-repo-url">Repository URL *</Label>
            <Input
              id="remote-repo-url"
              placeholder="https://github.com/org/my-repo.git"
              value={remoteUrl}
              onChange={(e) => setRemoteUrl(e.target.value)}
            />
          </div>

          {!showCreateBranch ? (
            <div className="space-y-2">
              <Label htmlFor="remote-branch">Branch</Label>
              <div className="flex gap-2">
                <Select 
                  value={remoteBranch} 
                  onValueChange={(branch) => setRemoteBranch(branch)}
                >
                  <SelectTrigger className="flex-1">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {!remoteBranches.includes(remoteBranch) && remoteBranch && (
                      <SelectItem value={remoteBranch}>{remoteBranch} (current)</SelectItem>
                    )}
                    {remoteBranches.map(b => (
                      <SelectItem key={b} value={b}>{b}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button 
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setShowCreateBranch(true);
                    setNewBranchName("");
                  }}
                >
                  New
                </Button>
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              <Label>Create New Branch</Label>
              <Input
                placeholder="branch-name"
                value={newBranchName}
                onChange={e => setNewBranchName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newBranchName.trim()) {
                    setRemoteBranch(newBranchName.trim());
                    setShowCreateBranch(false);
                  }
                }}
                autoFocus
              />
              <div className="flex gap-2">
                <Button 
                  size="sm"
                  className="flex-1"
                  onClick={() => {
                    setRemoteBranch(newBranchName.trim());
                    setShowCreateBranch(false);
                  }}
                  disabled={!newBranchName.trim()}
                >
                  Set Branch
                </Button>
                <Button 
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setShowCreateBranch(false);
                    setNewBranchName("");
                  }}
                >
                  Cancel
                </Button>
              </div>
            </div>
          )}
          
          {mergeStatus && !showCreateBranch && (
            <div className="text-xs text-muted-foreground border-t pt-2">
              {mergeStatus.canMergeClean ? (
                <span className="text-green-600">✓ Can merge cleanly</span>
              ) : (
                <span className="text-amber-600">⚠️ {mergeStatus.conflictingFiles.length} conflicts</span>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleCancel}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={!remoteUrl.trim() || isLoading}
          >
            {isLoading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Saving...
              </>
            ) : (
              "Save Remote"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

