/**
 * Version API service
 * Handles version-related API calls
 */

import { apiClient } from './client';

type VersionResponse = {
  version: string;
};

/**
 * Get application version
 */
export async function getVersion(): Promise<string> {
  const response = await apiClient.get<VersionResponse>('/version');
  return response.version;
}
