/**
 * Cluster information hook
 * Detects cluster type (OpenShift vs vanilla Kubernetes) by calling the /api/cluster-info endpoint
 */

import { useClusterInfo as useClusterInfoQuery } from '@/services/queries/use-cluster';

export type ClusterInfo = {
  isOpenShift: boolean;
  vertexEnabled: boolean;
  isLoading: boolean;
  isError: boolean;
};

/**
 * Detects whether the cluster is OpenShift or vanilla Kubernetes
 * and whether Vertex AI is enabled
 * Calls the /api/cluster-info endpoint which checks for project.openshift.io API group
 * and CLAUDE_CODE_USE_VERTEX environment variable
 */
export function useClusterInfo(): ClusterInfo {
  const { data, isLoading, isError } = useClusterInfoQuery();

  return {
    isOpenShift: data?.isOpenShift ?? false,
    vertexEnabled: data?.vertexEnabled ?? false,
    isLoading,
    isError,
  };
}

