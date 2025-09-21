import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { ToolResultBlock, ToolUseBlock } from "@/types/agentic-session";
import {
  ChevronDown,
  ChevronRight,
  Loader2,
  Check,
  X,
  Cog,
} from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

export type ToolMessageProps = {
  toolUseBlock?: ToolUseBlock;
  resultBlock?: ToolResultBlock;
  className?: string;
  borderless?: boolean;
};

const formatToolName = (toolName?: string) => {
  if (!toolName) return "Unknown Tool";
  // Remove mcp__ prefix and format nicely
  return toolName
    .replace(/^mcp__/, "")
    .replace(/_/g, " ")
    .split(" ")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
};

const formatToolInput = (input?: string) => {
  if (!input) return "{}";
  try {
    const parsed = JSON.parse(input);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return input;
  }
};

const truncateContent = (content: string, maxLength = 2000) => {
  if (content.length <= maxLength) return content;
  return (
    content.substring(0, maxLength) +
    "\n\n... [Content truncated - expand to view full result]"
  );
};

export const ToolMessage = React.forwardRef<HTMLDivElement, ToolMessageProps>(
  ({ toolUseBlock, resultBlock, className, borderless, ...props }, ref) => {
    const [isExpanded, setIsExpanded] = useState(false);

    const toolResultBlock = resultBlock;
    const isToolCall = Boolean(toolUseBlock && !toolResultBlock);
    const isToolResult = Boolean(toolResultBlock);

    // For tool calls/results, show collapsible interface
    const toolName = formatToolName(toolUseBlock?.name);
    const isLoading = isToolCall; // Tool call without result is loading
    const isError = toolResultBlock?.is_error === true;
    const isSuccess = isToolResult && !isError;

    return (
      <div ref={ref} className={cn("mb-4", className)} {...props}>
        <div className="flex items-start space-x-3">
          {/* Avatar */}
          <div className="flex-shrink-0">
            <div className="w-8 h-8 rounded-full flex items-center justify-center bg-purple-600">
              <Cog className="w-4 h-4 text-white" />
            </div>
          </div>

          {/* Tool Message Content */}
          <div className="flex-1 min-w-0">
            <div className={cn(borderless ? "p-0" : "bg-white rounded-lg border shadow-sm")}>
              {/* Collapsible Header */}
              <div
                className="flex items-center justify-between p-3 cursor-pointer hover:bg-gray-50 transition-colors"
                onClick={() => setIsExpanded(!isExpanded)}
              >
                <div className="flex items-center space-x-2">
                  {/* Status Icon */}
                  <div className="flex-shrink-0">
                    {isLoading && (
                      <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />
                    )}
                    {isSuccess && <Check className="w-4 h-4 text-green-500" />}
                    {isError && <X className="w-4 h-4 text-red-500" />}
                  </div>

                  {/* Tool Name */}
                  <div className="flex-1">
                    <Badge
                      variant="outline"
                      className={cn(
                        "text-xs",
                        isLoading && "animate-pulse",
                        isError && "border-red-200 text-red-700",
                        isSuccess && "border-green-200 text-green-700"
                      )}
                    >
                      {isLoading ? "Calling" : "Called"} {toolName}
                    </Badge>
                  </div>

                  {/* Expand/Collapse Icon */}
                  <div className="flex-shrink-0">
                    {isExpanded ? (
                      <ChevronDown className="w-4 h-4 text-gray-400" />
                    ) : (
                      <ChevronRight className="w-4 h-4 text-gray-400" />
                    )}
                  </div>
                </div>
              </div>

              {/* Expandable Content */}
              {isExpanded && (
                <div className="px-3 pb-3 space-y-3 bg-gray-50">
                  {/* Tool Input */}
                  {toolUseBlock?.input && (
                    <div>
                      <h4 className="text-xs font-medium text-gray-700 mb-1">
                        Input
                      </h4>
                      <div className="bg-gray-800 rounded text-xs p-2 overflow-x-auto">
                        <pre className="text-gray-100">
                          {formatToolInput(JSON.stringify(toolUseBlock.input))}
                        </pre>
                      </div>
                    </div>
                  )}

                  {/* Tool Result */}
                  {isToolResult && (
                    <div>
                      <h4 className="text-xs font-medium text-gray-700 mb-1">
                        Result{" "}
                        {isError && (
                          <span className="text-red-600">(Error)</span>
                        )}
                      </h4>
                      <div
                        className={cn(
                          "rounded p-2 text-xs overflow-x-auto text-gray-800",
                          isError
                            ? "bg-red-50 border border-red-200"
                            : "bg-white border"
                        )}
                      >
                        <ReactMarkdown remarkPlugins={[remarkGfm]}>
                          {truncateContent(
                            typeof toolResultBlock?.content === "string"
                              ? toolResultBlock.content
                              : JSON.stringify(toolResultBlock?.content ?? "")
                          )}
                        </ReactMarkdown>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    );
  }
);

ToolMessage.displayName = "ToolMessage";