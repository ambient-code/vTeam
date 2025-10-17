'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { FileQuestion } from 'lucide-react';

export default function SessionNotFound() {
  return (
    <div className="container mx-auto p-6">
      <Card className="max-w-lg mx-auto mt-12">
        <CardHeader>
          <div className="flex items-center gap-2">
            <FileQuestion className="h-5 w-5 text-muted-foreground" />
            <CardTitle>Session not found</CardTitle>
          </div>
          <CardDescription>
            The session you&apos;re looking for doesn&apos;t exist or has been deleted.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button onClick={() => window.history.back()}>Go back</Button>
        </CardContent>
      </Card>
    </div>
  );
}
