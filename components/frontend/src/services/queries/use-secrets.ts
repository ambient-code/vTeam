import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as secretsApi from '../api/secrets';

export function useSecretsList(projectName: string) {
  return useQuery({
    queryKey: ['secrets', 'list', projectName],
    queryFn: () => secretsApi.getSecretsList(projectName),
    enabled: !!projectName,
  });
}

export function useSecretsConfig(projectName: string) {
  return useQuery({
    queryKey: ['secrets', 'config', projectName],
    queryFn: () => secretsApi.getSecretsConfig(projectName),
    enabled: !!projectName,
  });
}

export function useSecretsValues(projectName: string) {
  return useQuery({
    queryKey: ['secrets', 'values', projectName],
    queryFn: () => secretsApi.getSecretsValues(projectName),
    enabled: !!projectName,
  });
}

export function useUpdateSecretsConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      secretName,
    }: {
      projectName: string;
      secretName: string;
    }) => secretsApi.updateSecretsConfig(projectName, secretName),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: ['secrets', 'config', projectName] });
    },
  });
}

export function useUpdateSecrets() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      secrets,
    }: {
      projectName: string;
      secrets: secretsApi.Secret[];
    }) => secretsApi.updateSecrets(projectName, secrets),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: ['secrets', 'values', projectName] });
    },
  });
}
