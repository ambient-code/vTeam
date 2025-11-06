/**
 * Mock Configuration
 * Controls whether mock data should be used instead of real backend calls
 */

/**
 * Enable mock mode via environment variable
 * Set NEXT_PUBLIC_USE_MOCKS=true to enable mocks
 */
export const USE_MOCKS = process.env.NEXT_PUBLIC_USE_MOCKS === 'true';

/**
 * Mock delay in milliseconds (simulates network latency)
 */
export const MOCK_DELAY = parseInt(process.env.NEXT_PUBLIC_MOCK_DELAY || '500', 10);

/**
 * Helper to simulate async delay
 */
export const mockDelay = (ms: number = MOCK_DELAY): Promise<void> => {
  return new Promise(resolve => setTimeout(resolve, ms));
};

/**
 * Log mock responses (helpful for debugging)
 */
export const LOG_MOCKS = process.env.NEXT_PUBLIC_LOG_MOCKS === 'true';

export function logMock(endpoint: string, data: unknown) {
  if (LOG_MOCKS) {
    console.log(`[MOCK] ${endpoint}`, data);
  }
}


