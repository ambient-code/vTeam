'use client';

import { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { CheckCircle2, Github, Loader2, AlertCircle } from 'lucide-react';
import { useGitHubStatus } from '@/services/queries';

type GitHubStepProps = {
  appSlug?: string;
  onConnectionVerified: () => void;
  isProcessing?: boolean;
};

export function GitHubStep({ appSlug, onConnectionVerified, isProcessing = false }: GitHubStepProps) {
  const { data: status, isLoading, refetch } = useGitHubStatus();
  const [isConnecting, setIsConnecting] = useState(false);

  useEffect(() => {
    if (status?.installed) {
      onConnectionVerified();
    }
  }, [status?.installed, onConnectionVerified]);

  // Poll for connection status when user returns from GitHub OAuth
  useEffect(() => {
    if (isConnecting) {
      const interval = setInterval(() => {
        refetch();
      }, 2000);

      return () => clearInterval(interval);
    }
  }, [isConnecting, refetch]);

  const handleConnect = () => {
    if (!appSlug) return;
    setIsConnecting(true);
    const setupUrl = new URL('/onboarding', window.location.origin);
    const redirectUri = encodeURIComponent(setupUrl.toString());
    const url = `https://github.com/apps/${appSlug}/installations/new?redirect_uri=${redirectUri}`;
    window.location.href = url;
  };

  if (isLoading || isProcessing) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Connect GitHub</CardTitle>
          <CardDescription>
            {isProcessing ? 'Verifying GitHub installation...' : 'Checking GitHub connection status...'}
          </CardDescription>
        </CardHeader>
        <CardContent className="flex items-center justify-center py-8">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Connect GitHub</CardTitle>
        <CardDescription>
          Enable your AI agents to work with GitHub repositories
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {status?.installed ? (
          <>
            <Alert className="border-green-200 bg-green-50">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              <AlertDescription className="text-green-800">
                GitHub is connected
                {status.githubUserId && ` as ${status.githubUserId}`}
              </AlertDescription>
            </Alert>

            <div className="space-y-3">
              <h3 className="text-sm font-semibold">What you can do with GitHub:</h3>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li className="flex items-start gap-2">
                  <CheckCircle2 className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
                  <span>Browse and work with repository code</span>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle2 className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
                  <span>Create forks and pull requests automatically</span>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle2 className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
                  <span>Let AI agents make code changes and commit them</span>
                </li>
              </ul>
            </div>
          </>
        ) : (
          <>
            <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Connect your GitHub account to enable AI agents to interact with your repositories.
              You&apos;ll be able to:
              </p>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li className="flex items-start gap-2">
                  <Github className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
                  <span>Browse repository files and branches</span>
                </li>
                <li className="flex items-start gap-2">
                  <Github className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
                  <span>Create and manage forks automatically</span>
                </li>
                <li className="flex items-start gap-2">
                  <Github className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
                  <span>Submit pull requests with AI-generated code</span>
                </li>
              </ul>
            </div>

            {!appSlug && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  GitHub App is not configured. Please contact your administrator.
                </AlertDescription>
              </Alert>
            )}

            <Button
              onClick={handleConnect}
              disabled={!appSlug || isConnecting}
              className="w-full"
              size="lg"
            >
              {isConnecting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Connecting to GitHub...
                </>
              ) : (
                <>
                  <Github className="mr-2 h-4 w-4" />
                  Connect GitHub Account
                </>
              )}
            </Button>

            <p className="text-xs text-muted-foreground text-center">
              You&apos;ll be redirected to GitHub to authorize the Ambient Code app
            </p>
          </>
        )}
      </CardContent>
    </Card>
  );
}

