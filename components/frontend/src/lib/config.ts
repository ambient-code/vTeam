// API configuration for frontend
const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8080/api'

/**
 * Get the API base URL for frontend requests
 */
export function getApiUrl(): string {
  // Frontend always calls its own API routes (e.g., /api/agentic-sessions)
  // These routes proxy to the internal backend service
  if (typeof window !== 'undefined') {
    // Client-side: use relative URLs to hit our Next.js API routes
    return '/api'
  }
  
  // Server-side: directly call backend
  return BACKEND_URL
}
export { BACKEND_URL }
