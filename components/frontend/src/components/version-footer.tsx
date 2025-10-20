'use client';

import { useVersion } from '@/services/queries/use-version';

export function VersionFooter() {
  const { data: version } = useVersion();

  if (!version) {
    return null;
  }

  return (
    <footer className="py-2 text-center">
      <p className="text-xs text-muted-foreground">Version {version}</p>
    </footer>
  );
}
