/**
 * React Query hooks for projects
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import * as projectsApi from '../api/projects';
import type {
  Project,
  CreateProjectRequest,
  UpdateProjectRequest,
  PermissionAssignment,
} from '@/types/api';

/**
 * Query keys for projects
 */
export const projectKeys = {
  all: ['projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: () => [...projectKeys.lists()] as const,
  details: () => [...projectKeys.all, 'detail'] as const,
  detail: (name: string) => [...projectKeys.details(), name] as const,
  permissions: (name: string) => [...projectKeys.detail(name), 'permissions'] as const,
};

/**
 * Hook to fetch all projects
 */
export function useProjects() {
  return useQuery({
    queryKey: projectKeys.list(),
    queryFn: projectsApi.listProjects,
  });
}

/**
 * Hook to fetch a single project
 */
export function useProject(name: string) {
  return useQuery({
    queryKey: projectKeys.detail(name),
    queryFn: () => projectsApi.getProject(name),
    enabled: !!name,
  });
}

/**
 * Hook to create a project
 */
export function useCreateProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateProjectRequest) => projectsApi.createProject(data),
    onSuccess: () => {
      // Invalidate projects list to refetch
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

/**
 * Hook to update a project
 */
export function useUpdateProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      name,
      data,
    }: {
      name: string;
      data: UpdateProjectRequest;
    }) => projectsApi.updateProject(name, data),
    onSuccess: (project: Project) => {
      // Update cached project details
      queryClient.setQueryData(projectKeys.detail(project.name), project);
      // Invalidate lists to reflect changes
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

/**
 * Hook to delete a project
 */
export function useDeleteProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => projectsApi.deleteProject(name),
    onSuccess: (_data, name) => {
      // Remove from cache
      queryClient.removeQueries({ queryKey: projectKeys.detail(name) });
      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

/**
 * Hook to fetch project permissions
 */
export function useProjectPermissions(projectName: string) {
  return useQuery({
    queryKey: projectKeys.permissions(projectName),
    queryFn: () => projectsApi.getProjectPermissions(projectName),
    enabled: !!projectName,
  });
}

/**
 * Hook to add project permission
 */
export function useAddProjectPermission() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      permission,
    }: {
      projectName: string;
      permission: PermissionAssignment;
    }) => projectsApi.addProjectPermission(projectName, permission),
    onSuccess: (_data, { projectName }) => {
      // Invalidate permissions to refetch
      queryClient.invalidateQueries({
        queryKey: projectKeys.permissions(projectName),
      });
    },
  });
}

/**
 * Hook to remove project permission
 */
export function useRemoveProjectPermission() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      subjectType,
      subjectName,
    }: {
      projectName: string;
      subjectType: string;
      subjectName: string;
    }) =>
      projectsApi.removeProjectPermission(projectName, subjectType, subjectName),
    onSuccess: (_data, { projectName }) => {
      // Invalidate permissions to refetch
      queryClient.invalidateQueries({
        queryKey: projectKeys.permissions(projectName),
      });
    },
  });
}
