/**
 * Secrets API service
 * Handles runner secrets and secret configuration
 */

import { apiClient } from './client';

export type Secret = {
  key: string;
  value: string;
};

export type SecretList = {
  items: { name: string }[];
};

export type SecretsConfig = {
  secretName: string;
};

/**
 * Get list of available secrets (K8s secrets)
 */
export async function getSecretsList(projectName: string): Promise<SecretList> {
  return apiClient.get<SecretList>(
    `/projects/${projectName}/secrets`
  );
}

/**
 * Get runner secrets configuration
 */
export async function getSecretsConfig(projectName: string): Promise<SecretsConfig> {
  return apiClient.get<SecretsConfig>(
    `/projects/${projectName}/runner-secrets/config`
  );
}

/**
 * Get runner secrets values
 */
export async function getSecretsValues(projectName: string): Promise<Secret[]> {
  // apiClient.get already unwraps the 'data' field from the response
  const data = await apiClient.get<Record<string, string>>(
    `/projects/${projectName}/runner-secrets`
  );
  return Object.entries<string>(data || {}).map(([key, value]) => ({ key, value }));
}

/**
 * Update runner secrets configuration
 */
export async function updateSecretsConfig(
  projectName: string,
  secretName: string
): Promise<void> {
  await apiClient.put<void, { secretName: string }>(
    `/projects/${projectName}/runner-secrets/config`,
    { secretName }
  );
}

/**
 * Update runner secrets values
 */
export async function updateSecrets(
  projectName: string,
  secrets: Secret[]
): Promise<void> {
  const data: Record<string, string> = Object.fromEntries(
    secrets.map(s => [s.key, s.value])
  );
  await apiClient.put<void, { data: Record<string, string> }>(
    `/projects/${projectName}/runner-secrets`,
    { data }
  );
}

/**
 * Get integration secrets values (GIT_*, JIRA_*, custom keys)
 * Hardcoded secret name: "ambient-non-vertex-integrations"
 */
export async function getIntegrationSecrets(projectName: string): Promise<Secret[]> {
  const data = await apiClient.get<Record<string, string>>(
    `/projects/${projectName}/integration-secrets`
  );
  return Object.entries<string>(data || {}).map(([key, value]) => ({ key, value }));
}

/**
 * Update integration secrets values (GIT_*, JIRA_*, custom keys)
 * Hardcoded secret name: "ambient-non-vertex-integrations"
 */
export async function updateIntegrationSecrets(
  projectName: string,
  secrets: Secret[]
): Promise<void> {
  const data: Record<string, string> = Object.fromEntries(
    secrets.map(s => [s.key, s.value])
  );
  await apiClient.put<void, { data: Record<string, string> }>(
    `/projects/${projectName}/integration-secrets`,
    { data }
  );
}
