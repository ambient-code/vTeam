"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { GitStatus } from "@/services/api/workspace";

type CommitChangesDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCommit: (message: string) => Promise<void>;
  gitStatus: GitStatus | null;
  directoryName: string;
  isCommitting?: boolean;
};

export function CommitChangesDialog({
  open,
  onOpenChange,
  onCommit,
  gitStatus,
  directoryName,
  isCommitting = false,
}: CommitChangesDialogProps) {
  const [commitMessage, setCommitMessage] = useState("");

  const handleCommit = async () => {
    if (!commitMessage.trim()) return;
    
    await onCommit(commitMessage.trim());
    setCommitMessage("");
  };

  const handleCancel = () => {
    setCommitMessage("");
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Commit Changes</DialogTitle>
          <DialogDescription>
            Commit {gitStatus?.uncommittedFiles || 0} files to {directoryName}
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="commit-message">Commit Message *</Label>
            <Input
              id="commit-message"
              placeholder="Update feature specification"
              value={commitMessage}
              onChange={(e) => setCommitMessage(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && commitMessage.trim()) {
                  handleCommit();
                }
              }}
              autoFocus
            />
          </div>
          
          {gitStatus && (
            <div className="text-xs text-muted-foreground bg-muted p-2 rounded">
              <div className="font-medium mb-1">Changes to commit:</div>
              <div className="space-y-0.5">
                <div>Files: {gitStatus.uncommittedFiles ?? 0}</div>
                <div className="text-green-600">+{gitStatus.totalAdded ?? 0} lines</div>
                {(gitStatus.totalRemoved ?? 0) > 0 && (
                  <div className="text-red-600">-{gitStatus.totalRemoved} lines</div>
                )}
              </div>
            </div>
          )}
        </div>
        
        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleCancel}
          >
            Cancel
          </Button>
          <Button
            onClick={handleCommit}
            disabled={!commitMessage.trim() || isCommitting}
          >
            {isCommitting ? (
              <>
                <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                Committing...
              </>
            ) : (
              'Commit'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

