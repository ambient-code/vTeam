"use client";

import { useState } from "react";
import { GitBranch, X, Link, Loader2 } from "lucide-react";
import { AccordionItem, AccordionTrigger, AccordionContent } from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

type Repository = {
  input: {
    url: string;
    branch?: string;
  };
};

type RepositoriesAccordionProps = {
  repositories?: Repository[];
  onAddRepository: () => void;
  onRemoveRepository: (repoName: string) => void;
};

export function RepositoriesAccordion({
  repositories = [],
  onAddRepository,
  onRemoveRepository,
}: RepositoriesAccordionProps) {
  const [removingRepo, setRemovingRepo] = useState<string | null>(null);

  const handleRemove = async (repoName: string) => {
    if (confirm(`Remove repository ${repoName}?`)) {
      setRemovingRepo(repoName);
      try {
        await onRemoveRepository(repoName);
      } finally {
        setRemovingRepo(null);
      }
    }
  };

  return (
    <AccordionItem value="context" className="border rounded-lg px-3 bg-white">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <Link className="h-4 w-4" />
          <span>Context</span>
          {repositories.length > 0 && (
            <Badge variant="secondary" className="ml-auto mr-2">
              {repositories.length}
            </Badge>
          )}
        </div>
      </AccordionTrigger>
      <AccordionContent className="pt-2 pb-3">
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Add additional context to improve AI responses.
          </p>
          
          {/* Repository List */}
          {repositories.length === 0 ? (
            <div className="text-center py-6">
              <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-gray-100 mb-2">
                <Link className="h-5 w-5 text-gray-400" />
              </div>
              <p className="text-sm text-muted-foreground mb-3">No context added yet</p>
              <Button size="sm" variant="outline" onClick={onAddRepository}>
                <Link className="mr-2 h-3 w-3" />
                Add Context
              </Button>
            </div>
          ) : (
            <div className="space-y-2">
              {repositories.map((repo, idx) => {
                const repoName = repo.input.url.split('/').pop()?.replace('.git', '') || `repo-${idx}`;
                const isRemoving = removingRepo === repoName;
                
                return (
                  <div key={idx} className="flex items-center gap-2 p-2 border rounded bg-muted/30 hover:bg-muted/50 transition-colors">
                    <GitBranch className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium truncate">{repoName}</div>
                      <div className="text-xs text-muted-foreground truncate">{repo.input.url}</div>
                    </div>
                    <Button 
                      variant="ghost"
                      size="sm" 
                      className="h-7 w-7 p-0 flex-shrink-0"
                      onClick={() => handleRemove(repoName)}
                      disabled={isRemoving}
                    >
                      {isRemoving ? (
                        <Loader2 className="h-3 w-3 animate-spin" />
                      ) : (
                        <X className="h-3 w-3" />
                      )}
                    </Button>
                  </div>
                );
              })}
              <Button onClick={onAddRepository} variant="outline" className="w-full" size="sm">
                <Link className="mr-2 h-3 w-3" />
                Add Context
              </Button>
            </div>
          )}
        </div>
      </AccordionContent>
    </AccordionItem>
  );
}

