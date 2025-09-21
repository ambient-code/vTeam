import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { CheckCircle2, XCircle, ChevronRight, ChevronDown } from "lucide-react";

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
  borderless?: boolean;
  defaultUsageExpanded?: boolean;
  defaultResultExpanded?: boolean;
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
    borderless,
    defaultUsageExpanded,
    defaultResultExpanded,
  } = props;

  const [usageExpanded, setUsageExpanded] = useState(!!defaultUsageExpanded);
  const [resultExpanded, setResultExpanded] = useState(!!defaultResultExpanded);

  return (
    <div className={cn("mb-4", className)}>
      <div className={cn(borderless ? "p-0" : "bg-white rounded-lg border shadow-sm p-3")}> 
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
            <div className="flex items-center justify-between mb-1">
              <div className="text-[11px] text-gray-500">Usage</div>
              <button
                className="text-xs text-blue-600 hover:underline inline-flex items-center gap-1"
                onClick={() => setUsageExpanded((e) => !e)}
                aria-expanded={usageExpanded}
              >
                {usageExpanded ? "Hide" : "Show"} details
                {usageExpanded ? (
                  <ChevronDown className="w-3 h-3 text-gray-500" />
                ) : (
                  <ChevronRight className="w-3 h-3 text-gray-500" />
                )}
              </button>
            </div>

            {!usageExpanded && (
              <div className="text-xs text-gray-600">Usage details hidden</div>
            )}

            {usageExpanded && (
              <pre className="bg-gray-50 border rounded p-2 whitespace-pre-wrap break-words text-xs text-gray-800">
                {JSON.stringify(usage, null, 2)}
              </pre>
            )}
          </div>
        )}

        {result && (
          <div className="mt-2">
            <div className="flex items-center justify-between mb-1">
              <div className="text-[11px] text-gray-500">Result</div>
              <button
                className="text-xs text-blue-600 hover:underline inline-flex items-center gap-1"
                onClick={() => setResultExpanded((e) => !e)}
                aria-expanded={resultExpanded}
              >
                {resultExpanded ? "Hide" : "Show"} details
                {resultExpanded ? (
                  <ChevronDown className="w-3 h-3 text-gray-500" />
                ) : (
                  <ChevronRight className="w-3 h-3 text-gray-500" />
                )}
              </button>
            </div>

            {!resultExpanded && (
              <div className="text-xs text-gray-600">Result details hidden</div>
            )}

            {resultExpanded && (
              <pre className="bg-gray-50 border rounded p-2 whitespace-pre-wrap break-words text-xs text-gray-800">
                {result}
              </pre>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default ResultMessage;


