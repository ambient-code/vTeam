import { useState, useEffect, useCallback } from 'react';
import { getApiUrl } from '@/lib/config';
import type { AgenticSession } from '@/types/agentic-session';
import type { SessionMessage } from '@/types';

type UseSessionDataReturn = {
  session: AgenticSession | null;
  liveMessages: SessionMessage[];
  loading: boolean;
  error: string | null;
  fetchSession: () => Promise<void>;
  fetchMessages: () => Promise<void>;
  refetch: () => Promise<void>;
};

export function useSessionData(projectName: string, sessionName: string): UseSessionDataReturn {
  const [session, setSession] = useState<AgenticSession | null>(null);
  const [liveMessages, setLiveMessages] = useState<SessionMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSession = useCallback(async () => {
    if (!projectName || !sessionName) return;

    try {
      const apiUrl = getApiUrl();
      const res = await fetch(
        `${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}`
      );
      if (!res.ok) {
        throw new Error(`Failed to fetch session: ${res.statusText}`);
      }
      const data = await res.json();
      setSession(data.session || null);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load session');
    } finally {
      setLoading(false);
    }
  }, [projectName, sessionName]);

  const fetchMessages = useCallback(async () => {
    if (!projectName || !sessionName) return;

    try {
      const apiUrl = getApiUrl();
      const res = await fetch(
        `${apiUrl}/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/messages`
      );
      if (!res.ok) {
        console.warn('Failed to fetch messages:', res.statusText);
        return;
      }
      const data = await res.json();
      if (Array.isArray(data.messages)) {
        setLiveMessages(data.messages);
      }
    } catch (err) {
      console.error('Error fetching messages:', err);
    }
  }, [projectName, sessionName]);

  const refetch = useCallback(async () => {
    await Promise.all([fetchSession(), fetchMessages()]);
  }, [fetchSession, fetchMessages]);

  useEffect(() => {
    if (projectName && sessionName) {
      refetch();
    }
  }, [projectName, sessionName, refetch]);

  return {
    session,
    liveMessages,
    loading,
    error,
    fetchSession,
    fetchMessages,
    refetch,
  };
}
