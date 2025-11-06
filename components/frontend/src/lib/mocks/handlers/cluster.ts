/**
 * Mock handlers for cluster and system endpoints
 */

import { NextResponse } from 'next/server';
import { createMockResponse } from '../api-handler';
import { mockClusterInfo } from '../cluster';
import { mockVersion } from '../cluster';

/**
 * GET /api/cluster-info
 * Returns cluster information
 */
export async function handleGetClusterInfo(): Promise<NextResponse> {
  return createMockResponse('/cluster-info', () => mockClusterInfo);
}

/**
 * GET /api/version
 * Returns application version information
 */
export async function handleGetVersion(): Promise<NextResponse> {
  return createMockResponse('/version', () => mockVersion);
}

