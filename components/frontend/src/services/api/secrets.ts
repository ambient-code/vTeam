import { getApiUrl } from '@/lib/config';

export type Secret = {
  key: string;
  value: string;
};

export type SecretList = {
  items: { name: string }[];
};

export type SecretsConfig = {
  secretName: string;
};

export async function getSecretsList(projectName: string): Promise<SecretList> {
  const apiUrl = getApiUrl();
  const response = await fetch(
    `${apiUrl}/projects/${encodeURIComponent(projectName)}/secrets`
  );
  if (!response.ok) {
    throw new Error('Failed to fetch secrets list');
  }
  return response.json();
}

export async function getSecretsConfig(projectName: string): Promise<SecretsConfig> {
  const apiUrl = getApiUrl();
  const response = await fetch(
    `${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets/config`
  );
  if (!response.ok) {
    throw new Error('Failed to fetch secrets config');
  }
  return response.json();
}

export async function getSecretsValues(projectName: string): Promise<Secret[]> {
  const apiUrl = getApiUrl();
  const response = await fetch(
    `${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets`
  );
  if (!response.ok) {
    throw new Error('Failed to fetch secrets values');
  }
  const data = await response.json();
  return Object.entries<string>(data.data || {}).map(([key, value]) => ({ key, value }));
}

export async function updateSecretsConfig(
  projectName: string,
  secretName: string
): Promise<void> {
  const apiUrl = getApiUrl();
  const response = await fetch(
    `${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets/config`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ secretName }),
    }
  );
  if (!response.ok) {
    throw new Error('Failed to update secrets config');
  }
}

export async function updateSecrets(
  projectName: string,
  secrets: Secret[]
): Promise<void> {
  const apiUrl = getApiUrl();
  const data: Record<string, string> = Object.fromEntries(
    secrets.map(s => [s.key, s.value])
  );
  const response = await fetch(
    `${apiUrl}/projects/${encodeURIComponent(projectName)}/runner-secrets`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ data }),
    }
  );
  if (!response.ok) {
    throw new Error('Failed to update secrets');
  }
}
