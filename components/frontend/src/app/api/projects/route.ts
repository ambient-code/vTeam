import { NextRequest, NextResponse } from "next/server";
import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";
import { USE_MOCKS } from "@/lib/mock-config";
import { handleListProjects, handleCreateProject } from "@/lib/mocks/handlers";

export async function GET(request: NextRequest) {
  // Return mock data if enabled
  if (USE_MOCKS) {
    return handleListProjects();
  }

  try {
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/projects`, {
      method: 'GET',
      headers,
    });

    // Forward the response from backend
    const data = await response.text();

    return new NextResponse(data, {
      status: response.status,
      headers: {
        "Content-Type": "application/json",
      },
    });
  } catch (error) {
    console.error("Failed to fetch projects:", error);
    return NextResponse.json(
      { error: "Failed to fetch projects" },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  // Return mock data if enabled
  if (USE_MOCKS) {
    return handleCreateProject(request);
  }

  try {
    const body = await request.text();

    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/projects`, {
      method: 'POST',
      headers,
      body: body,
    });

    // Forward the response from backend
    const data = await response.text();

    return new NextResponse(data, {
      status: response.status,
      headers: {
        "Content-Type": "application/json",
      },
    });
  } catch (error) {
    console.error("Failed to create project:", error);
    return NextResponse.json(
      { error: "Failed to create project" },
      { status: 500 }
    );
  }
}