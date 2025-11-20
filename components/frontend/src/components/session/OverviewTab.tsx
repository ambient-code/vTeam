"use client";

import React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Brain, Clock, RefreshCw, Sparkle, ExternalLink, Box, Container, HardDrive } from "lucide-react";
import { format } from "date-fns";
import { cn } from "@/lib/utils";
import type { AgenticSession } from "@/types/agentic-session";
import type { SessionMessage } from "@/types";
import { getK8sResourceStatusColor } from "@/lib/status-colors";

type Props = {
  session: AgenticSession;
  promptExpanded: boolean;
  setPromptExpanded: (v: boolean) => void;
  latestLiveMessage: SessionMessage | null;
  diffTotals: Record<number, { total_added: number; total_removed: number }>;
  onPush: (repoIndex: number) => Promise<void>;
  onAbandon: (repoIndex: number) => Promise<void>;
  busyRepo: Record<number, 'push' | 'abandon' | null>;
  buildGithubCompareUrl: (inUrl: string, inBranch?: string, outUrl?: string, outBranch?: string) => string | null;
  onRefreshDiff: () => Promise<void>;
  k8sResources?: {
    jobName?: string;
    jobStatus?: string;
    pods?: Array<{
      name: string;
      phase: string;
      containers: Array<{
        name: string;
        state: string;
        exitCode?: number;
        reason?: string;
      }>;
      isTempPod?: boolean;
    }>;
    pvcName?: string;
    pvcExists?: boolean;
    pvcSize?: string;
  };
};

// Utility to generate OpenShift console URLs
const getOpenShiftConsoleUrl = (namespace: string, resourceType: 'Job' | 'Pod' | 'PVC', resourceName: string): string | null => {
  // Try to derive console URL from current window location
  // OpenShift console is typically at console-openshift-console.apps.<cluster-domain>
  const hostname = window.location.hostname;
  
  // Check if we're on an OpenShift route (apps.*)
  if (hostname.includes('.apps.')) {
    const clusterDomain = hostname.split('.apps.')[1];
    const consoleUrl = `https://console-openshift-console.apps.${clusterDomain}`;
    
    const resourceMap = {
      'Job': 'batch~v1~Job',
      'Pod': 'core~v1~Pod',
      'PVC': 'core~v1~PersistentVolumeClaim',
    };
    
    return `${consoleUrl}/k8s/ns/${namespace}/${resourceMap[resourceType]}/${resourceName}`;
  }
  
  // Fallback: For local development or non-standard setups, return null
  return null;
};

export const OverviewTab: React.FC<Props> = ({ session, promptExpanded, setPromptExpanded, latestLiveMessage, diffTotals, onPush, onAbandon, busyRepo, buildGithubCompareUrl, onRefreshDiff, k8sResources }) => {
  const [refreshingDiff, setRefreshingDiff] = React.useState(false);
  const [expandedPods, setExpandedPods] = React.useState<Record<string, boolean>>({});

  const projectNamespace = session.metadata?.namespace || '';

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
                      className="mt-2 text-xs text-link hover:underline"
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
                  <span className="text-xs text-muted-foreground">{new Date(latestLiveMessage.timestamp).toLocaleTimeString()}</span>
                </div>
                <div className="relative max-h-40 overflow-hidden">
                  <pre className="whitespace-pre-wrap break-words bg-muted/50 rounded p-2 text-xs text-foreground">{JSON.stringify(latestLiveMessage.payload, null, 2)}</pre>
                  <div className="absolute inset-x-0 bottom-0 h-12 bg-gradient-to-t from-white to-transparent pointer-events-none" />
                </div>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">No messages yet</div>
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
                          <Badge variant="outline" className={session.spec?.interactive ? "bg-green-50 text-green-700 border-green-200" : "bg-muted/50 text-foreground/80 border-gray-200"}>
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

                {k8sResources && (
                  <div>
                    <div className="text-xs font-semibold text-muted-foreground mb-2">Kubernetes Resources</div>
                    <div className="space-y-2">
                      {/* PVC - Always shown at root level (owned by AgenticSession CR) */}
                      {k8sResources.pvcName && (
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className="text-xs">
                            <HardDrive className="w-3 h-3 mr-1" />
                            PVC
                          </Badge>
                          {(() => {
                            const consoleUrl = getOpenShiftConsoleUrl(projectNamespace, 'PVC', k8sResources.pvcName);
                            return consoleUrl ? (
                              <a
                                href={consoleUrl}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="font-mono text-xs text-link hover:text-link-hover hover:underline flex items-center gap-1"
                              >
                                {k8sResources.pvcName}
                                <ExternalLink className="w-3 h-3" />
                              </a>
                            ) : (
                              <span className="font-mono text-xs">{k8sResources.pvcName}</span>
                            );
                          })()}
                          <Badge className={`text-xs ${k8sResources.pvcExists ? 'bg-green-100 text-green-800 border-green-300 dark:bg-green-700 dark:text-white dark:border-green-700' : 'bg-red-100 text-red-800 border-red-300 dark:bg-red-700 dark:text-white dark:border-red-700'}`}>
                            {k8sResources.pvcExists ? 'Exists' : 'Not Found'}
                          </Badge>
                          {k8sResources.pvcSize && <span className="text-xs text-muted-foreground">{k8sResources.pvcSize}</span>}
                        </div>
                      )}
                      
                      {/* Temp Content Pods - Always at root level (for completed sessions) */}
                      {k8sResources.pods && k8sResources.pods.filter(p => p.isTempPod).map((pod) => (
                        <div key={pod.name} className="space-y-1">
                          <div className="flex items-center gap-2">
                            <button
                              onClick={() => setExpandedPods(prev => ({ ...prev, [pod.name]: !prev[pod.name] }))}
                              className="text-xs text-link hover:underline flex items-center gap-1"
                            >
                              {expandedPods[pod.name] ? 'Hide' : 'Show'}
                            </button>
                            <Badge variant="outline" className="text-xs">
                              <Container className="w-3 h-3 mr-1" />
                              Temp Pod
                            </Badge>
                            {(() => {
                              const consoleUrl = getOpenShiftConsoleUrl(projectNamespace, 'Pod', pod.name);
                              return consoleUrl ? (
                                <a
                                  href={consoleUrl}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="font-mono text-xs text-link hover:text-link-hover hover:underline flex items-center gap-1 truncate max-w-[250px]"
                                  title={pod.name}
                                >
                                  {pod.name}
                                  <ExternalLink className="w-3 h-3 flex-shrink-0" />
                                </a>
                              ) : (
                                <span className="font-mono text-xs truncate max-w-[250px]" title={pod.name}>
                                  {pod.name}
                                </span>
                              );
                            })()}
                            <Badge className={`text-xs ${getK8sResourceStatusColor(pod.phase)}`}>
                              {pod.phase}
                            </Badge>
                            <Badge variant="outline" className="text-xs bg-purple-50 text-purple-700 border-purple-200">
                              Workspace viewer
                            </Badge>
                          </div>
                          
                          {/* Temp pod containers */}
                          {expandedPods[pod.name] && pod.containers && pod.containers.length > 0 && (
                            <div className="ml-4 space-y-1 border-l-2 border-gray-200 pl-3">
                              {pod.containers.map((container) => (
                                <div key={container.name} className="flex items-center gap-2">
                                  <Badge variant="outline" className="text-xs">
                                    <Box className="w-3 h-3 mr-1" />
                                    {container.name}
                                  </Badge>
                                  <Badge className={`text-xs ${getK8sResourceStatusColor(container.state)}`}>
                                    {container.state}
                                  </Badge>
                                  {container.exitCode !== undefined && (
                                    <span className="text-xs text-muted-foreground">Exit: {container.exitCode}</span>
                                  )}
                                  {container.reason && (
                                    <span className="text-xs text-muted-foreground">({container.reason})</span>
                                  )}
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      ))}
                      
                      {/* Job - Only shown when job exists */}
                      {k8sResources.jobName && (
                      <div className="text-xs space-y-1">
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className="text-xs">
                            <Box className="w-3 h-3 mr-1" />
                            Job
                          </Badge>
                          {(() => {
                            const consoleUrl = getOpenShiftConsoleUrl(projectNamespace, 'Job', k8sResources.jobName);
                            return consoleUrl ? (
                              <a
                                href={consoleUrl}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="font-mono text-xs text-link hover:text-link-hover hover:underline flex items-center gap-1"
                              >
                                {k8sResources.jobName}
                                <ExternalLink className="w-3 h-3" />
                              </a>
                            ) : (
                              <span className="font-mono text-xs">{k8sResources.jobName}</span>
                            );
                          })()}
                          <Badge className={`text-xs ${getK8sResourceStatusColor(k8sResources.jobStatus || 'Unknown')}`}>
                            {k8sResources.jobStatus || 'Unknown'}
                          </Badge>
                        </div>
                        
                        {/* Job Pods - Only non-temp pods */}
                        {k8sResources.pods && k8sResources.pods.filter(p => !p.isTempPod).length > 0 && (
                          <div className="ml-4 space-y-1 border-l-2 border-gray-200 pl-3">
                            {k8sResources.pods.filter(p => !p.isTempPod).map((pod) => (
                              <div key={pod.name} className="space-y-1">
                                <div className="flex items-center gap-2">
                                  <button
                                    onClick={() => setExpandedPods(prev => ({ ...prev, [pod.name]: !prev[pod.name] }))}
                                    className="text-xs text-link hover:underline flex items-center gap-1"
                                  >
                                    {expandedPods[pod.name] ? 'Hide' : 'Show'}
                                  </button>
                                  <Badge variant="outline" className="text-xs">
                                    <Container className="w-3 h-3 mr-1" />
                                    Pod
                                  </Badge>
                                  {(() => {
                                    const consoleUrl = getOpenShiftConsoleUrl(projectNamespace, 'Pod', pod.name);
                                    return consoleUrl ? (
                                      <a
                                        href={consoleUrl}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="font-mono text-xs text-link hover:text-link-hover hover:underline flex items-center gap-1 truncate max-w-[200px]"
                                        title={pod.name}
                                      >
                                        {pod.name}
                                        <ExternalLink className="w-3 h-3 flex-shrink-0" />
                                      </a>
                                    ) : (
                                      <span className="font-mono text-xs truncate max-w-[200px]" title={pod.name}>
                                        {pod.name}
                                      </span>
                                    );
                                  })()}
                                  <Badge className={`text-xs ${getK8sResourceStatusColor(pod.phase)}`}>
                                    {pod.phase}
                                  </Badge>
                                  {pod.isTempPod && (
                                    <Badge variant="outline" className="text-xs bg-purple-50 text-purple-700 border-purple-200 dark:bg-purple-950/50 dark:text-purple-300 dark:border-purple-800">
                                      Workspace viewer
                                    </Badge>
                                  )}
                                </div>
                                
                                {expandedPods[pod.name] && pod.containers && (
                                  <div className="ml-4 space-y-1 border-l-2 border-gray-200 pl-3">
                                    {pod.containers.map((container) => (
                                      <div key={container.name} className="flex items-center gap-2">
                                        <Badge variant="outline" className="text-xs">
                                          <Box className="w-3 h-3 mr-1" />
                                          {container.name}
                                        </Badge>
                                        <Badge className={`text-xs ${getK8sResourceStatusColor(container.state)}`}>
                                          {container.state}
                                        </Badge>
                                        {container.exitCode !== undefined && (
                                          <span className="text-xs text-muted-foreground">Exit: {container.exitCode}</span>
                                        )}
                                        {container.reason && (
                                          <span className="text-xs text-muted-foreground">({container.reason})</span>
                                        )}
                                      </div>
                                    ))}
                                  </div>
                                )}
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                      )}
                    </div>
                  </div>
                )}

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <div className="text-xs font-semibold text-muted-foreground">Repositories</div>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={async () => {
                        setRefreshingDiff(true);
                        try {
                          await onRefreshDiff();
                        } finally {
                          setRefreshingDiff(false);
                        }
                      }}
                      disabled={refreshingDiff}
                      className="h-6 px-2"
                    >
                      <RefreshCw className={cn("h-3 w-3", refreshingDiff && "animate-spin")} />
                    </Button>
                  </div>
                  {session.spec.repos && session.spec.repos.length > 0 ? (
                    <div className="space-y-2">
                      {session.spec.repos.map((repo, idx) => {
                        const isMain = session.spec.mainRepoIndex === idx;
                        // Use the actual output branch, or default to sessions/{sessionName}
                        const outBranch = repo.output?.branch && repo.output.branch.trim() && repo.output.branch !== 'auto' 
                          ? repo.output.branch 
                          : `sessions/${session.metadata.name}`;
                        const compareUrl = buildGithubCompareUrl(repo.input.url, repo.input.branch || 'main', repo.output?.url, outBranch);
                        
                        // Check if temp pod is running and ready
                        const tempPod = k8sResources?.pods?.find(p => p.isTempPod);
                        const tempPodReady = tempPod?.phase === 'Running';
                        
                        const br = diffTotals[idx] || { total_added: 0, total_removed: 0 };
                        const hasChanges = tempPodReady && (br.total_added > 0 || br.total_removed > 0);
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
                            
                            {!tempPodReady ? (
                              <span className="text-xs text-muted-foreground italic">
                                (read-only - temp service not running)
                              </span>
                            ) : !hasChanges ? (
                              repo.status === 'pushed' && compareUrl ? (
                                <a 
                                  href={compareUrl} 
                                  target="_blank" 
                                  rel="noreferrer" 
                                  className="flex items-center gap-1 text-xs text-link hover:underline"
                                >
                                  Compare
                                  <ExternalLink className="h-3 w-3" />
                                </a>
                              ) : (
                                <span className="text-xs text-muted-foreground">no diff</span>
                              )
                            ) : (
                              <span className="flex items-center gap-2">
                                {br.total_added > 0 && (
                                  <span className="text-xs px-1 py-0.5 rounded border bg-green-50 text-green-700 border-green-200 dark:bg-green-950/50 dark:text-green-300 dark:border-green-800">
                                    +{br.total_added}
                                  </span>
                                )}
                                {br.total_removed > 0 && (
                                  <span className="text-xs px-1 py-0.5 rounded border bg-red-50 text-red-700 border-red-200 dark:bg-red-950/50 dark:text-red-300 dark:border-red-800">
                                    -{br.total_removed}
                                  </span>
                                )}
                              </span>
                            )}
                            {hasChanges && compareUrl && repo.status === 'pushed' ? (
                              <a 
                                href={compareUrl} 
                                target="_blank" 
                                rel="noreferrer" 
                                className="flex items-center gap-1 text-xs text-link hover:underline"
                              >
                                Compare
                                <ExternalLink className="h-3 w-3" />
                              </a>
                            ) : null}
                            {hasChanges && tempPodReady && (
                              repo.output?.url ? (
                                <div className="flex items-center gap-2">
                                  <Button size="sm" variant="secondary" onClick={() => onPush(idx)} disabled={!tempPodReady}>{busyRepo[idx] === 'push' ? 'Pushing…' : 'Push'}</Button>
                                  <Button size="sm" variant="outline" onClick={() => onAbandon(idx)} disabled={!tempPodReady}>{busyRepo[idx] === 'abandon' ? 'Abandoning…' : 'Abandon'}</Button>
                                </div>
                              ) : (
                                <Button size="sm" variant="outline" onClick={() => onAbandon(idx)} disabled={!tempPodReady}>{busyRepo[idx] === 'abandon' ? 'Abandoning…' : 'Abandon changes'}</Button>
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


