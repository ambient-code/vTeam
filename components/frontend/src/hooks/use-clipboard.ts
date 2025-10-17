/**
 * useClipboard hook
 * Provides copy to clipboard functionality with success state
 */

import { useState, useCallback } from 'react';

type UseClipboardReturn = {
  copy: (text: string) => Promise<void>;
  copied: boolean;
  error: Error | null;
};

export function useClipboard(resetDelay: number = 2000): UseClipboardReturn {
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const copy = useCallback(
    async (text: string) => {
      try {
        await navigator.clipboard.writeText(text);
        setCopied(true);
        setError(null);

        // Reset copied state after delay
        setTimeout(() => {
          setCopied(false);
        }, resetDelay);
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Failed to copy');
        setError(error);
        setCopied(false);
      }
    },
    [resetDelay]
  );

  return { copy, copied, error };
}
