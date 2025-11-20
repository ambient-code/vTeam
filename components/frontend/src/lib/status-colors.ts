/**
 * Centralized status color configuration
 * Single source of truth for status badge colors across the application
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
 * Status color configuration
 * Uses pastel backgrounds in light mode, solid backgrounds in dark mode
 */
export const STATUS_COLORS: Record<StatusColorKey, string> = {
  success: 'bg-green-100 text-green-800 border-green-200 dark:bg-green-700 dark:text-white dark:border-green-700',
  error: 'bg-red-100 text-red-800 border-red-200 dark:bg-red-700 dark:text-white dark:border-red-700',
  warning: 'bg-yellow-100 text-yellow-800 border-yellow-200 dark:bg-yellow-600 dark:text-white dark:border-yellow-600',
  info: 'bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-600 dark:text-white dark:border-blue-600',
  pending: 'bg-muted text-foreground border-border dark:bg-slate-600 dark:text-white dark:border-slate-600',
  running: 'bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-600 dark:text-white dark:border-blue-600',
  stopped: 'bg-muted text-foreground border-border dark:bg-slate-600 dark:text-white dark:border-slate-600',
  default: 'bg-muted text-foreground border-border dark:bg-slate-600 dark:text-white dark:border-slate-600',
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
