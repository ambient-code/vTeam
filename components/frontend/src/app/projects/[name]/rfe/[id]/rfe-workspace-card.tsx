"use client";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { FolderTree, AlertCircle, Loader2, Sprout, CheckCircle2 } from "lucide-react";
import type { RFEWorkflow } from "@/types/agentic-session";

type RfeWorkspaceCardProps = {
  workflow: RFEWorkflow;
  workflowWorkspace: string;
  isSeeded: boolean;
  seedingStatus: { checking: boolean };
  seedingError: string | null | undefined;
  seeding: boolean;
  onSeedWorkflow: () => Promise<void>;
};

export function RfeWorkspaceCard({
  workflow,
  workflowWorkspace,
  isSeeded,
  seedingStatus,
  seedingError,
  seeding,
  onSeedWorkflow,
}: RfeWorkspaceCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <FolderTree className="h-5 w-5" />
          Workspace & Repositories
        </CardTitle>
        <CardDescription>Shared workspace for this workflow and optional repos</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="text-sm text-muted-foreground">Workspace: {workflowWorkspace}</div>
        {(workflow as { parentOutcome?: string }).parentOutcome && (
          <div className="mt-2 text-sm">
            <span className="font-medium">Parent Outcome:</span>{' '}
            <Badge variant="outline">{(workflow as { parentOutcome?: string }).parentOutcome}</Badge>
          </div>
        )}
        {(workflow.umbrellaRepo || (workflow.supportingRepos || []).length > 0) && (
          <div className="mt-2 space-y-1">
            {workflow.umbrellaRepo && (
              <div className="text-sm">
                <span className="font-medium">Umbrella:</span> {workflow.umbrellaRepo.url}
                {workflow.umbrellaRepo.branch && (
                  <span className="text-muted-foreground"> @ {workflow.umbrellaRepo.branch}</span>
                )}
              </div>
            )}
            {(workflow.supportingRepos || []).map(
              (r: { url: string; branch?: string; clonePath?: string }, i: number) => (
                <div key={i} className="text-sm">
                  <span className="font-medium">Supporting:</span> {r.url}
                  {r.branch && <span className="text-muted-foreground"> @ {r.branch}</span>}
                </div>
              )
            )}
          </div>
        )}

        {!isSeeded && !seedingStatus.checking && workflow.umbrellaRepo && (
          <Alert variant="destructive" className="mt-4">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>Umbrella Repository Not Seeded</AlertTitle>
            <AlertDescription className="mt-2">
              <p className="mb-3">
                Before you can start working on phases, the umbrella repository needs to be seeded
                with:
              </p>
              <ul className="list-disc list-inside space-y-1 mb-3 text-sm">
                <li>Spec-Kit template files for spec-driven development</li>
                <li>Agent definition files in the .claude directory</li>
              </ul>
              {seedingError && (
                <div className="mb-3 p-2 bg-red-100 border border-red-300 rounded text-sm text-red-800">
                  <strong>Check Error:</strong> {seedingError}
                </div>
              )}
              <Button onClick={onSeedWorkflow} disabled={seeding} size="sm">
                {seeding ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Seeding Repository...
                  </>
                ) : (
                  <>
                    <Sprout className="mr-2 h-4 w-4" />
                    Seed Repository
                  </>
                )}
              </Button>
            </AlertDescription>
          </Alert>
        )}

        {seedingStatus.checking && workflow.umbrellaRepo && (
          <div className="mt-4 flex items-center gap-2 text-gray-600 bg-gray-50 p-3 rounded-lg">
            <Loader2 className="h-5 w-5 animate-spin" />
            <span className="text-sm">Checking repository seeding status...</span>
          </div>
        )}

        {isSeeded && (
          <div className="mt-4 flex items-center gap-2 text-green-700 bg-green-50 p-3 rounded-lg">
            <CheckCircle2 className="h-5 w-5 text-green-600" />
            <span className="text-sm font-medium">Repository seeded and ready</span>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
