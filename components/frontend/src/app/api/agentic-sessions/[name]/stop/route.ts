import { BACKEND_URL } from '@/lib/config';

// POST /api/agentic-sessions/[name]/stop - Stop an agentic session
export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string }> },
) {
	try {
		const { name } = await params;

		const response = await fetch(`${BACKEND_URL}/agentic-sessions/${name}/stop`, {
			method: 'POST',
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
		console.error('Error stopping agentic session:', error);
		return Response.json({ error: 'Failed to stop agentic session' }, { status: 500 });
	}
}
