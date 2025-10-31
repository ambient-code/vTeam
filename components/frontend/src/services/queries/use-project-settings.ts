import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as projectSettingsApi from '../api/project-settings';
import type { ProjectSettingsUpdateRequest } from '@/types/project-settings';

export const projectSettingsKeys = {
  all: ['projectSettings'] as const,
  detail: (projectName: string) => [...projectSettingsKeys.all, projectName] as const,
};

export function useProjectSettings(projectName: string) {
  return useQuery({
    queryKey: projectSettingsKeys.detail(projectName),
    queryFn: () => projectSettingsApi.getProjectSettings(projectName),
    enabled: !!projectName,
  });
}

export function useUpdateProjectSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      data,
    }: {
      projectName: string;
      data: ProjectSettingsUpdateRequest;
    }) => projectSettingsApi.updateProjectSettings(projectName, data),
    onSuccess: (_result, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: projectSettingsKeys.detail(projectName) });
      queryClient.invalidateQueries({ queryKey: ['projects', 'detail', projectName] });
    },
  });
}
