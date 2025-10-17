"use client";

import { formatDistanceToNow } from 'date-fns';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { RefreshCw, Square, Trash2, Copy } from 'lucide-react';
import { CloneSessionDialog } from '@/components/clone-session-dialog';
import type { AgenticSession } from '@/types/agentic-session';
import { getPhaseColor } from '@/utils/session-helpers';

type SessionHeaderProps = {
  session: AgenticSession;
  projectName: string;
  actionLoading: string | null;
  onRefresh: () => void;
  onStop: () => void;
  onDelete: () => void;
};

export function SessionHeader({
  session,
  projectName,
  actionLoading,
  onRefresh,
  onStop,
  onDelete,
}: SessionHeaderProps) {
  const phase = session.status?.phase || "Pending";
  const canStop = phase === "Running" || phase === "Creating";
  const canDelete = phase === "Completed" || phase === "Failed" || phase === "Stopped" || phase === "Error";

  return (
    <div className="flex items-start justify-between">
      <div>
        <h1 className="text-2xl font-semibold flex items-center gap-2">
          <span>{session.spec.displayName || session.metadata.name}</span>
          <Badge className={getPhaseColor(phase)}>
            {phase}
          </Badge>
        </h1>
        {session.spec.displayName && (
          <div className="text-sm text-gray-500">{session.metadata.name}</div>
        )}
        <div className="text-xs text-gray-500 mt-1">
          Created {formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}
        </div>
      </div>
      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={onRefresh}
          disabled={actionLoading === "refreshing"}
        >
          <RefreshCw className={`w-4 h-4 mr-2 ${actionLoading === "refreshing" ? "animate-spin" : ""}`} />
          Refresh
        </Button>
        {canStop && (
          <Button
            variant="outline"
            size="sm"
            onClick={onStop}
            disabled={actionLoading === "stopping"}
          >
            <Square className="w-4 h-4 mr-2" />
            Stop
          </Button>
        )}
        <CloneSessionDialog
          session={session}
          trigger={
            <Button variant="outline" size="sm">
              <Copy className="w-4 h-4 mr-2" />
              Clone
            </Button>
          }
          projectName={projectName}
        />
        {canDelete && (
          <Button
            variant="destructive"
            size="sm"
            onClick={onDelete}
            disabled={actionLoading === "deleting"}
          >
            <Trash2 className="w-4 h-4 mr-2" />
            Delete
          </Button>
        )}
      </div>
    </div>
  );
}
