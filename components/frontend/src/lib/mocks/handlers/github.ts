/**
 * Mock handlers for GitHub integration endpoints
 */

import { NextResponse } from 'next/server';
import { createMockResponse } from '../api-handler';
import { getMockGitHubStatus, connectMockGitHub } from '../github';
import type { GitHubConnectRequest } from '@/types/api';

/**
 * GET /api/auth/github/status
 * Returns GitHub connection status
 */
export async function handleGetGitHubStatus(): Promise<NextResponse> {
  const status = getMockGitHubStatus();
  return createMockResponse('/auth/github/status', () => status);
}

/**
 * POST /api/auth/github/install
 * Simulates GitHub App installation
 */
export async function handleInstallGitHub(request: Request): Promise<NextResponse> {
  let installationId = 12345678;
  
  try {
    const body = await request.json() as GitHubConnectRequest;
    installationId = body?.installationId || 12345678;
  } catch {
    // If JSON parsing fails, use default installation ID
  }
  
  // Simulate connecting GitHub
  const status = connectMockGitHub(installationId);
  
  return createMockResponse('/auth/github/install', () => ({
    message: 'GitHub App installed successfully',
    username: status.githubUserId || 'developer',
  }));
}

