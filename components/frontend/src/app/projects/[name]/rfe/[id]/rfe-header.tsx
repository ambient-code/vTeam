"use client";

import { Button } from "@/components/ui/button";
import { Loader2, Trash2 } from "lucide-react";
import type { RFEWorkflow } from "@/types/agentic-session";

type RfeHeaderProps = {
  workflow: RFEWorkflow;
  deleting: boolean;
  onDelete: () => Promise<void>;
};

export function RfeHeader({ workflow, deleting, onDelete }: RfeHeaderProps) {
  return (
    <div className="flex items-start justify-between">
      <div>
        <h1 className="text-3xl font-bold">{workflow.title}</h1>
        <p className="text-muted-foreground mt-1">{workflow.description}</p>
      </div>
      <Button variant="destructive" size="sm" onClick={onDelete} disabled={deleting}>
        {deleting ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Deleting…
          </>
        ) : (
          <>
            <Trash2 className="mr-2 h-4 w-4" />
            Delete Workflow
          </>
        )}
      </Button>
    </div>
  );
}
