"use client";

import React, { useMemo } from "react";
import type { AgentPersona } from "@/types/agentic-session";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";

type Props = {
  agents: AgentPersona[];
  selectedAgents: string[];
  onChange: (next: string[]) => void;
  maxAgents?: number;
  disabled?: boolean;
};

export function MultiAgentSelection({ agents, selectedAgents, onChange, maxAgents = 8, disabled = false }: Props) {
  const selectedCount = selectedAgents.length;
  const availableAgents = useMemo(() => agents || [], [agents]);

  const toggle = (persona: string) => {
    if (disabled) return;
    const isSelected = selectedAgents.includes(persona);
    if (isSelected) {
      onChange(selectedAgents.filter(p => p !== persona));
    } else if (selectedAgents.length < maxAgents) {
      onChange([...selectedAgents, persona]);
    }
  };

  const clearAll = () => {
    if (!disabled) onChange([]);
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="outline" size="sm" disabled={disabled}>
              Select agents ({selectedCount}/{maxAgents})
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-80 max-h-80 overflow-auto">
            {availableAgents.map(agent => {
              const checked = selectedAgents.includes(agent.persona);
              return (
                <DropdownMenuCheckboxItem
                  key={agent.persona}
                  checked={checked}
                  onCheckedChange={() => toggle(agent.persona)}
                >
                  <div className="flex flex-col">
                    <span className="text-sm">{agent.name}</span>
                    <span className="text-xs text-muted-foreground">{agent.role}</span>
                  </div>
                </DropdownMenuCheckboxItem>
              );
            })}
          </DropdownMenuContent>
        </DropdownMenu>
        <Button type="button" variant="ghost" size="sm" disabled={disabled || selectedCount === 0} onClick={clearAll}>
          Clear
        </Button>
      </div>

      {selectedCount > 0 && (
        <div className="flex flex-wrap gap-2">
          {selectedAgents.map(persona => {
            const agent = availableAgents.find(a => a.persona === persona);
            if (!agent) return null;
            return (
              <Badge key={persona} variant="secondary" className="flex items-center gap-1">
                {agent.name}
                <button
                  type="button"
                  className="ml-1 hover:opacity-70"
                  onClick={() => toggle(persona)}
                  disabled={disabled}
                >
                  ×
                </button>
              </Badge>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default MultiAgentSelection;


