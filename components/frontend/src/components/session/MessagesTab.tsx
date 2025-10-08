"use client";

import React from "react";
import { Button } from "@/components/ui/button";
import { Brain } from "lucide-react";
import { StreamMessage } from "@/components/ui/stream-message";
import type { AgenticSession } from "@/types/agentic-session";

export type MessagesTabProps = {
  session: AgenticSession;
  streamMessages: any[];
  chatInput: string;
  setChatInput: (v: string) => void;
  onSendChat: () => Promise<void>;
  onInterrupt: () => Promise<void>;
  onEndSession: () => Promise<void>;
  onGoToResults?: () => void;
};

const MessagesTab: React.FC<MessagesTabProps> = ({ session, streamMessages, chatInput, setChatInput, onSendChat, onInterrupt, onEndSession, onGoToResults }) => {
  return (
    <div className="flex flex-col gap-2 max-h-[60vh] overflow-y-auto pr-1">
      {streamMessages.map((m, idx) => (
        <StreamMessage key={`sm-${idx}`} message={m} isNewest={idx === streamMessages.length - 1} onGoToResults={onGoToResults} />
      ))}

      {streamMessages.length === 0 && (
        <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
          <Brain className="w-8 h-8 mx-auto mb-2 opacity-50" />
          <p className="text-sm">No messages yet</p>
          <p className="text-xs mt-1">
            {session.spec?.interactive ? "Start by sending a message below." : "This session is not interactive."}
          </p>
        </div>
      )}

      {session.spec?.interactive && (
        <div className="sticky bottom-0 border-t bg-white">
          <div className="p-3">
            <div className="border rounded-md p-3 space-y-2 bg-white">
              <textarea
                className="w-full border rounded p-2 text-sm"
                placeholder="Type a message to the agent..."
                value={chatInput}
                onChange={(e) => setChatInput(e.target.value)}
                rows={3}
              />
              <div className="flex items-center justify-between">
                <div className="text-xs text-muted-foreground">Interactive session</div>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" onClick={onInterrupt}>Interrupt agent</Button>
                  <Button variant="secondary" size="sm" onClick={onEndSession}>End session</Button>
                  <Button size="sm" onClick={onSendChat} disabled={!chatInput.trim()}>Send</Button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default MessagesTab;


