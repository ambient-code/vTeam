import { NextResponse } from 'next/server';

// Internal backend URL (not exposed externally)  
const BACKEND_URL = process.env.BACKEND_URL || 'http://backend-service:8080/api';

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params;
    const response = await fetch(`${BACKEND_URL}/research-sessions/${name}`);
    if (!response.ok) {
      throw new Error(`Backend responded with ${response.status}`);
    }
    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error fetching research session:', error);
    return NextResponse.json({ error: 'Failed to fetch research session' }, { status: 500 });
  }
}
