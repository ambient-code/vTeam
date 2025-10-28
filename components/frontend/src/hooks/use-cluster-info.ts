/**
 * Cluster information hook
 * Detects cluster type (OpenShift vs vanilla Kubernetes) by calling the /api/cluster-info endpoint
 */

import { useClusterInfo as useClusterInfoQuery } from '@/services/queries/use-cluster';

export type ClusterInfo = {
  isOpenShift: boolean;
  isLoading: boolean;
  isError: boolean;
};

/**
 * Detects whether the cluster is OpenShift or vanilla Kubernetes
 * Calls the /api/cluster-info endpoint which checks for project.openshift.io API group
 */
export function useClusterInfo(): ClusterInfo {
  const { data, isLoading, isError } = useClusterInfoQuery();

  return {
    isOpenShift: data?.isOpenShift ?? false,
    isLoading,
    isError,
  };
}

