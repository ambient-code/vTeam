"use client";

import { ReactNode } from "react";
import { cn } from "@/lib/utils";

type ProjectSubpageHeaderProps = {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  left?: ReactNode; // Optional left slot (breadcrumbs/sub-nav)
  className?: string;
};

export function ProjectSubpageHeader({ title, description, actions, left, className }: ProjectSubpageHeaderProps) {
  return (
    <div className={cn("flex items-center justify-between mb-6", className)}>
      <div className="flex items-center gap-3">
        {left && <div className="shrink-0">{left}</div>}
        <div>
          <h2 className="text-2xl font-semibold flex items-center gap-2">
            {title}
          </h2>
          {description && (
            <p className="text-muted-foreground">{description}</p>
          )}
        </div>
      </div>
      <div className="flex items-center gap-4">
        {actions}
      </div>
    </div>
  );
}


