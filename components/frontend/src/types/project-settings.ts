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
};

export type ProjectSettingsUpdateRequest = {
  projectName: string;
  adminUsers: string[];
  defaultSettings: ProjectDefaultSettings;
  resourceLimits: ProjectResourceLimits;
};