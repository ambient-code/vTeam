import { buildForwardHeadersAsync } from '@/lib/auth';
import { USE_MOCKS } from '@/lib/mock-config';
import { handleGetMe } from '@/lib/mocks/handlers';

export async function GET(request: Request) {
  // Return mock data if enabled
  if (USE_MOCKS) {
    return handleGetMe();
  }

  try {
    // Use the shared helper so dev oc whoami and env fallbacks apply uniformly
    const headers = await buildForwardHeadersAsync(request);
    const userId = headers['X-Forwarded-User'] || '';
    const email = headers['X-Forwarded-Email'] || '';
    const username = headers['X-Forwarded-Preferred-Username'] || '';
    const token = headers['X-Forwarded-Access-Token'] || '';

    if (!userId && !username && !email && !token) {
      return Response.json({ authenticated: false }, { status: 200 });
    }

    return Response.json({
      authenticated: true,
      userId,
      email,
      username,
      displayName: username || email || userId,
    });
  } catch (error) {
    console.error('Error reading user headers:', error);
    return Response.json({ authenticated: false }, { status: 200 });
  }
}


