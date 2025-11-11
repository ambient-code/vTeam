"use client";

import React, { useState, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Brain, Loader2, Settings, Sparkles, Users, Terminal } from "lucide-react";
import { StreamMessage } from "@/components/ui/stream-message";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuCheckboxItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import type { AgenticSession, MessageObject, ToolUseMessages } from "@/types/agentic-session";

type WorkflowMetadata = {
  commands: Array<{ id: string; name: string; slashCommand: string; description?: string }>;
  agents: Array<{ id: string; name: string; description?: string }>;
};

export type MessagesTabProps = {
  session: AgenticSession;
  streamMessages: Array<MessageObject | ToolUseMessages>;
  chatInput: string;
  setChatInput: (v: string) => void;
  onSendChat: () => Promise<void>;
  onInterrupt: () => Promise<void>;
  onEndSession: () => Promise<void>;
  onGoToResults?: () => void;
  onContinue: () => void;
  selectedAgents?: string[];
  autoSelectAgents?: boolean;
  workflowMetadata?: WorkflowMetadata;
  onSetSelectedAgents?: (agents: string[]) => void;
  onSetAutoSelectAgents?: (auto: boolean) => void;
  onCommandClick?: (slashCommand: string) => void;
};


const MessagesTab: React.FC<MessagesTabProps> = ({ session, streamMessages, chatInput, setChatInput, onSendChat, onInterrupt, onEndSession, onGoToResults, onContinue, selectedAgents = [], autoSelectAgents = false, workflowMetadata, onSetSelectedAgents, onSetAutoSelectAgents, onCommandClick }) => {
  const [sendingChat, setSendingChat] = useState(false);
  const [interrupting, setInterrupting] = useState(false);
  const [ending, setEnding] = useState(false);
  const [showSystemMessages, setShowSystemMessages] = useState(false);
  const [agentsPopoverOpen, setAgentsPopoverOpen] = useState(false);
  const [commandsPopoverOpen, setCommandsPopoverOpen] = useState(false);
  
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const [isAtBottom, setIsAtBottom] = useState(true);

  const phase = session?.status?.phase || "";
  const isInteractive = session?.spec?.interactive;
  
  // Only show chat interface when session is interactive AND in Running state
  const showChatInterface = isInteractive && phase === "Running";
  
  // Determine if session is in a terminal state
  const isTerminalState = ["Completed", "Failed", "Stopped"].includes(phase);
  const isCreating = ["Creating", "Pending"].includes(phase);

  // Filter out system messages unless showSystemMessages is true
  const filteredMessages = streamMessages.filter((msg) => {
    if (showSystemMessages) return true;
    
    // Hide system_message type by default
    // Check if msg has a type property and if it's a system_message
    if ('type' in msg && msg.type === "system_message") {
      return false;
    }
    
    return true;
  });

  // Check if user is scrolled to the bottom
  const checkIfAtBottom = () => {
    const container = messagesContainerRef.current;
    if (!container) return true;
    
    // For normal scroll (not reversed), we check if scrollTop + clientHeight >= scrollHeight
    const threshold = 50; // pixels from bottom to still consider "at bottom"
    const isBottom = container.scrollHeight - container.scrollTop - container.clientHeight < threshold;
    return isBottom;
  };

  // Handle scroll event to track if user is at bottom
  const handleScroll = () => {
    setIsAtBottom(checkIfAtBottom());
  };

  // Scroll to bottom function - only scrolls the messages container, not the whole page
  const scrollToBottom = () => {
    const container = messagesContainerRef.current;
    if (container) {
      container.scrollTop = container.scrollHeight;
    }
  };

  // Auto-scroll to bottom when new messages arrive, but only if user was already at bottom
  useEffect(() => {
    if (isAtBottom) {
      scrollToBottom();
    }
  }, [filteredMessages, isAtBottom]);

  // Initial scroll to bottom on mount
  useEffect(() => {
    scrollToBottom();
  }, []);

  const handleSendChat = async () => {
    setSendingChat(true);
    try {
      await onSendChat();
    } finally {
      setSendingChat(false);
    }
  };

  const handleInterrupt = async () => {
    setInterrupting(true);
    try {
      await onInterrupt();
    } finally {
      setInterrupting(false);
    }
  };

  const handleEndSession = async () => {
    setEnding(true);
    try {
      await onEndSession();
    } finally {
      setEnding(false);
    }
  };

  return (
    <div className="flex flex-col h-full">
      <div 
        ref={messagesContainerRef}
        onScroll={handleScroll}
        className="flex-1 flex flex-col gap-2 overflow-y-auto pr-1 pb-6"
      >
        {filteredMessages.map((m, idx) => (
          <StreamMessage key={`sm-${idx}`} message={m} isNewest={idx === filteredMessages.length - 1} onGoToResults={onGoToResults} />
        ))}

        {filteredMessages.length === 0 && (
          <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
            <Brain className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No messages yet</p>
            <p className="text-xs mt-1">
              {isInteractive 
                ? isCreating 
                  ? "Session is starting..."
                  : isTerminalState
                  ? `Session has ${phase.toLowerCase()}.`
                  : "Start by sending a message below."
                : "This session is not interactive."}
            </p>
          </div>
        )}
      </div>

      {/* Settings for non-interactive sessions with messages */}
      {!isInteractive && filteredMessages.length > 0 && (
        <div className="sticky bottom-0 border-t bg-gray-50">
          <div className="p-3">
            <div className="flex items-center gap-2">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
                    <Settings className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start">
                  <DropdownMenuCheckboxItem
                    checked={showSystemMessages}
                    onCheckedChange={setShowSystemMessages}
                  >
                    {showSystemMessages ? 'Hide' : 'Show'} system messages
                  </DropdownMenuCheckboxItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <p className="text-sm text-muted-foreground">Non-interactive session</p>
            </div>
          </div>
        </div>
      )}

      {showChatInterface && (
        <div className="sticky bottom-0 border-t bg-white">
          <div className="p-3">
            <div className="border rounded-md p-3 space-y-2 bg-white">
              <textarea
                className="w-full border rounded p-2 text-sm"
                placeholder="Type a message to the agent... (Press Enter to send, Shift+Enter for new line)"
                value={chatInput}
                onChange={(e) => setChatInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    if (chatInput.trim() && !sendingChat) {
                      handleSendChat();
                    }
                  }
                }}
                rows={3}
                disabled={sendingChat}
              />
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
                        <Settings className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="start">
                      <DropdownMenuCheckboxItem
                        checked={showSystemMessages}
                        onCheckedChange={setShowSystemMessages}
                      >
                        {showSystemMessages ? 'Hide' : 'Show'} system messages
                      </DropdownMenuCheckboxItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                  
                  {/* Agents Button with Popover */}
                  {workflowMetadata?.agents && workflowMetadata.agents.length > 0 && (
                    <Popover open={agentsPopoverOpen} onOpenChange={setAgentsPopoverOpen}>
                      <PopoverTrigger asChild>
                        <Button variant="outline" size="sm" className="h-7 gap-1.5">
                          <Users className="h-3.5 w-3.5" />
                          Agents
                          <Badge variant="secondary" className="ml-0.5 h-4 px-1.5 text-[10px] font-medium">
                            {autoSelectAgents ? "auto" : selectedAgents.length}
                          </Badge>
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent 
                        align="start" 
                        side="top" 
                        className="w-[500px]"
                      >
                        <div className="space-y-3">
                          <div className="space-y-2">
                            <h4 className="font-medium text-sm">Agent Selection</h4>
                            <p className="text-xs text-muted-foreground">
                              Select agents to collaborate with or enable automatic selection
                            </p>
                          </div>

                          {/* Auto-select checkbox */}
                          <div className="flex items-center space-x-2 pb-2 border-b">
                            <Checkbox
                              id="popover-auto-select-agents"
                              checked={autoSelectAgents}
                              onCheckedChange={(checked) => {
                                if (onSetAutoSelectAgents) {
                                  onSetAutoSelectAgents(!!checked);
                                  if (checked && onSetSelectedAgents) {
                                    onSetSelectedAgents([]);
                                  }
                                }
                              }}
                            />
                            <Sparkles className="h-3.5 w-3.5 text-purple-500" />
                            <Label htmlFor="popover-auto-select-agents" className="text-sm font-normal cursor-pointer">
                              Automatically select agents
                            </Label>
                          </div>
                          
                          {/* Agents list */}
                          <div 
                            className="max-h-[400px] overflow-y-scroll space-y-2 pr-2"
                            style={{
                              scrollbarWidth: 'thin',
                              scrollbarColor: '#d1d5db #f3f4f6'
                            }}
                          >
                            {workflowMetadata.agents.map((agent) => {
                              const agentNameShort = agent.name.split(' - ')[0];
                              const isSelected = selectedAgents.includes(agent.id);
                              
                              return (
                                <div
                                  key={agent.id}
                                  className={`p-3 rounded-md border cursor-pointer transition-colors ${
                                    isSelected 
                                      ? 'bg-blue-50 border-blue-300 hover:bg-blue-100' 
                                      : 'bg-muted/30 hover:bg-muted/50'
                                  } ${autoSelectAgents ? 'opacity-50 cursor-not-allowed' : ''}`}
                                  onClick={() => {
                                    if (!autoSelectAgents && onSetSelectedAgents) {
                                      if (isSelected) {
                                        onSetSelectedAgents(selectedAgents.filter(id => id !== agent.id));
                                      } else {
                                        onSetSelectedAgents([...selectedAgents, agent.id]);
                                      }
                                    }
                                  }}
                                >
                                  <div className="flex items-center justify-between mb-1">
                                    <h3 className="text-sm font-bold">
                                      {agent.name}
                                    </h3>
                                    <Button
                                      variant="outline"
                                      size="sm"
                                      className="flex-shrink-0 h-7 text-xs"
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        setChatInput(chatInput + `@${agentNameShort} `);
                                        setAgentsPopoverOpen(false);
                                      }}
                                    >
                                      @{agentNameShort}
                                    </Button>
                                  </div>
                                  {agent.description && (
                                    <p className="text-xs text-muted-foreground">
                                      {agent.description}
                                    </p>
                                  )}
                                </div>
                              );
                            })}
                          </div>
                        </div>
                      </PopoverContent>
                    </Popover>
                  )}

                  {/* Commands Button with Popover */}
                  {workflowMetadata?.commands && workflowMetadata.commands.length > 0 && (
                    <Popover open={commandsPopoverOpen} onOpenChange={setCommandsPopoverOpen}>
                      <PopoverTrigger asChild>
                        <Button variant="outline" size="sm" className="h-7 gap-1.5">
                          <Terminal className="h-3.5 w-3.5" />
                          Commands
                          <Badge variant="secondary" className="ml-0.5 h-4 px-1.5 text-[10px] font-medium">
                            {workflowMetadata.commands.length}
                          </Badge>
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent 
                        align="start" 
                        side="top" 
                        className="w-[500px]"
                      >
                        <div className="space-y-3">
                          <div className="space-y-2">
                            <h4 className="font-medium text-sm">Available Commands</h4>
                            <p className="text-xs text-muted-foreground">
                              Run workflow commands to perform specific actions
                            </p>
                          </div>
                          
                          {/* Commands list */}
                          <div 
                            className="max-h-[400px] overflow-y-scroll space-y-2 pr-2"
                            style={{
                              scrollbarWidth: 'thin',
                              scrollbarColor: '#d1d5db #f3f4f6'
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
                                      onClick={() => {
                                        if (onCommandClick) {
                                          onCommandClick(cmd.slashCommand);
                                          setCommandsPopoverOpen(false);
                                        }
                                      }}
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
                        </div>
                      </PopoverContent>
                    </Popover>
                  )}
                </div>
                <div className="flex gap-2">
                  <Button 
                    variant="outline" 
                    size="sm" 
                    onClick={handleInterrupt}
                    disabled={interrupting || sendingChat || ending}
                  >
                    {interrupting && <Loader2 className="w-3 h-3 mr-1 animate-spin" />}
                    Interrupt agent
                  </Button>
                  <Button 
                    variant="secondary" 
                    size="sm" 
                    onClick={handleEndSession}
                    disabled={ending || sendingChat || interrupting}
                  >
                    {ending && <Loader2 className="w-3 h-3 mr-1 animate-spin" />}
                    End session
                  </Button>
                  <Button 
                    size="sm" 
                    onClick={handleSendChat} 
                    disabled={!chatInput.trim() || sendingChat || interrupting || ending}
                  >
                    {sendingChat && <Loader2 className="w-3 h-3 mr-1 animate-spin" />}
                    Send
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {isInteractive && !showChatInterface && streamMessages.length > 0 && (
        <div className="sticky bottom-0 border-t bg-gray-50">
          <div className="p-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
                      <Settings className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuCheckboxItem
                      checked={showSystemMessages}
                      onCheckedChange={setShowSystemMessages}
                    >
                      {showSystemMessages ? 'Hide' : 'Show'} system messages
                    </DropdownMenuCheckboxItem>
                  </DropdownMenuContent>
                </DropdownMenu>
                <p className="text-sm text-muted-foreground">
                  {isCreating && "Chat will be available once the session is running..."}
                  {isTerminalState && (
                    <>
                      This session has {phase.toLowerCase()}. Chat is no longer available.
                      {onContinue && (
                        <>
                          {" "}
                          <button
                            onClick={onContinue}
                            className="text-blue-600 hover:underline font-medium"
                          >
                            Resume this session
                          </button>
                          {" "}to restart it.
                        </>
                      )}
                    </>
                  )}
                </p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default MessagesTab;


