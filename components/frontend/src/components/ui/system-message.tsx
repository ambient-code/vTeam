import React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Info } from "lucide-react";

export type SystemMessageProps = {
  subtype: string;
  data: Record<string, any>;
  className?: string;
  borderless?: boolean;
};

export const SystemMessage: React.FC<SystemMessageProps> = ({ subtype, data, className, borderless }) => {
  // Expect a simple string in data.message; fallback to JSON.stringify
  const text: string = typeof (data?.message) === 'string' ? data.message : (typeof data === 'string' ? data : JSON.stringify(data ?? {}, null, 2));

  return (
    <div className={cn("mb-4", className)}>
      <div className="flex items-start space-x-3">
        <div className="flex-shrink-0">
          <div className="w-8 h-8 rounded-full flex items-center justify-center bg-gray-600">
            <Info className="w-4 h-4 text-white" />
          </div>
        </div>

        <div className="flex-1 min-w-0">
          <div className={cn(borderless ? "p-0" : "bg-white rounded-lg border shadow-sm p-3")}> 
            <div className="flex items-center gap-2 mb-1">
              <Badge variant="secondary" className="text-xs">System</Badge>
              <span className="text-[10px] text-gray-500">{subtype || "system.message"}</span>
            </div>
            <div className="text-xs text-gray-700 whitespace-pre-wrap break-words">{text}</div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SystemMessage;


