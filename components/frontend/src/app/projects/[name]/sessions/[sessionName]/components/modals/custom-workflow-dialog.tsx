"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type CustomWorkflowDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (url: string, branch: string, path: string) => void;
  isActivating?: boolean;
};

export function CustomWorkflowDialog({
  open,
  onOpenChange,
  onSubmit,
  isActivating = false,
}: CustomWorkflowDialogProps) {
  const [customWorkflowUrl, setCustomWorkflowUrl] = useState("");
  const [customWorkflowBranch, setCustomWorkflowBranch] = useState("main");
  const [customWorkflowPath, setCustomWorkflowPath] = useState("");

  const handleSubmit = () => {
    if (!customWorkflowUrl.trim()) return;
    
    onSubmit(
      customWorkflowUrl.trim(),
      customWorkflowBranch.trim() || "main",
      customWorkflowPath.trim() || ""
    );
    
    // Reset form
    setCustomWorkflowUrl("");
    setCustomWorkflowBranch("main");
    setCustomWorkflowPath("");
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Load Custom Workflow</DialogTitle>
          <DialogDescription>
            Enter the Git repository URL and optional path for your custom workflow.
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="workflow-url">Git Repository URL *</Label>
            <Input
              id="workflow-url"
              placeholder="https://github.com/org/workflow-repo.git"
              value={customWorkflowUrl}
              onChange={(e) => setCustomWorkflowUrl(e.target.value)}
              disabled={isActivating}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="workflow-branch">Branch</Label>
            <Input
              id="workflow-branch"
              placeholder="main"
              value={customWorkflowBranch}
              onChange={(e) => setCustomWorkflowBranch(e.target.value)}
              disabled={isActivating}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="workflow-path">Path (optional)</Label>
            <Input
              id="workflow-path"
              placeholder="workflows/my-workflow"
              value={customWorkflowPath}
              onChange={(e) => setCustomWorkflowPath(e.target.value)}
              disabled={isActivating}
            />
            <p className="text-xs text-muted-foreground">
              Optional subdirectory within the repository containing the workflow
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isActivating}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!customWorkflowUrl.trim() || isActivating}
          >
            {isActivating ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Loading...
              </>
            ) : (
              'Load Workflow'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

