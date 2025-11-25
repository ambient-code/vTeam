// Utilities for extracting user auth context from Next.js API requests
// We avoid any dev fallbacks and strictly forward what is provided.

export type ForwardHeaders = Record<string, string>;

// Execute a shell command safely in Node.js runtime (server-side only)
async function tryExec(cmd: string): Promise<string | undefined> {
  if (typeof window !== 'undefined') return undefined;
  try {
    const { exec } = await import('node:child_process');
    const { promisify } = await import('node:util');
    const execAsync = promisify(exec);
    const { stdout } = await execAsync(cmd, { timeout: 2000 });
    return stdout?.trim() || undefined;
  } catch {
    return undefined;
  }
}

// Extract bearer token from either Authorization or X-Forwarded-Access-Token
export function extractAccessToken(request: Request): string | undefined {
  const forwarded = request.headers.get('X-Forwarded-Access-Token')?.trim();
  if (forwarded) return forwarded;
  const auth = request.headers.get('Authorization');
  if (!auth) return undefined;
  const match = auth.match(/^Bearer\s+(.+)$/i);
  if (match?.[1]) return match[1].trim();
  // Fallback to environment-provided token for local dev with oc login
  const envToken = process.env.OC_TOKEN?.trim();
  return envToken || undefined;
}

// Build headers to forward to backend, using only real incoming values.
export function buildForwardHeaders(request: Request, extra?: Record<string, string>): ForwardHeaders {
  const headers: ForwardHeaders = {
    'Content-Type': 'application/json',
  };

  const xfUser = request.headers.get('X-Forwarded-User');
  const xfEmail = request.headers.get('X-Forwarded-Email');
  const xfUsername = request.headers.get('X-Forwarded-Preferred-Username');
  const xfGroups = request.headers.get('X-Forwarded-Groups');
  const project = request.headers.get('X-OpenShift-Project');
  const token = extractAccessToken(request);

  if (xfUser) headers['X-Forwarded-User'] = xfUser;
  if (xfEmail) headers['X-Forwarded-Email'] = xfEmail;
  if (xfUsername) headers['X-Forwarded-Preferred-Username'] = xfUsername;
  if (xfGroups) headers['X-Forwarded-Groups'] = xfGroups;
  if (project) headers['X-OpenShift-Project'] = project;
  if (token) headers['X-Forwarded-Access-Token'] = token;

  // If still missing identity info, use environment (helpful for local oc login)
  if (!headers['X-Forwarded-User'] && process.env.OC_USER) {
    headers['X-Forwarded-User'] = process.env.OC_USER;
  }
  if (!headers['X-Forwarded-Preferred-Username'] && process.env.OC_USER) {
    headers['X-Forwarded-Preferred-Username'] = process.env.OC_USER;
  }
  if (!headers['X-Forwarded-Email'] && process.env.OC_EMAIL) {
    headers['X-Forwarded-Email'] = process.env.OC_EMAIL;
  }
  
  // Add token fallback for local development
  if (!headers['X-Forwarded-Access-Token'] && process.env.OC_TOKEN) {
    headers['X-Forwarded-Access-Token'] = process.env.OC_TOKEN;
  }

  // Optional dev-only automatic discovery via oc CLI
  // Enable by setting ENABLE_OC_WHOAMI=1 in your dev env
  const enableOc = process.env.ENABLE_OC_WHOAMI === '1' || process.env.ENABLE_OC_WHOAMI === 'true';
  const runningInNode = typeof window === 'undefined';
  const needsIdentity = !headers['X-Forwarded-User'] && !headers['X-Forwarded-Preferred-Username'];
  const needsToken = !headers['X-Forwarded-Access-Token'];

  // We cannot await top-level in this sync function, so expose best-effort sync
  // pattern by stashing promises on the object and resolving outside if needed.
  // For simplicity, perform a lazy, best-effort fetch and only if in server runtime.
  if (enableOc && runningInNode && (needsIdentity || needsToken)) {
    // Fire-and-forget: we won't block the request if oc isn't present
    (async () => {
      try {
        if (needsIdentity) {
          const user = await tryExec('oc whoami');
          if (user && !headers['X-Forwarded-User']) headers['X-Forwarded-User'] = user;
          if (user && !headers['X-Forwarded-Preferred-Username']) headers['X-Forwarded-Preferred-Username'] = user;
        }
        if (needsToken) {
          const t = await tryExec('oc whoami -t');
          if (t) headers['X-Forwarded-Access-Token'] = t;
        }
      } catch {
        // ignore
      }
    })();
  }

  if (extra) {
    for (const [k, v] of Object.entries(extra)) {
      if (v !== undefined && v !== null) headers[k] = String(v);
    }
  }

  return headers;
}

// Async version that can optionally consult oc CLI in dev and wait for results
export async function buildForwardHeadersAsync(request: Request, extra?: Record<string, string>): Promise<ForwardHeaders> {
  const headers = buildForwardHeaders(request, extra);

  // Local development mode: inject mock user when DISABLE_AUTH is true
  const disableAuth = process.env.DISABLE_AUTH === 'true';
  const mockUser = process.env.MOCK_USER || 'developer';
  
  if (disableAuth) {
    if (!headers['X-Forwarded-User']) headers['X-Forwarded-User'] = mockUser;
    if (!headers['X-Forwarded-Preferred-Username']) headers['X-Forwarded-Preferred-Username'] = mockUser;
    if (!headers['X-Forwarded-Email']) headers['X-Forwarded-Email'] = `${mockUser}@local.dev`;
    if (!headers['X-Forwarded-Access-Token']) headers['X-Forwarded-Access-Token'] = 'mock-token-for-local-dev';
    return headers;
  }

  const enableOc = process.env.ENABLE_OC_WHOAMI === '1' || process.env.ENABLE_OC_WHOAMI === 'true';
  const runningInNode = typeof window === 'undefined';
  const needsIdentity = !headers['X-Forwarded-User'] && !headers['X-Forwarded-Preferred-Username'];
  const needsToken = !headers['X-Forwarded-Access-Token'];

  if (enableOc && runningInNode && (needsIdentity || needsToken)) {
    if (needsIdentity) {
      const user = await tryExec('oc whoami');
      if (user && !headers['X-Forwarded-User']) headers['X-Forwarded-User'] = user;
      if (user && !headers['X-Forwarded-Preferred-Username']) headers['X-Forwarded-Preferred-Username'] = user;
    }
    if (needsToken) {
      const t = await tryExec('oc whoami -t');
      if (t) headers['X-Forwarded-Access-Token'] = t;
    }
  }

  return headers;
}


