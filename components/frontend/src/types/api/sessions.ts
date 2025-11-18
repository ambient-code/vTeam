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
  | 'Stopped'
  | 'Error';

export type LLMSettings = {
  model: string;
  temperature: number;
  maxTokens: number;
};

export type SessionRepoInput = {
  url: string;
  branch?: string;
};

export type SessionRepoOutput = {
  url: string;
  branch?: string;
};

export type SessionRepoStatus = 'pushed' | 'abandoned';

export type SessionRepo = {
  input: SessionRepoInput;
  output?: SessionRepoOutput;
  status?: SessionRepoStatus;
};

export type AgenticSessionSpec = {
  prompt: string;
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

export type AgenticSessionStatus = {
  phase: AgenticSessionPhase;
  message?: string;
  startTime?: string;
  completionTime?: string;
  jobName?: string;
  stateDir?: string;
  subtype?: string;
  is_error?: boolean;
  num_turns?: number;
  session_id?: string;
  total_cost_usd?: number | null;
  usage?: Record<string, unknown> | null;
  result?: string | null;
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
  prompt: string;
  llmSettings?: Partial<LLMSettings>;
  displayName?: string;
  timeout?: number;
  project?: string;
  parent_session_id?: string;
  environmentVariables?: Record<string, string>;
  interactive?: boolean;
  workspacePath?: string;
  repos?: SessionRepo[];
  mainRepoIndex?: number;
  autoPushOnComplete?: boolean;
  userContext?: UserContext;
  botAccount?: BotAccountRef;
  resourceOverrides?: ResourceOverrides;
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
