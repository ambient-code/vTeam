/**
 * Mock authentication data
 */

import type { User, AuthStatus } from '@/types/api';

export const mockUser: User = {
  username: 'developer',
  email: 'developer@example.com',
  displayName: 'Developer',
  groups: ['developers', 'project-leads'],
  roles: ['admin', 'developer'],
};

export const mockAuthStatus: AuthStatus = {
  authenticated: true,
  user: mockUser,
};

export const mockUnauthenticatedStatus: AuthStatus = {
  authenticated: false,
};


