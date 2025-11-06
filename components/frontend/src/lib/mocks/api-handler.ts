/**
 * Mock API handler utilities
 * Provides helpers for creating mock API responses
 */

import { NextRequest, NextResponse } from 'next/server';
import { mockDelay, logMock } from '../mock-config';

export type MockHandler<T = unknown> = () => T | Promise<T>;

/**
 * Create a mock API response with simulated delay
 */
export async function createMockResponse<T>(
  endpoint: string,
  handler: MockHandler<T>,
  status: number = 200
): Promise<NextResponse> {
  await mockDelay();
  
  const data = await handler();
  logMock(endpoint, data);
  
  return NextResponse.json(data, { status });
}

/**
 * Create a mock error response
 */
export async function createMockError(
  endpoint: string,
  message: string,
  status: number = 500
): Promise<NextResponse> {
  await mockDelay();
  
  const errorData = { error: message };
  logMock(endpoint, errorData);
  
  return NextResponse.json(errorData, { status });
}

/**
 * Create a mock success message response
 */
export async function createMockSuccess(
  endpoint: string,
  message: string
): Promise<NextResponse> {
  return createMockResponse(endpoint, () => ({ message }));
}

/**
 * Parse request body safely
 */
export async function parseRequestBody<T>(request: NextRequest): Promise<T | null> {
  try {
    return await request.json();
  } catch {
    return null;
  }
}


