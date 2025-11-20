/**
 * Centralized role color configuration
 * Single source of truth for permission role badge colors across the application
 *
 * Uses CSS custom properties defined in globals.css for theme consistency.
 * The design system automatically handles light/dark mode transitions.
 */

import { Eye, Edit, Shield } from "lucide-react";
import type { LucideIcon } from "lucide-react";

export type PermissionRole = 'view' | 'edit' | 'admin';

export type RoleConfig = {
  label: string;
  description: string;
  permissions: readonly string[];
  color: string;
  icon: LucideIcon;
};

/**
 * Role configuration using semantic design tokens
 * Automatically adapts to light/dark mode via CSS custom properties
 */
export const ROLE_COLORS: Record<PermissionRole, string> = {
  view: 'bg-role-view text-role-view-foreground',
  edit: 'bg-role-edit text-role-edit-foreground',
  admin: 'bg-role-admin text-role-admin-foreground',
};

/**
 * Complete role definitions including metadata
 * Used for permission management UI
 */
export const ROLE_DEFINITIONS: Record<PermissionRole, RoleConfig> = {
  view: {
    label: 'View',
    description: 'Can see sessions and duplicate to their own workspace',
    permissions: ['sessions:read', 'sessions:duplicate'] as const,
    color: ROLE_COLORS.view,
    icon: Eye,
  },
  edit: {
    label: 'Edit',
    description: 'Can create sessions in the workspace',
    permissions: ['sessions:read', 'sessions:create', 'sessions:duplicate'] as const,
    color: ROLE_COLORS.edit,
    icon: Edit,
  },
  admin: {
    label: 'Admin',
    description: 'Full workspace management access',
    permissions: ['*'] as const,
    color: ROLE_COLORS.admin,
    icon: Shield,
  },
} as const;
