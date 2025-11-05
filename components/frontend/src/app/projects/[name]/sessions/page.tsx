'use client';

import { useParams, usePathname } from 'next/navigation';
import Link from 'next/link';
import { formatDistanceToNow } from 'date-fns';
import { Plus, RefreshCw, MoreVertical, Square, Trash2, ArrowRight, Brain, Star, Settings, Users } from 'lucide-react';
import { cn } from '@/lib/utils';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { PageHeader } from '@/components/page-header';
import { EmptyState } from '@/components/empty-state';
import { ErrorMessage } from '@/components/error-message';
import { SessionPhaseBadge } from '@/components/status-badge';
import { Breadcrumbs } from '@/components/breadcrumbs';

import { useSessions, useStopSession, useDeleteSession, useContinueSession } from '@/services/queries';
import { successToast, errorToast } from '@/hooks/use-toast';
import { CreateSessionDialog } from '@/components/create-session-dialog';

export default function ProjectSessionsListPage() {
  const params = useParams();
  const pathname = usePathname();
  const projectName = params?.name as string;

  // React Query hooks replace all manual state management
  const { data: sessions = [], isLoading, error, refetch } = useSessions(projectName);
  const stopSessionMutation = useStopSession();
  const deleteSessionMutation = useDeleteSession();
  const continueSessionMutation = useContinueSession();

  const base = `/projects/${projectName}`;
  const navItems = [
    { href: `${base}/sessions`, label: "Sessions", icon: Star },
    { href: `${base}/permissions`, label: "Sharing", icon: Users },
    { href: `${base}/settings`, label: "Workspace Settings", icon: Settings },
  ];

  const handleStop = async (sessionName: string) => {
    stopSessionMutation.mutate(
      { projectName, sessionName },
      {
        onSuccess: () => {
          successToast(`Session "${sessionName}" stopped successfully`);
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to stop session');
        },
      }
    );
  };


  const handleDelete = async (sessionName: string) => {
    if (!confirm(`Delete agentic session "${sessionName}"? This action cannot be undone.`)) return;
    deleteSessionMutation.mutate(
      { projectName, sessionName },
      {
        onSuccess: () => {
          successToast(`Session "${sessionName}" deleted successfully`);
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to delete session');
        },
      }
    );
  };

  const handleContinue = async (sessionName: string) => {
    continueSessionMutation.mutate(
      { projectName, parentSessionName: sessionName },
      {
        onSuccess: () => {
          successToast(`Session "${sessionName}" restarted successfully`);
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to restart session');
        },
      }
    );
  };

  // Loading state
  if (!projectName || (isLoading && sessions.length === 0)) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="animate-spin h-8 w-8" />
          <span className="ml-2">Loading sessions...</span>
        </div>
      </div>
    );
  }

  // Sort sessions by creation time (newest first)
  const sortedSessions = [...sessions].sort((a, b) => {
    const aTime = a?.metadata?.creationTimestamp ? new Date(a.metadata.creationTimestamp).getTime() : 0;
    const bTime = b?.metadata?.creationTimestamp ? new Date(b.metadata.creationTimestamp).getTime() : 0;
    return bTime - aTime;
  });

  return (
    <div className="min-h-screen bg-[#f8fafc]">
      {/* Sticky header */}
      <div className="sticky top-0 z-20 bg-white border-b">
        <div className="container mx-auto px-6 py-4">
          <Breadcrumbs
            items={[
              { label: 'Workspaces', href: '/projects' },
              { label: projectName, href: `/projects/${projectName}` },
              { label: 'Agentic Sessions' },
            ]}
            className="mb-4"
          />
          <PageHeader
            title="Agentic Sessions"
            description="Manage and monitor agentic sessions for this workspace"
          />
        </div>
      </div>

      <div className="container mx-auto p-0">
        {/* Error state */}
        {error && (
          <div className="px-6 pt-4">
            <ErrorMessage error={error} onRetry={() => refetch()} />
          </div>
        )}

        {/* Content */}
        <div className="px-6 pt-4 flex gap-6">
          {/* Sidebar Navigation */}
          <aside className="w-56 shrink-0">
            <Card>
              <CardHeader>
                <CardTitle>Workspace</CardTitle>
              </CardHeader>
              <CardContent className="p-4">
                <div className="space-y-1">
                  {navItems.map((item) => {
                    const isActive = pathname === item.href || (item.href !== base && pathname?.startsWith(item.href));
                    const Icon = item.icon;
                    return (
                      <Link key={item.href} href={item.href}>
                        <Button 
                          variant={isActive ? "secondary" : "ghost"} 
                          className={cn("w-full justify-start", isActive && "font-semibold")}
                        >
                          <Icon className="w-4 h-4 mr-2" />
                          {item.label}
                        </Button>
                      </Link>
                    );
                  })}
                </div>
              </CardContent>
            </Card>
          </aside>

          {/* Main Content */}
          <Card className="flex-1">
            <CardHeader>
              <div className="flex items-start justify-between">
                <div>
                  <CardTitle>Agentic Sessions</CardTitle>
                  <CardDescription>
                    Sessions scoped to this workspace
                  </CardDescription>
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" onClick={() => refetch()} disabled={isLoading}>
                    <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                    Refresh
                  </Button>
                  <CreateSessionDialog
                    projectName={projectName}
                    onSuccess={() => refetch()}
                    trigger={
                      <Button>
                        <Plus className="w-4 h-4 mr-2" />
                        New Session
                      </Button>
                    }
                  />
                </div>
              </div>
            </CardHeader>
            <CardContent>
          {sessions.length === 0 ? (
            <EmptyState
              icon={Brain}
              title="No sessions found"
              description="Create your first agentic session"
            />
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="min-w-[180px]">Name</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Mode</TableHead>
                    <TableHead className="hidden md:table-cell">Model</TableHead>
                    <TableHead className="hidden lg:table-cell">Created</TableHead>
                    <TableHead className="hidden xl:table-cell">Cost</TableHead>
                    <TableHead className="w-[50px]">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sortedSessions.map((session) => {
                    const sessionName = session.metadata.name;
                    const phase = session.status?.phase || 'Pending';
                    const isActionPending =
                      (stopSessionMutation.isPending && stopSessionMutation.variables?.sessionName === sessionName) ||
                      (deleteSessionMutation.isPending && deleteSessionMutation.variables?.sessionName === sessionName);

                    return (
                      <TableRow key={session.metadata?.uid || session.metadata?.name}>
                        <TableCell className="font-medium min-w-[180px]">
                          <Link
                            href={`/projects/${projectName}/sessions/${session.metadata.name}`}
                            className="text-blue-600 hover:underline hover:text-blue-800 transition-colors block"
                          >
                            <div>
                              <div className="font-medium">{session.spec.displayName || session.metadata.name}</div>
                              {session.spec.displayName && (
                                <div className="text-xs text-gray-500 font-normal">{session.metadata.name}</div>
                              )}
                            </div>
                          </Link>
                        </TableCell>
                        <TableCell>
                          <SessionPhaseBadge phase={phase} />
                        </TableCell>
                        <TableCell>
                          <span className="text-xs px-2 py-1 rounded border bg-gray-50">
                            {session.spec?.interactive ? 'Interactive' : 'Headless'}
                          </span>
                        </TableCell>
                        <TableCell className="hidden md:table-cell">
                          <span className="text-sm text-gray-600 truncate max-w-[120px] block">
                            {session.spec.llmSettings.model}
                          </span>
                        </TableCell>
                        <TableCell className="hidden lg:table-cell">
                          {session.metadata?.creationTimestamp &&
                            formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}
                        </TableCell>
                        <TableCell className="hidden xl:table-cell">
                          {session.status?.total_cost_usd ? (
                            <span className="text-sm font-mono">${session.status.total_cost_usd.toFixed(4)}</span>
                          ) : (
                            <span className="text-sm text-gray-400">—</span>
                          )}
                        </TableCell>
                        <TableCell>
                          {isActionPending ? (
                            <Button variant="ghost" size="sm" className="h-8 w-8 p-0" disabled>
                              <RefreshCw className="h-4 w-4 animate-spin" />
                            </Button>
                          ) : (
                            <SessionActions
                              sessionName={sessionName}
                              phase={phase}
                              onStop={handleStop}
                              onContinue={handleContinue}
                              onDelete={handleDelete}
                            />
                          )}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

// Session actions component extracted for clarity
type SessionActionsProps = {
  sessionName: string;
  phase: string;
  onStop: (sessionName: string) => void;
  onContinue: (sessionName: string) => void;
  onDelete: (sessionName: string) => void;
};

function SessionActions({ sessionName, phase, onStop, onContinue, onDelete }: SessionActionsProps) {
  type RowAction = {
    key: string;
    label: string;
    onClick: () => void;
    icon: React.ReactNode;
    className?: string;
  };

  const actions: RowAction[] = [];

  if (phase === 'Pending' || phase === 'Creating' || phase === 'Running') {
    actions.push({
      key: 'stop',
      label: 'Stop',
      onClick: () => onStop(sessionName),
      icon: <Square className="h-4 w-4" />,
      className: 'text-orange-600',
    });
  }

  // Allow continue for all completed sessions (converts headless to interactive)
  if (phase === 'Completed' || phase === 'Failed' || phase === 'Stopped' || phase === 'Error') {
    actions.push({
      key: 'continue',
      label: 'Continue',
      onClick: () => onContinue(sessionName),
      icon: <ArrowRight className="h-4 w-4" />,
      className: 'text-green-600',
    });
  }

  // Delete is always available except when Creating
  if (phase !== 'Creating') {
    actions.push({
      key: 'delete',
      label: 'Delete',
      onClick: () => onDelete(sessionName),
      icon: <Trash2 className="h-4 w-4" />,
      className: 'text-red-600',
    });
  }

  // Single action: show as button
  if (actions.length === 1) {
    const action = actions[0];
    return (
      <Button variant="ghost" size="sm" className="h-8 w-8 p-0" onClick={action.onClick}>
        {action.icon}
      </Button>
    );
  }

  // No actions: show disabled button
  if (actions.length === 0) {
    return (
      <Button variant="ghost" size="sm" className="h-8 w-8 p-0" disabled>
        <MoreVertical className="h-4 w-4" />
      </Button>
    );
  }

  // Multiple actions: show dropdown
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
          <MoreVertical className="h-4 w-4" />
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {actions.map((action) => (
          <DropdownMenuItem key={action.key} onClick={action.onClick} className={action.className}>
            {action.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
