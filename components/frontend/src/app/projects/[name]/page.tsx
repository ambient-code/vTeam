'use client';

import { useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';

export default function ProjectDetailsPage() {
  const params = useParams();
  const router = useRouter();
  const projectName = params?.name as string;

  // Redirect to sessions page (default view)
  useEffect(() => {
    if (projectName) {
      router.replace(`/projects/${projectName}/sessions`);
    }
  }, [projectName, router]);

  return null;
}
