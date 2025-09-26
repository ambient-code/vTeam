import { APIError, RepoBlob, RepoTree, Workspace } from '@/types';

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
    repo: string,
    ref: string,
    path?: string
  ): Promise<RepoTree> {
    const params = new URLSearchParams({ repo, ref });
    if (path) params.append('path', path);
    return this.request(`/repo/tree?${params}`);
  }

  async getRepoBlob(
    repo: string,
    ref: string,
    path: string
  ): Promise<RepoBlob> {
    const params = new URLSearchParams({ repo, ref, path });
    return this.request(`/repo/blob?${params}`);
  }

  // Workspace management
  async listWorkspaces(projectName: string): Promise<{ workflows: Workspace[] }> {
    return this.request(`/projects/${projectName}/rfe-workflows`);
  }

  // WebSocket connection
  createWebSocketConnection(projectName: string, sessionId: string, wsBase?: string): WebSocket {
    let wsUrl: string;
    if (wsBase) {
      const base = wsBase.replace(/\/$/, '');
      wsUrl = `${base}/projects/${projectName}/sessions/${sessionId}/ws`;
    } else if (typeof window !== 'undefined') {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      wsUrl = `${protocol}//${window.location.host}/api/projects/${projectName}/sessions/${sessionId}/ws`;
    } else {
      // SSR fallback (not used by browser)
      wsUrl = `/api/projects/${projectName}/sessions/${sessionId}/ws`;
    }
    return new WebSocket(wsUrl);
  }
}

export const apiClient = new APIClient();