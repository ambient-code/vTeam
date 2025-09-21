import React from "react";
import { MessageObject, ToolUseMessages } from "@/types/agentic-session";
import { Message } from "@/components/ui/message";
import { ToolMessage } from "@/components/ui/tool-message";
import { ThinkingMessage } from "@/components/ui/thinking-message";
import { SystemMessage } from "@/components/ui/system-message";
import { ResultMessage } from "@/components/ui/result-message";

export type StreamMessageProps = {
  message: MessageObject | ToolUseMessages;
};


export const StreamMessage: React.FC<StreamMessageProps> = ({ message }) => {
  const isToolUsePair = (m: MessageObject | ToolUseMessages): m is ToolUseMessages =>
    m != null && typeof m === "object" && "toolUseBlock" in m && "resultBlock" in m;

  if (isToolUsePair(message)) {
    return <ToolMessage toolUseBlock={message.toolUseBlock} resultBlock={message.resultBlock} />;
  }

  const m = message as MessageObject;
  switch (m.type) {
    case "user_message":
    case "assistant_message": {
      if (typeof m.content === "string") {
        return <Message role={m.type === "assistant_message" ? "bot" : "user"} content={m.content} name="Claude AI" />;
      }
      // Thinking (new): show above, expandable
      switch (m.content.type) {
        case "thinking_block":
          return <ThinkingMessage block={m.content} />
        case "text_block":
          return <Message role={m.type === "assistant_message" ? "bot" : "user"} content={m.content.text} name="Claude AI" />
        case "tool_use_block":
          return <ToolMessage toolUseBlock={m.content} />
        case "tool_result_block":
          return <ToolMessage resultBlock={m.content} />
      }
    }
    case "system_message": {
      return <SystemMessage subtype={m.subtype} data={m.data} />;
    }
    case "result_message": {
      return (
        <ResultMessage
          duration_ms={m.duration_ms}
          duration_api_ms={m.duration_api_ms}
          is_error={m.is_error}
          num_turns={m.num_turns}
          session_id={m.session_id}
          total_cost_usd={m.total_cost_usd}
          usage={m.usage as any}
          result={m.result ?? undefined}
        />
      );
    }
    default:
      return null;
  }
};

export default StreamMessage;


