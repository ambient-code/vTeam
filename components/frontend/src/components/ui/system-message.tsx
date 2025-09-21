import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Info, ChevronDown, ChevronRight } from "lucide-react";

export type SystemMessageProps = {
  subtype: string;
  data: Record<string, any>;
  className?: string;
  borderless?: boolean;
};

export const SystemMessage: React.FC<SystemMessageProps> = ({ subtype, data, className, borderless }) => {
  const [expanded, setExpanded] = useState(false);

  const pretty = React.useMemo(() => JSON.stringify(data ?? {}, null, 2), [data]);

  return (
    <div className={cn("mb-4", className)}>
      <div className="flex items-start space-x-3">
        <div className="flex-shrink-0">
          <div className="w-8 h-8 rounded-full flex items-center justify-center bg-gray-600">
            <Info className="w-4 h-4 text-white" />
          </div>
        </div>

        <div className="flex-1 min-w-0">
          <div className={cn(borderless ? "p-0" : "bg-white rounded-lg border shadow-sm")}>
            <div className="flex items-center justify-between p-3">
              <div className="flex items-center gap-2">
                <Badge variant="secondary" className="text-xs">System</Badge>
                <span className="text-[10px] text-gray-500">{subtype || (data?.subtype as string) || "system"}</span>
              </div>
              <button
                className="text-xs text-blue-600 hover:underline flex items-center gap-1"
                onClick={() => setExpanded((e) => !e)}
                aria-expanded={expanded}
              >
                {expanded ? "Hide" : "Show"} details
                {expanded ? (
                  <ChevronDown className="w-3 h-3 text-gray-500" />
                ) : (
                  <ChevronRight className="w-3 h-3 text-gray-500" />
                )}
              </button>
            </div>

            {!expanded && (
              <div className="px-3 pb-3 text-xs text-gray-600">
                System message details hidden
              </div>
            )}

            {expanded && (
              <div className="px-3 pb-3">
                <pre className="bg-gray-50 border rounded p-2 whitespace-pre-wrap break-words text-xs text-gray-800">
                  {pretty}
                </pre>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default SystemMessage;


