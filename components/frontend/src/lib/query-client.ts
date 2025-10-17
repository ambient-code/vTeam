/**
 * React Query client configuration
 */

import { QueryClient, DefaultOptions } from '@tanstack/react-query';

const queryConfig: DefaultOptions = {
  queries: {
    // Stale time: 5 minutes - data is considered fresh for 5 minutes
    staleTime: 5 * 60 * 1000,

    // Cache time: 10 minutes - unused data is garbage collected after 10 minutes
    gcTime: 10 * 60 * 1000,

    // Retry failed requests once
    retry: 1,

    // Retry delay with exponential backoff
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),

    // Refetch on window focus in production
    refetchOnWindowFocus: process.env.NODE_ENV === 'production',

    // Don't refetch on mount if data is fresh
    refetchOnMount: false,
  },
  mutations: {
    // Retry mutations once
    retry: 1,
  },
};

/**
 * Creates a new QueryClient instance
 * Use this in server components or for testing
 */
export function makeQueryClient() {
  return new QueryClient({
    defaultOptions: queryConfig,
  });
}

/**
 * Browser query client singleton
 * Ensures we only create one client instance in the browser
 */
let browserQueryClient: QueryClient | undefined = undefined;

export function getQueryClient() {
  if (typeof window === 'undefined') {
    // Server: always create a new query client
    return makeQueryClient();
  } else {
    // Browser: reuse the same query client
    if (!browserQueryClient) {
      browserQueryClient = makeQueryClient();
    }
    return browserQueryClient;
  }
}
