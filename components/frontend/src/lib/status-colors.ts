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
  running: 'bg-status-info text-status-info-foreground border-status-info-border',
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
