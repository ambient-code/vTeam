/**
 * RFE (Request For Enhancement) Workflow API types
 */

export type WorkflowPhase =
  | 'pre'
  | 'ideate'
  | 'specify'
  | 'plan'
  | 'tasks'
  | 'implement'
  | 'review'
  | 'completed';

export type RFEWorkflowStatus = 'active' | 'completed' | 'failed' | 'paused';

export type RFESessionStatus = 'pending' | 'running' | 'completed' | 'failed';

export type GitRepository = {
  url: string;
  branch?: string;
};

export type AgentPersona = {
  persona: string;
  name: string;
  role: string;
  description: string;
};

export type ArtifactFile = {
  path: string;
  name: string;
  content: string;
  lastModified: string;
  size: number;
  type?: 'blob' | 'tree'; // blob = file, tree = directory
  agent?: string;
  phase?: string;
};

export type RFESession = {
  id: string;
  agentPersona: string;
  phase: WorkflowPhase;
  status: RFESessionStatus;
  startedAt?: string;
  completedAt?: string;
  result?: string;
  cost?: number;
};

export type PhaseResult = {
  phase: string;
  status: 'completed' | 'in_progress' | 'failed';
  agents: string[];
  artifacts: string[];
  summary: string;
  startedAt: string;
  completedAt?: string;
  metadata?: Record<string, unknown>;
};

export type JiraLink = {
  path: string;
  jiraKey: string;
};

export type RFEWorkflow = {
  id: string;
  title: string;
  description: string;
  branchName?: string;
  currentPhase?: WorkflowPhase;
  status?: RFEWorkflowStatus;
  umbrellaRepo?: GitRepository;
  supportingRepos?: GitRepository[];
  workspacePath?: string;
  parentOutcome?: string;
  agentSessions?: RFESession[];
  artifacts?: ArtifactFile[];
  createdAt: string;
  updatedAt: string;
  phaseResults?: Record<string, PhaseResult>;
  jiraLinks?: JiraLink[];
};

export type CreateRFEWorkflowRequest = {
  title: string;
  description: string;
  branchName: string;
  umbrellaRepo: GitRepository;
  supportingRepos?: GitRepository[];
  workspacePath?: string;
  parentOutcome?: string;
};

export type CreateRFEWorkflowResponse = {
  workflow: RFEWorkflow;
};

export type GetRFEWorkflowResponse = {
  workflow: RFEWorkflow;
};

export type ListRFEWorkflowsResponse = {
  workflows: RFEWorkflow[];
};

export type UpdateRFEWorkflowRequest = {
  title?: string;
  description?: string;
  status?: RFEWorkflowStatus;
  currentPhase?: WorkflowPhase;
  umbrellaRepo?: GitRepository;
  supportingRepos?: GitRepository[];
  parentOutcome?: string;
};

export type UpdateRFEWorkflowResponse = {
  workflow: RFEWorkflow;
};

export type StartPhaseRequest = {
  phase: WorkflowPhase;
  agents?: string[];
};

export type StartPhaseResponse = {
  message: string;
  sessionsCreated: string[];
};

export type GetArtifactsResponse = {
  artifacts: ArtifactFile[];
};

export type GetArtifactContentResponse = {
  artifact: ArtifactFile;
};

export type AgentProgressStatus = {
  status: string;
  sessionName?: string;
  completedAt?: string;
};

export type RFEWorkflowStatusResponse = {
  phase: WorkflowPhase;
  agentProgress: Record<string, AgentProgressStatus>;
  artifactCount: number;
  lastActivity: string;
};
