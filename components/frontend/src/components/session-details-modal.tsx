"use client";

import { format } from 'date-fns';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import type { AgenticSession } from '@/types/agentic-session';
import { getPhaseColor } from '@/utils/session-helpers';

function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  
  if (hours > 0) {
    const remainingMinutes = minutes % 60;
    const remainingSeconds = seconds % 60;
    return `${hours}h ${remainingMinutes}m ${remainingSeconds}s`;
  } else if (minutes > 0) {
    const remainingSeconds = seconds % 60;
    return `${minutes}m ${remainingSeconds}s`;
  } else {
    return `${seconds}s`;
  }
}

type SessionDetailsModalProps = {
  session: AgenticSession;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  durationMs?: number;
  k8sResources?: {
    pvcName?: string;
    pvcSize?: string;
  };
  messageCount: number;
};

export function SessionDetailsModal({
  session,
  open,
  onOpenChange,
  durationMs,
  k8sResources,
  messageCount,
}: SessionDetailsModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px] max-h-[90vh] overflow-y-auto">
        <DialogHeader className="space-y-3">
          <DialogTitle>Session Details</DialogTitle>
        </DialogHeader>
        
        <div className="space-y-4">
          <div className="space-y-3">
            <div className="flex items-start gap-3">
              <span className="font-semibold text-gray-700 min-w-[100px]">Status:</span>
              <Badge className={getPhaseColor(session.status?.phase || "Pending")}>
                {session.status?.phase || "Pending"}
              </Badge>
            </div>
            
            <div className="flex items-start gap-3">
              <span className="font-semibold text-gray-700 min-w-[100px]">Model:</span>
              <span className="text-gray-900">{session.spec.llmSettings.model}</span>
            </div>
            
            <div className="flex items-start gap-3">
              <span className="font-semibold text-gray-700 min-w-[100px]">Temperature:</span>
              <span className="text-gray-900">{session.spec.llmSettings.temperature}</span>
            </div>
            
            <div className="flex items-start gap-3">
              <span className="font-semibold text-gray-700 min-w-[100px]">Mode:</span>
              <span className="text-gray-900">{session.spec?.interactive ? "Interactive" : "Headless"}</span>
            </div>
            
            {session.status?.startTime && (
              <div className="flex items-start gap-3">
                <span className="font-semibold text-gray-700 min-w-[100px]">Started:</span>
                <span className="text-gray-900">{format(new Date(session.status.startTime), "PPp")}</span>
              </div>
            )}
            
            <div className="flex items-start gap-3">
              <span className="font-semibold text-gray-700 min-w-[100px]">Duration:</span>
              <span className="text-gray-900">{typeof durationMs === "number" ? formatDuration(durationMs) : "-"}</span>
            </div>
            
            {k8sResources?.pvcName && (
              <div className="flex items-start gap-3">
                <span className="font-semibold text-gray-700 min-w-[100px]">PVC:</span>
                <span className="text-gray-900 font-mono break-all">{k8sResources.pvcName}</span>
              </div>
            )}
            
            {k8sResources?.pvcSize && (
              <div className="flex items-start gap-3">
                <span className="font-semibold text-gray-700 min-w-[100px]">PVC Size:</span>
                <span className="text-gray-900">{k8sResources.pvcSize}</span>
              </div>
            )}
            
            {session.status?.jobName && (
              <div className="flex items-start gap-3">
                <span className="font-semibold text-gray-700 min-w-[100px]">K8s Job:</span>
                <span className="text-gray-900 font-mono break-all">{session.status.jobName}</span>
              </div>
            )}
            
            <div className="flex items-start gap-3">
              <span className="font-semibold text-gray-700 min-w-[100px]">Messages:</span>
              <span className="text-gray-900">{messageCount}</span>
            </div>
          </div>
          
          {session.spec.prompt && (
            <div className="pt-2">
              <div className="mb-2">
                <span className="font-semibold text-gray-700">Session prompt:</span>
              </div>
              <div className="max-h-[200px] overflow-y-auto p-4 bg-gray-50 rounded-md border border-gray-200">
                <p className="whitespace-pre-wrap text-sm text-gray-800 leading-relaxed">{session.spec.prompt}</p>
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

