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

// Helpers for Subagent rendering
const getInitials = (name?: string) => {
  if (!name) return "?";
  const parts = name.trim().split(/\s+/);
  if (parts.length === 1) return parts[0].charAt(0).toUpperCase();
  return (parts[0].charAt(0) + parts[parts.length - 1].charAt(0)).toUpperCase();
};

const hashStringToNumber = (str: string) => {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = (hash << 5) - hash + str.charCodeAt(i);
    hash |= 0; // Convert to 32bit integer
  }
  return Math.abs(hash);
};

const getColorClassesForName = (name: string) => {
  const colorChoices = [
    { avatarBg: "bg-purple-600", cardBg: "bg-purple-50", border: "border-purple-200", badgeText: "text-purple-700", badgeBorder: "border-purple-200" },
    { avatarBg: "bg-blue-600", cardBg: "bg-blue-50", border: "border-blue-200", badgeText: "text-blue-700", badgeBorder: "border-blue-200" },
    { avatarBg: "bg-emerald-600", cardBg: "bg-emerald-50", border: "border-emerald-200", badgeText: "text-emerald-700", badgeBorder: "border-emerald-200" },
    { avatarBg: "bg-teal-600", cardBg: "bg-teal-50", border: "border-teal-200", badgeText: "text-teal-700", badgeBorder: "border-teal-200" },
    { avatarBg: "bg-cyan-600", cardBg: "bg-cyan-50", border: "border-cyan-200", badgeText: "text-cyan-700", badgeBorder: "border-cyan-200" },
    { avatarBg: "bg-sky-600", cardBg: "bg-sky-50", border: "border-sky-200", badgeText: "text-sky-700", badgeBorder: "border-sky-200" },
    { avatarBg: "bg-indigo-600", cardBg: "bg-indigo-50", border: "border-indigo-200", badgeText: "text-indigo-700", badgeBorder: "border-indigo-200" },
    { avatarBg: "bg-fuchsia-600", cardBg: "bg-fuchsia-50", border: "border-fuchsia-200", badgeText: "text-fuchsia-700", badgeBorder: "border-fuchsia-200" },
    { avatarBg: "bg-rose-600", cardBg: "bg-rose-50", border: "border-rose-200", badgeText: "text-rose-700", badgeBorder: "border-rose-200" },
    { avatarBg: "bg-amber-600", cardBg: "bg-amber-50", border: "border-amber-200", badgeText: "text-amber-700", badgeBorder: "border-amber-200" },
  ];
  const idx = hashStringToNumber(name) % colorChoices.length;
  return colorChoices[idx];
};

const extractTextFromResultContent = (content: unknown): string => {
  try {
    if (typeof content === "string") return content;
    if (Array.isArray(content)) {
      const texts = content
        .map((item) => {
          if (item && typeof item === "object" && "text" in (item as Record<string, unknown>)) {
            return String((item as Record<string, unknown>).text ?? "");
          }
          return "";
        })
        .filter(Boolean);
      if (texts.length) return texts.join("\n\n");
    }
    if (content && typeof content === "object") {
      // Some schemas nest under content: []
      const maybe = (content as Record<string, unknown>).content;
      if (Array.isArray(maybe)) {
        const texts = maybe
          .map((item) => {
            if (item && typeof item === "object" && "text" in (item as Record<string, unknown>)) {
              return String((item as Record<string, unknown>).text ?? "");
            }
            return "";
          })
          .filter(Boolean);
        if (texts.length) return texts.join("\n\n");
      }
    }
    return JSON.stringify(content ?? "");
  } catch {
    return String(content ?? "");
  }
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

    // Subagent detection and data
    const inputData = (toolUseBlock?.input ?? undefined) as unknown as Record<string, unknown> | undefined;
    const subagentType = (inputData?.subagent_type as string) || undefined;
    const subagentDescription = (inputData?.description as string) || undefined;
    const subagentPrompt = (inputData?.prompt as string) || undefined;
    const isSubagent = Boolean(subagentType);
    const subagentClasses = subagentType ? getColorClassesForName(subagentType) : undefined;
    const displayName = isSubagent ? subagentType : toolName;

    return (
      <div ref={ref} className={cn("mb-4", className)} {...props}>
        <div className="flex items-start space-x-3">
          {/* Avatar */}
          <div className="flex-shrink-0">
            {isSubagent ? (
              <div className={cn("w-8 h-8 rounded-full flex items-center justify-center", subagentClasses?.avatarBg)}>
                <span className="text-white text-xs font-semibold">
                  {getInitials(subagentType)}
                </span>
              </div>
            ) : (
              <div className="w-8 h-8 rounded-full flex items-center justify-center bg-purple-600">
                <Cog className="w-4 h-4 text-white" />
              </div>
            )}
          </div>

          {/* Tool Message Content */}
          <div className="flex-1 min-w-0">
            <div
              className={cn(
                borderless ? "p-0" : "rounded-lg border shadow-sm",
                isSubagent ? subagentClasses?.cardBg : "bg-white",
                isSubagent ? subagentClasses?.border : undefined
              )}
            >
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
                        isSuccess && "border-green-200 text-green-700",
                        isSubagent && subagentClasses?.badgeBorder,
                        isSubagent && subagentClasses?.badgeText
                      )}
                    >
                      {isSubagent ? displayName : (isLoading ? "Calling" : "Called") + " " + displayName}
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

              {/* Subagent primary content (description + prompt) */}
              {isSubagent ? (
                <div className="px-3 pb-3 space-y-3">
                  {subagentDescription && (
                    <div>
                      <div className="text-sm text-gray-800">
                        <ReactMarkdown remarkPlugins={[remarkGfm]}>
                          {truncateContent(subagentDescription)}
                        </ReactMarkdown>
                      </div>
                      {isLoading && (
                        <div className="flex items-center gap-2 text-xs text-gray-500 mt-2">
                          <Loader2 className="w-3 h-3 animate-bounce" />
                          <span>Waiting for result…</span>
                        </div>
                      )}
                    </div>
                  )}

                  {isExpanded && subagentPrompt && (
                    <div>
                      <h4 className="text-xs font-medium text-gray-700 mb-1">Prompt</h4>
                      <div className="bg-white rounded text-xs p-2 overflow-x-auto border">
                        <ReactMarkdown remarkPlugins={[remarkGfm]}>
                          {truncateContent(subagentPrompt)}
                        </ReactMarkdown>
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                // Default tool rendering (existing behavior)
                isExpanded && (
                  <div className="px-3 pb-3 space-y-3 bg-gray-50">
                    {toolUseBlock?.input && (
                      <div>
                        <h4 className="text-xs font-medium text-gray-700 mb-1">Input</h4>
                        <div className="bg-gray-800 rounded text-xs p-2 overflow-x-auto">
                          <pre className="text-gray-100">
                            {formatToolInput(JSON.stringify(toolUseBlock.input))}
                          </pre>
                        </div>
                      </div>
                    )}

                    {isToolResult && (
                      <div>
                        <h4 className="text-xs font-medium text-gray-700 mb-1">
                          Result {isError && <span className="text-red-600">(Error)</span>}
                        </h4>
                        <div
                          className={cn(
                            "rounded p-2 text-xs overflow-x-auto text-gray-800",
                            isError ? "bg-red-50 border border-red-200" : "bg-white border"
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
                )
              )}
            </div>

            {/* Subagent Result Card (separate) */}
            {isSubagent && isToolResult && (
              <div
                className={cn(
                  "mt-2 rounded-lg border shadow-sm",
                  subagentClasses?.cardBg,
                  subagentClasses?.border
                )}
              >
                <div className="flex items-center justify-between p-3">
                  <div className="flex items-center space-x-2">
                    <div className="flex-shrink-0">
                      {isSuccess && <Check className="w-4 h-4 text-green-500" />}
                      {isError && <X className="w-4 h-4 text-red-500" />}
                    </div>
                    <div className="flex-1">
                      <Badge
                        variant="outline"
                        className={cn("text-xs", subagentClasses?.badgeBorder, subagentClasses?.badgeText)}
                      >
                        {displayName}
                      </Badge>
                    </div>
                  </div>
                </div>
                <div className="px-3 pb-3">
                  <div className={cn("rounded p-2 text-sm overflow-x-auto text-gray-800 bg-white border")}>
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>
                      {truncateContent(extractTextFromResultContent(toolResultBlock?.content as unknown))}
                    </ReactMarkdown>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }
);

ToolMessage.displayName = "ToolMessage";