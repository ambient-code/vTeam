"use client";

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Bot, Loader2 } from "lucide-react";
import { AVAILABLE_AGENTS } from "@/lib/agents";
import { useRfeWorkflowAgents } from "@/services/queries";

type RfeAgentsCardProps = {
  projectName: string;
  workflowId: string;
  selectedAgents: string[];
  onAgentsChange: (agents: string[]) => void;
};

export function RfeAgentsCard({
  projectName,
  workflowId,
  selectedAgents,
  onAgentsChange,
}: RfeAgentsCardProps) {
  // Use React Query hook for agents
  const { data: repoAgents = AVAILABLE_AGENTS, isLoading: loadingAgents } = useRfeWorkflowAgents(
    projectName,
    workflowId
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Bot className="h-5 w-5" />
          Agents
        </CardTitle>
        <CardDescription>
          {loadingAgents ? 'Loading agents from repository...' : 'Select agents to participate in workflow sessions'}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {loadingAgents ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : repoAgents.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            <Bot className="h-12 w-12 mx-auto mb-2 opacity-50" />
            <p>No agents found in repository .claude/agents directory</p>
            <p className="text-xs mt-1">Seed the repository to add agent definitions</p>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {repoAgents.map((agent) => {
                const isSelected = selectedAgents.includes(agent.persona);
                return (
                  <div
                    key={agent.persona}
                    className={`p-3 rounded-lg border transition-colors ${
                      isSelected ? 'bg-primary/5 border-primary' : 'bg-background border-border hover:border-primary/50'
                    }`}
                  >
                    <label className="flex items-start gap-3 cursor-pointer">
                      <Checkbox
                        checked={isSelected}
                        onCheckedChange={(checked) => {
                          onAgentsChange(
                            checked
                              ? [...selectedAgents, agent.persona]
                              : selectedAgents.filter(p => p !== agent.persona)
                          );
                        }}
                        className="mt-0.5"
                      />
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-sm">{agent.name}</div>
                        <div className="text-xs text-muted-foreground mt-0.5">{agent.role}</div>
                      </div>
                    </label>
                  </div>
                );
              })}
            </div>
            {selectedAgents.length > 0 && (
              <div className="mt-4 pt-4 border-t">
                <div className="text-sm font-medium mb-2">Selected Agents ({selectedAgents.length})</div>
                <div className="flex flex-wrap gap-2">
                  {selectedAgents.map(persona => {
                    const agent = repoAgents.find(a => a.persona === persona);
                    return agent ? (
                      <Badge key={persona} variant="secondary">
                        {agent.name}
                      </Badge>
                    ) : null;
                  })}
                </div>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}

