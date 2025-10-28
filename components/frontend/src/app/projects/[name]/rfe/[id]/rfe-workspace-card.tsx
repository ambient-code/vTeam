"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { FolderTree, AlertCircle, Loader2, Sprout, CheckCircle2, GitBranch, Edit } from "lucide-react";
import type { RFEWorkflow } from "@/types/agentic-session";
import { EditRepositoriesDialog } from "./edit-repositories-dialog";

type RfeWorkspaceCardProps = {
  workflow: RFEWorkflow;
  workflowWorkspace: string;
  isSeeded: boolean;
  seedingStatus: { checking: boolean; hasChecked?: boolean };
  seedingError: string | null | undefined;
  seeding: boolean;
  onSeedWorkflow: () => Promise<void>;
  onUpdateRepositories: (data: { umbrellaRepo: { url: string; branch?: string }; supportingRepos: { url: string; branch?: string }[] }) => Promise<void>;
  updating: boolean;
};

export function RfeWorkspaceCard({
  workflow,
  workflowWorkspace,
  isSeeded,
  seedingStatus,
  seedingError,
  seeding,
  onSeedWorkflow,
  onUpdateRepositories,
  updating,
}: RfeWorkspaceCardProps) {
  const [editDialogOpen, setEditDialogOpen] = useState(false);

  return (
    <>
      <EditRepositoriesDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        workflow={workflow}
        onSave={async (data) => {
          await onUpdateRepositories(data);
          setEditDialogOpen(false);
        }}
        isSaving={updating}
      />
      <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <FolderTree className="h-5 w-5" />
          Workspace & Repositories
        </CardTitle>
        <CardDescription>Shared workspace with spec repository (specs, planning docs, agent configs) and optional supporting repos</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="text-sm text-muted-foreground">Workspace: {workflowWorkspace}</div>

        {workflow.branchName && (
          <Alert className="mt-4 border-blue-200 bg-blue-50">
            <GitBranch className="h-4 w-4 text-blue-600" />
            <AlertTitle className="text-blue-900">Feature Branch</AlertTitle>
            <AlertDescription className="text-blue-800">
              All modifications will occur on feature branch{' '}
              <code className="px-2 py-1 bg-blue-100 text-blue-900 rounded font-semibold">
                {workflow.branchName}
              </code>
              {' '}for all supplied repositories.
            </AlertDescription>
          </Alert>
        )}

        {(workflow as { parentOutcome?: string }).parentOutcome && (
          <div className="mt-2 text-sm">
            <span className="font-medium">Parent Outcome:</span>{' '}
            <Badge variant="outline">{(workflow as { parentOutcome?: string }).parentOutcome}</Badge>
          </div>
        )}
        {(workflow.umbrellaRepo || (workflow.supportingRepos || []).length > 0) && (
          <div className="mt-2 space-y-1">
            {workflow.umbrellaRepo && (
              <div className="text-sm space-y-1">
                <div>
                  <span className="font-medium">Spec Repo:</span> {workflow.umbrellaRepo.url}
                </div>
                {workflow.umbrellaRepo.branch && (
                  <div className="ml-4 text-muted-foreground">
                    Base branch: <code className="text-xs bg-muted px-1 py-0.5 rounded">{workflow.umbrellaRepo.branch}</code>
                    {workflow.branchName && (
                      <span> → Feature branch <code className="text-xs bg-blue-50 text-blue-700 px-1 py-0.5 rounded">{workflow.branchName}</code> {isSeeded ? 'set up' : 'will be set up'}</span>
                    )}
                  </div>
                )}
              </div>
            )}
            {(workflow.supportingRepos || []).map(
              (r: { url: string; branch?: string; clonePath?: string }, i: number) => (
                <div key={i} className="text-sm space-y-1">
                  <div>
                    <span className="font-medium">Supporting:</span> {r.url}
                  </div>
                  {r.branch && (
                    <div className="ml-4 text-muted-foreground">
                      Base branch: <code className="text-xs bg-muted px-1 py-0.5 rounded">{r.branch}</code>
                      {workflow.branchName && (
                        <span> → Feature branch <code className="text-xs bg-blue-50 text-blue-700 px-1 py-0.5 rounded">{workflow.branchName}</code> {isSeeded ? 'set up' : 'will be set up'}</span>
                      )}
                    </div>
                  )}
                </div>
              )
            )}
          </div>
        )}

        {!isSeeded && !seedingStatus.checking && seedingStatus.hasChecked && workflow.umbrellaRepo && (
          <Alert variant="destructive" className="mt-4">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>Spec Repository Not Seeded</AlertTitle>
            <AlertDescription className="mt-2">
              <p className="mb-3">
                Before you can start working on phases, the spec repository needs to be seeded.
                This will:
              </p>
              <ul className="list-disc list-inside space-y-1 mb-3 text-sm">
                <li>Set up the feature branch{workflow.branchName && ` (${workflow.branchName})`} from the base branch</li>
                <li>Add Spec-Kit template files for spec-driven development</li>
                <li>Add agent definition files in the .claude directory</li>
              </ul>
              {seedingError && (
                <div className="mb-3 p-2 bg-red-100 border border-red-300 rounded text-sm text-red-800">
                  <strong>Seeding Failed:</strong> {seedingError}
                </div>
              )}
              <div className="flex gap-2">
                <Button
                  onClick={() => setEditDialogOpen(true)}
                  disabled={updating}
                  size="sm"
                  variant="outline"
                >
                  <Edit className="mr-2 h-4 w-4" />
                  Edit Repositories
                </Button>
                <Button onClick={onSeedWorkflow} disabled={seeding || updating} size="sm">
                  {seeding ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Seeding Repository...
                    </>
                  ) : (
                    <>
                      <Sprout className="mr-2 h-4 w-4" />
                      {seedingError ? "Retry Seeding" : "Seed Repository"}
                    </>
                  )}
                </Button>
              </div>
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
          <div className="mt-4 flex items-center justify-between text-green-700 bg-green-50 p-3 rounded-lg">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="h-5 w-5 text-green-600" />
              <span className="text-sm font-medium">Repository seeded and ready</span>
            </div>
            <Button
              onClick={() => setEditDialogOpen(true)}
              disabled={updating}
              size="sm"
              variant="outline"
            >
              <Edit className="mr-2 h-4 w-4" />
              Edit Repositories
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
    </>
  );
}
