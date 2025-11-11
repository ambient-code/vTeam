"use client";

import { useState } from "react";
import { Play, Loader2, Workflow, ChevronDown, ChevronRight, Sparkles, Info, AlertCircle } from "lucide-react";
import { AccordionItem, AccordionTrigger, AccordionContent } from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import type { WorkflowConfig } from "../../lib/types";

type WorkflowMetadata = {
  commands: Array<{ id: string; name: string; slashCommand: string; description?: string }>;
  agents: Array<{ id: string; name: string; description?: string }>;
};

type WorkflowsAccordionProps = {
  sessionPhase?: string;
  activeWorkflow: string | null;
  selectedWorkflow: string;
  pendingWorkflow: WorkflowConfig | null;
  workflowActivating: boolean;
  workflowMetadata?: WorkflowMetadata;
  ootbWorkflows: WorkflowConfig[];
  selectedAgents: string[];
  autoSelectAgents: boolean;
  isExpanded: boolean;
  onWorkflowChange: (value: string) => void;
  onActivateWorkflow: () => void;
  onCommandClick: (slashCommand: string) => void;
  onSetSelectedAgents: (agents: string[]) => void;
  onSetAutoSelectAgents: (auto: boolean) => void;
  onResume?: () => void;
};

export function WorkflowsAccordion({
  sessionPhase,
  activeWorkflow,
  selectedWorkflow,
  pendingWorkflow,
  workflowActivating,
  workflowMetadata,
  ootbWorkflows,
  selectedAgents,
  autoSelectAgents,
  isExpanded,
  onWorkflowChange,
  onActivateWorkflow,
  onCommandClick,
  onSetSelectedAgents,
  onSetAutoSelectAgents,
  onResume,
}: WorkflowsAccordionProps) {
  const [showCommandsList, setShowCommandsList] = useState(false);
  const [showAgentsList, setShowAgentsList] = useState(false);
  const [commandsScrollTop, setCommandsScrollTop] = useState(false);
  const [commandsScrollBottom, setCommandsScrollBottom] = useState(true);
  const [agentsScrollTop, setAgentsScrollTop] = useState(false);
  const [agentsScrollBottom, setAgentsScrollBottom] = useState(true);

  const isSessionStopped = sessionPhase === 'Stopped' || sessionPhase === 'Error' || sessionPhase === 'Completed';

  return (
    <AccordionItem value="workflows" className="border rounded-lg px-3 bg-white">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <Workflow className="h-4 w-4" />
          <span>Workflows</span>
          {activeWorkflow && !isExpanded && (
            <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
              {ootbWorkflows.find(w => w.id === activeWorkflow)?.name || "Custom Workflow"}
            </Badge>
          )}
        </div>
      </AccordionTrigger>
      <AccordionContent className="pt-2 pb-3">
        {isSessionStopped ? (
          <div className="py-8 flex flex-col items-center justify-center space-y-4">
            <Play className="h-12 w-12 text-muted-foreground/50" />
            <div className="text-center space-y-1">
              <h3 className="font-medium text-sm">Session not running</h3>
              <p className="text-sm text-muted-foreground">
                You need to resume this session to use workflows.
              </p>
            </div>
            {onResume && sessionPhase === 'Stopped' && (
              <Button
                onClick={onResume}
                size="sm"
                className="hover:border-green-600 hover:bg-green-50 group"
                variant="outline"
              >
                <Play className="w-4 h-4 mr-2 fill-green-200 stroke-green-600 group-hover:fill-green-500 group-hover:stroke-green-700 transition-colors" />
                Resume Session
              </Button>
            )}
          </div>
        ) : (
          <div className="space-y-3">
            {/* Workflow selector - always visible except when activating */}
            {!workflowActivating && (
              <>
                <p className="text-sm text-muted-foreground">
                  Workflows provide agents with pre-defined context and structured steps to follow.
                </p>
                
                <div>
                  <Select value={selectedWorkflow} onValueChange={onWorkflowChange} disabled={workflowActivating}>
                    <SelectTrigger className="w-full h-auto py-8">
                      <SelectValue placeholder="Generic chat" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">
                        <div className="flex flex-col items-start gap-0.5 py-1">
                          <span>General chat</span>
                          <span className="text-xs text-muted-foreground font-normal">
                            A general chat session with no structured workflow.
                          </span>
                        </div>
                      </SelectItem>
                      {ootbWorkflows.map((workflow) => (
                        <SelectItem 
                          key={workflow.id} 
                          value={workflow.id}
                          disabled={!workflow.enabled}
                        >
                          <div className="flex flex-col items-start gap-0.5 py-1">
                            <span>{workflow.name}</span>
                            <span className="text-xs text-muted-foreground font-normal">
                              {workflow.description}
                            </span>
                          </div>
                        </SelectItem>
                      ))}
                      <SelectSeparator />
                      <SelectItem value="custom">
                        <div className="flex flex-col items-start gap-0.5 py-1">
                          <span>Custom workflow...</span>
                          <span className="text-xs text-muted-foreground font-normal">
                            Load a workflow from a custom Git repository
                          </span>
                        </div>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                
                {/* Show workflow preview and activate/switch button */}
                {pendingWorkflow && (
                  <Alert variant="info">
                    <AlertCircle />
                    <AlertTitle>
                      Reload required
                    </AlertTitle>
                    <AlertDescription>
                      <div className="space-y-2 mt-2">
                        <p className="text-sm">
                          Please reload this chat session to switch to the new workflow. Your chat history will be preserved.
                        </p>
                        <Button 
                          onClick={onActivateWorkflow}
                          className="w-full mt-3"
                          size="sm"
                        >
                          <Play className="mr-2 h-4 w-4" />
                          Load new workflow
                        </Button>
                      </div>
                    </AlertDescription>
                  </Alert>
                )}
              </>
            )}
            
            {/* Show active workflow info */}
            {activeWorkflow && !workflowActivating && (
              <>
                {/* Commands Section */}
                {workflowMetadata?.commands && workflowMetadata.commands.length > 0 && (
                  <div className="space-y-2">
                    <div>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="w-full justify-between h-8 px-2"
                        onClick={() => setShowCommandsList(!showCommandsList)}
                      >
                        <span className="text-xs font-medium">
                          {showCommandsList ? 'Hide' : 'Show'} {workflowMetadata.commands.length} available command{workflowMetadata.commands.length !== 1 ? 's' : ''}
                        </span>
                        {showCommandsList ? (
                          <ChevronDown className="h-3 w-3" />
                        ) : (
                          <ChevronRight className="h-3 w-3" />
                        )}
                      </Button>

                      {showCommandsList && (
                        <div className="relative mt-2">
                          {commandsScrollTop && (
                            <div className="absolute top-0 left-0 right-0 h-8 bg-gradient-to-b from-white to-transparent pointer-events-none z-10" />
                          )}
                          <div 
                            className="max-h-[400px] overflow-y-auto space-y-2 pr-1"
                            onScroll={(e) => {
                              const target = e.currentTarget;
                              const isScrolledFromTop = target.scrollTop > 10;
                              const isScrolledToBottom = target.scrollHeight - target.scrollTop <= target.clientHeight + 10;
                              setCommandsScrollTop(isScrolledFromTop);
                              setCommandsScrollBottom(!isScrolledToBottom);
                            }}
                          >
                            {workflowMetadata.commands.map((cmd) => {
                              const commandTitle = cmd.name.includes('.') 
                                ? cmd.name.split('.').pop() 
                                : cmd.name;
                              
                              return (
                                <div
                                  key={cmd.id}
                                  className="p-3 rounded-md border bg-muted/30"
                                >
                                  <div className="flex items-center justify-between mb-1">
                                    <h3 className="text-sm font-bold capitalize">
                                      {commandTitle}
                                    </h3>
                                    <Button
                                      variant="outline"
                                      size="sm"
                                      className="flex-shrink-0 h-7 text-xs"
                                      onClick={() => onCommandClick(cmd.slashCommand)}
                                    >
                                      Run {cmd.slashCommand.replace(/^\/speckit\./, '/')}
                                    </Button>
                                  </div>
                                  {cmd.description && (
                                    <p className="text-xs text-muted-foreground">
                                      {cmd.description}
                                    </p>
                                  )}
                                </div>
                              );
                            })}
                          </div>
                          {commandsScrollBottom && (
                            <div className="absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-white to-transparent pointer-events-none z-10" />
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {workflowMetadata?.commands?.length === 0 && (
                  <p className="text-xs text-muted-foreground text-left py-2">
                    No commands found in this workflow
                  </p>
                )}

                {/* Agents Section */}
                {workflowMetadata?.agents && workflowMetadata.agents.length > 0 && (
                  <div className="space-y-2">
                    <div>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="w-full justify-between h-8 px-2"
                        onClick={() => setShowAgentsList(!showAgentsList)}
                      >
                        <span className="text-xs font-medium">
                          {showAgentsList ? 'Hide' : 'Show'} {workflowMetadata.agents.length} available agent{workflowMetadata.agents.length !== 1 ? 's' : ''}
                        </span>
                        {showAgentsList ? (
                          <ChevronDown className="h-3 w-3" />
                        ) : (
                          <ChevronRight className="h-3 w-3" />
                        )}
                      </Button>

                      {showAgentsList && (
                        <div className="mt-2 pt-2 mx-3 space-y-2">
                          {/* Auto-select checkbox */}
                          <div className="flex items-center space-x-2 pb-2">
                            <Checkbox
                              id="auto-select-agents"
                              checked={autoSelectAgents}
                              onCheckedChange={(checked) => {
                                onSetAutoSelectAgents(!!checked);
                                if (checked) onSetSelectedAgents([]);
                              }}
                            />
                            <Sparkles className="h-3 w-3 text-purple-500" />
                            <Label htmlFor="auto-select-agents" className="text-sm font-normal cursor-pointer">
                              Automatically select recommended agents for each task
                            </Label>
                          </div>
                          
                          {/* Scrollable agents list */}
                          <div className="relative">
                            {agentsScrollTop && (
                              <div className="absolute top-0 left-0 right-0 h-8 bg-gradient-to-b from-white to-transparent pointer-events-none z-10" />
                            )}
                            <div 
                              className="max-h-48 overflow-y-auto space-y-1 pr-1"
                              onScroll={(e) => {
                                const target = e.currentTarget;
                                const isScrolledFromTop = target.scrollTop > 10;
                                const isScrolledToBottom = target.scrollHeight - target.scrollTop <= target.clientHeight + 10;
                                setAgentsScrollTop(isScrolledFromTop);
                                setAgentsScrollBottom(!isScrolledToBottom);
                              }}
                            >
                              <div className="space-y-1 space-x-6">
                                {workflowMetadata.agents.map((agent) => (
                                  <div key={agent.id} className="flex items-center gap-2 group">
                                    <Checkbox
                                      id={`agent-${agent.id}`}
                                      checked={selectedAgents.includes(agent.id)}
                                      disabled={autoSelectAgents}
                                      onCheckedChange={(checked) => {
                                        if (checked) {
                                          onSetSelectedAgents([...selectedAgents, agent.id]);
                                        } else {
                                          onSetSelectedAgents(selectedAgents.filter(id => id !== agent.id));
                                        }
                                      }}
                                    />
                                    <Label
                                      htmlFor={`agent-${agent.id}`}
                                      className="text-sm font-normal cursor-pointer"
                                    >
                                      {agent.name}
                                    </Label>
                                    <Popover>
                                      <PopoverTrigger asChild>
                                        <button
                                          className="p-0.5 hover:bg-gray-100 rounded flex-shrink-0"
                                          onClick={(e) => {
                                            e.preventDefault();
                                            e.stopPropagation();
                                          }}
                                        >
                                          <Info className="h-3.5 w-3.5 text-muted-foreground" />
                                        </button>
                                      </PopoverTrigger>
                                      <PopoverContent className="max-w-xs" align="start">
                                        <div className="space-y-2">
                                          <p className="font-semibold text-sm">{agent.name}</p>
                                          <p className="text-xs text-muted-foreground">{agent.description}</p>
                                        </div>
                                      </PopoverContent>
                                    </Popover>
                                  </div>
                                ))}
                              </div>
                            </div>
                            {agentsScrollBottom && (
                              <div className="absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-white to-transparent pointer-events-none z-10" />
                            )}
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {workflowMetadata?.agents?.length === 0 && (
                  <p className="text-xs text-muted-foreground text-left py-2">
                    No agents found in this workflow
                  </p>
                )}
              </>
            )}
            
            {/* Show activating/switching state */}
            {workflowActivating && (
              <Alert>
                <Loader2 className="h-4 w-4 animate-spin" />
                <AlertTitle>{activeWorkflow ? 'Switching Workflow...' : 'Activating Workflow...'}</AlertTitle>
                <AlertDescription>
                  <div className="space-y-2">
                    <p>Please wait. This may take 10-20 seconds...</p>
                  </div>
                </AlertDescription>
              </Alert>
            )}
          </div>
        )}
      </AccordionContent>
    </AccordionItem>
  );
}

