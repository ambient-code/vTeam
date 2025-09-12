"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { ProjectSelector } from "@/components/project-selector";
import { Plus } from "lucide-react";
import { UserBubble } from "@/components/user-bubble";



export function Navigation() {
  const pathname = usePathname();
  const segments = pathname.split("/").filter(Boolean);
  const isProjectDetail = segments[0] === "projects" && !!segments[1] && segments[1] !== "new";

  return (
    <nav className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto px-6">
        <div className="flex h-16 items-center justify-between gap-4">
          <div className="flex items-center gap-6">
            <Link href="/" className="text-xl font-bold">
              Ambient Agentic Runner
            </Link>
          </div>
          <div className="flex items-center gap-3">
            {isProjectDetail && (
              <>
                <ProjectSelector />
                <Link href="/projects/new">
                  <Button variant="ghost" size="icon" aria-label="Create project">
                    <Plus className="w-4 h-4" />
                  </Button>
                </Link>
              </>
            )}
            <UserBubble />
          </div>
        </div>
      </div>
    </nav>
  );
}