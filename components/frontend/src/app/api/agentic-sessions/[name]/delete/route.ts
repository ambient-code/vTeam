import { BACKEND_URL } from '@/lib/config';

// DELETE /api/agentic-sessions/[name] - Delete an agentic session
export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ name: string }> },
) {
	try {
		const { name } = await params;

		const response = await fetch(`${BACKEND_URL}/agentic-sessions/${name}`, {
			method: 'DELETE',
		});

		if (!response.ok) {
			const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
			return Response.json(errorData, { status: response.status });
		}

		const data = await response.json();
		return Response.json(data);
	} catch (error) {
		console.error('Error deleting agentic session:', error);
		return Response.json({ error: 'Failed to delete agentic session' }, { status: 500 });
	}
}
