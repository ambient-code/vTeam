"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Loader2, Trash2 } from "lucide-react";
import type { RFEWorkflow } from "@/types/agentic-session";

type RfeHeaderProps = {
  workflow: RFEWorkflow;
  projectName: string;
  deleting: boolean;
  onDelete: () => Promise<void>;
};

export function RfeHeader({ workflow, projectName, deleting, onDelete }: RfeHeaderProps) {
  return (
    <div className="flex items-start justify-between">
      <div className="flex items-center gap-4">
        <Link href={`/projects/${encodeURIComponent(projectName)}/rfe`}>
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to RFE Workspaces
          </Button>
        </Link>
        <div>
          <h1 className="text-3xl font-bold">{workflow.title}</h1>
          <p className="text-muted-foreground mt-1">{workflow.description}</p>
        </div>
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
