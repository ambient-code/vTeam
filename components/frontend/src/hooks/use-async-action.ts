import { useState, useCallback } from 'react';

type AsyncActionState = {
  isLoading: boolean;
  error: Error | null;
};

type UseAsyncActionReturn<TArgs extends unknown[], TResult> = {
  execute: (...args: TArgs) => Promise<TResult | undefined>;
  isLoading: boolean;
  error: Error | null;
  reset: () => void;
};

export function useAsyncAction<TArgs extends unknown[], TResult>(
  action: (...args: TArgs) => Promise<TResult>
): UseAsyncActionReturn<TArgs, TResult> {
  const [state, setState] = useState<AsyncActionState>({
    isLoading: false,
    error: null,
  });

  const execute = useCallback(
    async (...args: TArgs): Promise<TResult | undefined> => {
      setState({ isLoading: true, error: null });
      try {
        const result = await action(...args);
        setState({ isLoading: false, error: null });
        return result;
      } catch (err) {
        const error = err instanceof Error ? err : new Error(String(err));
        setState({ isLoading: false, error });
        return undefined;
      }
    },
    [action]
  );

  const reset = useCallback(() => {
    setState({ isLoading: false, error: null });
  }, []);

  return {
    execute,
    isLoading: state.isLoading,
    error: state.error,
    reset,
  };
}
