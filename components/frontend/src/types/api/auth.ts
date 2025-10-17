/**
 * Authentication and authorization API types
 */

export type User = {
  username: string;
  email?: string;
  displayName?: string;
  groups?: string[];
  roles?: string[];
};

export type AuthStatus = {
  authenticated: boolean;
  user?: User;
};

export type LoginRequest = {
  username: string;
  password: string;
};

export type LoginResponse = {
  token: string;
  user: User;
};

export type LogoutResponse = {
  message: string;
};

export type RefreshTokenResponse = {
  token: string;
};
