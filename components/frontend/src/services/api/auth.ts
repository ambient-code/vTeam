/**
 * Authentication API service
 */

import { apiClient } from './client';

export type UserProfile = {
  authenticated: boolean;
  userId?: string;
  email?: string;
  username?: string;
  displayName?: string;
};

/**
 * Get current user profile
 */
export async function getCurrentUser(): Promise<UserProfile> {
  try {
    return await apiClient.get<UserProfile>('/me');
  } catch {
    return { authenticated: false };
  }
}

