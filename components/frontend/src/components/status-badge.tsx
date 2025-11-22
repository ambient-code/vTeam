/**
 * Status Badge Component
 * Consistent badge styling for different status types
 */

import * as React from 'react';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { STATUS_COLORS, type StatusColorKey } from '@/lib/status-colors';
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
    icon: React.ComponentType<{ className?: string }>;
    label: string;
  }
> = {
  success: {
    icon: CheckCircle2,
    label: 'Success',
  },
  error: {
    icon: XCircle,
    label: 'Error',
  },
  warning: {
    icon: AlertCircle,
    label: 'Warning',
  },
  info: {
    icon: AlertCircle,
    label: 'Info',
  },
  pending: {
    icon: Clock,
    label: 'Pending',
  },
  running: {
    icon: Loader2,
    label: 'Running',
  },
  stopped: {
    icon: Square,
    label: 'Stopped',
  },
  default: {
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
  const lowerStatus = status.toLowerCase();
  // Validate status is a known variant before casting
  const normalizedStatus: StatusVariant =
    (lowerStatus in STATUS_CONFIG)
      ? (lowerStatus as StatusVariant)
      : 'default';
  const config = STATUS_CONFIG[normalizedStatus];
  const Icon = config.icon;
  const displayLabel = label || config.label;

  // Get color from centralized configuration
  const colorClasses = STATUS_COLORS[normalizedStatus as StatusColorKey] || STATUS_COLORS.default;

  return (
    <Badge
      variant="outline"
      className={cn('flex items-center gap-1.5 font-medium', colorClasses, className)}
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
