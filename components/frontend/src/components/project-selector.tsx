"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useProjects } from "@/services/queries";

export function ProjectSelector() {
  const router = useRouter();
  const pathname = usePathname();
  const { data: projects = [] } = useProjects();
  const [value, setValue] = useState<string>("");

  useEffect(() => {
    // Hydrate selection from URL or storage
    const match = pathname?.match(/^\/projects\/([^\/]+)/);
    const urlProject = match?.[1];
    const stored = typeof window !== "undefined" ? localStorage.getItem("selectedProject") || "" : "";
    const initial = urlProject || stored;
    if (initial && projects.some(p => p.name === initial)) {
      setValue(initial);
    }
  }, [pathname, projects]);

  const onChange = (newValue: string) => {
    setValue(newValue);
    try { localStorage.setItem("selectedProject", newValue); } catch {}
    router.push(`/projects/${encodeURIComponent(newValue)}`);
  };

  return (
    <div className="min-w-[220px]">
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger className="w-[240px]" disabled={projects.length === 0}>
          <SelectValue placeholder="Select project" />
        </SelectTrigger>
        <SelectContent>
          {projects.length === 0 ? (
            <SelectGroup>
              <SelectLabel>No projects</SelectLabel>
            </SelectGroup>
          ) : (
            projects.map((p) => (
              <SelectItem key={p.name} value={p.name}>
                {p.displayName || p.name}
              </SelectItem>
            ))
          )}
        </SelectContent>
      </Select>
    </div>
  );
}


