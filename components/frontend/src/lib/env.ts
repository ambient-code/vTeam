/**
 * Environment variable configuration
 * Provides type-safe access to environment variables
 */

type Environment = 'development' | 'production' | 'test';

type EnvConfig = {
  // Node environment
  NODE_ENV: Environment;

  // Backend API URL (server-side only)
  BACKEND_URL: string;

  // GitHub configuration (public)
  GITHUB_APP_SLUG: string;

  // Version information (public, optional)
  VTEAM_VERSION?: string;

  // Feedback URL (public, optional)
  FEEDBACK_URL?: string;

  // OpenShift identity (server-side only, optional)
  OC_TOKEN?: string;
  OC_USER?: string;
  OC_EMAIL?: string;
  ENABLE_OC_WHOAMI?: boolean;
};

function getEnv(key: string, defaultValue?: string): string {
  const value = process.env[key];
  if (value === undefined || value === '') {
    if (defaultValue !== undefined) {
      return defaultValue;
    }
    throw new Error(`Missing required environment variable: ${key}`);
  }
  return value;
}

function getOptionalEnv(key: string): string | undefined {
  const value = process.env[key];
  return value === '' ? undefined : value;
}

function getBooleanEnv(key: string, defaultValue = false): boolean {
  const value = process.env[key];
  if (value === undefined || value === '') {
    return defaultValue;
  }
  return value === '1' || value.toLowerCase() === 'true';
}

/**
 * Server-side environment configuration
 * Only available in server components and API routes
 */
export const env: EnvConfig = {
  NODE_ENV: (process.env.NODE_ENV || 'development') as Environment,
  BACKEND_URL: getEnv('BACKEND_URL', 'http://localhost:8080/api'),
  GITHUB_APP_SLUG: getEnv('GITHUB_APP_SLUG', 'ambient-code-vteam'),
  VTEAM_VERSION: getOptionalEnv('VTEAM_VERSION') || 'latest',
  FEEDBACK_URL: getOptionalEnv('FEEDBACK_URL'),
  OC_TOKEN: getOptionalEnv('OC_TOKEN'),
  OC_USER: getOptionalEnv('OC_USER'),
  OC_EMAIL: getOptionalEnv('OC_EMAIL'),
  ENABLE_OC_WHOAMI: getBooleanEnv('ENABLE_OC_WHOAMI', false),
};

/**
 * Public environment variables
 * These are available in both server and client components
 */
export const publicEnv = {
  GITHUB_APP_SLUG: env.GITHUB_APP_SLUG,
  VTEAM_VERSION: env.VTEAM_VERSION,
  FEEDBACK_URL: env.FEEDBACK_URL,
};

/**
 * Check if running in development mode
 */
export const isDevelopment = env.NODE_ENV === 'development';

/**
 * Check if running in production mode
 */
export const isProduction = env.NODE_ENV === 'production';

/**
 * Check if running in test mode
 */
export const isTest = env.NODE_ENV === 'test';
