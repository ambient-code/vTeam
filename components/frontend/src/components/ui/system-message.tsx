import React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Info } from "lucide-react";

export type SystemMessageProps = {
  subtype: string;
  data: Record<string, any>;
  className?: string;
};

export const SystemMessage: React.FC<SystemMessageProps> = ({ subtype, data, className }) => {
  return (
    <div className={cn("mb-4", className)}>
      <div className="flex items-start space-x-3">
        <div className="flex-shrink-0">
          <div className="w-8 h-8 rounded-full flex items-center justify-center bg-gray-600">
            <Info className="w-4 h-4 text-white" />
          </div>
        </div>

        <div className="flex-1 min-w-0">
          <div className="bg-white rounded-lg border shadow-sm p-3">
            <div className="flex items-center justify-between mb-2">
              <Badge variant="secondary" className="text-xs">System</Badge>
              <span className="text-[10px] text-gray-500">{subtype || (data?.subtype as string) || "system"}</span>
            </div>

            <pre className="bg-gray-50 border rounded p-2 whitespace-pre-wrap break-words text-xs text-gray-800">
              {JSON.stringify(data ?? {}, null, 2)}
            </pre>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SystemMessage;


