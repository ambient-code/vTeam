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
 * Get artifacts for an RFE workflow by listing files from the spec repository
 * This uses the /repo/tree endpoint to list files from the workflow's umbrella repository
 */
export async function getWorkflowArtifacts(
  projectName: string,
  workflowId: string
): Promise<ArtifactFile[]> {
  // First, get the workflow to retrieve repo URL and branch
  const workflow = await getRfeWorkflow(projectName, workflowId);
  
  if (!workflow.umbrellaRepo?.url || !workflow.branchName) {
    return [];
  }
  
  // Use the /repo/tree endpoint to list files from the repository
  const treeResponse = await apiClient.get<{ path: string; entries: Array<{ name: string; type: string; size: number }> }>(
    `/projects/${projectName}/repo/tree`,
    {
      params: {
        repo: workflow.umbrellaRepo.url,
        ref: workflow.branchName,
        path: '', // Root directory
      }
    }
  );
  
  // Convert tree entries to ArtifactFile format
  // Include both files (blob) and directories (tree)
  const artifacts: ArtifactFile[] = treeResponse.entries.map(entry => ({
    path: entry.name,
    name: entry.name,
    content: '', // Content not fetched in list view
    lastModified: new Date().toISOString(), // Not available from tree endpoint
    size: entry.size || 0,
    type: entry.type as 'blob' | 'tree', // Preserve type for UI
  }));
  
  return artifacts;
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

/**
 * Check if an RFE workflow has been seeded
 */
export async function checkRfeWorkflowSeeding(
  projectName: string,
  workflowId: string
): Promise<{ isSeeded: boolean }> {
  return apiClient.get<{ isSeeded: boolean }>(
    `/projects/${projectName}/rfe-workflows/${workflowId}/check-seeding`
  );
}

/**
 * Seed an RFE workflow
 */
export async function seedRfeWorkflow(
  projectName: string,
  workflowId: string
): Promise<void> {
  await apiClient.post<void>(
    `/projects/${projectName}/rfe-workflows/${workflowId}/seed`
  );
}

/**
 * Get Jira issue for a workflow path
 */
export async function getWorkflowJiraIssue(
  projectName: string,
  workflowId: string,
  path: string
): Promise<{ self: string; key: string } | null> {
  try {
    return await apiClient.get<{ self: string; key: string }>(
      `/projects/${projectName}/rfe-workflows/${workflowId}/jira`,
      {
        params: { path },
      }
    );
  } catch {
    return null;
  }
}

/**
 * Publish a workflow path to Jira (create or update issue)
 */
export async function publishWorkflowPathToJira(
  projectName: string,
  workflowId: string,
  path: string
): Promise<{ self: string; key: string }> {
  return apiClient.post<{ self: string; key: string }, { path: string }>(
    `/projects/${projectName}/rfe-workflows/${workflowId}/jira`,
    { path }
  );
}

/**
 * Link an existing session to an RFE workflow
 */
export async function linkSessionToWorkflow(
  projectName: string,
  workflowId: string,
  sessionName: string,
  phase?: string
): Promise<{ message: string; session: string }> {
  return apiClient.post<{ message: string; session: string }, { existingName: string; phase?: string }>(
    `/projects/${projectName}/rfe-workflows/${workflowId}/sessions/link`,
    { existingName: sessionName, phase }
  );
}
