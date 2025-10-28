/**
 * React Query hooks for cluster information
 */

import { useQuery } from '@tanstack/react-query';
import { getClusterInfo } from '@/services/api/cluster';

/**
 * Hook to get cluster information (OpenShift vs Kubernetes)
 * Detects cluster type by calling /api/cluster-info endpoint
 */
export function useClusterInfo() {
  return useQuery({
    queryKey: ['cluster-info'],
    queryFn: getClusterInfo,
    staleTime: Infinity, // Cluster type doesn't change, cache forever
    retry: 3, // Retry a few times on failure
  });
}

