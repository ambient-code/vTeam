/**
 * GitHub integration API types
 */

export type GitHubStatus = {
  installed: boolean;
  installationId?: number;
  githubUserId?: string;
  userId?: string;
  host?: string;
  updatedAt?: string;
  // Legacy OAuth fields (deprecated)
  connected?: boolean;
  username?: string;
  scopes?: string[];
};

export type GitHubFork = {
  name: string;
  fullName: string;
  owner: string;
  url: string;
  defaultBranch: string;
  private: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ListForksResponse = {
  forks: GitHubFork[];
};

export type CreateForkRequest = {
  owner: string;
  repo: string;
  organization?: string;
};

export type CreateForkResponse = {
  fork: GitHubFork;
};

export type PRDiff = {
  files: PRDiffFile[];
  additions: number;
  deletions: number;
  changes: number;
};

export type PRDiffFile = {
  filename: string;
  status: 'added' | 'modified' | 'deleted' | 'renamed';
  additions: number;
  deletions: number;
  changes: number;
  patch?: string;
};

export type GetPRDiffResponse = {
  diff: PRDiff;
};

export type CreatePRRequest = {
  owner: string;
  repo: string;
  title: string;
  body: string;
  head: string;
  base: string;
  draft?: boolean;
};

export type CreatePRResponse = {
  url: string;
  number: number;
};

export type GitHubConnectRequest = {
  installationId: number;
  // Legacy OAuth fields (deprecated)
  code?: string;
  state?: string;
};

export type GitHubConnectResponse = {
  message: string;
  username: string;
};

export type GitHubDisconnectResponse = {
  message: string;
};
