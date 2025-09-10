import { NextResponse } from 'next/server';

const BACKEND_URL = process.env.BACKEND_URL || 'http://backend-service:8080/api';

type Params = Promise<{
	name: string;
}>;

export async function POST(
	request: Request,
	{ params }: { params: Params }
) {
	try {
		const { name } = await params;

		const response = await fetch(`${BACKEND_URL}/research-sessions/${name}/stop`, {
			method: 'POST',
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
		console.error('Error stopping research session:', error);
		return NextResponse.json(
			{ error: 'Failed to stop research session' },
			{ status: 500 }
		);
	}
}

