/**
 * Repository API service
 * Handles repository file and tree operations
 */

import { apiClient } from './client';

type RepoParams = {
  repo: string;
  ref: string;
  path: string;
};

type TreeEntry = {
  type: string;
  name?: string;
  path?: string;
  sha?: string;
};

type TreeResponse = {
  entries: TreeEntry[];
  sha?: string;
};

/**
 * Get file content (blob) from repository
 */
export async function getRepoBlob(
  projectName: string,
  params: RepoParams
): Promise<Response> {
  // Return raw Response for status checking and text extraction
  return apiClient.getRaw(
    `/projects/${projectName}/repo/blob`,
    {
      params: {
        repo: params.repo,
        ref: params.ref,
        path: params.path,
      },
    }
  );
}

/**
 * Get directory tree from repository
 */
export async function getRepoTree(
  projectName: string,
  params: RepoParams
): Promise<TreeResponse> {
  const url = `/projects/${encodeURIComponent(projectName)}/repo/tree`;
  
  return apiClient.get<TreeResponse>(url, {
    params: {
      repo: params.repo,
      ref: params.ref,
      path: params.path,
    },
  });
}

/**
 * Check if a file exists in repository
 */
export async function checkFileExists(
  projectName: string,
  params: RepoParams
): Promise<boolean> {
  try {
    const response = await getRepoBlob(projectName, params);
    return response.ok;
  } catch {
    return false;
  }
}

