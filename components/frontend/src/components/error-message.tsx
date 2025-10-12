"use client";

/**
 * ErrorMessage component
 * Displays error messages with optional retry action
 */

import { AlertCircle } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { ApiClientError } from '@/types/api';

type ErrorMessageProps = {
  error: Error | ApiClientError | unknown;
  title?: string;
  onRetry?: () => void;
};

export function ErrorMessage({ error, title = 'Error', onRetry }: ErrorMessageProps) {
  const message = error instanceof Error ? error.message : 'An unknown error occurred';

  const errorCode =
    error instanceof ApiClientError && error.code
      ? ` (${error.code})`
      : '';

  return (
    <Alert variant="destructive">
      <AlertCircle className="h-4 w-4" />
      <AlertTitle>{title}{errorCode}</AlertTitle>
      <AlertDescription className="mt-2 flex flex-col gap-2">
        <p>{message}</p>
        {onRetry && (
          <div>
            <Button
              variant="outline"
              size="sm"
              onClick={onRetry}
              className="mt-2"
            >
              Try Again
            </Button>
          </div>
        )}
      </AlertDescription>
    </Alert>
  );
}
