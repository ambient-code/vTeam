import { NextResponse } from 'next/server';

// Internal backend URL (not exposed externally)
const BACKEND_URL = process.env.BACKEND_URL || 'http://backend-service:8080/api';

export async function GET() {
  try {
    const response = await fetch(`${BACKEND_URL}/research-sessions`);
    if (!response.ok) {
      throw new Error(`Backend responded with ${response.status}`);
    }
    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error fetching research sessions:', error);
    return NextResponse.json({ error: 'Failed to fetch research sessions' }, { status: 500 });
  }
}

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const response = await fetch(`${BACKEND_URL}/research-sessions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });
    
    if (!response.ok) {
      throw new Error(`Backend responded with ${response.status}`);
    }
    
    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error creating research session:', error);
    return NextResponse.json({ error: 'Failed to create research session' }, { status: 500 });
  }
}