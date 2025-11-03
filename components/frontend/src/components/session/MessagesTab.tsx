"use client";

import React, { useState } from "react";
import { Button } from "@/components/ui/button";
import { Brain, Loader2, Settings } from "lucide-react";
import { StreamMessage } from "@/components/ui/stream-message";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuCheckboxItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { AgenticSession, MessageObject, ToolUseMessages } from "@/types/agentic-session";

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
};


const MessagesTab: React.FC<MessagesTabProps> = ({ session, streamMessages, chatInput, setChatInput, onSendChat, onInterrupt, onEndSession, onGoToResults, onContinue}) => {
  const [sendingChat, setSendingChat] = useState(false);
  const [interrupting, setInterrupting] = useState(false);
  const [ending, setEnding] = useState(false);
  const [showSystemMessages, setShowSystemMessages] = useState(false);

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
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-2 max-h-[60vh] overflow-y-auto pr-1">
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
                  <div className="text-xs text-muted-foreground">Interactive session</div>
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
                            Continue this session
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


