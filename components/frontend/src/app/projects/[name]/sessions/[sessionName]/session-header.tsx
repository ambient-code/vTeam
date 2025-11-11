"use client";

import { useState } from 'react';
import { formatDistanceToNow, format } from 'date-fns';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { RefreshCw, Square, Trash2, Copy, MoreVertical, Info } from 'lucide-react';
import { CloneSessionDialog } from '@/components/clone-session-dialog';
import { SessionDetailsModal } from '@/components/session-details-modal';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from '@/components/ui/dropdown-menu';
import type { AgenticSession } from '@/types/agentic-session';
import { getPhaseColor } from '@/utils/session-helpers';

type SessionHeaderProps = {
  session: AgenticSession;
  projectName: string;
  actionLoading: string | null;
  onRefresh: () => void;
  onStop: () => void;
  onDelete: () => void;
  durationMs?: number;
  k8sResources?: {
    pvcName?: string;
    pvcSize?: string;
  };
  messageCount: number;
};

export function SessionHeader({
  session,
  projectName,
  actionLoading,
  onRefresh,
  onStop,
  onDelete,
  durationMs,
  k8sResources,
  messageCount,
}: SessionHeaderProps) {
  const [detailsModalOpen, setDetailsModalOpen] = useState(false);
  
  const phase = session.status?.phase || "Pending";
  const canStop = phase === "Running" || phase === "Creating";
  const canDelete = phase === "Completed" || phase === "Failed" || phase === "Stopped" || phase === "Error";

  const started = session.status?.startTime 
    ? format(new Date(session.status.startTime), "PPp")
    : null;

  return (
    <>
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
          <div className="text-xs text-gray-500 mt-3">
            <span>Started {started || formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}</span>
            <span className="mx-1">•</span>
            <button 
              onClick={() => setDetailsModalOpen(true)}
              className="text-blue-600 hover:underline"
            >
              View details
            </button>
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
          
          {/* Actions dropdown menu */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm">
                <MoreVertical className="w-4 h-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => setDetailsModalOpen(true)}>
                <Info className="w-4 h-4 mr-2" />
                View details
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <CloneSessionDialog
                session={session}
                trigger={
                  <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
                    <Copy className="w-4 h-4 mr-2" />
                    Clone
                  </DropdownMenuItem>
                }
                projectName={projectName}
              />
              {canDelete && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={onDelete}
                    disabled={actionLoading === "deleting"}
                    className="text-red-600"
                  >
                    <Trash2 className="w-4 h-4 mr-2" />
                    {actionLoading === "deleting" ? "Deleting..." : "Delete"}
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <SessionDetailsModal
        session={session}
        open={detailsModalOpen}
        onOpenChange={setDetailsModalOpen}
        durationMs={durationMs}
        k8sResources={k8sResources}
        messageCount={messageCount}
      />
    </>
  );
}
