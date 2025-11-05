'use client';

import { useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';

export default function ProjectSessionsListPage() {
  const params = useParams();
  const router = useRouter();
  const projectName = params?.name as string;

  // Redirect to main workspace page (sessions is the default view)
  useEffect(() => {
    if (projectName) {
      router.replace(`/projects/${projectName}`);
    }
  }, [projectName, router]);

  return null;
}
