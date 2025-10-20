import { env } from '@/lib/env';

export async function GET() {
  return Response.json({
    version: env.VTEAM_VERSION,
  });
}
