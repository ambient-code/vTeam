'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { CheckCircle2 } from 'lucide-react';

export function WelcomeStep() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Welcome to Ambient Code Platform</CardTitle>
        <CardDescription>
          Let&apos;s get you set up with your AI-powered workspace
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <p className="text-sm text-muted-foreground">
          The Ambient Code Platform is a Kubernetes-native AI automation platform that orchestrates
          intelligent agentic sessions through containerized microservices. In just a few steps,
          you&apos;ll be ready to create your first AI-powered project.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-semibold">What you&apos;ll set up:</h3>
          <ul className="space-y-3">
            <li className="flex items-start gap-3">
              <CheckCircle2 className="h-5 w-5 text-primary mt-0.5 flex-shrink-0" />
              <div>
                <p className="text-sm font-medium">Create your first project</p>
                <p className="text-sm text-muted-foreground">
                  Projects provide isolated workspaces for your AI sessions
                </p>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <CheckCircle2 className="h-5 w-5 text-primary mt-0.5 flex-shrink-0" />
              <div>
                <p className="text-sm font-medium">Connect GitHub</p>
                <p className="text-sm text-muted-foreground">
                  Enable AI agents to work with your repositories
                </p>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <CheckCircle2 className="h-5 w-5 text-primary mt-0.5 flex-shrink-0" />
              <div>
                <p className="text-sm font-medium">Review and complete</p>
                <p className="text-sm text-muted-foreground">
                  Confirm your settings and start building
                </p>
              </div>
            </li>
          </ul>
        </div>

        <div className="rounded-lg bg-muted p-4">
          <p className="text-sm text-muted-foreground">
            <span className="font-semibold">Estimated time:</span> Less than 5 minutes
          </p>
        </div>
      </CardContent>
    </Card>
  );
}

