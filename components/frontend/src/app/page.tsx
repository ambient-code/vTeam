"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";

export default function HomeRedirect() {
  const router = useRouter();
  useEffect(() => {
    // Redirect to RFE workflows as the new main interface
    router.replace("/projects");
  }, [router]);

  return (
    <div className="container mx-auto py-8">
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <Loader2 className="mx-auto h-8 w-8 animate-spin mb-4" />
          <p className="text-muted-foreground">Redirecting to Workspaces...</p>
        </div>
      </div>
    </div>
  );
}