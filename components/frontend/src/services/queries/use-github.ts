/**
 * React Query hooks for GitHub integration
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import * as githubApi from '../api/github';
import type {
  CreateForkRequest,
  CreatePRRequest,
  GitHubConnectRequest,
} from '@/types/api';

/**
 * Query keys for GitHub
 */
export const githubKeys = {
  all: ['github'] as const,
  status: () => [...githubKeys.all, 'status'] as const,
  forks: () => [...githubKeys.all, 'forks'] as const,
  forksForProject: (projectName: string, upstreamRepo?: string) =>
    [...githubKeys.forks(), projectName, upstreamRepo] as const,
  diff: (owner: string, repo: string, prNumber: number) =>
    [...githubKeys.all, 'diff', owner, repo, prNumber] as const,
};

/**
 * Hook to fetch GitHub connection status
 */
export function useGitHubStatus() {
  return useQuery({
    queryKey: githubKeys.status(),
    queryFn: githubApi.getGitHubStatus,
    // Check status less frequently
    staleTime: 60 * 1000, // 1 minute
  });
}

/**
 * Hook to fetch GitHub forks
 */
export function useGitHubForks(projectName?: string, upstreamRepo?: string) {
  return useQuery({
    queryKey: githubKeys.forksForProject(projectName || '', upstreamRepo),
    queryFn: () => githubApi.listGitHubForks(projectName, upstreamRepo),
    // Only fetch if both projectName and upstreamRepo are provided
    enabled: !!projectName && !!upstreamRepo,
    // Forks don't change often
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * Hook to fetch PR diff
 */
export function usePRDiff(
  owner: string,
  repo: string,
  prNumber: number,
  projectName?: string
) {
  return useQuery({
    queryKey: githubKeys.diff(owner, repo, prNumber),
    queryFn: () => githubApi.getPRDiff(owner, repo, prNumber, projectName),
    enabled: !!owner && !!repo && !!prNumber,
    // Diffs are relatively static
    staleTime: 60 * 1000, // 1 minute
  });
}

/**
 * Hook to connect GitHub
 */
export function useConnectGitHub() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: GitHubConnectRequest) => githubApi.connectGitHub(data),
    onSuccess: () => {
      // Invalidate status to show connected state
      queryClient.invalidateQueries({ queryKey: githubKeys.status() });
    },
  });
}

/**
 * Hook to disconnect GitHub
 */
export function useDisconnectGitHub() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: githubApi.disconnectGitHub,
    onSuccess: () => {
      // Invalidate status to show disconnected state
      queryClient.invalidateQueries({ queryKey: githubKeys.status() });
      // Clear forks cache
      queryClient.invalidateQueries({ queryKey: githubKeys.forks() });
    },
  });
}

/**
 * Hook to create a GitHub fork
 */
export function useCreateGitHubFork() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      data,
      projectName,
    }: {
      data: CreateForkRequest;
      projectName?: string;
    }) => githubApi.createGitHubFork(data, projectName),
    onSuccess: (_fork, { projectName }) => {
      // Invalidate all forks queries for this project
      if (projectName) {
        queryClient.invalidateQueries({
          queryKey: githubKeys.forksForProject(projectName),
        });
      } else {
        queryClient.invalidateQueries({ queryKey: githubKeys.forks() });
      }
    },
  });
}

/**
 * Hook to create a pull request
 */
export function useCreatePullRequest() {
  return useMutation({
    mutationFn: ({
      data,
      projectName,
    }: {
      data: CreatePRRequest;
      projectName?: string;
    }) => githubApi.createPullRequest(data, projectName),
  });
}
