'use client';

import { useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';

export default function ProjectSettingsPage() {
  const params = useParams();
  const router = useRouter();
  const projectName = params?.name as string;

  // Redirect to main workspace page
  useEffect(() => {
    if (projectName) {
      router.replace(`/projects/${projectName}?section=settings`);
    }
  }, [projectName, router]);

  return null;
}