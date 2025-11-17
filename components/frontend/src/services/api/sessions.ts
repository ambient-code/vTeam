/**
 * Agentic Sessions API service
 * Handles all session-related API calls
 */

import { apiClient } from './client';
import type {
  AgenticSession,
  CreateAgenticSessionRequest,
  CreateAgenticSessionResponse,
  GetAgenticSessionResponse,
  ListAgenticSessionsResponse,
  StopAgenticSessionRequest,
  StopAgenticSessionResponse,
  CloneAgenticSessionRequest,
  CloneAgenticSessionResponse,
  Message,
  GetSessionMessagesResponse,
} from '@/types/api';

/**
 * List sessions for a project
 */
export async function listSessions(projectName: string): Promise<AgenticSession[]> {
  const response = await apiClient.get<ListAgenticSessionsResponse | AgenticSession[]>(
    `/projects/${projectName}/agentic-sessions`
  );
  // Handle both wrapped and unwrapped responses
  if (Array.isArray(response)) {
    return response;
  }
  return response.items || [];
}

/**
 * Get a single session
 */
export async function getSession(
  projectName: string,
  sessionName: string
): Promise<AgenticSession> {
  const response = await apiClient.get<GetAgenticSessionResponse | AgenticSession>(
    `/projects/${projectName}/agentic-sessions/${sessionName}`
  );
  // Handle both wrapped and unwrapped responses
  if ('session' in response && response.session) {
    return response.session;
  }
  return response as AgenticSession;
}

/**
 * Create a new session
 */
export async function createSession(
  projectName: string,
  data: CreateAgenticSessionRequest
): Promise<AgenticSession> {
  const response = await apiClient.post<
    CreateAgenticSessionResponse,
    CreateAgenticSessionRequest
  >(`/projects/${projectName}/agentic-sessions`, data);
  
  // Backend returns simplified response, fetch the full session object
  return await getSession(projectName, response.name);
}

/**
 * Stop a running session
 */
export async function stopSession(
  projectName: string,
  sessionName: string,
  data?: StopAgenticSessionRequest
): Promise<string> {
  const response = await apiClient.post<
    StopAgenticSessionResponse,
    StopAgenticSessionRequest | undefined
  >(`/projects/${projectName}/agentic-sessions/${sessionName}/stop`, data);
  return response.message;
}

/**
 * Start/restart a session
 */
export async function startSession(
  projectName: string,
  sessionName: string
): Promise<{ message: string }> {
  return apiClient.post<{ message: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/start`
  );
}

/**
 * Clone an existing session
 */
export async function cloneSession(
  projectName: string,
  sessionName: string,
  data: CloneAgenticSessionRequest
): Promise<AgenticSession> {
  const response = await apiClient.post<
    CloneAgenticSessionResponse,
    CloneAgenticSessionRequest
  >(`/projects/${projectName}/agentic-sessions/${sessionName}/clone`, data);
  return response.session;
}

/**
 * Get session messages
 */
export async function getSessionMessages(
  projectName: string,
  sessionName: string
): Promise<Message[]> {
  const response = await apiClient.get<GetSessionMessagesResponse>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/messages`
  );
  return response.messages;
}

/**
 * Delete a session
 */
export async function deleteSession(
  projectName: string,
  sessionName: string
): Promise<void> {
  await apiClient.delete(`/projects/${projectName}/agentic-sessions/${sessionName}`);
}

/**
 * Send a chat message to an interactive session
 */
export async function sendChatMessage(
  projectName: string,
  sessionName: string,
  content: string
): Promise<void> {
  await apiClient.post<void, { content: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/messages`,
    { content }
  );
}

/**
 * Send a control message (interrupt, end_session) to a session
 */
export async function sendControlMessage(
  projectName: string,
  sessionName: string,
  type: 'interrupt' | 'end_session'
): Promise<void> {
  await apiClient.post<void, { type: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/messages`,
    { type }
  );
}

/**
 * Get K8s resource information (job, pods, PVC) for a session
 */
export async function getSessionK8sResources(
  projectName: string,
  sessionName: string
): Promise<{
  jobName: string;
  jobStatus?: string;
  pods?: Array<{
    name: string;
    phase: string;
    containers: Array<{
      name: string;
      state: string;
      exitCode?: number;
      reason?: string;
    }>;
  }>;
  pvcName: string;
  pvcExists: boolean;
  pvcSize?: string;
}> {
  return apiClient.get(`/projects/${projectName}/agentic-sessions/${sessionName}/k8s-resources`);
}
