/**
 * React Query hooks for RFE workflows
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import * as rfeApi from '../api/rfe';
import type {
  RFEWorkflow,
  CreateRFEWorkflowRequest,
  UpdateRFEWorkflowRequest,
  StartPhaseRequest,
} from '@/types/api';

/**
 * Query keys for RFE workflows
 */
export const rfeKeys = {
  all: ['rfe'] as const,
  lists: () => [...rfeKeys.all, 'list'] as const,
  list: (projectName: string) => [...rfeKeys.lists(), projectName] as const,
  details: () => [...rfeKeys.all, 'detail'] as const,
  detail: (projectName: string, workflowId: string) =>
    [...rfeKeys.details(), projectName, workflowId] as const,
  status: (projectName: string, workflowId: string) =>
    [...rfeKeys.detail(projectName, workflowId), 'status'] as const,
  artifacts: (projectName: string, workflowId: string) =>
    [...rfeKeys.detail(projectName, workflowId), 'artifacts'] as const,
  artifact: (projectName: string, workflowId: string, artifactPath: string) =>
    [...rfeKeys.artifacts(projectName, workflowId), artifactPath] as const,
  sessions: (projectName: string, workflowId: string) =>
    [...rfeKeys.detail(projectName, workflowId), 'sessions'] as const,
  agents: (projectName: string, workflowId: string) =>
    [...rfeKeys.detail(projectName, workflowId), 'agents'] as const,
};

/**
 * Hook to fetch RFE workflows for a project
 */
export function useRfeWorkflows(projectName: string) {
  return useQuery({
    queryKey: rfeKeys.list(projectName),
    queryFn: () => rfeApi.listRfeWorkflows(projectName),
    enabled: !!projectName,
  });
}

/**
 * Hook to fetch a single RFE workflow
 */
export function useRfeWorkflow(projectName: string, workflowId: string) {
  return useQuery({
    queryKey: rfeKeys.detail(projectName, workflowId),
    queryFn: () => rfeApi.getRfeWorkflow(projectName, workflowId),
    enabled: !!projectName && !!workflowId,
  });
}

/**
 * Hook to fetch RFE workflow status
 */
export function useRfeWorkflowStatus(projectName: string, workflowId: string) {
  return useQuery({
    queryKey: rfeKeys.status(projectName, workflowId),
    queryFn: () => rfeApi.getRfeWorkflowStatus(projectName, workflowId),
    enabled: !!projectName && !!workflowId,
    // Poll every 10 seconds for active workflows
    refetchInterval: 10000,
  });
}

/**
 * Hook to fetch workflow artifacts
 */
export function useWorkflowArtifacts(projectName: string, workflowId: string) {
  return useQuery({
    queryKey: rfeKeys.artifacts(projectName, workflowId),
    queryFn: () => rfeApi.getWorkflowArtifacts(projectName, workflowId),
    enabled: !!projectName && !!workflowId,
  });
}

/**
 * Hook to fetch a specific artifact's content
 */
export function useArtifactContent(
  projectName: string,
  workflowId: string,
  artifactPath: string
) {
  return useQuery({
    queryKey: rfeKeys.artifact(projectName, workflowId, artifactPath),
    queryFn: () => rfeApi.getArtifactContent(projectName, workflowId, artifactPath),
    enabled: !!projectName && !!workflowId && !!artifactPath,
  });
}

/**
 * Hook to fetch sessions for an RFE workflow
 */
export function useRfeWorkflowSessions(projectName: string, workflowId: string) {
  return useQuery({
    queryKey: rfeKeys.sessions(projectName, workflowId),
    queryFn: () => rfeApi.getRfeWorkflowSessions(projectName, workflowId),
    enabled: !!projectName && !!workflowId,
  });
}

/**
 * Hook to fetch agents for an RFE workflow
 */
export function useRfeWorkflowAgents(projectName: string, workflowId: string) {
  return useQuery({
    queryKey: rfeKeys.agents(projectName, workflowId),
    queryFn: () => rfeApi.getRfeWorkflowAgents(projectName, workflowId),
    enabled: !!projectName && !!workflowId,
    staleTime: 5 * 60 * 1000, // 5 minutes - agents don't change frequently
  });
}

/**
 * Hook to create an RFE workflow
 */
export function useCreateRfeWorkflow() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      data,
    }: {
      projectName: string;
      data: CreateRFEWorkflowRequest;
    }) => rfeApi.createRfeWorkflow(projectName, data),
    onSuccess: (_workflow, { projectName }) => {
      // Invalidate workflows list to refetch
      queryClient.invalidateQueries({
        queryKey: rfeKeys.list(projectName),
      });
    },
  });
}

/**
 * Hook to update an RFE workflow
 */
export function useUpdateRfeWorkflow() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      workflowId,
      data,
    }: {
      projectName: string;
      workflowId: string;
      data: UpdateRFEWorkflowRequest;
    }) => rfeApi.updateRfeWorkflow(projectName, workflowId, data),
    onSuccess: (workflow: RFEWorkflow, { projectName, workflowId }) => {
      // Update cached workflow details
      queryClient.setQueryData(
        rfeKeys.detail(projectName, workflowId),
        workflow
      );
      // Invalidate list to reflect changes
      queryClient.invalidateQueries({
        queryKey: rfeKeys.list(projectName),
      });
    },
  });
}

/**
 * Hook to delete an RFE workflow
 */
export function useDeleteRfeWorkflow() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      workflowId,
    }: {
      projectName: string;
      workflowId: string;
    }) => rfeApi.deleteRfeWorkflow(projectName, workflowId),
    onSuccess: (_data, { projectName, workflowId }) => {
      // Remove from cache
      queryClient.removeQueries({
        queryKey: rfeKeys.detail(projectName, workflowId),
      });
      // Invalidate list
      queryClient.invalidateQueries({
        queryKey: rfeKeys.list(projectName),
      });
    },
  });
}

/**
 * Hook to start a workflow phase
 */
export function useStartWorkflowPhase() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      workflowId,
      data,
    }: {
      projectName: string;
      workflowId: string;
      data: StartPhaseRequest;
    }) => rfeApi.startWorkflowPhase(projectName, workflowId, data),
    onSuccess: (_sessionsCreated, { projectName, workflowId }) => {
      // Invalidate workflow to refetch updated state
      queryClient.invalidateQueries({
        queryKey: rfeKeys.detail(projectName, workflowId),
      });
      // Invalidate status to get fresh data
      queryClient.invalidateQueries({
        queryKey: rfeKeys.status(projectName, workflowId),
      });
    },
  });
}
