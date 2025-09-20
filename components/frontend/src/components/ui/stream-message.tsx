import React from "react";
import { MessageObject, ContentBlock } from "@/types/agentic-session";
import { Message } from "@/components/ui/message";
import { ToolMessage } from "@/components/ui/tool-message";
import { ThinkingMessage } from "@/components/ui/thinking-message";
import { SystemMessage } from "@/components/ui/system-message";
import { ResultMessage } from "@/components/ui/result-message";

export type StreamMessageProps = {
  message: MessageObject;
};

const hasToolBlocks = (blocks: ContentBlock[] | undefined) =>
  Array.isArray(blocks) && blocks.some((b) => b.type === "tool_use_block" || b.type === "tool_result_block");

const getTextFromAssistant = (blocks: ContentBlock[] | undefined): string => {
  if (!Array.isArray(blocks)) return "";
  const tb = blocks.find((b) => b.type === "text_block") as Extract<ContentBlock, { type: "text_block" }> | undefined;
  return tb?.text || "";
};

export const StreamMessage: React.FC<StreamMessageProps> = ({ message }) => {
  switch (message.type) {
    case "user_message": {
      const text = typeof message.content === "string" ? message.content : "";
      return <Message role="user" content={text} name="You" />;
    }
    case "assistant_message": {
      const blocks = message.content;
      // Thinking (new): show above, expandable
      if (Array.isArray(blocks) && blocks.some((b) => b.type === "thinking_block")) {
        return (
          <>
            <ThinkingMessage blocks={blocks} />
            {hasToolBlocks(blocks) ? (
              <ToolMessage message={message as any} />
            ) : (
              <Message role="bot" content={getTextFromAssistant(blocks)} name="Claude AI" />
            )}
          </>
        );
      }
      // Tool use/result
      if (hasToolBlocks(blocks)) {
        return <ToolMessage message={message as any} />;
      }
      // Plain text
      return <Message role="bot" content={getTextFromAssistant(blocks)} name="Claude AI" />;
    }
    case "system_message": {
      return <SystemMessage subtype={message.subtype} data={message.data} />;
    }
    case "result_message": {
      return (
        <ResultMessage
          duration_ms={message.duration_ms}
          duration_api_ms={message.duration_api_ms}
          is_error={message.is_error}
          num_turns={message.num_turns}
          session_id={message.session_id}
          total_cost_usd={message.total_cost_usd}
          usage={message.usage as any}
          result={message.result ?? undefined}
        />
      );
    }
    default:
      return null;
  }
};

export default StreamMessage;


