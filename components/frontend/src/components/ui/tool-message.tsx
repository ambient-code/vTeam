"use client";

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
import type { Components } from "react-markdown";
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

type ExpandableMarkdownProps = {
  content: string;
  maxLength?: number;
  className?: string;
};

const ExpandableMarkdown: React.FC<ExpandableMarkdownProps> = ({
  content,
  maxLength = 2000,
  className,
}) => {
  const [expanded, setExpanded] = useState(false);
  const shouldTruncate = content.length > maxLength;
  const display = expanded || !shouldTruncate ? content : content.substring(0, maxLength);

  // Match Message.tsx rendering so headers/code look correct
  const markdownComponents: Components = {
    code: ({
      inline,
      className,
      children,
      ...props
    }: {
      inline?: boolean;
      className?: string;
      children?: React.ReactNode;
    } & React.HTMLAttributes<HTMLElement>) => {
      return inline ? (
        <code className="bg-gray-100 px-1 py-0.5 rounded text-xs" {...(props as React.HTMLAttributes<HTMLElement>)}>
          {children}
        </code>
      ) : (
        <pre className="bg-gray-800 text-gray-100 p-2 rounded text-xs overflow-x-auto">
          <code className={className} {...(props as React.HTMLAttributes<HTMLElement>)}>
            {children}
          </code>
        </pre>
      );
    },
    p: ({ children }) => <p className="text-gray-600 leading-relaxed mb-2 text-sm">{children}</p>,
    h1: ({ children }) => <h1 className="text-lg font-bold text-gray-800 mb-2">{children}</h1>,
    h2: ({ children }) => <h2 className="text-md font-semibold text-gray-800 mb-2">{children}</h2>,
    h3: ({ children }) => <h3 className="text-sm font-medium text-gray-800 mb-1">{children}</h3>,
  };

  return (
    <div className={cn("max-w-none", className)}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
        {display}
      </ReactMarkdown>
      {shouldTruncate && (
        <div className="mt-2">
          <button
            type="button"
            onClick={() => setExpanded(!expanded)}
            className="text-xs px-2 py-1 rounded border bg-white hover:bg-gray-50 text-gray-700"
          >
            {expanded ? "Show less" : "Show more"}
          </button>
        </div>
      )}
    </div>
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
    
    // Compact mode for simple tool calls (non-subagent)
    const isCompact = !isSubagent;

    return (
      <div ref={ref} className={cn(isCompact ? "mb-1" : "mb-4", className)} {...props}>
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
              <div className="w-6 h-6 rounded-full flex items-center justify-center bg-purple-600">
                <Cog className="w-3 h-3 text-white" />
              </div>
            )}
          </div>

          {/* Tool Message Content */}
          <div className="flex-1 min-w-0">
            <div
              className={cn(
                isCompact ? "" : (borderless ? "p-0" : "rounded-lg border shadow-sm"),
                isSubagent ? subagentClasses?.cardBg : "",
                isSubagent ? subagentClasses?.border : undefined
              )}
            >
              {/* Collapsible Header */}
              <div
                className={cn(
                  "flex items-center justify-between cursor-pointer hover:bg-gray-50 transition-colors",
                  isCompact ? "py-1 px-0" : "p-3"
                )}
                onClick={() => setIsExpanded(!isExpanded)}
              >
                <div className={cn("flex items-center", isCompact ? "space-x-1.5" : "space-x-2")}>
                  {/* Status Icon */}
                  {!isCompact && (
                    <div className="flex-shrink-0">
                      {isLoading && (
                        <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />
                      )}
                      {isSuccess && <Check className="w-4 h-4 text-green-500" />}
                      {isError && <X className="w-4 h-4 text-red-500" />}
                    </div>
                  )}
                  {isCompact && (
                    <div className="flex-shrink-0">
                      {isLoading && (
                        <Loader2 className="w-3 h-3 text-blue-500 animate-spin" />
                      )}
                      {isError && <X className="w-3 h-3 text-red-500" />}
                    </div>
                  )}

                  {/* Tool Name */}
                  <div className="flex-1 flex items-center min-h-0">
                    <Badge
                      variant="outline"
                      className={cn(
                        "text-xs",
                        isLoading && "animate-pulse",
                        isError && "border-red-200 text-red-700",
                        isSuccess && "border-green-200 text-green-700",
                        isSubagent && subagentClasses?.badgeBorder,
                        isSubagent && subagentClasses?.badgeText,
                        isCompact && "!py-0 px-1.5 leading-tight"
                      )}
                    >
                      {isSubagent ? displayName : (isLoading ? "Calling" : "Called") + " " + displayName}
                    </Badge>
                  </div>

                  {/* Expand/Collapse Icon */}
                  <div className="flex-shrink-0">
                    {isExpanded ? (
                      <ChevronDown className={cn(isCompact ? "w-3 h-3" : "w-4 h-4", "text-gray-400")} />
                    ) : (
                      <ChevronRight className={cn(isCompact ? "w-3 h-3" : "w-4 h-4", "text-gray-400")} />
                    )}
                  </div>
                </div>
              </div>

              {/* Subagent primary content (description + prompt) */}
              {isSubagent ? (
                <div className="px-3 pb-3 space-y-3">
                  <div>
                    {subagentDescription && subagentDescription.trim() ? (
                      <div className="text-gray-800">
                        <ExpandableMarkdown className="prose-sm" content={subagentDescription} />
                      </div>
                    ) : isLoading ? (
                      <div className="text-gray-500 text-sm italic">
                        Working on your request...
                      </div>
                    ) : null}
                    
                    {isLoading && subagentDescription && subagentDescription.trim() && (
                      <div className="flex items-center gap-2 text-xs text-gray-500 mt-2">
                        <Loader2 className="w-3 h-3 animate-bounce" />
                        <span>Waiting for result…</span>
                      </div>
                    )}
                  </div>

                  {isExpanded && subagentPrompt && (
                    <div>
                      <h4 className="text-xs font-medium text-gray-700 mb-1">Prompt</h4>
                      <div className="rounded p-2 overflow-x-auto">
                        <ExpandableMarkdown className="prose-sm" content={subagentPrompt} />
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
                            "rounded p-2 overflow-x-auto text-gray-800",
                            isError && "bg-red-50 border border-red-200"
                          )}
                        >
                          <ExpandableMarkdown
                            className="prose-sm"
                            content={
                              typeof toolResultBlock?.content === "string"
                                ? (toolResultBlock?.content as string)
                                : JSON.stringify(toolResultBlock?.content ?? "")
                            }
                          />
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
                  <div className={cn("rounded p-2 overflow-x-auto text-gray-800")}>
                    <ExpandableMarkdown className="prose-sm" content={extractTextFromResultContent(toolResultBlock?.content as unknown)} />
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