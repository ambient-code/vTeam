/**
 * Base API client with error handling
 * Provides typed fetch wrapper with automatic error parsing
 */

import { ApiClientError, isApiError, type ApiResult } from '@/types/api';

type RequestConfig = RequestInit & {
  params?: Record<string, string | number | boolean>;
};

/**
 * Base URL for API requests
 * This client is only used client-side to call Next.js API routes
 */
export function getApiBaseUrl(): string {
  // Client-side only: use relative path (Next.js will proxy to backend)
  // or use NEXT_PUBLIC_API_URL if configured
  return process.env.NEXT_PUBLIC_API_URL || '/api';
}

/**
 * Build URL with query parameters
 * Note: This is only used client-side to call Next.js API routes
 */
function buildUrl(path: string, params?: Record<string, string | number | boolean>): string {
  const baseUrl = getApiBaseUrl();
  
  // Normalize paths for concatenation
  const normalizedBase = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  
  // Build the full path
  let fullUrl = `${normalizedBase}${normalizedPath}`;
  
  // Add query parameters if provided
  if (params) {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      searchParams.append(key, String(value));
    });
    const queryString = searchParams.toString();
    if (queryString) {
      fullUrl += `?${queryString}`;
    }
  }

  return fullUrl;
}

/**
 * Parse API response
 * Handles both success and error responses
 */
async function parseResponse<T>(response: Response): Promise<T> {
  const contentType = response.headers.get('content-type');
  const isJson = contentType?.includes('application/json');

  // Parse JSON response
  const data: ApiResult<T> = isJson ? await response.json() : await response.text();

  // Handle error responses
  if (!response.ok) {
    // Only check isApiError if data is an object (not a string/HTML response)
    if (typeof data === 'object' && data !== null && isApiError(data)) {
      throw new ApiClientError(data.error, data.code, data.details);
    }
    throw new ApiClientError(
      `HTTP ${response.status}: ${response.statusText}`,
      String(response.status)
    );
  }

  // Handle success responses
  if (isJson && typeof data === 'object' && data !== null && 'data' in data) {
    return (data as { data: T }).data;
  }

  return data as T;
}

/**
 * Make an API request with automatic error handling
 */
async function request<T>(
  path: string,
  config: RequestConfig = {}
): Promise<T> {
  const { params, ...fetchConfig } = config;
  const url = buildUrl(path, params);

  const defaultHeaders: HeadersInit = {
    'Content-Type': 'application/json',
  };

  // Merge headers
  const headers = {
    ...defaultHeaders,
    ...fetchConfig.headers,
  };

  try {
    const response = await fetch(url, {
      ...fetchConfig,
      headers,
    });

    return await parseResponse<T>(response);
  } catch (error) {
    // Re-throw ApiClientError as-is
    if (error instanceof ApiClientError) {
      throw error;
    }

    // Wrap other errors
    throw new ApiClientError(
      error instanceof Error ? error.message : 'Unknown error occurred'
    );
  }
}

/**
 * API client methods
 */
export const apiClient = {
  /**
   * GET request
   */
  get: <T>(path: string, config?: RequestConfig): Promise<T> => {
    return request<T>(path, { ...config, method: 'GET' });
  },

  /**
   * POST request
   */
  post: <T, D = unknown>(path: string, data?: D, config?: RequestConfig): Promise<T> => {
    return request<T>(path, {
      ...config,
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  },

  /**
   * PUT request
   */
  put: <T, D = unknown>(path: string, data?: D, config?: RequestConfig): Promise<T> => {
    return request<T>(path, {
      ...config,
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  },

  /**
   * PATCH request
   */
  patch: <T, D = unknown>(path: string, data?: D, config?: RequestConfig): Promise<T> => {
    return request<T>(path, {
      ...config,
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    });
  },

  /**
   * DELETE request
   */
  delete: <T>(path: string, config?: RequestConfig): Promise<T> => {
    return request<T>(path, { ...config, method: 'DELETE' });
  },

  /**
   * GET request that returns raw Response (for blob/text content)
   */
  getRaw: async (path: string, config?: RequestConfig): Promise<Response> => {
    const { params, ...fetchConfig } = config || {};
    const url = buildUrl(path, params);
    const headers = {
      ...fetchConfig.headers,
    };
    return fetch(url, {
      ...fetchConfig,
      method: 'GET',
      headers,
    });
  },

  /**
   * PUT request with raw text body
   */
  putText: async (path: string, content: string, config?: RequestConfig): Promise<void> => {
    const url = buildUrl(path, config?.params);
    const response = await fetch(url, {
      ...config,
      method: 'PUT',
      headers: {
        'Content-Type': 'text/plain; charset=utf-8',
        ...config?.headers,
      },
      body: content,
    });
    
    if (!response.ok) {
      const errorText = await response.text().catch(() => 'Unknown error');
      throw new ApiClientError(errorText || `HTTP ${response.status}`);
    }
  },
};
