"use client";

import React from "react";
import { MessageObject, ToolUseMessages } from "@/types/agentic-session";
import { LoadingDots, Message } from "@/components/ui/message";
import { ToolMessage } from "@/components/ui/tool-message";
import { ThinkingMessage } from "@/components/ui/thinking-message";
import { SystemMessage } from "@/components/ui/system-message";
import { Button } from "@/components/ui/button";

export type StreamMessageProps = {
  message: MessageObject | ToolUseMessages;
  onGoToResults?: () => void;
  plainCard?: boolean;
  isNewest?: boolean;
};


export const StreamMessage: React.FC<StreamMessageProps> = ({ message, onGoToResults, plainCard=false, isNewest=false }) => {
  const isToolUsePair = (m: MessageObject | ToolUseMessages): m is ToolUseMessages =>
    m != null && typeof m === "object" && "toolUseBlock" in m && "resultBlock" in m;

  if (isToolUsePair(message)) {
    return <ToolMessage toolUseBlock={message.toolUseBlock} resultBlock={message.resultBlock} />;
  }

  const m = message as MessageObject;
  switch (m.type) {
    case "agent_running": {
      if (!isNewest) return null;
      return <LoadingDots />;
    }
    case "agent_waiting": {
      if (!isNewest) return null;
      return (
        <span className="text-xs text-gray-500">Waiting for input...</span>
      )
    }
    case "user_message":
    case "agent_message": {
      if (typeof m.content === "string") {
        return <Message role={m.type === "agent_message" ? "bot" : "user"} content={m.content} name="Claude AI" borderless={plainCard}/>;
      }
      switch (m.content.type) {
        case "thinking_block":
          return <ThinkingMessage block={m.content} />
        case "text_block":
          return <Message role={m.type === "agent_message" ? "bot" : "user"} content={m.content.text} name="Claude AI" borderless={plainCard}/>
        case "tool_use_block":
          return <ToolMessage toolUseBlock={m.content} borderless={plainCard}/>
        case "tool_result_block":
          return <ToolMessage resultBlock={m.content} borderless={plainCard}/>
      }
    }
    case "system_message": {
      return <SystemMessage subtype={m.subtype} data={m.data} borderless={plainCard}/>;
    }
    case "result_message": {
      // Show a minimal message with an action to open full results tab
      return (
        <Message
          borderless={plainCard}
          role="bot"
          content={m.is_error ? "Agent completed with errors." : "Agent completed successfully."}
          name="Claude AI"
          actions={
            <div className="flex items-center justify-between">
              <div className="text-xs text-gray-500">
                Duration: {m.duration_ms} ms • API: {m.duration_api_ms} ms • Turns: {m.num_turns}
              </div>
              <Button variant='link' size="sm" className="ml-3" onClick={onGoToResults}>Go to Results</Button>
            </div>
          }
        />
      );
    }
    default:
      return null;
  }
};

export default StreamMessage;


