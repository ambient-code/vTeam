/**
 * React Query hooks for version
 */

import { useQuery } from '@tanstack/react-query';
import * as versionApi from '../api/version';

/**
 * Query keys for version
 */
export const versionKeys = {
  all: ['version'] as const,
  current: () => [...versionKeys.all, 'current'] as const,
};

/**
 * Hook to fetch application version
 */
export function useVersion() {
  return useQuery({
    queryKey: versionKeys.current(),
    queryFn: versionApi.getVersion,
    staleTime: 5 * 60 * 1000, // Cache version for 5 minutes
    retry: false, // Don't retry on failure
  });
}
