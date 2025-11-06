"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { RefreshCw, Save, Loader2, Info } from "lucide-react";
import { Plus, Trash2, Eye, EyeOff } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { successToast, errorToast } from "@/hooks/use-toast";
import { useProject, useUpdateProject } from "@/services/queries/use-projects";
import { useSecretsList, useSecretsConfig, useSecretsValues, useUpdateSecretsConfig, useUpdateSecrets } from "@/services/queries/use-secrets";
import { useMemo } from "react";

type SettingsSectionProps = {
  projectName: string;
};

export function SettingsSection({ projectName }: SettingsSectionProps) {
  const [formData, setFormData] = useState({ displayName: "", description: "" });
  const [secretName, setSecretName] = useState<string>("");
  const [secrets, setSecrets] = useState<Array<{ key: string; value: string }>>([]);
  const [mode, setMode] = useState<"existing" | "new">("existing");
  const [showValues, setShowValues] = useState<Record<number, boolean>>({});
  const [anthropicApiKey, setAnthropicApiKey] = useState<string>("");
  const [showAnthropicKey, setShowAnthropicKey] = useState<boolean>(false);
  const [gitUserName, setGitUserName] = useState<string>("");
  const [gitUserEmail, setGitUserEmail] = useState<string>("");
  const [gitToken, setGitToken] = useState<string>("");
  const [showGitToken, setShowGitToken] = useState<boolean>(false);
  const [jiraUrl, setJiraUrl] = useState<string>("");
  const [jiraProject, setJiraProject] = useState<string>("");
  const [jiraEmail, setJiraEmail] = useState<string>("");
  const [jiraToken, setJiraToken] = useState<string>("");
  const [showJiraToken, setShowJiraToken] = useState<boolean>(false);
  const FIXED_KEYS = useMemo(() => ["ANTHROPIC_API_KEY","GIT_USER_NAME","GIT_USER_EMAIL","GIT_TOKEN","JIRA_URL","JIRA_PROJECT","JIRA_EMAIL","JIRA_API_TOKEN"] as const, []);

  // React Query hooks
  const { data: project, isLoading: projectLoading, refetch: refetchProject } = useProject(projectName);
  const { data: secretsList } = useSecretsList(projectName);
  const { data: secretsConfig } = useSecretsConfig(projectName);
  const { data: secretsValues } = useSecretsValues(projectName);
  const updateProjectMutation = useUpdateProject();
  const updateSecretsConfigMutation = useUpdateSecretsConfig();
  const updateSecretsMutation = useUpdateSecrets();

  // Sync project data to form
  useEffect(() => {
    if (project) {
      setFormData({ displayName: project.displayName || "", description: project.description || "" });
    }
  }, [project]);

  // Sync secrets config to state
  useEffect(() => {
    if (secretsConfig) {
      if (secretsConfig.secretName) {
        setSecretName(secretsConfig.secretName);
        setMode("existing");
      } else {
        setSecretName("ambient-runner-secrets");
        setMode("new");
      }
    }
  }, [secretsConfig]);

  // Sync secrets values to state
  useEffect(() => {
    if (secretsValues) {
      const byKey: Record<string, string> = Object.fromEntries(secretsValues.map(s => [s.key, s.value]));
      setAnthropicApiKey(byKey["ANTHROPIC_API_KEY"] || "");
      setGitUserName(byKey["GIT_USER_NAME"] || "");
      setGitUserEmail(byKey["GIT_USER_EMAIL"] || "");
      setGitToken(byKey["GIT_TOKEN"] || "");
      setJiraUrl(byKey["JIRA_URL"] || "");
      setJiraProject(byKey["JIRA_PROJECT"] || "");
      setJiraEmail(byKey["JIRA_EMAIL"] || "");
      setJiraToken(byKey["JIRA_API_TOKEN"] || "");
      setSecrets(secretsValues.filter(s => !FIXED_KEYS.includes(s.key as typeof FIXED_KEYS[number])));
    }
  }, [secretsValues, FIXED_KEYS]);

  const handleRefresh = () => {
    void refetchProject();
  };

  const handleSave = () => {
    if (!project) return;
    updateProjectMutation.mutate(
      {
        name: projectName,
        data: {
          displayName: formData.displayName.trim(),
          description: formData.description.trim() || undefined,
          annotations: project.annotations || {},
        },
      },
      {
        onSuccess: () => {
          successToast("Project settings updated successfully!");
        },
        onError: (error) => {
          const message = error instanceof Error ? error.message : "Failed to update project";
          errorToast(message);
        },
      }
    );
  };

  const handleSaveSecrets = () => {
    if (!projectName) return;

    const name = secretName.trim() || "ambient-runner-secrets";

    // First update config
    updateSecretsConfigMutation.mutate(
      { projectName, secretName: name },
      {
        onSuccess: () => {
          // Then update secrets values
          const data: Record<string, string> = {};
          if (anthropicApiKey) data["ANTHROPIC_API_KEY"] = anthropicApiKey;
          if (gitUserName) data["GIT_USER_NAME"] = gitUserName;
          if (gitUserEmail) data["GIT_USER_EMAIL"] = gitUserEmail;
          if (gitToken) data["GIT_TOKEN"] = gitToken;
          if (jiraUrl) data["JIRA_URL"] = jiraUrl;
          if (jiraProject) data["JIRA_PROJECT"] = jiraProject;
          if (jiraEmail) data["JIRA_EMAIL"] = jiraEmail;
          if (jiraToken) data["JIRA_API_TOKEN"] = jiraToken;
          for (const { key, value } of secrets) {
            if (!key) continue;
            if (FIXED_KEYS.includes(key as typeof FIXED_KEYS[number])) continue;
            data[key] = value ?? "";
          }

          updateSecretsMutation.mutate(
            {
              projectName,
              secrets: Object.entries(data).map(([key, value]) => ({ key, value })),
            },
            {
              onSuccess: () => {
                successToast("Secrets saved successfully!");
              },
              onError: (error) => {
                const message = error instanceof Error ? error.message : "Failed to save secrets";
                errorToast(message);
              },
            }
          );
        },
        onError: (error) => {
          const message = error instanceof Error ? error.message : "Failed to save secret config";
          errorToast(message);
        },
      }
    );
  };

  const addSecretRow = () => {
    setSecrets((prev) => [...prev, { key: "", value: "" }]);
  };

  const removeSecretRow = (idx: number) => {
    setSecrets((prev) => prev.filter((_, i) => i !== idx));
  };

  return (
    <div className="flex-1 space-y-6">
      {/* Only show project metadata editor on OpenShift */}
      {project?.isOpenShift ? (
        <Card>
          <CardHeader>
            <CardTitle>General Settings</CardTitle>
            <CardDescription>Basic workspace configuration</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="displayName">Display Name</Label>
              <Input
                id="displayName"
                value={formData.displayName}
                onChange={(e) => setFormData((prev) => ({ ...prev, displayName: e.target.value }))}
                placeholder="My Awesome Workspace"
                maxLength={100}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="workspaceName">Workspace Name</Label>
              <Input
                id="workspaceName"
                value={projectName}
                readOnly
                disabled
                className="bg-muted/80 text-muted-foreground"
              />
              <p className="text-sm text-muted-foreground">Workspace name cannot be changed after creation</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={formData.description}
                onChange={(e) => setFormData((prev) => ({ ...prev, description: e.target.value }))}
                placeholder="Describe the purpose and goals of this workspace..."
                maxLength={500}
                rows={3}
              />
            </div>
            <div className="flex gap-3 pt-2">
              <Button onClick={handleSave} disabled={updateProjectMutation.isPending || projectLoading || !project}>
                {updateProjectMutation.isPending ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Saving...
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4 mr-2" />
                    Save Changes
                  </>
                )}
              </Button>
              <Button variant="outline" onClick={handleRefresh} disabled={updateProjectMutation.isPending || projectLoading}>
                <RefreshCw className={`w-4 h-4 mr-2 ${projectLoading ? "animate-spin" : ""}`} />
                Reset
              </Button>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Alert>
          <Info className="h-4 w-4" />
          <AlertDescription>
            Running on vanilla Kubernetes. Project display name and description editing is not available.
            The project namespace is: <strong>{projectName}</strong>
          </AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Runner Secrets</CardTitle>
          <CardDescription>
            Configure the Secret and manage key/value pairs used by workspace runners.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-3">
              <div>
                <Label>Runner Secret</Label>
                <div className="text-sm text-muted-foreground">Using: {secretName || "ambient-runner-secrets"}</div>
              </div>
            </div>
            <Tabs value={mode} onValueChange={(v) => setMode(v as typeof mode)}>
              <TabsList>
                <TabsTrigger value="existing">Use existing</TabsTrigger>
                <TabsTrigger value="new">Create new</TabsTrigger>
              </TabsList>
              <TabsContent value="existing">
                <div className="flex gap-2 items-center pt-2">
                  {(secretsList?.items?.length ?? 0) > 0 && (
                    <Select
                      value={secretName}
                      onValueChange={setSecretName}
                    >
                      <SelectTrigger className="w-80">
                        <SelectValue placeholder="Select a secret..." />
                      </SelectTrigger>
                      <SelectContent>
                        {secretsList?.items?.map((s) => (
                          <SelectItem key={s.name} value={s.name}>{s.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                </div>
                {(secretsList?.items?.length ?? 0) === 0 ? (
                  <div className="mt-2 text-sm text-amber-600">No runner secrets found in this project. Use the &quot;Create new&quot; tab to create one.</div>
                ) : (!secretName ? (
                  <div className="mt-2 text-sm text-muted-foreground">No secret selected. You can still add key/value pairs below and Save; they will be written to the default secret name.</div>
                ) : null)}
              </TabsContent>
              <TabsContent value="new">
                <div className="flex gap-2 items-center pt-2">
                  <Input
                    id="secretName"
                    value={secretName}
                    onChange={(e) => setSecretName(e.target.value)}
                    placeholder="ambient-runner-secrets"
                    maxLength={253}
                  />
                </div>
              </TabsContent>
            </Tabs>
          </div>

          {(mode === "new" || (mode === "existing" && !!secretName)) && (
            <div className="pt-2 space-y-4">
              <div className="space-y-2">
                <Label>Additional Secrets</Label>
                <div className="space-y-2">
                  {secrets.length === 0 && (
                    <div className="text-sm text-muted-foreground">No keys configured.</div>
                  )}
                  {secrets.map((item, idx) => (
                    <div key={idx} className="flex gap-2 items-center">
                      <Input
                        value={item.key}
                        onChange={(e) =>
                          setSecrets((prev) => prev.map((it, i) => (i === idx ? { ...it, key: e.target.value } : it)))
                        }
                        placeholder="KEY"
                        className="w-1/3"
                      />
                      <div className="flex-1 flex items-center gap-2">
                        <Input
                          type={showValues[idx] ? "text" : "password"}
                          value={item.value}
                          onChange={(e) =>
                            setSecrets((prev) => prev.map((it, i) => (i === idx ? { ...it, value: e.target.value } : it)))
                          }
                          placeholder="value"
                          className="flex-1"
                        />
                        <Button
                          type="button"
                          variant="ghost"
                          onClick={() => setShowValues((prev) => ({ ...prev, [idx]: !prev[idx] }))}
                          aria-label={showValues[idx] ? "Hide value" : "Show value"}
                        >
                          {showValues[idx] ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                        </Button>
                      </div>
                      <Button variant="ghost" onClick={() => removeSecretRow(idx)} aria-label="Remove row">
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  ))}
                </div>
              </div>
              <Button variant="outline" onClick={addSecretRow} className="w-full">
                <Plus className="w-4 h-4 mr-2" /> Add Secret
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {(mode === "new" || (mode === "existing" && !!secretName)) && (
        <div className="flex gap-3">
          <Button
            onClick={handleSaveSecrets}
            disabled={
              updateSecretsConfigMutation.isPending ||
              updateSecretsMutation.isPending ||
              (mode === "existing" && ((secretsList?.items?.length ?? 0) === 0 || !secretName))
            }
          >
            {(updateSecretsConfigMutation.isPending || updateSecretsMutation.isPending) ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Saving Secrets
              </>
            ) : (
              <>
                <Save className="w-4 h-4 mr-2" />
                Save Secrets
              </>
            )}
          </Button>
        </div>
      )}

      {(mode === "new" || (mode === "existing" && !!secretName)) && (
        <Card>
          <CardHeader>
            <CardTitle>API Keys & Integrations</CardTitle>
            <CardDescription>
              Configure API keys and external integrations for your workspace runners.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="space-y-3">
              <Label className="text-base font-semibold">Anthropic API Key (Required)</Label>
              <div className="text-xs text-muted-foreground mb-3">Your Anthropic API key for Claude Code runner</div>
              <div className="flex items-center gap-2">
                <Input
                  id="anthropicApiKey"
                  type={showAnthropicKey ? "text" : "password"}
                  placeholder="sk-ant-..."
                  value={anthropicApiKey}
                  onChange={(e) => setAnthropicApiKey(e.target.value)}
                  className="flex-1"
                />
                <Button type="button" variant="ghost" onClick={() => setShowAnthropicKey((v) => !v)} aria-label={showAnthropicKey ? "Hide key" : "Show key"}>
                  {showAnthropicKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </Button>
              </div>
            </div>

            <div className="pt-4 space-y-3 border-t">
              <Label className="text-base font-semibold">Git Integration (Optional)</Label>
              <div className="text-xs text-muted-foreground mb-3">Configure Git credentials for repository operations (clone, commit, push)</div>
              <div className="text-xs text-blue-600 bg-blue-50 border border-blue-200 rounded p-2 mb-3">
                <strong>Note:</strong> These fields are only needed if you have not connected a GitHub Application. When GitHub App integration is configured, it will be used automatically and these fields will serve as a fallback.
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label htmlFor="gitUserName">Git User Name</Label>
                  <Input id="gitUserName" placeholder="Your Name" value={gitUserName} onChange={(e) => setGitUserName(e.target.value)} />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="gitUserEmail">Git User Email</Label>
                  <Input id="gitUserEmail" placeholder="you@example.com" value={gitUserEmail} onChange={(e) => setGitUserEmail(e.target.value)} />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="gitToken">GitHub API Token</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="gitToken"
                    type={showGitToken ? "text" : "password"}
                    placeholder="ghp_... or glpat-..."
                    value={gitToken}
                    onChange={(e) => setGitToken(e.target.value)}
                    className="flex-1"
                  />
                  <Button type="button" variant="ghost" onClick={() => setShowGitToken((v) => !v)} aria-label={showGitToken ? "Hide token" : "Show token"}>
                    {showGitToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </Button>
                </div>
                <div className="text-xs text-muted-foreground">GitHub personal access token or fine-grained token for git operations and API access</div>
              </div>
              <div className="text-xs text-muted-foreground">Git credentials will be saved with keys: GIT_USER_NAME, GIT_USER_EMAIL, GIT_TOKEN</div>
            </div>

            <div className="pt-4 space-y-3 border-t">
              <Label className="text-base font-semibold">Jira Integration (Optional)</Label>
              <div className="text-xs text-muted-foreground mb-3">Configure Jira integration for issue management</div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label htmlFor="jiraUrl">Jira Base URL</Label>
                  <Input id="jiraUrl" placeholder="https://your-domain.atlassian.net" value={jiraUrl} onChange={(e) => setJiraUrl(e.target.value)} />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="jiraProject">Jira Project Key</Label>
                  <Input id="jiraProject" placeholder="ABC" value={jiraProject} onChange={(e) => setJiraProject(e.target.value)} />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="jiraEmail">Jira Email/Username</Label>
                  <Input id="jiraEmail" placeholder="you@example.com" value={jiraEmail} onChange={(e) => setJiraEmail(e.target.value)} />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="jiraToken">Jira API Token</Label>
                  <div className="flex items-center gap-2">
                    <Input id="jiraToken" type={showJiraToken ? "text" : "password"} placeholder="token" value={jiraToken} onChange={(e) => setJiraToken(e.target.value)} />
                    <Button type="button" variant="ghost" onClick={() => setShowJiraToken((v) => !v)} aria-label={showJiraToken ? "Hide token" : "Show token"}>
                      {showJiraToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </Button>
                  </div>
                </div>
              </div>
              <div className="text-xs text-muted-foreground">Jira credentials will be saved with keys: JIRA_URL, JIRA_PROJECT, JIRA_EMAIL, JIRA_API_TOKEN</div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

