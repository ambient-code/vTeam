/**
 * React Query hooks for authentication
 */

import { useQuery } from '@tanstack/react-query';
import * as authApi from '../api/auth';

/**
 * Query keys for auth
 */
export const authKeys = {
  all: ['auth'] as const,
  currentUser: () => [...authKeys.all, 'currentUser'] as const,
};

/**
 * Hook to fetch current user profile
 */
export function useCurrentUser() {
  return useQuery({
    queryKey: authKeys.currentUser(),
    queryFn: authApi.getCurrentUser,
    staleTime: 5 * 60 * 1000, // 5 minutes - user info doesn't change often
  });
}

