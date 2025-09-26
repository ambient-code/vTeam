export type AgenticSessionPhase = "Pending" | "Creating" | "Running" | "Completed" | "Failed" | "Stopped" | "Error";

export type LLMSettings = {
	model: string;
	temperature: number;
	maxTokens: number;
};

// Generic repo type used by RFE workflows (retains optional clonePath)
export type GitRepository = {
    url: string;
    branch?: string;
};

// Unified multi-repo session mapping
export type SessionRepoInput = {
    url: string;
    branch?: string;
};
export type SessionRepoOutput = {
    url: string;
    branch?: string;
};
export type SessionRepo = {
    input: SessionRepoInput;
    output?: SessionRepoOutput;
};

export type AgenticSessionSpec = {
	prompt: string;
	llmSettings: LLMSettings;
	timeout: number;
	displayName?: string;
	project?: string;
	interactive?: boolean;
	// Multi-repo support
	repos?: SessionRepo[];
	mainRepoIndex?: number;
};

// -----------------------------
// Content Block Types
// -----------------------------
export type TextBlock = {
	type: "text_block";
	text: string;
}
export type ThinkingBlock = {
	type: "thinking_block";
	thinking: string;
	signature: string;
}
export type ToolUseBlock = {
	type: "tool_use_block";
	id: string;
	name: string;
	input: Record<string, any>;
}
export type ToolResultBlock = {
	type: "tool_result_block";
	tool_use_id: string;
	content?: string | Array<Record<string, any>> | null;
	is_error?: boolean | null;
};

export type ContentBlock = TextBlock | ThinkingBlock | ToolUseBlock | ToolResultBlock;

export type ToolUseMessages = {
	type: "tool_use_messages";
	toolUseBlock: ToolUseBlock;
	resultBlock: ToolResultBlock;
	timestamp: string;
}
	
// -----------------------------
// Message Types
// -----------------------------
export type Message = UserMessage | AgentMessage | SystemMessage | ResultMessage | ToolUseMessages | AgentRunningMessage | AgentWaitingMessage;

export type AgentRunningMessage = {
	type: "agent_running";
	timestamp: string;
}
export type AgentWaitingMessage = {
	type: "agent_waiting";
	timestamp: string;
}

export type UserMessage = {
	type: "user_message";
	content: ContentBlock | string;
	timestamp: string;
}
export type AgentMessage = {
	type: "agent_message";
	content: ContentBlock;
	model: string;
	timestamp: string;
}
export type SystemMessage = {
	type: "system_message";
	subtype: string;
	data: Record<string, any>;
	timestamp: string;
}
export type ResultMessage = {
	type: "result_message";
	subtype: string;
	duration_ms: number;
	duration_api_ms: number;
	is_error: boolean;
	num_turns: number;
	session_id: string;
	total_cost_usd?: number | null;
	usage?: Record<string, any> | null;
	result?: string | null;
	timestamp: string;
}

// Backwards-compatible message type consumed by frontend components.
// Prefer using StreamMessage going forward.
export type MessageObject = Message;

export type AgenticSessionStatus = {
	phase: AgenticSessionPhase;
	message?: string;
	startTime?: string;
	completionTime?: string;
	jobName?: string;
  	// Storage & counts (align with CRD)
  	stateDir?: string;
	// Runner result summary fields
	subtype?: string;
	is_error?: boolean;
	num_turns?: number;
	session_id?: string;
	total_cost_usd?: number | null;
	usage?: Record<string, any> | null;
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
  	environmentVariables?: Record<string, string>;
	interactive?: boolean;
	workspacePath?: string;
	// Multi-repo support
	repos?: SessionRepo[];
	mainRepoIndex?: number;
	labels?: Record<string, string>;
	annotations?: Record<string, string>;
};

// New types for RFE workflows
export type WorkflowPhase = "pre" | "ideate" | "specify" | "plan" | "tasks" | "implement" | "review" | "completed";

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
	agent?: string;
	phase?: string;
};

export type RFESession = {
	id: string;
	agentPersona: string; // Agent persona key (e.g., "ENGINEERING_MANAGER")
	phase: WorkflowPhase;
	status: string; // "pending", "running", "completed", "failed"
	startedAt?: string;
	completedAt?: string;
	result?: string;
	cost?: number;
};

export type RFEWorkflow = {
	id: string;
	title: string;
	description: string;
  currentPhase?: WorkflowPhase; // derived in UI
  status?: "active" | "completed" | "failed" | "paused"; // derived in UI
  // New CRD-aligned repo fields
  umbrellaRepo?: GitRepository; // required in CRD, but optional in API reads
  supportingRepos?: GitRepository[];
  workspacePath?: string; // CRD-aligned optional path
  agentSessions?: RFESession[];
  artifacts?: ArtifactFile[];
	createdAt: string;
	updatedAt: string;
  phaseResults?: { [phase: string]: PhaseResult };
  jiraLinks?: Array<{ path: string; jiraKey: string }>;
};

export type CreateRFEWorkflowRequest = {
	title: string;
	description: string;
  umbrellaRepo: GitRepository;
  supportingRepos?: GitRepository[];
  workspacePath?: string;
};

export type PhaseResult = {
	phase: string;
	status: string; // "completed", "in_progress", "failed"
	agents: string[]; // agents that worked on this phase
	artifacts: string[]; // artifact paths created in this phase
	summary: string;
	startedAt: string;
	completedAt?: string;
	metadata?: { [key: string]: unknown };
};

export type RFEWorkflowStatus = {
	phase: WorkflowPhase;
	agentProgress: {
		[agentPersona: string]: {
			status: AgenticSessionPhase;
			sessionName?: string;
			completedAt?: string;
		};
	};
	artifactCount: number;
	lastActivity: string;
};

export type { Project } from "@/types/project";
