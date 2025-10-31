import { apiClient } from './client';
import type { ProjectSettings, ProjectSettingsUpdateRequest } from '@/types/project-settings';

export async function getProjectSettings(projectName: string): Promise<ProjectSettings> {
  return apiClient.get<ProjectSettings>(`/projects/${projectName}/settings`);
}

export async function updateProjectSettings(
  projectName: string,
  data: ProjectSettingsUpdateRequest
): Promise<{ message: string; project: string }> {
  return apiClient.put<{ message: string; project: string }, ProjectSettingsUpdateRequest>(
    `/projects/${projectName}/settings`,
    data
  );
}
