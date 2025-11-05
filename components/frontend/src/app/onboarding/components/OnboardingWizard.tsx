'use client';

import { useState, useCallback, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { ArrowLeft, ArrowRight, X } from 'lucide-react';
import { WelcomeStep } from './steps/WelcomeStep';
import { CreateProjectStep } from './steps/CreateProjectStep';
import { GitHubStep } from './steps/GitHubStep';
import { ReviewStep } from './steps/ReviewStep';
import { useConnectGitHub } from '@/services/queries';
import type { WizardData, WizardStep } from '../types';

const WIZARD_STEPS: WizardStep[] = [
  {
    id: 'welcome',
    title: 'Welcome',
    description: 'Get started with Ambient Code',
  },
  {
    id: 'github',
    title: 'Connect GitHub',
    description: 'Enable repository access',
  },
  {
    id: 'project',
    title: 'Create Project',
    description: 'Set up your workspace',
  },
  {
    id: 'review',
    title: 'Review',
    description: 'Confirm and complete',
  },
];

type OnboardingWizardProps = {
  appSlug?: string;
  canSkip?: boolean;
};

export function OnboardingWizard({ appSlug, canSkip = false }: OnboardingWizardProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const connectGitHubMutation = useConnectGitHub();
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [wizardData, setWizardData] = useState<WizardData>({
    projectName: '',
    projectDisplayName: '',
    projectDescription: '',
    githubConnected: false,
  });
  const [processingInstallation, setProcessingInstallation] = useState(false);

  const currentStep = WIZARD_STEPS[currentStepIndex];
  const isFirstStep = currentStepIndex === 0;
  const isLastStep = currentStepIndex === WIZARD_STEPS.length - 1;
  const progress = ((currentStepIndex + 1) / WIZARD_STEPS.length) * 100;

  const handleNext = useCallback(() => {
    if (!isLastStep) {
      setCurrentStepIndex((prev) => prev + 1);
    }
  }, [isLastStep]);

  const handleBack = useCallback(() => {
    if (!isFirstStep) {
      setCurrentStepIndex((prev) => prev - 1);
    }
  }, [isFirstStep]);

  const handleProjectCreated = useCallback(
    (projectName: string) => {
      setWizardData((prev) => ({
        ...prev,
        createdProjectName: projectName,
      }));
      handleNext();
    },
    [handleNext]
  );

  const handleGitHubVerified = useCallback(() => {
    setWizardData((prev) => ({
      ...prev,
      githubConnected: true,
    }));
  }, []);

  const handleComplete = useCallback(() => {
    const projectName = wizardData.createdProjectName || wizardData.projectName;
    if (projectName) {
      router.push(`/projects/${projectName}`);
    } else {
      router.push('/projects');
    }
  }, [wizardData, router]);

  const handleSkip = useCallback(() => {
    router.push('/projects');
  }, [router]);

  const canProceedToNext = useCallback(() => {
    switch (currentStep.id) {
      case 'welcome':
        return true;
      case 'github':
        return wizardData.githubConnected;
      case 'project':
        return !!wizardData.createdProjectName;
      case 'review':
        return true;
      default:
        return false;
    }
  }, [currentStep.id, wizardData]);

  // Handle GitHub installation callback
  useEffect(() => {
    const installationId = searchParams.get('installation_id');
    const isOnGitHubStep = currentStep.id === 'github';
    
    if (installationId && isOnGitHubStep && !processingInstallation && !wizardData.githubConnected) {
      setProcessingInstallation(true);
      
      connectGitHubMutation.mutate(
        { installationId: Number(installationId) },
        {
          onSuccess: () => {
            handleGitHubVerified();
            // Remove the installation_id from URL without navigation
            const newUrl = new URL(window.location.href);
            newUrl.searchParams.delete('installation_id');
            window.history.replaceState({}, '', newUrl);
          },
          onError: (error) => {
            console.error('Failed to connect GitHub:', error);
            setProcessingInstallation(false);
          },
        }
      );
    }
  }, [searchParams, currentStep.id, processingInstallation, wizardData.githubConnected, connectGitHubMutation, handleGitHubVerified]);

  return (
    <div className="min-h-screen bg-background">
      <div className="container max-w-3xl mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h1 className="text-3xl font-bold">Get Started</h1>
              <p className="text-muted-foreground mt-1">
                Set up your Ambient Code workspace
              </p>
            </div>
            {canSkip && (
              <Button variant="ghost" size="sm" onClick={handleSkip}>
                <X className="h-4 w-4 mr-2" />
                Skip
              </Button>
            )}
          </div>

          {/* Progress */}
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="font-medium">
                Step {currentStepIndex + 1} of {WIZARD_STEPS.length}
              </span>
              <span className="text-muted-foreground">{currentStep.title}</span>
            </div>
            <Progress value={progress} className="h-2" />
          </div>

          {/* Step Indicators */}
          <div className="mt-6 flex items-center justify-between">
            {WIZARD_STEPS.map((step, index) => (
              <div
                key={step.id}
                className={`flex-1 ${index !== WIZARD_STEPS.length - 1 ? 'border-r' : ''}`}
              >
                <div className="flex flex-col items-center px-2">
                  <div
                    className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-semibold mb-2 ${
                      index === currentStepIndex
                        ? 'bg-primary text-primary-foreground'
                        : index < currentStepIndex
                        ? 'bg-green-600 text-white'
                        : 'bg-muted text-muted-foreground'
                    }`}
                  >
                    {index + 1}
                  </div>
                  <div className="text-center">
                    <p
                      className={`text-xs font-medium ${
                        index === currentStepIndex
                          ? 'text-foreground'
                          : 'text-muted-foreground'
                      }`}
                    >
                      {step.title}
                    </p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Step Content */}
        <div className="mb-8">
          {currentStep.id === 'welcome' && <WelcomeStep />}
          {currentStep.id === 'project' && (
            <CreateProjectStep
              onProjectCreated={handleProjectCreated}
              initialData={{
                name: wizardData.projectName,
                displayName: wizardData.projectDisplayName,
                description: wizardData.projectDescription,
              }}
            />
          )}
          {currentStep.id === 'github' && (
            <GitHubStep 
              appSlug={appSlug} 
              onConnectionVerified={handleGitHubVerified} 
              isProcessing={processingInstallation}
            />
          )}
          {currentStep.id === 'review' && <ReviewStep wizardData={wizardData} />}
        </div>

        {/* Navigation */}
        <div className="flex items-center justify-between">
          <Button
            variant="outline"
            onClick={handleBack}
            disabled={isFirstStep}
            className="min-w-[100px]"
          >
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>

          {isLastStep ? (
            <Button onClick={handleComplete} size="lg" className="min-w-[150px]">
              Go to Project
              <ArrowRight className="h-4 w-4 ml-2" />
            </Button>
          ) : (
            <Button
              onClick={handleNext}
              disabled={!canProceedToNext()}
              size="lg"
              className="min-w-[100px]"
            >
              Next
              <ArrowRight className="h-4 w-4 ml-2" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

