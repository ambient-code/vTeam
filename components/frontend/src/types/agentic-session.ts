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
	knownHostsSecret?: string;
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
	websiteURL: string;
	llmSettings: LLMSettings;
	timeout: number;
	displayName?: string;
	gitConfig?: GitConfig;
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
	websiteURL?: string;
	llmSettings?: Partial<LLMSettings>;
	timeout?: number;
	gitConfig?: GitConfig;
};
