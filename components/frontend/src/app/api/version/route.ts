import { env } from '@/lib/env';
import { USE_MOCKS } from '@/lib/mock-config';
import { handleGetVersion } from '@/lib/mocks/handlers';

export async function GET() {
  // Return mock data if enabled
  if (USE_MOCKS) {
    return handleGetVersion();
  }

  return Response.json({
    version: env.VTEAM_VERSION,
  });
}
