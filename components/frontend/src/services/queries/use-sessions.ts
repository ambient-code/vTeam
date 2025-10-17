/**
 * React Query hooks for agentic sessions
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import * as sessionsApi from '../api/sessions';
import type {
  AgenticSession,
  CreateAgenticSessionRequest,
  StopAgenticSessionRequest,
  CloneAgenticSessionRequest,
} from '@/types/api';

/**
 * Query keys for sessions
 */
export const sessionKeys = {
  all: ['sessions'] as const,
  lists: () => [...sessionKeys.all, 'list'] as const,
  list: (projectName: string) => [...sessionKeys.lists(), projectName] as const,
  details: () => [...sessionKeys.all, 'detail'] as const,
  detail: (projectName: string, sessionName: string) =>
    [...sessionKeys.details(), projectName, sessionName] as const,
  messages: (projectName: string, sessionName: string) =>
    [...sessionKeys.detail(projectName, sessionName), 'messages'] as const,
};

/**
 * Hook to fetch sessions for a project
 */
export function useSessions(projectName: string) {
  return useQuery({
    queryKey: sessionKeys.list(projectName),
    queryFn: () => sessionsApi.listSessions(projectName),
    enabled: !!projectName,
  });
}

/**
 * Hook to fetch a single session
 */
export function useSession(projectName: string, sessionName: string) {
  return useQuery({
    queryKey: sessionKeys.detail(projectName, sessionName),
    queryFn: () => sessionsApi.getSession(projectName, sessionName),
    enabled: !!projectName && !!sessionName,
    // Poll for status updates on running sessions
    refetchInterval: (query) => {
      const session = query.state.data as AgenticSession | undefined;
      const isRunning =
        session?.status?.phase === 'Running' ||
        session?.status?.phase === 'Creating' ||
        session?.status?.phase === 'Pending';
      return isRunning ? 5000 : false; // Poll every 5 seconds if running
    },
  });
}

/**
 * Hook to fetch session messages
 */
export function useSessionMessages(projectName: string, sessionName: string, sessionPhase?: string) {
  return useQuery({
    queryKey: sessionKeys.messages(projectName, sessionName),
    queryFn: () => sessionsApi.getSessionMessages(projectName, sessionName),
    enabled: !!projectName && !!sessionName,
    // Messages are typically handled via WebSocket, so longer stale time
    staleTime: 5 * 1000, // 5 seconds
    // Poll for message updates on running sessions
    refetchInterval: () => {
      const isRunning =
        sessionPhase === 'Running' ||
        sessionPhase === 'Creating' ||
        sessionPhase === 'Pending';
      return isRunning ? 5000 : false; // Poll every 5 seconds if running
    },
  });
}

/**
 * Hook to create a session
 */
export function useCreateSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      data,
    }: {
      projectName: string;
      data: CreateAgenticSessionRequest;
    }) => sessionsApi.createSession(projectName, data),
    onSuccess: (_session, { projectName }) => {
      // Invalidate and refetch sessions list
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all', // Refetch both active and inactive queries
      });
    },
  });
}

/**
 * Hook to stop a session
 */
export function useStopSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      data,
    }: {
      projectName: string;
      sessionName: string;
      data?: StopAgenticSessionRequest;
    }) => sessionsApi.stopSession(projectName, sessionName, data),
    onSuccess: (_message, { projectName, sessionName }) => {
      // Invalidate session details to refetch status
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
        refetchType: 'all',
      });
      // Invalidate list to update session count
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

/**
 * Hook to start/restart a session
 */
export function useStartSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
    }: {
      projectName: string;
      sessionName: string;
    }) => sessionsApi.startSession(projectName, sessionName),
    onSuccess: (_response, { projectName, sessionName }) => {
      // Invalidate session details to refetch status
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
        refetchType: 'all',
      });
      // Invalidate list to update session count
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

/**
 * Hook to clone a session
 */
export function useCloneSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      data,
    }: {
      projectName: string;
      sessionName: string;
      data: CloneAgenticSessionRequest;
    }) => sessionsApi.cloneSession(projectName, sessionName, data),
    onSuccess: (_session, { projectName }) => {
      // Invalidate and refetch sessions list to show new cloned session
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all', // Refetch both active and inactive queries
      });
    },
  });
}

/**
 * Hook to delete a session
 */
export function useDeleteSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
    }: {
      projectName: string;
      sessionName: string;
    }) => sessionsApi.deleteSession(projectName, sessionName),
    onSuccess: (_data, { projectName, sessionName }) => {
      // Remove from cache
      queryClient.removeQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
      });
      // Invalidate list
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

/**
 * Hook to send chat message to interactive session
 */
export function useSendChatMessage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      content,
    }: {
      projectName: string;
      sessionName: string;
      content: string;
    }) => sessionsApi.sendChatMessage(projectName, sessionName, content),
    onSuccess: (_data, { projectName, sessionName }) => {
      // Invalidate messages to refetch
      queryClient.invalidateQueries({
        queryKey: sessionKeys.messages(projectName, sessionName),
      });
      // Invalidate session to update status
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
      });
    },
  });
}

/**
 * Hook to send control message (interrupt, end_session)
 */
export function useSendControlMessage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      type,
    }: {
      projectName: string;
      sessionName: string;
      type: 'interrupt' | 'end_session';
    }) => sessionsApi.sendControlMessage(projectName, sessionName, type),
    onSuccess: (_data, { projectName, sessionName }) => {
      // Invalidate messages to refetch
      queryClient.invalidateQueries({
        queryKey: sessionKeys.messages(projectName, sessionName),
      });
      // Invalidate session to update status
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
      });
    },
  });
}

/**
 * Hook to fetch K8s resources (job, pods, PVC) for a session
 */
export function useSessionK8sResources(projectName: string, sessionName: string) {
  return useQuery({
    queryKey: [...sessionKeys.detail(projectName, sessionName), 'k8s-resources'] as const,
    queryFn: () => sessionsApi.getSessionK8sResources(projectName, sessionName),
    enabled: !!projectName && !!sessionName,
    refetchInterval: 5000, // Poll every 5 seconds
  });
}

/**
 * Hook to continue a session (restarts the existing session)
 */
export function useContinueSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      parentSessionName,
    }: {
      projectName: string;
      parentSessionName: string;
    }) => {
      // Restart the existing session by updating its status to Creating
      return sessionsApi.startSession(projectName, parentSessionName);
    },
    onSuccess: (_response, { projectName, parentSessionName }) => {
      // Invalidate session details to refetch status
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, parentSessionName),
        refetchType: 'all',
      });
      // Invalidate list to update session count
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}
