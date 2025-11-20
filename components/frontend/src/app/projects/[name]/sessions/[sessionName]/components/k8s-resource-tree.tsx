'use client';

import { useState } from 'react';
import { ChevronRight, ChevronDown, Box, Container, HardDrive, AlertCircle, CheckCircle2, Clock } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { getK8sResourceStatusColor, STATUS_COLORS } from '@/lib/status-colors';

type K8sResourceTreeProps = {
  jobName?: string;
  jobStatus?: string;
  pods?: Array<{
    name: string;
    phase: string;
    containers: Array<{
      name: string;
      state: string;
      exitCode?: number;
      reason?: string;
    }>;
    events?: string[];
  }>;
  pvcName?: string;
  pvcExists?: boolean;
  pvcSize?: string;
  events?: string[];
};

export function K8sResourceTree({
  jobName,
  jobStatus = 'Unknown',
  pods = [],
  pvcName,
  pvcExists,
  pvcSize,
  events = [],
}: K8sResourceTreeProps) {
  const [expandedJob, setExpandedJob] = useState(true);
  const [expandedPods, setExpandedPods] = useState<Record<string, boolean>>({});

  const getStatusIcon = (status: string) => {
    const lower = status.toLowerCase();
    if (lower.includes('running') || lower.includes('active')) return <Clock className="w-3 h-3" />;
    if (lower.includes('succeeded') || lower.includes('completed')) return <CheckCircle2 className="w-3 h-3" />;
    if (lower.includes('failed') || lower.includes('error')) return <AlertCircle className="w-3 h-3" />;
    return <Clock className="w-3 h-3" />;
  };

  const EventsDialog = ({ events, title }: { events: string[]; title: string }) => {
    const [open, setOpen] = useState(false);
    return (
      <Dialog open={open} onOpenChange={setOpen}>
        <Button variant="outline" size="sm" className="h-6 text-xs" onClick={() => setOpen(true)}>
          Events ({events.length})
        </Button>
        <DialogContent className="max-w-2xl max-h-[600px] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
            <DialogDescription>Kubernetes events for this resource</DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            {events.length === 0 ? (
              <p className="text-sm text-muted-foreground">No events</p>
            ) : (
              events.map((event, idx) => (
                <div key={idx} className="text-xs font-mono bg-muted/50 p-2 rounded border">
                  {event}
                </div>
              ))
            )}
          </div>
        </DialogContent>
      </Dialog>
    );
  };

  if (!jobName) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Kubernetes Resources</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">No job information available</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Kubernetes Resources</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {/* Job */}
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <button
              onClick={() => setExpandedJob(!expandedJob)}
              className="p-0 hover:bg-muted rounded transition-colors"
            >
              {expandedJob ? (
                <ChevronDown className="w-4 h-4 text-muted-foreground" />
              ) : (
                <ChevronRight className="w-4 h-4 text-muted-foreground" />
              )}
            </button>
            <Badge variant="outline" className="text-xs">
              <Box className="w-3 h-3 mr-1" />
              Job
            </Badge>
            <span className="text-sm font-mono">{jobName}</span>
            <Badge className={`text-xs ${getK8sResourceStatusColor(jobStatus)}`}>
              {getStatusIcon(jobStatus)}
              <span className="ml-1">{jobStatus}</span>
            </Badge>
            {events.length > 0 && <EventsDialog events={events} title={`Job: ${jobName}`} />}
          </div>

          {expandedJob && (
            <div className="ml-6 space-y-2 border-l-2 border-gray-200 pl-4">
              {/* Pods */}
              {pods.map((pod) => (
                <div key={pod.name} className="space-y-1">
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => setExpandedPods({ ...expandedPods, [pod.name]: !expandedPods[pod.name] })}
                      className="p-0 hover:bg-muted rounded transition-colors"
                    >
                      {expandedPods[pod.name] ? (
                        <ChevronDown className="w-4 h-4 text-muted-foreground" />
                      ) : (
                        <ChevronRight className="w-4 h-4 text-muted-foreground" />
                      )}
                    </button>
                    <Badge variant="outline" className="text-xs">
                      <Container className="w-3 h-3 mr-1" />
                      Pod
                    </Badge>
                    <span className="text-sm font-mono truncate max-w-xs" title={pod.name}>
                      {pod.name}
                    </span>
                    <Badge className={`text-xs ${getK8sResourceStatusColor(pod.phase)}`}>
                      {getStatusIcon(pod.phase)}
                      <span className="ml-1">{pod.phase}</span>
                    </Badge>
                    {pod.events && pod.events.length > 0 && (
                      <EventsDialog events={pod.events} title={`Pod: ${pod.name}`} />
                    )}
                  </div>

                  {expandedPods[pod.name] && (
                    <div className="ml-6 space-y-1 border-l-2 border-gray-200 pl-4">
                      {/* Containers */}
                      {pod.containers.map((container) => (
                        <div key={container.name} className="flex items-center gap-2">
                          <Badge variant="outline" className="text-xs">
                            <Box className="w-3 h-3 mr-1" />
                            Container
                          </Badge>
                          <span className="text-sm font-mono">{container.name}</span>
                          <Badge className={`text-xs ${getK8sResourceStatusColor(container.state)}`}>
                            {getStatusIcon(container.state)}
                            <span className="ml-1">{container.state}</span>
                          </Badge>
                          {container.exitCode !== undefined && (
                            <span className="text-xs text-muted-foreground">Exit: {container.exitCode}</span>
                          )}
                          {container.reason && (
                            <span className="text-xs text-muted-foreground">({container.reason})</span>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              ))}

              {/* PVC */}
              {pvcName && (
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-xs">
                    <HardDrive className="w-3 h-3 mr-1" />
                    PVC
                  </Badge>
                  <span className="text-sm font-mono">{pvcName}</span>
                  <Badge className={`text-xs ${pvcExists ? STATUS_COLORS.success : STATUS_COLORS.error}`}>
                    {pvcExists ? 'Exists' : 'Not Found'}
                  </Badge>
                  {pvcSize && <span className="text-xs text-muted-foreground">{pvcSize}</span>}
                </div>
              )}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

