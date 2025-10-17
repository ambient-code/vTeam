/**
 * API service for project access keys
 */

import { apiClient } from './client';

// Types
export type ProjectKey = {
  id: string;
  name: string;
  description?: string;
  createdAt?: string;
  lastUsedAt?: string;
  role?: 'view' | 'edit' | 'admin';
};

export type CreateKeyRequest = {
  name: string;
  description?: string;
  role?: 'view' | 'edit' | 'admin';
};

export type CreateKeyResponse = {
  id: string;
  name: string;
  key: string;
  description?: string;
  role?: 'view' | 'edit' | 'admin';
};

export type ListKeysResponse = {
  items: ProjectKey[];
};

/**
 * List all access keys for a project
 */
export async function listKeys(projectName: string): Promise<ProjectKey[]> {
  const response = await apiClient.get<ListKeysResponse>(`/projects/${projectName}/keys`);
  return response.items || [];
}

/**
 * Create a new access key for a project
 */
export async function createKey(
  projectName: string,
  data: CreateKeyRequest
): Promise<CreateKeyResponse> {
  return apiClient.post<CreateKeyResponse, CreateKeyRequest>(`/projects/${projectName}/keys`, data);
}

/**
 * Delete an access key
 */
export async function deleteKey(projectName: string, keyId: string): Promise<void> {
  await apiClient.delete(`/projects/${projectName}/keys/${keyId}`);
}
