// Project types for the Ambient Agentic Runner frontend
// Based on the OpenAPI contract specifications from backend tests

export interface ObjectMeta {
  name: string;
  namespace?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  creationTimestamp?: string;
  resourceVersion?: string;
  uid?: string;
}

export interface BotAccount {
  name: string;
  description?: string;
}

export type PermissionRole = "view" | "edit" | "admin";

export type SubjectType = "user" | "group";

export type PermissionAssignment = {
  subjectType: SubjectType;
  subjectName: string;
  role: PermissionRole;
  permissions?: string[];
  memberCount?: number;
  grantedAt?: string;
  grantedBy?: string;
};

export interface Model {
  name: string;
  displayName: string;
  costPerToken: number;
  maxTokens: number;
  default?: boolean;
}

export interface ResourceLimits {
  cpu: string;
  memory: string;
  storage: string;
  maxDurationMinutes: number;
}

export interface Integration {
  type: string;
  enabled: boolean;
}

export interface AvailableResources {
  models: Model[];
  resourceLimits: ResourceLimits;
  priorityClasses: string[];
  integrations: Integration[];
}

export interface ProjectDefaults {
  model: string;
  temperature: number;
  maxTokens: number;
  timeout: number;
  priorityClass: string;
}

export interface ProjectConstraints {
  maxConcurrentSessions: number;
  maxSessionsPerUser: number;
  maxCostPerSession: number;
  maxCostPerUserPerDay: number;
  allowSessionCloning: boolean;
  allowBotAccounts: boolean;
}

export interface AmbientProjectSpec {
  displayName: string;
  description?: string;
  bots?: BotAccount[];
  groupAccess?: PermissionAssignment[];
  availableResources: AvailableResources;
  defaults: ProjectDefaults;
  constraints: ProjectConstraints;
}

export interface CurrentUsage {
  activeSessions: number;
  totalCostToday: number;
}

export interface ProjectCondition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
}

export interface AmbientProjectStatus {
  phase?: string;
  botsCreated?: number;
  groupBindingsCreated?: number;
  lastReconciled?: string;
  currentUsage?: CurrentUsage;
  conditions?: ProjectCondition[];
}


// Flat DTO used by frontend UIs when backend formats Project responses
export type Project = {
  name: string;
  displayName?: string; // Empty on vanilla k8s, set on OpenShift
  description?: string; // Empty on vanilla k8s, set on OpenShift
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  creationTimestamp?: string;
  status?: string; // e.g., "Active" | "Pending" | "Error"
  isOpenShift?: boolean; // Indicates if cluster is OpenShift (affects available features)
};


export interface CreateProjectRequest {
  name: string;
  displayName?: string; // Optional: only used on OpenShift
  description?: string; // Optional: only used on OpenShift
}

export type ProjectPhase = "Pending" | "Active" | "Error" | "Terminating";
