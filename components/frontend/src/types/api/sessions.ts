/**
 * Agentic Session API types
 * These types align with the backend Go structs and Kubernetes CRD
 */

export type UserContext = {
  userId: string;
  displayName: string;
  groups: string[];
};

export type BotAccountRef = {
  name: string;
};

export type ResourceOverrides = {
  cpu?: string;
  memory?: string;
  storageClass?: string;
  priorityClass?: string;
};

export type AgenticSessionPhase =
  | 'Pending'
  | 'Creating'
  | 'Running'
  | 'Completed'
  | 'Failed'
  | 'Stopped';

export type LLMSettings = {
  model: string;
  temperature: number;
  maxTokens: number;
};

export type SessionRepo = {
  url: string;
  branch?: string;
};

export type AgenticSessionSpec = {
  initialPrompt?: string;
  llmSettings: LLMSettings;
  timeout: number;
  displayName?: string;
  project?: string;
  interactive?: boolean;
  repos?: SessionRepo[];
  mainRepoIndex?: number;
  activeWorkflow?: {
    gitUrl: string;
    branch: string;
    path?: string;
  };
};

export type ReconciledRepo = {
  url: string;
  branch: string;
  name?: string;
  status?: 'Cloning' | 'Ready' | 'Failed';
  clonedAt?: string;
};

export type ReconciledWorkflow = {
  gitUrl: string;
  branch: string;
  status?: 'Cloning' | 'Active' | 'Failed';
  appliedAt?: string;
};

export type SessionCondition = {
  type: string;
  status: 'True' | 'False' | 'Unknown';
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
  observedGeneration?: number;
};

export type AgenticSessionStatus = {
  observedGeneration?: number;
  phase: AgenticSessionPhase;
  startTime?: string;
  completionTime?: string;
  jobName?: string;
  runnerPodName?: string;
  reconciledRepos?: ReconciledRepo[];
  reconciledWorkflow?: ReconciledWorkflow;
  sdkSessionId?: string;
  sdkRestartCount?: number;
  conditions?: SessionCondition[];
};

export type AgenticSession = {
  metadata: {
    name: string;
    namespace: string;
    creationTimestamp: string;
    uid: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
  };
  spec: AgenticSessionSpec;
  status?: AgenticSessionStatus;
};

export type CreateAgenticSessionRequest = {
  initialPrompt?: string;
  llmSettings?: Partial<LLMSettings>;
  displayName?: string;
  timeout?: number;
  project?: string;
  parent_session_id?: string;
  environmentVariables?: Record<string, string>;
  interactive?: boolean;
  repos?: SessionRepo[];
  autoPushOnComplete?: boolean;
  userContext?: UserContext;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
};

export type CreateAgenticSessionResponse = {
  message: string;
  name: string;
  uid: string;
};

export type GetAgenticSessionResponse = {
  session: AgenticSession;
};

export type ListAgenticSessionsResponse = {
  items: AgenticSession[];
};

export type StopAgenticSessionRequest = {
  reason?: string;
};

export type StopAgenticSessionResponse = {
  message: string;
};

export type CloneAgenticSessionRequest = {
  targetProject: string;
  newSessionName: string;
};

export type CloneAgenticSessionResponse = {
  session: AgenticSession;
};

// Message content block types
export type TextBlock = {
  type: 'text_block';
  text: string;
};

export type ThinkingBlock = {
  type: 'thinking_block';
  thinking: string;
  signature: string;
};

export type ToolUseBlock = {
  type: 'tool_use_block';
  id: string;
  name: string;
  input: Record<string, unknown>;
};

export type ToolResultBlock = {
  type: 'tool_result_block';
  tool_use_id: string;
  content?: string | Array<Record<string, unknown>> | null;
  is_error?: boolean | null;
};

export type ContentBlock = TextBlock | ThinkingBlock | ToolUseBlock | ToolResultBlock;

// Message types
export type UserMessage = {
  type: 'user_message';
  content: ContentBlock | string;
  timestamp: string;
};

export type AgentMessage = {
  type: 'agent_message';
  content: ContentBlock;
  model: string;
  timestamp: string;
};

export type SystemMessage = {
  type: 'system_message';
  subtype: string;
  data: Record<string, unknown>;
  timestamp: string;
};

export type ResultMessage = {
  type: 'result_message';
  subtype: string;
  duration_ms: number;
  duration_api_ms: number;
  is_error: boolean;
  num_turns: number;
  session_id: string;
  total_cost_usd?: number | null;
  usage?: Record<string, unknown> | null;
  result?: string | null;
  timestamp: string;
};

export type ToolUseMessages = {
  type: 'tool_use_messages';
  toolUseBlock: ToolUseBlock;
  resultBlock: ToolResultBlock;
  timestamp: string;
};

export type AgentRunningMessage = {
  type: 'agent_running';
  timestamp: string;
};

export type AgentWaitingMessage = {
  type: 'agent_waiting';
  timestamp: string;
};

export type Message =
  | UserMessage
  | AgentMessage
  | SystemMessage
  | ResultMessage
  | ToolUseMessages
  | AgentRunningMessage
  | AgentWaitingMessage;

export type GetSessionMessagesResponse = {
  messages: Message[];
};
