import { NextRequest, NextResponse } from 'next/server';
import { buildForwardHeadersAsync } from '@/lib/auth';
import { BACKEND_URL } from '@/lib/config';

type RouteContext = {
  params: Promise<{
    name: string;
    id: string;
  }>;
};

export async function GET(request: NextRequest, context: RouteContext) {
  const { name: projectName, id: workflowId } = await context.params;
  
  const headers = await buildForwardHeadersAsync(request);

  const resp = await fetch(
    `${BACKEND_URL}/projects/${encodeURIComponent(projectName)}/rfe-workflows/${encodeURIComponent(workflowId)}/check-seeding`,
    {
      method: 'GET',
      headers,
    }
  );

  if (!resp.ok) {
    const text = await resp.text();
    return NextResponse.json(
      { error: text || 'Failed to check seeding status' },
      { status: resp.status }
    );
  }

  const data = await resp.json();
  return NextResponse.json(data);
}

