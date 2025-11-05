/**
 * Mock handlers for project endpoints
 */

import { NextRequest, NextResponse } from 'next/server';
import { createMockResponse, createMockError } from '../api-handler';
import { mockProjects, getMockProject } from '../projects';
import type { CreateProjectRequest } from '@/types/api';

/**
 * GET /api/projects
 * Returns list of all projects
 */
export async function handleListProjects(): Promise<NextResponse> {
  return createMockResponse('/projects', () => ({ items: mockProjects }));
}

/**
 * POST /api/projects
 * Creates a new project
 */
export async function handleCreateProject(request: NextRequest): Promise<NextResponse> {
  try {
    const body = await request.json() as CreateProjectRequest;
    const newProject = {
      name: body?.name || 'new-project',
      displayName: body?.displayName || 'New Project',
      description: body?.description || '',
      labels: body?.labels || {},
      annotations: {},
      creationTimestamp: new Date().toISOString(),
      status: 'active' as const,
      isOpenShift: true,
    };
    mockProjects.push(newProject);
    return createMockResponse('/projects', () => newProject, 201);
  } catch (error) {
    console.error("Mock project creation failed:", error);
    return NextResponse.json(
      { error: "Failed to create project" },
      { status: 500 }
    );
  }
}

/**
 * GET /api/projects/[name]
 * Returns a single project by name
 */
export async function handleGetProject(name: string): Promise<NextResponse> {
  const project = getMockProject(name);
  if (!project) {
    return createMockError(`/projects/${name}`, 'Project not found', 404);
  }
  return createMockResponse(`/projects/${name}`, () => project);
}

/**
 * DELETE /api/projects/[name]
 * Deletes a project
 */
export async function handleDeleteProject(name: string): Promise<NextResponse> {
  const index = mockProjects.findIndex(p => p.name === name);
  if (index === -1) {
    return createMockError(`/projects/${name}`, 'Project not found', 404);
  }
  mockProjects.splice(index, 1);
  return createMockResponse(`/projects/${name}`, () => ({ 
    message: 'Project deleted successfully' 
  }));
}

