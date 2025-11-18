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

/**
 * Git merge status types
 */
export type GitMergeStatus = {
  canMergeClean: boolean;
  localChanges: number;
  remoteCommitsAhead: number;
  conflictingFiles: string[];
  remoteBranchExists: boolean;
};

/**
 * Get git merge status for artifacts directory
 */
export async function getGitMergeStatus(
  projectName: string,
  sessionName: string,
  path: string = 'artifacts',
  branch: string = 'main'
): Promise<GitMergeStatus> {
  const response = await apiClient.get<GitMergeStatus>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/git/merge-status`,
    { params: { path, branch } }
  );
  return response;
}

/**
 * Pull changes from remote
 */
export async function gitPull(
  projectName: string,
  sessionName: string,
  path: string = 'artifacts',
  branch: string = 'main'
): Promise<void> {
  await apiClient.post<void, { path: string; branch: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/git/pull`,
    { path, branch }
  );
}

/**
 * Push changes to remote
 */
export async function gitPush(
  projectName: string,
  sessionName: string,
  path: string = 'artifacts',
  branch: string = 'main',
  message?: string
): Promise<void> {
  await apiClient.post<void, { path: string; branch: string; message?: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/git/push`,
    { path, branch, message }
  );
}

/**
 * Create a new git branch
 */
export async function gitCreateBranch(
  projectName: string,
  sessionName: string,
  branchName: string,
  path: string = 'artifacts'
): Promise<void> {
  await apiClient.post<void, { path: string; branchName: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/git/create-branch`,
    { path, branchName }
  );
}

/**
 * List remote branches
 */
export async function gitListBranches(
  projectName: string,
  sessionName: string,
  path: string = 'artifacts'
): Promise<string[]> {
  const response = await apiClient.get<{ branches: string[] }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/git/list-branches`,
    { params: { path } }
  );
  return response.branches;
}

/**
 * Git status types
 */
export type GitStatus = {
  branch?: string;
  remoteUrl?: string;
  ahead?: number;
  behind?: number;
  staged?: number;
  unstaged?: number;
  untracked?: number;
  hasRemote?: boolean;
  initialized?: boolean;
  hasChanges?: boolean;
  uncommittedFiles?: number;
  filesAdded?: number;
  filesRemoved?: number;
  totalAdded?: number;
  totalRemoved?: number;
};

/**
 * Get git status for a directory
 */
export async function gitStatus(
  projectName: string,
  sessionName: string,
  path: string
): Promise<GitStatus> {
  const response = await apiClient.get<GitStatus>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/git/status`,
    { params: { path } }
  );
  return response;
}

/**
 * Configure git remote for a directory
 */
export async function configureGitRemote(
  projectName: string,
  sessionName: string,
  path: string,
  remoteUrl: string,
  branch: string = 'main'
): Promise<void> {
  await apiClient.post<void, { path: string; remoteUrl: string; branch: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/git/configure-remote`,
    { path, remoteUrl, branch }
  );
}

/**
 * Synchronize git (commit, pull, push)
 */
export async function synchronizeGit(
  projectName: string,
  sessionName: string,
  path: string,
  message?: string,
  branch?: string
): Promise<void> {
  await apiClient.post<void, { path: string; message?: string; branch?: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/git/synchronize`,
    { path, message, branch }
  );
}

