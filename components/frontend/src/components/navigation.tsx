"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { UserBubble } from "@/components/user-bubble";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Plug } from "lucide-react";



export function Navigation() {
  // const pathname = usePathname();
  // const segments = pathname?.split("/").filter(Boolean) || [];
  const router = useRouter();

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
            <DropdownMenu>
              <DropdownMenuTrigger className="outline-none">
                <UserBubble />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onSelect={() => router.push('/integrations')}>
                  <Plug className="w-4 h-4 mr-2" />
                  Integrations
                  </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
    </nav>
  );
}