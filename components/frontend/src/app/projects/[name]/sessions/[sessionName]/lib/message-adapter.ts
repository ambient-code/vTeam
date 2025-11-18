import type { SessionMessage } from "@/types";
import type { MessageObject, ToolUseMessages } from "@/types/agentic-session";
import type { RawWireMessage, InnerEnvelope, ToolUseBlockWithTimestamp, ToolResultBlockWithTimestamp } from "./types";

/**
 * Converts raw wire messages from the backend into structured MessageObject and ToolUseMessages
 * for display in the UI. This handles all the complex message parsing and transformation logic.
 */
export function adaptSessionMessages(
  messages: SessionMessage[],
  isInteractive: boolean = false
): Array<MessageObject | ToolUseMessages> {
  try {
    const toolUseBlocks: ToolUseBlockWithTimestamp[] = [];
    const toolResultBlocks: ToolResultBlockWithTimestamp[] = [];
    const agenticMessages: MessageObject[] = [];

  for (const raw of messages as RawWireMessage[]) {
    const envelope: InnerEnvelope = ((raw?.payload as InnerEnvelope) ?? (raw as unknown as InnerEnvelope)) || {};
    const innerType: string = (raw as unknown as InnerEnvelope)?.type || envelope.type || "";
    const innerTs: string = raw?.timestamp || envelope.timestamp || new Date().toISOString();
    const payloadValue = envelope.payload;
    const innerPayload: Record<string, unknown> = (payloadValue && typeof payloadValue === 'object' && !Array.isArray(payloadValue))
      ? (payloadValue as Record<string, unknown>)
      : ((typeof envelope === 'object' ? (envelope as unknown as Record<string, unknown>) : {}) as Record<string, unknown>);
    const partial = (envelope.partial as InnerEnvelope["partial"]) || ((raw as unknown as { partial?: InnerEnvelope["partial"] })?.partial) || undefined;

    switch (innerType) {
      case "message.partial": {
        const text = partial?.data || "";
        if (text) {
          agenticMessages.push({
            type: "agent_message",
            content: { type: "text_block", text },
            model: "claude",
            timestamp: innerTs,
          });
        }
        break;
      }
      case "agent.message": {
        if (partial?.data) {
          const text = String(partial.data || "");
          if (text) {
            agenticMessages.push({
              type: "agent_message",
              content: { type: "text_block", text },
              model: "claude",
              timestamp: innerTs,
            });
            break;
          }
        }

        const toolName = (innerPayload?.tool as string | undefined);
        const toolInput = (innerPayload?.input as Record<string, unknown> | undefined) || {};
        const providedId = (innerPayload?.id as string | undefined);
        const result = innerPayload?.tool_result as unknown as { tool_use_id?: string; content?: unknown; is_error?: boolean } | undefined;
        
        if (toolName) {
          const id = providedId ? String(providedId) : String(envelope?.seq ?? `${toolName}-${toolUseBlocks.length}`);
          toolUseBlocks.push({
            block: { type: "tool_use_block", id, name: toolName, input: toolInput },
            timestamp: innerTs,
          });
        } else if (result?.tool_use_id) {
          toolResultBlocks.push({
            block: {
              type: "tool_result_block",
              tool_use_id: String(result.tool_use_id),
              content: (result.content as string | Array<Record<string, unknown>> | null | undefined) ?? null,
              is_error: Boolean(result.is_error),
            },
            timestamp: innerTs,
          });
        } else if ((innerPayload as Record<string, unknown>)?.type === 'result.message') {
          let rp: Record<string, unknown> = (innerPayload.payload as Record<string, unknown>) || {};
          if (rp && typeof rp === 'object' && 'payload' in rp && rp.payload && typeof rp.payload === 'object') {
            rp = rp.payload as Record<string, unknown>;
          }
          agenticMessages.push({
            type: "result_message",
            subtype: String(rp.subtype || ""),
            duration_ms: Number(rp.duration_ms || 0),
            duration_api_ms: Number(rp.duration_api_ms || 0),
            is_error: Boolean(rp.is_error || false),
            num_turns: Number(rp.num_turns || 0),
            session_id: String(rp.session_id || ""),
            total_cost_usd: (typeof rp.total_cost_usd === 'number' ? rp.total_cost_usd : null),
            usage: (typeof rp.usage === 'object' && rp.usage ? rp.usage as Record<string, unknown> : null),
            result: (typeof rp.result === 'string' ? rp.result : null),
            timestamp: innerTs,
          });
          if (typeof rp.result === 'string' && rp.result.trim()) {
            agenticMessages.push({
              type: "agent_message",
              content: { type: "text_block", text: String(rp.result) },
              model: "claude",
              timestamp: innerTs,
            });
          }
        } else {
          const envelopePayload = envelope.payload;
          const contentText = (innerPayload.content as Record<string, unknown> | undefined)?.text;
          const messageText = innerPayload.message;
          const nestedContentText = (innerPayload.payload as Record<string, unknown> | undefined)?.content as Record<string, unknown> | undefined;
          const text = (typeof envelopePayload === 'string')
            ? String(envelopePayload)
            : (
                (typeof contentText === 'string' ? String(contentText) : undefined)
                || (typeof messageText === 'string' ? String(messageText) : undefined)
                || (typeof nestedContentText?.text === 'string' ? String(nestedContentText.text) : '')
              );
          if (text) {
            agenticMessages.push({
              type: "agent_message",
              content: { type: "text_block", text },
              model: "claude",
              timestamp: innerTs,
            });
          }
        }
        break;
      }
      case "system.message": {
        let text = "";
        let isDebug = false;
        
        // The envelope object might have message/payload at different levels
        // Try envelope.payload first, then fall back to envelope itself
        const envelopeObj = envelope as { message?: string; payload?: string | { message?: string; payload?: string; debug?: boolean }; debug?: boolean };
        
        // Check if envelope.payload is a string
        if (typeof envelopeObj.payload === 'string') {
          text = envelopeObj.payload;
        }
        // Check if envelope.payload is an object with message or payload
        else if (typeof envelopeObj.payload === 'object' && envelopeObj.payload !== null) {
          const payloadObj = envelopeObj.payload as { message?: string; payload?: string; debug?: boolean };
          text = payloadObj.message || (typeof payloadObj.payload === 'string' ? payloadObj.payload : "");
          isDebug = payloadObj.debug === true;
        }
        // Fall back to envelope.message directly
        else if (typeof envelopeObj.message === 'string') {
          text = envelopeObj.message;
        }
        
        if (envelopeObj.debug === true) {
          isDebug = true;
        }
        
        // Always create a system message - show the raw envelope if we couldn't extract text
        agenticMessages.push({
          type: "system_message",
          subtype: "system.message",
          data: { 
            message: text || `[system event: ${JSON.stringify(envelope)}]`,
            debug: isDebug 
          },
          timestamp: innerTs,
        });
        break;
      }
      case "user.message":
      case "user_message": {
        const text = (innerPayload?.content as string | undefined) || "";
        if (text) {
          agenticMessages.push({
            type: "user_message",
            content: { type: "text_block", text },
            timestamp: innerTs,
          });
        }
        break;
      }
      case "agent.running": {
        agenticMessages.push({ type: "agent_running", timestamp: innerTs });
        break;
      }
      case "agent.waiting": {
        agenticMessages.push({ type: "agent_waiting", timestamp: innerTs });
        break;
      }
      default: {
        agenticMessages.push({
          type: "system_message",
          subtype: innerType || "unknown",
          data: innerPayload || {},
          timestamp: innerTs,
        });
      }
    }
  }

  const toolUseMessages: ToolUseMessages[] = [];
  for (const tu of toolUseBlocks) {
    const match = toolResultBlocks.find((tr) => tr.block.tool_use_id === tu.block.id);
    if (match) {
      toolUseMessages.push({
        type: "tool_use_messages",
        timestamp: tu.timestamp,
        toolUseBlock: tu.block,
        resultBlock: match.block,
      });
    } else {
      toolUseMessages.push({
        type: "tool_use_messages",
        timestamp: tu.timestamp,
        toolUseBlock: tu.block,
        resultBlock: { type: "tool_result_block", tool_use_id: tu.block.id, content: null, is_error: false },
      });
    }
  }

    const all = [...agenticMessages, ...toolUseMessages];
    const sorted = all.sort((a, b) => {
      const at = new Date(a.timestamp || 0).getTime();
      const bt = new Date(b.timestamp || 0).getTime();
      return at - bt;
    });
    
    return isInteractive ? sorted.filter((m) => m.type !== "result_message") : sorted;
  } catch (error) {
    console.error('Failed to adapt session messages:', error);
    return []; // Return empty array on error
  }
}

