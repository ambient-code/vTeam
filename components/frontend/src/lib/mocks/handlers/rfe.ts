/**
 * Mock handlers for RFE workflow endpoints
 */

import { NextResponse } from 'next/server';
import { createMockResponse } from '../api-handler';
import { getMockRFEWorkflowsByProject } from '../rfe';

/**
 * GET /api/projects/[name]/rfe-workflows
 * Returns list of RFE workflows for a project
 */
export async function handleListRFEWorkflows(projectName: string): Promise<NextResponse> {
  const workflows = getMockRFEWorkflowsByProject(projectName);
  return createMockResponse(
    `/projects/${projectName}/rfe-workflows`, 
    () => ({ workflows })
  );
}

