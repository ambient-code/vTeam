import { NextRequest, NextResponse } from "next/server";
import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: projectName } = await params;
    const headers = await buildForwardHeadersAsync(request);

    // Get query parameters
    const searchParams = request.nextUrl.searchParams;
    const repo = searchParams.get('repo');
    const ref = searchParams.get('ref');
    const path = searchParams.get('path');

    // Build query string
    const queryParams = new URLSearchParams();
    if (repo) queryParams.set('repo', repo);
    if (ref) queryParams.set('ref', ref);
    if (path) queryParams.set('path', path);

    // Forward the request to the backend
    const response = await fetch(
      `${BACKEND_URL}/projects/${projectName}/repo/blob?${queryParams.toString()}`,
      {
        method: "GET",
        headers,
      }
    );

    // Forward the response from backend
    const data = await response.text();

    return new NextResponse(data, {
      status: response.status,
      headers: {
        "Content-Type": "application/json",
      },
    });
  } catch (error) {
    console.error("Failed to fetch repo blob:", error);
    return NextResponse.json(
      { error: "Failed to fetch repo blob" },
      { status: 500 }
    );
  }
}
