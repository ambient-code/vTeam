import { BACKEND_URL } from '@/lib/config';

/**
 * GET /api/cluster-info
 * Returns cluster information (OpenShift vs vanilla Kubernetes)
 * This endpoint does not require authentication as it's public cluster information
 */
export async function GET() {
  try {
    const response = await fetch(`${BACKEND_URL}/cluster-info`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      return Response.json(errorData, { status: response.status });
    }

    const data = await response.json();
    return Response.json(data);
  } catch (error) {
    console.error('Error fetching cluster info:', error);
    return Response.json({ error: 'Failed to fetch cluster info' }, { status: 500 });
  }
}


