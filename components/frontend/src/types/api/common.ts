/**
 * Common API types and utilities
 */

export type ApiResponse<T> = {
  data: T;
  error?: never;
};

export type ApiError = {
  error: string;
  code?: string;
  details?: Record<string, unknown>;
};

export type ApiResult<T> = ApiResponse<T> | ApiError;

export function isApiError<T>(result: ApiResult<T>): result is ApiError {
  return 'error' in result && result.error !== undefined;
}

export function isApiSuccess<T>(result: ApiResult<T>): result is ApiResponse<T> {
  return 'data' in result && !('error' in result);
}

export class ApiClientError extends Error {
  constructor(
    message: string,
    public code?: string,
    public details?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'ApiClientError';
  }
}
