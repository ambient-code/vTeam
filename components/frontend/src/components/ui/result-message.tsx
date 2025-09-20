import React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { CheckCircle2, XCircle } from "lucide-react";

export type ResultMessageProps = {
  duration_ms: number;
  duration_api_ms: number;
  is_error: boolean;
  num_turns: number;
  session_id: string;
  total_cost_usd?: number | null;
  usage?: Record<string, any> | null;
  result?: string | null;
  className?: string;
};

export const ResultMessage: React.FC<ResultMessageProps> = (props) => {
  const {
    duration_ms,
    duration_api_ms,
    is_error,
    num_turns,
    session_id,
    total_cost_usd,
    usage,
    result,
    className,
  } = props;

  return (
    <div className={cn("mb-4", className)}>
      <div className="bg-white rounded-lg border shadow-sm p-3">
        <div className="flex items-center justify-between mb-2">
          <Badge variant={is_error ? "destructive" : "secondary"} className="text-xs">
            {is_error ? (
              <span className="inline-flex items-center"><XCircle className="w-3 h-3 mr-1" /> Error</span>
            ) : (
              <span className="inline-flex items-center"><CheckCircle2 className="w-3 h-3 mr-1" /> Success</span>
            )}
          </Badge>
          <span className="text-[10px] text-gray-500">{session_id}</span>
        </div>

        <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs text-gray-700">
          <div><span className="font-medium">Duration:</span> {duration_ms} ms</div>
          <div><span className="font-medium">API:</span> {duration_api_ms} ms</div>
          <div><span className="font-medium">Turns:</span> {num_turns}</div>
          {typeof total_cost_usd === "number" && <div><span className="font-medium">Cost:</span> ${total_cost_usd.toFixed(4)}</div>}
        </div>

        {usage && (
          <div className="mt-2">
            <div className="text-[11px] text-gray-500 mb-1">Usage</div>
            <pre className="bg-gray-50 border rounded p-2 whitespace-pre-wrap break-words text-xs text-gray-800">
              {JSON.stringify(usage, null, 2)}
            </pre>
          </div>
        )}

        {result && (
          <div className="mt-2">
            <div className="text-[11px] text-gray-500 mb-1">Result</div>
            <pre className="bg-gray-50 border rounded p-2 whitespace-pre-wrap break-words text-xs text-gray-800">
              {result}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
};

export default ResultMessage;


