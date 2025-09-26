"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";

type Me = {
  authenticated: boolean;
  userId?: string;
  email?: string;
  username?: string;
  displayName?: string;
};

export function UserBubble() {
  const [me, setMe] = useState<Me | null>(null);

  useEffect(() => {
    const run = async () => {
      try {
        const res = await fetch("/api/me", { cache: "no-store" });
        const data = await res.json();
        setMe(data);
      } catch {
        setMe({ authenticated: false });
      }
    };
    run();
  }, []);

  const initials = (me?.displayName || me?.username || me?.email || "?")
    .split(/[\s@._-]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0]?.toUpperCase())
    .join("");

  if (!me) return <div className="w-8 h-8 rounded-full bg-muted animate-pulse" />;

  if (!me.authenticated) {
    return (
      <Button variant="ghost" size="sm">Sign in</Button>
    );
  }

  return (
    <Button variant="ghost" size="sm" className="m-2 p-1 pr-2 cursor-pointer" asChild>
      <div className="flex items-center gap-2">
        <Avatar>
          <AvatarImage alt={me.displayName || initials} />
          <AvatarFallback>{initials || "?"}</AvatarFallback>
        </Avatar>
        <span className="hidden sm:block text-sm text-muted-foreground">{me.displayName}</span>
      </div>
    </Button>
  );
}


