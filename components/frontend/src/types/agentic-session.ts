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
    status?: "pushed" | "abandoned";
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
	// Active workflow for dynamic workflow switching
	activeWorkflow?: {
		gitUrl: string;
		branch: string;
		path?: string;
	};
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
	input: Record<string, unknown>;
}
export type ToolResultBlock = {
	type: "tool_result_block";
	tool_use_id: string;
	content?: string | Array<Record<string, unknown>> | null;
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
	data: Record<string, unknown>;
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
	usage?: Record<string, unknown> | null;
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
	// Multi-repo support
	repos?: SessionRepo[];
	mainRepoIndex?: number;
	autoPushOnComplete?: boolean;
	labels?: Record<string, string>;
	annotations?: Record<string, string>;
};

export type AgentPersona = {
	persona: string;
	name: string;
	role: string;
	description: string;
};

export type { Project } from "@/types/project";
