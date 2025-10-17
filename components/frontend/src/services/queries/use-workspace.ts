/**
 * React Query hooks for workspace operations
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import * as workspaceApi from '../api/workspace';

/**
 * Query keys for workspace
 */
export const workspaceKeys = {
  all: ['workspace'] as const,
  lists: () => [...workspaceKeys.all, 'list'] as const,
  list: (projectName: string, sessionName: string, path?: string) =>
    [...workspaceKeys.lists(), projectName, sessionName, path] as const,
  files: () => [...workspaceKeys.all, 'file'] as const,
  file: (projectName: string, sessionName: string, path: string) =>
    [...workspaceKeys.files(), projectName, sessionName, path] as const,
  diffs: () => [...workspaceKeys.all, 'diff'] as const,
  diff: (projectName: string, sessionName: string, repoIndex: number) =>
    [...workspaceKeys.diffs(), projectName, sessionName, repoIndex] as const,
};

/**
 * Hook to list workspace directory
 */
export function useWorkspaceList(
  projectName: string,
  sessionName: string,
  path?: string,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: workspaceKeys.list(projectName, sessionName, path),
    queryFn: () => workspaceApi.listWorkspace(projectName, sessionName, path),
    enabled: !!projectName && !!sessionName && (options?.enabled ?? true),
    staleTime: 5 * 1000, // 5 seconds
  });
}

/**
 * Hook to read workspace file
 */
export function useWorkspaceFile(
  projectName: string,
  sessionName: string,
  path: string,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: workspaceKeys.file(projectName, sessionName, path),
    queryFn: () => workspaceApi.readWorkspaceFile(projectName, sessionName, path),
    enabled: !!projectName && !!sessionName && !!path && (options?.enabled ?? true),
    staleTime: 10 * 1000, // 10 seconds
  });
}

/**
 * Hook to write workspace file
 */
export function useWriteWorkspaceFile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      path,
      content,
    }: {
      projectName: string;
      sessionName: string;
      path: string;
      content: string;
    }) => workspaceApi.writeWorkspaceFile(projectName, sessionName, path, content),
    onSuccess: (_data, { projectName, sessionName, path }) => {
      // Invalidate the specific file
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.file(projectName, sessionName, path),
      });
      // Invalidate parent directory listing
      const parentPath = path.split('/').slice(0, -1).join('/');
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.list(projectName, sessionName, parentPath || undefined),
      });
    },
  });
}

/**
 * Hook to get GitHub diff for a session repo
 */
export function useSessionGitHubDiff(
  projectName: string,
  sessionName: string,
  repoIndex: number,
  repoPath: string,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: workspaceKeys.diff(projectName, sessionName, repoIndex),
    queryFn: () =>
      workspaceApi.getSessionGitHubDiff(projectName, sessionName, repoIndex, repoPath),
    enabled: !!projectName && !!sessionName && (options?.enabled ?? true),
    staleTime: 10 * 1000, // 10 seconds
  });
}

/**
 * Hook to fetch all GitHub diffs for session repos
 */
export function useAllSessionGitHubDiffs(
  projectName: string,
  sessionName: string,
  repos: Array<{ input: { url: string; branch: string }; output?: { url: string; branch: string } }> | undefined,
  deriveRepoFolder: (url: string) => string,
  options?: { enabled?: boolean; sessionPhase?: string }
) {
  const queryClient = useQueryClient();

  return useQuery({
    queryKey: [...workspaceKeys.diffs(), projectName, sessionName, 'all'],
    queryFn: async () => {
      if (!repos || repos.length === 0) return {};

      const diffs = await Promise.all(
        repos.map(async (repo, idx) => {
          const url = repo?.input?.url || "";
          if (!url) return { idx, diff: { files: { added: 0, removed: 0 }, total_added: 0, total_removed: 0 } };

          const folder = deriveRepoFolder(url);
          const repoPath = `/sessions/${sessionName}/workspace/${folder}`;

          try {
            const diff = await queryClient.fetchQuery({
              queryKey: workspaceKeys.diff(projectName, sessionName, idx),
              queryFn: () => workspaceApi.getSessionGitHubDiff(projectName, sessionName, idx, repoPath),
            });
            return { idx, diff };
          } catch {
            return { idx, diff: { files: { added: 0, removed: 0 }, total_added: 0, total_removed: 0 } };
          }
        })
      );

      const totals: Record<number, { files: { added: number; removed: number }; total_added: number; total_removed: number }> = {};
      diffs.forEach(({ idx, diff }) => {
        totals[idx] = diff;
      });
      return totals;
    },
    enabled: !!projectName && !!sessionName && !!repos && (options?.enabled ?? true),
    staleTime: 10 * 1000, // 10 seconds
    // Poll for diff updates when session is running
    refetchInterval: () => {
      const isRunning =
        options?.sessionPhase === 'Running' ||
        options?.sessionPhase === 'Creating' ||
        options?.sessionPhase === 'Pending';
      return isRunning ? 10000 : false; // Poll every 10 seconds if running
    },
  });
}

/**
 * Hook to push session changes to GitHub
 */
export function usePushSessionToGitHub() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      repoIndex,
      repoPath,
    }: {
      projectName: string;
      sessionName: string;
      repoIndex: number;
      repoPath: string;
    }) => workspaceApi.pushSessionToGitHub(projectName, sessionName, repoIndex, repoPath),
    onSuccess: (_data, { projectName, sessionName, repoIndex }) => {
      // Invalidate diff to show changes were pushed
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.diff(projectName, sessionName, repoIndex),
      });
      // Invalidate session to update status
      queryClient.invalidateQueries({
        queryKey: ['sessions', 'detail', projectName, sessionName],
      });
    },
  });
}

/**
 * Hook to abandon session changes
 */
export function useAbandonSessionChanges() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      repoIndex,
      repoPath,
    }: {
      projectName: string;
      sessionName: string;
      repoIndex: number;
      repoPath: string;
    }) => workspaceApi.abandonSessionChanges(projectName, sessionName, repoIndex, repoPath),
    onSuccess: (_data, { projectName, sessionName, repoIndex }) => {
      // Invalidate diff to show changes were abandoned
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.diff(projectName, sessionName, repoIndex),
      });
      // Invalidate workspace to refresh file listing
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.lists(),
      });
    },
  });
}

