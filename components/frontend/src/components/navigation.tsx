"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { UserBubble } from "@/components/user-bubble";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Plug, LogOut } from "lucide-react";

type NavigationProps = {
  feedbackUrl?: string;
};

export function Navigation({ feedbackUrl }: NavigationProps) {
  // const pathname = usePathname();
  // const segments = pathname?.split("/").filter(Boolean) || [];
  const router = useRouter();

  const handleLogout = () => {
    // Redirect to oauth-proxy logout endpoint  
    // This clears the OpenShift OAuth session and redirects back to login  
    window.location.href = '/oauth/sign_out';  
  };

  return (
    <nav className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto px-6">
        <div className="flex h-16 items-center justify-between gap-4">
          <div className="flex items-center gap-6">
            <Link href="/" className="text-xl font-bold">
              Ambient Code Platform
            </Link>
          </div>
          <div className="flex items-center gap-3">
            {feedbackUrl && (
              <a 
                href={feedbackUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                Share feedback
              </a>
            )}
            <DropdownMenu>
              <DropdownMenuTrigger className="outline-none">
                <UserBubble />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onSelect={() => router.push('/integrations')}>
                  <Plug className="w-4 h-4 mr-2" />
                  Integrations
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={handleLogout}>
                  <LogOut className="w-4 h-4 mr-2" />
                  Logout
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
    </nav>
  );
}