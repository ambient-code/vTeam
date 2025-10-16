/**
 * GitHub Integration API service
 * Handles all GitHub-related API calls
 */

import { apiClient } from './client';
import type {
  GitHubStatus,
  GitHubFork,
  ListForksResponse,
  CreateForkRequest,
  CreateForkResponse,
  GetPRDiffResponse,
  PRDiff,
  CreatePRRequest,
  CreatePRResponse,
  GitHubConnectRequest,
  GitHubConnectResponse,
  GitHubDisconnectResponse,
} from '@/types/api';

/**
 * Get GitHub connection status
 */
export async function getGitHubStatus(): Promise<GitHubStatus> {
  return apiClient.get<GitHubStatus>('/auth/github/status');
}

/**
 * Connect GitHub account via GitHub App installation
 */
export async function connectGitHub(data: GitHubConnectRequest): Promise<string> {
  const response = await apiClient.post<GitHubConnectResponse, GitHubConnectRequest>(
    '/auth/github/install',
    data
  );
  return response.username;
}

/**
 * Disconnect GitHub account
 */
export async function disconnectGitHub(): Promise<string> {
  const response = await apiClient.post<GitHubDisconnectResponse>(
    '/auth/github/disconnect'
  );
  return response.message;
}

/**
 * List user's GitHub forks
 */
export async function listGitHubForks(
  projectName?: string,
  upstreamRepo?: string
): Promise<GitHubFork[]> {
  if (!projectName) {
    throw new Error('projectName is required for listGitHubForks');
  }
  if (!upstreamRepo) {
    throw new Error('upstreamRepo is required for listGitHubForks');
  }
  const response = await apiClient.get<ListForksResponse>(
    `/projects/${projectName}/users/forks?upstreamRepo=${encodeURIComponent(upstreamRepo)}`
  );
  return response.forks;
}

/**
 * Create a GitHub fork
 */
export async function createGitHubFork(
  data: CreateForkRequest,
  projectName?: string
): Promise<GitHubFork> {
  if (!projectName) {
    throw new Error('projectName is required for createGitHubFork');
  }
  const response = await apiClient.post<CreateForkResponse, CreateForkRequest>(
    `/projects/${projectName}/users/forks`,
    data
  );
  return response.fork;
}

/**
 * Get PR diff
 */
export async function getPRDiff(
  owner: string,
  repo: string,
  prNumber: number,
  projectName?: string
): Promise<PRDiff> {
  const path = projectName
    ? `/projects/${projectName}/github/pr/${owner}/${repo}/${prNumber}/diff`
    : `/github/pr/${owner}/${repo}/${prNumber}/diff`;
  const response = await apiClient.get<GetPRDiffResponse>(path);
  return response.diff;
}

/**
 * Create a pull request
 */
export async function createPullRequest(
  data: CreatePRRequest,
  projectName?: string
): Promise<{ url: string; number: number }> {
  const path = projectName
    ? `/projects/${projectName}/github/pr`
    : '/github/pr';
  return apiClient.post<CreatePRResponse, CreatePRRequest>(path, data);
}
