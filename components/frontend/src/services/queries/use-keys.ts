/**
 * React Query hooks for project access keys
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import * as keysApi from '../api/keys';

// Query key factory
export const keysKeys = {
  all: ['keys'] as const,
  lists: () => [...keysKeys.all, 'list'] as const,
  list: (projectName: string) => [...keysKeys.lists(), projectName] as const,
};

/**
 * Hook to list all access keys for a project
 */
export function useKeys(projectName: string) {
  return useQuery({
    queryKey: keysKeys.list(projectName),
    queryFn: () => keysApi.listKeys(projectName),
    staleTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!projectName,
  });
}

/**
 * Hook to create a new access key
 */
export function useCreateKey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectName, data }: { projectName: string; data: keysApi.CreateKeyRequest }) =>
      keysApi.createKey(projectName, data),
    onSuccess: (_data, variables) => {
      // Invalidate keys list to refetch
      queryClient.invalidateQueries({ queryKey: keysKeys.list(variables.projectName) });
    },
  });
}

/**
 * Hook to delete an access key
 */
export function useDeleteKey() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectName, keyId }: { projectName: string; keyId: string }) =>
      keysApi.deleteKey(projectName, keyId),
    onSuccess: (_data, variables) => {
      // Invalidate keys list to refetch
      queryClient.invalidateQueries({ queryKey: keysKeys.list(variables.projectName) });
    },
  });
}
