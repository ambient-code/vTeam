/**
 * Mock handlers for agentic session endpoints
 */

import { NextResponse } from 'next/server';
import { createMockResponse } from '../api-handler';
import { getMockSessionsByProject } from '../sessions';

/**
 * GET /api/projects/[name]/agentic-sessions
 * Returns list of sessions for a project
 */
export async function handleListSessions(projectName: string): Promise<NextResponse> {
  const sessions = getMockSessionsByProject(projectName);
  return createMockResponse(
    `/projects/${projectName}/agentic-sessions`, 
    () => ({ items: sessions })
  );
}

