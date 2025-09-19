import { BACKEND_URL } from '@/lib/config';

// GET /api/agentic-sessions/[name] - Get a specific agentic session
export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> },
) {
  try {
    const { name } = await params;
    const response = await fetch(`${BACKEND_URL}/agentic-sessions/${name}`);
    if (!response.ok) {
      if (response.status === 404) {
        return Response.json({ error: 'Agentic session not found' }, { status: 404 });
      }
      throw new Error(`Backend responded with status: ${response.status}`);
    }
    const data = await response.json();
    return Response.json(data);
  } catch (error) {
    console.error('Error fetching agentic session:', error);
    return Response.json({ error: 'Failed to fetch agentic session' }, { status: 500 });
  }
}
