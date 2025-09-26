import { NextRequest, NextResponse } from 'next/server';
import { buildForwardHeadersAsync } from '@/lib/auth';
import { BACKEND_URL } from '@/lib/config';

type RouteContext = {
  params: Promise<{
    name: string;
    id: string;
  }>;
};

export async function POST(request: NextRequest, context: RouteContext) {
  const { name: projectName, id: workflowId } = await context.params;
  
  // Forward auth headers properly
  const headers = await buildForwardHeadersAsync(request);

  // Forward request body (optional seeding config)
  let body = null;
  try {
    body = await request.json();
  } catch {
    // No body provided, use defaults
  }

  const resp = await fetch(
    `${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/rfe-workflows/${encodeURIComponent(workflowId)}/seed`,
    {
      method: 'POST',
      headers,
      body: body ? JSON.stringify(body) : undefined,
    }
  );

  if (!resp.ok) {
    const text = await resp.text();
    return NextResponse.json(
      { error: text || 'Failed to start seeding' },
      { status: resp.status }
    );
  }

  const data = await resp.json();
  return NextResponse.json(data);
}

