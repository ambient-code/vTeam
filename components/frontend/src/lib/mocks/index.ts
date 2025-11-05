/**
 * Mock data exports
 * Central export point for all mock data
 */

export * from './projects';
export * from './sessions';
export * from './rfe';
export * from './auth';
export * from './github';
export * from './cluster';

// Re-export GitHub helper functions explicitly for clarity
export { 
  getMockGitHubStatus, 
  connectMockGitHub, 
  disconnectMockGitHub,
  resetMockGitHub 
} from './github';


