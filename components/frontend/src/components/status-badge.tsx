/**
 * Status Badge Component
 * Consistent badge styling for different status types
 */

import * as React from 'react';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import {
  CheckCircle2,
  XCircle,
  AlertCircle,
  Clock,
  Loader2,
  Square,
} from 'lucide-react';

export type StatusVariant =
  | 'success'
  | 'error'
  | 'warning'
  | 'info'
  | 'pending'
  | 'running'
  | 'stopped'
  | 'default';

export type StatusBadgeProps = {
  status: StatusVariant | string;
  label?: string;
  showIcon?: boolean;
  className?: string;
  pulse?: boolean;
};

const STATUS_CONFIG: Record<
  StatusVariant,
  {
    color: string;
    icon: React.ComponentType<{ className?: string }>;
    label: string;
  }
> = {
  success: {
    color: 'bg-green-100 text-green-800 border-green-200',
    icon: CheckCircle2,
    label: 'Success',
  },
  error: {
    color: 'bg-red-100 text-red-800 border-red-200',
    icon: XCircle,
    label: 'Error',
  },
  warning: {
    color: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    icon: AlertCircle,
    label: 'Warning',
  },
  info: {
    color: 'bg-blue-100 text-blue-800 border-blue-200',
    icon: AlertCircle,
    label: 'Info',
  },
  pending: {
    color: 'bg-gray-100 text-gray-800 border-gray-200',
    icon: Clock,
    label: 'Pending',
  },
  running: {
    color: 'bg-blue-100 text-blue-800 border-blue-200',
    icon: Loader2,
    label: 'Running',
  },
  stopped: {
    color: 'bg-gray-100 text-gray-800 border-gray-200',
    icon: Square,
    label: 'Stopped',
  },
  default: {
    color: 'bg-gray-100 text-gray-800 border-gray-200',
    icon: AlertCircle,
    label: 'Unknown',
  },
};

export function StatusBadge({
  status,
  label,
  showIcon = true,
  className,
  pulse = false,
}: StatusBadgeProps) {
  const normalizedStatus = (status.toLowerCase() as StatusVariant) || 'default';
  const config = STATUS_CONFIG[normalizedStatus] || STATUS_CONFIG.default;
  const Icon = config.icon;
  const displayLabel = label || config.label;

  return (
    <Badge
      variant="outline"
      className={cn('flex items-center gap-1.5 font-medium', config.color, className)}
    >
      {showIcon && (
        <Icon
          className={cn(
            'h-3 w-3',
            pulse && 'animate-pulse',
            normalizedStatus === 'running' && 'animate-spin'
          )}
        />
      )}
      {displayLabel}
    </Badge>
  );
}

/**
 * Session phase badge with appropriate styling
 */
export function SessionPhaseBadge({ phase }: { phase: string }) {
  const statusMap: Record<string, StatusVariant> = {
    pending: 'pending',
    creating: 'pending',
    running: 'running',
    completed: 'success',
    failed: 'error',
    stopped: 'stopped',
    error: 'error',
  };

  const status = statusMap[phase.toLowerCase()] || 'default';

  return <StatusBadge status={status} label={phase} pulse={status === 'running'} />;
}

/**
 * Project status badge
 */
export function ProjectStatusBadge({ status }: { status: string }) {
  const statusMap: Record<string, StatusVariant> = {
    active: 'success',
    archived: 'warning',
    pending: 'pending',
    error: 'error',
    terminating: 'warning',
  };

  const variant = statusMap[status.toLowerCase()] || 'default';

  return <StatusBadge status={variant} label={status} />;
}

/**
 * RFE workflow phase badge
 */
export function RFEPhaseBadge({ phase }: { phase: string }) {
  const statusMap: Record<string, StatusVariant> = {
    pre: 'pending',
    ideate: 'info',
    specify: 'info',
    plan: 'info',
    tasks: 'info',
    implement: 'running',
    review: 'warning',
    completed: 'success',
  };

  const status = statusMap[phase.toLowerCase()] || 'default';

  return <StatusBadge status={status} label={phase} pulse={status === 'running'} />;
}
