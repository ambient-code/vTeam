/**
 * Centralized status color configuration
 * Single source of truth for status badge colors across the application
 *
 * Uses CSS custom properties defined in globals.css for theme consistency.
 * The design system automatically handles light/dark mode transitions.
 */

export type StatusColorKey =
  | 'success'
  | 'error'
  | 'warning'
  | 'info'
  | 'pending'
  | 'running'
  | 'stopped'
  | 'default';

/**
 * Status color configuration using semantic design tokens
 * Automatically adapts to light/dark mode via CSS custom properties
 */
export const STATUS_COLORS: Record<StatusColorKey, string> = {
  success: 'bg-status-success text-status-success-foreground border-status-success-border',
  error: 'bg-status-error text-status-error-foreground border-status-error-border',
  warning: 'bg-status-warning text-status-warning-foreground border-status-warning-border',
  info: 'bg-status-info text-status-info-foreground border-status-info-border',
  pending: 'bg-muted text-muted-foreground border-border',
  running: 'bg-blue-600 text-white border-blue-600',
  stopped: 'bg-muted text-muted-foreground border-border',
  default: 'bg-muted text-muted-foreground border-border',
};

/**
 * Map session phases to status color keys
 */
export const SESSION_PHASE_TO_STATUS: Record<string, StatusColorKey> = {
  pending: 'warning',
  creating: 'info',
  running: 'running',
  completed: 'success',
  failed: 'error',
  error: 'error',
  stopped: 'stopped',
};

/**
 * Get status color for a session phase
 */
export function getSessionPhaseColor(phase: string): string {
  const key = SESSION_PHASE_TO_STATUS[phase.toLowerCase()] || 'default';
  return STATUS_COLORS[key];
}

/**
 * Get status color for Kubernetes resource statuses
 * Handles Job, Pod, Container, and other K8s resource states
 */
export function getK8sResourceStatusColor(status: string): string {
  const lower = status.toLowerCase();

  // Running/Active states
  if (lower.includes('running') || lower.includes('active')) {
    return STATUS_COLORS.running;
  }

  // Success states
  if (lower.includes('succeeded') || lower.includes('completed')) {
    return STATUS_COLORS.success;
  }

  // Error states
  if (lower.includes('failed') || lower.includes('error')) {
    return STATUS_COLORS.error;
  }

  // Waiting/Pending states
  if (lower.includes('waiting') || lower.includes('pending')) {
    return STATUS_COLORS.warning;
  }

  // Terminating states
  if (lower.includes('terminating') || lower.includes('terminated')) {
    return STATUS_COLORS.stopped;
  }

  // Not found states
  if (lower.includes('notfound') || lower.includes('not found')) {
    return STATUS_COLORS.warning;
  }

  // Default
  return STATUS_COLORS.default;
}
