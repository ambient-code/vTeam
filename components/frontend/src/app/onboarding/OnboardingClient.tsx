'use client';

import { useProjects } from '@/services/queries';
import { OnboardingWizard } from './components/OnboardingWizard';
import { Skeleton } from '@/components/ui/skeleton';

type OnboardingClientProps = {
  appSlug?: string;
};

export default function OnboardingClient({ appSlug }: OnboardingClientProps) {
  const { data: projects, isLoading } = useProjects();

  // If user already has projects, allow them to skip onboarding
  const canSkip = !isLoading && (projects?.length ?? 0) > 0;

  // Show loading state while checking projects
  if (isLoading) {
    return (
      <div className="min-h-screen bg-background">
        <div className="container max-w-3xl mx-auto px-4 py-8">
          <div className="space-y-6">
            <div className="space-y-2">
              <Skeleton className="h-10 w-64" />
              <Skeleton className="h-4 w-96" />
            </div>
            <div className="space-y-4">
              <Skeleton className="h-2 w-full" />
              <div className="flex gap-4">
                {[1, 2, 3, 4].map((i) => (
                  <Skeleton key={i} className="h-16 flex-1" />
                ))}
              </div>
            </div>
            <Skeleton className="h-96 w-full" />
          </div>
        </div>
      </div>
    );
  }

  return <OnboardingWizard appSlug={appSlug} canSkip={canSkip} />;
}

