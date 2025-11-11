"use client";

import React, { useState, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { MessageSquare, Loader2, Settings, Sparkles, Users, Terminal } from "lucide-react";
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
import type { WorkflowMetadata } from "@/app/projects/[name]/sessions/[sessionName]/lib/types";

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
  
  // Autocomplete state
  const [autocompleteOpen, setAutocompleteOpen] = useState(false);
  const [autocompleteType, setAutocompleteType] = useState<'agent' | 'command' | null>(null);
  const [autocompleteFilter, setAutocompleteFilter] = useState('');
  const [autocompleteTriggerPos, setAutocompleteTriggerPos] = useState(0);
  const [autocompleteSelectedIndex, setAutocompleteSelectedIndex] = useState(0);
  
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const autocompleteRef = useRef<HTMLDivElement>(null);
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

  // Click outside to close autocomplete
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (autocompleteOpen && 
          autocompleteRef.current && 
          !autocompleteRef.current.contains(event.target as Node) &&
          textareaRef.current &&
          !textareaRef.current.contains(event.target as Node)) {
        setAutocompleteOpen(false);
        setAutocompleteType(null);
        setAutocompleteFilter('');
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [autocompleteOpen]);

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

  // Get filtered autocomplete items
  const getFilteredItems = () => {
    if (!autocompleteType) return [];
    
    if (autocompleteType === 'agent' && workflowMetadata?.agents) {
      const filter = autocompleteFilter.toLowerCase();
      return workflowMetadata.agents.filter(agent => 
        agent.name.toLowerCase().includes(filter)
      );
    }
    
    if (autocompleteType === 'command' && workflowMetadata?.commands) {
      const filter = autocompleteFilter.toLowerCase();
      return workflowMetadata.commands.filter(cmd => 
        cmd.name.toLowerCase().includes(filter) || 
        cmd.slashCommand.toLowerCase().includes(filter)
      );
    }
    
    return [];
  };

  const filteredAutocompleteItems = getFilteredItems();

  // Handle autocomplete selection
  const handleAutocompleteSelect = (item: { id: string; name: string; slashCommand?: string; description?: string }) => {
    if (!textareaRef.current) return;
    
    const cursorPos = textareaRef.current.selectionStart;
    const textBefore = chatInput.substring(0, autocompleteTriggerPos);
    const textAfter = chatInput.substring(cursorPos);
    
    let insertText = '';
    if (autocompleteType === 'agent') {
      const agentNameShort = item.name.split(' - ')[0];
      insertText = `@${agentNameShort} `;
    } else if (autocompleteType === 'command') {
      insertText = `${item.slashCommand} `;
    }
    
    const newText = textBefore + insertText + textAfter;
    setChatInput(newText);
    
    // Reset autocomplete
    setAutocompleteOpen(false);
    setAutocompleteType(null);
    setAutocompleteFilter('');
    setAutocompleteSelectedIndex(0);
    
    // Set cursor position after insert
    setTimeout(() => {
      if (textareaRef.current) {
        const newCursorPos = textBefore.length + insertText.length;
        textareaRef.current.selectionStart = newCursorPos;
        textareaRef.current.selectionEnd = newCursorPos;
        textareaRef.current.focus();
      }
    }, 0);
  };

  // Handle input change to detect @ or /
  const handleChatInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    const cursorPos = e.target.selectionStart;
    
    setChatInput(newValue);
    
    // Check if we should show autocomplete
    if (cursorPos > 0) {
      const charBeforeCursor = newValue[cursorPos - 1];
      const textBeforeCursor = newValue.substring(0, cursorPos);
      
      // Check for @ or / trigger
      if (charBeforeCursor === '@' || charBeforeCursor === '/') {
        // Make sure it's at the start or after whitespace
        if (cursorPos === 1 || /\s/.test(newValue[cursorPos - 2])) {
          setAutocompleteTriggerPos(cursorPos - 1);
          setAutocompleteType(charBeforeCursor === '@' ? 'agent' : 'command');
          setAutocompleteFilter('');
          setAutocompleteSelectedIndex(0);
          setAutocompleteOpen(true);
          return;
        }
      }
      
      // Update filter if autocomplete is open
      if (autocompleteOpen) {
        const filterText = textBeforeCursor.substring(autocompleteTriggerPos + 1);
        
        // Close if we've moved past the trigger or hit whitespace
        if (cursorPos <= autocompleteTriggerPos || /\s/.test(filterText)) {
          setAutocompleteOpen(false);
          setAutocompleteType(null);
          setAutocompleteFilter('');
        } else {
          setAutocompleteFilter(filterText);
          setAutocompleteSelectedIndex(0);
        }
      }
    } else {
      // Cursor at start, close autocomplete
      if (autocompleteOpen) {
        setAutocompleteOpen(false);
        setAutocompleteType(null);
        setAutocompleteFilter('');
      }
    }
  };

  return (
    <div className="flex flex-col h-full">
      <div 
        ref={messagesContainerRef}
        onScroll={handleScroll}
        className="flex-1 flex flex-col gap-2 overflow-y-auto px-3 pb-2 scrollbar-thin"
      >
        {filteredMessages.map((m, idx) => (
          <StreamMessage key={`sm-${idx}`} message={m} isNewest={idx === filteredMessages.length - 1} onGoToResults={onGoToResults} />
        ))}

        {filteredMessages.length === 0 && !isCreating && (
          <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
            <MessageSquare className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No messages yet</p>
            <p className="text-xs mt-1">
              {isInteractive 
                ? isTerminalState
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
        <div className="sticky bottom-0 bg-white">
          <div className="px-2 pt-2 pb-0 space-y-1.5">
              <div className="relative">
                <textarea
                  ref={textareaRef}
                  className="w-full border rounded p-2 text-sm"
                  placeholder="Type a message to the agent... (Press Enter to send, Shift+Enter for new line)"
                  value={chatInput}
                  onChange={handleChatInputChange}
                  onKeyDown={(e) => {
                    // Handle autocomplete navigation
                    if (autocompleteOpen && filteredAutocompleteItems.length > 0) {
                      if (e.key === "ArrowDown") {
                        e.preventDefault();
                        setAutocompleteSelectedIndex(prev => 
                          prev < filteredAutocompleteItems.length - 1 ? prev + 1 : prev
                        );
                        return;
                      }
                      if (e.key === "ArrowUp") {
                        e.preventDefault();
                        setAutocompleteSelectedIndex(prev => prev > 0 ? prev - 1 : 0);
                        return;
                      }
                      if (e.key === "Enter" || e.key === "Tab") {
                        e.preventDefault();
                        handleAutocompleteSelect(filteredAutocompleteItems[autocompleteSelectedIndex]);
                        return;
                      }
                      if (e.key === "Escape") {
                        e.preventDefault();
                        setAutocompleteOpen(false);
                        setAutocompleteType(null);
                        setAutocompleteFilter('');
                        return;
                      }
                    }
                    
                    // Regular enter to send
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
                
                {/* Autocomplete popup */}
                {autocompleteOpen && (
                  <div 
                    ref={autocompleteRef}
                    className="absolute z-[100] bg-white border-2 border-blue-500 rounded-md shadow-lg max-h-60 overflow-y-auto w-80"
                    style={{
                      bottom: '100%',
                      left: '0px',
                      marginBottom: '5px',
                    }}
                  >
                    {filteredAutocompleteItems.length === 0 ? (
                      <div className="px-3 py-2 text-sm text-muted-foreground">
                        No {autocompleteType === 'agent' ? 'agents' : 'commands'} found
                      </div>
                    ) : (
                      <>
                    {filteredAutocompleteItems.map((item, index) => {
                      if (autocompleteType === 'agent') {
                        const agent = item as { id: string; name: string; description?: string };
                        const agentNameShort = agent.name.split(' - ')[0];
                        
                        return (
                          <div
                            key={agent.id}
                            className={`px-3 py-2 cursor-pointer border-b last:border-b-0 ${
                              index === autocompleteSelectedIndex
                                ? 'bg-blue-50'
                                : 'hover:bg-gray-50'
                            }`}
                            onClick={() => handleAutocompleteSelect(agent)}
                            onMouseEnter={() => setAutocompleteSelectedIndex(index)}
                          >
                            <div className="font-medium text-sm">@{agentNameShort}</div>
                            <div className="text-xs text-muted-foreground truncate">
                              {agent.name}
                            </div>
                          </div>
                        );
                      } else {
                        const cmd = item as { id: string; name: string; slashCommand: string; description?: string };
                        const commandTitle = cmd.name.includes('.') 
                          ? cmd.name.split('.').pop() 
                          : cmd.name;
                        
                        return (
                          <div
                            key={cmd.id}
                            className={`px-3 py-2 cursor-pointer border-b last:border-b-0 ${
                              index === autocompleteSelectedIndex
                                ? 'bg-blue-50'
                                : 'hover:bg-gray-50'
                            }`}
                            onClick={() => handleAutocompleteSelect(cmd)}
                            onMouseEnter={() => setAutocompleteSelectedIndex(index)}
                          >
                            <div className="font-medium text-sm">{cmd.slashCommand}</div>
                            <div className="text-xs text-muted-foreground truncate capitalize">
                              {commandTitle}
                            </div>
                          </div>
                        );
                      }
                    })}
                    </>
                    )}
                  </div>
                )}
              </div>
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
                            className="max-h-[400px] overflow-y-scroll space-y-2 pr-2 scrollbar-thin"
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
                            className="max-h-[400px] overflow-y-scroll space-y-2 pr-2 scrollbar-thin"
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


