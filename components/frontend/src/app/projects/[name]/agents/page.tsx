"use client";

import { AVAILABLE_AGENTS, groupAgentsByRole } from "@/lib/agents";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Users, Briefcase, Palette, FileText, Settings } from "lucide-react";

const categoryIcons = {
  "Engineering": Briefcase,
  "Design": Palette,
  "Product": Users,
  "Content": FileText,
  "Process & Leadership": Settings,
};

const categoryDescriptions = {
  "Engineering": "Technical leadership, implementation, and architecture",
  "Design": "User experience, design systems, and accessibility",
  "Product": "Strategy, customer feedback, and business value",
  "Content": "Documentation, content strategy, and technical writing",
  "Process & Leadership": "Team coordination, agile practices, and delivery management",
};

export default function AgentsPage() {
  const groupedAgents = groupAgentsByRole();
  const categories = Object.keys(groupedAgents).sort();

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-4xl font-bold mb-2">Claude Code Agents</h1>
        <p className="text-muted-foreground text-lg">
          Meet the {AVAILABLE_AGENTS.length} specialized AI agents available in the vTeam workflow system.
          Each agent brings unique expertise and perspective to help you build better software.
        </p>
      </div>

      <div className="mb-8 p-6 bg-muted/50 rounded-lg">
        <h3 className="text-lg font-semibold mb-4">How to Use Agents</h3>
        <div className="space-y-4">
          <div>
            <div className="font-medium text-foreground mb-1">In RFE Workflows</div>
            <p className="text-sm text-muted-foreground leading-relaxed">
              During RFE sessions you can request specific agents by name to get their specialized perspectives.
            </p>
            <div className="mt-2 text-sm italic text-muted-foreground/80 pl-4 border-l-2 border-muted-foreground/20">
              Example: &quot;Use Stella Staff Engineer to review the architecture&quot;
            </div>
          </div>
          <div>
            <div className="font-medium text-foreground mb-1">Custom Teams</div>
            <p className="text-sm text-muted-foreground leading-relaxed">
              Mix and match agents based on your feature&apos;s specific needs by requesting a set of agents by name.
            </p>
            <div className="mt-2 text-sm italic text-muted-foreground/80 pl-4 border-l-2 border-muted-foreground/20">
              Example: &quot;Make sure Product Parker and Stella Staff Engineer both review the document&quot;
            </div>
          </div>
        </div>
      </div>

      <div className="space-y-8">
        {categories.map((category) => {
          const Icon = categoryIcons[category as keyof typeof categoryIcons] || Users;
          const agents = groupedAgents[category];

          return (
            <div key={category} className="space-y-4">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 rounded-lg bg-primary/10">
                  <Icon className="w-6 h-6 text-primary" />
                </div>
                <div>
                  <h2 className="text-2xl font-semibold">{category}</h2>
                  <p className="text-sm text-muted-foreground">
                    {categoryDescriptions[category as keyof typeof categoryDescriptions]}
                  </p>
                </div>
                <Badge variant="secondary" className="ml-auto">
                  {agents.length} {agents.length === 1 ? "agent" : "agents"}
                </Badge>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {agents.map((agent) => (
                  <Card key={agent.persona} className="hover:shadow-md transition-shadow">
                    <CardHeader>
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <CardTitle className="text-lg">{agent.name}</CardTitle>
                          <CardDescription className="text-sm mt-1">
                            {agent.role}
                          </CardDescription>
                        </div>
                      </div>
                    </CardHeader>
                    <CardContent>
                      <p className="text-sm text-muted-foreground leading-relaxed">
                        {agent.description}
                      </p>
                      <div className="mt-4 pt-4 border-t">
                        <code className="text-xs bg-muted px-2 py-1 rounded">
                          {agent.persona}
                        </code>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
