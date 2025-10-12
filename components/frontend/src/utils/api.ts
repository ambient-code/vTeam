import { APIError, RepoBlob, RepoTree } from '@/types';

const API_BASE = '/api';

class APIClient {
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${API_BASE}${endpoint}`;
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      const error: APIError = await response.json().catch(() => ({
        error: `HTTP ${response.status}: ${response.statusText}`,
      }));
      throw new Error(error.error || 'API request failed');
    }

    return response.json();
  }

  // Repository browsing
  async getRepoTree(
    projectName: string,
    repo: string,
    ref: string,
    path?: string
  ): Promise<RepoTree> {
    const params = new URLSearchParams({ project: projectName, repo, ref });
    if (path) params.append('path', path);
    return this.request(`/repo/tree?${params}`);
  }

  async getRepoBlob(
    projectName: string,
    repo: string,
    ref: string,
    path: string
  ): Promise<RepoBlob> {
    const params = new URLSearchParams({ project: projectName, repo, ref, path });
    return this.request(`/repo/blob?${params}`);
  }
}

export const apiClient = new APIClient();