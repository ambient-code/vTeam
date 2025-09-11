export type ResearchSessionPhase = "Pending" | "Creating" | "Running" | "Completed" | "Failed" | "Stopped" | "Error";

export type LLMSettings = {
	model: string;
	temperature: number;
	maxTokens: number;
};

export type ResearchSessionSpec = {
	prompt: string;
	websiteURL: string;
	llmSettings: LLMSettings;
	timeout: number;
	displayName?: string;
};

export type MessageObject = {
	content?: string;
	tool_use_id?: string;
	tool_use_name?: string;
	tool_use_input?: string;
	tool_use_is_error?: boolean;
};

export type ResearchSessionStatus = {
	phase: ResearchSessionPhase;
	message?: string;
	startTime?: string;
	completionTime?: string;
	jobName?: string;
	finalOutput?: string;
	cost?: number;
	messages?: MessageObject[];
};

export type ResearchSession = {
	metadata: {
		name: string;
		namespace: string;
		creationTimestamp: string;
		uid: string;
	};
	spec: ResearchSessionSpec;
	status?: ResearchSessionStatus;
};

export type CreateResearchSessionRequest = {
	prompt: string;
	websiteURL: string;
	llmSettings?: Partial<LLMSettings>;
	timeout?: number;
};
