import { BACKEND_URL } from "@/lib/config";

export async function GET() {
  try {
    // No auth required for public OOTB workflows endpoint
    const response = await fetch(`${BACKEND_URL}/workflows/ootb`, {
      method: 'GET',
      headers: {
        "Content-Type": "application/json",
      },
    });

    // Forward the response from backend
    const data = await response.text();

    return new Response(data, {
      status: response.status,
      headers: {
        "Content-Type": "application/json",
      },
    });
  } catch (error) {
    console.error("Failed to fetch OOTB workflows:", error);
    return new Response(
      JSON.stringify({ error: "Failed to fetch OOTB workflows" }),
      { 
        status: 500,
        headers: { "Content-Type": "application/json" }
      }
    );
  }
}

