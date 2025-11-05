/**
 * Mock GitHub integration data
 */

import type { GitHubStatus } from '@/types/api';

// Mutable state for GitHub connection (pre-connected by default in developer mode)
let githubConnected = true;
let githubInstallationId: number | undefined = 12345678;

/**
 * Get current GitHub status
 */
export function getMockGitHubStatus(): GitHubStatus {
  if (githubConnected && githubInstallationId) {
    return {
      installed: true,
      installationId: githubInstallationId,
      githubUserId: 'developer',
      userId: 'developer',
      host: 'github.com',
      updatedAt: new Date().toISOString(),
    };
  }
  
  return {
    installed: false,
  };
}

/**
 * Simulate connecting GitHub (called during onboarding)
 */
export function connectMockGitHub(installationId: number): GitHubStatus {
  githubConnected = true;
  githubInstallationId = installationId;
  return getMockGitHubStatus();
}

/**
 * Simulate disconnecting GitHub
 */
export function disconnectMockGitHub(): void {
  githubConnected = false;
  githubInstallationId = undefined;
}

/**
 * Reset to initial state (for testing)
 */
export function resetMockGitHub(): void {
  githubConnected = false;
  githubInstallationId = undefined;
}

/**
 * Mock GitHub user data
 */
export const mockGitHubUser = {
  login: 'developer',
  name: 'Developer',
  email: 'developer@example.com',
  avatarUrl: 'https://avatars.githubusercontent.com/u/1?v=4',
  connected: true,
};

/**
 * Mock GitHub repositories
 */
export const mockGitHubRepos = [
  {
    name: 'api-services',
    fullName: 'demo-org/api-services',
    description: 'Backend API and microservices',
    private: false,
    defaultBranch: 'main',
    url: 'https://github.com/demo-org/api-services',
  },
  {
    name: 'web-app',
    fullName: 'demo-org/web-app',
    description: 'Full-stack web application',
    private: false,
    defaultBranch: 'main',
    url: 'https://github.com/demo-org/web-app',
  },
  {
    name: 'design-system',
    fullName: 'demo-org/design-system',
    description: 'Shared design system components',
    private: false,
    defaultBranch: 'main',
    url: 'https://github.com/demo-org/design-system',
  },
];

/**
 * Mock GitHub forks
 */
export const mockGitHubForks = [
  {
    owner: 'developer',
    repo: 'api-services',
    fullName: 'developer/api-services',
    url: 'https://github.com/developer/api-services',
    defaultBranch: 'main',
    private: false,
    createdAt: '2025-01-10T10:00:00Z',
    updatedAt: '2025-11-05T10:00:00Z',
  },
  {
    owner: 'developer',
    repo: 'web-app',
    fullName: 'developer/web-app',
    url: 'https://github.com/developer/web-app',
    defaultBranch: 'main',
    private: false,
    createdAt: '2025-01-15T10:00:00Z',
    updatedAt: '2025-11-05T10:00:00Z',
  },
];


