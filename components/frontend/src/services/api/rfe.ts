/**
 * RFE (Request For Enhancement) Workflow API service
 * Handles all RFE workflow-related API calls
 */

import { apiClient } from './client';
import type {
  RFEWorkflow,
  CreateRFEWorkflowRequest,
  GetRFEWorkflowResponse,
  ListRFEWorkflowsResponse,
  UpdateRFEWorkflowRequest,
  StartPhaseRequest,
  StartPhaseResponse,
  GetArtifactsResponse,
  GetArtifactContentResponse,
  RFEWorkflowStatusResponse,
  ArtifactFile,
  AgenticSession,
  AgentPersona,
} from '@/types/api';

/**
 * List RFE workflows for a project
 */
export async function listRfeWorkflows(projectName: string): Promise<RFEWorkflow[]> {
  const response = await apiClient.get<ListRFEWorkflowsResponse | RFEWorkflow[]>(
    `/projects/${projectName}/rfe-workflows`
  );
  // Handle both wrapped and unwrapped responses
  if (Array.isArray(response)) {
    return response;
  }
  return response.workflows || [];
}

/**
 * Get a single RFE workflow
 */
export async function getRfeWorkflow(
  projectName: string,
  workflowId: string
): Promise<RFEWorkflow> {
  const response = await apiClient.get<GetRFEWorkflowResponse | RFEWorkflow>(
    `/projects/${projectName}/rfe-workflows/${workflowId}`
  );
  // Handle both wrapped and unwrapped responses
  if ('workflow' in response && response.workflow) {
    return response.workflow;
  }
  return response as RFEWorkflow;
}

/**
 * Create a new RFE workflow
 */
export async function createRfeWorkflow(
  projectName: string,
  data: CreateRFEWorkflowRequest
): Promise<RFEWorkflow> {
  return apiClient.post<RFEWorkflow, CreateRFEWorkflowRequest>(
    `/projects/${projectName}/rfe-workflows`,
    data
  );
}

/**
 * Update an existing RFE workflow
 */
export async function updateRfeWorkflow(
  projectName: string,
  workflowId: string,
  data: UpdateRFEWorkflowRequest
): Promise<RFEWorkflow> {
  return apiClient.put<RFEWorkflow, UpdateRFEWorkflowRequest>(
    `/projects/${projectName}/rfe-workflows/${workflowId}`,
    data
  );
}

/**
 * Delete an RFE workflow
 */
export async function deleteRfeWorkflow(
  projectName: string,
  workflowId: string
): Promise<void> {
  await apiClient.delete(`/projects/${projectName}/rfe-workflows/${workflowId}`);
}

/**
 * Start a workflow phase
 */
export async function startWorkflowPhase(
  projectName: string,
  workflowId: string,
  data: StartPhaseRequest
): Promise<string[]> {
  const response = await apiClient.post<StartPhaseResponse, StartPhaseRequest>(
    `/projects/${projectName}/rfe-workflows/${workflowId}/phase`,
    data
  );
  return response.sessionsCreated;
}

/**
 * Get RFE workflow status
 */
export async function getRfeWorkflowStatus(
  projectName: string,
  workflowId: string
): Promise<RFEWorkflowStatusResponse> {
  return apiClient.get<RFEWorkflowStatusResponse>(
    `/projects/${projectName}/rfe-workflows/${workflowId}/status`
  );
}

/**
 * Get artifacts for an RFE workflow
 */
export async function getWorkflowArtifacts(
  projectName: string,
  workflowId: string
): Promise<ArtifactFile[]> {
  const response = await apiClient.get<GetArtifactsResponse>(
    `/projects/${projectName}/rfe-workflows/${workflowId}/artifacts`
  );
  return response.artifacts;
}

/**
 * Get a specific artifact's content
 */
export async function getArtifactContent(
  projectName: string,
  workflowId: string,
  artifactPath: string
): Promise<ArtifactFile> {
  const response = await apiClient.get<GetArtifactContentResponse>(
    `/projects/${projectName}/rfe-workflows/${workflowId}/artifacts/${encodeURIComponent(artifactPath)}`
  );
  return response.artifact;
}

/**
 * Get sessions for an RFE workflow
 */
export async function getRfeWorkflowSessions(
  projectName: string,
  workflowId: string
): Promise<AgenticSession[]> {
  const response = await apiClient.get<{ sessions: AgenticSession[] }>(
    `/projects/${projectName}/rfe-workflows/${workflowId}/sessions`
  );
  return response.sessions;
}

/**
 * Get agents for an RFE workflow
 */
export async function getRfeWorkflowAgents(
  projectName: string,
  workflowId: string
): Promise<AgentPersona[]> {
  const response = await apiClient.get<{ agents: AgentPersona[] }>(
    `/projects/${projectName}/rfe-workflows/${workflowId}/agents`
  );
  return response.agents;
}
