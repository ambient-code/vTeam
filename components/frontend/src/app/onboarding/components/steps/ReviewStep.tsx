'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { CheckCircle2, Rocket } from 'lucide-react';
import type { WizardData } from '../../types';

type ReviewStepProps = {
  wizardData: WizardData;
};

export function ReviewStep({ wizardData }: ReviewStepProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Rocket className="h-5 w-5" />
          You&apos;re All Set!
        </CardTitle>
        <CardDescription>Review your configuration and start building</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="rounded-lg bg-green-50 border border-green-200 p-4">
          <div className="flex items-start gap-3">
            <CheckCircle2 className="h-5 w-5 text-green-600 mt-0.5 flex-shrink-0" />
            <div className="space-y-1">
              <p className="text-sm font-semibold text-green-900">
                Onboarding Complete!
              </p>
              <p className="text-sm text-green-800">
                Your workspace is configured and ready to use.
              </p>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-semibold">Configuration Summary</h3>

          <div className="space-y-3">
            <div className="flex items-start justify-between py-2 border-b">
              <div className="space-y-1">
                <p className="text-sm font-medium">Project</p>
                <p className="text-sm text-muted-foreground">
                  {wizardData.projectDisplayName}
                </p>
                <p className="text-xs text-muted-foreground font-mono">
                  {wizardData.createdProjectName || wizardData.projectName}
                </p>
              </div>
              <Badge variant="default" className="bg-green-600">
                <CheckCircle2 className="h-3 w-3 mr-1" />
                Created
              </Badge>
            </div>

            <div className="flex items-start justify-between py-2 border-b">
              <div className="space-y-1">
                <p className="text-sm font-medium">GitHub Integration</p>
                <p className="text-sm text-muted-foreground">
                  Repository access enabled
                </p>
              </div>
              <Badge
                variant={wizardData.githubConnected ? 'default' : 'secondary'}
                className={wizardData.githubConnected ? 'bg-green-600' : ''}
              >
                {wizardData.githubConnected ? (
                  <>
                    <CheckCircle2 className="h-3 w-3 mr-1" />
                    Connected
                  </>
                ) : (
                  'Not Connected'
                )}
              </Badge>
            </div>
          </div>
        </div>

        <div className="space-y-3">
          <h3 className="text-sm font-semibold">Next Steps</h3>
          <ul className="space-y-2 text-sm text-muted-foreground">
            <li className="flex items-start gap-2">
              <span className="text-primary font-semibold mt-0.5">1.</span>
              <span>Create your first agentic session to start AI-powered development</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-primary font-semibold mt-0.5">2.</span>
              <span>Configure runner secrets and integrations in project settings</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-primary font-semibold mt-0.5">3.</span>
              <span>Explore RFE workflows for collaborative feature planning</span>
            </li>
          </ul>
        </div>

        <div className="rounded-lg bg-muted p-4">
          <p className="text-sm text-muted-foreground">
            <span className="font-semibold">Pro Tip:</span> Visit the project settings to configure
            additional integrations like Jira, customize resource limits, and manage team permissions.
          </p>
        </div>
      </CardContent>
    </Card>
  );
}

