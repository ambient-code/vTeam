import type { SessionMessage } from "@/types";
import type { ToolUseBlock, ToolResultBlock } from "@/types/agentic-session";

export type RawWireMessage = SessionMessage & { payload?: unknown; timestamp?: string };

export type InnerEnvelope = {
  type?: string;
  timestamp?: string;
  payload?: Record<string, unknown> | string;
  partial?: { id: string; index: number; total: number; data: string };
  seq?: number;
};

export type ToolUseBlockWithTimestamp = {
  block: ToolUseBlock;
  timestamp: string;
};

export type ToolResultBlockWithTimestamp = {
  block: ToolResultBlock;
  timestamp: string;
};

export type GitStatus = {
  initialized: boolean;
  hasChanges: boolean;
  uncommittedFiles: number;
  filesAdded: number;
  filesRemoved: number;
  totalAdded: number;
  totalRemoved: number;
};

export type DirectoryOption = {
  type: 'artifacts' | 'repo' | 'workflow';
  name: string;
  path: string;
};

export type DirectoryRemote = {
  url: string;
  branch: string;
};

export type WorkflowConfig = {
  id: string;
  name: string;
  description: string;
  gitUrl: string;
  branch: string;
  path?: string;
  enabled: boolean;
};

export type WorkflowCommand = {
  id: string;
  name: string;
  slashCommand: string;
  description?: string;
  icon?: string;
};

export type WorkflowAgent = {
  id: string;
  name: string;
  description?: string;
};

export type WorkflowMetadata = {
  commands: Array<WorkflowCommand>;
  agents: Array<WorkflowAgent>;
};

