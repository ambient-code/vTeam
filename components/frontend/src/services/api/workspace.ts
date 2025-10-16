/**
 * Workspace API service
 * Handles session workspace (PVC) operations
 */

import { apiClient } from './client';

export type WorkspaceItem = {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
  modifiedAt: string;
};

export type ListWorkspaceResponse = {
  items: WorkspaceItem[];
};

/**
 * List workspace directory contents
 */
export async function listWorkspace(
  projectName: string,
  sessionName: string,
  path?: string
): Promise<WorkspaceItem[]> {
  const params = path ? { path } : undefined;
  const response = await apiClient.get<ListWorkspaceResponse>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/workspace`,
    { params }
  );
  return response.items;
}

/**
 * Read workspace file content
 */
export async function readWorkspaceFile(
  projectName: string,
  sessionName: string,
  path: string
): Promise<string> {
  const response = await apiClient.getRaw(
    `/projects/${projectName}/agentic-sessions/${sessionName}/workspace/${encodeURIComponent(path)}`
  );
  if (!response.ok) {
    throw new Error('Failed to read workspace file');
  }
  return response.text();
}

/**
 * Write workspace file content
 */
export async function writeWorkspaceFile(
  projectName: string,
  sessionName: string,
  path: string,
  content: string
): Promise<void> {
  await apiClient.putText(
    `/projects/${projectName}/agentic-sessions/${sessionName}/workspace/${encodeURIComponent(path)}`,
    content
  );
}

/**
 * Get GitHub diff for a session repository
 */
export async function getSessionGitHubDiff(
  projectName: string,
  sessionName: string,
  repoIndex: number,
  repoPath: string
): Promise<{ files: { added: number; removed: number }; total_added: number; total_removed: number }> {
  const response = await apiClient.get<{
    files?: { added?: number; removed?: number };
    total_added?: number;
    total_removed?: number;
  }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/github/diff`,
    {
      params: { repoIndex: String(repoIndex), repoPath },
    }
  );
  return {
    files: {
      added: response.files?.added ?? 0,
      removed: response.files?.removed ?? 0,
    },
    total_added: response.total_added ?? 0,
    total_removed: response.total_removed ?? 0,
  };
}

/**
 * Push session changes to GitHub
 */
export async function pushSessionToGitHub(
  projectName: string,
  sessionName: string,
  repoIndex: number,
  repoPath: string
): Promise<void> {
  await apiClient.post<void, { repoIndex: number; repoPath: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/github/push`,
    { repoIndex, repoPath }
  );
}

/**
 * Abandon session changes (reset to upstream)
 */
export async function abandonSessionChanges(
  projectName: string,
  sessionName: string,
  repoIndex: number,
  repoPath: string
): Promise<void> {
  await apiClient.post<void, { repoIndex: number; repoPath: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/github/abandon`,
    { repoIndex, repoPath }
  );
}

