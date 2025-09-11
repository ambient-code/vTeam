import { NextResponse } from 'next/server';

const BACKEND_URL = process.env.BACKEND_URL || 'http://backend-service:8080/api';

type Params = Promise<{
	name: string;
}>;

export async function DELETE(
	request: Request,
	{ params }: { params: Params }
) {
	try {
		const { name } = await params;

		const response = await fetch(`${BACKEND_URL}/research-sessions/${name}`, {
			method: 'DELETE',
			headers: {
				'Content-Type': 'application/json',
			},
		});

		if (!response.ok) {
			const error = await response.text();
			throw new Error(`Backend responded with ${response.status}: ${error}`);
		}

		const data = await response.json();
		return NextResponse.json(data);
	} catch (error) {
		console.error('Error deleting research session:', error);
		return NextResponse.json(
			{ error: 'Failed to delete research session' },
			{ status: 500 }
		);
	}
}

