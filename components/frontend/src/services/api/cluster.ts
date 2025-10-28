/**
 * Cluster API service
 * Handles cluster information and detection
 */

import { apiClient } from './client';

export type ClusterInfo = {
  isOpenShift: boolean;
};

/**
 * Get cluster information (OpenShift vs vanilla Kubernetes)
 * This endpoint does not require authentication
 */
export async function getClusterInfo(): Promise<ClusterInfo> {
  return apiClient.get<ClusterInfo>('/cluster-info');
}

