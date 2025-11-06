/**
 * Mock handlers for authentication endpoints
 */

import { NextResponse } from 'next/server';
import { createMockResponse } from '../api-handler';
import { mockAuthStatus } from '../auth';

/**
 * GET /api/me
 * Returns current user authentication status
 */
export async function handleGetMe(): Promise<NextResponse> {
  return createMockResponse('/me', () => mockAuthStatus);
}

