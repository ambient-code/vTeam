export type LLMSettings = {
  model: string;
  temperature: number;
  maxTokens: number;
};

export type ProjectDefaultSettings = {
  llmSettings: LLMSettings;
  defaultTimeout: number;
  allowedWebsiteDomains?: string[];
  maxConcurrentSessions: number;
};

export type ProjectResourceLimits = {
  maxCpuPerSession: string;
  maxMemoryPerSession: string;
  maxStoragePerSession: string;
  diskQuotaGB: number;
};

export type ObjectMeta = {
  name: string;
  namespace: string;
  creationTimestamp: string;
  uid?: string;
};

export type ProjectSettings = {
  projectName: string;
  adminUsers: string[];
  defaultSettings: ProjectDefaultSettings;
  resourceLimits: ProjectResourceLimits;
  metadata: ObjectMeta;
  repos?: ProjectRepo[];
};

export type ProjectSettingsUpdateRequest = {
  projectName: string;
  adminUsers: string[];
  defaultSettings: ProjectDefaultSettings;
  resourceLimits: ProjectResourceLimits;
  repos?: ProjectRepo[];
};

// New types for repository management
export type GroupAccess = {
  groupName: string;
  role: 'admin' | 'edit' | 'view';
};

export type ProjectRepo = {
  name: string;
  url: string;
  defaultBranch?: string;
};