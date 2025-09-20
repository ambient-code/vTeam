export type AgenticSessionPhase = "Pending" | "Creating" | "Running" | "Completed" | "Failed" | "Stopped" | "Error";

export type LLMSettings = {
	model: string;
	temperature: number;
	maxTokens: number;
};

export type GitUser = {
	name: string;
	email: string;
};

export type GitAuthentication = {
	sshKeySecret?: string;
	tokenSecret?: string;
};

export type GitRepository = {
	url: string;
	branch?: string;
	clonePath?: string;
};

export type GitConfig = {
	user?: GitUser;
	authentication?: GitAuthentication;
	repositories?: GitRepository[];
};

export type AgenticSessionSpec = {
	prompt: string;
	llmSettings: LLMSettings;
	timeout: number;
	displayName?: string;
	gitConfig?: GitConfig;
	project?: string;
};

export type MessageObject = {
	content?: string;
	tool_use_id?: string;
	tool_use_name?: string;
	tool_use_input?: string;
	tool_use_is_error?: boolean;
};

export type AgenticSessionStatus = {
	phase: AgenticSessionPhase;
	message?: string;
	startTime?: string;
	completionTime?: string;
	jobName?: string;
	finalOutput?: string;
	cost?: number;
	messages?: MessageObject[];
};

export type AgenticSession = {
	metadata: {
		name: string;
		namespace: string;
		creationTimestamp: string;
		uid: string;
	};
	spec: AgenticSessionSpec;
	status?: AgenticSessionStatus;
};

export type CreateAgenticSessionRequest = {
	prompt: string;
	llmSettings?: Partial<LLMSettings>;
	timeout?: number;
	gitConfig?: GitConfig;
	project?: string;
  	environmentVariables?: Record<string, string>;
	// New fields for agent sessions
	agentPersona?: string;
	workflowPhase?: string;
	parentRFE?: string;
	sharedWorkspace?: string;
};

// New types for RFE workflows
export type WorkflowPhase = "pre" | "specify" | "plan" | "tasks" | "review" | "completed";

export type AgentPersona = {
	persona: string;
	name: string;
	role: string;
	expertise: string[];
	description?: string;
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
	targetRepoUrl: string;
	targetRepoBranch: string;
  agentSessions?: RFESession[];
  artifacts?: ArtifactFile[];
	createdAt: string;
	updatedAt: string;
  phaseResults?: { [phase: string]: PhaseResult };
};

export type CreateRFEWorkflowRequest = {
	title: string;
	description: string;
	targetRepoUrl: string;
	targetRepoBranch: string;
	gitUserName?: string;
	gitUserEmail?: string;
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
