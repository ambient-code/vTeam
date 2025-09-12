"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";

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
    <div className="flex items-center gap-2">
      <div className="w-8 h-8 rounded-full bg-primary/10 text-primary flex items-center justify-center text-sm font-medium">
        {initials || "?"}
      </div>
      <span className="hidden sm:block text-sm text-muted-foreground">{me.displayName}</span>
    </div>
  );
}


