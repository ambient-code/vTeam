import { apiClient } from "./client";

export type OOTBWorkflow = {
  id: string;
  name: string;
  description: string;
  gitUrl: string;
  branch: string;
  path?: string;
  enabled: boolean;
};

export type ListOOTBWorkflowsResponse = {
  workflows: OOTBWorkflow[];
};

export async function listOOTBWorkflows(projectName?: string): Promise<OOTBWorkflow[]> {
  const response = await apiClient.get<ListOOTBWorkflowsResponse>(
    "/workflows/ootb",
    projectName ? { params: { project: projectName } } : undefined
  );
  return response.workflows;
}

export type WorkflowCommand = {
  id: string;
  name: string;
  description: string;
  slashCommand: string;
  icon?: string;
};

export type WorkflowAgent = {
  id: string;
  name: string;
  description: string;
  tools?: string[];
};

export type WorkflowConfig = {
  name?: string;
  description?: string;
  systemPrompt?: string;
  artifactsDir?: string;
};

export type WorkflowMetadataResponse = {
  commands: WorkflowCommand[];
  agents: WorkflowAgent[];
  config?: WorkflowConfig;
};

export async function getWorkflowMetadata(
  projectName: string,
  sessionName: string
): Promise<WorkflowMetadataResponse> {
  const response = await apiClient.get<WorkflowMetadataResponse>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/workflow/metadata`
  );
  return response;
}

