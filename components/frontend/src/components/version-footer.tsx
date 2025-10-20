'use client';

import { useEffect, useState } from 'react';

export function VersionFooter() {
  const [version, setVersion] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/version')
      .then((res) => res.json())
      .then((data) => setVersion(data.version))
      .catch((err) => console.error('Failed to fetch version:', err));
  }, []);

  if (!version) {
    return null;
  }

  return (
    <footer className="py-2 text-center">
      <p className="text-xs text-muted-foreground">Version {version}</p>
    </footer>
  );
}
