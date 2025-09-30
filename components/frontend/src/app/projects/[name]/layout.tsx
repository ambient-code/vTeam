"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Home, KeyRound, Settings, Users, Sparkles, ArrowLeft, GitBranch } from "lucide-react";

export default function ProjectSectionLayout({ children, params }: { children: React.ReactNode; params: Promise<{ name: string }> }) {
  const pathname = usePathname();

  const base = pathname?.split("/").slice(0, 3).join("/") || "/projects";
  // base is /projects/[name]

  const items = [
    { href: base, label: "Overview", icon: Home },
    { href: `${base}/rfe`, label: "RFE Workspaces", icon: GitBranch },
    { href: `${base}/sessions`, label: "Sessions", icon: Sparkles },
    { href: `${base}/keys`, label: "Keys", icon: KeyRound },
    { href: `${base}/permissions`, label: "Permissions", icon: Users },
    { href: `${base}/settings`, label: "Settings", icon: Settings },
  ];

  return (
    <div className="container mx-auto p-0 h-full">
      <div className="flex h-full">
        <aside className="w-56 shrink-0 border-r p-4 sticky top-16 self-start max-h-[calc(100vh-4rem)]">
          <div className="mb-3">
            <Link href="/projects">
              <Button variant="ghost" size="sm" className="w-full justify-start">
                <ArrowLeft className="w-4 h-4 mr-2" />
                Back to Projects
              </Button>
            </Link>
          </div>
          <div className="space-y-1">
            {items.map((item) => {
              const isActive = pathname === item.href || (item.href !== base && pathname?.startsWith(item.href));
              const Icon = item.icon;
              return (
                <Link key={item.href} href={item.href}>
                  <Button variant={isActive ? "secondary" : "ghost"} className={cn("w-full justify-start", isActive && "font-semibold")}> 
                    <Icon className="w-4 h-4 mr-2" />
                    {item.label}
                  </Button>
                </Link>
              );
            })}
          </div>
        </aside>
        <section className="flex-1 p-6 overflow-y-auto max-h-[calc(100vh-4rem)]">{children}</section>
      </div>
    </div>
  );
}


