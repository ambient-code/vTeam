import OnboardingClient from './OnboardingClient';

export const dynamic = 'force-dynamic';
export const revalidate = 0;

export default function OnboardingPage() {
  const appSlug = process.env.GITHUB_APP_SLUG;
  return <OnboardingClient appSlug={appSlug} />;
}

