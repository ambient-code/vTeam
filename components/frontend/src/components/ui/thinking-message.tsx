"use client";

import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Loader2, Brain } from "lucide-react";
import type { ThinkingBlock } from "@/types/agentic-session";

export type ThinkingMessageProps = {
  block: ThinkingBlock;
  className?: string;
};

export const ThinkingMessage: React.FC<ThinkingMessageProps> = ({ block, className }) => {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className={cn("mb-4", className)}>
      <div className="flex items-start space-x-3">
        <div className="flex-shrink-0">
          <div className="w-8 h-8 rounded-full flex items-center justify-center bg-yellow-500">
            <Brain className="w-4 h-4 text-white" />
          </div>
        </div>

        <div className="flex-1 min-w-0">
          <div className="bg-card rounded-lg border shadow-sm p-3">
            <div className="flex items-center justify-between mb-2">
              <Badge variant="outline" className="text-xs">Thinking</Badge>
              <button
                className="text-xs text-link hover:underline"
                onClick={() => setExpanded((e) => !e)}
              >
                {expanded ? "Hide" : "Show"} details
              </button>
            </div>

            {!expanded && (
              <div className="flex items-center text-muted-foreground text-xs">
                <Loader2 className="w-3 h-3 mr-2 animate-spin" /> Hidden reasoning available
              </div>
            )}

            {expanded && (
              <div className="space-y-3">
                  <div className="text-xs">
                    <div className="mb-1 text-muted-foreground">
                      <span className="font-semibold">Signature:</span> {block.signature}
                    </div>
                    <pre className="bg-muted/50 border rounded p-2 whitespace-pre-wrap break-words text-foreground">
                      {block.thinking}
                    </pre>
                  </div>
                
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default ThinkingMessage;


