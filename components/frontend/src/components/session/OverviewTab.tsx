"use client";

import React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Brain, Clock, RefreshCw, Sparkle, ExternalLink } from "lucide-react";
import { format, formatDistanceToNow } from "date-fns";
import { cn } from "@/lib/utils";
import type { AgenticSession } from "@/types/agentic-session";

type Props = {
  session: AgenticSession;
  promptExpanded: boolean;
  setPromptExpanded: (v: boolean) => void;
  latestLiveMessage: any;
  subagentStats: { uniqueCount: number; orderedTypes: string[] };
  diffTotals: Record<number, { added: number; modified: number; deleted: number; renamed: number; untracked: number }>;
  onPush: (repoIndex: number) => Promise<void>;
  onAbandon: (repoIndex: number) => Promise<void>;
  busyRepo: Record<number, 'push' | 'abandon' | null>;
  buildGithubCompareUrl: (inUrl: string, inBranch?: string, outUrl?: string, outBranch?: string) => string | null;
};

export const OverviewTab: React.FC<Props> = ({ session, promptExpanded, setPromptExpanded, latestLiveMessage, subagentStats, diffTotals, onPush, onAbandon, busyRepo, buildGithubCompareUrl }) => {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Brain className="w-5 h-5 mr-2" />
              Initial Prompt
            </CardTitle>
          </CardHeader>
          <CardContent>
            {(() => {
              const promptText = session.spec.prompt || "";
              const promptIsLong = promptText.length > 400;
              return (
                <>
                  <div className={cn("relative", !promptExpanded && promptIsLong ? "max-h-40 overflow-hidden" : "")}>
                    <p className="whitespace-pre-wrap text-sm">{promptText}</p>
                    {!promptExpanded && promptIsLong ? (
                      <div className="absolute inset-x-0 bottom-0 h-12 bg-gradient-to-t from-white to-transparent pointer-events-none" />
                    ) : null}
                  </div>
                  {promptIsLong && (
                    <button
                      className="mt-2 text-xs text-blue-600 hover:underline"
                      onClick={() => setPromptExpanded(!promptExpanded)}
                      aria-expanded={promptExpanded}
                      aria-controls="initial-prompt"
                    >
                      {promptExpanded ? "View less" : "View more"}
                    </button>
                  )}
                </>
              );
            })()}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Latest Message</CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            {latestLiveMessage ? (
              <div className="space-y-2 text-sm">
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-xs">{latestLiveMessage.type}</Badge>
                  <span className="text-xs text-gray-500">{new Date(latestLiveMessage.timestamp).toLocaleTimeString()}</span>
                </div>
                <pre className="whitespace-pre-wrap break-words bg-gray-50 rounded p-2 text-xs text-gray-800">{JSON.stringify(latestLiveMessage.payload, null, 2)}</pre>
              </div>
            ) : (
              <div className="text-sm text-gray-500">No messages yet</div>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 gap-6">
        {session.status && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Clock className="w-5 h-5 mr-2" />
                System Status & Configuration
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4 text-sm">
                <div>
                  <div className="text-xs font-semibold text-muted-foreground mb-2">Runtime</div>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    {session.status.message && (
                      <div>
                        <p className="font-semibold">Status</p>
                        <p className="text-muted-foreground">{session.status.message}</p>
                      </div>
                    )}
                    {session.status.startTime && (
                      <div>
                        <p className="font-semibold">Started</p>
                        <p className="text-muted-foreground">{format(new Date(session.status.startTime), "PPp")}</p>
                      </div>
                    )}
                    {session.status.completionTime && (
                      <div>
                        <p className="font-semibold">Completed</p>
                        <p className="text-muted-foreground">{format(new Date(session.status.completionTime), "PPp")}</p>
                      </div>
                    )}
                    {session.status.jobName && (
                      <div>
                        <p className="font-semibold">K8s Job</p>
                        <div className="flex items-center gap-2">
                          <p className="text-muted-foreground font-mono text-xs">{session.status.jobName}</p>
                          <Badge variant="outline" className={session.spec?.interactive ? "bg-green-50 text-green-700 border-green-200" : "bg-gray-50 text-gray-700 border-gray-200"}>
                            {session.spec?.interactive ? "Interactive" : "Headless"}
                          </Badge>
                        </div>
                      </div>
                    )}
                  </div>
                </div>

                <div>
                  <div className="text-xs font-semibold text-muted-foreground mb-2">LLM Config</div>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div>
                      <p className="font-semibold">Model</p>
                      <p className="text-muted-foreground">{session.spec.llmSettings.model}</p>
                    </div>
                    <div>
                      <p className="font-semibold">Temperature</p>
                      <p className="text-muted-foreground">{session.spec.llmSettings.temperature}</p>
                    </div>
                    <div>
                      <p className="font-semibold">Max Tokens</p>
                      <p className="text-muted-foreground">{session.spec.llmSettings.maxTokens}</p>
                    </div>
                    <div>
                      <p className="font-semibold">Timeout</p>
                      <p className="text-muted-foreground">{session.spec.timeout}s</p>
                    </div>
                  </div>
                </div>

                <div>
                  <div className="text-xs font-semibold text-muted-foreground mb-2">Repositories</div>
                  {session.spec.repos && session.spec.repos.length > 0 ? (
                    <div className="space-y-2">
                      {session.spec.repos.map((repo, idx) => {
                        const isMain = session.spec.mainRepoIndex === idx;
                        // Use the actual output branch, or default to sessions/{sessionName}
                        const outBranch = repo.output?.branch && repo.output.branch.trim() && repo.output.branch !== 'auto' 
                          ? repo.output.branch 
                          : `sessions/${session.metadata.name}`;
                        const compareUrl = buildGithubCompareUrl(repo.input.url, repo.input.branch || 'main', repo.output?.url, outBranch);
                        const br = diffTotals[idx] || { added: 0, modified: 0, deleted: 0, renamed: 0, untracked: 0 };
                        const total = br.added + br.modified + br.deleted + br.renamed + br.untracked;
                        return (
                          <div key={idx} className="flex items-center gap-2 text-sm font-mono">
                            {isMain && <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded font-sans">MAIN</span>}
                            <span className="text-muted-foreground break-all">{repo.input.url}</span>
                            <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded font-sans">{repo.input.branch || "main"}</span>
                            <span className="text-muted-foreground">→</span>
                            <span className="text-muted-foreground break-all">{repo.output?.url || "(no push)"}</span>
                            {repo.output?.url && (
                              <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded font-sans">{repo.output?.branch || (
                                <div className="flex items-center gap-1">
                                <Sparkle
                                className="h-3 w-3 text-muted-foreground"
                                />
                                auto
                                </div>
                               )}</span>
                            )}
                            {repo.status && (
                              <span className="text-xs px-2 py-0.5 rounded font-sans border border-muted-foreground/20">
                                {repo.status}
                              </span>
                            )}
                            <span className="flex-1" />
                            {total === 0 ? (
                              repo.status === 'pushed' && compareUrl ? (
                                <a 
                                  href={compareUrl} 
                                  target="_blank" 
                                  rel="noreferrer" 
                                  className="flex items-center gap-1 text-xs text-blue-600 hover:underline"
                                >
                                  Compare
                                  <ExternalLink className="h-3 w-3" />
                                </a>
                              ) : (
                                <span className="text-xs text-muted-foreground">no diff</span>
                              )
                            ) : (
                              <span className="flex items-center gap-1">
                                {br.added > 0 && <span className="text-xs px-1 py-0.5 rounded border bg-green-50 text-green-700">+ {br.added}</span>}
                                {br.modified > 0 && <span className="text-xs px-1 py-0.5 rounded border bg-yellow-50 text-yellow-700">~ {br.modified}</span>}
                                {br.deleted > 0 && <span className="text-xs px-1 py-0.5 rounded border bg-red-50 text-red-700">- {br.deleted}</span>}
                              </span>
                            )}
                            {total > 0 && compareUrl && repo.status === 'pushed' ? (
                              <a 
                                href={compareUrl} 
                                target="_blank" 
                                rel="noreferrer" 
                                className="flex items-center gap-1 text-xs text-blue-600 hover:underline"
                              >
                                Compare
                                <ExternalLink className="h-3 w-3" />
                              </a>
                            ) : null}
                            {total > 0 && (
                              repo.output?.url ? (
                                <div className="flex items-center gap-2">
                                  <Button size="sm" variant="secondary" onClick={() => onPush(idx)}>{busyRepo[idx] === 'push' ? 'Pushing…' : 'Push'}</Button>
                                  <Button size="sm" variant="outline" onClick={() => onAbandon(idx)}>{busyRepo[idx] === 'abandon' ? 'Abandoning…' : 'Abandon'}</Button>
                                </div>
                              ) : (
                                <Button size="sm" variant="outline" onClick={() => onAbandon(idx)}>{busyRepo[idx] === 'abandon' ? 'Abandoning…' : 'Abandon changes'}</Button>
                              )
                            )}
                          </div>
                        );
                      })}
                    </div>
                  ) : (
                    <p className="text-muted-foreground">No repositories configured</p>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
};

export default OverviewTab;


