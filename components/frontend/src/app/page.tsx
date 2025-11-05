"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { useProjects } from "@/services/queries";

export default function HomeRedirect() {
  const router = useRouter();
  const { data: projects, isLoading } = useProjects();

  useEffect(() => {
    if (!isLoading) {
      // If user has no projects, redirect to onboarding
      if (!projects || projects.length === 0) {
        router.replace("/onboarding");
      } else {
        // Otherwise redirect to projects page
        router.replace("/projects");
      }
    }
  }, [projects, isLoading, router]);

  return (
    <div className="container mx-auto py-8">
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <Loader2 className="mx-auto h-8 w-8 animate-spin mb-4" />
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </div>
    </div>
  );
}