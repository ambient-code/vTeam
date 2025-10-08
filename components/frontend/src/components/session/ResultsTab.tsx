"use client";

import React from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type Props = {
  result?: string | null;
  components?: any;
};

const ResultsTab: React.FC<Props> = ({ result, components }) => {
  if (!result) return <div className="text-sm text-muted-foreground">No results yet</div>;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Agent Results</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="bg-white rounded-lg prose prose-sm max-w-none prose-headings:text-gray-900 prose-p:text-gray-700 prose-strong:text-gray-900 prose-code:bg-gray-100 prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-pre:bg-gray-900 prose-pre:text-gray-100">
          <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]} components={components}>
            {result}
          </ReactMarkdown>
        </div>
      </CardContent>
    </Card>
  );
};

export default ResultsTab;


