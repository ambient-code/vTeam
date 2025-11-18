"use client";

import React from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type ResultMeta = {
  subtype?: string;
  duration_ms?: number;
  duration_api_ms?: number;
  is_error?: boolean;
  num_turns?: number;
  session_id?: string;
  total_cost_usd?: number | null;
  usage?: Record<string, unknown> | null;
};

type Props = {
  result?: string | null;
  meta?: ResultMeta | null;
  components?: Record<string, React.ComponentType<unknown>>;
};

const ResultsTab: React.FC<Props> = ({ result, meta, components }) => {
  if (!result && !meta) return <div className="text-sm text-muted-foreground">No artifacts yet</div>;
  return (
    <Card>
      <CardHeader>
        <CardTitle>
          Agent Artifacts
        </CardTitle>
      </CardHeader>
      <CardContent>
        {result ? (
          <div className="bg-white rounded-lg prose prose-sm max-w-none prose-headings:text-gray-900 prose-p:text-gray-700 prose-strong:text-gray-900 prose-code:bg-gray-100 prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-pre:bg-gray-900 prose-pre:text-gray-100">
            <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]} components={components}>
              {result}
            </ReactMarkdown>
          </div>
        ) : null}

        {meta ? (
          <div className="mt-4 border rounded-md p-3 bg-white">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
              {typeof meta.subtype === 'string' && meta.subtype ? (
                <div>
                  <div className="text-xs text-muted-foreground">Status</div>
                  <div className="font-medium capitalize">{meta.subtype}{meta.is_error ? " (error)" : ""}</div>
                </div>
              ) : null}
              {typeof meta.num_turns === 'number' ? (
                <div>
                  <div className="text-xs text-muted-foreground">Turns</div>
                  <div className="font-medium">{meta.num_turns}</div>
                </div>
              ) : null}
              {typeof meta.duration_ms === 'number' ? (
                <div>
                  <div className="text-xs text-muted-foreground">Duration</div>
                  <div className="font-medium">{meta.duration_ms} ms</div>
                </div>
              ) : null}
              {typeof meta.duration_api_ms === 'number' ? (
                <div>
                  <div className="text-xs text-muted-foreground">API Time</div>
                  <div className="font-medium">{meta.duration_api_ms} ms</div>
                </div>
              ) : null}
              {typeof meta.total_cost_usd === 'number' ? (
                <div>
                  <div className="text-xs text-muted-foreground">Cost (USD)</div>
                  <div className="font-medium">${meta.total_cost_usd.toFixed(6)}</div>
                </div>
              ) : null}
              {typeof meta.session_id === 'string' && meta.session_id ? (
                <div className="sm:col-span-2">
                  <div className="text-xs text-muted-foreground">Session ID</div>
                  <div className="font-mono text-xs break-all">{meta.session_id}</div>
                </div>
              ) : null}
            </div>

            {meta.usage ? (
              <div className="mt-3">
                <div className="text-xs text-muted-foreground mb-1">Usage</div>
                <pre className="bg-gray-900 text-gray-100 rounded p-3 text-xs overflow-auto"><code>{JSON.stringify(meta.usage, null, 2)}</code></pre>
              </div>
            ) : null}
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
};

export default ResultsTab;


