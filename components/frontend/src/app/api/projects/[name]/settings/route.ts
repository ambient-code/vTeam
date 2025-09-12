import { NextRequest, NextResponse } from "next/server";
import { BACKEND_URL } from "@/lib/config";

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: projectName } = await params;

    // Forward the request to the backend
    const response = await fetch(`${BACKEND_URL}/projects/${projectName}/settings`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        // Forward authentication headers from the client request
        "X-User-ID": request.headers.get("X-User-ID") || "",
        "X-User-Groups": request.headers.get("X-User-Groups") || "",
      },
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
    console.error("Failed to fetch project settings:", error);
    return NextResponse.json(
      { error: "Failed to fetch project settings" },
      { status: 500 }
    );
  }
}

export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: projectName } = await params;
    const body = await request.text();

    // Forward the request to the backend
    const response = await fetch(`${BACKEND_URL}/projects/${projectName}/settings`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        // Forward authentication headers from the client request
        "X-User-ID": request.headers.get("X-User-ID") || "",
        "X-User-Groups": request.headers.get("X-User-Groups") || "",
      },
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
    console.error("Failed to update project settings:", error);
    return NextResponse.json(
      { error: "Failed to update project settings" },
      { status: 500 }
    );
  }
}