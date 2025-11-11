"use client";

import { useState } from "react";
import { Loader2, Info } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";

type AddContextModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAddRepository: (url: string, branch: string) => Promise<void>;
  isLoading?: boolean;
};

export function AddContextModal({
  open,
  onOpenChange,
  onAddRepository,
  isLoading = false,
}: AddContextModalProps) {
  const [contextUrl, setContextUrl] = useState("");
  const [contextBranch, setContextBranch] = useState("main");

  const handleSubmit = async () => {
    if (!contextUrl.trim()) return;
    
    await onAddRepository(contextUrl.trim(), contextBranch.trim() || 'main');
    
    // Reset form
    setContextUrl("");
    setContextBranch("main");
  };

  const handleCancel = () => {
    setContextUrl("");
    setContextBranch("main");
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Add Context</DialogTitle>
          <DialogDescription>
            Add additional context to improve AI responses.
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <Alert>
            <Info className="h-4 w-4" />
            <AlertDescription>
              Note: additional data sources like Jira, Google Drive, files, and MCP Servers are on the roadmap!
            </AlertDescription>
          </Alert>

          <div className="space-y-2">
            <Label htmlFor="context-url">Repository URL</Label>
            <Input
              id="context-url"
              placeholder="https://github.com/org/repo"
              value={contextUrl}
              onChange={(e) => setContextUrl(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Currently supports GitHub repositories for code context
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="context-branch">Branch (optional)</Label>
            <Input
              id="context-branch"
              placeholder="main"
              value={contextBranch}
              onChange={(e) => setContextBranch(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Leave empty to use the default branch
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={handleCancel}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSubmit}
            disabled={!contextUrl.trim() || isLoading}
          >
            {isLoading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Adding...
              </>
            ) : (
              'Add'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

