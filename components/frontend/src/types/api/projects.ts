/**
 * Project API types
 * These types align with the backend Go structs
 */

export type ProjectStatus = 'active' | 'archived' | 'pending' | 'error' | 'terminating';

export type Project = {
  name: string;
  displayName: string; // Empty on vanilla k8s, set on OpenShift
  description?: string; // Empty on vanilla k8s, set on OpenShift
  labels: Record<string, string>;
  annotations: Record<string, string>;
  creationTimestamp: string;
  status: ProjectStatus;
  isOpenShift: boolean; // Indicates if cluster is OpenShift (affects available features)
  namespace?: string;
  resourceVersion?: string;
  uid?: string;
};

export type CreateProjectRequest = {
  name: string;
  displayName?: string; // Optional: only used on OpenShift
  description?: string; // Optional: only used on OpenShift
  labels?: Record<string, string>;
};

export type CreateProjectResponse = {
  project: Project;
};

export type UpdateProjectRequest = {
  displayName?: string;
  description?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
};

export type UpdateProjectResponse = {
  project: Project;
};

export type ListProjectsResponse = {
  items: Project[];
};

export type GetProjectResponse = {
  project: Project;
};

export type DeleteProjectResponse = {
  message: string;
};

export type PermissionRole = 'view' | 'edit' | 'admin';

export type SubjectType = 'user' | 'group';

export type PermissionAssignment = {
  subjectType: SubjectType;
  subjectName: string;
  role: PermissionRole;
  permissions?: string[];
  memberCount?: number;
  grantedAt?: string;
  grantedBy?: string;
};

export type BotAccount = {
  name: string;
  description?: string;
};

export type Model = {
  name: string;
  displayName: string;
  costPerToken: number;
  maxTokens: number;
  default?: boolean;
};

export type ResourceLimits = {
  cpu: string;
  memory: string;
  storage: string;
  maxDurationMinutes: number;
};

export type Integration = {
  type: string;
  enabled: boolean;
};

export type AvailableResources = {
  models: Model[];
  resourceLimits: ResourceLimits;
  priorityClasses: string[];
  integrations: Integration[];
};

export type ProjectDefaults = {
  model: string;
  temperature: number;
  maxTokens: number;
  timeout: number;
  priorityClass: string;
};

export type ProjectConstraints = {
  maxConcurrentSessions: number;
  maxSessionsPerUser: number;
  maxCostPerSession: number;
  maxCostPerUserPerDay: number;
  allowSessionCloning: boolean;
  allowBotAccounts: boolean;
};

export type CurrentUsage = {
  activeSessions: number;
  totalCostToday: number;
};

export type ProjectCondition = {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
};
